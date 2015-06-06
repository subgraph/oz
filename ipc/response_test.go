package ipc

import (
	"testing"
)

func TestRegister(t *testing.T) {
	rm := newResponseManager()
	if len(rm.responseMap) != 0 {
		t.Error("responseMap should be empty")
	}
	rm.register(1)
	rm.register(2)
	rm.register(1)
	if len(rm.responseMap) != 2 {
		t.Errorf("responseMap should have 2 items, not %d", len(rm.responseMap))
	}
}

func TestRemoveById(t *testing.T) {
	rm := newResponseManager()
	rm.register(1)
	rm.register(2)
	rm.register(3)

	rm.removeById(2, true)
	rm.removeById(2, true)

	if len(rm.responseMap) != 2 {
		t.Errorf("responseMap should have 2 items, not %d", len(rm.responseMap))
	}
	rm.removeById(1, true)
	rm.removeById(3, true)
	if len(rm.responseMap) != 0 {
		t.Errorf("responseMap should have 0 items, not %d", len(rm.responseMap))
	}
}

func TestHandle(t *testing.T) {
	m := new(Message)
	rm := newResponseManager()
	rr := rm.register(1)
	rm.register(2)
	m.MsgID = 3
	if rm.handle(m) {
		t.Errorf("handle() should have returned false")
	}
	go func() {
		<-rr.Chan()
	}()
	m.MsgID = 1
	if !rm.handle(m) {
		t.Errorf("handle() should have returned true")
	}
	if len(rm.responseMap) != 2 {
		t.Errorf("responseMap should have 2 items after handling message")

	}
}
