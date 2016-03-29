package xdg

import (
	"fmt"
	"go/build"
	"io/ioutil"
	"os"
	"path"
	"strings"
)

// Paths determines which directories to search. The first directory
// containing a matching file is used. Here is the order:
//
// Override is always checked first.
//
// Directories specified in the XDG spec are searched after "XDGSuffix" is
// appended.
//
// For configuration files, these are:
//
//	$XDG_CONFIG_HOME (or $HOME/.config when not set)
//	Directories in $XDG_CONFIG_DIRS (or /etc/xdg when not set)
//
// For data files, these are:
//
//	$XDG_DATA_HOME (or $HOME/.local/share when not set)
//	Directories in $XDG_DATA_DIRS (or /usr/local/share:/usr/share when not set)
//
// For runtime files, these are:
//
//	$XDG_RUNTIME_DIR (or /tmp when not set; implementation defined)
//
// Finally, the directory specified by GoImportPath is searched in all
// source directories reported by the `go/build` package.
type Paths struct {
	// When non-empty, this will be the first directory searched.
	Override string

	// The suffix path appended to XDG directories.
	// i.e., "wingo" and NOT "/home/andrew/.config/wingo"
	XDGSuffix string

	// The path in which your data files live inside your project directory.
	// This should include your Go import path plus the directory containing
	// files in your repo. This is used as a last resort to find files.
	// (And it will only work if your package was installed using the GOPATH
	// environment.)
	//
	// N.B. XDGSuffix is not used here,
	// i.e., "github.com/BurntSushi/wingo/config"
	GoImportPath string
}

// MustPanic takes the return values of ConfigFile or DataFile, reads the file
// into a []byte, and returns the bytes.
//
// If the operation does not succeed, it panics.
func (ps Paths) MustPanic(fpath string, err error) []byte {
	if err != nil {
		panic(err)
	}
	bs, err := ioutil.ReadFile(fpath)
	if err != nil {
		panic(err)
	}
	return bs
}

// MustError is like MustPanic, but instead of panicing when something goes
// wrong, it prints the error to stderr and calls os.Exit(1).
func (ps Paths) MustError(fpath string, err error) []byte {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not read %s: %s", fpath, err)
		os.Exit(1)
	}
	bs, err := ioutil.ReadFile(fpath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not read %s: %s", fpath, err)
		os.Exit(1)
	}
	return bs
}

// ConfigFile returns a file path containing the configuration file
// specified. If one cannot be found, an error will be returned which
// contains a list of all file paths searched.
func (ps Paths) ConfigFile(name string) (string, error) {
	home := os.Getenv("HOME")
	xdgHome := os.Getenv("XDG_CONFIG_HOME")
	xdgDirs := os.Getenv("XDG_CONFIG_DIRS")

	// We're going to accumulate a list of directories for places to inspect
	// for configuration files. Basically, this includes following the
	// xdg basedir spec for the XDG_CONFIG_HOME and XDG_CONFIG_DIRS environment
	// variables.
	try := make([]string, 0)

	// from override
	if len(ps.Override) > 0 {
		try = append(try, ps.Override)
	}

	// XDG_CONFIG_HOME
	if len(xdgHome) > 0 && strings.HasPrefix(xdgHome, "/") {
		try = append(try, path.Join(xdgHome, ps.XDGSuffix))
	} else if len(home) > 0 {
		try = append(try, path.Join(home, ".config", ps.XDGSuffix))
	}

	// XDG_CONFIG_DIRS
	if len(xdgDirs) > 0 {
		for _, p := range strings.Split(xdgDirs, ":") {
			// XDG basedir spec does not allow relative paths
			if !strings.HasPrefix(p, "/") {
				continue
			}
			try = append(try, path.Join(p, ps.XDGSuffix))
		}
	} else {
		try = append(try, path.Join("/", "etc", "xdg", ps.XDGSuffix))
	}

	// Add directories from GOPATH. Last resort.
	for _, dir := range build.Default.SrcDirs() {
		d := path.Join(dir, ps.GoImportPath)
		try = append(try, d)
	}

	return searchPaths(try, name)
}

// DataFile returns a file path containing the data file
// specified. If one cannot be found, an error will be returned which
// contains a list of all file paths searched.
func (ps Paths) DataFile(name string) (string, error) {
	home := os.Getenv("HOME")
	xdgHome := os.Getenv("XDG_DATA_HOME")
	xdgDirs := os.Getenv("XDG_DATA_DIRS")

	// We're going to accumulate a list of directories for places to inspect
	// for data files. Basically, this includes following the
	// xdg basedir spec for the XDG_DATA_HOME and XDG_DATA_DIRS environment
	// variables.
	try := make([]string, 0)

	// from override
	if len(ps.Override) > 0 {
		try = append(try, ps.Override)
	}

	// XDG_DATA_HOME
	if len(xdgHome) > 0 && strings.HasPrefix(xdgHome, "/") {
		try = append(try, path.Join(xdgHome, ps.XDGSuffix))
	} else if len(home) > 0 {
		try = append(try, path.Join(home, ".local", "share", ps.XDGSuffix))
	}

	// XDG_CONFIG_DIRS
	if len(xdgDirs) > 0 {
		for _, p := range strings.Split(xdgDirs, ":") {
			// XDG basedir spec does not allow relative paths
			if !strings.HasPrefix(p, "/") {
				continue
			}
			try = append(try, path.Join(p, ps.XDGSuffix))
		}
	} else {
		try = append(try, path.Join("/", "usr", "local", "share", ps.XDGSuffix))
		try = append(try, path.Join("/", "usr", "share", ps.XDGSuffix))
	}

	// Add directories from GOPATH. Last resort.
	for _, dir := range build.Default.SrcDirs() {
		d := path.Join(dir, ps.GoImportPath)
		try = append(try, d)
	}

	return searchPaths(try, name)
}

// RuntimeFile returns a file path containing the runtime file
// specified. If one cannot be found, an error will be returned which
// contains a list of all file paths searched.
func (ps Paths) RuntimeFile(name string) (string, error) {
	xdgRuntime := os.Getenv("XDG_RUNTIME_DIR")

	try := make([]string, 0)

	// from override
	if len(ps.Override) > 0 {
		try = append(try, ps.Override)
	}

	// XDG_RUNTIME_DIR
	if len(xdgRuntime) > 0 {
		try = append(try, path.Join(xdgRuntime, ps.XDGSuffix))
	} else {
		try = append(try, path.Join(os.TempDir(), ps.XDGSuffix))
	}

	// Add directories from GOPATH. Last resort.
	for _, dir := range build.Default.SrcDirs() {
		d := path.Join(dir, ps.GoImportPath)
		try = append(try, d)
	}

	return searchPaths(try, name)
}

func searchPaths(paths []string, suffix string) (string, error) {
	// Now use the first one and keep track of the ones we've tried.
	tried := make([]string, 0, len(paths))
	for _, dir := range paths {
		if len(dir) == 0 {
			continue
		}

		fpath := path.Join(dir, suffix)
		if exists(fpath) {
			return fpath, nil
		} else {
			tried = append(tried, fpath)
		}
	}

	// Show the user where we've looked for config files...
	triedStr := strings.Join(tried, ", ")
	return "", fmt.Errorf("Could not find a '%s' file. Tried "+
		"the following paths: %s", suffix, triedStr)
}

func exists(p string) bool {
	_, err := os.Stat(p)
	return err == nil || os.IsExist(err)
}
