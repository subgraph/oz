package ozinit

import (
	"errors"
	"fmt"
	"github.com/subgraph/oz/ipc"
)

func clientConnect(addr string) (*ipc.MsgConn, error) {
	return ipc.Connect(addr, messageFactory, nil)
}

func clientSend(addr string, msg interface{}) (*ipc.Message, error) {
	c, err := clientConnect(addr)
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

func Ping(addr string) error {
	resp, err := clientSend(addr, new(PingMsg))
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

func RunProgram(addr, cpath, pwd string, args []string, noexec bool) error {
	c, err := clientConnect(addr)
	if err != nil {
		return err
	}
	rr, err := c.ExchangeMsg(&RunProgramMsg{Path: cpath, Args: args, Pwd: pwd, NoExec: noexec})
	resp := <-rr.Chan()
	rr.Done()
	c.Close()
	if err != nil {
		return err
	}
	switch body := resp.Body.(type) {
	case *ErrorMsg:
		return errors.New(body.Msg)
	case *OkMsg:
		return nil
	default:
		return fmt.Errorf("Unexpected message type received: %+v", body)
	}
}

func RunShell(addr, term string) (int, error) {
	c, err := clientConnect(addr)
	if err != nil {
		return 0, err
	}
	rr, err := c.ExchangeMsg(&RunShellMsg{Term: term})
	resp := <-rr.Chan()
	rr.Done()
	c.Close()
	if err != nil {
		return 0, err
	}
	switch body := resp.Body.(type) {
	case *ErrorMsg:
		return 0, errors.New(body.Msg)
	case *OkMsg:
		if len(resp.Fds) == 0 {
			return 0, errors.New("RunShell message returned Ok, but no file descriptor received")
		}
		return resp.Fds[0], nil
	default:
		return 0, fmt.Errorf("Unexpected message type received: %+v", body)
	}
}

func SetupForwarder(addr, proto, daddr string, fd uintptr) error {
	c, err := clientConnect(addr)
	if err != nil {
		return err
	}
	rr, err := c.ExchangeMsg(&ForwarderSuccessMsg{Addr: daddr, Proto: proto}, int(fd))
	if err != nil {
		return fmt.Errorf("Error %v: %+v", err, rr)
	}
	resp := <-rr.Chan()
	switch body := resp.Body.(type) {
	case *ErrorMsg:
		return errors.New(body.Msg)
	case *OkMsg:
		return nil
	default:
		return fmt.Errorf("Unexpected message type received: %+v", body)
	}

}
