package ipc

import (
	"encoding/json"
	"errors"
	"net"
	"syscall"

	"github.com/op/go-logging"
	"reflect"
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

const maxFdCount = 3

var log = logging.MustGetLogger("oz")

type MsgConn struct {
	msgs     chan *Message
	addr     *net.UnixAddr
	conn     *net.UnixConn
	buf      [1024]byte
	oob      []byte
	handlers handlerMap
	factory  MsgFactory
	isClosed bool
	done chan bool
	idGen <-chan int
	respMan *responseManager
}

func NewMsgConn(factory MsgFactory, address string) *MsgConn {
	mc := new(MsgConn)
	mc.addr = &net.UnixAddr{address, "unixgram"}
	mc.oob = createOobBuffer()
	mc.msgs = make(chan *Message)
	mc.handlers = make(map[string]reflect.Value)
	mc.factory = factory
	mc.done = make(chan bool)
	mc.idGen = newIdGen(mc.done)
	mc.respMan = newResponseManager()
	return mc
}

func newIdGen(done <-chan bool) <-chan int {
	ch := make(chan int)
	go idGenLoop(done, ch)
	return ch
}

func idGenLoop(done <-chan bool, out chan <- int) {
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

func (mc *MsgConn) Listen() error {
	if mc.conn != nil {
		return errors.New("cannot Listen(), already connected")
	}
	conn, err := net.ListenUnixgram("unixgram", mc.addr)
	if err != nil {
		return err
	}
	if err := setPassCred(conn); err != nil {
		return err
	}
	mc.conn = conn
	return nil
}

func (mc *MsgConn) Connect() error {
	if mc.conn != nil {
		return errors.New("cannot Connect(), already connected")
	}
	clientAddr,err := CreateRandomAddress("@oz-")
	if err != nil {
		return err
	}
	conn, err := net.DialUnix("unixgram", &net.UnixAddr{clientAddr, "unixgram"}, nil)
	if err != nil {
		return err
	}
	mc.conn = conn
	go mc.readLoop()
	return nil
}

func CreateRandomAddress(prefix string) (string,error) {
	var bs [16]byte
	n,err := rand.Read(bs[:])
	if n != len(bs) {
		return "", errors.New("incomplete read of random bytes for client name")
	}
	if err != nil {
		return "", errors.New("error reading random bytes for client name: "+ err.Error())
	}
	return prefix+ hex.EncodeToString(bs[:]),nil
}

func (mc *MsgConn) Run() error {
	go mc.readLoop()
	for m := range mc.msgs {
		if err := mc.handlers.dispatch(m); err != nil {
			return fmt.Errorf("error dispatching message: %v", err)
		}
	}
	return nil
}

func (mc *MsgConn) readLoop() {
	for {
		if mc.processOneMessage() {
			return
		}
	}
}

func (mc *MsgConn) processOneMessage() bool {
	m,err := mc.readMessage()
	if err != nil {
		close(mc.msgs)
		if !mc.isClosed {
			log.Warning("error on MsgConn.readMessage(): %v", err)
		}
		return true
	}
	if !mc.respMan.handle(m) {
		mc.msgs <- m
	}
	return false
}

func (mc *MsgConn) Close() error {
	mc.isClosed = true
	close(mc.done)
	return mc.conn.Close()
}

func createOobBuffer() []byte {
	oobSize := syscall.CmsgSpace(syscall.SizeofUcred) + syscall.CmsgSpace(4*maxFdCount)
	return make([]byte, oobSize)
}

func (mc *MsgConn) readMessage() (*Message, error) {
	n, oobn, _, a, err := mc.conn.ReadMsgUnix(mc.buf[:], mc.oob)
	if err != nil {
		return nil, err
	}
	m, err := mc.parseMessage(mc.buf[:n])
	if err != nil {
		return nil, err
	}
	m.mconn = mc
	m.Peer = a

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
		if err := mc.handlers.addHandler(args[0]); err != nil {
			return err
		}
		args = args[1:]
	}
	return nil
}

func (mc *MsgConn) SendMsg(msg interface{}, fds... int) error {
	return mc.sendMessage(msg, <-mc.idGen, mc.addr, fds...)
}

func (mc *MsgConn) ExchangeMsg(msg interface{}, fds... int) (ResponseReader, error) {
	id := <-mc.idGen
	rr := mc.respMan.register(id)

	if err := mc.sendMessage(msg, id, mc.addr, fds...); err != nil {
		rr.Done()
		return nil, err
	}
	return rr,nil
}

func (mc *MsgConn) sendMessage(msg interface{}, msgID int, dst *net.UnixAddr, fds... int) error {
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
	return mc.sendRaw(raw, dst, fds...)
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
	bodyBytes,err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	base := new(BaseMsg)
	base.Type = msgType
	base.MsgID = msgID
	base.Body = bodyBytes
	return base, nil
}

func (mc *MsgConn) sendRaw(data []byte, dst *net.UnixAddr, fds ...int) error {
	if len(fds) > 0 {
		return mc.sendWithFds(data, dst, fds)
	}
	return mc.write(data, dst)
}

func (mc *MsgConn) write(data []byte, dst *net.UnixAddr) error {
	if dst != nil {
		_,err := mc.conn.WriteToUnix(data, dst)
		return err
	}
	_,err := mc.conn.Write(data)
	return err
}

func (mc *MsgConn) sendWithFds(data []byte, dst *net.UnixAddr, fds []int) error {
	oob := syscall.UnixRights(fds...)
	_,_,err := mc.conn.WriteMsgUnix(data, oob, dst)
	return err
}

