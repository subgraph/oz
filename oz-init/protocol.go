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

var messageFactory = ipc.NewMsgFactory(
	new(OkMsg),
	new(ErrorMsg),
	new(PingMsg),
	new(RunShellMsg),
)
