package fs

import (
	"fmt"
	"os"
	"os/user"
	"path"
	"strconv"
	"strings"
	"syscall"

	"github.com/op/go-logging"
	"github.com/subgraph/oz"
)

type directory struct {
	path  string
	empty bool
}

type Filesystem struct {
	log *logging.Logger
	home         string
	base         string
	root         string
	userID       string
	noDefaults   bool
	noSysAndProc bool
	whitelist    []*mountItem
	blacklist    []*mountItem
}

func (fs *Filesystem) Root() string {
	return fs.root
}

func (fs *Filesystem) addWhitelist(path, target string, readonly bool) error {
	item, err := fs.newItem(path, target, readonly)
	if err != nil {
		return err
	}
	fs.whitelist = append(fs.whitelist, item)
	return nil
}

func (fs *Filesystem) addBlacklist(path string) error {
	item, err := fs.newItem(path, "", false)
	if err != nil {
		return err
	}
	fs.blacklist = append(fs.blacklist, item)
	return nil
}

func (fs *Filesystem) newItem(path, target string, readonly bool) (*mountItem, error) {
	p, err := fs.resolveVars(path)
	if err != nil {
		return nil, err
	}
	return &mountItem{
		path:   p,
		target: target,
		//readonly: readonly,
		fs: fs,
	}, nil
}

func NewFromProfile(profile *oz.Profile, log *logging.Logger) *Filesystem {
	fs := NewFilesystem(profile.Name, log)
	for _,wl := range profile.Whitelist {
		fs.addWhitelist(wl.Path, wl.Path, wl.ReadOnly)
	}
	for _,bl := range profile.Blacklist {
		fs.addBlacklist(bl.Path)
	}
	fs.noDefaults = profile.NoDefaults
	fs.noSysAndProc = profile.NoSysProc
	return fs
}

func NewFilesystem(name string, log *logging.Logger) *Filesystem {

	fs := new(Filesystem)
	fs.log = log
	if log == nil {
		fs.log = logging.MustGetLogger("oz")
	}
	fs.base = path.Join("/srv/oz", name)
	fs.root = path.Join(fs.base, "rootfs")

	u, err := user.Current()
	if err != nil {
		panic("Failed to look up current user: " + err.Error())
	}
	fs.home = u.HomeDir
	fs.userID = strconv.Itoa(os.Getuid())

	return fs
}

/*
func xcreateEmptyDirectories(base string, paths []string) error {
	for _, p := range paths {
		target := path.Join(base, p)
		if err := createEmptyDir(p, target); err != nil {
			return err
		}
	}
	return nil
}

func createEmptyDir(source, target string) error {
	return nil
}

func createSubdirs(base string, subdirs []string) error {
	for _, sdir := range subdirs {
		path := path.Join(base, sdir)
		if err := createDirTree(path); err != nil {
			return err
		}
	}
	return nil
}

func createDirTree(path string) error {
	st, err := os.Stat(path)
	if err == nil {
		if !st.IsDir() {
			return fmt.Errorf("cannot create directory %s because path already exists and is not directory", path)
		}
		return nil
	}
	if !os.IsNotExist(err) {
		return fmt.Errorf("unexpected error attempting Stat() on path %s: %v", path, err)
	}
	if err := os.MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("error creating directory tree %s: %v", path, err)
	}
	return nil
}
*/

// bindMount performs a bind mount of the source path item so that it is visible
// at the target path.  By default the mount is flagged MS_NOSUID and MS_NODEV
// but additional flags can be passed in extraFlags.  If the readonly flag is
// set the bind mount is remounted as MS_RDONLY.
func bindMount(source, target string, readonly bool, extraFlags uintptr) error {
	flags := syscall.MS_BIND | syscall.MS_NOSUID | syscall.MS_NODEV | extraFlags
	if err := syscall.Mount(source, target, "", flags, ""); err != nil {
		return fmt.Errorf("failed to bind mount %s to %s: %v", source, target, err)
	}
	if readonly {
		flags |= syscall.MS_RDONLY | syscall.MS_REMOUNT
		if err := syscall.Mount("", target, "", flags, ""); err != nil {
			return fmt.Errorf("failed to remount %s as RDONLY: %v", target, err)
		}
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
