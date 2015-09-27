package oz

import (
	"fmt"
	"os"
	"path"
	"syscall"
)

func checkConfigPermissions(fpath string) error {
	pd := path.Dir(fpath)
	for _, fp := range []string{pd, fpath} {
		if err := checkPathRootPermissions(fp); err != nil {
			return fmt.Errorf("file `%s` is %s", fp, err)
		}
	}
	return nil
}

func checkPathRootPermissions(fpath string) error {
	fstat, err := os.Stat(fpath)
	if err != nil {
		return err
	}
	if (fstat.Mode().Perm() & syscall.S_IWOTH) != 0 {
		return fmt.Errorf("writable by everyone!")
	}
	if (fstat.Mode().Perm()&syscall.S_IWGRP) != 0 && fstat.Sys().(*syscall.Stat_t).Gid != 0 {
		return fmt.Errorf("writable by someone else than root!")
	}
	return nil
}
