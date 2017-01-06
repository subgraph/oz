package ozinit

import "github.com/subgraph/oz/ipc"

type OkMsg struct {
	_ string "Ok"
}

type ErrorMsg struct {
	Msg string "Error"
}

type PingMsg struct {
	Data string "Ping"
}

type RunShellMsg struct {
	Term string "RunShell"
}

type RunProgramMsg struct {
	Args []string "RunProgram"
	Pwd  string
	Path string
}

type ForwarderSuccessMsg struct {
	Port string "ForwarderSuccess"
	Proto string
	Addr string
}

var messageFactory = ipc.NewMsgFactory(
	new(OkMsg),
	new(ErrorMsg),
	new(PingMsg),
	new(RunShellMsg),
	new(RunProgramMsg),
	new(ForwarderSuccessMsg),
)
