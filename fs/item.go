package fs

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"syscall"
	"github.com/op/go-logging"
)

type MountFlag int

func (mf MountFlag) isSet(f MountFlag) bool {
	return mf&f == f
}

const (
	MountReadOnly MountFlag = 1 << iota
	MountCreateIfAbsent
)

type mountItem struct {
	path   string
	target string
	flags  MountFlag
	fs     *Filesystem
}

func (mi *mountItem) targetPath() string {
	root := mi.fs.root
	if mi.target != "" {
		return path.Join(root, mi.target)
	}
	return path.Join(root, mi.path)
}

func (mi *mountItem) bind() error {
	if strings.Contains(mi.path, "*") {
		return mi.bindGlobbed()
	}
	return mi.bindItem()
}

func (mi *mountItem) bindGlobbed() error {
	if mi.target != "" {
		mi.fs.log.Warning("Ignoring target directory (%s) for mount item containing glob character: (%s)", mi.target, mi.path)
		mi.target = ""
	}
	globbed, err := filepath.Glob(mi.path)
	if err != nil {
		return err
	}
	savedPath := mi.path
	for _, p := range globbed {
		if strings.Contains(p, "*") {
			// XXX
			continue
		}
		mi.path = p
		if err := mi.bind(); err != nil {
			// XXX
			mi.path = savedPath
			return err
		}
	}
	mi.path = savedPath
	return nil
}

func (mi *mountItem) readSourceInfo(src string) (os.FileInfo, error) {
	if fi, err := os.Stat(src); err == nil {
		return fi, nil
	} else if !os.IsNotExist(err) {
		return nil, err
	}

	if !mi.flags.isSet(MountCreateIfAbsent) {
		return nil, fmt.Errorf("source path (%s) does not exist", src)
	}

	if !strings.HasPrefix(src, mi.fs.home) {
		return nil, fmt.Errorf("mount item (%s) has flag MountCreateIfAbsent, but is not child of home directory (%s)", src, mi.fs.home)
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

func (mi *mountItem) bindItem() error {
	src, err := filepath.EvalSymlinks(mi.path)
	if err != nil {
		return fmt.Errorf("error resolving symlinks for path (%s): %v", mi.path, err)
	}

	sinfo, err := mi.readSourceInfo(src)
	if err != nil {
		// XXX
		return err
	}

	target := mi.targetPath()
	_, err = os.Stat(target)
	if err == nil || !os.IsNotExist(err) {
		mi.fs.log.Warning("Target (%s) already exists, ignoring", target)
		return nil
	}
	if sinfo.IsDir() {
		if err := os.MkdirAll(target, sinfo.Mode().Perm()); err != nil {
			return err
		}
	} else {
		if err := createEmptyFile(target, 0750); err != nil {
			return err
		}
	}

	if err := copyPathPermissions(mi.fs.root, src); err != nil {
		return fmt.Errorf("failed to copy path permissions for (%s): %v", src, err)
	}
	return bindMount(src, target, mi.flags.isSet(MountReadOnly), 0)
}

func (mi *mountItem) blacklist() error {
	if strings.Contains(mi.path, "*") {
		return mi.blacklistGlobbed()
	}
	return blacklistItem(mi.path, mi.fs.log)
}

func (mi *mountItem) blacklistGlobbed() error {
	globbed, err := filepath.Glob(mi.path)
	if err != nil {
		// XXX
	}
	for _, p := range globbed {
		if err := blacklistItem(p, mi.fs.log); err != nil {
			return err
		}
	}
	return nil
}

func blacklistItem(path string, log *logging.Logger) error {
	p, err := filepath.EvalSymlinks(path)
	if err != nil {
		log.Warning("Symlink evaluation failed for path: %s", path)
		return err
	}
	fi, err := os.Stat(p)
	if err != nil {
		if os.IsNotExist(err) {
			log.Info("Blacklist item (%s) does not exist", p)
			return nil
		}
		return err
	}

	src := emptyFilePath
	if fi.IsDir() {
		src = emptyDirPath
	}
	if err := syscall.Mount(src, p, "none", syscall.MS_BIND, "mode=400,gid=0"); err != nil {
		// XXX warning
		return err
	}
	// XXX log success

	return nil
}
