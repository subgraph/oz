package daemon

import (
	"fmt"
	"github.com/subgraph/oz/fs"
	"os"
	"path"
	"runtime"
	"syscall"
)

var basicBindDirs = []string{
	"/bin", "/lib", "/lib64", "/usr", "/etc",
}

var basicEmptyDirs = []string{
	"/sbin", "/var", "/var/lib",
	"/var/cache", "/home", "/boot",
	"/tmp", "/run", "/run/user",
	"/run/lock", "/root",
	"/opt", "/srv", "/dev", "/proc",
	"/sys", "/mnt", "/media",
	//"/run/shm",
}

var basicSymlinks = [][2]string{
	{"/run", "/var/run"},
	{"/tmp", "/var/tmp"},
	{"/run/lock", "/var/lock"},
	{"/dev/shm", "/run/shm"},
}

var deviceSymlinks = [][2]string{
	{"/proc/self/fd", "/dev/fd"},
	{"/proc/self/fd/2", "/dev/stderr"},
	{"/proc/self/fd/0", "/dev/stdin"},
	{"/proc/self/fd/1", "/dev/stdout"},
	{"/dev/pts/ptmx", "/dev/ptmx"},
}

var basicBlacklist = []string{
	"/usr/sbin", "/sbin", "${PATH}/su",
	"${PATH}/sudo", "${PATH}/fusermount",
	"${PATH}/xinput", "${PATH}/strace",
	"${PATH}/mount", "${PATH}/umount",
}

type fsDeviceDefinition struct {
	path string
	mode uint32
	dev  int
	perm uint32
}

const ugorw = syscall.S_IRUSR | syscall.S_IWUSR | syscall.S_IRGRP | syscall.S_IWGRP | syscall.S_IROTH | syscall.S_IWOTH
const urwgr = syscall.S_IRUSR | syscall.S_IWUSR | syscall.S_IRGRP
const urw = syscall.S_IRUSR | syscall.S_IWUSR

var basicDevices = []fsDeviceDefinition{
	{path: "/dev/full", mode: syscall.S_IFCHR | ugorw, dev: _makedev(1, 7), perm: 0666},
	{path: "/dev/null", mode: syscall.S_IFCHR | ugorw, dev: _makedev(1, 3), perm: 0666},
	{path: "/dev/random", mode: syscall.S_IFCHR | ugorw, dev: _makedev(1, 8), perm: 0666},

	{path: "/dev/console", mode: syscall.S_IFCHR | urw, dev: _makedev(5, 1), perm: 0600},
	{path: "/dev/tty", mode: syscall.S_IFCHR | ugorw, dev: _makedev(5, 0), perm: 0666},
	{path: "/dev/tty1", mode: syscall.S_IFREG | urwgr, dev: 0, perm: 0640},
	{path: "/dev/tty2", mode: syscall.S_IFREG | urwgr, dev: 0, perm: 0640},
	{path: "/dev/tty3", mode: syscall.S_IFREG | urwgr, dev: 0, perm: 0640},
	{path: "/dev/tty4", mode: syscall.S_IFREG | urwgr, dev: 0, perm: 0640},

	{path: "/dev/urandom", mode: syscall.S_IFCHR | ugorw, dev: _makedev(1, 9), perm: 0666},
	{path: "/dev/zero", mode: syscall.S_IFCHR | ugorw, dev: _makedev(1, 5), perm: 0666},
}

func _makedev(x, y int) int {
	return (((x) << 8) | (y))
}

func setupRootfs(fsys *fs.Filesystem) error {
	if err := os.MkdirAll(fsys.Root(), 0755); err != nil {
		return fmt.Errorf("could not create rootfs path '%s': %v", fsys.Root(), err)
	}
	// XXX It's possible this doesn't work.
	// see: https://github.com/golang/go/issues/1954

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if err := syscall.Unshare(syscall.CLONE_NEWNS); err != nil {
		return fmt.Errorf("could not unshare mount ns: %v", err)
	}
	if err := syscall.Mount("", "/", "", syscall.MS_PRIVATE|syscall.MS_REC, ""); err != nil {
		return fmt.Errorf("failed to set MS_PRIVATE on '%s': %v", "/", err)
	}

	flags := uintptr(syscall.MS_NOSUID | syscall.MS_NOEXEC | syscall.MS_NODEV)
	if err := syscall.Mount("", fsys.Root(), "tmpfs", flags, "mode=755,gid=0"); err != nil {
		return fmt.Errorf("failed to mount tmpfs on '%s': %v", fsys.Root(), err)
	}

	if err := syscall.Mount("", fsys.Root(), "", syscall.MS_PRIVATE, ""); err != nil {
		return fmt.Errorf("failed to set MS_PRIVATE on '%s': %v", fsys.Root(), err)
	}

	for _, p := range basicBindDirs {
		if err := fsys.BindPath(p, fs.BindReadOnly, nil); err != nil {
			return fmt.Errorf("failed to bind directory '%s': %v", p, err)
		}
	}

	for _, p := range basicEmptyDirs {
		if err := fsys.CreateEmptyDir(p); err != nil {
			return fmt.Errorf("failed to create empty directory '%s': %v", p, err)
		}
	}

	dp := path.Join(fsys.Root(), "dev")
	if err := syscall.Mount("", dp, "tmpfs", syscall.MS_NOSUID|syscall.MS_NOEXEC, "mode=755"); err != nil {
		return err

	}
	for _, d := range basicDevices {
		if err := fsys.CreateDevice(d.path, d.dev, d.mode, d.perm); err != nil {
			return err
		}
	}

	for _, sl := range append(basicSymlinks, deviceSymlinks...) {
		if err := fsys.CreateSymlink(sl[0], sl[1]); err != nil {
			return err
		}
	}

	if err := fsys.CreateBlacklistPaths(); err != nil {
		return err
	}

	for _, bl := range basicBlacklist {
		if err := fsys.BlacklistPath(bl, nil); err != nil {
			return err
		}
	}
	return nil
}
