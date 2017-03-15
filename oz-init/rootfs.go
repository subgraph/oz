package ozinit

import (
	"fmt"
	"os"
	"os/user"
	"path"
	"strconv"
	"syscall"

	"github.com/naegelejd/go-acl"
	"github.com/op/go-logging"

	"github.com/subgraph/oz/fs"
)

var basicBindDirs = []string{
	"/bin", "/lib", "/lib64", "/usr", "/etc",
}

var basicEmptyDirs = []string{
	"/boot", "/dev", "/home", "/media", "/mnt",
	"/opt", "/proc", "/root", "/run", "/run/lock", "/run/user",
	"/sbin", "/srv", "/sys", "/tmp", "/var", "/var/lib", "/var/lib/dbus",
	"/var/cache", "/var/crash", "/run/resolvconf",
}

var basicEmptyUserDirs = []string{
	"/run/dbus",
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
	/*"${PATH}/dbus-daemon", "${PATH}/dbus-launch", "${PATH}/pulseaudio",*/
	"/usr/lib/gvfs",

	"/usr/sbin", "/sbin",

	"/etc/machine-id", "/etc/shadow", "/etc/shadow-", "/etc/fstab",
	"${PATH}/sudo", "${PATH}/su",
	"${PATH}/xinput", "${PATH}/strace",
	"${PATH}/mount", "${PATH}/umount",
	"${PATH}/fusermount",
}

var basicWhiteList = []string{
	"${HOME}/.config/mimeapps.list",
}

/*	"/etc/X11",*/

type fsDeviceDefinition struct {
	path string
	mode uint32
	dev  int
	gid  int
}

const ugorw = syscall.S_IRUSR | syscall.S_IWUSR | syscall.S_IRGRP | syscall.S_IWGRP | syscall.S_IROTH | syscall.S_IWOTH
const urwgr = syscall.S_IRUSR | syscall.S_IWUSR | syscall.S_IRGRP
const urwgw = syscall.S_IRUSR | syscall.S_IWUSR | syscall.S_IWGRP
const urw = syscall.S_IRUSR | syscall.S_IWUSR

var basicDevices = []fsDeviceDefinition{
	{path: "/dev/full", mode: syscall.S_IFCHR | ugorw, dev: _makedev(1, 7)},
	{path: "/dev/null", mode: syscall.S_IFCHR | ugorw, dev: _makedev(1, 3)},
	{path: "/dev/random", mode: syscall.S_IFCHR | ugorw, dev: _makedev(1, 8)},

	{path: "/dev/console", mode: syscall.S_IFCHR | urw, dev: _makedev(5, 1)},
	{path: "/dev/tty", mode: syscall.S_IFCHR | ugorw, dev: _makedev(5, 0), gid: 5},
	{path: "/dev/tty0", mode: syscall.S_IFCHR | urwgw, dev: _makedev(4, 0), gid: 5},
	{path: "/dev/tty1", mode: syscall.S_IFCHR | urwgw, dev: _makedev(4, 1), gid: 5},
	{path: "/dev/tty2", mode: syscall.S_IFCHR | urwgw, dev: _makedev(4, 2), gid: 5},
	{path: "/dev/tty3", mode: syscall.S_IFCHR | urwgw, dev: _makedev(4, 3), gid: 5},
	{path: "/dev/tty4", mode: syscall.S_IFCHR | urwgw, dev: _makedev(4, 4), gid: 5},
	{path: "/dev/tty5", mode: syscall.S_IFCHR | urwgw, dev: _makedev(4, 5), gid: 5},

	{path: "/dev/urandom", mode: syscall.S_IFCHR | ugorw, dev: _makedev(1, 9)},
	{path: "/dev/zero", mode: syscall.S_IFCHR | ugorw, dev: _makedev(1, 5)},
}

func _makedev(x, y int) int {
	return (((x) << 8) | (y))
}

func setupRootfs(fsys *fs.Filesystem, user *user.User, uid, gid uint32, display int, useFullDev bool, log *logging.Logger) error {
	if err := os.MkdirAll(fsys.Root(), 0755); err != nil {
		return fmt.Errorf("could not create rootfs path '%s': %v", fsys.Root(), err)
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
		if err := fsys.BindPath(p, fs.BindReadOnly, display); err != nil {
			return fmt.Errorf("failed to bind directory '%s': %v", p, err)
		}
	}

	userMountDir := path.Join("/media", user.Username)
	basicEmptyDirs = append(basicEmptyDirs, userMountDir)
	for _, p := range basicEmptyDirs {
		//log.Debug("Creating empty dir: %s", p)
		if err := fsys.CreateEmptyDir(p); err != nil {
			return fmt.Errorf("failed to create empty directory '%s': %v", p, err)
		}
	}

	if err := setupMountDirectory(fsys, userMountDir); err != nil {
		return fmt.Errorf("failed to create mount directory: %v", err)
	}

	basicEmptyUserDirs = append(basicEmptyUserDirs, user.HomeDir)
	for _, p := range basicEmptyUserDirs {
		//log.Debug("Creating empty user dir: %s", p)
		if err := fsys.CreateEmptyDir(p); err != nil {
			return fmt.Errorf("failed to create empty user directory '%s': %v", p, err)
		}
		if err := os.Chown(path.Join(fsys.Root(), p), int(uid), int(gid)); err != nil {
			return fmt.Errorf("failed to chown user dir: %v", err)
		}
	}

	rup := path.Join(fsys.Root(), "/run/user", strconv.FormatUint(uint64(uid), 10))
	if err := os.MkdirAll(rup, 0700); err != nil {
		return fmt.Errorf("failed to create user rundir: %v", err)
	}
	if err := os.Chown(rup, int(uid), int(gid)); err != nil {
		return fmt.Errorf("failed to chown user rundir: %v", err)
	}

	dp := path.Join(fsys.Root(), "dev")
	if err := syscall.Mount("", dp, "tmpfs", syscall.MS_NOSUID|syscall.MS_NOEXEC, "mode=755"); err != nil {
		return err
	}

	if !useFullDev {
		for _, d := range basicDevices {
			if err := fsys.CreateDevice(d.path, d.dev, d.mode, d.gid); err != nil {
				return err
			}
		}

		smp := path.Join(fsys.Root(), "/dev", "/shm")
		smflags := uintptr(syscall.MS_NOSUID | syscall.MS_NOEXEC | syscall.MS_REC)
		if err := os.MkdirAll(smp, 0755); err != nil {
			return err
		}
		if err := syscall.Mount("", smp, "tmpfs", smflags, "mode=1777"); err != nil {
			return err
		}
	}

	tp := path.Join(fsys.Root(), "/tmp")
	tflags := uintptr(syscall.MS_NODEV | syscall.MS_NOSUID | syscall.MS_NOEXEC | syscall.MS_REC)
	if err := syscall.Mount("", tp, "tmpfs", tflags, "mode=777"); err != nil {
		return err
	}

	for _, sl := range append(basicSymlinks, deviceSymlinks...) {
		if _, err := fsys.CreateSymlink(sl[0], sl[1]); err != nil {
			return err
		}
	}

	if err := fsys.CreateBlacklistPaths(); err != nil {
		return err
	}

	for _, bl := range basicBlacklist {
		if err := fsys.BlacklistPath(bl, display); err != nil {
			return err
			//log.Warning("Unable to blacklist %s: %v", bl, err)
		}
	}
	return nil
}

func setupMountDirectory(fsys *fs.Filesystem, src string) error {
	acls, err := acl.GetFileAccess(src)
	if err != nil {
		return err
	}
	defer acls.Free()

	target := path.Join(fsys.Root(), src)
	if err := acls.SetFileAccess(target); err != nil {
		return fmt.Errorf("%v on %s", err, target)
	}
	return nil
}
