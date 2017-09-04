package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/subgraph/oz"
	"github.com/subgraph/oz/oz-daemon"
	"github.com/subgraph/oz/oz-init"

	"github.com/codegangsta/cli"
)

type fnRunType func()

var runFunc fnRunType

func init() {
	switch path.Base(os.Args[0]) {
	case "oz":
		runFunc = runApplication
	default:
		runFunc = runSandboxed
	}
}

var OzConfig *oz.Config

func main() {
	var err error
	if err = checkRecursingSandbox(); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}

	oz.CheckSettingsOverRide()
	OzConfig, err = oz.LoadConfig(oz.DefaultConfigPath)

	runFunc()
}

func runSandboxed() {
	apath := os.Args[0]
	if !filepath.IsAbs(apath) {
		epath, err := exec.LookPath(apath)
		apath = epath
		if err != nil {
			fmt.Fprintf(os.Stderr, "Cannot find executable for `%s`: %v\n", apath, err)
			os.Exit(1)
		}
	}
	ephemeral := false
	if OzConfig.EnableEphemerals {
		running, err := daemon.IsRunning(apath, os.Args[1:])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error communicating with daemon: %v\n", err)
			os.Exit(1)
		}
		if running == false {
			profile, err := daemon.GetProfile(apath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error fetching profile: %v\n", err)
				os.Exit(1)
			}
			if ozinit.ProfileHasEphemerals(profile) == true {
				chanb := make(chan bool, 1)
				go promptEphemeralLaunch(chanb, profile.Name)
				ephemeral = <-chanb
			}
		}
	}
	if err := daemon.Launch("0", apath, os.Args[1:], false, ephemeral); err != nil {
		fmt.Fprintf(os.Stderr, "launch command failed: %v.\n", err)
		os.Exit(1)
	}
}

func runApplication() {
	app := cli.NewApp()

	app.Name = "oz"
	app.Usage = "command line interface to Oz sandboxes"
	app.Author = "Subgraph"
	app.Email = "info@subgraph.com"
	app.Version = oz.OzVersion
	app.EnableBashCompletion = true
	app.Commands = []cli.Command{
		{
			Name:   "profiles",
			Usage:  "list available application profiles",
			Action: handleProfiles,
		},
		{
			Name:   "launch",
			Usage:  "launch an application profile",
			Action: handleLaunch,
			Flags: []cli.Flag{
				cli.BoolFlag{
					Name: "noexec, n",
				},
				cli.BoolFlag{
					Name: "ephemeral, e",
				},
			},
		},
		{
			Name:   "list",
			Usage:  "list running sandboxes",
			Action: handleList,
			Flags: []cli.Flag{
				cli.BoolFlag{
					Name: "verbose, v",
				},
			},
		},
		{
			Name:   "shell",
			Usage:  "start a shell in a running sandbox",
			Action: handleShell,
		},
		{
			Name:   "mount",
			Usage:  "cause a sandbox to mount a file from the host",
			Action: handleMount,
		},
		{
			Name:   "umount",
			Usage:  "undo a previous oz mount",
			Action: handleUmount,
		},
		{
			Name:   "kill",
			Usage:  "terminate a running sandbox",
			Action: handleKill,
		},
		{
			Name:   "killall",
			Usage:  "terminate all running sandboxes",
			Action: handleKillall,
		},
		{
			Name:   "relaunchxpra",
			Usage:  "relaunch xpra client for a running sandbox (\"all\" for all sandboxes)",
			Action: handleRelaunchXpraClient,
		},
		{
			Name:   "logs",
			Usage:  "display oz-daemon logs",
			Action: handleLogs,
			Flags: []cli.Flag{
				cli.BoolFlag{
					Name: "f",
				},
			},
		},
		{
			Name:   "listbridges",
			Usage:  "list configured bridges",
			Action: handleListBridges,
		},
		{
			Name:   "forward",
			Usage:  "setup forwarder",
			Action: handleForward,
			Flags: []cli.Flag{
				cli.IntFlag{
					Name:  "sandbox",
					Usage: "Sandbox number, e.g. 1",
					Value: -1,
				},
				cli.StringFlag{
					Name:  "name",
					Usage: "Name of forwarder, e.g. dynamic-onionshare-server",
				},
				cli.StringFlag{
					Name:  "port",
					Usage: "Target port, e.g. tcp",
				},
			},
		},
		{
			Name:   "listforwarders",
			Usage:  "list forwarders",
			Action: handleListForwarders,
			Flags: []cli.Flag{
				cli.IntFlag{
					Name:  "sandbox",
					Usage: "Sandbox number, e.g. 1",
					Value: -1,
				},
			},
		},
		{
			Name:   "listproxies",
			Usage:  "list established proxy circuits",
			Action: handleListProxies,
		},
	}
	app.Run(os.Args)
}

func handleProfiles(c *cli.Context) {
	ps, err := daemon.ListProfiles()
	if err != nil {
		fmt.Printf("Error listing profiles: %v\n", err)
		os.Exit(1)
	}
	for i, p := range ps {
		fmt.Printf("%2d) %-30s %s\n", i+1, p.Name, p.Path)
	}
}

func handleLaunch(c *cli.Context) {
	noexec := c.Bool("noexec")
	ephemeral := c.Bool("ephemeral")
	if !OzConfig.EnableEphemerals {
		ephemeral = false
	}
	if len(c.Args()) == 0 {
		fmt.Println("Argument needed to launch command")
		os.Exit(1)
	}
	err := daemon.Launch(c.Args()[0], "", c.Args()[1:], noexec, ephemeral)
	if err != nil {
		fmt.Printf("launch command failed: %v\n", err)
		os.Exit(1)
	}
}

func handleList(c *cli.Context) {
	verbose := c.Bool("verbose")
	sboxes, err := daemon.ListSandboxes()
	if err != nil {
		fmt.Printf("Error listing running sandboxes: %v\n", err)
		os.Exit(1)
	}
	if len(sboxes) == 0 {
		fmt.Println("No running sandboxes")
		return
	}
	for _, sb := range sboxes {
		ephemeral := ""
		if sb.Ephemeral {
			ephemeral = " [ephemeral]"
		}
		fmt.Printf("%2d) %s%s\n", sb.Id, sb.Profile, ephemeral)
	}
}

func handleListBridges(c *cli.Context) {
	bridges, err := daemon.ListBridges()
	if err != nil {
		fmt.Printf("Error listing configured bridges: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(strings.Join(bridges, ","))
}

func handleMount(c *cli.Context) {
	if len(c.Args()) < 2 {
		fmt.Println("oz mount <sandbox_id> <paths...>")
		os.Exit(1)
	}
	id, err := strconv.Atoi(c.Args()[0])
	if err != nil {
		fmt.Println("Sandbox id argument must be an integer")
		os.Exit(1)
	}
	start := 1
	readOnly := false
	if c.Args()[1] == "--readonly" {
		readOnly = true
		start = 2
	}

	err = daemon.MountFiles(id, c.Args()[start:], readOnly)
	if err != nil {
		fmt.Println("MountFiles FAIL", err)
	}
}

func handleUmount(c *cli.Context) {
	if len(c.Args()) < 2 {
		fmt.Println("oz unmount <sandbox_id> <path>")
		os.Exit(1)
	}
	id, err := strconv.Atoi(c.Args()[0])
	if err != nil {
		fmt.Println("Sandbox id argument must be an integer")
		os.Exit(1)
	}

	err = daemon.UnmountFile(id, c.Args()[1])
	if err != nil {
		fmt.Println("UnmountFile FAIL", err)
	}
}

func handleShell(c *cli.Context) {
	if len(c.Args()) == 0 {
		fmt.Println("Sandbox id argument needed")
		os.Exit(1)
	}
	id, err := strconv.Atoi(c.Args()[0])
	if err != nil {
		fmt.Println("Sandbox id argument must be an integer")
		os.Exit(1)
	}

	sb, err := getSandboxById(id)
	if err != nil {
		fmt.Printf("Error retrieving sandbox list: %v\n", err)
		os.Exit(1)
	}
	if sb == nil {
		fmt.Printf("No sandbox found with id = %d\n", id)
		os.Exit(1)
	}

	chanb := make(chan bool, 1)
	go promptConfirmShell(chanb, sb.Profile, id)
	prompt := <-chanb
	if !prompt {
		fmt.Printf("Denied shell execution... \n")
		os.Exit(0)
	}

	term := os.Getenv("TERM")
	fd, err := ozinit.RunShell(sb.Address, term)
	if err != nil {
		fmt.Printf("start shell command failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Entering interactive shell in `%s`\n\n", sb.Profile)
	st, err := SetRawTerminal(0)
	HandleResize(fd)
	f := os.NewFile(uintptr(fd), "")
	go io.Copy(f, os.Stdin)
	io.Copy(os.Stdout, f)
	if err := RestoreTerminal(0, st); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	fmt.Println("done..")
}

func getSandboxById(id int) (*daemon.SandboxInfo, error) {
	sboxes, err := daemon.ListSandboxes()
	if err != nil {
		return nil, err
	}
	for _, sb := range sboxes {
		if id == sb.Id {
			return &sb, nil
		}
	}
	return nil, nil
}

func handleKillall(c *cli.Context) {
	if err := daemon.KillAllSandboxes(); err != nil {
		fmt.Fprintf(os.Stderr, "Killall command failed: %s.\n", err)
		os.Exit(1)
	}
}

func handleKill(c *cli.Context) {
	if len(c.Args()) == 0 {
		fmt.Fprintf(os.Stderr, "Need a sandbox id to kill\n")
		os.Exit(1)
	}
	if c.Args()[0] == "all" {
		if err := daemon.KillAllSandboxes(); err != nil {
			fmt.Fprintf(os.Stderr, "Killall command failed: %s.\n", err)
			os.Exit(1)
		}
		return
	}
	id, err := strconv.Atoi(c.Args()[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not parse id value %s\n", c.Args()[0])
		os.Exit(1)
	}
	if err := daemon.KillSandbox(id); err != nil {
		fmt.Fprintf(os.Stderr, "Kill command failed: %s.\n", err)
		os.Exit(1)
	}

}
func handleLogs(c *cli.Context) {
	follow := c.Bool("f")
	ch, err := daemon.Logs(0, follow)
	if err != nil {
		fmt.Println("Logs failed", err)
		os.Exit(1)
	}
	for ll := range ch {
		fmt.Println(ll)
	}
}

func handleRelaunchXpraClient(c *cli.Context) {
	if len(c.Args()) == 0 {
		fmt.Fprintf(os.Stderr, "Need a sandbox id to relaunch\n")
		os.Exit(1)
	}
	if c.Args()[0] == "all" {
		if err := daemon.RelaunchAllXpraClient(); err != nil {
			fmt.Fprintf(os.Stderr, "Killall command failed: %s.\n", err)
			os.Exit(1)
		}
		return
	} else {
		id, err := strconv.Atoi(c.Args()[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Could not parse id value %s\n", c.Args()[0])
			os.Exit(1)
		}
		if err := daemon.RelaunchXpraClient(id); err != nil {
			fmt.Fprintf(os.Stderr, "Relaunch command failed: %s.\n", err)
			os.Exit(1)
		}
	}
}

func handleForward(c *cli.Context) {
	var out string
	var err error
	id := c.Int("sandbox")
	if id == -1 {
		fmt.Fprintf(os.Stderr, "Need a sandbox id to create a forwarder\n")
		os.Exit(1)
	}
	name, port := c.String("name"), c.String("port")
	if name == "" || port == "" {
		fmt.Fprintf(os.Stderr, "Missing required arguments.\n")
		os.Exit(1)
	}
	if out, err = daemon.AskForwarder(id, c.String("name"), c.String("port")); err != nil {
		fmt.Fprintf(os.Stderr, "Fowarder command failed: %s.\n", err)
		os.Exit(1)
	}
	fmt.Println("Listener established: " + out)
}

func handleListForwarders(c *cli.Context) {
	id := c.Int("sandbox")
	if id == -1 {
		fmt.Fprintf(os.Stderr, "Need a sandbox id to list forwarders\n")
		os.Exit(1)
	}
	forwarders, err := daemon.ListForwarders(id)
	if err != nil {
		fmt.Fprintf(os.Stderr, "List forwarders failed: %+v %+v", err, forwarders)
		os.Exit(1)
	}

	fmt.Printf("Listeners for sandbox %d:\n", id)
	for _, r := range forwarders {
		fmt.Printf("  %s: %s => %s\n", r.Name, r.Desc, r.Target)
	}
}

func handleListProxies(c *cli.Context) {
	res, err := daemon.ListProxies()
	if err != nil {
		fmt.Printf("Error listing established proxies: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Result: %d entries ...\n", len(res))
	fmt.Println(strings.Join(res, "\n"))
}


func checkRecursingSandbox() error {
	hostname, _ := os.Hostname()
	fsbox := path.Join("/tmp", "oz-sandbox")
	bsbox, err := ioutil.ReadFile(fsbox)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("Unknown error checking for sandbox file: %v")
	}
	ssbox := string(bsbox)
	if ssbox != "" {
		if path.Base(os.Args[0]) == "oz" {
			return fmt.Errorf("Cannot run oz client inside of existing sandbox!")
		}
		if path.Base(os.Args[0]) == hostname {
			// TODO: We should just exec cmd+suffix here
			return fmt.Errorf("Cannot recursively launch sandbox `%s`!", hostname)
		}
		// TODO: Attempting to launch sandboxed application in another sandbox.
		//       Send back to daemon for launching from host.
		return fmt.Errorf("Cannot run a sandbox from inside a running sandbox!")
	}

	return nil
}
