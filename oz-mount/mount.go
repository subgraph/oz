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

import (
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
		log.Error("Could not load configuration: %s\n", oz.DefaultConfigPath, err)
		os.Exit(1)
	}

	fsys := fs.NewFilesystem(config, log)
	start := 1;
	readonly := false;
	if os.Args[1] == "--readonly" {
		start = 2;
		readonly = true;
	}
	for _, fpath := range os.Args[start:] {
		if !strings.HasPrefix(fpath, "/home/") {
			log.Warning("Ignored `%s`, only files inside of home are permitted!", fpath)
			continue
		}
		switch mode {
		case MOUNT:
			mount(fpath, readonly, fsys, log)
		case UMOUNT:
			unmount(fpath, fsys, log)
		}
	}

	os.Exit(0)
}

func mount(fpath string, readonly bool, fsys *fs.Filesystem, log *logging.Logger) {
	if _, err := os.Stat(fpath); err == nil {
		//log.Notice("Adding file `%s`.", fpath)
		flags := fs.BindCanCreate
		if readonly {
			flags |= fs.BindReadOnly
		}
		if err := fsys.BindPath(fpath, flags, nil); err != nil {
			log.Error("%v while adding `%s`!", err, fpath)
			os.Exit(1)
		}
	}
}

func unmount(fpath string, fsys *fs.Filesystem, log *logging.Logger) {
	sbpath := path.Join(fsys.Root(), fpath)
	if _, err := os.Stat(sbpath); err == nil {
		//log.Notice("Removing file `%s`.", fpath)
		if err := fsys.UnbindPath(fpath); err != nil {
			log.Error("%v while removing `%s`!", err, fpath)
			os.Exit(1)
		}
	} else {
		log.Error("%v error while removing `%s`!", err, fpath)
	}
}

func createLogger() *logging.Logger {
	l := logging.MustGetLogger("oz-init")
	be := logging.NewLogBackend(os.Stderr, "", 0)
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
