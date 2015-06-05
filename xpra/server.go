package xpra
import (
	"github.com/subgraph/oz"
	"fmt"
	"os"
	"os/exec"
)

var xpraServerDefaultArgs = []string{
	"--no-mdns",
	//"--pulseaudio",
	"--no-pulseaudio",
	"--input-method=SCIM",
}

func NewServer(config *oz.XServerConf, display uint64, workdir string) *Xpra {
	x := new(Xpra)
	x.Config = config
	x.Display = display
	x.WorkDir = workdir
	x.xpraArgs = prepareServerArgs(config, display, workdir)
	x.Process = exec.Command("/usr/bin/xpra", x.xpraArgs...)
	x.Process.Env = append(os.Environ(), "TMPDIR="+workdir)

	return x
}

func prepareServerArgs(config *oz.XServerConf, display uint64, workdir string) []string {
	args := getDefaultArgs(config)
	args = append(args, xpraServerDefaultArgs...)
	args = append(args,
		fmt.Sprintf("--socket-dir=%s", workdir),
		"start",
		fmt.Sprintf(":%d", display),
	)
	return args
}

