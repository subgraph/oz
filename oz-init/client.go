package ozinit
import (
	"github.com/subgraph/oz/ipc"
	"errors"
	"fmt"
)

func clientConnect(addr string) (*ipc.MsgConn, error) {
	c := ipc.NewMsgConn(messageFactory, addr)
	if err := c.Connect(); err != nil {
		return nil, err
	}
	return c, nil
}

func clientSend(addr string, msg interface{}) (*ipc.Message, error) {
	c,err := clientConnect(addr)
	if err != nil {
		return nil, err
	}
	rr, err := c.ExchangeMsg(msg)
	resp := <- rr.Chan()
	rr.Done()

	c.Close()
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func Ping(addr string) error {
	resp,err := clientSend(addr, new(PingMsg))
	if err != nil {
		return err
	}
	switch body := resp.Body.(type) {
	case *PingMsg:
		return nil
	case *ErrorMsg:
		return errors.New(body.Msg)
	default:
		return fmt.Errorf("Unexpected message received: %+v", body)
	}
}

func RunShell(addr, term string) (int, error) {
	c,err := clientConnect(addr)
	if err != nil {
		return 0, err
	}
	rr,err := c.ExchangeMsg(&RunShellMsg{Term: term})
	resp := <- rr.Chan()
	rr.Done()
	c.Close()
	if err != nil {
		return 0, err
	}
	switch body := resp.Body.(type) {
	case *ErrorMsg:
		return 0,errors.New(body.Msg)
	case *OkMsg:
		if len(resp.Fds) == 0 {
			return 0, errors.New("RunShell message returned Ok, but no file descriptor received")
		}
		return resp.Fds[0], nil
	default:
		return 0, fmt.Errorf("Unexpected message type received: %+v", body)
	}
}


