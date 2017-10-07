package xpra

import (
	"crypto/md5"
	"fmt"
	"github.com/subgraph/oz"
	"io"
	"os"
	"os/exec"
	"syscall"

	"github.com/op/go-logging"
)

var xpraClientDefaultArgs = []string{
	"--title=@title@",
	"--compress=0",
	//"--delay-tray",
	//"--border=auto",
	"--no-keyboard-sync",
}

func NewClient(config *oz.XServerConf, display uint64, cred *syscall.Credential, spath, workdir, hostname string, log *logging.Logger) *Xpra {
	x := new(Xpra)
	x.Config = config
	x.Display = display
	x.WorkDir = workdir
	x.xpraArgs = prepareClientArgs(config, display, workdir, log)

	x.xpraArgs = append([]string{"-mode=blacklist", "/usr/bin/xpra"}, x.xpraArgs...)

	x.Process = exec.Command(spath, x.xpraArgs...)
	x.Process.SysProcAttr = &syscall.SysProcAttr{
		Credential: cred,
	}
	x.Process.Env = []string{
		"DISPLAY=:0",
		"XPRA_CLIPBOARD_LIMIT=45",
		"XPRA_CLIPBOARDS=CLIPBOARD",
		fmt.Sprintf("TMPDIR=%s", workdir),
		fmt.Sprintf("XPRA_SOCKET_HOSTNAME=%s", hostname),
	}

	/* Inject optional environment variables for XServer from profile XServer config */

	for _, EnvItem := range config.Environment {
		if EnvItem.Name != "" {
			if EnvItem.Value != "" {
				log.Info("Setting XServerConfig environment variable: %s=%s\n", EnvItem.Name, EnvItem.Value)
				x.Process.Env = append(x.Process.Env, EnvItem.Name+"="+EnvItem.Value)
			}
		}
	}

	if err := writeFakeProfile(x.Process); err != nil {
		return nil
	}

	return x
}

func prepareClientArgs(config *oz.XServerConf, display uint64, workdir string, log *logging.Logger) []string {
	args := getDefaultArgs(config)
	args = append(args, xpraClientDefaultArgs...)
	if !config.EnableTray {
		args = append(args, "--no-tray")
	} else {
		args = append(args, "--tray")
		if exists(config.TrayIcon, "Tray icon", log) {
			args = append(args, fmt.Sprintf("--tray-icon=%s", config.TrayIcon))
		}
	}
	if exists(config.WindowIcon, "Window icon", log) {
		args = append(args, fmt.Sprintf("--window-icon=%s", config.WindowIcon))
	}
	if config.Border {
		h := md5.New()
		io.WriteString(h, workdir)
		args = append(args, "--border=#"+fmt.Sprintf("%x", h.Sum(nil)[0:3]))
	}
	args = append(args,
		fmt.Sprintf("--socket-dir=%s", workdir),
		"attach",
		fmt.Sprintf(":%d", display),
	)
	return args
}

func exists(path, label string, log *logging.Logger) bool {
	if path == "" {
		return false
	}
	if _, err := os.Stat(path); err != nil {
		if !os.IsNotExist(err) {
			log.Notice("%s file missing at %s, ignored.", label, path)
		} else {
			log.Warning("Error reading file info for %s: %v", path, err)
		}
		return false
	}
	return true
}
