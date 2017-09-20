package oz

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path"
)

type Config struct {
	ProfileDir       string   `json:"profile_dir" desc:"Directory containing the sandbox profiles"`
	ShellPath        string   `json:"shell_path" desc:"Path of the shell used when entering a sandbox"`
	PrefixPath       string   `json:"prefix_path" desc:"Prefix path containing the oz executables"`
	EtcPrefix        string   `json:"etc_prefix" desc:"Prefix for configuration files"`
	SandboxPath      string   `json:"sandbox_path" desc:"Path of the sandboxes base"`
	OpenVPNRunPath   string   `json:"openvpn_run_path" desc: "Path for OpenVPN run state"`
	OpenVPNConfDir   string   `json:"openvpn_conf_dir" desc: "Path for OpenVPN conf files"`
	OpenVPNGroup     string   `json:"openvpn_group" desc: "GID for OpenVPN process"`
	RouteTableBase   int      `json:"route_table_base" desc: "Base for routing table"`
	DivertSuffix     string   `json:"divert_suffix" desc:"Suffix using for dpkg-divert of application executables, can be left empty when using a divert path"`
	DivertPath       bool     `json:"divert_path" desc:"Whether the diverted executable should be moved out of the path"`
	NMIgnoreFile     string   `json:"nm_ignore_file" desc:"Path to the NetworkManager ignore config file, disables the warning if empty"`
	UseFullDev       bool     `json:"use_full_dev" desc:"Give sandboxes full access to devices instead of a restricted set"`
	AllowRootShell   bool     `json:"allow_root_shell" desc:"Allow entering a sandbox shell as root"`
	LogXpra          bool     `json:"log_xpra" desc:"Log output of Xpra"`
	EnableEphemerals bool     `json:"enable_ephemerals" desc:"Enable prompting to launch sandbox in ephemeral mode"`
	EnvironmentVars  []string `json:"environment_vars" desc:"Default environment variables passed to sandboxes"`
	DefaultGroups    []string `json:"default_groups" desc:"List of default group names that can be used inside the sandbox"`
	EtcIncludes      []string `json:"etc_includes" desc:"Elements to include in the etc directory in the sandbox"`
}

const OzVersion = "0.0.1"

var DefaultConfigPath = "/etc/oz/oz.conf"

func CheckSettingsOverRide() {
	nConfPath := os.Getenv("OZ_CONFIG_PATH")

	if nConfPath != "" {
		DefaultConfigPath = nConfPath
	}
}

var DefaultEtcIncludes = []string{
	"/etc/alternatives/",
	"/etc/ssl/certs/",
	"/etc/console-setup/",
	"/etc/dbus-1/",
	"/etc/default/locale",
	"/etc/fonts/",
	"/etc/gnome/defaults.list",
	"/etc/group",
	//"/etc/group-",
	"/etc/gtk-2.0/",
	"/etc/gtk-3.0/",
	"/etc/host.conf",
	"/etc/inputrc",
	"/etc/locale.alias",
	"/etc/localtime",
	"/etc/magic",
	"/etc/magic.mime",
	"/etc/mailcap",
	"/etc/mailcap.order",
	"/etc/mime.types",
	"/etc/passwd",
	//"/etc/passwd-",
	"/etc/protocols",
	"/etc/pulse/",
	"/etc/resolvconf/run/resolv.conf",
	"/etc/services",
	"/etc/shells",
	"/etc/terminfo/",
	"/etc/timezone",
	"/etc/vconsole.conf",
	"/etc/xdg/-mimeapps.list",
	"/etc/xdg/user-dirs.conf",
	"/etc/xdg/user-dirs.defaults",
	"/etc/xpra/",
	"/etc/X11/",

	//"/etc/debian_version",
	//"/etc/os-release",
	//"/etc/issue",
	//"/etc/issue.net",
}

func NewDefaultConfig() *Config {
	return &Config{
		ProfileDir:       "/var/lib/oz/cells.d",
		ShellPath:        "/bin/bash",
		PrefixPath:       "/usr/local",
		EtcPrefix:        "/etc/oz",
		SandboxPath:      "/srv/oz",
		OpenVPNRunPath:   "/var/run/openvpn",
		OpenVPNConfDir:   "/var/lib/oz/openvpn",
		OpenVPNGroup:     "oz-openvpn",
		RouteTableBase:   8000,
		DivertPath:       true,
		NMIgnoreFile:     "/etc/NetworkManager/conf.d/oz.conf",
		DivertSuffix:     "",
		UseFullDev:       false,
		AllowRootShell:   false,
		LogXpra:          true,
		EnableEphemerals: false,
		EnvironmentVars: []string{
			"USER", "USERNAME", "LOGNAME",
			"LANG", "LANGUAGE", "_", "TZ=UTC",
			"XDG_SESSION_TYPE", "XDG_RUNTIME_DIR", "XDG_DATA_DIRS",
			"XDG_SEAT", "XDG_SESSION_TYPE", "XDG_SESSION_ID", "GNOME_DESKTOP_SESSION_ID=this-is-deprecated",
		},
		DefaultGroups: []string{
			"audio", "video",
		},
	}
}

func LoadConfig(cpath string) (*Config, error) {
	if _, err := os.Stat(cpath); os.IsNotExist(err) {
		return nil, err
	}
	if err := checkConfigPermissions(cpath); err != nil {
		return nil, err
	}

	bs, err := ioutil.ReadFile(cpath)
	if err != nil {
		return nil, err
	}
	c := NewDefaultConfig()
	if err := json.Unmarshal(bs, c); err != nil {
		return nil, err
	}

	if c.DivertSuffix == "" && c.DivertPath == false {
		c.DivertSuffix = "unsafe"
	}

	if len(c.EtcIncludes) == 0 {
		c.EtcIncludes = DefaultEtcIncludes
	} else {
		c.EtcIncludes = append(c.EtcIncludes, DefaultEtcIncludes...)
	}
	c.EtcIncludes = append(c.EtcIncludes, c.EtcPrefix)
	if c.EtcPrefix != path.Dir(DefaultConfigPath) {
		c.EtcIncludes = append(c.EtcIncludes, path.Dir(DefaultConfigPath))
	}
	return c, nil
}
