package oz

import (
	"encoding/json"
	"io/ioutil"
)

type Config struct {
	ProfileDir     	string `json:"profile_dir"`
	ShellPath      	string `json:"shell_path"`
	SandboxPath    	string `json:"sandbox_path"`
	AllowRootShell 	bool   `json:"allow_root_shell"`
	LogXpra        	bool   `json:"log_xpra"`
}

const DefaultConfigPath = "/etc/oz/oz.conf"

func NewDefaultConfig() *Config {
	return &Config{
		ProfileDir:     "/var/lib/oz/cells.d",
		ShellPath:      "/bin/bash",
		SandboxPath:    "/srv/oz",
		AllowRootShell: false,
		LogXpra:        false,
	}
}

func LoadConfig(path string) (*Config, error) {
	bs, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	c := NewDefaultConfig()
	if err := json.Unmarshal(bs, c); err != nil {
		return nil, err
	}
	return c, nil
}
