package seccomp

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"syscall"

	"github.com/subgraph/go-seccomp"
	"github.com/subgraph/oz"

	"github.com/op/go-logging"
)

func createLogger() *logging.Logger {
	l := logging.MustGetLogger("seccomp-wrapper")
	be := logging.NewLogBackend(os.Stderr, "", 0)
	f := logging.MustStringFormatter("%{level:.1s} %{message}")
	fbe := logging.NewBackendFormatter(be, f)
	logging.SetBackend(fbe)
	return l
}

func Main() {
	log := createLogger()

	if len(os.Args) < 3 {
		log.Error("seccomp-wrapper: Not enough arguments.")
		os.Exit(1)
	}

	if os.Getppid() != 1 {
		log.Error("oz-seccomp wrapper must be called from oz-init!")
		os.Exit(1)
	}

	cmd := os.Args[2]
	cmdArgs := os.Args[2:]

	config, err := oz.LoadConfig(oz.DefaultConfigPath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Info("Configuration file (%s) is missing, using defaults.", oz.DefaultConfigPath)
			config = oz.NewDefaultConfig()
		} else {
			log.Error("Could not load configuration: %s", oz.DefaultConfigPath, err)
			os.Exit(1)
		}
	}

	p := new(oz.Profile)
	if err := json.NewDecoder(os.Stdin).Decode(&p); err != nil {
		log.Error("unable to decode profile data: %v", err)
		os.Exit(1)
	}
/*
	p, err := loadProfile(config.ProfileDir, pname)

	if err != nil {
		log.Error("Could not load profile %s: %v", pname, err)
		os.Exit(1)
	}
*/
	switch os.Args[1] {
	case "-w":
		if p.Seccomp.Seccomp_Whitelist == "" {
			log.Error("No seccomp policy file.")
			os.Exit(1)
		}
		filter, err := seccomp.Compile(p.Seccomp.Seccomp_Whitelist)
		if err != nil {
			log.Error("Seccomp filter compile failed: %v", err)
			os.Exit(1)
		}
		err = seccomp.Install(filter)
		if err != nil {
			log.Error("Error (seccomp): %v", err)
			os.Exit(1)
		}
		err = syscall.Exec(cmd, cmdArgs, os.Environ())
		if err != nil {
			log.Error("Error (exec): %v", err)
			os.Exit(1)
		}
	case "-b":
		if p.Seccomp.Seccomp_Blacklist == "" {
			p.Seccomp.Seccomp_Blacklist = path.Join(config.EtcPrefix, "blacklist-generic.seccomp")
		}
		filter, err := seccomp.CompileBlacklist(p.Seccomp.Seccomp_Blacklist)
		if err != nil {
			log.Error("Seccomp blacklist filter compile failed: %v", err)
			os.Exit(1)
		}
		err = seccomp.InstallBlacklist(filter)
		if err != nil {
			log.Error("Error (seccomp): %v", err)
			os.Exit(1)
		}
		err = syscall.Exec(cmd, cmdArgs, os.Environ())
		if err != nil {
			log.Error("Error (exec): %v", err)
			os.Exit(1)
		}
	default:
		fmt.Println("Bad switch.")
		os.Exit(1)
	}
}

func loadProfile(dir, name string) (*oz.Profile, error) {
	ps, err := oz.LoadProfiles(dir)
	if err != nil {
		return nil, err
	}
	for _, p := range ps {
		if name == p.Name {
			return p, nil
		}
	}
	return nil, fmt.Errorf("no profile named '%s'", name)
}
