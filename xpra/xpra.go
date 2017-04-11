package xpra

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/user"
	"path"
	"strconv"
	"syscall"

	"github.com/subgraph/oz"
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
	"--cursors",
	"--encoding=rgb",
}

func getDefaultArgs(config *oz.XServerConf) []string {
	args := []string{}
	args = append(args, xpraDefaultArgs...)
	if config.DisableClipboard {
		args = append(args, "--no-clipboard")
	} else {
		args = append(args, "--clipboard")
	}

	// Temporarily disabled
	/*
		switch config.AudioMode {
		case oz.PROFILE_AUDIO_NONE, "":
	*/
	args = append(args, "--no-microphone", "--no-speaker")
	/*
		case oz.PROFILE_AUDIO_SPEAKER:
			args = append(args, "--no-microphone", "--speaker")
		case oz.PROFILE_AUDIO_FULL:
			args = append(args, "--microphone", "--speaker")
		}
	*/
	if config.EnableNotifications {
		args = append(args, "--notifications")
	} else {
		args = append(args, "--no-notifications")
	}

	return args
}

func (x *Xpra) Stop(cred *syscall.Credential) ([]byte, error) {
	cmd := exec.Command("/usr/bin/xpra",
		"--socket-dir="+x.WorkDir,
		"stop",
		fmt.Sprintf(":%d", x.Display),
	)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Credential: cred,
	}
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
	if err := createSubdirs(u.HomeDir, uid, gid, 0750, ".Xoz", name); err != nil {
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

func writeFakeProfile(cmd *exec.Cmd) error {
	pi, err := cmd.StdinPipe()
	if err != nil {
		return nil
	}
	emptyProfile := new(oz.Profile)
	emptyProfile.Seccomp.Mode = "blacklist"
	emptyProfile.Seccomp.Enforce = true
	jdata, err := json.Marshal(emptyProfile)
	if err != nil {
		return err
	}
	io.Copy(pi, bytes.NewBuffer(jdata))
	pi.Close()

	return nil
}
