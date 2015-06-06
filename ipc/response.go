package ipc

import (
	"sync"
	"time"
)

type ResponseReader interface {
	Chan() <-chan *Message
	Done()
}

type responseWaiter struct {
	rm      *responseManager
	id      int
	timeout time.Time
	ch      chan *Message
}

func (rw *responseWaiter) Chan() <-chan *Message {
	return rw.ch
}

func (rw *responseWaiter) Done() {
	rw.rm.lock.Lock()
	defer rw.rm.lock.Unlock()
	close(rw.ch)
	delete(rw.rm.responseMap, rw.id)
}

type responseManager struct {
	lock        sync.Locker
	responseMap map[int]*responseWaiter
}

func newResponseManager() *responseManager {
	rm := new(responseManager)
	rm.lock = new(sync.Mutex)
	rm.responseMap = make(map[int]*responseWaiter)
	return rm
}

func (rm *responseManager) register(id int) ResponseReader {
	ch := make(chan *Message)
	rm.lock.Lock()
	defer rm.lock.Unlock()
	rm.removeById(id, true)
	rw := &responseWaiter{
		rm: rm,
		id: id,
		ch: ch,
	}
	rm.responseMap[id] = rw
	return rw
}

func (rm *responseManager) handle(m *Message) bool {
	rm.lock.Lock()
	defer rm.lock.Unlock()
	rw := rm.responseMap[m.MsgID]
	if rw == nil {
		return false
	}
	rw.ch <- m
	return true
}

func (rm *responseManager) removeById(id int, klose bool) *responseWaiter {
	rw := rm.responseMap[id]
	if rw == nil {
		return nil
	}
	delete(rm.responseMap, id)
	if klose {
		close(rw.ch)
	}
	return rw
}
