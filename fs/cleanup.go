package fs

import (
	"errors"
	"io/ioutil"
	"os"
	"sort"
	"strings"
	"syscall"
	
	// External
	"github.com/op/go-logging"
)

func (fs *Filesystem) Cleanup() error {
	if fs.base == "" {
		msg := "cannot Cleanup() filesystem, fs.base is empty"
		fs.log.Warning(msg)
		return errors.New(msg)
	}
	fs.log.Info("Cleanup() called on filesystem at root %s", fs.root)

	for {
		mnts, err := getMountsBelow(fs.base)
		if err != nil {
			return err
		}
		if len(mnts) == 0 {
			return nil
		}
		atLeastOne, lastErr := mnts.unmountAll(fs.log)
		if !atLeastOne {
			return lastErr
		}
	}
}

func (mnts mountEntries) unmountAll(log *logging.Logger) (bool, error) {
	reterr := error(nil)
	atLeastOne := false
	for _, m := range mnts {
		log.Debug("Unmounting mountpoint: %s", m.dir)
		if _, err := os.Stat(m.dir); os.IsNotExist(err) {
			continue
		}
		if err := syscall.Unmount(m.dir, 0); err != nil {
			log.Warning("Failed to unmount mountpoint %s: %v", m.dir, err)
			reterr = err
		} else {
			atLeastOne = true
		}
	}
	return atLeastOne, reterr
}

type mountEntry struct {
	src     string
	dir     string
	fs      string
	options string
}

type mountEntries []*mountEntry

func (m mountEntries) Len() int           { return len(m) }
func (m mountEntries) Swap(i, j int)      { m[i], m[j] = m[j], m[i] }
func (m mountEntries) Less(i, j int) bool { return m[i].depth() > m[j].depth() }

func (me mountEntry) depth() int { return strings.Count(me.dir, "/") }

func getMountsBelow(base string) (mountEntries, error) {
	mnts, err := getProcMounts()
	if err != nil {
		return nil, err
	}
	sort.Sort(mnts)
	var filtered mountEntries
	for _, m := range mnts {
		if strings.HasPrefix(m.dir, base) {
			filtered = append(filtered, m)
		}
	}
	return filtered, nil
}

func (m mountEntries) contains(dir string) bool {
	for _, mnt := range m {
		if dir == mnt.dir {
			return true
		}
	}
	return false
}

func getProcMounts() (mountEntries, error) {
	lines, err := readProcMounts()
	if err != nil {
		return nil, err
	}
	var entries mountEntries
	for _, line := range lines {
		parts := strings.Fields(line)
		if len(parts) >= 4 {
			entries = append(entries, &mountEntry{
				src:     parts[0],
				dir:     parts[1],
				fs:      parts[2],
				options: parts[3],
			})
		}

	}
	return entries, nil
}

func readProcMounts() ([]string, error) {
	content, err := ioutil.ReadFile("/proc/mounts")
	if err != nil {
		return nil, err
	}
	return strings.Split(string(content), "\n"), nil
}
