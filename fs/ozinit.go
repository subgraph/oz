package fs

import (
	"os"
	"path"
	"syscall"
)

// OzInit is run from the oz-init process and performs post chroot filesystem initialization
func (fs *Filesystem) OzInit() error {
	if err := fs.ozinitMountDev(); err != nil {
		return err
	}
	if err := fs.ozinitMountSysProc(); err != nil {
		return err
	}
	if err := fs.ozinitCreateSymlinks(); err != nil {
		return err
	}
	if err := fs.ozinitBlacklistItems(); err != nil {
		return err
	}
	return nil
}

func (fs *Filesystem) ozinitMountDev() error {
	flags := uintptr(syscall.MS_NOSUID | syscall.MS_REC | syscall.MS_NOEXEC)
	if err := syscall.Mount("none", "/dev", "devtmpfs", flags, ""); err != nil {
		fs.log.Warning("Failed to mount devtmpfs: %v", err)
		return err
	}

	if err := mountSpecial("/dev/shm", "tmpfs"); err != nil {
		fs.log.Warning("Failed to mount shm directory: %v", err)
		return err
	}
	if err := mountSpecial("/dev/pts", "devpts"); err != nil {
		fs.log.Warning("Failed to mount pts directory: %v", err)
		return err
	}
	return nil
}

func mountSpecial(path, mtype string) error {
	flags := uintptr(syscall.MS_NOSUID | syscall.MS_REC | syscall.MS_NOEXEC)
	if err := os.MkdirAll(path, 0755); err != nil {
		return err
	}
	return syscall.Mount(path, path, mtype, flags, "")
}

func (fs *Filesystem) ozinitMountSysProc() error {
	if fs.noSysAndProc {
		return nil
	}
	flags := uintptr(syscall.MS_NOSUID | syscall.MS_REC | syscall.MS_NOEXEC | syscall.MS_NODEV)
	proc := "/proc"
	if err := syscall.Mount("proc", proc, "proc", flags, ""); err != nil {
		fs.log.Warning("Failed to mount /proc: %v", err)
		return err
	}
	roMounts := []string{
		"sysrq-trigger",
		"bus",
		"irq",
		"sys/kernel/hotplug",
	}
	for _, rom := range roMounts {
		p := path.Join(proc, rom)
		if err := bindMount(p, p, true, 0); err != nil {
			fs.log.Warning("Failed to RO mount %s: %v", p, err)
			return err
		}
	}

	if err := syscall.Mount("sysfs", "/sys", "sysfs", syscall.MS_RDONLY|flags, ""); err != nil {
		fs.log.Warning("Failed to mount /sys: %v", err)
		return err
	}

	return nil
}

func (fs *Filesystem) ozinitCreateSymlinks() error {
	for _, sl := range basicSymlinks {
		if err := syscall.Symlink(sl[0], sl[1]); err != nil {
			return err
		}
	}
	return nil
}

func (fs *Filesystem) ozinitBlacklistItems() error {
	if err := createBlacklistDir(emptyDirPath); err != nil {
		return err
	}
	if err := createBlacklistFile(emptyFilePath); err != nil {
		return err
	}
	for _, item := range fs.blacklist {
		if err := item.blacklist(); err != nil {
			return err
		}
	}
	return nil
}

func createBlacklistDir(path string) error {
	if err := os.MkdirAll(path, 0000); err != nil {
		return err
	}
	return setBlacklistPerms(path, 0500)
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
