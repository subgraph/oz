package daemon
import (
	"github.com/subgraph/oz"
	"github.com/subgraph/oz/fs"
	"os/exec"
	"github.com/subgraph/oz/ipc"
	"syscall"
	"fmt"
	"io"
	"bufio"
	"os/user"
)

const initPath = "/usr/local/bin/oz-init"


type Sandbox struct {
	daemon *daemonState
	id int
	profile *oz.Profile
	init *exec.Cmd
	fs *fs.Filesystem
	stderr io.ReadCloser
	addr string
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
const initCloneFlags = syscall.CLONE_NEWNS | syscall.CLONE_NEWIPC | syscall.CLONE_NEWPID | syscall.CLONE_NEWUTS

func createInitCommand(addr, name, chroot string, uid uint32) *exec.Cmd {
	cmd := exec.Command(initPath)
	cmd.Dir = "/"
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Chroot: chroot,
		Cloneflags: initCloneFlags,
	}
	cmd.Env = []string{
		"INIT_ADDRESS="+addr,
		"INIT_PROFILE="+name,
		fmt.Sprintf("INIT_UID=%d", uid),
	}
	return cmd
}

func (d *daemonState) launch(p *oz.Profile, uid uint32) (*Sandbox, error) {
	u,err := user.LookupId(fmt.Sprintf("%d", uid))
	if err != nil {
		return nil, fmt.Errorf("failed to lookup user for uid=%d: %v", uid, err)
	}
	fs := fs.NewFromProfile(p, u, d.log)
	if err := fs.Setup(); err != nil {
		return nil, err
	}
	addr,err := ipc.CreateRandomAddress("@oz-init-")
	if err != nil {
		return nil, err
	}
	cmd := createInitCommand(addr, p.Name, fs.Root(), uid)
	pp,err := cmd.StderrPipe()
	if err != nil {
		fs.Cleanup()
		return nil, fmt.Errorf("error creating stderr pipe for init process: %v", err)

	}
	if err := cmd.Start(); err != nil {
		fs.Cleanup()
		return nil, err
	}
	sbox := &Sandbox{
		daemon: d,
		id: d.nextSboxId,
		profile: p,
		init: cmd,
		fs: fs,
		addr: addr,
		stderr: pp,
	}
	go sbox.logMessages()
	d.nextSboxId += 1
	d.sandboxes = append(d.sandboxes, sbox)
	return sbox,nil
}

func (sbox *Sandbox) remove() {
	sboxes := []*Sandbox{}
	for _,sb := range sbox.daemon.sandboxes {
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
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) > 1 {
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
	switch(c) {
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
