package ns

import (
	"errors"
	"os"
	"path"
	"strconv"
	"syscall"
)

type Namespace struct {
	Path string
	Type uintptr
}

const (
	CLONE_NEWNS   = syscall.CLONE_NEWNS
	CLONE_NEWUTS  = syscall.CLONE_NEWUTS
	CLONE_NEWIPC  = syscall.CLONE_NEWIPC
	CLONE_NEWNET  = syscall.CLONE_NEWNET
	CLONE_NEWUSER = syscall.CLONE_NEWUSER
	CLONE_NEWPID  = syscall.CLONE_NEWPID
)

var (
	Types []Namespace
)

func init() {
	Types = []Namespace{
		Namespace{Path: "ns/user", Type: syscall.CLONE_NEWUSER},
		Namespace{Path: "ns/ipc", Type: syscall.CLONE_NEWIPC},
		Namespace{Path: "ns/uts", Type: syscall.CLONE_NEWUTS},
		Namespace{Path: "ns/net", Type: syscall.CLONE_NEWNET},
		Namespace{Path: "ns/pid", Type: syscall.CLONE_NEWPID},
		Namespace{Path: "ns/mnt", Type: syscall.CLONE_NEWNS},
	}
}

func Set(fd, nsType uintptr) error {
	_, _, err := syscall.Syscall(SYS_SETNS, uintptr(fd), uintptr(nsType), 0)
	if err != 0 {
		return errors.New("Unable to set namespace")
	}

	return nil
}

func GetPath(pid int, nsType uintptr) (string, error) {
	var nsPath string

	for _, n := range Types {
		if n.Type == nsType {
			nsPath = path.Join("/", "proc", strconv.Itoa(pid), n.Path)
			break
		}
	}

	if nsPath == "" {
		return "", errors.New("Unable to find namespace type")
	}

	return nsPath, nil
}

func OpenProcess(pid int, nsType uintptr) (uintptr, error) {
	nsPath, err := GetPath(pid, nsType)
	if err != nil {
		return 0, err
	}

	return Open(nsPath)
}

func Open(nsPath string) (uintptr, error) {
	fd, err := os.Open(nsPath)
	if err != nil {
		return 0, err
	}

	return fd.Fd(), nil
}

func Close(fd uintptr) error {
	return syscall.Close(int(fd))
}
