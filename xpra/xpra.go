package xpra

import (
	"github.com/subgraph/oz"
	"os/exec"
)

type Xpra struct {
	// Server working directory where unix socket is created
	WorkDir string

	Config *oz.XServerConf

	// Running xpra process
	Process *exec.Cmd

	// Display number
	Display uint64

	// Arguments passed to xpra command
	xpraArgs []string
}

var xpraDefaultArgs = []string{
	"--no-daemon",
	"--mmap",
	"--no-sharing",
	"--bell",
	"--system-tray",
	"--xsettings",
	//"--no-xsettings",
	"--notifications",
	"--cursors",
	"--encoding=rgb",
	"--no-pulseaudio",
}

func getDefaultArgs(config *oz.XServerConf) []string {
	args := []string{}
	args = append(args, xpraDefaultArgs...)
	if config.DisableClipboard {
		args = append(args, "--no-clipboard")
	} else {
		args = append(args, "--clipboard")
	}

	if config.DisableAudio {
		args = append(args, "--no-microphone", "--no-speaker")
	} else {
		args = append(args, "--microphone", "--speaker")
	}

	return args
}
