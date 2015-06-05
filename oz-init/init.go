package ozinit

import (
	"os"
	"github.com/codegangsta/cli"
	"github.com/subgraph/oz"
	"github.com/subgraph/oz/fs"
	"github.com/subgraph/oz/ipc"
	"os/exec"
	"syscall"
	"github.com/op/go-logging"
	"github.com/kr/pty"
	"fmt"
	"github.com/subgraph/oz/xpra"
	"os/user"
	"strconv"
	"path"
)

const profileDirectory = "/var/lib/oz/cells.d"

type initState struct {
	log *logging.Logger
	user *user.User
	fs *fs.Filesystem
}

//var log = createLogger()

// By convention oz-init writes log messages to stderr with a single character
// prefix indicating the logging level.  These messages are read one line at a time
// over a pipe by oz-daemon and translated into appropriate log events.
func createLogger() *logging.Logger {
	l := logging.MustGetLogger("oz-init")
	be := logging.NewLogBackend(os.Stderr, "", 0)
	f := logging.MustStringFormatter("%{level:.1s} %{message}")
	fbe := logging.NewBackendFormatter(be,f)
	logging.SetBackend(fbe)
	return l
}

var allowRootShell = false
var profileName = "none"

func Main() {
	app := cli.NewApp()
	app.Name = "oz-init"
	app.Action = runInit
	app.Flags = []cli.Flag {
		cli.StringFlag {
			Name: "address",
			Usage: "unix socket address to listen for commands on",
			EnvVar: "INIT_ADDRESS",
		},
		cli.StringFlag{
			Name: "profile",
			Usage: "name of profile to launch",
			EnvVar: "INIT_PROFILE",
		},
		cli.IntFlag{
			Name: "uid",
			EnvVar: "INIT_UID",
		},
	}
	app.Run(os.Args)
}

func runInit(c *cli.Context) {
	st := new(initState)
	st.log = createLogger()
	address := c.String("address")
	profile := c.String("profile")
	uid := uint32(c.Int("uid"))
	if address == "" {
		st.log.Error("Error: missing required 'address' argument")
		os.Exit(1)
	}
	if profile == "" {
		st.log.Error("Error: missing required 'profile' argument")
		os.Exit(1)
	}
	profileName = profile
	st.log.Info("Starting oz-init for profile: %s", profile)
	st.log.Info("Socket address: %s", address)
	p,err := loadProfile(profile)
	if err != nil {
		st.log.Error("Could not load profile %s: %v", profile, err)
		os.Exit(1)
	}
	u,err := user.LookupId(strconv.FormatUint(uint64(uid), 10))
	if err != nil {
		st.log.Error("Failed to lookup user with uid=%d: %v", uid, err)
		os.Exit(1)
	}
	st.user = u

	fs := fs.NewFromProfile(p, u, st.log)
	if err := fs.OzInit(); err != nil {
		st.log.Error("Error: setting up filesystem failed: %v\n", err)
		os.Exit(1)
	}
	st.fs = fs
	if p.XServer.Enabled {
		st.startXpraServer(&p.XServer, fs)
	}

	oz.ReapChildProcs(st.log, st.handleChildExit)

	serv := ipc.NewMsgConn(messageFactory, address)
	serv.AddHandlers(
		handlePing,
		st.handleRunShell,
	)
	serv.Listen()
	serv.Run()
	st.log.Info("oz-init exiting...")
}

func (is *initState) startXpraServer (config *oz.XServerConf,  fs *fs.Filesystem) {
	workdir := fs.Xpra()
	if workdir == "" {
		is.log.Warning("Xpra work directory not set")
		return
	}
	logpath := path.Join(workdir, "xpra-server.out")
	f,err := os.Create(logpath)
	if err != nil {
		is.log.Warning("Failed to open xpra logfile (%s): %v", logpath, err)
		return
	}
	defer f.Close()
	xpra := xpra.NewServer(config, 123, workdir)
	xpra.Process.Stdout = f
	xpra.Process.Stderr = f
	xpra.Process.Env = []string{
		"HOME="+ is.user.HomeDir,
	}
	is.log.Info("Starting xpra server")
	if err := xpra.Process.Start(); err != nil {
		is.log.Warning("Failed to start xpra server: %v", err)
	}
}

func loadProfile(name string) (*oz.Profile, error) {
	ps,err := oz.LoadProfiles(profileDirectory)
	if err != nil {
		return nil, err
	}
	for _,p := range ps {
		if name == p.Name {
			return p, nil
		}
	}
	return nil, fmt.Errorf("no profile named '%s'", name)
}

func handlePing(ping *PingMsg, msg *ipc.Message) error {
	return msg.Respond(&PingMsg{Data: ping.Data})
}

func (is *initState) handleRunShell(rs *RunShellMsg, msg *ipc.Message) error {
	if msg.Ucred == nil {
		return msg.Respond(&ErrorMsg{"No credentials received for RunShell command"})
	}
	if msg.Ucred.Uid == 0 || msg.Ucred.Gid == 0 && !allowRootShell {
		return msg.Respond(&ErrorMsg{"Cannot open shell because allowRootShell is disabled"})
	}
	is.log.Info("Starting shell with uid = %d, gid = %d", msg.Ucred.Uid, msg.Ucred.Gid)
	cmd := exec.Command("/bin/sh", "-i")
	cmd.SysProcAttr = &syscall.SysProcAttr{}
	cmd.SysProcAttr.Credential = &syscall.Credential{
		Uid: msg.Ucred.Uid,
		Gid: msg.Ucred.Gid,
	}
	if rs.Term != "" {
		cmd.Env = append(cmd.Env, "TERM="+rs.Term)
	}
	cmd.Env = append(cmd.Env, "PATH=/usr/bin:/bin")
	cmd.Env = append(cmd.Env, fmt.Sprintf("PS1=[%s] $ ", profileName))
	is.log.Info("Executing shell...")
	f,err := ptyStart(cmd)
	defer f.Close()
	if err != nil {
		return msg.Respond(&ErrorMsg{err.Error()})
	}
	err = msg.Respond(&OkMsg{}, int(f.Fd()))
	return err
}

func ptyStart(c *exec.Cmd) (ptty *os.File, err error) {
	ptty,tty, err := pty.Open()
	if err != nil {
		return nil, err
	}
	defer tty.Close()
	c.Stdin = tty
	c.Stdout = tty
	c.Stderr = tty
	if c.SysProcAttr == nil {
		c.SysProcAttr = &syscall.SysProcAttr{}
	}
	c.SysProcAttr.Setctty = true
	c.SysProcAttr.Setsid = true
	if err := c.Start(); err != nil {
		ptty.Close()
		return nil, err
	}
	return ptty, nil
}

func (is *initState) handleChildExit(pid int, wstatus syscall.WaitStatus) {
	is.log.Debug("Child process pid=%d exited with status %d", pid, wstatus.ExitStatus())
}

