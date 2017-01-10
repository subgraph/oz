package ipc

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"syscall"
)

func NewMsgFactory(msgTypes ...interface{}) MsgFactory {
	mf := (MsgFactory)(make(map[string]func() interface{}))
	for _, mt := range msgTypes {
		if err := mf.register(mt); err != nil {
			defaultLog.Fatalf("failed adding (%T) in NewMsgFactory: %v", mt, err)
			return nil
		}
	}
	return mf
}

type MsgFactory map[string](func() interface{})

func (mf MsgFactory) create(msgType string) (interface{}, error) {
	f, ok := mf[msgType]
	if !ok {
		return nil, fmt.Errorf("cannot create msg type: %s %v", msgType, ok)
	}
	return f(), nil
}

func (mf MsgFactory) register(mt interface{}) error {
	t := reflect.TypeOf(mt)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return errors.New("not a structure")
	}
	if t.NumField() == 0 || len(t.Field(0).Tag) == 0 {
		return errors.New("no tag on first field of structure")
	}
	tag := string(t.Field(0).Tag)

	mf[tag] = func() interface{} {
		v := reflect.New(t)
		return v.Interface()
	}
	return nil
}

type Message struct {
	Type  string
	MsgID int
	Body  interface{}
	Ucred *syscall.Ucred
	Fds   []int
	mconn *MsgConn
}

type BaseMsg struct {
	Type       string
	MsgID      int
	IsResponse bool
	Body       json.RawMessage
}

func (mc *MsgConn) parseMessage(data []byte) (*Message, error) {
	var base BaseMsg
	if err := json.Unmarshal(data, &base); err != nil {
		return nil, err
	}
	body, err := mc.factory.create(base.Type)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(base.Body, body); err != nil {
		return nil, err
	}
	m := new(Message)
	m.Type = base.Type
	m.MsgID = base.MsgID
	m.Body = body
	return m, nil
}

func (m *Message) Free() {
	for _, fd := range m.Fds {
		syscall.Close(fd)
	}
	m.Fds = nil
}

func (m *Message) parseControlData(data []byte) error {
	cmsgs, err := syscall.ParseSocketControlMessage(data)
	if err != nil {
		return err
	}
	for _, cmsg := range cmsgs {
		switch cmsg.Header.Type {
		case syscall.SCM_CREDENTIALS:
			cred, err := syscall.ParseUnixCredentials(&cmsg)
			if err != nil {
				return err
			}
			m.Ucred = cred
		case syscall.SCM_RIGHTS:
			fds, err := syscall.ParseUnixRights(&cmsg)
			if err != nil {
				return err
			}
			m.Fds = fds
		}
	}
	return nil
}

func (m *Message) Respond(msg interface{}, fds ...int) error {
	return m.mconn.sendMessage(msg, m.MsgID, fds...)
}
