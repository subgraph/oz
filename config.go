package oz

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"syscall"
)

type Config struct {
	ProfileDir      string   `json:"profile_dir"`
	ShellPath       string   `json:"shell_path"`
	SandboxPath     string   `json:"sandbox_path"`
	BridgeMACAddr   string   `json:"bridge_mac"`
	NMIgnoreFile    string   `json:"nm_ignore_file"`
	UseFullDev      bool     `json:"use_full_dev"`
	AllowRootShell  bool     `json:"allow_root_shell"`
	LogXpra         bool     `json:"log_xpra"`
	EnvironmentVars []string `json:"environment_vars"`
}

const DefaultConfigPath = "/etc/oz/oz.conf"

func NewDefaultConfig() *Config {
	return &Config{
		ProfileDir:      "/var/lib/oz/cells.d",
		ShellPath:       "/bin/bash",
		SandboxPath:     "/srv/oz",
		NMIgnoreFile:    "/etc/NetworkManager/conf.d/oz.conf",
		BridgeMACAddr:   "6A:A8:2E:56:E8:9C",
		UseFullDev:      false,
		AllowRootShell:  false,
		LogXpra:         false,
		EnvironmentVars: []string{
			"USER", "USERNAME", "LOGNAME",
			"LANG", "LANGUAGE", "_",
		},
	}
}

func LoadConfig(cpath string) (*Config, error) {
	if _, err := os.Stat(cpath); os.IsNotExist(err) {
		return nil,err
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
	return c, nil
}

func checkConfigPermissions(fpath string) error {
	pd := path.Dir(fpath)
	for _, fp := range []string{pd, fpath} {
		if err := checkPathRootPermissions(fp); err != nil {
			return fmt.Errorf("file (%s) is %s", fp, err)
		}
	}
	return nil
}

func checkPathRootPermissions(fpath string) error {
	fstat, err := os.Stat(fpath)
	if err != nil {
		return err
	}
	if (fstat.Mode().Perm() & syscall.S_IWOTH) != 0 {
		return fmt.Errorf("writable by everyone!", fpath)
	}
	if (fstat.Mode().Perm() & syscall.S_IWGRP) != 0 && fstat.Sys().(*syscall.Stat_t).Gid != 0 {
		return fmt.Errorf("writable by someone else than root!", err)
	}
	return nil
}
