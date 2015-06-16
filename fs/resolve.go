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
	const userVar = "${USER}"

	switch {
	case strings.HasPrefix(p, pathVar):
		resolved, err := exec.LookPath(p[len(pathVar):])
		if err != nil {
			return "", fmt.Errorf("failed to resolve %s", p)
		}
		return resolved, nil

	case strings.HasPrefix(p, homeVar):
		if fs.user == nil {
			return p, nil
		}
		return path.Join(fs.user.HomeDir, p[len(homeVar):]), nil

	case strings.Contains(p, uidVar):
		if fs.user == nil {
			return p, nil
		}
		return strings.Replace(p, uidVar, fs.user.Uid, -1), nil

	case strings.Contains(p, userVar):
		if fs.user == nil {
			return p, nil
		}
		return strings.Replace(p, userVar, fs.user.Username, -1), nil
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
