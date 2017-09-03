package seccomp

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path"
	"syscall"

	"github.com/subgraph/oz"
	seccomp "github.com/twtiger/gosecco"

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

var log *logging.Logger

func init() {
	log = createLogger()
}

func Main() {

	modeptr := flag.String("mode", "whitelist", "Mode: whitelist, blacklist, train")
	policyptr := flag.String("policy", "", "Policy path")

	flag.Parse()

	args := flag.Args()

	if len(args) < 1 {
		log.Error("oz-seccomp: no command.")
		os.Exit(1)
	}

	cmdArgs := []string{args[0]}

	var settings seccomp.SeccompSettings


	cmd := args[0]
	cmdArgs = args
	fpath := ""

	oz.CheckSettingsOverRide()
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
	if *modeptr != "train" {
		if err := json.NewDecoder(os.Stdin).Decode(&p); err != nil {
			log.Error("unable to decode profile data: %v", err)
			os.Exit(1)
		}
	}

	switch *modeptr {
	case "train":

		settings.DefaultPositiveAction = "allow"
		settings.DefaultNegativeAction = "trace"
		settings.DefaultPolicyAction = "trace"

		if *policyptr == "" {
			fpath = path.Join(config.EtcPrefix, "training-generic.seccomp")
		} else {
			fpath = *policyptr
		}

		filter, err := seccomp.Prepare(fpath, settings)

		if err != nil {
			log.Error("[FATAL] Seccomp filter compile failed: %v", err)
			os.Exit(1)
		}
		err = seccomp.Install(filter)
		if err != nil {
			log.Error("[FATAL] Error (seccomp): %v", err)
			os.Exit(1)
		}
		err = syscall.Exec(cmd, cmdArgs, os.Environ())
		if err != nil {
			log.Error("[FATAL] Error (exec): %v %s", err, cmd)
			os.Exit(1)
		}
	case "whitelist":

		settings.ExtraDefinitions = p.Seccomp.ExtraDefs
		settings.DefaultPositiveAction = "allow"
		settings.DefaultNegativeAction = "kill"
		settings.DefaultPolicyAction = "kill"

		enforce := true
		fpath := ""
		if p.Seccomp.Mode == oz.PROFILE_SECCOMP_WHITELIST {
			if p.Seccomp.Whitelist == "" {
				log.Error("[FATAL] No seccomp policy file.")
				os.Exit(1)
			}
			fpath = p.Seccomp.Whitelist
			enforce = p.Seccomp.Enforce
		} else if p.Seccomp.Mode == oz.PROFILE_SECCOMP_TRAIN {
			if enforce == true {
				log.Error("Oz profile configured for seccomp enforcement while training. Enforce mode set to false.")
				enforce = false
			}
			fpath = path.Join(config.EtcPrefix, "training-generic.seccomp")
		}
		if enforce == false {
			settings.DefaultNegativeAction = "trace"
			settings.DefaultPolicyAction = "trace"
		}
		filter, err := seccomp.Prepare(fpath, settings)
		if err != nil {
			log.Error("[FATAL] Seccomp filter compile failed: %v", err)
			os.Exit(1)
		}
		err = seccomp.Install(filter)
		if err != nil {
			log.Error("[FATAL] Error (seccomp): %v", err)
			os.Exit(1)
		}
		err = syscall.Exec(cmd, cmdArgs, os.Environ())
		if err != nil {
			log.Error("[FATAL] Error (exec): %v %s", err, cmd)
			os.Exit(1)
		}
	case "blacklist":

		settings.ExtraDefinitions = p.Seccomp.ExtraDefs
		settings.DefaultPositiveAction = "kill"
		settings.DefaultNegativeAction = "allow"
		settings.DefaultPolicyAction = "allow"
		enforce := p.Seccomp.Enforce

		if p.Seccomp.Blacklist == "" {
			p.Seccomp.Blacklist = path.Join(config.EtcPrefix, "blacklist-generic.seccomp")
		}

		if enforce == false {
			settings.DefaultPositiveAction = "trace"
		}
		filter, err := seccomp.Prepare(p.Seccomp.Blacklist, settings)
		if err != nil {
			log.Error("[FATAL] Seccomp blacklist filter compile failed: %v", err)
			os.Exit(1)
		}
		err = seccomp.InstallBlacklist(filter)
		if err != nil {
			log.Error("[FATAL] Error (seccomp): %v", err)
			os.Exit(1)
		}
		log.Info("%s %v\n", cmd, cmdArgs)
		err = syscall.Exec(cmd, cmdArgs, os.Environ())
		if err != nil {
			log.Error("[FATAL] Error (exec): %v %s", err, cmd)
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
