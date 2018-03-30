package frame

import (
	log "github.com/kwins/iceberg/frame/icelog"
	"github.com/kwins/iceberg/frame/protocol"
	"sync"
)

const bucketNo = 16

// Dispatcher 请求分发
type Dispatcher struct {
	p    *sync.Pool
	reqs []*Holder
}

// NewDispatcher new dispatcher
func NewDispatcher() *Dispatcher {
	dispatcher := new(Dispatcher)
	dispatcher.p = &sync.Pool{
		New: func() interface{} {
			return make(chan *protocol.Proto)
		}}
	dispatcher.reqs = make([]*Holder, bucketNo)
	for i := range dispatcher.reqs {
		dispatcher.reqs[i] = NewHolder(i)
	}
	return dispatcher
}

// Incoming 回调
func (dispatcher *Dispatcher) Incoming(b []byte, ca *ConnActor) {
	var resp protocol.Proto
	resp.UnSerialize(b)
	h := dispatcher.reqs[resp.GetRequestID()%bucketNo]
	if ch := h.Get(resp.GetRequestID()); ch != nil {
		ch <- &resp
	} else {
		log.Warnf("%s not found origin request[%d]. drop dispatcher response detail:%s",
			resp.GetBizid(), resp.GetRequestID(), resp.String())
	}
}

// Put put request enqueue
func (dispatcher *Dispatcher) Put(requstID int64) chan *protocol.Proto {
	ch := dispatcher.p.Get().(chan *protocol.Proto)
	dispatcher.reqs[requstID%bucketNo].Put(requstID, ch)
	return ch
}

// Delete give up req id
func (dispatcher *Dispatcher) Delete(id int64) {
	dispatcher.reqs[id%bucketNo].Delete(id)
}
