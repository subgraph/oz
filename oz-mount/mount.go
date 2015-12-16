// +build linux,!gccgo
package mount

// extern int enter_mount_namespace(void);
/*
#include <stdlib.h>
__attribute__((constructor)) void init(void) {
	if (enter_mount_namespace() < 0) {
		exit(EXIT_FAILURE);
	}
}
*/
import "C"

/*
	As per the setns documentation, it is impossible to enter a
	mount namespace from a multithreaded process.
	One MUST insure that opening the namespace happens when the process
	has only one thread. This is impossible from golang, as such we call
	this C function as a constructor to ensure that it is executed
	before the go scheduler launches other threads.
*/

import (
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/subgraph/oz"
	"github.com/subgraph/oz/fs"

	"github.com/op/go-logging"
)

const (
	MOUNT = 1 << iota
	UMOUNT
)

func Main(mode int) {
	log := createLogger()
	config, err := loadConfig()
	if err != nil {
		log.Error("Could not load configuration: %s (%+v)", oz.DefaultConfigPath, err)
		os.Exit(1)
	}
	fsys := fs.NewFilesystem(config, log)
	homedir := os.Getenv("_OZ_HOMEDIR")
	if homedir == "" {
		log.Error("Homedir must be set!")
		os.Exit(1)
	}
	os.Setenv("_OZ_HOMEDIR", "")

	start := 1
	readonly := false
	if os.Args[1] == "--readonly" {
		start = 2
		readonly = true
	}
	for _, fpath := range os.Args[start:] {
		cpath, err := cleanPath(fpath, homedir)
		if err != nil || cpath == "" {
			log.Error("%v", err)
			os.Exit(1)
		}
		switch mode {
		case MOUNT:
			mount(cpath, readonly, fsys, log)
		case UMOUNT:
			unmount(cpath, fsys, log)
		default:
			log.Error("Unknown mode!")
			os.Exit(1)
		}
	}

	os.Exit(0)
}

func cleanPath(spath, homedir string) (string, error) {
	spath = path.Clean(spath)
	if !path.IsAbs(spath) {
		spath = path.Join(homedir, spath)
	}
	if !strings.HasPrefix(spath, homedir) {
		return "", fmt.Errorf("only files inside of the user home are permitted")
	}
	return spath, nil
}

func mount(fpath string, readonly bool, fsys *fs.Filesystem, log *logging.Logger) {
	//log.Notice("Adding file `%s`.", fpath)
	// TODO: Check if target is empty directory (and not a mountpoint) and allow the bind in that case
	if _, err := os.Stat(fpath); err != nil {
		log.Error("%v", err)
		os.Exit(1)
	}
	flags := 0 //fs.BindCanCreate
	if readonly {
		flags |= fs.BindReadOnly
	}
	if err := fsys.BindPath(fpath, flags, nil); err != nil {
		log.Error("%v", err)
		os.Exit(1)
	}
}

func unmount(fpath string, fsys *fs.Filesystem, log *logging.Logger) {
	sbpath := path.Join(fsys.Root(), fpath)
	if _, err := os.Stat(sbpath); err == nil {
		//log.Notice("Removing file `%s`.", fpath)
		if err := fsys.UnbindPath(fpath); err != nil {
			log.Error("%v", err)
			os.Exit(1)
		}
	} else {
		log.Warning("%v", err)
	}
}

func createLogger() *logging.Logger {
	l := logging.MustGetLogger("oz-init")
	be := logging.NewLogBackend(os.Stdout, "", 0)
	f := logging.MustStringFormatter("%{level:.1s} %{message}")
	fbe := logging.NewBackendFormatter(be, f)
	logging.SetBackend(fbe)
	return l
}

func loadConfig() (*oz.Config, error) {
	config, err := oz.LoadConfig(oz.DefaultConfigPath)
	if err != nil {
		if os.IsNotExist(err) {
			config = oz.NewDefaultConfig()
		} else {
			return nil, err
		}
	}

	return config, nil
}
