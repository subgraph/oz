package fs

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/subgraph/go-xdgdirs"
)

func ResolvePathNoGlob(p string, d int, u *user.User, xdgDirs *xdgdirs.Dirs) (string, error) {
	return resolveVars(p, d, u, xdgDirs)
}

func resolvePath(p string, d int, u *user.User, xdgDirs *xdgdirs.Dirs) ([]string, error) {
	p, err := resolveVars(p, d, u, xdgDirs)
	if err != nil {
		return nil, err
	}
	return resolveGlob(p)
}

func resolveVars(p string, d int, u *user.User, xdgDirs *xdgdirs.Dirs) (string, error) {
	const pathVar = "${PATH}/"
	const homeVar = "${HOME}"
	const uidVar = "${UID}"
	const userVar = "${USER}"
	const displayVar = "${DISPLAY}"


	fmt.Println("IS XDG: %s => %b", p, xdgdirs.IsXDGDir(p))
	switch {
	case strings.HasPrefix(p, pathVar):
		emptyPath := false
		if os.Getenv("PATH") == "" {
			emptyPath = true
			os.Setenv("PATH", "/bin:/usr/bin:/sbin:/usr/sbin")
		}
		resolved, err := exec.LookPath(p[len(pathVar):])
		if emptyPath {
			os.Setenv("PATH", "") // Do not use Unsetenv, incompatible with golang 1.3
		}
		if err != nil {
			return "", fmt.Errorf("failed to resolve %s", p)
		}
		return resolved, nil

	case strings.HasPrefix(p, homeVar):
		if u == nil {
			return p, nil
		}
		return path.Join(u.HomeDir, p[len(homeVar):]), nil

	case strings.Contains(p, displayVar):
		if d < 0 {
			return p, nil
		}
		return strings.Replace(p, displayVar, strconv.Itoa(d), -1), nil

	case strings.Contains(p, uidVar):
		if u == nil {
			return p, nil
		}
		return strings.Replace(p, uidVar, u.Uid, -1), nil

	case strings.Contains(p, userVar):
		if u == nil {
			return p, nil
		}
		return strings.Replace(p, userVar, u.Username, -1), nil
	case xdgdirs.IsXDGDir(p):
		if u == nil || xdgDirs == nil {
			return p, nil
		}
		return xdgDirs.ResolvePath(p), nil
	}
	return p, nil
}

func isGlobbed(p string) bool {
	return strings.Contains(p, "*")
}

func resolveGlob(p string) ([]string, error) {
	if !isGlobbed(p) {
		return []string{p}, nil
	}
	list, err := filepath.Glob(p)
	if err != nil {
		return nil, fmt.Errorf("failed to glob resolve %s: %v", p, err)
	}
	return list, nil
}
