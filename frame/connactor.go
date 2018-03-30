package frame

import (
	"context"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	log "github.com/kwins/iceberg/frame/icelog"
	"github.com/kwins/iceberg/frame/protocol"
)

const (
	CA_OK        = iota // 0:连接正常
	CA_BROKEN    = iota // 1:连接已断开
	CA_RECONNING = iota // 2:正在重连
	CA_ABANDON   = iota // 3:重连失败放弃connactor对象
)

var connActorID uint32 // 分配ID的静态变量，用原子操作改变它的值

const sendPackBufSize = 1024

// ConnActorType TCP 连接类型 1-passive 2-active
type ConnActorType int8

const (
	passiveConnActor ConnActorType = 1
	activeConnActor  ConnActorType = 2
)

// ConnActor 连接对象
type ConnActor struct {
	id uint32

	c net.Conn

	reconn bool

	connType ConnActorType

	requestHolder *Dispatcher

	p *sync.Pool
	// 0:连接正常  1:连接已断开  2:正在重连 3:重连失败放弃connactor对象
	status int32

	ctx    context.Context
	cancel context.CancelFunc

	sendChan chan []byte
	stopWait sync.WaitGroup

	stopOnce sync.Once
}

// NewPassiveConnActor Iceberg下层服务需建立此种连接，用于接收并处理数据
func NewPassiveConnActor(c net.Conn) *ConnActor {
	ca := ConnActor{c: c, reconn: false}
	ca.ctx, ca.cancel = context.WithCancel(context.TODO())
	ca.id = atomic.AddUint32(&connActorID, CA_BROKEN)
	ca.connType = passiveConnActor
	ca.p = &sync.Pool{
		New: func() interface{} {
			return new(icecontext)
		}}
	ca.initConnActor(c)
	return &ca
}

// NewActiveConnActor 生成一个主动的连接，主动向对端发送请求并等待响应的连接
func NewActiveConnActor(c net.Conn) *ConnActor {
	ca := ConnActor{c: c, reconn: true}
	ca.ctx, ca.cancel = context.WithCancel(context.TODO())
	ca.id = atomic.AddUint32(&connActorID, CA_BROKEN)
	ca.requestHolder = NewDispatcher()
	ca.connType = activeConnActor
	ca.initConnActor(c)
	return &ca
}

func (connActor *ConnActor) initConnActor(c net.Conn) {
	go ContinuousRecvPack(c, connActor.processInComing)

	connActor.sendChan = make(chan []byte, sendPackBufSize)
	connActor.stopWait.Add(1)
	go func() {
		defer connActor.stopWait.Done()
		var failCount int
		var tempDelay = 5 * time.Millisecond
		for {
			select {
			case msg, ok := <-connActor.sendChan:
				if ok {
					if atomic.LoadInt32(&connActor.status) != CA_OK {
						failCount++
						// 连接断开，准备重发
						time.Sleep(tempDelay * time.Duration(failCount))
						connActor.sendChan <- msg
						continue
					}
					sentbytes := SendAll(connActor.c, msg)
					if sentbytes != len(msg) {
						// 发送失败
						failCount++
						if failCount > 5 {
							if connActor.requestHolder != nil {
								var req protocol.Proto
								req.UnSerialize(msg)
								connActor.requestHolder.Delete(req.RequestID)
							}
						} else {
							// 准备重发
							log.Warnf("send data to %s fail,repush to send chan len=%d", connActor.RemoteAddr(), len(msg))
							connActor.sendChan <- msg
							continue
						}
					} else {
						failCount = 0
						log.Debugf("send data to %s finished,data len=%d", connActor.RemoteAddr(), len(msg))
					}
				}
			case <-connActor.ctx.Done():
				var req protocol.Proto
				for {
					select {
					case msg, ok := <-connActor.sendChan:
						if !ok {
							return
						}
						req.UnSerialize(msg)
						log.Warnf("drop msg %s now, msg len=%d", req.GetBizid(), len(msg))
						connActor.requestHolder.Delete(req.RequestID)
					default:
						return
					}
				}
			}
		}
	}()
}

// Write 向连接上写入数据, 参数：
// b []byte 待写入的数据
// 返回值：n 成功写入的字节数;  err 写入时发生的错误
func (connActor *ConnActor) Write(b []byte) error {
	if atomic.LoadInt32(&connActor.status) != CA_OK {
		if !connActor.reconn || !connActor.reDial() {
			return ErrClosed
		}
	}
	select {
	case connActor.sendChan <- b:
		return nil
	default:
		return ErrBlocking
	}
}

// RequestAndReponse 向特定的服务发送请求，并等待响应
func (connActor *ConnActor) RequestAndReponse(b []byte,
	requstID int64) (*protocol.Proto, error) {
	// 先把请求加入请求池中
	ch := connActor.requestHolder.Put(requstID)

	if err := connActor.Write(b); err != nil {
		return nil, err
	}
	// 等待响应
	select {
	case resp, ok := <-ch:
		if !ok {
			return nil, ErrTimeout
		}
		// 超时的 chan *protocal.Proto不能再放到池子中，因为已经关闭了
		connActor.requestHolder.p.Put(ch)
		connActor.requestHolder.Delete(requstID)
		return resp, nil
	case <-connActor.ctx.Done():
		return nil, ErrClosed
	}
}

// Close 连接关闭
func (connActor *ConnActor) Close() {
	connActor.stopOnce.Do(func() {
		atomic.StoreInt32(&connActor.status, CA_ABANDON)
		connActor.reconn = false
		connActor.cancel()
		connActor.stopWait.Wait()
		close(connActor.sendChan)
		if connActor.c != nil {
			connActor.c.Close()
		}
	})
	log.Debugf("close connactor. %s-%s",
		connActor.c.LocalAddr().String(), connActor.RemoteAddr())
}

// RemoteAddr 取得连接的目的地址
func (connActor *ConnActor) RemoteAddr() string {
	return connActor.c.RemoteAddr().String()
}

// Status 获取连接状态
func (connActor *ConnActor) Status() int32 {
	return atomic.LoadInt32(&connActor.status)
}

func (connActor *ConnActor) reDial() bool {
	// NOTE:重连操作不能重入!
	if !atomic.CompareAndSwapInt32(&connActor.status, CA_BROKEN, CA_RECONNING) {
		return false
	}

	log.Warnf("Try to redial to:%s", connActor.RemoteAddr())
	var tempDelay = 5 * time.Millisecond
	for {
		conn, err := net.Dial("tcp", connActor.c.RemoteAddr().String())
		if err == nil {
			connActor.c = conn
			go ContinuousRecvPack(connActor.c, connActor.processInComing)
			atomic.StoreInt32(&connActor.status, CA_OK)
			log.Debugf("reDial successed. %s-%s", connActor.c.LocalAddr().String(), connActor.RemoteAddr())
			return true
		}
		if tempDelay > time.Second {
			atomic.StoreInt32(&connActor.status, CA_ABANDON)
			log.Error("reDial failed!")
			connActor.Close()
			return false
		}

		time.Sleep(tempDelay)
		tempDelay *= 2
	}
}

func (connActor *ConnActor) processInComing(packbuf []byte) {
	if packbuf == nil { // 连接断开
		log.Warnf("Learn about connection broken. %s-%s",
			connActor.c.LocalAddr().String(), connActor.RemoteAddr())
		atomic.StoreInt32(&connActor.status, CA_BROKEN)
		if !connActor.reconn {
			connActor.Close()
			return
		}

		// 重建连接
		if connActor.reconn {
			connActor.reDial()
		}
		return
	}

	// 将接收到的数据交给回调接口处理
	switch connActor.connType {
	case passiveConnActor:
		var r protocol.Proto
		if err := r.UnSerialize(packbuf); err != nil {
			log.Errorf("receive bad pack,unserialize fail,detail=%s", err.Error())
			return
		}

		var w = r.Shadow()
		c := connActor.p.Get().(*icecontext)
		c.Reset(&r, &w)
		var s = Instance()
		if sd := s.getMethod(r.GetServeMethod()); sd == nil {
			c.Response().FillErrInfo(http.StatusNotFound, ErrMethodNotFound)
		} else {
			log.Info(r.AsString())
			for i := range s.prepare {
				if err := s.prepare[i](c); err != nil {
					c.Response().FillErrInfo(
						http.StatusInternalServerError, err)
					goto REPLY
				}
			}

			if err := sd.Handler(s.service, c); err != nil {
				c.Response().FillErrInfo(
					http.StatusInternalServerError, err)
			} else if len(c.Response().GetBody()) == 0 {
				c.JSON2(0, "success", nil)
			}

			for i := range s.after {
				if err := s.after[i](c); err != nil {
					c.Response().FillErrInfo(
						http.StatusInternalServerError, err)
					c.Response().Body = nil
				}
			}
		REPLY:
			log.Info(c.Response().AsString())
		}
		connActor.p.Put(c)
		// 写回响应数据
		b, _ := c.Response().Serialize()
		connActor.Write(b)
	case activeConnActor:
		connActor.requestHolder.Incoming(packbuf, connActor)
	}
}
