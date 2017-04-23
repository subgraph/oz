package daemon

import "github.com/subgraph/oz/ipc"

const SocketName = "@oz-control"

type OkMsg struct {
	_ string "Ok"
}

type NotOkMsg struct {
	_ string "NotOk"
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

type ListForwardersResp struct {
	Name       string      "ListForwardersResp"
	Forwarders []Forwarder "ListForwardersResp"
}

type ListBridgesMsg struct {
	_ string "ListBridges"
}

type ListBridgesResp struct {
	Bridges []string "ListBridgesResp"
}

type IsRunningMsg struct {
	Path string "IsRunning"
	Gids []uint32
	Args []string
	Env  []string
}

type GetProfileMsg struct {
	Path string "GetProfile"
	Gids []uint32
	Env  []string
}

type GetProfileResp struct {
	Profile string "Profile"
}

type LaunchMsg struct {
	Index     int "Launch"
	Path      string
	Name      string
	Pwd       string
	Gids      []uint32
	Args      []string
	Env       []string
	NoExec    bool
	Ephemeral bool
}

type ListSandboxesMsg struct {
	_ string "ListSandboxes"
}

type SandboxInfo struct {
	Id        int
	Address   string
	Profile   string
	Mounts    []string
	Ephemeral bool
}

type ListSandboxesResp struct {
	Sandboxes []SandboxInfo "ListSandboxesResp"
}

type KillSandboxMsg struct {
	Id int "KillSandbox"
}

type RelaunchXpraClientMsg struct {
	Id int "RelaunchXpraClient"
}

type MountFilesMsg struct {
	Id       int "MountFiles"
	Files    []string
	ReadOnly bool
}

type UnmountFileMsg struct {
	Id   int "UnmountFile"
	File string
}

type LogsMsg struct {
	Count  int "Logs"
	Follow bool
}

type LogData struct {
	Lines []string "LogData"
}

type ListForwardersMsg struct {
	Id int "ListForwarders"
}

type AskForwarderMsg struct {
	Id   int "AskForwarder"
	Name string
	Addr string
	Port string
}

type Forwarder struct {
	Name   string "Forwarder"
	Desc   string
	Target string
}

type ForwarderSuccessMsg struct {
	Proto string "ForwarderSuccess"
	Addr  string
	Port  string
}

var messageFactory = ipc.NewMsgFactory(
	new(PingMsg),
	new(OkMsg),
	new(NotOkMsg),
	new(ErrorMsg),
	new(GetConfigMsg),
	new(ListProfilesMsg),
	new(ListProfilesResp),
	new(LaunchMsg),
	new(IsRunningMsg),
	new(GetProfileMsg),
	new(GetProfileResp),
	new(ListSandboxesMsg),
	new(ListSandboxesResp),
	new(KillSandboxMsg),
	new(RelaunchXpraClientMsg),
	new(MountFilesMsg),
	new(UnmountFileMsg),
	new(LogsMsg),
	new(LogData),
	new(AskForwarderMsg),
	new(ForwarderSuccessMsg),
	new(ListForwardersMsg),
	new(ListForwardersResp),
	new(ListBridgesMsg),
	new(ListBridgesResp),
)
