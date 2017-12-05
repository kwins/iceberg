package frame

import (
	"context"
	"errors"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	log "github.com/kwins/iceberg/frame/icelog"
	"github.com/kwins/iceberg/frame/protocol"
	"github.com/opentracing/opentracing-go"
)

const (
	CA_OK        = iota // 0:连接正常
	CA_BROKEN    = iota // 1:连接已断开
	CA_RECONNING = iota // 2:正在重连
	CA_ABANDON   = iota // 3:重连失败放弃connactor对象
)

// 定义错误类型
var (
	ErrBlocking       = errors.New("operation blocking")
	ErrClosed         = errors.New("connection is closed")
	ErrTimeout        = errors.New("netio timeout")
	ErrMethodNotFound = errors.New("method not found")
)

var connActorID uint32 // 分配ID的静态变量，用原子操作改变它的值

const sendPackBufSize = 4096

// ConnActorType TCP 连接类型 1-passive 2-active
type ConnActorType int8

const (
	passiveConnActor ConnActorType = 1
	activeConnActor  ConnActorType = 2
)

// ConnActor 连接对象
type ConnActor struct {
	id            uint32   //
	c             net.Conn //
	reconn        bool     //
	connType      ConnActorType
	requestHolder *Dispatcher //

	status int32 // 0:连接正常  1:连接已断开  2:正在重连 3:重连失败放弃connactor对象

	sendChan chan []byte    //
	stopChan chan struct{}  //
	stopWait sync.WaitGroup //
	// method map[string]
	stopOnce sync.Once //
}

// NewPassiveConnActor Iceberg下层服务需建立此种连接，用于接收并处理数据
func NewPassiveConnActor(c net.Conn) *ConnActor {
	ca := ConnActor{c: c, reconn: false}
	ca.id = atomic.AddUint32(&connActorID, CA_BROKEN)
	ca.connType = passiveConnActor
	ca.initConnActor(c)

	return &ca
}

// NewActiveConnActor 生成一个主动的连接，主动向对端发送请求并等待响应的连接
func NewActiveConnActor(c net.Conn) *ConnActor {
	ca := ConnActor{c: c, reconn: true}
	ca.id = atomic.AddUint32(&connActorID, CA_BROKEN)
	ca.requestHolder = NewDispatcher()
	ca.connType = activeConnActor
	ca.initConnActor(c)

	return &ca
}

func (connActor *ConnActor) initConnActor(c net.Conn) {
	go ContinuousRecvPack(c, connActor.processInComing)

	connActor.sendChan = make(chan []byte, sendPackBufSize)
	connActor.stopChan = make(chan struct{})
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
							connActor.sendChan <- msg
							continue
						}
					}
					failCount = 0
				}
			case <-connActor.stopChan:
				var req protocol.Proto
				for {
					msg, ok := <-connActor.sendChan
					if !ok {
						return
					}
					req.UnSerialize(msg)
					log.Warnf("%s connect[%s-%s] is closed, drop msg now, msg len=%d", req.PrintableBizID, connActor.c.LocalAddr().String(), connActor.RemoteAddr(), len(msg))
					connActor.requestHolder.Delete(req.RequestID)
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
	default:
		return ErrBlocking
	}
	return nil
}

// RequestAndReponse 向特定的服务发送请求，并等待响应
func (connActor *ConnActor) RequestAndReponse(b []byte, requstID int64) (*protocol.Proto, error) {
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
	}
}

// Close 连接关闭
func (connActor *ConnActor) Close() {
	connActor.stopOnce.Do(func() {
		atomic.StoreInt32(&connActor.status, CA_ABANDON)
		connActor.reconn = false
		close(connActor.stopChan)
		close(connActor.sendChan)
		connActor.stopWait.Wait()
		if connActor.c != nil {
			connActor.c.Close()
		}
	})
	log.Infof("Close connactor. %s-%s", connActor.c.LocalAddr().String(), connActor.RemoteAddr())
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
			log.Infof("reDial successed. %s-%s", connActor.c.LocalAddr().String(), connActor.RemoteAddr())
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

		var task protocol.Proto
		task.UnSerialize(packbuf)
		var resp = task.Shadow()
		var s = DiscoverInstance()

		if sd := s.getMethod(task.GetServeMethod()); sd == nil {
			resp.FillErrInfo(
				http.StatusInternalServerError,
				errNotFoundMethod)
		} else {

			// 默认context中携带bizid，用于分布式服务追踪
			todo := context.TODO()
			bizid := task.GetBizid()
			ctx := context.WithValue(todo, "bizid", bizid)

			// 如果服务增加了zipkin的配置，则在context中增加span的信息
			if span := SpanFromTask(&task); span != nil {
				ctx = opentracing.ContextWithSpan(ctx, span)
				defer span.Finish()
			}

			// call method
			out, err := sd.Handler(s.service,
				ctx, task.GetFormat(), task.GetBody())
			if err != nil {
				resp.FillErrInfo(http.StatusInternalServerError, err)
			} else {
				resp.Body = out
			}
		}
		b, _ := resp.Serialize()
		// 写回响应数据
		connActor.Write(b)

	case activeConnActor:
		connActor.requestHolder.Incoming(packbuf, connActor)
	}
}
