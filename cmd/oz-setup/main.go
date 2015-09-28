package main

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"reflect"
	"strconv"
	"strings"
	"syscall"

	"github.com/subgraph/oz"

	"github.com/codegangsta/cli"
)

var PathDpkgDivert string
var PathDpkg string
var OzConfig *oz.Config
var OzProfiles *oz.Profiles
var OzProfile *oz.Profile

func init() {
	checkRoot()
	PathDpkgDivert = checkDpkgDivert()
	PathDpkg = checkDpkg()
}

func main() {
	app := cli.NewApp()

	app.Name = "oz-utils"
	app.Usage = "command line interface to install, remove, and create Oz sandboxes\nYou can specify a package name, a binary path, or a Oz profile file "
	app.Author = "Subgraph"
	app.Email = "info@subgraph.com"
	app.Version = oz.OzVersion
	app.EnableBashCompletion = true

	flagsHookMode := []cli.Flag{
		cli.BoolFlag{
			Name:  "hook",
			Usage: "Run in hook mode, not normally used by the end user",
		},
	}

	flagsForce := []cli.Flag{
		cli.BoolFlag{
			Name:  "force, f",
			Usage: "Force the command to run through non fatal errors",
		},
	}

	app.Commands = []cli.Command{
		{
			Name:  "config",
			Usage: "check and show Oz configurations",
			Subcommands: []cli.Command{
				{
					Name:   "check",
					Usage:  "check oz configuration and profiles for errors",
					Action: handleConfigcheck,
				},
				{
					Name:   "show",
					Usage:  "prints ouf oz configuration",
					Action: handleConfigshow,
				},
			},
		},
		{
			Name:   "install",
			Usage:  "install binary diversion for a program",
			Action: handleInstall,
			Flags:  append(flagsForce, flagsHookMode...),
		},
		{
			Name:   "remove",
			Usage:  "remove a binary diversion for a program",
			Action: handleRemove,
			Flags:  append(flagsForce, flagsHookMode...),
		},
		{
			Name:   "status",
			Usage:  "show the status of a binary diversion for a program",
			Action: handleStatus,
		},
		{
			Name:   "create",
			Usage:  "create a new sandbox profile",
			Action: handleCreate,
		},
	}

	app.Run(os.Args)
}

func handleConfigcheck(c *cli.Context) {
	_, err := oz.LoadConfig(oz.DefaultConfigPath)
	if err != nil {
		if !os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "Could not load configuration `%s`: %v\n", oz.DefaultConfigPath, err)
			os.Exit(1)
		}
	}

	OzConfig = loadConfig()
	_, err = oz.LoadProfiles(OzConfig.ProfileDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to load profiles from `%s`: %v\n", OzConfig.ProfileDir, err)
		os.Exit(1)
	}

	fmt.Println("Configurations and profiles ok!")
	os.Exit(0)
}

func handleConfigshow(c *cli.Context) {
	config, err := oz.LoadConfig(oz.DefaultConfigPath)
	useDefaults := false
	if err != nil {
		if os.IsNotExist(err) {
			config = oz.NewDefaultConfig()
			useDefaults = true
		} else {
			fmt.Fprintf(os.Stderr, "Could not load configuration `%s`: %v\n", oz.DefaultConfigPath, err)
			os.Exit(1)
		}
	}

	v := reflect.ValueOf(*config)
	vt := reflect.TypeOf(*config)
	maxFieldLength := 0
	for i := 0; i < v.NumField(); i++ {
		flen := len(vt.Field(i).Tag.Get("json"))
		if flen > maxFieldLength {
			maxFieldLength = flen
		}
	}
	maxValueLength := 0
	for i := 0; i < v.NumField(); i++ {
		sval := fmt.Sprintf("%v", v.Field(i).Interface())
		flen := len(sval)
		if flen > maxValueLength {
			maxValueLength = flen
		}
	}

	sfmt := "%-" + strconv.Itoa(maxFieldLength) + "s: %-" + strconv.Itoa(maxValueLength) + "v"
	hfmt := "%-" + strconv.Itoa(maxFieldLength) + "s: %s\n"

	if !useDefaults {
		fmt.Printf(hfmt, "Config file", oz.DefaultConfigPath)
	} else {
		fmt.Printf(hfmt, "Config file", "Not found - using defaults")
	}

	for i := 0; i < len(fmt.Sprintf(sfmt, "", ""))+2; i++ {
		fmt.Print("#")
	}
	fmt.Println("")

	for i := 0; i < v.NumField(); i++ {
		fval := fmt.Sprintf("%v", v.Field(i).Interface())
		fmt.Printf(sfmt, vt.Field(i).Tag.Get("json"), fval)
		desc := vt.Field(i).Tag.Get("desc")
		if desc != "" {
			fmt.Printf(" # %s", desc)
		}

		fmt.Println("")
	}

	os.Exit(0)
}

func handleInstall(c *cli.Context) {
	OzConfig = loadConfig()
	pname := c.Args()[0]
	OzProfile, err := loadProfile(pname, OzConfig.ProfileDir)
	if err != nil || OzProfile == nil {
		installExit(c.Bool("hook"), fmt.Errorf("Unable to load profiles for %s (%v).\n", pname, err))
		return // For clarity
	}

	if OzConfig.DivertSuffix == "" && OzConfig.DivertPath == false {
		installExit(c.Bool("hook"), fmt.Errorf("Divert requires a suffix to be set.\n"))
		return // For clarity
	}

	divertInstall := func(cpath string) {
		isInstalled, err := isDivertInstalled(cpath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unknown error: %+v\n", err)
			os.Exit(1)
		}
		if isInstalled == true {
			fmt.Println("Divert already installed for ", cpath)
			if c.Bool("force") {
				return
			}
			os.Exit(0)
		}

		dpkgArgs := []string{
			"--add",
			"--package",
			"oz",
			"--rename",
			"--divert",
			getBinaryPath(cpath),
			cpath,
		}

		cdir := path.Dir(getBinaryPath(cpath))
		if _, err := os.Stat(cdir); os.IsNotExist(err) {
			um := syscall.Umask(0)
			os.Mkdir(cdir, 0755)
			os.Chown(cdir, 0, 0)
			syscall.Umask(um)
		}

		_, err = exec.Command(PathDpkgDivert, dpkgArgs...).Output()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Dpkg divert command `%s %+s` failed: %s", PathDpkgDivert, dpkgArgs, err)
			os.Exit(1)
		}

		clientbin := path.Join(OzConfig.PrefixPath, "bin", "oz")
		err = syscall.Symlink(clientbin, cpath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create symlink %s", err)
			os.Exit(1)
		}

		fmt.Printf("Successfully installed Oz sandbox for: %s.\n", cpath)
	}

	paths := append([]string{}, OzProfile.Path)
	paths = append(paths, OzProfile.Paths...)
	for _, pp := range paths {
		divertInstall(pp)
	}

	fmt.Printf("Successfully installed Oz sandbox for: %s.\n", OzProfile.Name)
}

func handleRemove(c *cli.Context) {
	OzConfig = loadConfig()
	pname := c.Args()[0]
	OzProfile, err := loadProfile(pname, OzConfig.ProfileDir)
	if err != nil || OzProfile == nil {
		installExit(c.Bool("hook"), fmt.Errorf("Unable to load profiles for %s.\n", pname))
		return // For clarity
	}

	if OzConfig.DivertSuffix == "" && OzConfig.DivertPath == false {
		installExit(c.Bool("hook"), fmt.Errorf("Divert requires a suffix to be set.\n"))
		return // For clarity
	}

	divertRemove := func(cpath string) {
		isInstalled, err := isDivertInstalled(cpath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unknown error: %+v\n", err)
			os.Exit(1)
		}
		if isInstalled == false {
			fmt.Println("Divert is not installed for ", cpath)
			if c.Bool("force") {
				return
			}
			os.Exit(0)
		}

		os.Remove(cpath)

		dpkgArgs := []string{
			"--rename",
			"--package",
			"oz",
			"--remove",
			cpath,
		}

		_, err = exec.Command(PathDpkgDivert, dpkgArgs...).Output()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Dpkg divert command `%s %+s` failed: %s", PathDpkgDivert, dpkgArgs, err)
			os.Exit(1)
		}

		fmt.Printf("Successfully remove jail for: %s.\n", cpath)
	}

	paths := append([]string{}, OzProfile.Path)
	paths = append(paths, OzProfile.Paths...)
	for _, pp := range paths {
		divertRemove(pp)
	}

	fmt.Printf("Successfully remove jail for: %s.\n", OzProfile.Name)
}

func handleStatus(c *cli.Context) {
	OzConfig = loadConfig()
	if len(c.Args()) == 0 {
		fmt.Fprintf(os.Stderr, "You must supply the name of a profile or an executable path.")
		os.Exit(1)
	}
	pname := c.Args()[0]
	OzProfile, err := loadProfile(pname, OzConfig.ProfileDir)
	if err != nil || OzProfile == nil {
		fmt.Fprintf(os.Stderr, "Unable to load profiles (%s): %v.\n", pname, err)
		os.Exit(1)
	}

	if OzConfig.DivertSuffix == "" && OzConfig.DivertPath == false {
		fmt.Fprintf(os.Stderr, "Divert requires a suffix to be set.\n")
		os.Exit(1)
	}

	checkInstalled := func(cpath string) {
		isInstalled, err := isDivertInstalled(cpath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unknown error: %+v\n", err)
			os.Exit(1)
		}
		sfmt := "%-37s\033[0m%s\n"
		if isInstalled {
			fmt.Printf("\033[0;32m"+sfmt, "Package divert is installed for: ", cpath)
		} else {
			fmt.Printf("\033[0;31m"+sfmt, "Package divert is not installed for: ", cpath)
		}
	}

	paths := append([]string{}, OzProfile.Path)
	paths = append(paths, OzProfile.Paths...)
	for _, pp := range paths {
		checkInstalled(pp)
	}
}

func handleCreate(c *cli.Context) {
	OzConfig = loadConfig()

	fmt.Println("The weasels ran off with this command... please come back later!")
	os.Exit(1)
}

/*
* UTILITIES
 */

func checkRoot() {
	if os.Getuid() != 0 {
		fmt.Fprintf(os.Stderr, "%s should be used as root.\n", os.Args[0])
		os.Exit(1)
	}
}

func checkDpkgDivert() string {
	ddpath, err := exec.LookPath("dpkg-divert")
	if err != nil {
		fmt.Fprintln(os.Stderr, "You do not appear to have dpkg-divert, are you not running Debian/Ubuntu?")
		os.Exit(1)
	}
	return ddpath
}

func checkDpkg() string {
	dpath, err := exec.LookPath("dpkg")
	if err != nil {
		fmt.Fprintln(os.Stderr, "You do not appear to have dpkg, are you not running Debian/Ubuntu?")
		os.Exit(1)
	}
	return dpath
}

func isDivertInstalled(bpath string) (bool, error) {
	outp, err := exec.Command(PathDpkgDivert, "--truename", bpath).Output()
	if err != nil {
		return false, err
	}
	dpath := strings.TrimSpace(string(outp))

	isInstalled := (dpath == getBinaryPath(string(bpath)))
	if isInstalled {
		_, err := os.Readlink(bpath)
		if err != nil {
			return false, fmt.Errorf("`%s` appears to be diverted but is not installed", dpath)
		}
	}
	return isInstalled, nil
}

func getBinaryPath(bpath string) string {
	bpath = strings.TrimSpace(string(bpath))

	if OzConfig.DivertSuffix != "" {
		if strings.HasSuffix(bpath, "."+OzConfig.DivertSuffix) == false {
			bpath += "." + OzConfig.DivertSuffix
		}
	}

	if OzConfig.DivertPath == true {
		bpath = path.Join(path.Dir(bpath)+"-oz", path.Base(bpath))
	}

	return bpath
}

func loadConfig() *oz.Config {
	config, err := oz.LoadConfig(oz.DefaultConfigPath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "Configuration file (%s) is missing, using defaults.\n", oz.DefaultConfigPath)
			config = oz.NewDefaultConfig()
		} else {
			fmt.Fprintf(os.Stderr, "Could not load configuration `%s`: %v", oz.DefaultConfigPath, err)
			os.Exit(1)
		}
	}

	return config
}

func loadProfile(name, profileDir string) (*oz.Profile, error) {
	ps, err := oz.LoadProfiles(profileDir)
	if err != nil {
		return nil, err
	}

	p, err := ps.GetProfileByName(name)
	if err != nil || p == nil {
		return ps.GetProfileByPath(name)
	}
	return p, nil

}

func installExit(hook bool, err error) {
	if hook {
		os.Exit(0)
	} else {
		fmt.Fprintf(os.Stderr, "%s", err)
		os.Exit(1)
	}
}
