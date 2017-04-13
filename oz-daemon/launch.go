package daemon

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/url"
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
	"github.com/subgraph/oz/openvpn"
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
	waiting      sync.WaitGroup
	iface        *network.OzVeth
	mountedFiles []string
	rawEnv       []string
	forwarders   []ActiveForwarder
	ovpn         *OpenVPN
}

type OpenVPN struct {
	cmd      *exec.Cmd
	runtoken string
}

type ActiveForwarder struct {
	name string
	desc string
	dest string
}

func createPidfilePath(base, prefix string) (string, error) {
	bs := make([]byte, 8)
	_, err := rand.Read(bs)
	if err != nil {
		return "", err
	}

	return path.Join(base, fmt.Sprintf("%s-%s.pid", prefix, hex.EncodeToString(bs))), nil
}

func createRunToken(prefix string) (string, error) {
	bs := make([]byte, 8)
	_, err := rand.Read(bs)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s-%s", prefix, hex.EncodeToString(bs)), nil
}

func createSocketPath(base, prefix string) (string, error) {
	bs := make([]byte, 8)
	_, err := rand.Read(bs)
	if err != nil {
		return "", err
	}

	return path.Join(base, fmt.Sprintf("%s-%s", prefix, hex.EncodeToString(bs))), nil
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

	cmd.Env = []string{}

	return cmd
}

func (d *daemonState) launch(p *oz.Profile, msg *LaunchMsg, rawEnv []string, uid, gid uint32, log *logging.Logger) (*Sandbox, error) {

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

	socketPath, err := createSocketPath(path.Join(d.config.SandboxPath, "sockets"), "oz-init-control")
	if err != nil {
		return nil, fmt.Errorf("Failed to create random socket path: %v", err)
	}
	initPath := path.Join(d.config.PrefixPath, "bin", "oz-init")
	cmd := createInitCommand(initPath, (p.Networking.Nettype != network.TYPE_HOST))
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
		fs:      fs.NewFilesystem(d.config, log, u, p),
		//addr:    path.Join(rootfs, ozinit.SocketAddress),
		addr:   socketPath,
		stderr: pp,
		rawEnv: rawEnv,
	}

	sbox.ready.Add(1)
	sbox.waiting.Add(1)
	go sbox.logMessages()

	sbox.waiting.Wait()

	if p.Networking.Nettype == network.TYPE_BRIDGE {
		if err := sbox.configureBridgedIface(); err != nil {
			cmd.Process.Kill()
			return nil, fmt.Errorf("Unable to setup bridged networking: %+v", err)
		}
		if p.Networking.VPNConf.VpnType == "openvpn" {
			var ovpn OpenVPN
			ovpn.runtoken, err = createRunToken("openvpn")
			sbox.ovpn = &ovpn
			if err != nil {
				return nil, fmt.Errorf("Unable to create run token: %+v", err)
			}
			ovpn.cmd, err = sbox.startOpenVPN(ovpn.runtoken)
			if err != nil {
				return nil, fmt.Errorf("Unable to start VPN: %+v", err)
			}
			log.Info("VPN started, pid %n\n", ovpn.cmd.Process.Pid)
		}

	}
	cmd.Process.Signal(syscall.SIGUSR1)

	wgNet := new(sync.WaitGroup)
	if p.Networking.Nettype != network.TYPE_HOST &&
		p.Networking.Nettype != network.TYPE_NONE &&
		len(p.Networking.Sockets) > 0 {
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

func (sbox *Sandbox) startOpenVPN(runtoken string) (c *exec.Cmd, err error) {
	bname := "oz-" + sbox.getBridgeName()
	bip := sbox.iface.GetVethBridge().GetIP()
	rtable := fmt.Sprintf("%d", sbox.daemon.config.RouteTableBase+sbox.id)
	conf := sbox.profile.Networking.VPNConf.ConfigPath
	if conf == "" {
		sbox.daemon.log.Warning("OpenVPN Conf not specified for %s (id=%d)", sbox.profile.Name, sbox.id)
		return nil, err
	}
	authpath := sbox.profile.Networking.VPNConf.UserPassFilePath
	if authpath == "" {
		sbox.daemon.log.Warning("OpenVPN credential locations not specified for %s (id=%d)", sbox.profile.Name, sbox.id)
		return nil, err
	}
	return openvpn.StartOpenVPN(sbox.daemon.config, conf, bip, rtable, bname, authpath, runtoken)
}

func (sbox *Sandbox) configureBridgedIface() error {
	bname := sbox.getBridgeName()
	sbox.daemon.log.Infof("Configuring bridged networking on bridge '%s' for %s (id=%d)",
		bname, sbox.profile.Name, sbox.id)

	br, err := sbox.daemon.bridges.GetBridge(bname)
	if err != nil {
		return err
	}
	veth, err := br.NewVeth(sbox.id, sbox.init.Process.Pid)
	if err != nil {
		return err
	}
	if err := veth.Setup(); err != nil {
		veth.Delete()
		return err
	}
	sbox.iface = veth
	return nil
}

func (sbox *Sandbox) getBridgeName() string {
	if name := sbox.profile.Networking.Bridge; name != "" {
		return name
	}
	return "default"
}

func (sbox *Sandbox) launchProgram(binpath, cpath, pwd string, args []string, log *logging.Logger) {
	if sbox.profile.AllowFiles {
		sbox.whitelistArgumentFiles(binpath, pwd, args, log)
	}
	err := ozinit.RunProgram(sbox.addr, cpath, pwd, args)
	if err != nil {
		log.Error("run program command failed: %v", err)
		pid := sbox.init.Process.Pid
		err = syscall.Kill(pid, syscall.SIGTERM)

		if err == nil {
			log.Error("Attempted to self-destruct sandbox...")
		} else {
			log.Error("Attempt to kill sandbox failed: %v", err)
		}
	}
}

func (sbox *Sandbox) SetupDynamicForwarder(name, port string, log *logging.Logger) (desc string, e error) {
	// TODO: Put error checking here
	var lp oz.ExternalForwarder
	var f *os.File
	var fd uintptr
	dest := ""

	for _, l := range sbox.profile.ExternalForwarders {
		if l.Name == name {
			lp = l
			break
		}
	}
	if lp.ExtProto == "unix" {
		socketPath, err := createSocketPath(path.Join(sbox.daemon.config.SandboxPath, "sockets"), "oz-dynamic-listener")
		l, err := net.ListenUnix("unix", &net.UnixAddr{socketPath, "unix"})
		if err != nil {
			log.Warning("Socket creation failure: %+s", err)
			return "", err
		}
		if lp.SocketOwner != "" {
			u, err := user.Lookup(lp.SocketOwner)
			if err != nil {
				return "", fmt.Errorf("failed to lookup user for uid=%d: %v", u.Uid, err)
			}
			uid, err := strconv.Atoi(u.Uid)
			if err != nil {
				return "", err
			}
			err = syscall.Chown(socketPath, uid, 0)
			if err != nil {
				return "", fmt.Errorf("failed to set ownership of socket %s to uid %d: %v", socketPath, uid, err)
			}
		}

		f, err = l.File()
		if err != nil {
			log.Warning("File object access failed: %+s", err)
			return "", err
		}
		fd = f.Fd()
		desc = socketPath
	} else {
		return "", fmt.Errorf("unimplemented external protocol type: %s", lp.ExtProto)
	}

	if lp.Proto == "tcp" {
		if lp.TargetHost != "" {
			if lp.TargetHost != "127.0.0.1" {
				return "", fmt.Errorf("Unimplemented connectivity to %s\n", lp.TargetHost)
			}
			if lp.Dynamic {
				if port != "" {
					dest = lp.TargetHost + ":" + port
				} else {
					return "", fmt.Errorf("Port missing.")
				}
			} else {
				if lp.TargetPort != "" {
					dest = lp.TargetHost + ":" + lp.TargetPort
				} else {
					return "", fmt.Errorf("Port missing.")
				}
			}
		}
	} else {
		return "", fmt.Errorf("Unimplemented target protocol type %s\n", lp.Proto)
	}
	err := ozinit.SetupForwarder(sbox.addr, lp.Proto, dest, fd)
	if err != nil {
		log.Warning("Error setting up forwarder: %+s", err)
		return "", err
	}
	sbox.forwarders = append(sbox.forwarders, ActiveForwarder{name: name, desc: desc, dest: dest})
	/*
		if sbox.forwarders[name] != nil {
			sbox.forwarders[name] = append(sbox.forwarders[name], desc)
		} else {
			sbox.forwarders[name] = []string{desc}
		}
	*/
	return desc, nil
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
		if strings.HasPrefix(fpath, "file://") {
			fpath = strings.Replace(fpath, "file://", "", 1)
			fpath, _ = url.QueryUnescape(fpath)
		}
		if filepath.IsAbs(fpath) == false {
			fpath = path.Join(pwd, fpath)
		}
		if !strings.HasPrefix(fpath, sbox.user.HomeDir) && !strings.HasPrefix(fpath, "/media/user") {
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
			if sb.iface != nil {
				sb.iface.Delete()
				sb.iface = nil
			}
			//		sb.fs.Cleanup()
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
	seenWaiting := false
	for scanner.Scan() {
		line := scanner.Text()
		if line == "WAITING" && !seenWaiting {
			sbox.daemon.log.Info("oz-init (%s) is waiting for init", sbox.profile.Name)
			seenWaiting = true
			sbox.waiting.Done()
		} else if line == "OK" && !seenOk {
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
		path.Join(sbox.daemon.config.PrefixPath, "bin", "oz-seccomp"),
		xpraPath,
		sbox.profile.Name,
		sbox.daemon.log)

	sbox.xpra.Process.Env = append(sbox.rawEnv, sbox.xpra.Process.Env...)

	//sbox.daemon.log.Debug("%s %s", strings.Join(sbox.xpra.Process.Env, " "), strings.Join(sbox.xpra.Process.Args, " "))
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
	go sbox.logPipeOutput(stdout, "xpra-client-stdout")
	go sbox.logPipeOutput(stderr, "xpra-client-stderr")
}

func (sbox *Sandbox) logPipeOutput(p io.Reader, label string) {
	scanner := bufio.NewScanner(p)
	for scanner.Scan() {
		line := scanner.Text()
		sbox.daemon.log.Info("[%s] (%s) %s", sbox.profile.Name, label, line)
	}
}
