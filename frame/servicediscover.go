package frame

import (
	"context"
	"errors"
	"fmt"
	"net"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/kwins/iceberg/frame/config"
	log "github.com/kwins/iceberg/frame/icelog"
	"github.com/kwins/iceberg/frame/protocol"

	"github.com/coreos/etcd/clientv3"
	"github.com/opentracing/opentracing-go"
	zipkin "github.com/openzipkin/zipkin-go-opentracing"
	"google.golang.org/grpc"
)

const root = "/"

// TopoChange 拓扑变化通过的数据结构
// URI 拓扑在服务体系树中的位置;
// Conn 到新实例的连接;
// NodeHashKey 实例节点的HashKey;
// 标识当前变更是否是新增一个实点;
type TopoChange struct {
	URI         string
	Conn        *ConnActor
	NodeHashKey string
	NewNode     bool
}

var (
	errForgetSelfURI        = errors.New("Forget self uri ?")
	errConnectURIIsNil      = errors.New("Connect uri not found")
	errNotFoundConnect      = errors.New("Not found connection")
	errConnectRemoteAddrnil = errors.New("Connect remoteAddr is null")
)

// Discover 服务发现的类结构
// topology 拓扑表的根节点;
// topoLocker 保护拓扑表的锁;
// kapi 连接到etcd的客户端会话handle;
// selfURI 当前进程自己在服务树中的位置;
// name 服务名称;
// localListenAddr 本地监听的地址;
type Discover struct {
	// all topo info
	topology   map[string]*ConsistentHash // 系统的拓扑结构; key是接口的URI
	topoLocker sync.RWMutex

	// etcd client api
	kapi *clientv3.Client

	// hold all connect that have visited.
	connholder map[string]*ConnActor
	connLocker sync.RWMutex

	// self uri that register to etcd
	// you can register multi uri
	selfURI []string
	name    string

	// your server
	service interface{} // 提供服务

	// middleware
	prepare []Middleware
	after   []Middleware

	// server describe
	mdLocker sync.RWMutex // method
	md       map[string]*MethodDesc

	// 其他服务方法映射
	mtLocker sync.RWMutex
	mdtables map[string]*Medesc

	localListenAddr string

	innerid int64 // 内部请求ID

	ctx    context.Context
	cancel context.CancelFunc
}

var discoverOnce sync.Once
var instance *Discover

// Instance 返回GateSvr的单例对象
func Instance() *Discover {
	discoverOnce.Do(func() {
		instance = new(Discover)
		instance.md = make(map[string]*MethodDesc)
		instance.mdtables = make(map[string]*Medesc)
		instance.ctx, instance.cancel = context.WithCancel(context.TODO())
		instance.topology = make(map[string]*ConsistentHash)
		instance.connholder = make(map[string]*ConnActor)
	})
	return instance
}

// RegisterAndServe 后端服务注册并开启
func RegisterAndServe(sd *ServiceDesc, ss interface{}, cfg *config.BaseCfg) {
	ht := reflect.TypeOf(sd.HandlerType).Elem()
	st := reflect.TypeOf(ss)
	if !st.Implements(ht) {
		log.Fatalf("iceberg: RegisterAndServe found the handler of type %v that does not satisfy %v", st, ht)
		return
	}

	sh := NewSignalHandler()
	var h Singal
	sht := reflect.TypeOf((*Singal)(nil)).Elem()
	if !st.Implements(sht) {
		h = defaultServerSignal
	} else {
		h = ss.(Singal)
	}
	sh.Register(syscall.SIGTERM, h)
	sh.Register(syscall.SIGQUIT, h)
	sh.Register(syscall.SIGINT, h)
	sh.Start()

	s := Instance()

	// 注册本服务信息
	s.service = ss
	for i := range sd.Methods {
		d := &sd.Methods[i]
		s.mdLocker.Lock()
		s.md[d.MethodName] = d
		s.mdLocker.Unlock()
	}

	// 向ETCD注册信息
	s.Start(sd.ServiceName, cfg, sd.ServiceURI, "")
	// 监听
	add, err := net.ResolveTCPAddr("tcp", s.localListenAddr)
	if err != nil {
		panic(err.Error())
	}
	listener, err := net.ListenTCP("tcp", add)
	if err != nil {
		panic(err.Error())
	}

	for {
		c, err := listener.Accept()
		if err != nil {
			log.Error("iceberg:", err.Error())
			continue
		}
		ca := ConnActor{c: c, reconn: false}
		ca.ctx, ca.cancel = context.WithCancel(context.TODO())
		ca.id = atomic.AddUint32(&connActorID, CA_BROKEN)
		ca.connType = passiveConnActor
		ca.p = &sync.Pool{
			New: func() interface{} {
				return new(icecontext)
			}}
		ca.initConnActor(c)
	}
}

// GetInnerID 获取内部服务ID
func GetInnerID() int64 {
	return atomic.AddInt64(&Instance().innerid, 1)
}

// MeTables 获取方法 集合
func MeTables() map[string]Medesc {
	var mt = make(map[string]Medesc)
	Instance().mtLocker.RLock()
	for k, v := range Instance().mdtables {
		mt[k] = *v
	}
	Instance().mtLocker.RUnlock()
	return mt
}

// DeliverTo deliver request to anthor serve
func DeliverTo(task *protocol.Proto) (*protocol.Proto, error) {
	conn, err := Instance().Get(task.GetServeURI())
	if err != nil {
		log.Error(err.Error())
		return nil, err
	}
	var b []byte
	if b, err = task.Serialize(); err != nil {
		return nil, err
	}
	resp, err := conn.RequestAndReponse(b, task.GetRequestID())
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// Prepare 添加prepare middleware
func Prepare(mw ...Middleware) {
	Instance().prepare = append(Instance().prepare, mw...)
}

// After 添加after middleware
func After(mw ...Middleware) {
	Instance().after = append(Instance().after, mw...)
}

// Start 开启服务发现机制
func (discover *Discover) Start(srvName string, cfg *config.BaseCfg, selfURI []string, address string) {
	discover.selfURI = selfURI
	discover.name = srvName
	discover.localListenAddr = address

	if err := discover.readyEtcd(&cfg.Etcd); err != nil {
		panic(err.Error())
	}
	if len(selfURI) == 0 {
		panic(errForgetSelfURI.Error())
	}
	// 监听内网环境IP
	if discover.localListenAddr == "" {
		discover.localListenAddr = Netip() + ":" + RandPort()
	}
	// 注册自己
	if err := discover.selfRegist(); err != nil {
		panic(err.Error())
	}
	go discover.discover()
	// 程序启动告警
	log.Infof("%s start up,local listen addr:%s,serve uri:%v", discover.name, discover.localListenAddr, discover.selfURI)
}

// StartZipkinTrace 启动zipkin 暂不使用
func (discover *Discover) StartZipkinTrace(endPoint, srvHost, srvName string) error {
	// Create our HTTP collector.
	if endPoint == "" {
		return nil
	}
	collector, err := zipkin.NewHTTPCollector(endPoint)
	if err != nil {
		return err
	}

	// Create our recorder.
	recorder := zipkin.NewRecorder(collector, true, srvHost, srvName)

	// Create our tracer.
	tracer, err := zipkin.NewTracer(
		recorder, zipkin.ClientServerSameSpan(true),
	)
	if err != nil {
		return err
	}
	// Explicitely set our tracer to be the default tracer.
	opentracing.InitGlobalTracer(tracer)
	log.Infof("start zipkin trace endpoint:%s,srvHost:%s,srvName:%s", endPoint, srvHost, srvName)

	return nil
}

// Get 获取URI对应的一个可用连接
func (discover *Discover) Get(URI string) (*ConnActor, error) {
	discover.topoLocker.RLock()
	if node, ok := discover.topology[URI]; ok {
		discover.topoLocker.RUnlock()
		return discover.getConnActor(node.Leastload(), URI)
	}
	discover.topoLocker.RUnlock()
	return nil, fmt.Errorf("%s not found in topology", URI)
}

// Allowed 是否允许不认证直接访问，给Gateway使用
func (discover *Discover) Allowed(path string) bool {
	low := strings.ToLower(path)
	ps := strings.Split(low, "/")
	psl := len(ps)
	if psl < 4 {
		return false
	}

	mk := strings.Join(intercept(ps), "@")

	discover.mtLocker.RLock()
	if md := discover.mdtables[mk]; md == nil {
		discover.mtLocker.RUnlock()
		return false
	} else {
		discover.mtLocker.RUnlock()
		return md.Allowed
	}
}

func intercept(ss []string) []string {
	return ss[2:5]
}

// 从连接池中拿到远端连接句柄
func (discover *Discover) getConnActor(remoteAddr, uri string) (*ConnActor, error) {
	if len(remoteAddr) == 0 {
		return nil, errConnectRemoteAddrnil
	}
	if len(uri) == 0 {
		return nil, errConnectURIIsNil
	}
	// 找到了节点。取出/新建连接
	var connactor *ConnActor

	createConn := func() error {
		log.Debug("try ot connect:", remoteAddr)
		c, err := net.Dial("tcp", remoteAddr)
		if err != nil {
			log.Error(err.Error())
			return err
		}
		log.Debugf("connect backend serve %s success[%s]", remoteAddr, uri)
		connactor = NewActiveConnActor(c)
		discover.connLocker.Lock()
		discover.connholder[remoteAddr] = connactor
		discover.connLocker.Unlock()
		return nil
	}

	discover.connLocker.RLock()
	connactor, found := discover.connholder[remoteAddr]
	discover.connLocker.RUnlock()
	if !found {
		if err := createConn(); err != nil {
			return nil, err
		}
	} else if connactor.Status() == CA_ABANDON {
		if err := createConn(); err != nil {
			return nil, err
		}
	}
	return connactor, nil
}

// Dispatch 找出请求被分派到哪一个实例去处理
// URI 请求的接口路径
func (discover *Discover) Dispatch(URI string) (*ConnActor, error) {
	// 使用贪婪匹配模式，即匹配更长的URI

	var (
		matchedLen  int
		remoteAddr  string
		registerURI string
	)

	discover.topoLocker.RLock()
	defer discover.topoLocker.RUnlock()

	for k, v := range discover.topology {
		if strings.HasPrefix(URI, k) {
			if len(k) > matchedLen {
				matchedLen = len(k)
				registerURI = k
				if remoteAddr = v.Leastload(); remoteAddr == "" {
					return nil, errNotFoundConnect
				}
			}
		}
	}
	if matchedLen > 0 {
		// 找到了节点。取出/新建连接
		return discover.getConnActor(remoteAddr, registerURI)
	}
	return nil, errNotFoundConnect
}

func (discover *Discover) selfRegist() error {
	if len(discover.selfURI) == 0 {
		return errForgetSelfURI
	}
	for _, uri := range discover.selfURI {
		resp, err := discover.kapi.Grant(context.TODO(), 21)
		if err != nil {
			return err
		}
		leaseResp, err := discover.kapi.KeepAlive(context.TODO(), resp.ID)
		if err != nil {
			return err
		}

		svrURI := uri + "/provider/name"
		log.Debugf("set %s=%s", svrURI, discover.name)
		_, err = discover.kapi.Put(context.TODO(), svrURI, discover.name, clientv3.WithLease(resp.ID))
		if err != nil {
			return err
		}

		// 先KeepAlive 在Put临时节点
		svrURI = uri + "/provider/instances/" + discover.localListenAddr
		log.Debugf("set %s=%s with leaseid=%x", svrURI, discover.localListenAddr, resp.ID)
		_, err = discover.kapi.Put(context.TODO(), svrURI, discover.localListenAddr, clientv3.WithLease(resp.ID))
		if err != nil {
			return err
		}

		// 注册方法表
		for k, v := range discover.md {
			mdname := uri + "/" + strings.ToLower(v.MethodName) + "/provider/allowed/" + v.Allowed
			_, err := discover.kapi.Put(context.TODO(), mdname, k, clientv3.WithLease(resp.ID))
			if err != nil {
				return err
			}
		}

		go func(uri string, leaseid clientv3.LeaseID) {
			t := time.NewTicker(time.Second * 10)
			for {
				select {
				case <-leaseResp:
				case <-t.C:
					gResp, err := discover.kapi.Get(context.TODO(), uri)
					if err != nil || len(gResp.Kvs) == 0 {
						log.Fatalf("iceberg:%s svr uri %s get fail,detail=%v",
							discover.name, uri, err)
						discover.kapi.Put(context.TODO(),
							uri, discover.localListenAddr, clientv3.WithLease(leaseid))
					}
				case <-discover.ctx.Done():
					return
				}
			}
		}(svrURI, resp.ID)
	}
	return nil
}

func (discover *Discover) getMethod(mdName string) *MethodDesc {
	discover.mdLocker.RLock()
	defer discover.mdLocker.RUnlock()
	if d, ok := discover.md[mdName]; ok {
		return d
	}
	return nil
}

func (discover *Discover) readyEtcd(cfg *config.EtcdCfg) error {
	api, err := clientv3.New(clientv3.Config{
		Endpoints: cfg.EndPoints,
		Username:  cfg.User,
		Password:  cfg.Psw,

		DialOptions: []grpc.DialOption{
			grpc.WithTimeout(time.Second * 3),
			grpc.WithInsecure(),
		},

		DialTimeout: time.Second * cfg.Timeout,
	})
	if err != nil {
		return err
	}
	discover.kapi = api
	resp, err := discover.kapi.Get(context.TODO(), root, clientv3.WithPrefix())
	if err != nil {
		return err
	}
	for _, subNode := range resp.Kvs {
		if len(subNode.Key) == 0 || len(subNode.Value) == 0 {
			log.Warnf("iceberg:ready etcd key=%s value=%s", string(subNode.Key), string(subNode.Value))
		} else {
			discover.setTopo(string(subNode.Key), string(subNode.Value))
		}
	}
	return nil
}

func (discover *Discover) discover() {
	ch := discover.kapi.Watch(context.TODO(), root, clientv3.WithPrefix())
	for {
		select {
		case notify := <-ch:
			if notify.Err() != nil {
				log.Warn("iceberg:", notify.Err())
				continue
			}
			for _, event := range notify.Events {
				key := string(event.Kv.Key)
				value := string(event.Kv.Value)
				log.Debugf("iceberg:watch event:%s key:%s value:%s leasid:%x",
					event.Type.String(), key, value, event.Kv.Lease)
				switch event.Type {
				case clientv3.EventTypePut:
					discover.setTopo(key, value)
				case clientv3.EventTypeDelete:
					discover.rmTopo(key, value)
				}
			}
		case <-discover.ctx.Done():
			log.Infof("iceberg:dicover watch graceful exit.")
			return
		}
	}
}

func (discover *Discover) setTopo(key, value string) {
	segment := strings.Split(string(key), "/")
	var segl int
	if segl = len(segment); segl < 3 {
		return
	}
	if leafname := segment[segl-1]; leafname == "config" {

	} else if leafname == "name" {

	} else if segment[segl-2] == "instances" {
		interfaceURI := strings.Join(segment[:segl-3], "/")
		discover.regist(interfaceURI, value)

	} else if segment[segl-2] == "allowed" {
		discover.addMethod(key, value)
	}
}

func (discover *Discover) addMethod(mdkey, mdValue string) {
	if len(mdkey) < len(root) {
		return
	}

	ns := strings.Split(mdkey, "/")
	nsl := len(ns)
	if nsl < 4 {
		return
	}
	var md Medesc
	if ns[nsl-1] == "true" {
		md.Allowed = true
	} else {
		md.Allowed = false
	}
	md.MdName = mdValue
	mk := strings.Join(intercept(ns), "@")
	discover.mtLocker.Lock()
	discover.mdtables[mk] = &md
	discover.mtLocker.Unlock()
}

func (discover *Discover) delMethod(mdkey string) {
	ns := strings.Split(mdkey, "/")
	nsl := len(ns)
	if nsl < 4 {
		return
	}
	mk := strings.Join(intercept(ns), "@")
	discover.mtLocker.Lock()
	delete(discover.mdtables, mk)
	discover.mtLocker.Unlock()
}

func (discover *Discover) rmTopo(key, value string) {
	segment := strings.Split(string(key), "/")
	var l int
	if l = len(segment); l < 3 {
		return
	}
	if leafname := segment[l-1]; leafname == "config" {
		// TO DO
	} else if leafname == "name" {
		// TO DO
	} else if segment[l-2] == "instances" {
		interfaceURI := strings.Join(segment[:l-3], "/")
		log.Debug("rmTopo:", interfaceURI, " ", segment[l-1])
		discover.unRegist(interfaceURI, segment[l-1])
	} else if segment[l-2] == "allowed" {
		// discover.delMethod(key)
	}
}

// 注册一个后台服务接口
func (discover *Discover) regist(URI string, svrAddr string) {
	if len(URI) == 0 {
		return
	}

	// 过滤掉监听到自己的状态变化产生的通知
	if discover.localListenAddr == svrAddr {
		log.Debugf("iceberg:discover self node changed %s", svrAddr)
		return
	}

	discover.topoLocker.Lock()
	defer discover.topoLocker.Unlock()
	var topo *ConsistentHash
	var found bool

	if topo, found = discover.topology[URI]; !found {
		topo = NewConsistentHash()
		discover.topology[URI] = topo
		log.Debugf("Regist a new service at direction %s, the addr is %s", URI, svrAddr)
	}

	// 用后台服务的地址作为key来生成hash节点
	log.Debugf("AddNode: %s svrAddr:%s", URI, svrAddr)
	topo.AddNode(svrAddr)
}

// 注销一个后台服务接口
// URI 要操作的服务
// nodeHashKey 指定的实例hash节点，如果不指定则清空所有节点
func (discover *Discover) unRegist(URI string, nodeHashKey string) {
	if len(URI) == 0 {
		return
	}
	if len(nodeHashKey) > 0 {
		discover.topoLocker.Lock()
		defer discover.topoLocker.Unlock()
		if topo, found := discover.topology[URI]; found {
			remoteAddr := topo.RmNode([]byte(nodeHashKey))
			log.Debugf("Remove backend serve %s, nodeHashKey %s remoteAddr %s.",
				URI, nodeHashKey, remoteAddr)
			// 清掉已经建立的连接
			if remoteAddr != "" {
				if connactor, found := discover.connholder[remoteAddr]; found {
					if connactor != nil {
						connactor.Close()
					}
					delete(discover.connholder, remoteAddr)
				}
			}
			if len(topo.nodeList) == 0 {
				log.Debugf("Remove backend topology:%s", URI)
				delete(discover.topology, URI)
			}
		}
	} else {
		discover.topoLocker.Lock()
		defer discover.topoLocker.Unlock()
		if topo, found := discover.topology[URI]; found {
			log.Infof("Remove all backend serve %s.", URI)
			topo.Clear()
			delete(discover.topology, URI)

			// 清掉已经建立的连接
			for _, remoteAddr := range topo.AllNode() {
				if connactor, found := discover.connholder[remoteAddr]; found {
					if connactor != nil {
						connactor.Close()
					}
					delete(discover.connholder, remoteAddr)
				}
			}
		}
	}
}

// quit unregister
func (discover *Discover) quit() {
	// 停止Etcd Watch
	discover.cancel()
	// 先删除ETCD节点，再关闭连接，不然会出现ETCD节点丢失的情况
	for _, v := range discover.selfURI {
		uri := v + "/provider/instances/" + discover.localListenAddr
		discover.kapi.Delete(context.TODO(), uri)
		log.Debugf("iceberg:%s quit delete etcd key:%s", discover.name, uri)
	}

	discover.kapi.Close()
	discover.connLocker.RLock()
	for k, c := range discover.connholder {
		if c != nil {
			delete(discover.connholder, k)
			c.Close()
		}
	}
	discover.connLocker.RUnlock()
}
