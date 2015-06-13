package daemon

import (
	"fmt"
	"os/user"
	"strings"
	"syscall"

	"github.com/subgraph/oz"
	"github.com/subgraph/oz/fs"
	"github.com/subgraph/oz/ipc"
	"github.com/subgraph/oz/network"

	"github.com/op/go-logging"
	"os"
)

type daemonState struct {
	log         *logging.Logger
	config      *oz.Config
	profiles    oz.Profiles
	sandboxes   []*Sandbox
	nextSboxId  int
	nextDisplay int
	memBackend  *logging.ChannelMemoryBackend
	backends    []logging.Backend
	network     *network.HostNetwork
}

func Main() {
	d := initialize()

	err := runServer(
		d.log,
		d.handlePing,
		d.handleListProfiles,
		d.handleLaunch,
		d.handleListSandboxes,
		d.handleClean,
		d.handleKillSandbox,
		d.handleLogs,
	)
	if err != nil {
		d.log.Warning("Error running server: %v", err)
	}
}

func initialize() *daemonState {
	d := &daemonState{}
	d.initializeLogging()
	var config *oz.Config
	config, err := oz.LoadConfig(oz.DefaultConfigPath)
	if err != nil {
		d.log.Info("Could not load config file (%s), using default config", oz.DefaultConfigPath)
		config = oz.NewDefaultConfig()
	}
	d.log.Info("Oz Global Config: %+v", config)
	d.config = config
	ps, err := oz.LoadProfiles(config.ProfileDir)
	if err != nil {
		d.log.Fatalf("Failed to load profiles: %v", err)
	}
	d.Debug("%d profiles loaded", len(ps))
	d.profiles = ps
	oz.ReapChildProcs(d.log, d.handleChildExit)
	d.nextSboxId = 1
	d.nextDisplay = 100

	for _, pp := range d.profiles {
		if pp.Networking.Nettype == "bridge" {
			d.log.Info("Initializing bridge networking")
			htn, err := network.BridgeInit(d.config.BridgeMACAddr, d.config.NMIgnoreFile, d.log)
			if err != nil {
				d.log.Fatalf("Failed to initialize bridge networking: %+v", err)
				return nil
			}

			d.network = htn

			network.NetPrint(d.log)

			break
		}
	}

	return d
}

func (d *daemonState) handleChildExit(pid int, wstatus syscall.WaitStatus) {
	d.Debug("Child process pid=%d exited with status %d", pid, wstatus.ExitStatus())

	for _, sbox := range d.sandboxes {
		if sbox.init.Process.Pid == pid {
			sbox.remove(d.log)
			return
		}
	}
	d.Notice("No sandbox found with oz-init pid = %d", pid)
}

func runServer(log *logging.Logger, args ...interface{}) error {
	s, err := ipc.NewServer(SocketName, messageFactory, log, args...)
	if err != nil {
		return err
	}
	return s.Run()
}

func (d *daemonState) handlePing(msg *PingMsg, m *ipc.Message) error {
	d.Debug("received ping with data [%s]", msg.Data)
	return m.Respond(&PingMsg{msg.Data})
}

func (d *daemonState) handleListProfiles(msg *ListProfilesMsg, m *ipc.Message) error {
	r := new(ListProfilesResp)
	index := 1
	for _, p := range d.profiles {
		r.Profiles = append(r.Profiles, Profile{Index: index, Name: p.Name, Path: p.Path})
		index += 1
	}
	return m.Respond(r)
}

func (d *daemonState) handleLaunch(msg *LaunchMsg, m *ipc.Message) error {
	d.Debug("Launch message received: %+v", msg)
	p, err := d.getProfileByIdxOrName(msg.Index, msg.Name)
	if err != nil {
		return m.Respond(&ErrorMsg{err.Error()})
	}
	d.Debug("Would launch %s", p.Name)
	env := d.sanitizeEnvironment(p, msg.Env)
	_, err = d.launch(p, env, m.Ucred.Uid, m.Ucred.Gid, d.log)
	if err != nil {
		d.Warning("launch of %s failed: %v", p.Name, err)
		return m.Respond(&ErrorMsg{err.Error()})
	}
	return m.Respond(&OkMsg{})
}

func (d *daemonState) sanitizeEnvironment(p *oz.Profile, oldEnv []string) ([]string) {
	newEnv := []string{}
	
	for _, EnvItem := range d.config.EnvironmentVars {
		for _, OldItem := range oldEnv {
			if strings.HasPrefix(OldItem, EnvItem+"=") {
				newEnv = append(newEnv, EnvItem+"="+strings.Replace(OldItem, EnvItem+"=", "", 1))

				break
			}
		}
	}

	for _, EnvItem := range p.Environment {
		if EnvItem.Value != "" {
			d.log.Info("Setting environment variable: %s=%s\n", EnvItem.Name, EnvItem.Value)

			newEnv = append(newEnv, EnvItem.Name+"="+EnvItem.Value)
		} else {
			for _, OldItem := range oldEnv {
				if strings.HasPrefix(OldItem, EnvItem.Name+"=") {
					NewValue := strings.Replace(OldItem, EnvItem.Name+"=", "", 1)
					newEnv = append(newEnv, EnvItem.Name+"="+NewValue)

					d.log.Info("Cloning environment variable: %s=%s\n", EnvItem.Name, NewValue)

					break
				}
			}
		}
	}
	
	return newEnv
}

func (d *daemonState) handleKillSandbox(msg *KillSandboxMsg, m *ipc.Message) error {
	sbox := d.sandboxById(msg.Id)
	if sbox == nil {
		return m.Respond(&ErrorMsg{fmt.Sprintf("no sandbox found with id = %d", msg.Id)})
	}
	if err := sbox.init.Process.Signal(os.Interrupt); err != nil {
		return m.Respond(&ErrorMsg{fmt.Sprintf("failed to send interrupt signal: %v", err)})
	}
	return m.Respond(&OkMsg{})
}

func (d *daemonState) sandboxById(id int) *Sandbox {
	for _, sb := range d.sandboxes {
		if sb.id == id {
			return sb
		}
	}
	return nil
}

func (d *daemonState) getProfileByIdxOrName(index int, name string) (*oz.Profile, error) {
	if len(name) == 0 {
		if index < 1 || index > len(d.profiles) {
			return nil, fmt.Errorf("not a valid profile index (%d)", index)
		}
		return d.profiles[index-1], nil
	}

	for _, p := range d.profiles {
		if p.Name == name {
			return p, nil
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
	p, err := d.getProfileByIdxOrName(clean.Index, clean.Name)
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
	u, _ := user.Current()
	fs := fs.NewFromProfile(p, u, d.config.SandboxPath, d.config.UseFullDev, d.log)
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
