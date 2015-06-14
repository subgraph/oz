package oz

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path"
)

type Profile struct {
	// Name of this profile
	Name string
	// Path to binary to launch
	Path string
	// Optional path of binary to watch for watchdog purposes if different than Path
	Watchdog string
	// Optional wrapper binary to use when launching command (ex: tsocks)
	Wrapper string
	// If true launch one container per instance, otherwise run all instances in same container
	Multi bool
	// Disable mounting of sys and proc inside the container
	NoSysProc bool
	// Disable bind mounting of default directories (etc,usr,bin,lib,lib64)
	// Also disables default blacklist items (/sbin, /usr/sbin, /usr/bin/sudo)
	// Normally not used
	NoDefaults bool
	// List of paths to bind mount inside jail
	Whitelist []WhitelistItem
	// List of paths to blacklist inside jail
	Blacklist []BlacklistItem
	// Optional XServer config
	XServer XServerConf
	// List of environment variables
	Environment []EnvVar
	// Networking
	Networking NetworkProfile
}

type XServerConf struct {
	Enabled          bool
	TrayIcon         string `json:"tray_icon"`
	WindowIcon       string `json:"window_icon"`
	EnableTray       bool   `json:"enable_tray"`
	UseDBUS          bool   `json:"use_dbus"`
	UsePulseAudio    bool   `json:"use_pulse_audio"`
	DisableClipboard bool   `json:"disable_clipboard"`
	DisableAudio     bool   `json:"disable_audio"`
}

type WhitelistItem struct {
	Path     string
	ReadOnly bool
}

type BlacklistItem struct {
	Path string
}

type EnvVar struct {
	Name  string
	Value string
}

// Container network definition
type NetworkProfile struct {
	// One of empty, host, bridge
	Nettype string

	// Name of the bridge to attach to
	//Bridge string

	// List of Sockets we want to attach to the jail
	//  Applies to Nettype: bridge and empty only
	//Sockets []nsnat.ConfigSocket
}

const defaultProfileDirectory = "/var/lib/oz/cells.d"

var loadedProfiles []*Profile

type Profiles []*Profile

func (ps Profiles) GetProfileByName(name string) (*Profile, error) {
	if loadedProfiles == nil {
		ps, err := LoadProfiles(defaultProfileDirectory)
		if err != nil {
			return nil, err
		}
		loadedProfiles = ps
	}

	for _, p := range loadedProfiles {
		if p.Name == name {
			return p, nil
		}
	}
	return nil, nil
}

func (ps Profiles) GetProfileByPath(bpath string) (*Profile, error) {
	if loadedProfiles == nil {
		ps, err := LoadProfiles(defaultProfileDirectory)
		if err != nil {
			return nil, err
		}
		loadedProfiles = ps
	}

	for _, p := range loadedProfiles {
		if p.Path == bpath {
			return p, nil
		}
	}
	return nil, nil
}

func LoadProfiles(dir string) (Profiles, error) {
	fs, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	ps := []*Profile{}
	for _, f := range fs {
		if !f.IsDir() {
			name := path.Join(dir, f.Name())
			p, err := loadProfileFile(name)
			if err != nil {
				return nil, fmt.Errorf("error loading '%s': %v", f.Name(), err)
			}
			ps = append(ps, p)
		}
	}
	
	loadedProfiles = ps
	return ps, nil
}

func loadProfileFile(file string) (*Profile, error) {
	bs, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}
	p := new(Profile)
	if err := json.Unmarshal(bs, p); err != nil {
		return nil, err
	}
	if p.Name == "" {
		p.Name = path.Base(p.Path)
	}
	return p, nil
}
