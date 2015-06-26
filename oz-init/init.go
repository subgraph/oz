package ozinit

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"os/signal"
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
	"path"
)

const EnvPrefix = "INIT_ENV_"

type initState struct {
	log       *logging.Logger
	profile   *oz.Profile
	config    *oz.Config
	sockaddr  string
	launchEnv []string
	lock      sync.Mutex
	children  map[int]*exec.Cmd
	uid       int
	gid       int
	user      *user.User
	display   int
	fs        *fs.Filesystem
	ipcServer *ipc.MsgServer
	xpra      *xpra.Xpra
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

	if os.Getuid() != 0 {
		log.Error("oz-init must run as root\n")
		os.Exit(1)
	}

	if os.Getpid() != 1 {
		log.Error("oz-init must be launched in new pid namespace.")
		os.Exit(1)
	}

	getvar := func(name string) string {
		val := os.Getenv(name)
		if val == "" {
			log.Error("Error: missing required '%s' argument", name)
			os.Exit(1)
		}
		return val
	}
	pname := getvar("INIT_PROFILE")
	sockaddr := getvar("INIT_SOCKET")
	uidval := getvar("INIT_UID")
	dispval := os.Getenv("INIT_DISPLAY")

	stnip := os.Getenv("INIT_ADDR")
	stnvhost := os.Getenv("INIT_VHOST")
	stnvguest := os.Getenv("INIT_VGUEST")
	stngateway := os.Getenv("INIT_GATEWAY")

	var config *oz.Config
	config, err := oz.LoadConfig(oz.DefaultConfigPath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Info("Configuration file (%s) is missing, using defaults.", oz.DefaultConfigPath)
			config = oz.NewDefaultConfig()
		} else {
			log.Error("Could not load configuration: %s", oz.DefaultConfigPath, err)
			os.Exit(1)
		}
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

	env = append(env, "PATH=/usr/bin:/bin")

	if p.XServer.Enabled {
		env = append(env, "DISPLAY=:"+strconv.Itoa(display))
	}

	return &initState{
		log:       log,
		config:    config,
		sockaddr:  sockaddr,
		launchEnv: env,
		profile:   p,
		children:  make(map[int]*exec.Cmd),
		uid:       uid,
		gid:       gid,
		user:      u,
		display:   display,
		fs:        fs.NewFilesystem(config, log),
		network:   stn,
	}
}

func (st *initState) runInit() {
	st.log.Info("Starting oz-init for profile: %s", st.profile.Name)
	sigs := make(chan os.Signal)
	signal.Notify(sigs, syscall.SIGTERM, os.Interrupt)

	s, err := ipc.NewServer(st.sockaddr, messageFactory, st.log,
		handlePing,
		st.handleRunProgram,
		st.handleRunShell,
	)
	if err != nil {
		st.log.Error("NewServer failed: %v", err)
		os.Exit(1)
	}

	if err := os.Chown(st.sockaddr, st.uid, st.gid); err != nil {
		st.log.Warning("Failed to chown oz-init control socket: %v", err)
	}

	if err := st.setupFilesystem(nil); err != nil {
		st.log.Error("Failed to setup filesytem: %v", err)
		os.Exit(1)
	}

	if st.user != nil && st.user.HomeDir != "" {
		st.launchEnv = append(st.launchEnv, "HOME="+st.user.HomeDir)
	}

	if st.profile.Networking.Nettype != network.TYPE_HOST {
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
	if syscall.Setdomainname([]byte("local")) != nil {
		st.log.Error("Failed to set domainname")
	}
	st.log.Info("Hostname set to (%s.local)", st.profile.Name)

	oz.ReapChildProcs(st.log, st.handleChildExit)

	if st.profile.XServer.Enabled {
		st.xpraReady.Add(1)
		st.startXpraServer()
	}
	st.xpraReady.Wait()
	st.log.Info("XPRA started")

	os.Stderr.WriteString("OK\n")

	go st.processSignals(sigs, s)

	st.ipcServer = s

	if err := s.Run(); err != nil {
		st.log.Warning("MsgServer.Run() return err: %v", err)
	}
	st.log.Info("oz-init exiting...")
}

func (st *initState) startXpraServer() {
	if st.user == nil {
		st.log.Warning("Cannot start xpra server because no user is set")
		return
	}
	workdir := path.Join(st.user.HomeDir, ".Xoz", st.profile.Name)
	st.log.Info("xpra work dir is %s", workdir)
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
		st.xpraReady.Done()
	}
	st.xpra = xpra
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

func (st *initState) launchApplication(cpath, pwd string, cmdArgs []string) (*exec.Cmd, error) {
	suffix := ""
	if st.config.DivertSuffix != "" {
		suffix = "." + st.config.DivertSuffix
	}
	if cpath == "" {
		cpath = st.profile.Path
	}
	cmd := exec.Command(cpath + suffix)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		st.log.Warning("Failed to create stdout pipe: %v", err)
		return nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		st.log.Warning("Failed to create stderr pipe: %v", err)
		return nil, err
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{}
	cmd.SysProcAttr.Credential = &syscall.Credential{
		Uid: uint32(st.uid),
		Gid: uint32(st.gid),
	}
	cmd.Env = append(cmd.Env, st.launchEnv...)

	cmd.Args = append(cmd.Args, cmdArgs...)

	if _, err := os.Stat(pwd); err == nil {
		cmd.Dir = pwd
	}

	if err := cmd.Start(); err != nil {
		st.log.Warning("Failed to start application (%s): %v", st.profile.Path, err)
		return nil, err
	}
	st.addChildProcess(cmd)

	go st.readApplicationOutput(stdout, "stdout")
	go st.readApplicationOutput(stderr, "stderr")

	return cmd, nil
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

func (st *initState) handleRunProgram(rp *RunProgramMsg, msg *ipc.Message) error {
	st.log.Info("Run program message received: %+v", rp)
	_, err := st.launchApplication(rp.Path, rp.Pwd, rp.Args)
	if err != nil {
		err := msg.Respond(&ErrorMsg{Msg: err.Error()})
		return err
	} else {
		err := msg.Respond(&OkMsg{})
		return err
	}
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
	cmd.Env = append(cmd.Env, st.launchEnv...)
	if rs.Term != "" {
		cmd.Env = append(cmd.Env, "TERM="+rs.Term)
	}
	if msg.Ucred.Uid != 0 && msg.Ucred.Gid != 0 {
		if st.user != nil && st.user.HomeDir != "" {
			cmd.Dir = st.user.HomeDir
		}
	}
	cmd.Env = append(cmd.Env, fmt.Sprintf("PS1=[%s] $ ", st.profile.Name))
	st.log.Info("Executing shell...")
	f, err := ptyStart(cmd)
	defer f.Close()
	if err != nil {
		return msg.Respond(&ErrorMsg{err.Error()})
	}
	st.addChildProcess(cmd)
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

func (st *initState) addChildProcess(cmd *exec.Cmd) {
	st.lock.Lock()
	defer st.lock.Unlock()
	st.children[cmd.Process.Pid] = cmd
}

func (st *initState) removeChildProcess(pid int) bool {
	st.lock.Lock()
	defer st.lock.Unlock()
	if _, ok := st.children[pid]; ok {
		delete(st.children, pid)
		return true
	}
	return false
}

func (st *initState) handleChildExit(pid int, wstatus syscall.WaitStatus) {
	st.log.Debug("Child process pid=%d exited with status %d", pid, wstatus.ExitStatus())
	st.removeChildProcess(pid)
}

func (st *initState) processSignals(c <-chan os.Signal, s *ipc.MsgServer) {
	for {
		sig := <-c
		st.log.Info("Recieved signal (%v)", sig)
		st.shutdown()
	}
}

func (st *initState) shutdown() {
	for _, c := range st.childrenVector() {
		c.Process.Signal(os.Interrupt)
	}

	st.shutdownXpra()

	if st.ipcServer != nil {
		st.ipcServer.Close()
	}
}

func (st *initState) shutdownXpra() {
	if st.xpra == nil {
		return
	}
	out, err := st.xpra.Stop()
	if err != nil {
		st.log.Warning("Error running xpra stop: %v", err)
		return
	}

	for _, line := range strings.Split(string(out), "\n") {
		if len(line) > 0 {
			st.log.Debug("(xpra stop) %s", line)
		}
	}
}

func (st *initState) childrenVector() []*exec.Cmd {
	st.lock.Lock()
	defer st.lock.Unlock()
	cs := make([]*exec.Cmd, 0, len(st.children))
	for _, v := range st.children {
		cs = append(cs, v)
	}
	return cs
}

func (st *initState) setupFilesystem(extra []oz.WhitelistItem) error {

	fs := fs.NewFilesystem(st.config, st.log)

	if err := setupRootfs(fs); err != nil {
		return err
	}

	if err := st.bindWhitelist(fs, st.profile.Whitelist); err != nil {
		return err
	}

	if err := st.bindWhitelist(fs, extra); err != nil {
		return err
	}

	if err := st.applyBlacklist(fs, st.profile.Blacklist); err != nil {
		return err
	}

	if st.profile.XServer.Enabled {
		xprapath, err := xpra.CreateDir(st.user, st.profile.Name)
		if err != nil {
			return err
		}
		if err := fs.BindPath(xprapath, 0, nil); err != nil {
			return err
		}
	}

	if err := fs.Chroot(); err != nil {
		return err
	}

	mo := &mountOps{}
	if st.config.UseFullDev {
		mo.add(fs.MountFullDev)
	}
	mo.add(fs.MountShm, fs.MountTmp, fs.MountPts)
	if !st.profile.NoSysProc {
		mo.add(fs.MountProc, fs.MountSys)
	}
	return mo.run()
}

func (st *initState) bindWhitelist(fsys *fs.Filesystem, wlist []oz.WhitelistItem) error {
	if wlist == nil {
		return nil
	}
	for _, wl := range wlist {
		flags := fs.BindCanCreate
		if wl.ReadOnly {
			flags |= fs.BindReadOnly
		}
		if err := fsys.BindPath(wl.Path, flags, st.user); err != nil {
			return err
		}
	}
	return nil
}

func (st *initState) applyBlacklist(fsys *fs.Filesystem, blist []oz.BlacklistItem) error {
	if blist == nil {
		return nil
	}
	for _, bl := range blist {
		if err := fsys.BlacklistPath(bl.Path, st.user); err != nil {
			return err
		}
	}
	return nil
}

type mountOps struct {
	ops []func() error
}

func (mo *mountOps) add(f ...func() error) {
	mo.ops = append(mo.ops, f...)
}

func (mo *mountOps) run() error {
	for _, f := range mo.ops {
		if err := f(); err != nil {
			return err
		}
	}
	return nil
}
