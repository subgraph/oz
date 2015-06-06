package daemon

import "github.com/subgraph/oz/ipc"

const SocketName = "@oz-control"

type OkMsg struct {
	_ string "Ok"
}

type ErrorMsg struct {
	Msg string "Error"
}

type PingMsg struct {
	Data string "Ping"
}

type ListProfilesMsg struct {
	_ string "ListProfiles"
}

type Profile struct {
	Index int
	Name  string
	Path  string
}

type ListProfilesResp struct {
	Profiles []Profile "ListProfilesResp"
}

type LaunchMsg struct {
	Index int "Launch"
	Name  string
}

type InitNetworkMsg struct {
	Index int "NetworkBridge"
	Name  string
}

type ListSandboxesMsg struct {
	_ string "ListSandboxes"
}

type SandboxInfo struct {
	Id      int
	Address string
	Profile string
}

type ListSandboxesResp struct {
	Sandboxes []SandboxInfo "ListSandboxesResp"
}

type CleanMsg struct {
	Index int "Clean"
	Name  string
}

type LogsMsg struct {
	Count  int "Logs"
	Follow bool
}

type LogData struct {
	Lines []string "LogData"
}

var messageFactory = ipc.NewMsgFactory(
	new(PingMsg),
	new(OkMsg),
	new(ErrorMsg),
	new(ListProfilesMsg),
	new(ListProfilesResp),
	new(LaunchMsg),
	new(ListSandboxesMsg),
	new(ListSandboxesResp),
	new(CleanMsg),
	new(LogsMsg),
	new(LogData),
)
