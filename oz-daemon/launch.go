package daemon

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/user"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"

	"github.com/subgraph/oz"
	"github.com/subgraph/oz/network"
	"github.com/subgraph/oz/oz-init"
	"github.com/subgraph/oz/xpra"

	"github.com/op/go-logging"
	"github.com/subgraph/oz/fs"
)

type Sandbox struct {
	daemon       *daemonState
	id           int
	display      int
	profile      *oz.Profile
	init         *exec.Cmd
	user         *user.User
	cred         *syscall.Credential
	fs           *fs.Filesystem
	stderr       io.ReadCloser
	addr         string
	xpra         *xpra.Xpra
	ready        sync.WaitGroup
	network      *network.SandboxNetwork
	mountedFiles []string
}

func createSocketPath(base string) (string, error) {
	bs := make([]byte, 8)
	_, err := rand.Read(bs)
	if err != nil {
		return "", err
	}

	return path.Join(base, fmt.Sprintf("oz-init-control-%s", hex.EncodeToString(bs))), nil
}

func createInitCommand(initPath string, cloneNet bool) *exec.Cmd {
	cmd := exec.Command(initPath)
	cmd.Dir = "/"

	cloneFlags := uintptr(syscall.CLONE_NEWNS)
	cloneFlags |= syscall.CLONE_NEWIPC
	cloneFlags |= syscall.CLONE_NEWPID
	cloneFlags |= syscall.CLONE_NEWUTS

	if cloneNet {
		cloneFlags |= syscall.CLONE_NEWNET
	}

	cmd.SysProcAttr = &syscall.SysProcAttr{
		//Chroot:     chroot,
		Cloneflags: cloneFlags,
	}
	
	return cmd
}

func (d *daemonState) launch(p *oz.Profile, msg *LaunchMsg, uid, gid uint32, log *logging.Logger) (*Sandbox, error) {

	/*
		u, err := user.LookupId(fmt.Sprintf("%d", uid))
		if err != nil {
			return nil, fmt.Errorf("failed to lookup user for uid=%d: %v", uid, err)
		}


		fs := fs.NewFromProfile(p, u, d.config.SandboxPath, d.config.UseFullDev, d.log)
		if err := fs.Setup(d.config.ProfileDir); err != nil {
			return nil, err
		}
	*/
	u, err := user.LookupId(strconv.FormatUint(uint64(uid), 10))
	if err != nil {
		return nil, fmt.Errorf("Failed to look up user with uid=%ld: %v", uid, err)
	}
	groups, err := d.sanitizeGroups(p, u.Username, msg.Gids)
	if err != nil {
		return nil, fmt.Errorf("Unable to sanitize user groups: %v", err)
	}

	display := 0
	if p.XServer.Enabled && p.Networking.Nettype == network.TYPE_HOST {
		display = d.nextDisplay
		d.nextDisplay += 1
	}

	stn := new(network.SandboxNetwork)
	stn.Nettype = p.Networking.Nettype
	if p.Networking.Nettype == network.TYPE_BRIDGE {
		stn, err = network.PrepareSandboxNetwork(d.network, log)
		if err != nil {
			return nil, fmt.Errorf("Unable to prepare veth network: %+v", err)
		}
	}

	socketPath, err := createSocketPath(path.Join(d.config.SandboxPath, "sockets"))
	if err != nil {
		return nil, fmt.Errorf("Failed to create random socket path: %v", err)
	}
	initPath := path.Join(d.config.PrefixPath, "bin", "oz-init")
	cmd := createInitCommand(initPath, (stn.Nettype != network.TYPE_HOST))
	pp, err := cmd.StderrPipe()
	if err != nil {
		//fs.Cleanup()
		return nil, fmt.Errorf("error creating stderr pipe for init process: %v", err)
	}
	pi, err := cmd.StdinPipe()
	if err != nil {
		//fs.Cleanup()
		return nil, fmt.Errorf("error creating stdin pipe for init process: %v", err)
	}

	jdata, err := json.Marshal(ozinit.InitData{
		Display:   display,
		User:      *u,
		Uid:       uid,
		Gid:       gid,
		Gids:      groups,
		Network:   *stn,
		Profile:   *p,
		Config:    *d.config,
		Sockaddr:  socketPath,
		LaunchEnv: msg.Env,
	})
	if err != nil {
		return nil, fmt.Errorf("Unable to marshal init state: %+v", err)
	}
	io.Copy(pi, bytes.NewBuffer(jdata))
	pi.Close()

	if err := cmd.Start(); err != nil {
		//fs.Cleanup()
		return nil, fmt.Errorf("Unable to start process: %+v", err)
	}
	//rootfs := path.Join(d.config.SandboxPath, "rootfs")
	sbox := &Sandbox{
		daemon:  d,
		id:      d.nextSboxId,
		display: display,
		profile: p,
		init:    cmd,
		cred:    &syscall.Credential{Uid: uid, Gid: gid, Groups: msg.Gids},
		user:    u,
		fs:      fs.NewFilesystem(d.config, log),
		//addr:    path.Join(rootfs, ozinit.SocketAddress),
		addr:    socketPath,
		stderr:  pp,
		network: stn,
	}

	if p.Networking.Nettype == network.TYPE_BRIDGE {
		if err := network.NetInit(stn, d.network, cmd.Process.Pid, log); err != nil {
			cmd.Process.Kill()
			//fs.Cleanup()
			return nil, fmt.Errorf("Unable to create veth networking: %+v", err)
		}
	}

	sbox.ready.Add(1)
	go sbox.logMessages()

	wgNet := new(sync.WaitGroup)
	if p.Networking.Nettype != network.TYPE_HOST && len(p.Networking.Sockets) > 0 {
		wgNet.Add(1)
		go func() {
			defer wgNet.Done()
			sbox.ready.Wait()
			err := network.ProxySetup(sbox.init.Process.Pid, p.Networking.Sockets, d.log, sbox.ready)
			if err != nil {
				log.Warning("Unable to create connection proxy: %+s", err)
			}
		}()
	}

	if !msg.Noexec {
		go func() {
			sbox.ready.Wait()
			wgNet.Wait()
			go sbox.launchProgram(d.config.PrefixPath, msg.Path, msg.Pwd, msg.Args, log)
		}()
	}

	if sbox.profile.XServer.Enabled {
		go func() {
			sbox.ready.Wait()
			go sbox.startXpraClient()
		}()
	}

	d.nextSboxId += 1
	d.sandboxes = append(d.sandboxes, sbox)
	return sbox, nil
}

func (d *daemonState) sanitizeGroups(p *oz.Profile, username string, gids []uint32) (map[string]uint32, error) {
	allowedGroups := d.config.DefaultGroups
	allowedGroups = append(allowedGroups, p.AllowedGroups...)
	if len(d.systemGroups) == 0 {
		if err := d.cacheSystemGroups(); err != nil {
			return nil, err
		}
	}
	groups := map[string]uint32{}
	for _, sg := range d.systemGroups {
		for _, gg := range allowedGroups {
			if sg.Name == gg {
				found := false
				for _, uname := range sg.Members {
					if uname == username {
						found = true
						break
					}
				}
				if !found {
					continue
				}
				d.log.Debug("Allowing user: %s (%d)", gg, sg.Gid)
				groups[sg.Name] = sg.Gid
				break
			}
		}
	}

	return groups, nil
}

func (sbox *Sandbox) launchProgram(binpath, cpath, pwd string, args []string, log *logging.Logger) {
	if sbox.profile.AllowFiles {
		sbox.whitelistArgumentFiles(binpath, pwd, args, log)
	}
	err := ozinit.RunProgram(sbox.addr, cpath, pwd, args)
	if err != nil {
		log.Error("start shell command failed: %v", err)
	}
}

func (sbox *Sandbox) MountFiles(files []string, readonly bool, binpath string, log *logging.Logger) error {
	pmnt := path.Join(binpath, "bin", "oz-mount")
	args := files
	if readonly {
		args = append([]string{"--readonly"}, files...)
	}
	cmnt := exec.Command(pmnt, args...)
	cmnt.Env = []string{
		"_OZ_NSPID=" + strconv.Itoa(sbox.init.Process.Pid),
		"_OZ_HOMEDIR=" + sbox.user.HomeDir,
	}
	log.Debug("Attempting to add file with %s to sandbox %s: %+s", pmnt, sbox.profile.Name, files)
	pout, err := cmnt.CombinedOutput()
	if err != nil || cmnt.ProcessState.Success() == false {
		log.Warning("Unable to bind files to sandbox: %s", string(pout))
		return fmt.Errorf("%s", string(pout[2:]))
	}
	for _, mfile := range files {
		found := false
		for _, mmfile := range sbox.mountedFiles {
			if mfile == mmfile {
				found = true
				break
			}
		}
		if !found {
			sbox.mountedFiles = append(sbox.mountedFiles, mfile)
		}
	}
	log.Info("%s", string(pout))
	return nil
}

func (sbox *Sandbox) UnmountFile(file, binpath string, log *logging.Logger) error {
	pmnt := path.Join(binpath, "bin", "oz-umount")
	cmnt := exec.Command(pmnt, file)
	cmnt.Env = []string{
		"_OZ_NSPID=" + strconv.Itoa(sbox.init.Process.Pid),
		"_OZ_HOMEDIR=" + sbox.user.HomeDir,
	}
	pout, err := cmnt.CombinedOutput()
	if err != nil || cmnt.ProcessState.Success() == false {
		log.Warning("Unable to unbind file from sandbox: %s", string(pout))
		return fmt.Errorf("%s", string(pout[2:]))
	}
	for i, item := range sbox.mountedFiles {
		if item == file {
			sbox.mountedFiles = append(sbox.mountedFiles[:i], sbox.mountedFiles[i+1:]...)
		}
	}
	log.Info("%s", string(pout))
	return nil
}

func (sbox *Sandbox) whitelistArgumentFiles(binpath, pwd string, args []string, log *logging.Logger) {
	var files []string
	for _, fpath := range args {
		if filepath.IsAbs(fpath) == false {
			fpath = path.Join(pwd, fpath)
		}
		if !strings.HasPrefix(fpath, "/home/") {
			continue
		}
		if _, err := os.Stat(fpath); err == nil {
			log.Notice("Adding file `%s` to sandbox `%s`.", fpath, sbox.profile.Name)
			files = append(files, fpath)
		}
	}
	if len(files) > 0 {
		sbox.MountFiles(files, false, binpath, log)
	}
}

func (sbox *Sandbox) remove(log *logging.Logger) {
	sboxes := []*Sandbox{}
	for _, sb := range sbox.daemon.sandboxes {
		if sb == sbox {
			//		sb.fs.Cleanup()
			if sb.profile.Networking.Nettype == network.TYPE_BRIDGE {
				sb.network.Cleanup(log)
			}
			os.Remove(sb.addr)
		} else {
			sboxes = append(sboxes, sb)
		}
	}
	sbox.daemon.sandboxes = sboxes
}

func (sbox *Sandbox) logMessages() {
	scanner := bufio.NewScanner(sbox.stderr)
	seenOk := false
	for scanner.Scan() {
		line := scanner.Text()
		if line == "OK" && !seenOk {
			sbox.daemon.log.Info("oz-init (%s) is ready", sbox.profile.Name)
			seenOk = true
			sbox.ready.Done()
		} else if len(line) > 1 {
			sbox.logLine(line)
		}
	}
	sbox.stderr.Close()
}

func (sbox *Sandbox) logLine(line string) {
	if len(line) < 2 {
		return
	}
	f := sbox.getLogFunc(line[0])
	msg := line[2:]
	if f != nil {
		f("[%s] %s", sbox.profile.Name, msg)
	} else {
		sbox.daemon.log.Info("[%s] %s", sbox.profile.Name, line)
	}
}

func (sbox *Sandbox) getLogFunc(c byte) func(string, ...interface{}) {
	log := sbox.daemon.log
	switch c {
	case 'D':
		return log.Debug
	case 'I':
		return log.Info
	case 'N':
		return log.Notice
	case 'W':
		return log.Warning
	case 'E':
		return log.Error
	case 'C':
		return log.Critical
	}
	return nil
}

func (sbox *Sandbox) startXpraClient() {
	u, err := user.LookupId(fmt.Sprintf("%d", sbox.cred.Uid))
	if err != nil {
		sbox.daemon.Error("Failed to lookup user for uid=%d, cannot start xpra", sbox.cred.Uid)
		return
	}
	xpraPath := path.Join(u.HomeDir, ".Xoz", sbox.profile.Name)
	sbox.xpra = xpra.NewClient(
		&sbox.profile.XServer,
		uint64(sbox.display),
		sbox.cred,
		xpraPath,
		sbox.profile.Name,
		sbox.daemon.log)

	if sbox.daemon.config.LogXpra {
		sbox.setupXpraLogging()
	}
	if err := sbox.xpra.Process.Start(); err != nil {
		sbox.daemon.Warning("Failed to start xpra client: %v", err)
	}
}

func (sbox *Sandbox) setupXpraLogging() {
	stdout, err := sbox.xpra.Process.StdoutPipe()
	if err != nil {
		sbox.daemon.Warning("Failed to create xpra stdout pipe: %v", err)
		return
	}
	stderr, err := sbox.xpra.Process.StderrPipe()
	if err != nil {
		stdout.Close()
		sbox.daemon.Warning("Failed to create xpra stderr pipe: %v", err)
	}
	go sbox.logPipeOutput(stdout, "xpra-stdout")
	go sbox.logPipeOutput(stderr, "xpra-stderr")
}

func (sbox *Sandbox) logPipeOutput(p io.Reader, label string) {
	scanner := bufio.NewScanner(p)
	for scanner.Scan() {
		line := scanner.Text()
		sbox.daemon.log.Info("(%s) %s", label, line)
	}
}
