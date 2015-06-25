package xpra

import (
	"errors"
	"fmt"
	"github.com/subgraph/oz"
	"os"
	"os/exec"
	"os/user"
	"path"
	"strconv"
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

func (x *Xpra) Stop() ([]byte, error) {
	cmd := exec.Command("/usr/bin/xpra",
		"--socket-dir="+x.WorkDir,
		"stop",
		fmt.Sprintf(":%d", x.Display),
	)
	cmd.Env = []string{"TMPDIR=" + x.WorkDir}
	return cmd.Output()
}

func GetPath(u *user.User, name string) string {
	return path.Join(u.HomeDir, ".Xoz", name)
}

func CreateDir(u *user.User, name string) (string, error) {
	uid, gid, err := userIds(u)
	if err != nil {
		return "", err
	}
	dir := GetPath(u, name)
	if err := createSubdirs(u.HomeDir, uid, gid, 0755, ".Xoz", name); err != nil {
		return "", fmt.Errorf("failed to create xpra directory (%s): %v", dir, err)
	}
	return dir, nil
}

func createSubdirs(base string, uid, gid int, mode os.FileMode, subdirs ...string) error {
	dir := base
	for _, sd := range subdirs {
		dir = path.Join(dir, sd)
		if err := os.Mkdir(dir, mode); err != nil && !os.IsExist(err) {
			return err
		}
		if err := os.Chown(dir, uid, gid); err != nil {
			return err
		}
	}
	return nil
}

func userIds(user *user.User) (int, int, error) {
	uid, err := strconv.Atoi(user.Uid)
	if err != nil {
		return -1, -1, errors.New("failed to parse uid from user struct: " + err.Error())
	}
	gid, err := strconv.Atoi(user.Gid)
	if err != nil {
		return -1, -1, errors.New("failed to parse gid from user struct: " + err.Error())
	}
	return uid, gid, nil
}
