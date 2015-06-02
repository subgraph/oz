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
)

const profileDirectory = "/var/lib/oz/cells.d"

var log = createLogger()

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

	}
	app.Run(os.Args)
}

func runInit(c *cli.Context) {
	address := c.String("address")
	profile := c.String("profile")
	if address == "" {
		log.Error("Error: missing required 'address' argument")
		os.Exit(1)
	}
	if profile == "" {
		log.Error("Error: missing required 'profile' argument")
		os.Exit(1)
	}
	profileName = profile
	log.Info("Starting oz-init for profile: %s", profile)
	log.Info("Socket address: %s", address)
	p,err := loadProfile(profile)
	if err != nil {
		log.Error("Could not load profile %s: %v", profile, err)
		os.Exit(1)
	}

	fs := fs.NewFromProfile(p, log)
	if err := fs.OzInit(); err != nil {
		log.Error("Error: setting up filesystem failed: %v\n", err)
		os.Exit(1)
	}

	oz.ReapChildProcs(log, handleChildExit)

	serv := ipc.NewMsgConn(messageFactory, address)
	serv.AddHandlers(
		handlePing,
		handleRunShell,
	)
	serv.Listen()
	serv.Run()
	log.Info("oz-init exiting...")
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

func handleRunShell(rs *RunShellMsg, msg *ipc.Message) error {
	if msg.Ucred == nil {
		return msg.Respond(&ErrorMsg{"No credentials received for RunShell command"})
	}
	if msg.Ucred.Uid == 0 || msg.Ucred.Gid == 0 && !allowRootShell {
		return msg.Respond(&ErrorMsg{"Cannot open shell because allowRootShell is disabled"})
	}
	log.Info("Starting shell with uid = %d, gid = %d", msg.Ucred.Uid, msg.Ucred.Gid)
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
	log.Info("Executing shell...")
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

func handleChildExit(pid int, wstatus syscall.WaitStatus) {
	log.Debug("Child process pid=%d exited with status %d", pid, wstatus.ExitStatus())
}

