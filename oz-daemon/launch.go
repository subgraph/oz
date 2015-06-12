package daemon

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"os/user"
	"path"
	"sync"
	"syscall"

	"github.com/subgraph/oz"
	"github.com/subgraph/oz/fs"
	"github.com/subgraph/oz/network"
	"github.com/subgraph/oz/xpra"

	"github.com/op/go-logging"
	"github.com/subgraph/oz/oz-init"
)

const initPath = "/usr/local/bin/oz-init"

type Sandbox struct {
	daemon  *daemonState
	id      int
	display int
	profile *oz.Profile
	init    *exec.Cmd
	cred    *syscall.Credential
	fs      *fs.Filesystem
	stderr  io.ReadCloser
	addr    string
	xpra    *xpra.Xpra
	ready   sync.WaitGroup
	network *network.SandboxNetwork
}

/*
func findSandbox(id int) *Sandbox {
	for _, sb := range sandboxes {
		if sb.id == id {
			return sb
		}
	}
	return nil
}
*/

func createInitCommand(name, chroot string, env []string, uid uint32, display int, stn *network.SandboxNetwork, nettype string) *exec.Cmd {
	cmd := exec.Command(initPath)
	cmd.Dir = "/"

	cloneFlags := uintptr(syscall.CLONE_NEWNS)
	cloneFlags |= syscall.CLONE_NEWIPC
	cloneFlags |= syscall.CLONE_NEWPID
	cloneFlags |= syscall.CLONE_NEWUTS

	if nettype != "host" {
		cloneFlags |= syscall.CLONE_NEWNET
	}

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Chroot:     chroot,
		Cloneflags: cloneFlags,
	}
	cmd.Env = []string{
		"INIT_PROFILE=" + name,
		fmt.Sprintf("INIT_UID=%d", uid),
	}

	if stn.Ip != "" {
		cmd.Env = append(cmd.Env, "INIT_ADDR="+stn.Ip)
		cmd.Env = append(cmd.Env, "INIT_VHOST="+stn.VethHost)
		cmd.Env = append(cmd.Env, "INIT_VGUEST="+stn.VethGuest)
		cmd.Env = append(cmd.Env, "INIT_GATEWAY="+stn.Gateway.String()+"/"+stn.Class)
	}

	cmd.Env = append(cmd.Env, fmt.Sprintf("INIT_DISPLAY=%d", display))

	for _, e := range env {
		cmd.Env = append(cmd.Env, ozinit.EnvPrefix+e)
	}

	return cmd
}

func (d *daemonState) launch(p *oz.Profile, env []string, uid, gid uint32, log *logging.Logger) (*Sandbox, error) {
	u, err := user.LookupId(fmt.Sprintf("%d", uid))
	if err != nil {
		return nil, fmt.Errorf("failed to lookup user for uid=%d: %v", uid, err)
	}
	fs := fs.NewFromProfile(p, u, d.config.SandboxPath, d.config.UseFullDev, d.log)
	if err := fs.Setup(d.config.ProfileDir); err != nil {
		return nil, err
	}
	display := 0
	if p.XServer.Enabled && p.Networking.Nettype == "host" {
		display = d.nextDisplay
		d.nextDisplay += 1
	}

	stn := new(network.SandboxNetwork)
	if p.Networking.Nettype == "bridge" {
		stn, err = network.PrepareSandboxNetwork(d.network, log)
		if err != nil {
			return nil, fmt.Errorf("Unable to prepare veth network: %+v", err)
		}
	}

	cmd := createInitCommand(p.Name, fs.Root(), env, uid, display, stn, p.Networking.Nettype)
	log.Debug("Command environment: %+v", cmd.Env)
	pp, err := cmd.StderrPipe()
	if err != nil {
		fs.Cleanup()
		return nil, fmt.Errorf("error creating stderr pipe for init process: %v", err)

	}

	if err := cmd.Start(); err != nil {
		fs.Cleanup()
		return nil, fmt.Errorf("Unable to start process: %+v", err)
	}

	sbox := &Sandbox{
		daemon:  d,
		id:      d.nextSboxId,
		display: display,
		profile: p,
		init:    cmd,
		cred:    &syscall.Credential{Uid: uid, Gid: gid},
		fs:      fs,
		addr:    path.Join(fs.Root(), "tmp", "oz-init-control"),
		stderr:  pp,
		network: stn,
	}

	if p.Networking.Nettype == "bridge" {
		if err := network.NetInit(stn, d.network, cmd.Process.Pid, log); err != nil {
			cmd.Process.Kill()
			fs.Cleanup()
			return nil, fmt.Errorf("Unable to create veth networking: %+v", err)
		}
	}

	sbox.ready.Add(1)
	go sbox.logMessages()
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

func (sbox *Sandbox) remove(log *logging.Logger) {
	sboxes := []*Sandbox{}
	for _, sb := range sbox.daemon.sandboxes {
		if sb == sbox {
			sb.fs.Cleanup()
			if sb.profile.Networking.Nettype == "bridge" {
				sb.network.Cleanup(log)
			}
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
	sbox.xpra = xpra.NewClient(
		&sbox.profile.XServer,
		uint64(sbox.display),
		sbox.cred,
		sbox.fs.Xpra(),
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
