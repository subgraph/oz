package main

import (
	"fmt"
	"os"
	"os/exec"
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
			Flags:  flagsHookMode,
		},
		{
			Name:   "remove",
			Usage:  "remove a binary diversion for a program",
			Action: handleRemove,
			Flags:  flagsHookMode,
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
	fmt.Println("Here be dragons!")
	os.Exit(1)
}

func handleConfigshow(c *cli.Context) {
	config, err := oz.LoadConfig(oz.DefaultConfigPath)
	useDefaults := false
	if err != nil {
		if os.IsNotExist(err) {
			config = oz.NewDefaultConfig()
			useDefaults = true
		} else {
			fmt.Fprintf(os.Stderr, "Could not load configuration: %s", oz.DefaultConfigPath, err)
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

	for i := 0; i < len(fmt.Sprintf(sfmt, "", "")); i++ {
		fmt.Print("=")
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

	if OzConfig.DivertSuffix == "" {
		installExit(c.Bool("hook"), fmt.Errorf("Divert requires a suffix to be set.\n"))
		return // For clarity
	}

	isInstalled, err := isDivertInstalled(OzProfile.Path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unknown error: %+v\n", err)
		os.Exit(1)
	}
	if isInstalled == true {
		fmt.Println("Divert already installed for ", OzProfile.Path)
		os.Exit(0)
	}

	dpkgArgs := []string{
		"--add",
		"--package",
		"oz",
		"--rename",
		"--divert",
		getBinaryPath(OzProfile.Path),
		OzProfile.Path,
	}

	_, err = exec.Command(PathDpkgDivert, dpkgArgs...).Output()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Dpkg divert command `%s %+s` failed: %s", PathDpkgDivert, dpkgArgs, err)
		os.Exit(1)
	}

	err = syscall.Symlink(OzConfig.ClientPath, OzProfile.Path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create symlink %s", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully installed Oz sandbox for: %s.\n", OzProfile.Path)
}

func handleRemove(c *cli.Context) {
	OzConfig = loadConfig()
	pname := c.Args()[0]
	OzProfile, err := loadProfile(pname, OzConfig.ProfileDir)
	if err != nil || OzProfile == nil {
		installExit(c.Bool("hook"), fmt.Errorf("Unable to load profiles for %s.\n", pname))
		return // For clarity
	}

	if OzConfig.DivertSuffix == "" {
		installExit(c.Bool("hook"), fmt.Errorf("Divert requires a suffix to be set.\n"))
		return // For clarity
	}

	isInstalled, err := isDivertInstalled(OzProfile.Path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unknown error: %+v\n", err)
		os.Exit(1)
	}
	if isInstalled == false {
		fmt.Println("Divert is not installed for ", OzProfile.Path)
		os.Exit(0)
	}

	os.Remove(OzProfile.Path)

	dpkgArgs := []string{
		"--rename",
		"--package",
		"oz",
		"--remove",
		OzProfile.Path,
	}

	_, err = exec.Command(PathDpkgDivert, dpkgArgs...).Output()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Dpkg divert command `%s %+s` failed: %s", PathDpkgDivert, dpkgArgs, err)
		os.Exit(1)
	}

	fmt.Printf("Successfully remove jail for: %s.\n", OzProfile.Path)
}

func handleStatus(c *cli.Context) {
	OzConfig = loadConfig()
	pname := c.Args()[0]
	OzProfile, err := loadProfile(pname, OzConfig.ProfileDir)
	if err != nil || OzProfile == nil {
		fmt.Fprintf(os.Stderr, "Unable to load profiles (%s).\n", err)
		os.Exit(1)
	}

	if OzConfig.DivertSuffix == "" {
		fmt.Fprintf(os.Stderr, "Divert requires a suffix to be set.\n")
		os.Exit(1)
	}

	isInstalled, err := isDivertInstalled(OzProfile.Path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unknown error: %+v\n", err)
		os.Exit(1)
	}
	if isInstalled {
		fmt.Println("Package divert is \033[0;32minstalled\033[0m for: ", OzProfile.Path)
	} else {
		fmt.Println("Package divert is \033[0;31mnot installed\033[0m for: ", OzProfile.Path)
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

	if strings.HasSuffix(bpath, "."+OzConfig.DivertSuffix) == false {
		bpath += "." + OzConfig.DivertSuffix
	}

	return bpath
}

func loadConfig() *oz.Config {
	config, err := oz.LoadConfig(oz.DefaultConfigPath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintln(os.Stderr, "Configuration file (%s) is missing, using defaults.", oz.DefaultConfigPath)
			config = oz.NewDefaultConfig()
		} else {
			fmt.Fprintln(os.Stderr, "Could not load configuration: %s", oz.DefaultConfigPath, err)
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

	return ps.GetProfileByName(name)

}

func installExit(hook bool, err error) {
	if hook {
		os.Exit(0)
	} else {
		fmt.Fprintf(os.Stderr, "%s", err)
		os.Exit(1)
	}
}
