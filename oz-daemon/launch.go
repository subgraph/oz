package daemon

import (
	"bufio"
	"fmt"
	"github.com/subgraph/oz"
	"github.com/subgraph/oz/fs"
	"github.com/subgraph/oz/xpra"
	"io"
	"os/exec"
	"os/user"
	"path"
	"sync"
	"syscall"
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
const initCloneFlags = syscall.CLONE_NEWNS | syscall.CLONE_NEWIPC | syscall.CLONE_NEWPID | syscall.CLONE_NEWUTS | syscall.CLONE_NEWNET

func createInitCommand(name, chroot string, uid uint32, display int) *exec.Cmd {
	cmd := exec.Command(initPath)
	cmd.Dir = "/"
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Chroot:     chroot,
		Cloneflags: initCloneFlags,
	}
	cmd.Env = []string{
		"INIT_PROFILE=" + name,
		fmt.Sprintf("INIT_UID=%d", uid),
	}
	if display > 0 {
		cmd.Env = append(cmd.Env, fmt.Sprintf("INIT_DISPLAY=%d", display))
	}
	return cmd
}

func (d *daemonState) launch(p *oz.Profile, uid, gid uint32) (*Sandbox, error) {
	u, err := user.LookupId(fmt.Sprintf("%d", uid))
	if err != nil {
		return nil, fmt.Errorf("failed to lookup user for uid=%d: %v", uid, err)
	}
	fs := fs.NewFromProfile(p, u, d.config.SandboxPath, d.log)
	if err := fs.Setup(); err != nil {
		return nil, err
	}
	display := 0
	if p.XServer.Enabled {
		display = d.nextDisplay
		d.nextDisplay += 1
	}

	cmd := createInitCommand(p.Name, fs.Root(), uid, display)
	pp, err := cmd.StderrPipe()
	if err != nil {
		fs.Cleanup()
		return nil, fmt.Errorf("error creating stderr pipe for init process: %v", err)

	}
	if err := cmd.Start(); err != nil {
		fs.Cleanup()
		return nil, err
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

func (sbox *Sandbox) remove() {
	sboxes := []*Sandbox{}
	for _, sb := range sbox.daemon.sandboxes {
		if sb == sbox {
			sb.fs.Cleanup()
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
