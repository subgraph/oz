package fs

import (
	"fmt"
	"os"
	"path"
	"strings"
	"syscall"

	"github.com/op/go-logging"
	"github.com/subgraph/oz"
	"os/user"
	"path/filepath"
)

type Filesystem struct {
	log    *logging.Logger
	base   string
	chroot bool
}

func NewFilesystem(config *oz.Config, log *logging.Logger) *Filesystem {
	if log == nil {
		log = logging.MustGetLogger("oz")
	}
	return &Filesystem{
		base: config.SandboxPath,
		log:  log,
	}
}

func (fs *Filesystem) Root() string {
	return path.Join(fs.base, "rootfs")
}

func (fs *Filesystem) absPath(p string) string {
	if fs.chroot {
		return p
	}
	return path.Join(fs.Root(), p)
}

func (fs *Filesystem) CreateEmptyDir(target string) error {
	fi, err := os.Stat(target)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(fs.absPath(target), fi.Mode().Perm()); err != nil {
		return err
	}
	return copyFileInfo(fi, target)
}

func (fs *Filesystem) CreateDevice(devpath string, dev int, mode uint32) error {
	p := fs.absPath(devpath)
	um := syscall.Umask(0)
	if err := syscall.Mknod(p, mode, dev); err != nil {
		return fmt.Errorf("failed to mknod device '%s': %v", p, err)
	}
	syscall.Umask(um)
	return nil
}

func (fs *Filesystem) CreateSymlink(oldpath, newpath string) error {
	if err := syscall.Symlink(oldpath, fs.absPath(newpath)); err != nil {
		return fmt.Errorf("failed to symlink %s to %s: %v", fs.absPath(newpath), oldpath, err)
	}
	return nil
}

func (fs *Filesystem) BindPath(target string, flags int, u *user.User) error {
	return fs.bindResolve(target, "", flags, u)
}

func (fs *Filesystem) BindTo(from string, to string, flags int, u *user.User) error {
	return fs.bindResolve(from, to, flags, u)
}

const (
	BindReadOnly = 1 << iota
	BindCanCreate
)

func (fs *Filesystem) bindResolve(from string, to string, flags int, u *user.User) error {
	if (to == "") || (from == to) {
		return fs.bindSame(from, flags, u)
	}
	if isGlobbed(to) {
		return fmt.Errorf("bind target (%s) cannot have globbed path", to)
	}
	t, err := resolveVars(to, u)
	if err != nil {
		return err
	}
	if isGlobbed(from) {
		return fmt.Errorf("bind src (%s) cannot have globbed path with separate target path (%s)", from, to)
	}
	f, err := resolveVars(from, u)
	if err != nil {
		return err
	}
	return fs.bind(f, t, flags, u)
}

func (fs *Filesystem) bindSame(p string, flags int, u *user.User) error {
	ps, err := resolvePath(p, u)
	if err != nil {
		return err
	}
	for _, p := range ps {
		if err := fs.bind(p, p, flags, u); err != nil {
			return err
		}
	}
	return nil
}

func (fs *Filesystem) bind(from string, to string, flags int, u *user.User) error {
	src, err := filepath.EvalSymlinks(from)
	if err != nil {
		return fmt.Errorf("error resolving symlinks for path (%s): %v", from, err)
	}
	cc := flags&BindCanCreate != 0
	sinfo, err := readSourceInfo(src, cc, u)
	if err != nil {
		return fmt.Errorf("failed to bind path (%s): %v", src, err)
	}

	if to == "" {
		to = from
	}
	to = path.Join(fs.Root(), to)

	_, err = os.Stat(to)
	if err == nil || !os.IsNotExist(err) {
		fs.log.Warning("Target (%s > %s) already exists, ignoring", src, to)
		return nil
	}

	if sinfo.IsDir() {
		if err := os.MkdirAll(to, sinfo.Mode().Perm()); err != nil {
			return err
		}
	} else {
		if err := createEmptyFile(to, 0750); err != nil {
			return err
		}
	}

	if err := copyPathPermissions(fs.Root(), src); err != nil {
		return fmt.Errorf("failed to copy path permissions for (%s): %v", src, err)
	}
	fs.log.Info("bind mounting %s -> %s", src, to)
	mntflags := syscall.MS_NOSUID | syscall.MS_NODEV
	if flags&BindReadOnly != 0 {
		mntflags |= syscall.MS_RDONLY
	} else {
		flags |= syscall.MS_NOEXEC
	}
	return bindMount(src, to, mntflags)
}

func readSourceInfo(src string, cancreate bool, u *user.User) (os.FileInfo, error) {
	if fi, err := os.Stat(src); err == nil {
		return fi, nil
	} else if !os.IsNotExist(err) {
		return nil, err
	}

	if u == nil || !cancreate {
		return nil, fmt.Errorf("source path (%s) does not exist", src)
	}

	home := u.HomeDir
	if !strings.HasPrefix(src, home) {
		return nil, fmt.Errorf("mount item (%s) has flag MountCreateIfAbsent, but is not child of home directory (%s)", src, home)
	}

	if err := os.MkdirAll(src, 0750); err != nil {
		return nil, err
	}

	pinfo, err := os.Stat(path.Dir(src))
	if err != nil {
		return nil, err
	}

	if err := copyFileInfo(pinfo, src); err != nil {
		return nil, err
	}

	return os.Stat(src)
}

func (fs *Filesystem) BlacklistPath(target string, u *user.User) error {
	ps, err := resolvePath(target, u)
	if err != nil {
		return nil
	}
	for _, p := range ps {
		if err := fs.blacklist(p); err != nil {
			return err
		}
	}
	return nil
}

func (fs *Filesystem) blacklist(target string) error {
	t, err := filepath.EvalSymlinks(target)
	if err != nil {
		return fmt.Errorf("symlink evaluation failed while blacklisting path %s: %v", target, err)
	}
	fi, err := os.Stat(t)
	if err != nil {
		if os.IsNotExist(err) {
			fs.log.Info("Blacklist path (%s) does not exist", t)
			return nil
		}
		return err
	}
	src := emptyFilePath
	if fi.IsDir() {
		src = emptyDirPath
	}

	if err := syscall.Mount(fs.absPath(src), fs.absPath(t), "", syscall.MS_BIND, "mode=400,gid=0"); err != nil {
		return fmt.Errorf("failed to bind %s -> %s for blacklist: %v", src, t, err)
	}
	if err := remount(fs.absPath(t), syscall.MS_RDONLY); err != nil {
		return fmt.Errorf("failed to bind %s -> %s for blacklist: %v", src, t, err)
	}
	return nil
}

func (fs *Filesystem) Chroot() error {
	if fs.chroot {
		return fmt.Errorf("filesystem is already in chroot()")
	}
	fs.log.Debug("chroot to %s", fs.Root())
	if err := syscall.Chroot(fs.Root()); err != nil {
		return fmt.Errorf("chroot to %s failed: %v", fs.Root(), err)
	}
	if err := os.Chdir("/"); err != nil {
		return fmt.Errorf("chdir to / after chroot() failed: %v", err)
	}
	fs.chroot = true
	return nil
}

func (fs *Filesystem) MountProc() error {
	err := fs.mountSpecial("/proc", "proc", 0, "")
	if err != nil {
		return err
	}
	roMounts := []string{
		"sysrq-trigger",
		"bus",
		"irq",
		"sys/kernel/hotplug",
	}
	for _, rom := range roMounts {
		if _, err := os.Stat(rom); err == nil {
			if err := bindMount(rom, rom, syscall.MS_RDONLY); err != nil {
				return fmt.Errorf("remount RO of %s failed: %v", rom, err)
			}
		}
	}
	return nil
}

func (fs *Filesystem) MountFullDev() error {
	return fs.mountSpecial("/dev", "devtmpfs", 0, "")
}

func (fs *Filesystem) MountSys() error {
	return fs.mountSpecial("/sys", "sysfs", syscall.MS_RDONLY, "")
}

func (fs *Filesystem) MountTmp() error {
	return fs.mountSpecial("/tmp", "tmpfs", syscall.MS_NODEV, "")
}

func (fs *Filesystem) MountPts() error {
	return fs.mountSpecial("/dev/pts", "devpts", 0, "newinstance,ptmxmode=0666")
}

func (fs *Filesystem) MountShm() error {
	return fs.mountSpecial("/dev/shm", "tmpfs", syscall.MS_NODEV, "")
}

func (fs *Filesystem) mountSpecial(path, mtype string, flags int, args string) error {
	if !fs.chroot {
		return fmt.Errorf("cannot mount %s (%s) until Chroot() is called.", path, mtype)
	}
	fs.log.Debug("Mounting %s [%s]", path, mtype)
	if err := os.MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("failed to create mount point (%s): %v", path, err)
	}
	mountFlags := uintptr(flags | syscall.MS_NOSUID | syscall.MS_NOEXEC | syscall.MS_REC)
	return syscall.Mount("", path, mtype, mountFlags, args)
}

func bindMount(source, target string, flags int) error {
	if err := syscall.Mount(source, target, "", syscall.MS_BIND, ""); err != nil {
		return fmt.Errorf("bind mount of %s -> %s failed: %v", source, target, err)
	}
	if flags != 0 {
		return remount(target, flags)
	}
	return nil
}

func remount(target string, flags int) error {
	fl := uintptr(flags | syscall.MS_BIND | syscall.MS_REMOUNT)
	if err := syscall.Mount("", target, "", fl, ""); err != nil {
		return fmt.Errorf("failed to remount %s with flags %x: %v", target, flags, err)
	}
	return nil
}

const emptyFilePath = "/oz.ro.file"
const emptyDirPath = "/oz.ro.dir"

func (fs *Filesystem) CreateBlacklistPaths() error {
	if err := createBlacklistDir(fs.absPath(emptyDirPath)); err != nil {
		return err
	}
	if err := createBlacklistFile(fs.absPath(emptyFilePath)); err != nil {
		return err
	}
	return nil
}

func createBlacklistDir(p string) error {
	if err := os.MkdirAll(p, 0000); err != nil {
		return err
	}
	return setBlacklistPerms(p, 0500)
}

func createBlacklistFile(path string) error {
	fd, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := fd.Close(); err != nil {
		return err
	}
	return setBlacklistPerms(path, 0400)
}

func setBlacklistPerms(path string, mode os.FileMode) error {
	if err := os.Chown(path, 0, 0); err != nil {
		return err
	}
	if err := os.Chmod(path, mode); err != nil {
		return err
	}
	return nil
}

func createEmptyFile(name string, mode os.FileMode) error {
	if err := os.MkdirAll(path.Dir(name), 0750); err != nil {
		return err
	}
	fd, err := os.Create(name)
	if err != nil {
		return err
	}
	if err := fd.Close(); err != nil {
		return err
	}
	if err := os.Chmod(name, mode); err != nil {
		return err
	}
	return nil
}

func copyPathPermissions(root, src string) error {
	current := "/"
	for _, part := range strings.Split(src, "/") {
		if part == "" {
			continue
		}
		current = path.Join(current, part)
		target := path.Join(root, current)
		if err := copyFilePermissions(current, target); err != nil {
			return err
		}
	}
	return nil
}

func copyFilePermissions(src, target string) error {
	fi, err := os.Stat(src)
	if err != nil {
		return err
	}
	return copyFileInfo(fi, target)
}

func copyFileInfo(info os.FileInfo, target string) error {
	st := info.Sys().(*syscall.Stat_t)
	os.Chown(target, int(st.Uid), int(st.Gid))
	os.Chmod(target, info.Mode().Perm())
	return nil
}
