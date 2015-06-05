package xpra
import (
	"github.com/subgraph/oz"
	"fmt"
	"os"
	"github.com/op/go-logging"
	"os/exec"
	"syscall"
)

var xpraClientDefaultArgs = []string{
	"--title=@title@",
	"--compress=0",
	//"--delay-tray",
	//"--border=auto",
	"--no-keyboard-sync",
}

func NewClient(config *oz.XServerConf, display uint64, cred *syscall.Credential, workdir string, hostname string, log *logging.Logger) *Xpra {
	x := new(Xpra)
	x.Config = config
	x.Display = display
	x.WorkDir = workdir
	x.xpraArgs = prepareClientArgs(config, display, workdir, log)
	x.Process = exec.Command("/usr/bin/xpra", x.xpraArgs...)
	x.Process.SysProcAttr = &syscall.SysProcAttr{
		Credential: cred,
	}
	x.Process.Env = []string{
		"DISPLAY=:0",
		fmt.Sprintf("TMPDIR=%s", workdir),
		fmt.Sprintf("XPRA_SOCKET_HOSTNAME=%s", hostname),
	}
	return x
}

func prepareClientArgs(config *oz.XServerConf, display uint64, workdir string, log *logging.Logger) []string {
	args := getDefaultArgs(config)
	args = append(args, xpraClientDefaultArgs...)
	if !config.EnableTray {
		args = append(args, "--no-tray")
	}
	if exists(config.TrayIcon, "Tray icon", log) {
		args = append(args, fmt.Sprintf("--tray-icon=%s", config.TrayIcon))
	}
	if exists(config.WindowIcon, "Window icon", log) {
		args = append(args, fmt.Sprintf("--window-icon=%s", config.WindowIcon))
	}
	args = append(args,
		fmt.Sprintf("--socket-dir=%s", workdir),
		"attach",
		fmt.Sprintf(":%d", display),
	)
	return args
}

func exists(path,label string, log *logging.Logger) bool {
	if path == "" {
		return false
	}
	if _,err := os.Stat(path); err != nil {
		if !os.IsNotExist(err) {
			log.Notice("%s file missing at %s, ignored.", label, path)
		} else {
			log.Warning("Error reading file info for %s: %v", path, err)
		}
		return false
	}
	return true
}
