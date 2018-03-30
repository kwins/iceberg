package frame

import (
	"container/list"
	log "github.com/kwins/iceberg/frame/icelog"
	"github.com/kwins/iceberg/frame/protocol"
	"sync"
	"time"
)

const cell = time.Second * 10
const ticks = 1

// Holder holder that hold all request
type Holder struct {
	id      int
	locker  sync.RWMutex
	request map[int64]chan *protocol.Proto
	bulked  map[time.Time]*list.List
}

// NewHolder new holder
func NewHolder(i int) *Holder {
	hd := new(Holder)
	hd.id = i
	hd.request = make(map[int64]chan *protocol.Proto)
	hd.bulked = make(map[time.Time]*list.List)
	go hd.autoGc()
	return hd
}

// Get get request chan
func (h *Holder) Get(reqID int64) chan *protocol.Proto {
	h.locker.RLock()
	if req, found := h.request[reqID]; found {
		h.locker.RUnlock()
		return req
	}
	h.locker.RUnlock()
	return nil
}

// Put put request
func (h *Holder) Put(reqID int64, ch chan *protocol.Proto) {
	h.locker.Lock()
	if _, found := h.request[reqID]; found {
		h.locker.Unlock()
		log.Warn("the request is existed, requestID=", reqID)
		return
	}

	point := ShapingTime(time.Now(), cell, ticks)
	if l, found := h.bulked[point]; found {
		l.PushFront(reqID)
	} else {
		ll := list.New()
		ll.PushFront(reqID)
		h.bulked[point] = ll
	}

	h.request[reqID] = ch
	h.locker.Unlock()
	return
}

// GiveUp give up request
func (h *Holder) GiveUp(reqID int64) {
	h.locker.Lock()
	if _, found := h.request[reqID]; found {
		delete(h.request, reqID)
	}
	h.locker.Unlock()
}

// Delete delete req
func (h *Holder) Delete(reqID int64) {
	h.locker.Lock()
	delete(h.request, reqID)
	h.locker.Unlock()
}

// autoGc auto close timeout request
func (h *Holder) autoGc() {
	t := time.NewTicker(cell)
	for {
		now := <-t.C
		var tmp = make(map[time.Time]*list.List)

		h.locker.RLock()
		for k, v := range h.bulked {
			tmp[k] = v
		}
		h.locker.RUnlock()

		for k, v := range tmp {
			if k.Before(now) {
				h.locker.Lock()
				for e := v.Front(); e != nil; e = e.Next() {
					reqid := e.Value.(int64)
					if ch := h.request[reqid]; ch != nil {
						close(ch)
						delete(h.request, reqid)
						log.Warn("delete timeout request, requestID=", reqid)
					}
				}
				delete(h.bulked, k)
				h.locker.Unlock()
			}
		}
	}
}
