package oz

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"regexp"
	"strings"

	"github.com/subgraph/oz/network"
)

type Profile struct {
	// Name of this profile
	Name string
	// Path to binary to launch
	Path string
	// List of path to binaries matching this sandbox
	Paths []string
	// Path of the config file
	ProfilePath string `json:"-"`
	// Default parameters to pass to the program
	DefaultParams []string `json:"default_params"`
	// Pass command-line arguments
	RejectUserArgs bool `json:"reject_user_args"`
	// Autoshutdown the sandbox when the process exits. One of (no, yes, soft), defaults to yes
	AutoShutdown ShutdownMode `json:"auto_shutdown"`
	// Optional list of executable names to watch for exit in case initial command spawns and exit
	Watchdog []string
	// Optional wrapper binary to use when launching command (ex: tsocks)
	Wrapper string
	// If true launch one sandbox per instance, otherwise run all instances in same sandbox
	Multi bool
	// Disable mounting of sys and proc inside the sandbox
	NoSysProc bool
	// Disable bind mounting of default directories (etc,usr,bin,lib,lib64)
	// Also disables default blacklist items (/sbin, /usr/sbin, /usr/bin/sudo)
	// Normally not used
	NoDefaults bool
	// Allow bind mounting of files passed as arguments inside the sandbox
	AllowFiles    bool     `json:"allow_files"`
	AllowedGroups []string `json:"allowed_groups"`
	// Optional directory where per-process logs will be output
	LogDir string `json:"log_dir"`
	// List of paths to bind mount inside jail
	Whitelist []WhitelistItem
	// List of paths to blacklist inside jail
	Blacklist []BlacklistItem
	// Shared Folders
	SharedFolders []string `json:"shared_folders"`
	// Optional XServer config
	XServer XServerConf
	// List of environment variables
	Environment []EnvVar
	// Networking
	Networking NetworkProfile
	// Firewall
	Firewall []FWRule
	// Seccomp
	Seccomp SeccompConf
	// External Forwarders
	ExternalForwarders []ExternalForwarder `json:"external_forwarders"`
}

type ShutdownMode string

const (
	PROFILE_SHUTDOWN_NO  ShutdownMode = "no"
	PROFILE_SHUTDOWN_YES ShutdownMode = "yes"
	//PROFILE_SHUTDOWN_SOFT     ShutdownMode = "soft" // Unimplemented
)

type AudioMode string

const (
	PROFILE_AUDIO_NONE    AudioMode = "none"
	PROFILE_AUDIO_SPEAKER AudioMode = "speaker"
	PROFILE_AUDIO_FULL    AudioMode = "full"
	PROFILE_AUDIO_PULSE   AudioMode = "pulseaudio"
)

type XServerConf struct {
	Enabled             bool
	TrayIcon            string    `json:"tray_icon"`
	WindowIcon          string    `json:"window_icon"`
	EnableTray          bool      `json:"enable_tray"`
	EnableNotifications bool      `json:"enable_notifications"`
	DisableClipboard    bool      `json:"disable_clipboard"`
	AudioMode           AudioMode `json:"audio_mode"`
	PulseAudio          bool      `json:"pulseaudio"`
	Border              bool      `json:"border"`
	Environment         []EnvVar  `json:"env"`
}

type SeccompMode string

const (
	PROFILE_SECCOMP_TRAIN     SeccompMode = "train"
	PROFILE_SECCOMP_WHITELIST SeccompMode = "whitelist"
	PROFILE_SECCOMP_BLACKLIST SeccompMode = "blacklist"
	PROFILE_SECCOMP_DISABLED  SeccompMode = "disabled"
)

type SeccompConf struct {
	Mode        SeccompMode
	Enforce     bool
	Debug       bool
	Train       bool
	TrainOutput string `json:"train_output"`
	Whitelist   string
	Blacklist   string
	ExtraDefs   []string
}

type VPNConf struct {
	VpnType          string `json:"type"`
	ConfigPath       string
	DNS              []string
	UserPassFilePath string `json:"authfile"`
}

type ExternalForwarder struct {
	Name        string
	Dynamic     bool
	Multi       bool
	ExtProto    string
	Proto       string
	Addr        string
	TargetHost  string
	TargetPort  string
	SocketOwner string
}

type WhitelistItem struct {
	Path        string
	Target      string
	Symlink     string `json:-"`
	ReadOnly    bool   `json:"read_only"`
	CanCreate   bool   `json:"can_create"`
	Ignore      bool   `json:"ignore"`
	Force       bool
	NoFollow    bool `json:"no_follow"`
	AllowSetuid bool `json:"allow_suid"`
}

type BlacklistItem struct {
	Path     string
	NoFollow bool `json:"no_follow"`
}

type FWRule struct {
	Whitelist bool   `json:"whitelist"`
	DstHost   string `json:"dst_host"`
	DstPort   int    `json:"dst_port"`
}

type EnvVar struct {
	Name  string
	Value string
}

type DNSMode string

const (
	PROFILE_NETWORK_DNS_NONE DNSMode = "none"
	PROFILE_NETWORK_DNS_PASS DNSMode = "pass"
	PROFILE_NETWORK_DNS_DHCP DNSMode = "dhcp"
)

// Sandbox network definition
type NetworkProfile struct {
	// One of empty, host, bridge
	Nettype network.NetType `json:"type"`

	// Name of the bridge to attach to
	Bridge string

	// VPN type
	VPNConf VPNConf `json:"vpn"`

	// List of Sockets we want to attach to the jail
	//  Applies to Nettype: bridge and empty only
	Sockets []network.ProxyConfig

	// Hardcoded least significant byte of the IP address
	//  Applies to Nettype: bridge only
	IpByte uint `json:"ip_byte"`

	// DNS Mode one of: pass, none, dhcp
	//  Applies to Nettype: bridge only
	DNSMode DNSMode `json:"dns_mode"`

	// Additional data for the hosts file
	Hosts string
}

const defaultProfileDirectory = "/var/lib/oz/cells.d"

var loadedProfiles []*Profile

type Profiles []*Profile

func NewDefaultProfile() *Profile {
	return &Profile{
		Multi:         false,
		AllowFiles:    false,
		AllowedGroups: []string{},
		XServer: XServerConf{
			Enabled:             true,
			EnableTray:          false,
			EnableNotifications: false,
			AudioMode:           PROFILE_AUDIO_NONE,
			Border:              false,
		},
	}
}

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
		for _, pp := range p.Paths {
			if pp == bpath {
				return p, nil
			}
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
			if strings.HasSuffix(f.Name(), ".json") {
				p, err := loadProfileFile(name)
				if err != nil {
					return nil, fmt.Errorf("error loading '%s': %v", f.Name(), err)
				}
				ps = append(ps, p)
			}
		}
	}

	loadedProfiles = ps
	return ps, nil
}

var commentRegexp = regexp.MustCompile("^[ \t]*#")

func loadProfileFile(fpath string) (*Profile, error) {
	if err := checkConfigPermissions(fpath); err != nil {
		return nil, err
	}

	file, err := os.Open(fpath)
	if err != nil {
		return nil, err
	}
	scanner := bufio.NewScanner(file)
	bs := ""
	for scanner.Scan() {
		line := scanner.Text()
		if !commentRegexp.MatchString(line) {
			bs += line + "\n"
		}
	}
	p := new(Profile)
	if err := json.Unmarshal([]byte(bs), p); err != nil {
		return nil, err
	}
	if p.Name == "" {
		p.Name = path.Base(p.Path)
	}
	if p.AutoShutdown == "" {
		p.AutoShutdown = PROFILE_SHUTDOWN_YES
	}
	if p.XServer.AudioMode == "" {
		p.XServer.AudioMode = PROFILE_AUDIO_NONE
	}
	if p.Seccomp.Mode == "" {
		p.Seccomp.Mode = PROFILE_SECCOMP_DISABLED
	}
	if p.Networking.IpByte <= 1 || p.Networking.IpByte > 254 {
		p.Networking.IpByte = 0
	}
	p.ProfilePath = fpath
	return p, nil
}
