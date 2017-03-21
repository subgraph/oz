package fs

import (
	"fmt"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/op/go-logging"

	"github.com/subgraph/go-xdgdirs"
	"github.com/subgraph/oz"
)

type Filesystem struct {
	log     *logging.Logger
	base    string
	chroot  bool
	xdgDirs *xdgdirs.Dirs
	user    *user.User
	profile *oz.Profile
}

func NewFilesystem(config *oz.Config, log *logging.Logger, u *user.User, p *oz.Profile) *Filesystem {
	if log == nil {
		log = logging.MustGetLogger("oz")
	}

	dirs := new(xdgdirs.Dirs)
	if u != nil {
		dirs.Load(u.HomeDir)
	}
	return &Filesystem{
		base:    config.SandboxPath,
		log:     log,
		user:    u,
		xdgDirs: dirs,
		profile: p,
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

	return copyFileInfo(fi, fs.absPath(target))
}

func (fs *Filesystem) CreateDevice(devpath string, dev int, mode uint32, gid int) error {
	p := fs.absPath(devpath)
	um := syscall.Umask(0)
	if err := syscall.Mknod(p, mode, dev); err != nil {
		return fmt.Errorf("failed to mknod device '%s': %v", p, err)
	}
	if gid > 0 {
		if err := os.Chown(p, 0, gid); err != nil {
			return fmt.Errorf("failed to change group for device '%s': %v", p, err)
		}
	}
	syscall.Umask(um)
	return nil
}

func (fs *Filesystem) CreateSymlink(oldpath, newpath string) (string, error) {
	if err := syscall.Symlink(oldpath, fs.absPath(newpath)); err != nil {
		return "", fmt.Errorf("failed to symlink %s to %s: %v", fs.absPath(newpath), oldpath, err)
	}
	return fs.absPath(newpath), nil
}

func (fs *Filesystem) BindPath(from string, flags int, display int) error {
	return fs.bindResolve(from, "", flags, display)
}

func (fs *Filesystem) BindTo(from, to string, flags int, display int) error {
	return fs.bindResolve(from, to, flags, display)
}

const (
	BindReadOnly = 1 << iota
	BindCanCreate
	BindIgnore
	BindForce
	BindNoFollow
	BindAllowSetuid
)

func (fs *Filesystem) bindResolve(from string, to string, flags int, display int) error {
	if (to == "") || (from == to) {
		return fs.bindSame(from, flags, display)
	}
	if isGlobbed(to) {
		return fmt.Errorf("bind target (%s) cannot have globbed path", to)
	}
	t, err := resolveVars(to, display, fs.user, fs.xdgDirs, fs.profile)
	if err != nil {
		return err
	}
	if isGlobbed(from) {
		return fmt.Errorf("bind src (%s) cannot have globbed path with separate target path (%s)", from, to)
	}
	f, err := resolveVars(from, display, fs.user, fs.xdgDirs, fs.profile)
	if err != nil {
		return err
	}
	return fs.bind(f, t, flags)
}

func (fs *Filesystem) bindSame(p string, flags int, display int) error {
	ps, err := resolvePath(p, display, fs.user, fs.xdgDirs, fs.profile)
	if err != nil {
		return err
	}
	for _, p := range ps {
		if err := fs.bind(p, p, flags); err != nil {
			return err
		}
	}
	return nil
}

func (fs *Filesystem) bind(from string, to string, flags int) error {
	cc := flags&BindCanCreate != 0
	ii := flags&BindIgnore != 0
	ff := flags&BindForce != 0
	nf := flags&BindNoFollow != 0
	var src string
	var err error
	if !nf {
		src, err = filepath.EvalSymlinks(from)
		if err != nil && !cc && !ii {
			return fmt.Errorf("error resolving symlinks for path (%s): %v", from, err)
		}
	}
	if src == "" {
		src = from
	}
	sinfo, err := readSourceInfo(src, cc, fs)
	if err != nil {
		if !ii {
			return fmt.Errorf("failed to bind path (%s): %v", src, err)
		} else {
			fs.log.Warning("bind target (%s) missing and ignored!", src)
			return nil
		}
	}
	if sinfo == nil {
		fs.log.Warning("bind target (%s) does not exist and has been ignored!", src)
		return nil
	}

	if to == "" {
		to = from
	}
	to = path.Join(fs.Root(), to)

	s, err := os.Stat(to)
	if !ff && (err == nil || !os.IsNotExist(err)) {
		fs.log.Warning("Target (%s > %s) already exists, ignoring: %v %v", src, to, err, s)
		return nil
	}

	if sinfo.IsDir() {
		if err := os.MkdirAll(to, sinfo.Mode().Perm()); err != nil {
			return err
		}
	} else if os.IsNotExist(err) {
		if err := createEmptyFile(to, 0750); err != nil {
			return err
		}
	}

	if err := copyPathPermissions(fs.Root(), src); err != nil {
		return fmt.Errorf("failed to copy path permissions for (%s): %v", src, err)
	}

	rolog := " "
	sulog := " "
	mntflags := syscall.MS_NODEV
	if flags&BindReadOnly != 0 {
		mntflags |= syscall.MS_RDONLY
		rolog = "(as readonly) "
	} else {
		flags |= syscall.MS_NOEXEC
	}
	if flags&BindAllowSetuid != 0 {
		sulog = "(setuid allowed) "
	} else {
		mntflags |= syscall.MS_NOSUID
	}
	fs.log.Info("bind mounting %s%s%s -> %s", rolog, sulog, src, to)
	return bindMount(src, to, mntflags)
}

func (fs *Filesystem) UnbindPath(to string) error {
	to = path.Join(fs.Root(), to)

	_, err := os.Stat(to)
	if err != nil {
		fs.log.Warning("Target (%s) does not exist, ignoring", to)
		return nil
	}

	// XXX
	fs.log.Info("unbinding %s", to)
	if err := syscall.Unmount(to, syscall.MNT_DETACH /* | syscall.MNT_FORCE*/); err != nil {
		return err
	}

	return os.Remove(to)
}

func readSourceInfo(src string, cancreate bool, fs *Filesystem) (os.FileInfo, error) {
	u := fs.user
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

	hi, err := os.Stat(home)
	if err != nil {
		return nil, err
	}

	// Create the tree inside the user's home directory, chown it to the user

	if err := fs.MkdirAllChownParent(src, 0750, hi); err != nil {
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

func (fs *Filesystem) BlacklistPath(target string, display int) error {
	ps, err := resolvePath(target, display, fs.user, fs.xdgDirs, fs.profile)
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
	t, err := filepath.EvalSymlinks(fs.absPath(target))
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

	if err := syscall.Mount(fs.absPath(src), t, "", syscall.MS_BIND, "mode=400,gid=0"); err != nil {
		return fmt.Errorf("failed to bind %s -> %s for blacklist: %v", src, t, err)
	} else {
		fs.log.Info("Blacklisted path: %s -> %s", src, t)
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
	return fs.mountSpecial("/dev/pts", "devpts", 0, "newinstance,mode=620,gid=5,ptmxmode=0600")
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
	if err := rdonlyBindBlacklistItem(fs.absPath(emptyDirPath)); err != nil {
		return err
	}

	if err := createBlacklistFile(fs.absPath(emptyFilePath)); err != nil {
		return err
	}
	if err := rdonlyBindBlacklistItem(fs.absPath(emptyFilePath)); err != nil {
		return err
	}
	return nil
}

func rdonlyBindBlacklistItem(target string) error {
	if err := syscall.Mount(target, target, "", syscall.MS_BIND, "mode=400,gid=0"); err != nil {
		return err
	}
	if err := remount(target, syscall.MS_RDONLY); err != nil {
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

func (fs *Filesystem) GetUser() *user.User {
	return fs.user
}

func (fs *Filesystem) GetProfile() *oz.Profile {
	return fs.profile
}

func (fs *Filesystem) GetXDGDirs() *xdgdirs.Dirs {
	return fs.xdgDirs
}

func (fs *Filesystem) MkdirAllChownParent(pathstr string, perm os.FileMode, parent os.FileInfo) error {

	/*
		Function borrowed from golang src and modified to chown all created subdirs to
		parent FileInfo passed as arg
	*/

	// Fast path: if we can tell whether path is a directory or file, stop with success or error.
	dir, err := os.Stat(pathstr)
	if err == nil {
		if dir.IsDir() {
			return nil
		}
		return &os.PathError{"mkdir", pathstr, syscall.ENOTDIR}
	}

	// Slow path: make sure parent exists and then call Mkdir for path.
	i := len(pathstr)
	for i > 0 && os.IsPathSeparator(pathstr[i-1]) { // Skip trailing path separator.
		i--
	}

	j := i
	for j > 0 && !os.IsPathSeparator(pathstr[j-1]) { // Scan backward over element.
		j--
	}

	if j > 1 {
		// Create parent
		err = fs.MkdirAllChownParent(pathstr[0:j-1], perm, parent)
		if err != nil {
			return err
		}
	}

	// Parent now exists; invoke Mkdir and use its result.
	err = os.Mkdir(pathstr, perm)
	if err != nil {
		// Handle arguments like "foo/." by
		// double-checking that directory doesn't exist.
		dir, err1 := os.Lstat(pathstr)
		if err1 == nil && dir.IsDir() {
			return nil
		}
		return err
	}
	st := parent.Sys().(*syscall.Stat_t)
	os.Chown(pathstr, int(st.Uid), int(st.Gid))
	os.Chmod(pathstr, parent.Mode().Perm())

	if err != nil {
		return err
	}
	return nil
}
