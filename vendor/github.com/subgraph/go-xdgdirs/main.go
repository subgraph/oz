package xdgdirs

// https://wiki.archlinux.org/index.php/Xdg_user_directories
// https://askubuntu.com/questions/457047/how-can-i-get-the-xdg-default-user-directories-from-python
// https://standards.freedesktop.org/basedir-spec/basedir-spec-latest.html
// https://github.com/cep21/xdgbasedir
// https://github.com/BurntSushi/xdg

import (
	"bufio"
	//"fmt"
	"os"
	"path"
	"regexp"
	"strings"

	"github.com/BurntSushi/xdg"
)

const (
	FILE_DIRS = "user-dirs"
	SUFFIX_USER = "dirs"
	SUFFIX_GLOBAL = "defaults"
	
)

var xdgVarRegexp = regexp.MustCompile("^(\\${XDG_([A-Z0-9_-]+)_DIR})/?.*") 

var commentRegexp = regexp.MustCompile("^[ \t]*#")

type Dirs struct {
	xdgPaths      *xdg.Paths
	xdgHome       string
	xdgDirs       map[string]string
	xdgUserConf   string
	xdgGlobalConf string
}

func (x *Dirs) Load(home string) *Dirs {
	var cf string
	x.xdgUserConf = strings.Join([]string{FILE_DIRS, SUFFIX_USER}, ".")
	x.xdgGlobalConf = strings.Join([]string{FILE_DIRS, SUFFIX_GLOBAL}, ".")
	x.xdgPaths = new(xdg.Paths)
	x.xdgDirs = make(map[string]string)
	
	x.xdgHome = home
	if x.xdgHome == "" {
		x.xdgHome = os.Getenv("HOME")
	}
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", x.xdgHome)
	cf, _ = x.xdgPaths.ConfigFile(x.xdgUserConf)
	if cf == "" {
		cf, _ = x.xdgPaths.ConfigFile(x.xdgGlobalConf)
	}
	os.Setenv("HOME", oldHome)

	if cf != "" {
		x.loadUserDirs(cf)
		return x
	}
	return nil
}

func readCommentedFile(fpath string) (string, error) {
	file, err := os.Open(fpath)
	if err != nil {
		return "", err
	}
	scanner := bufio.NewScanner(file)
	bs := ""
	for scanner.Scan() {
		line := scanner.Text()
		if !commentRegexp.MatchString(line) {
			bs += line + "\n"
		}
	}

	return bs, nil
}

func (x *Dirs)loadUserDirs(fpath string) error {
	bs, err := readCommentedFile(fpath)
	if err != nil {
		panic(err)
	}
	for _, line := range strings.Split(bs, "\n") {
		toks := strings.Split(line, "=")
		if toks == nil || len(toks) <= 1 {
			continue
		}
		vn := toks[0]
		vns := strings.Split(toks[0], "_")
		if len(vns) > 2 {
			vn = vns[1]
		}
		vl := strings.Trim(toks[1], "\"")
		if !strings.HasPrefix(vl, "$HOME") {
			vl = path.Join("$HOME", vl)
		}
		x.xdgDirs[vn] = vl
	}

	return nil
}

func (x *Dirs) GetDirs() map[string]string {
	return x.xdgDirs
}

func (x *Dirs)GetDir(name string) string {
	vn := name
	if strings.HasPrefix(vn, "XDG_") {
		vn = strings.Join(strings.Split(vn, "_")[1:], "")
	}
	if strings.HasSuffix(vn, "_DIR") {
		vn = strings.Split(vn, "_")[0]
	}
	dirName := x.xdgDirs[name]
	if dirName != "" {
		if !strings.HasPrefix(dirName, "$HOME") {
			dirName = path.Join("$HOME", dirName)
		}
		dirName = strings.Replace(dirName, "$HOME", x.xdgHome, 1)
	}
	return dirName
}

func (x *Dirs) ResolvePath(xdgPath string) string {
	for dir, _ := range x.xdgDirs {
		xdgdir := x.GetDir(dir)
		if xdgdir != dir {
			xdgPath = strings.Replace(xdgPath, "${XDG_"+dir+"_DIR}", x.GetDir(dir), 1)
		} else {
			xdgPath = xdgdir
		}
	}
	return xdgPath
}

func IsXDGDir(s string) bool {
	return xdgVarRegexp.MatchString(s)
}

