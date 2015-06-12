package fs

import (
	"errors"
	"fmt"
	"os"
	"os/user"
	"path"
	"strconv"
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

var basicBlacklist = []string{
	"/usr/sbin", "/sbin", "${PATH}/su",
	"${PATH}/sudo", "${PATH}/fusermount",
	"${PATH}/xinput", "${PATH}/strace",
	"${PATH}/mount", "${PATH}/umount",
}

const emptyFilePath = "/tmp/oz.ro.file"
const emptyDirPath = "/tmp/oz.ro.dir"

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

func (fs *Filesystem) Setup(profilesPath string) error {
	profilePathInBindDirs := false
	for _, bd := range basicBindDirs {
		if bd == profilesPath {
			profilePathInBindDirs = true
			break
		}
	}

	if profilePathInBindDirs == false {
		basicBindDirs = append(basicBindDirs, profilesPath)
	}

	if fs.xpra != "" {
		if err := fs.createXpraDir(); err != nil {
			return err
		}
		item, err := fs.newItem(fs.xpra, fs.xpra, false)
		if err != nil {
			return err
		}
		fs.whitelist = append(fs.whitelist, item)
	}
	if err := fs.setupRootfs(); err != nil {
		return err
	}
	if err := fs.setupChroot(); err != nil {
		return err
	}
	if fs.fullDevices == false {
		if err := fs.setupDev(); err != nil {
			return err
		}
	}

	return fs.setupMountItems()
}

func (fs *Filesystem) createXpraDir() error {
	uid, gid, err := userIds(fs.user)
	if err != nil {
		return err
	}
	dir := path.Join(fs.user.HomeDir, ".Xoz", fs.name)
	if err := createSubdirs(fs.user.HomeDir, uid, gid, 0755, ".Xoz", fs.name); err != nil {
		return fmt.Errorf("failed to create xpra directory (%s): %v", dir, err)
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

func (fs *Filesystem) setupRootfs() error {
	if err := os.MkdirAll(fs.base, 0755); err != nil {
		return fmt.Errorf("unable to create directory (%s): %v", fs.base, err)
	}
	flags := uintptr(syscall.MS_NOSUID | syscall.MS_NOEXEC | syscall.MS_NODEV)
	data := "mode=755,gid=0"
	if err := syscall.Mount(fs.base, fs.base, "tmpfs", flags, data); err != nil {
		return fmt.Errorf("failed to create base tmpfs at %s: %v", fs.base, err)
	}
	/*
	   // Currently unused
	   	// create extra directories
	   	extra := []string{"sockets", "dev"}
	   	for _, sub := range extra {
	   		d := path.Join(fs.base, sub)
	   		if err := os.Mkdir(d, 0755); err != nil {
	   			return fmt.Errorf("unable to create directory (%s): %v", d, err)
	   		}
	   	}
	*/
	return nil
}

func (fs *Filesystem) setupChroot() error {
	var err error
	if fs.noDefaults {
		err = createEmptyDirectories(fs.root, basicBindDirs)
	} else {
		err = bindBasicDirectories(fs.root, basicBindDirs)
	}
	if err != nil {
		return err
	}
	err = createEmptyDirectories(fs.root, basicEmptyDirs)
	if err != nil {
		return err
	}
	return setupTmp(fs.root)
}

func (fs *Filesystem) setupDev() error {
	devPath := path.Join(fs.root, "dev")
	flags := uintptr(syscall.MS_NOSUID | syscall.MS_NOEXEC)
	if err := syscall.Mount("none", devPath, "tmpfs", flags, ""); err != nil {
		fs.log.Warning("Failed to mount devtmpfs: %v", err)
		return err
	}

	for _, dev := range basicDevices {
		path := path.Join(fs.root, dev.path)
		if err := syscall.Mknod(path, dev.mode, dev.dev); err != nil {
			return fmt.Errorf("Failed to mknod device %s: %+v", path, err)
		}
		if err := os.Chmod(path, os.FileMode(dev.perm)); err != nil {
			return fmt.Errorf("Unable to set permissions for device %s: %+v", dev.path, err)
		}
	}

	shmPath := path.Join(devPath, "shm")
	if err := mountSpecial(shmPath, "tmpfs"); err != nil {
		fs.log.Warning("Failed to mount shm directory: %v", err)
		return err
	}

	return nil
}

func bindBasicDirectories(root string, dirs []string) error {
	for _, src := range dirs {
		st, err := os.Lstat(src)
		if err != nil {
			return err
		}
		mode := st.Mode()
		target := path.Join(root, src)
		if err := os.MkdirAll(target, mode.Perm()); err != nil {
			return err
		}
		if err := bindMount(src, target, true, 0); err != nil {
			return err
		}
	}
	return nil
}

func createEmptyDirectories(root string, dirs []string) error {
	for _, p := range dirs {
		target := path.Join(root, p)
		if err := createEmptyDirectory(p, target); err != nil {
			return err
		}
	}
	return nil
}

func createEmptyDirectory(source, target string) error {
	fi, err := os.Stat(source)
	if err != nil {
		return err
	}
	mode := fi.Mode()
	if err := os.MkdirAll(target, mode.Perm()); err != nil {
		return err
	}
	if err := copyFileInfo(fi, target); err != nil {
		return err
	}
	return nil
}

func setupTmp(root string) error {
	target := path.Join(root, "tmp")
	if err := os.Chmod(target, 0777); err != nil {
		return err
	}
	return bindMount(target, target, false, syscall.MS_NOEXEC)
}

func (fs *Filesystem) setupMountItems() error {
	for _, item := range fs.whitelist {
		if err := item.bind(); err != nil {
			// XXX
		}
	}
	return nil
}
