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

type GetConfigMsg struct {
	Data string "GetConfig"
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
	Index  int "Launch"
	Path   string
	Name   string
	Pwd    string
	Args   []string
	Env    []string
	Noexec bool
}

type ListSandboxesMsg struct {
	_ string "ListSandboxes"
}

type SandboxInfo struct {
	Id      int
	Address string
	Profile string
	Mounts  []string
}

type ListSandboxesResp struct {
	Sandboxes []SandboxInfo "ListSandboxesResp"
}

type KillSandboxMsg struct {
	Id int "KillSandbox"
}

type MountFilesMsg struct {
	Id int "MountFiles"
	Files []string
	ReadOnly bool
}

type UnmountFileMsg struct {
	Id int "UnmountFile"
	File string
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
	new(GetConfigMsg),
	new(ListProfilesMsg),
	new(ListProfilesResp),
	new(LaunchMsg),
	new(ListSandboxesMsg),
	new(ListSandboxesResp),
	new(KillSandboxMsg),
	new(MountFilesMsg),
	new(UnmountFileMsg),
	new(LogsMsg),
	new(LogData),
)
