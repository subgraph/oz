package daemon
import (
	"github.com/subgraph/oz/ipc"
	"errors"
	"strconv"
	"fmt"
)

func clientConnect() (*ipc.MsgConn, error) {
	return ipc.Connect(SocketName, messageFactory, nil)
}

func clientSend(msg interface{}) (*ipc.Message, error) {
	c,err := clientConnect()
	if err != nil {
		return nil, err
	}
	defer c.Close()
	rr, err := c.ExchangeMsg(msg)
	if err != nil {
		return nil, err
	}

	resp := <- rr.Chan()
	rr.Done()
	return resp,nil
}

func ListProfiles() ([]Profile, error) {
	resp,err := clientSend(new(ListProfilesMsg))
	if err != nil {
		return nil, err
	}
	body,ok := resp.Body.(*ListProfilesResp)
	if !ok {
		return nil, errors.New("ListProfiles response was not expected type")
	}
	return body.Profiles, nil
}

func ListSandboxes() ([]SandboxInfo, error) {
	resp,err := clientSend(&ListSandboxesMsg{})
	if err != nil {
		return nil, err
	}
	body,ok := resp.Body.(*ListSandboxesResp)
	if !ok {
		return nil, errors.New("ListSandboxes response was not expected type")
	}
	return body.Sandboxes, nil
}

func Launch(arg string) error {
	idx,name,err := parseProfileArg(arg)
	if err != nil {
		return err
	}
	resp,err := clientSend(&LaunchMsg{
		Index: idx,
		Name: name,
	})
	if err != nil {
		return err
	}
	switch body := resp.Body.(type) {
	case *ErrorMsg:
		fmt.Printf("error was %s\n", body.Msg)
	case *OkMsg:
		fmt.Println("ok received")
	default:
		fmt.Printf("Unexpected message received %+v", body)
	}
	return nil
}

func Clean(arg string) error {
	idx,name,err := parseProfileArg(arg)
	if err != nil {
		return err
	}
	resp,err := clientSend(&CleanMsg{
		Index: idx,
		Name: name,
	})
	if err != nil {
		return err
	}
	// TODO collapse this logic into a function like clientSend
	switch body := resp.Body.(type) {
	case *ErrorMsg:
		return errors.New(body.Msg)
	case *OkMsg:
		return nil
	default:
		return fmt.Errorf("Unexpected message received %+v", body)
	}
}

func parseProfileArg(arg string) (int, string, error) {
	if len(arg) == 0 {
		return 0, "", errors.New("profile argument needed")
	}
	if n,err := strconv.Atoi(arg); err == nil {
		return n, "", nil
	}
	return 0, arg, nil
}

func Logs(count int, follow bool) (chan string, error) {
	c,err := clientConnect()
	if err != nil {
		return nil, err
	}
	rr,err := c.ExchangeMsg(&LogsMsg{Count: count, Follow: follow})
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
