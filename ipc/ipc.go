package ipc

import (
	"encoding/json"
	"errors"
	"net"
	"syscall"

	"encoding/binary"
	"fmt"
	"github.com/op/go-logging"
	"io"
	"reflect"
)

const maxFdCount = 3
const maxMessageSz = 128 * 1024
const bufferSz = 1024

type MsgConn struct {
	log      *logging.Logger
	conn     *net.UnixConn
	buf      []byte
	oob      []byte
	disp     *msgDispatcher
	factory  MsgFactory
	isClosed bool
	idGen    <-chan int
	respMan  *responseManager
	onClose  func()
}

type MsgServer struct {
	isClosed bool
	log      *logging.Logger
	disp     *msgDispatcher
	factory  MsgFactory
	listener *net.UnixListener
	done     chan bool
	idGen    <-chan int
}

func NewServer(address string, factory MsgFactory, log *logging.Logger, handlers ...interface{}) (*MsgServer, error) {
	md, err := createDispatcher(log, handlers...)
	if err != nil {
		return nil, err
	}

	listener, err := net.ListenUnix("unix", &net.UnixAddr{address, "unix"})
	if err != nil {
		md.close()
		return nil, err
	}
	if err := setPassCred(listener); err != nil {
		return nil, errors.New("Failed to set SO_PASSCRED on listening socket: " + err.Error())
	}
	done := make(chan bool)
	idGen := newIdGen(done)
	return &MsgServer{
		log:      log,
		disp:     md,
		factory:  factory,
		listener: listener,
		done:     done,
		idGen:    idGen,
	}, nil
}

func (s *MsgServer) Run() error {
	for !s.isClosed {
		conn, err := s.listener.AcceptUnix()
		if err != nil {
			if s.isClosed {
				return nil
			}
			return err
		}
		if err := setPassCred(conn); err != nil {
			return errors.New("Failed to set SO_PASSCRED on accepted socket connection:" + err.Error())
		}
		mc := &MsgConn{
			log:     s.log,
			conn:    conn,
			disp:    s.disp,
			buf:     make([]byte, bufferSz),
			oob:     createOobBuffer(),
			factory: s.factory,
			idGen:   s.idGen,
			respMan: newResponseManager(),
		}
		go mc.readLoop()
	}
	return nil
}

func (s *MsgServer) Close() error {
	if s.isClosed {
		return nil
	}
	s.isClosed = true
	s.disp.close()
	close(s.done)
	return s.listener.Close()
}

func Connect(address string, factory MsgFactory, log *logging.Logger, handlers ...interface{}) (*MsgConn, error) {
	md, err := createDispatcher(log, handlers...)
	if err != nil {
		return nil, err
	}
	conn, err := net.DialUnix("unix", nil, &net.UnixAddr{address, "unix"})
	if err != nil {
		return nil, err
	}
	done := make(chan bool)
	idGen := newIdGen(done)
	mc := &MsgConn{
		log:     log,
		conn:    conn,
		disp:    md,
		oob:     createOobBuffer(),
		factory: factory,
		idGen:   idGen,
		respMan: newResponseManager(),
		onClose: func() {
			md.close()
			close(done)
		},
	}
	go mc.readLoop()
	return mc, nil
}

func newIdGen(done <-chan bool) <-chan int {
	ch := make(chan int)
	go idGenLoop(done, ch)
	return ch
}

func idGenLoop(done <-chan bool, out chan<- int) {
	current := int(1)
	for {
		select {
		case out <- current:
			current += 1
		case <-done:
			return
		}
	}
}

func (mc *MsgConn) readLoop() {
	for {
		if mc.processOneMessage() {
			return
		}
	}
}

func (mc *MsgConn) logger() *logging.Logger {
	if mc.log != nil {
		return mc.log
	}
	return defaultLog
}

func (mc *MsgConn) processOneMessage() bool {
	m, err := mc.readMessage()
	if err != nil {
		if err == io.EOF {
			mc.Close()
			return true
		}
		// Fix for bug in 1.5.1 wrapping EOF in OpError
		switch t := err.(type) {
		case *net.OpError:
			if t.Err == io.EOF {
				mc.Close()
				return true
			}
		}
		if !mc.isClosed {
			mc.logger().Warning("error on MsgConn.readMessage(): %v, %s", err)
		}
		return true
	}
	if !mc.respMan.handle(m) {
		mc.disp.dispatch(m)
	}
	return false
}

func (mc *MsgConn) Close() error {
	mc.isClosed = true
	if mc.onClose != nil {
		mc.onClose()
	}
	return mc.conn.Close()
}

func createOobBuffer() []byte {
	oobSize := syscall.CmsgSpace(syscall.SizeofUcred) + syscall.CmsgSpace(4*maxFdCount)
	return make([]byte, oobSize)
}

func (mc *MsgConn) readMessage() (*Message, error) {
	var szbuf [4]byte
	n, oobn, _, _, err := mc.conn.ReadMsgUnix(szbuf[:], mc.oob)
	if err != nil {
		return nil, err
	}
	sz := binary.BigEndian.Uint32(szbuf[:])
	if sz > maxMessageSz {
		return nil, fmt.Errorf("message size of (%d) exceeds maximum message size (%d)", sz, maxMessageSz)
	}
	if sz > uint32(len(mc.buf)) {
		mc.buf = make([]byte, sz)
	}
	n, _, _, _, err = mc.conn.ReadMsgUnix(mc.buf[:sz], nil)
	if err != nil {
		return nil, err
	}
	m, err := mc.parseMessage(mc.buf[:n])
	if err != nil {
		return nil, err
	}
	m.mconn = mc

	if oobn > 0 {
		err := m.parseControlData(mc.oob[:oobn])
		if err != nil {
		}
	}
	return m, nil
}

// AddHandlers registers a list of message handling functions with a MsgConn instance.
// Each handler function must have two arguments and return a single error value.  The
// first argument must be pointer to a message structure type.  A message structure type
// is a structure that must have a struct tag on the first field:
//
//    type FooMsg struct {
//        Stuff string  "Foo"   // <------ struct tag
//        // etc...
//    }
//
//    type SimpleMsg struct {
//        dummy int "Simple"   // struct has no fields, so add an unexported dummy field just for the tag
//    }
//
// The second argument to a handler function must have type *ipc.Message.  After a handler function
// has been registered, received messages matching the first argument will be dispatched to the corresponding
// handler function.
//
//     func fooHandler(foo *FooMsg, msg *ipc.Message) error { /* ... */ }
//     func simpleHandler(simple *SimpleMsg, msg *ipc.Message) error { /* ... */ }
//
//     /* register fooHandler() to handle incoming FooMsg and SimpleHandler to handle SimpleMsg */
//     conn.AddHandlers(fooHandler, simpleHandler)
//

func (mc *MsgConn) AddHandlers(args ...interface{}) error {
	for len(args) > 0 {
		if err := mc.disp.hmap.addHandler(args[0]); err != nil {
			return err
		}
		args = args[1:]
	}
	return nil
}

func (mc *MsgConn) SendMsg(msg interface{}, fds ...int) error {
	return mc.sendMessage(msg, <-mc.idGen, fds...)
}

func (mc *MsgConn) ExchangeMsg(msg interface{}, fds ...int) (ResponseReader, error) {
	id := <-mc.idGen
	rr := mc.respMan.register(id)

	if err := mc.sendMessage(msg, id, fds...); err != nil {
		rr.Done()
		return nil, err
	}
	return rr, nil
}

func (mc *MsgConn) sendMessage(msg interface{}, msgID int, fds ...int) error {
	msgType, err := getMessageType(msg)
	if err != nil {
		return err
	}
	base, err := mc.newBaseMessage(msgType, msgID, msg)
	if err != nil {
		return err
	}
	raw, err := json.Marshal(base)
	if err != nil {
		return err
	}
	buf := make([]byte, len(raw)+4)
	binary.BigEndian.PutUint32(buf, uint32(len(raw)))
	copy(buf[4:], raw)
	return mc.sendRaw(buf, fds...)
}

func getMessageType(msg interface{}) (string, error) {
	t := reflect.TypeOf(msg)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return "", fmt.Errorf("sendMessage() msg (%T) is not a struct", msg)
	}
	if t.NumField() == 0 || len(t.Field(0).Tag) == 0 {
		return "", fmt.Errorf("sendMessage() msg struct (%T) does not have tag on first field")
	}
	return string(t.Field(0).Tag), nil
}

func (mc *MsgConn) newBaseMessage(msgType string, msgID int, body interface{}) (*BaseMsg, error) {
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	base := new(BaseMsg)
	base.Type = msgType
	base.MsgID = msgID
	base.Body = bodyBytes
	return base, nil
}

func (mc *MsgConn) sendRaw(data []byte, fds ...int) error {
	if len(fds) > 0 {
		return mc.sendWithFds(data, fds)
	}
	_, err := mc.conn.Write(data)
	return err
}

func (mc *MsgConn) sendWithFds(data []byte, fds []int) error {
	oob := syscall.UnixRights(fds...)
	_, _, err := mc.conn.WriteMsgUnix(data, oob, nil)
	return err
}
