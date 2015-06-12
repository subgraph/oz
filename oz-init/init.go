package ozinit

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"os/user"
	"strconv"
	"strings"
	"sync"
	"syscall"

	"github.com/subgraph/oz"
	"github.com/subgraph/oz/fs"
	"github.com/subgraph/oz/ipc"
	"github.com/subgraph/oz/network"
	"github.com/subgraph/oz/xpra"

	"github.com/kr/pty"
	"github.com/op/go-logging"
	"os/signal"
)

const SocketAddress = "/tmp/oz-init-control"
const EnvPrefix = "INIT_ENV_"

type initState struct {
	log       *logging.Logger
	profile   *oz.Profile
	config    *oz.Config
	launchEnv []string
	uid       int
	gid       int
	user      *user.User
	display   int
	fs        *fs.Filesystem
	xpraReady sync.WaitGroup
	network   *network.SandboxNetwork
}

// By convention oz-init writes log messages to stderr with a single character
// prefix indicating the logging level.  These messages are read one line at a time
// over a pipe by oz-daemon and translated into appropriate log events.
func createLogger() *logging.Logger {
	l := logging.MustGetLogger("oz-init")
	be := logging.NewLogBackend(os.Stderr, "", 0)
	f := logging.MustStringFormatter("%{level:.1s} %{message}")
	fbe := logging.NewBackendFormatter(be, f)
	logging.SetBackend(fbe)
	return l
}

func Main() {
	parseArgs().runInit()
}

func parseArgs() *initState {
	log := createLogger()
	getvar := func(name string) string {
		val := os.Getenv(name)
		if val == "" {
			log.Error("Error: missing required '%s' argument", name)
			os.Exit(1)
		}
		return val
	}
	pname := getvar("INIT_PROFILE")
	uidval := getvar("INIT_UID")
	dispval := os.Getenv("INIT_DISPLAY")

	stnip := os.Getenv("INIT_ADDR")
	stnvhost := os.Getenv("INIT_VHOST")
	stnvguest := os.Getenv("INIT_VGUEST")
	stngateway := os.Getenv("INIT_GATEWAY")

	var config *oz.Config
	config, err := oz.LoadConfig(oz.DefaultConfigPath)
	if err != nil {
		log.Info("Could not load config file (%s), using default config", oz.DefaultConfigPath)
		config = oz.NewDefaultConfig()
	}

	p, err := loadProfile(config.ProfileDir, pname)
	if err != nil {
		log.Error("Could not load profile %s: %v", pname, err)
		os.Exit(1)
	}
	uid, err := strconv.Atoi(uidval)
	if err != nil {
		log.Error("Could not parse INIT_UID argument (%s) into an integer: %v", uidval, err)
		os.Exit(1)
	}
	u, err := user.LookupId(uidval)
	if err != nil {
		log.Error("Failed to look up user with uid=%s: %v", uidval, err)
		os.Exit(1)
	}
	gid, err := strconv.Atoi(u.Gid)
	if err != nil {
		log.Error("Failed to parse gid value (%s) from user struct: %v", u.Gid, err)
		os.Exit(1)
	}
	display := 0
	if dispval != "" {
		d, err := strconv.Atoi(dispval)
		if err != nil {
			log.Error("Unable to parse display (%s) into an integer: %v", dispval, err)
			os.Exit(1)
		}
		display = d
	}

	stn := new(network.SandboxNetwork)
	if stnip != "" {
		gateway, _, err := net.ParseCIDR(stngateway)
		if err != nil {
			log.Error("Unable to parse network configuration gateway (%s): %v", stngateway, err)
			os.Exit(1)
		}

		stn.Ip = stnip
		stn.VethHost = stnvhost
		stn.VethGuest = stnvguest
		stn.Gateway = gateway
	}

	env := []string{}
	for _, e := range os.Environ() {
		if strings.HasPrefix(e, EnvPrefix) {
			e = e[len(EnvPrefix):]
			log.Debug("Adding (%s) to launch environment", e)
			env = append(env, e)
		}
	}

	return &initState{
		log:       log,
		config:    config,
		launchEnv: env,
		profile:   p,
		uid:       uid,
		gid:       gid,
		user:      u,
		display:   display,
		fs:        fs.NewFromProfile(p, u, config.SandboxPath, config.UseFullDev, log),
		network:   stn,
	}
}

func (st *initState) runInit() {
	st.log.Info("Starting oz-init for profile: %s", st.profile.Name)
	sigs := make(chan os.Signal)
	signal.Notify(sigs, syscall.SIGTERM, os.Interrupt)

	if st.profile.Networking.Nettype != "host" {
		err := network.NetSetup(st.network)
		if err != nil {
			st.log.Error("Unable to setup networking: %+v", err)
			os.Exit(1)
		}
	}
	network.NetPrint(st.log)

	if syscall.Sethostname([]byte(st.profile.Name)) != nil {
		st.log.Error("Failed to set hostname to (%s)", st.profile.Name)
	}
	st.log.Info("Hostname set to (%s)", st.profile.Name)

	if err := st.fs.OzInit(); err != nil {
		st.log.Error("Error: setting up filesystem failed: %v\n", err)
		os.Exit(1)
	}
	oz.ReapChildProcs(st.log, st.handleChildExit)

	if st.profile.XServer.Enabled {
		st.xpraReady.Add(1)
		st.startXpraServer()
	}
	st.xpraReady.Wait()
	st.launchApplication()

	s, err := ipc.NewServer(SocketAddress, messageFactory, st.log,
		handlePing,
		st.handleRunShell,
	)
	if err != nil {
		st.log.Error("NewServer failed: %v", err)
		os.Exit(1)
	}
	if err := os.Chown(SocketAddress, st.uid, st.gid); err != nil {
		st.log.Warning("Failed to chown oz-init control socket: %v", err)
	}
	os.Stderr.WriteString("OK\n")

	go st.processSignals(sigs, s)

	if err := s.Run(); err != nil {
		st.log.Warning("MsgServer.Run() return err: %v", err)
	}
	st.log.Info("oz-init exiting...")
}

func (st *initState) startXpraServer() {
	workdir := st.fs.Xpra()
	if workdir == "" {
		st.log.Warning("Xpra work directory not set")
		return
	}
	xpra := xpra.NewServer(&st.profile.XServer, uint64(st.display), workdir)
	p, err := xpra.Process.StderrPipe()
	if err != nil {
		st.log.Warning("Error creating stderr pipe for xpra output: %v", err)
		os.Exit(1)
	}
	go st.readXpraOutput(p)
	xpra.Process.Env = []string{
		"HOME=" + st.user.HomeDir,
	}
	xpra.Process.SysProcAttr = &syscall.SysProcAttr{}
	xpra.Process.SysProcAttr.Credential = &syscall.Credential{
		Uid: uint32(st.uid),
		Gid: uint32(st.gid),
	}
	st.log.Info("Starting xpra server")
	if err := xpra.Process.Start(); err != nil {
		st.log.Warning("Failed to start xpra server: %v", err)
	}
}

func (st *initState) readXpraOutput(r io.ReadCloser) {
	sc := bufio.NewScanner(r)
	seenReady := false
	for sc.Scan() {
		line := sc.Text()
		if len(line) > 0 {
			if strings.Contains(line, "xpra is ready.") && !seenReady {
				seenReady = true
				st.xpraReady.Done()
				if !st.config.LogXpra {
					r.Close()
					return
				}
			}
			if st.config.LogXpra {
				st.log.Debug("(xpra) %s", line)
			}
		}
	}
}

func (st *initState) launchApplication() {
	cmd := exec.Command(st.profile.Path)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		st.log.Warning("Failed to create stdout pipe: %v", err)
		return
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		st.log.Warning("Failed to create stderr pipe: %v", err)
		return
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{}
	cmd.SysProcAttr.Credential = &syscall.Credential{
		Uid: uint32(st.uid),
		Gid: uint32(st.gid),
	}
	cmd.Env = append(st.launchEnv,
		fmt.Sprintf("DISPLAY=:%d", st.display),
	)
	if err := cmd.Start(); err != nil {
		st.log.Warning("Failed to start application (%s): %v", st.profile.Path, err)
		return
	}
	go st.readApplicationOutput(stdout, "stdout")
	go st.readApplicationOutput(stderr, "stderr")
}

func (st *initState) readApplicationOutput(r io.ReadCloser, label string) {
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		line := sc.Text()
		st.log.Debug("(%s) %s", label, line)
	}
}

func loadProfile(dir, name string) (*oz.Profile, error) {
	ps, err := oz.LoadProfiles(dir)
	if err != nil {
		return nil, err
	}
	for _, p := range ps {
		if name == p.Name {
			return p, nil
		}
	}
	return nil, fmt.Errorf("no profile named '%s'", name)
}

func handlePing(ping *PingMsg, msg *ipc.Message) error {
	return msg.Respond(&PingMsg{Data: ping.Data})
}

func (st *initState) handleRunShell(rs *RunShellMsg, msg *ipc.Message) error {
	if msg.Ucred == nil {
		return msg.Respond(&ErrorMsg{"No credentials received for RunShell command"})
	}
	if (msg.Ucred.Uid == 0 || msg.Ucred.Gid == 0) && st.config.AllowRootShell != true {
		return msg.Respond(&ErrorMsg{"Cannot open shell because allowRootShell is disabled"})
	}
	st.log.Info("Starting shell with uid = %d, gid = %d", msg.Ucred.Uid, msg.Ucred.Gid)
	cmd := exec.Command(st.config.ShellPath, "-i")
	cmd.SysProcAttr = &syscall.SysProcAttr{}
	cmd.SysProcAttr.Credential = &syscall.Credential{
		Uid: msg.Ucred.Uid,
		Gid: msg.Ucred.Gid,
	}
	if rs.Term != "" {
		cmd.Env = append(cmd.Env, "TERM="+rs.Term)
	}
	if msg.Ucred.Uid != 0 && msg.Ucred.Gid != 0 {
		if homedir, _ := st.fs.GetHomeDir(); homedir != "" {
			cmd.Dir = homedir
			cmd.Env = append(cmd.Env, "HOME="+homedir)
		}
	}
	if st.profile.XServer.Enabled {
		cmd.Env = append(cmd.Env, "DISPLAY=:"+strconv.Itoa(st.display))
	}
	cmd.Env = append(cmd.Env, "PATH=/usr/bin:/bin")
	cmd.Env = append(cmd.Env, fmt.Sprintf("PS1=[%s] $ ", st.profile.Name))
	st.log.Info("Executing shell...")
	f, err := ptyStart(cmd)
	defer f.Close()
	if err != nil {
		return msg.Respond(&ErrorMsg{err.Error()})
	}
	err = msg.Respond(&OkMsg{}, int(f.Fd()))
	return err
}

func ptyStart(c *exec.Cmd) (ptty *os.File, err error) {
	ptty, tty, err := pty.Open()
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

func (st *initState) processSignals(c <-chan os.Signal, s *ipc.MsgServer) {
	for {
		sig := <-c
		st.log.Info("Recieved signal (%v)", sig)
		s.Close()
	}
}
