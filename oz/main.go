package main

import (
	"fmt"
	"io"
	"os"
	"path"
	"strconv"

	"github.com/subgraph/oz"
	"github.com/subgraph/oz/oz-daemon"
	"github.com/subgraph/oz/oz-init"

	"github.com/codegangsta/cli"
)

type fnRunType func()

var runFunc fnRunType
var runBasename string

func init() {
	runBasename = path.Base(os.Args[0])

	switch runBasename {
	case "oz":
		runFunc = runApplication
	default:
		runFunc = runSandbox
	}
}

func main() {
	runFunc()
}

func runSandbox() {
	hostname, _ := os.Hostname()
	if runBasename == hostname {
		fmt.Fprintf(os.Stderr, "Cannot launch from inside a sandbox.\n")
		os.Exit(1)
	}

	err := daemon.Launch(runBasename, os.Args[1:], os.Environ(), false)
	if err != nil {
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
			},
		},
		{
			Name:   "list",
			Usage:  "list running sandboxes",
			Action: handleList,
		},
		{
			Name:   "shell",
			Usage:  "start a shell in a running sandbox",
			Action: handleShell,
		},
		{
			Name:   "clean",
			Action: handleClean,
		},
		{
			Name:   "kill",
			Action: handleKill,
		},
		{
			Name:   "logs",
			Action: handleLogs,
			Flags: []cli.Flag{
				cli.BoolFlag{
					Name: "f",
				},
			},
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
	if len(c.Args()) == 0 {
		fmt.Println("Argument needed to launch command")
		os.Exit(1)
	}
	err := daemon.Launch(c.Args()[0], c.Args()[1:], os.Environ(), noexec)
	if err != nil {
		fmt.Printf("launch command failed: %v\n", err)
		os.Exit(1)
	}
}

func handleList(c *cli.Context) {
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
		fmt.Printf("%2d) %s\n", sb.Id, sb.Profile)

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
	term := os.Getenv("TERM")
	fd, err := ozinit.RunShell(sb.Address, term)
	if err != nil {
		fmt.Printf("start shell command failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Entering interactive shell?\n")
	st, err := SetRawTerminal(0)
	HandleResize(fd)
	f := os.NewFile(uintptr(fd), "")
	go io.Copy(f, os.Stdin)
	io.Copy(os.Stdout, f)
	RestoreTerminal(0, st)
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

func handleClean(c *cli.Context) {
	if len(c.Args()) == 0 {
		fmt.Println("Need a profile to clean")
		os.Exit(1)
	}
	err := daemon.Clean(c.Args()[0])
	if err != nil {
		fmt.Println("Clean failed:", err)
	}
}

func handleKill(c *cli.Context) {
	if len(c.Args()) == 0 {
		fmt.Println("Need a sandbox id to kill\n")
		os.Exit(1)
	}
	id, err := strconv.Atoi(c.Args()[0])
	if err != nil {
		fmt.Printf("Could not parse id value %s\n", c.Args()[0])
		os.Exit(1)
	}
	if err := daemon.KillSandbox(id); err != nil {
		fmt.Println("Kill command failed:", err)
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
