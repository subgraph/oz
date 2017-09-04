package seccomp

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
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
	policyptr := flag.String("policy", "", "seccomp policy path")
	profilepath := flag.String("profile", "", "optional seccomp profile path")
	newprivs := flag.Bool("allow-new-privs", false, "allow traced program to set new seccomp filters")

	flag.Parse()

	args := flag.Args()

	var settings seccomp.SeccompSettings

	if len(args) < 1 {
		log.Fatal("oz-seccomp: must specify a command to be traced.")
	}

	cmd := args[0]
	cmdArgs := args
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

		if *profilepath != "" && *profilepath != "-" {
			fbytes, err := ioutil.ReadFile(*profilepath)
			if err != nil {
				log.Fatal("unable to read profile data from file: ", err)
			}
			if err := json.Unmarshal(fbytes, &p); err != nil {
				log.Fatal("unable to decode profile data from file: ", err)
			}
		} else {
			if err := json.NewDecoder(os.Stdin).Decode(&p); err != nil {
				log.Fatal("unable to decode profile data: ", err)
			}
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
			log.Fatal("[FATAL] Seccomp filter compile failed: ", err)
		}
		if *newprivs {
			err = seccomp.LockedLoad(filter)
		} else {
			err = seccomp.Install(filter)
		}
		if err != nil {
			if *newprivs {
				log.Fatal("[FATAL] Error loading seccomp filter: ", err)
			} else {
				log.Fatal("[FATAL] Error installing seccomp filter: ", err)
			}
		}
		err = syscall.Exec(cmd, cmdArgs, os.Environ())
		if err != nil {
			log.Fatal("[FATAL] Error (exec): ", err, " / ", cmd)
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
				log.Fatal("[FATAL] profile referenced no seccomp whitelist policy file.")
			}
			fpath = p.Seccomp.Whitelist
			enforce = p.Seccomp.Enforce
		} else if p.Seccomp.Mode == oz.PROFILE_SECCOMP_TRAIN {
			if enforce == true {
				log.Error("Oz profile configured for seccomp enforcement while training. Enforce mode set to false.")
				enforce = false
			}
			fpath = path.Join(config.EtcPrefix, "training-generic.seccomp")
		} else if p.Seccomp.Mode == oz.PROFILE_SECCOMP_DISABLED {
			log.Fatal("Cannot run seccomp in whitelist mode if seccomp is disabled in profile.")
		}

		if enforce == false {
			settings.DefaultNegativeAction = "trace"
			settings.DefaultPolicyAction = "trace"
		}
		filter, err := seccomp.Prepare(fpath, settings)
		if err != nil {
			log.Fatal("[FATAL] Seccomp filter compile failed: ", err)
		}
		if *newprivs {
			err = seccomp.LockedLoad(filter)
		} else {
			err = seccomp.Install(filter)
		}
		if err != nil {
			if *newprivs {
				log.Fatal("[FATAL] Error loading seccomp filter: ", err)
			} else {
				log.Fatal("[FATAL] Error installing seccomp filter: ", err)
			}
		}
		err = syscall.Exec(cmd, cmdArgs, os.Environ())
		if err != nil {
			log.Fatal("[FATAL] Error (exec): ", err, " / ", cmd)
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
			log.Fatal("[FATAL] Seccomp blacklist filter compile failed: ", err)
		}
		err = seccomp.InstallBlacklist(filter)
		if err != nil {
			log.Fatal("[FATAL] Error installing seccomp blacklist: ", err)
		}
		log.Info("%s %v\n", cmd, cmdArgs)
		err = syscall.Exec(cmd, cmdArgs, os.Environ())
		if err != nil {
			log.Fatal("[FATAL] Error (exec): ", err, " / ", cmd)
		}
	default:
		log.Fatal("Invalid mode specified (must be whitelist, blacklist, or train)")
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
