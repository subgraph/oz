package daemon

import (
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/subgraph/oz/ipc"
)

func clientConnect() (*ipc.MsgConn, error) {
	return ipc.Connect(SocketName, messageFactory, nil)
}

func clientSend(msg interface{}) (*ipc.Message, error) {
	c, err := clientConnect()
	if err != nil {
		return nil, err
	}
	defer c.Close()
	rr, err := c.ExchangeMsg(msg)
	if err != nil {
		return nil, err
	}

	resp := <-rr.Chan()
	rr.Done()
	return resp, nil
}

func ListProfiles() ([]Profile, error) {
	resp, err := clientSend(new(ListProfilesMsg))
	if err != nil {
		return nil, err
	}
	body, ok := resp.Body.(*ListProfilesResp)
	if !ok {
		return nil, errors.New("ListProfiles response was not expected type")
	}
	return body.Profiles, nil
}

func ListForwarders(id int) ([]Forwarder, error) {
	resp, err := clientSend(&ListForwardersMsg{Id: id})
	if err != nil {
		return nil, err
	}
	body, ok := resp.Body.(*ListForwardersResp)
	if !ok {
		return nil, errors.New("ListForwarders response was not expected type")
	}
	return body.Forwarders, nil
}

func ListSandboxes() ([]SandboxInfo, error) {
	resp, err := clientSend(&ListSandboxesMsg{})
	if err != nil {
		return nil, err
	}
	body, ok := resp.Body.(*ListSandboxesResp)
	if !ok {
		return nil, errors.New("ListSandboxes response was not expected type")
	}
	return body.Sandboxes, nil
}

func ListBridges() ([]string, error) {
	resp, err := clientSend(&ListBridgesMsg{})
	if err != nil {
		return nil, err
	}
	body, ok := resp.Body.(*ListBridgesResp)
	if !ok {
		return nil, errors.New("ListBridges response was not expected type")
	}
	return body.Bridges, nil
}

func Launch(arg, cpath string, args []string, noexec bool) error {
	idx, name, err := parseProfileArg(arg)
	if err != nil {
		return err
	}
	pwd, _ := os.Getwd()
	groups, _ := os.Getgroups()
	gg := []uint32{}
	if len(groups) > 0 {
		gg = make([]uint32, len(groups))
		for i, v := range groups {
			gg[i] = uint32(v)
		}
	}
	resp, err := clientSend(&LaunchMsg{
		Index:  idx,
		Name:   name,
		Path:   cpath,
		Pwd:    pwd,
		Gids:   gg,
		Args:   args,
		Env:    os.Environ(),
		Noexec: noexec,
	})
	if err != nil {
		return err
	}
	switch body := resp.Body.(type) {
	case *ErrorMsg:
		fmt.Printf("error was %s\n", body.Msg)
	case *OkMsg:
		fmt.Println("ok received from application launch request")
	default:
		fmt.Printf("Unexpected message received %+v", body)
	}
	return nil
}

func KillAllSandboxes() error {
	return KillSandbox(-1)
}

func KillSandbox(id int) error {
	resp, err := clientSend(&KillSandboxMsg{Id: id})
	if err != nil {
		return err
	}
	switch body := resp.Body.(type) {
	case *ErrorMsg:
		return errors.New(body.Msg)
	case *OkMsg:
		return nil
	default:
		return fmt.Errorf("Unexpected message received %+v", body)
	}
}

func RelaunchXpraClient(id int) error {
	resp, err := clientSend(&RelaunchXpraClientMsg{Id: id})
	if err != nil {
		return err
	}
	switch body := resp.Body.(type) {
	case *ErrorMsg:
		return errors.New(body.Msg)
	case *OkMsg:
		return nil
	default:
		return fmt.Errorf("Unexpected message received %+v", body)
	}
}

func RelaunchAllXpraClient() error {
	return RelaunchXpraClient(-1)
}

func MountFiles(id int, files []string, readOnly bool) error {
	mountFilesMsg := MountFilesMsg{
		Id:       id,
		Files:    files,
		ReadOnly: readOnly,
	}
	resp, err := clientSend(&mountFilesMsg)
	if err != nil {
		return err
	}
	switch body := resp.Body.(type) {
	case *ErrorMsg:
		return errors.New(body.Msg)
	case *OkMsg:
		return nil
	default:
		return fmt.Errorf("Unexpected message received %+v", body)
	}
}

func UnmountFile(id int, file string) error {
	unmountFileMsg := UnmountFileMsg{
		Id:   id,
		File: file,
	}
	resp, err := clientSend(&unmountFileMsg)
	if err != nil {
		return err
	}
	switch body := resp.Body.(type) {
	case *ErrorMsg:
		return errors.New(body.Msg)
	case *OkMsg:
		return nil
	default:
		return fmt.Errorf("Unexpected message received %+v", body)
	}
}

func AskForwarder(id int, name, port string) (string, error) {
	askForwarderMsg := AskForwarderMsg{
		Id:   id,
		Name: name,
		Port: port,
	}
	resp, err := clientSend(&askForwarderMsg)
	if err != nil {
		return "", err
	}
	body, ok := resp.Body.(*ForwarderSuccessMsg)
	if !ok {
		return "", fmt.Errorf("Unexpected message received %+v", body)
	} else {
		return body.Addr, nil
	}
}

func parseProfileArg(arg string) (int, string, error) {
	if len(arg) == 0 {
		return 0, "", errors.New("profile argument needed")
	}
	if n, err := strconv.Atoi(arg); err == nil {
		return n, "", nil
	}
	return 0, arg, nil
}

func Logs(count int, follow bool) (chan string, error) {
	c, err := clientConnect()
	if err != nil {
		return nil, err
	}
	rr, err := c.ExchangeMsg(&LogsMsg{Count: count, Follow: follow})
	if err != nil {
		return nil, err
	}
	out := make(chan string)
	go dumpLogs(out, rr)
	return out, nil
}

func dumpLogs(out chan<- string, rr ipc.ResponseReader) {
	for resp := range rr.Chan() {
		switch body := resp.Body.(type) {
		case *OkMsg:
			rr.Done()
			close(out)
			return
		case *LogData:
			for _, ll := range body.Lines {
				out <- ll
			}
		default:
			out <- fmt.Sprintf("Unexpected response type (%T)", body)
		}
	}
}
