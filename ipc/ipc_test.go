package ipc
import (
	"testing"
	"sync"
	"os"
)

type TestMsg struct {
	t int "Test"
}

type testConnection struct {
	server *MsgConn
	client *MsgConn
	wg sync.WaitGroup
	called bool
}

type testServer struct {
	conn *MsgConn
	wg sync.WaitGroup
}
const testSocket = "@test"
var testFactory = NewMsgFactory(new(TestMsg))

func testConnect(handler func(*TestMsg, *Message) error) (*testConnection, error) {
	s := NewMsgConn(testFactory, testSocket)
	c := NewMsgConn(testFactory, testSocket)
	tc := &testConnection{
		server: s,
		client: c,
	}
	wrapper := func(tm *TestMsg, msg *Message) error {
		err := handler(tm, msg)
		tc.called = true
		tc.wg.Done()
		return err
	}
	if err := s.AddHandlers(wrapper); err != nil {
		return nil, err
	}
	if err := s.Listen(); err != nil {
		return nil, err
	}
	if err := c.Connect(); err != nil {
		return nil, err
	}
	tc.wg.Add(1)
	go tc.server.Run()
	return tc, nil
}

func runTest(t *testing.T, handler func(*TestMsg, *Message) error, tester func(*testConnection)) {
	tc,err := testConnect(handler)
	if err != nil {
		t.Error("error setting up test connection:", err)
	}
	tester(tc)
	tc.wait()
	if !tc.called {
		t.Error("handler function not called")
	}
}

func (tc *testConnection) wait() {
	tc.wg.Wait()
	tc.client.Close()
	tc.server.Close()
}

func TestUcred(t *testing.T) {
	handler := func(tm *TestMsg, msg *Message) error {
		uid := uint32(os.Getuid())
		gid := uint32(os.Getgid())
		pid := int32(os.Getpid())
		u := msg.Ucred
		if u.Uid != uid || u.Gid != gid || u.Pid != pid {
			t.Errorf("ucred (%d/%d/%d) does not match process (%d/%d/%d)", u.Uid, u.Gid, u.Pid, uid, gid, pid)
		}
		return nil
	}
	runTest(t, handler, func(tc *testConnection) {
		tc.client.SendMsg(&TestMsg{})
	})

}

func TestPassFDs(t *testing.T) {
	fds := []int{1,2}
	handler := func(tm *TestMsg, msg *Message) error {
		if len(msg.Fds) != len(fds) {
			t.Errorf("Expecting %d descriptors, got %d", len(fds), len(msg.Fds))
		}
		return nil
	}
	runTest(t, handler, func(tc *testConnection) {
		tc.client.SendMsg(&TestMsg{}, fds...)

	})
}