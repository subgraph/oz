package ozinit

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"os/signal"
	"os/user"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"
	//"time"

	"github.com/subgraph/oz"
	"github.com/subgraph/oz/fs"
	"github.com/subgraph/oz/ipc"
	"github.com/subgraph/oz/network"
	"github.com/subgraph/oz/xpra"

	"github.com/kr/pty"
	"github.com/op/go-logging"
)

type procState struct {
	cmd   *exec.Cmd
	track bool
}

type initState struct {
	log               *logging.Logger
	profile           *oz.Profile
	config            *oz.Config
	sockaddr          string
	launchEnv         []string
	lock              sync.Mutex
	children          map[int]procState
	uid               uint32
	gid               uint32
	gids              map[string]uint32
	user              *user.User
	display           int
	fs                *fs.Filesystem
	ipcServer         *ipc.MsgServer
	xpra              *xpra.Xpra
	xpraReady         sync.WaitGroup
	network           *network.SandboxNetwork
	dbusUuid          string
	shutdownRequested bool
}

type InitData struct {
	Profile   oz.Profile
	Config    oz.Config
	Sockaddr  string
	LaunchEnv []string
	Uid       uint32
	Gid       uint32
	Gids      map[string]uint32
	User      user.User
	Network   network.SandboxNetwork
	Display   int
}

const (
	DBUS_VAR_REGEXP = "[A-Za-z_]+=[a-zA-Z_:-@]+=/tmp/.+"
)

var dbusValidVar = regexp.MustCompile(DBUS_VAR_REGEXP)

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
	parseArgs().waitForParentReady().runInit()
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

	initData := new(InitData)
	if err := json.NewDecoder(os.Stdin).Decode(&initData); err != nil {
		log.Error("unable to decode init data: %v", err)
		os.Exit(1)
	}
	log.Debug("Init state: %+v", initData)

	if (initData.User.Uid != strconv.Itoa(int(initData.Uid))) || (initData.Uid == 0) {
		log.Error("invalid uid or user passed to init.")
		os.Exit(1)
	}

	env := []string{}
	env = append(env, initData.LaunchEnv...)
	env = append(env, "PATH=/usr/bin:/bin")

	if initData.Profile.XServer.Enabled {
		env = append(env, "DISPLAY=:"+strconv.Itoa(initData.Display))
	}

	return &initState{
		log:       log,
		config:    &initData.Config,
		sockaddr:  initData.Sockaddr,
		launchEnv: env,
		profile:   &initData.Profile,
		children:  make(map[int]procState),
		uid:       initData.Uid,
		gid:       initData.Gid,
		gids:      initData.Gids,
		user:      &initData.User,
		display:   initData.Display,
		fs:        fs.NewFilesystem(&initData.Config, log),
		network:   &initData.Network,
	}
}

func (st *initState) waitForParentReady() *initState {
	// Signal the daemon we are ready
	os.Stderr.WriteString("WAITING\n")

	c := make(chan os.Signal)
	signal.Notify(c, syscall.SIGUSR1)

	sig := <-c
	st.log.Info("Recieved SIGUSR1 from parent (%v), ready to init.", sig)
	signal.Stop(c)

	return st
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

	if err := os.Chown(st.sockaddr, int(st.uid), int(st.gid)); err != nil {
		st.log.Warning("Failed to chown oz-init control socket: %v", err)
	}

	wlExtras := []oz.WhitelistItem{}
	blExtras := []oz.BlacklistItem{}
	//wlExtras = append(wlExtras, oz.WhitelistItem{Path: "/etc/oz/mimeapps.list", Target: "${HOME}/.config/mimeapps.list", ReadOnly: true})
	//wlExtras = append(wlExtras, oz.WhitelistItem{Path: "/etc/oz/mimeapps.list", Target: "/etc/gnome/defaults.list", Force: true, ReadOnly: true})
	//blExtras = append(blExtras, oz.BlacklistItem{Path: "/etc/shadow"})
	//blExtras = append(blExtras, oz.BlacklistItem{Path: "/etc/shadow-"})

	if st.profile.XServer.AudioMode == oz.PROFILE_AUDIO_PULSE {
		wlExtras = append(wlExtras, oz.WhitelistItem{Path: "/run/user/${UID}/pulse/native", Ignore: true})
		wlExtras = append(wlExtras, oz.WhitelistItem{Path: "${HOME}/.config/pulse/cookie", Ignore: true, ReadOnly: true})
		wlExtras = append(wlExtras, oz.WhitelistItem{Path: "/dev/shm/pulse-shm-*", Ignore: true})
	}

	if err := st.setupFilesystem(wlExtras, blExtras); err != nil {
		st.log.Error("Failed to setup filesytem: %v", err)
		os.Exit(1)
	}

	if st.user != nil && st.user.HomeDir != "" {
		st.launchEnv = append(st.launchEnv, "HOME="+st.user.HomeDir)
	}

	if st.profile.Networking.Nettype != network.TYPE_HOST ||
		st.profile.Networking.Nettype != network.TYPE_NONE {
		err := network.NetSetup(st.network)
		if err != nil {
			st.log.Error("Unable to setup networking: %+v", err)
			os.Exit(1)
		}
	}
	network.NetPrint(st.log)

	if syscall.Sethostname([]byte(st.profile.Name)) != nil {
		st.log.Error("Failed to set hostname to (%s)", st.profile.Name)
		os.Exit(1)
	}
	if syscall.Setdomainname([]byte("local")) != nil {
		st.log.Error("Failed to set domainname")
	}
	st.log.Info("Hostname set to (%s.local)", st.profile.Name)

	if err := st.setupDbus(); err != nil {
		st.log.Error("Unable to setup dbus: %v", err)
		os.Exit(1)
	}

	oz.ReapChildProcs(st.log, st.handleChildExit)

	if st.profile.XServer.Enabled {
		st.xpraReady.Add(1)
		st.startXpraServer()
		st.xpraReady.Wait()
		st.log.Info("XPRA started")
	}

	if st.needsDbus() {
		if err := st.getDbusSession(); err != nil {
			st.log.Error("Unable to get dbus session information: %v", err)
			os.Exit(1)
		}
	}

	fsbx := path.Join("/tmp", "oz-sandbox")
	err = ioutil.WriteFile(fsbx, []byte(st.profile.Name), 0644)

	// Signal the daemon we are ready
	os.Stderr.WriteString("OK\n")

	go st.processSignals(sigs, s)

	st.ipcServer = s

	if err := s.Run(); err != nil {
		st.log.Warning("MsgServer.Run() return err: %v", err)
	}
	st.log.Info("oz-init exiting...")
}

func (st *initState) needsDbus() bool {
	return (st.profile.XServer.AudioMode == oz.PROFILE_AUDIO_FULL ||
		st.profile.XServer.AudioMode == oz.PROFILE_AUDIO_SPEAKER ||
		st.profile.XServer.EnableNotifications == true)
}

func (st *initState) setupDbus() error {
	exec.Command("/usr/bin/dbus-uuidgen", "--ensure").Run()
	buuid, err := exec.Command("/usr/bin/dbus-uuidgen", "--get").CombinedOutput()
	if err != nil || string(buuid) == "" {
		return fmt.Errorf("dbus-uuidgen failed: %v %v", err, string(buuid))
	}
	st.dbusUuid = strings.TrimSpace(string(bytes.Trim(buuid, "\x00")))
	st.log.Debug("dbus-uuid: %s", st.dbusUuid)
	return nil
}

func (st *initState) getDbusSession() error {
	args := []string{
		"--autolaunch",
		st.dbusUuid,
		"--sh-syntax",
		"--close-stderr",
	}
	dcmd := exec.Command("/usr/bin/dbus-launch", args...)
	dcmd.Env = append([]string{}, st.launchEnv...)
	//st.log.Debug("%s /usr/bin/dbus-launch %s", strings.Join(dcmd.Env, " "), strings.Join(args, " "))
	dcmd.SysProcAttr = &syscall.SysProcAttr{}
	dcmd.SysProcAttr.Credential = &syscall.Credential{
		Uid: st.uid,
		Gid: st.gid,
	}

	benvs, err := dcmd.Output()
	if err != nil && len(benvs) <= 1 {
		return fmt.Errorf("dbus-launch failed: %v %v", err, string(benvs))
	}
	benvs = bytes.Trim(benvs, "\x00")
	senvs := strings.TrimSpace(string(benvs))
	senvs = strings.Replace(senvs, "export ", "", -1)
	senvs = strings.Replace(senvs, ";", "", -1)
	senvs = strings.Replace(senvs, "'", "", -1)
	dbusenv := ""
	for _, line := range strings.Split(senvs, "\n") {
		if dbusValidVar.MatchString(line) {
			dbusenv = line
			break
		}
	}
	if dbusenv != "" {
		st.launchEnv = append(st.launchEnv, dbusenv)
		vv := strings.Split(dbusenv, "=")
		os.Setenv(vv[0], strings.Join(vv[1:], "="))
	}
	return nil
}

func (st *initState) startXpraServer() {
	if st.user == nil {
		st.log.Warning("Cannot start xpra server because no user is set")
		return
	}
	workdir := path.Join(st.user.HomeDir, ".Xoz", st.profile.Name)
	st.log.Info("xpra work dir is %s", workdir)
	spath := path.Join(st.config.PrefixPath, "bin", "oz-seccomp")
	xpra := xpra.NewServer(&st.profile.XServer, uint64(st.display), spath, workdir)
	//st.log.Debug("%s %s", strings.Join(xpra.Process.Env, " "), strings.Join(xpra.Process.Args, " "))
	if xpra == nil {
		st.log.Error("Error creating xpra server command")
		os.Exit(1)
	}
	p, err := xpra.Process.StderrPipe()
	if err != nil {
		st.log.Error("Error creating stderr pipe for xpra output: %v", err)
		os.Exit(1)
	}
	go st.readXpraOutput(p)
	xpra.Process.Env = []string{
		"HOME=" + st.user.HomeDir,
	}

	groups := append([]uint32{}, st.gid)
	if gid, gexists := st.gids["video"]; gexists {
		groups = append(groups, gid)
	}
	if st.profile.XServer.AudioMode != oz.PROFILE_AUDIO_NONE {
		if gid, gexists := st.gids["audio"]; gexists {
			groups = append(groups, gid)
		}
	}

	xpra.Process.SysProcAttr = &syscall.SysProcAttr{}
	xpra.Process.SysProcAttr.Credential = &syscall.Credential{
		Uid:    st.uid,
		Gid:    st.gid,
		Groups: groups,
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
			//if strings.Contains(line, "_OZ_XXSTARTEDXX") &&
			//	strings.Contains(line, "has terminated") && !seenReady {
			if strings.Contains(line, "xpra is ready.") && !seenReady {
				seenReady = true
				st.xpraReady.Done()
				if !st.config.LogXpra {
					r.Close()
					return
				}
			}
			if st.config.LogXpra {
				st.log.Debug("(xpra-server) %s", line)
			}
		}
	}
}

func (st *initState) launchApplication(cpath, pwd string, cmdArgs []string) (*exec.Cmd, error) {
	if cpath == "" {
		cpath = st.profile.Path
	}
	if st.config.DivertSuffix != "" {
		cpath += "." + st.config.DivertSuffix
	}
	if st.config.DivertPath {
		cpath = path.Join(path.Dir(cpath)+"-oz", path.Base(cpath))
	}
	if len(st.profile.DefaultParams) > 0 {
		cmdArgs = append(st.profile.DefaultParams, cmdArgs...)
	}

	switch st.profile.Seccomp.Mode {
	case oz.PROFILE_SECCOMP_TRAIN:
		st.log.Notice("Enabling seccomp training mode for : %s", cpath)
		spath := path.Join(st.config.PrefixPath, "bin", "oz-seccomp")
		cmdArgs = append([]string{spath, "-mode=whitelist", cpath}, cmdArgs...)
		cpath = path.Join(st.config.PrefixPath, "bin", "oz-seccomp-tracer")
	case oz.PROFILE_SECCOMP_WHITELIST:
		st.log.Notice("Enabling seccomp whitelist for: %s", cpath)
		if st.profile.Seccomp.Enforce == false {
			spath := path.Join(st.config.PrefixPath, "bin", "oz-seccomp")
			cmdArgs = append([]string{spath, "-mode=whitelist", cpath}, cmdArgs...)
			cpath = path.Join(st.config.PrefixPath, "bin", "oz-seccomp-tracer")
		} else {
			cmdArgs = append([]string{"-mode=whitelist", cpath}, cmdArgs...)
			cpath = path.Join(st.config.PrefixPath, "bin", "oz-seccomp")
		}
	case oz.PROFILE_SECCOMP_BLACKLIST:
		st.log.Notice("Enabling seccomp blacklist for: %s", cpath)
		if st.profile.Seccomp.Enforce == false {
			spath := path.Join(st.config.PrefixPath, "bin", "oz-seccomp")
			cmdArgs = append([]string{spath, "-mode=blacklist", cpath}, cmdArgs...)
			cpath = path.Join(st.config.PrefixPath, "bin", "oz-seccomp-tracer")
		} else {
			cmdArgs = append([]string{"-mode=blacklist", cpath}, cmdArgs...)
			cpath = path.Join(st.config.PrefixPath, "bin", "oz-seccomp")
		}
	}

	cmd := exec.Command(cpath)
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
	groups := append([]uint32{}, st.gid)
	for _, gid := range st.gids {
		groups = append(groups, gid)
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{}
	cmd.SysProcAttr.Credential = &syscall.Credential{
		Uid:    st.uid,
		Gid:    st.gid,
		Groups: groups,
	}
	cmd.Env = append(cmd.Env, st.launchEnv...)

	if st.profile.Seccomp.Mode == oz.PROFILE_SECCOMP_WHITELIST ||
		st.profile.Seccomp.Mode == oz.PROFILE_SECCOMP_BLACKLIST || st.profile.Seccomp.Mode == oz.PROFILE_SECCOMP_TRAIN {
		pi, err := cmd.StdinPipe()
		if err != nil {
			return nil, fmt.Errorf("error creating stdin pipe for seccomp process: %v", err)
		}
		jdata, err := json.Marshal(st.profile)
		if err != nil {
			return nil, fmt.Errorf("Unable to marshal seccomp state: %+v", err)
		}
		io.Copy(pi, bytes.NewBuffer(jdata))
		pi.Close()
	}

	cmd.Args = append(cmd.Args, cmdArgs...)

	if pwd == "" {
		pwd = st.user.HomeDir
	}
	if _, err := os.Stat(pwd); err == nil {
		cmd.Dir = pwd
	}

	if err := cmd.Start(); err != nil {
		st.log.Warning("Failed to start application (%s): %v", st.profile.Path, err)
		return nil, err
	}
	st.addChildProcess(cmd, true)

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
	groups := append([]uint32{}, st.gid)
	if msg.Ucred.Uid != 0 && msg.Ucred.Gid != 0 {
		for _, gid := range st.gids {
			groups = append(groups, gid)
		}
	}
	st.log.Info("Starting shell with uid = %d, gid = %d", msg.Ucred.Uid, msg.Ucred.Gid)
	cmd := exec.Command(st.config.ShellPath, "-i")
	cmd.SysProcAttr = &syscall.SysProcAttr{}
	cmd.SysProcAttr.Credential = &syscall.Credential{
		Uid:    msg.Ucred.Uid,
		Gid:    msg.Ucred.Gid,
		Groups: groups,
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
	st.addChildProcess(cmd, false)
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

func (st *initState) addChildProcess(cmd *exec.Cmd, track bool) {
	st.lock.Lock()
	defer st.lock.Unlock()
	st.children[cmd.Process.Pid] = procState{cmd: cmd, track: track}
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
	st.log.Debug("Child process pid=%d exited from init with status %d", pid, wstatus.ExitStatus())
	track := st.children[pid].track
	st.removeChildProcess(pid)

	for _, proc := range st.children {
		if proc.track {
			return
		}
	}

	if len(st.profile.Watchdog) > 0 {
		//if st.getProcessExists(st.profile.Watchdog) {
		//	return
		//} else {
		//	var ww sync.WaitGroup
		//	ww.Add(1)
		//	time.AfterFunc(time.Second*5, func() {
		//		ww.Done()
		//		st.log.Info("Watchdog timeout expired")
		//	})
		//	ww.Wait()
		track = !st.getProcessExists(st.profile.Watchdog)
		//}
	}
	if track == true && st.profile.AutoShutdown == oz.PROFILE_SHUTDOWN_YES {
		st.log.Info("Shutting down sandbox after child exit.")
		st.shutdown()
	}
}

func (st *initState) getProcessExists(pnames []string) bool {
	paths, _ := filepath.Glob("/proc/[0-9]*/cmdline")
	for _, path := range paths {
		pr, err := ioutil.ReadFile(path)
		if err != nil {
			continue
		}
		prs := []byte{}
		for _, prb := range pr {
			if prb == 0x00 {
				break
			}
			prs = append(prs, prb)
		}
		cmdb := filepath.Base(string(prs))
		if cmdb == "." {
			continue
		}
		for _, pname := range pnames {
			if pname == cmdb {
				return true
			}
		}
	}
	return false
}

func (st *initState) processSignals(c <-chan os.Signal, s *ipc.MsgServer) {
	for {
		sig := <-c
		st.log.Info("Recieved signal (%v)", sig)
		st.shutdown()
	}
}

func (st *initState) shutdown() {
	if st.shutdownRequested {
		return
	}
	st.shutdownRequested = true
	for _, c := range st.childrenVector() {
		c.cmd.Process.Signal(os.Interrupt)
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
	creds := &syscall.Credential{
		Uid: uint32(st.uid),
		Gid: uint32(st.gid),
	}
	out, err := st.xpra.Stop(creds)
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

func (st *initState) childrenVector() []procState {
	st.lock.Lock()
	defer st.lock.Unlock()
	cs := make([]procState, 0, len(st.children))
	for _, v := range st.children {
		cs = append(cs, v)
	}
	return cs
}

func (st *initState) setupFilesystem(extra_whitelist []oz.WhitelistItem, extra_blacklist []oz.BlacklistItem) error {

	fs := fs.NewFilesystem(st.config, st.log)

	if err := setupRootfs(fs, st.user, st.uid, st.gid, st.display, st.config.UseFullDev, st.log); err != nil {
		return err
	}

	if err := st.bindWhitelist(fs, extra_whitelist); err != nil {
		return err
	}

	if err := st.bindWhitelist(fs, st.profile.Whitelist); err != nil {
		return err
	}

	if err := st.applyBlacklist(fs, extra_blacklist); err != nil {
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
		if err := fs.BindPath(xprapath, 0, st.display, nil); err != nil {
			return err
		}
	}

	if err := fs.Chroot(); err != nil {
		return err
	}

	mo := &mountOps{}
	if st.config.UseFullDev {
		mo.add(fs.MountFullDev, fs.MountShm)
	}
	mo.add( /*fs.MountTmp, */ fs.MountPts)
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
		flags := 0
		if wl.CanCreate {
			flags |= fs.BindCanCreate
		}
		if wl.Ignore {
			flags |= fs.BindIgnore
		}
		if wl.ReadOnly {
			flags |= fs.BindReadOnly
		}
		if wl.Force {
			flags |= fs.BindForce
		}
		if wl.NoFollow {
			flags |= fs.BindNoFollow
		}
		if wl.Path == "" {
			continue
		}
		if err := fsys.BindTo(wl.Path, wl.Target, flags, st.display, st.user); err != nil {
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
		if bl.Path == "" {
			continue
		}
		if err := fsys.BlacklistPath(bl.Path, st.display, st.user); err != nil {
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
