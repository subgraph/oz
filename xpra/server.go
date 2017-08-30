package xpra

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/subgraph/oz"
)

var xpraServerDefaultArgs = []string{
	"--no-mdns",
	//"--pulseaudio",
	"--input-method=keep",
}

func NewServer(config *oz.XServerConf, display uint64, spath, workdir string) *Xpra {
	x := new(Xpra)
	x.Config = config
	x.Display = display
	x.WorkDir = workdir
	x.xpraArgs = prepareServerArgs(config, display, workdir)

	x.xpraArgs = append([]string{"-mode=blacklist", "/usr/bin/xpra"}, x.xpraArgs...)
	x.Process = exec.Command(spath, x.xpraArgs...)
	x.Process.Env = append(os.Environ(),
		"TMPDIR="+workdir,
		"XPRA_CLIPBOARD_LIMIT=45",
		"XPRA_CLIPBOARDS=CLIPBOARD",
	)

	if err := writeFakeProfile(x.Process); err != nil {
		return nil
	}

	return x
}

func prepareServerArgs(config *oz.XServerConf, display uint64, workdir string) []string {
	args := getDefaultArgs(config)
	//args = append(args, "--start-child \"/bin/echo _OZ_XXSTARTEDXX\"")
	args = append(args, xpraServerDefaultArgs...)
	//if config.AudioMode == oz.PROFILE_AUDIO_FULL || config.AudioMode == oz.PROFILE_AUDIO_SPEAKER {
	//	args = append(args, "--pulseaudio")
	//} else {
	//	args = append(args, "--no-pulseaudio")
	//}
	args = append(args,
		fmt.Sprintf("--bind=%s", workdir),
		fmt.Sprintf("--socket-dir=%s", workdir),
		"start",
		fmt.Sprintf(":%d", display),
	)
	return args
}
