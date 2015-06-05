package daemon

import (
	"fmt"

	"github.com/op/go-logging"
	"github.com/subgraph/oz"
	"github.com/subgraph/oz/ipc"
	"syscall"
	"github.com/subgraph/oz/fs"
	"os/user"
)

type daemonState struct {
	log *logging.Logger
	profiles oz.Profiles
	sandboxes []*Sandbox
	nextSboxId int
	nextDisplay int
	memBackend *logging.ChannelMemoryBackend
	backends []logging.Backend
}

func Main() {
	d := initialize()

	err := runServer(
		d.handlePing,
		d.handleListProfiles,
		d.handleLaunch,
		d.handleListSandboxes,
		d.handleClean,
		d.handleLogs,
	)
	if err != nil {
		d.log.Warning("Error running server: %v", err)
	}
}

func initialize() *daemonState {
	d := &daemonState{}
	d.initializeLogging()
	ps,err := oz.LoadProfiles("/var/lib/oz/cells.d")
	if err != nil {
		d.log.Fatalf("Failed to load profiles: %v", err)
	}
	d.Debug("%d profiles loaded", len(ps))
	d.profiles = ps
	oz.ReapChildProcs(d.log, d.handleChildExit)
	d.nextSboxId = 1
	d.nextDisplay = 100
	return d
}


func (d *daemonState) handleChildExit(pid int, wstatus syscall.WaitStatus) {
	d.Debug("Child process pid=%d exited with status %d", pid, wstatus.ExitStatus())

	for _,sbox := range d.sandboxes {
		if sbox.init.Process.Pid == pid {
			sbox.remove()
			return
		}
	}
	d.Notice("No sandbox found with oz-init pid = %d", pid)
}

func runServer(args ...interface{}) error {
	serv := ipc.NewMsgConn(messageFactory, SocketName)
	if err := serv.AddHandlers(args...); err != nil {
		return err
	}
	if err := serv.Listen(); err != nil {
		return err
	}
	return serv.Run()
}

func (d * daemonState) handlePing(msg *PingMsg, m *ipc.Message) error {
	d.Debug("received ping with data [%s]", msg.Data)
	return m.Respond(&PingMsg{msg.Data})
}

func (d * daemonState) handleListProfiles(msg *ListProfilesMsg, m *ipc.Message) error {
	r := new(ListProfilesResp)
	index := 1
	for _,p := range d.profiles {
		r.Profiles = append(r.Profiles, Profile{Index: index, Name: p.Name, Path: p.Path})
		index += 1
	}
	return m.Respond(r)
}

func (d *daemonState) handleLaunch(msg *LaunchMsg, m *ipc.Message) error {
	d.Debug("Launch message received: %+v", msg)
	p,err := d.getProfileByIdxOrName(msg.Index, msg.Name)
	if err != nil {
		return m.Respond(&ErrorMsg{err.Error()})
	}
	d.Debug("Would launch %s", p.Name)
	_,err = d.launch(p, m.Ucred.Uid)
	if err != nil {
		d.Warning("launch of %s failed: %v", p.Name, err)
		return m.Respond(&ErrorMsg{err.Error()})
	}
	return m.Respond(&OkMsg{})
}

func (d *daemonState) getProfileByIdxOrName(index int, name string) (*oz.Profile, error) {
	if len(name) == 0 {
		if index < 1 || index > len(d.profiles) {
			return nil, fmt.Errorf("not a valid profile index (%d)", index)
		}
		return d.profiles[index-1], nil
	}

	for _,p := range d.profiles {
		if p.Name == name {
			return p,nil
		}
	}
	return nil, fmt.Errorf("could not find profile name '%s'", name)
}

func (d *daemonState) handleListSandboxes(list *ListSandboxesMsg, msg *ipc.Message) error {
	r := new(ListSandboxesResp)
	for _, sb := range d.sandboxes {
		r.Sandboxes = append(r.Sandboxes, SandboxInfo{Id: sb.id, Address: sb.addr, Profile: sb.profile.Name})
	}
	return msg.Respond(r)
}

func (d *daemonState) handleClean(clean *CleanMsg, msg *ipc.Message) error {
	p,err := d.getProfileByIdxOrName(clean.Index, clean.Name)
	if err != nil {
		return msg.Respond(&ErrorMsg{err.Error()})
	}
	for _, sb := range d.sandboxes {
		if sb.profile.Name == p.Name {
			errmsg := fmt.Sprintf("Cannot clean profile '%s' because there are sandboxes running for this profile", p.Name)
			return msg.Respond(&ErrorMsg{errmsg})
		}
	}
	// XXX
	u,_ := user.Current()
	fs := fs.NewFromProfile(p, u, d.log)
	if err := fs.Cleanup(); err != nil {
		return msg.Respond(&ErrorMsg{err.Error()})
	}
	return msg.Respond(&OkMsg{})
}

func (d *daemonState) handleLogs(logs *LogsMsg, msg *ipc.Message) error {
	for n := d.memBackend.Head(); n != nil; n = n.Next() {
		s := n.Record.Formatted(0)
		msg.Respond(&LogData{Lines: []string{s}})
	}
	if logs.Follow {
		d.followLogs(msg)
		return nil
	}
	msg.Respond(&OkMsg{})
	return nil
}

