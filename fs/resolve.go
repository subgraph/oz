package fs

import (
	"fmt"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
)

func (fs *Filesystem) resolvePath(p string) ([]string, error) {
	p, err := fs.resolveVars(p)
	if err != nil {
		return nil, err
	}
	return fs.resolveGlob(p)
}

func (fs *Filesystem) resolveVars(p string) (string, error) {
	const pathVar = "${PATH}/"
	const homeVar = "${HOME}"
	const uidVar = "${UID}"

	switch {
	case strings.HasPrefix(p, pathVar):
		resolved, err := exec.LookPath(p[len(pathVar):])
		if err != nil {
			return "", fmt.Errorf("failed to resolve %s", p)
		}
		return resolved, nil

	case strings.HasPrefix(p, homeVar):
		return path.Join(fs.user.HomeDir, p[len(homeVar):]), nil

	case strings.HasPrefix(p, uidVar):
		return strings.Replace(p, uidVar, fs.userID, -1), nil
	}
	return p, nil
}

func (fs *Filesystem) resolveGlob(p string) ([]string, error) {
	if !strings.Contains(p, "*") {
		return []string{p}, nil
	}
	list, err := filepath.Glob(p)
	if err != nil {
		return nil, fmt.Errorf("failed to glob resolve %s: %v", p, err)
	}
	return list, nil
}
