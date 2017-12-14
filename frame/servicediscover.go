package frame

import (
	"context"
	"errors"
	"net"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/coreos/etcd/clientv3"
	"github.com/kwins/iceberg/frame/config"
	log "github.com/kwins/iceberg/frame/icelog"
	"github.com/kwins/iceberg/frame/protocol"
	"github.com/opentracing/opentracing-go"
	zipkin "github.com/openzipkin/zipkin-go-opentracing"
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
	errNotFoundMethod       = errors.New("Not found Method!")
	errForgetEtcdCfg        = errors.New("forget set etcd config or not start etcd server?")
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

	// server describe
	mdLocker sync.RWMutex // method
	md       map[string]*MethodDesc

	localListenAddr string

	innerid int64 // 内部请求ID
}

var discoverOnce sync.Once
var discoverInstance *Discover

// DiscoverInstance 返回GateSvr的单例对象
func DiscoverInstance() *Discover {
	discoverOnce.Do(func() {
		discoverInstance = new(Discover)
		discoverInstance.md = make(map[string]*MethodDesc)
		discoverInstance.topology = make(map[string]*ConsistentHash)
		discoverInstance.connholder = make(map[string]*ConnActor)
	})
	return discoverInstance
}

// RegisterAndServe 后端服务注册并开启
func RegisterAndServe(sd *ServiceDesc, ss interface{}, cfg *config.BaseCfg) {
	ht := reflect.TypeOf(sd.HandlerType).Elem()
	st := reflect.TypeOf(ss)
	if !st.Implements(ht) {
		log.Fatalf("iceberg: RegisterAndServe found the handler of type %v that does not satisfy %v", st, ht)
		return
	}
	s := DiscoverInstance()

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
		ca.id = atomic.AddUint32(&connActorID, CA_BROKEN)
		ca.connType = passiveConnActor
		ca.initConnActor(c)
	}
}

// GetInnerID 获取内部服务ID
func GetInnerID() int64 {
	return atomic.AddInt64(&DiscoverInstance().innerid, 1)
}

// DeliverTo deliver request to anthor serve
func DeliverTo(task *protocol.Proto) (*protocol.Proto, error) {
	conn, err := DiscoverInstance().DrectDispatch(task.GetServeURI())
	if err != nil {
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

	if err := discover.StartZipkinTrace(cfg.Zipkin.EndPoints,
		discover.localListenAddr, discover.name); err != nil {
		log.Error(err.Error())
	}

	go discover.discover()

	log.Infof("%s start up,local listen addr:%s,serve uri:%v", discover.name, discover.localListenAddr, discover.selfURI)
}

// StartZipkinTrace 启动zipkin
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

// DrectDispatch 负载均衡的进行分发消息
// 分发的时候会进行负载均衡的处理
// URI 请求的接口路径
// 匹配节点时会进行完全匹配
func (discover *Discover) DrectDispatch(URI string) (*ConnActor, error) {
	discover.topoLocker.RLock()
	if node, ok := discover.topology[URI]; ok {
		discover.topoLocker.RUnlock()

		return discover.getConnActor(node.Leastload(), URI)
	}
	discover.topoLocker.RUnlock()
	return nil, errConnectURIIsNil
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
		c, err := net.Dial("tcp", remoteAddr)
		if err != nil {
			return err
		}
		log.Debugf("init connect [%s] -> [%s] backend serve.", remoteAddr, uri)
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
		svrURI := uri + "/provider/name"
		_, err := discover.kapi.Put(context.TODO(), svrURI, discover.name)
		if err != nil {
			return err
		}
		log.Debugf("set %s=%s", svrURI, discover.name)
		svrURI = uri + "/provider/instances/" + discover.localListenAddr
		resp, err := discover.kapi.Grant(context.TODO(), 0)
		if err != nil {
			return err
		}
		_, err = discover.kapi.KeepAlive(context.TODO(), resp.ID)
		if err != nil {
			return err
		}
		log.Debugf("set %s=%s", svrURI, discover.localListenAddr)
		_, err = discover.kapi.Put(context.TODO(), svrURI, discover.localListenAddr, clientv3.WithLease(resp.ID))
		if err != nil {
			return err
		}
		// 主动获取本服务节点状态，防止意外情况
		go func(k, v string, leaseid clientv3.LeaseID) {
			for {
				_, err = discover.kapi.Get(context.TODO(), k)
				if err != nil {
					log.Fatalf("iceberg:%s svr uri %s get fail,detail=%s", discover.name, k, err.Error())
					discover.kapi.Put(context.TODO(), k, v, clientv3.WithLease(resp.ID))
				}
				select {
				case <-time.After(time.Second * 2):
				}
			}
		}(svrURI, discover.localListenAddr, resp.ID)
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
	if len(cfg.EndPoints) == 0 {
		return errForgetEtcdCfg
	}
	api, err := clientv3.New(clientv3.Config{
		Endpoints:   cfg.EndPoints,
		Username:    cfg.User,
		Password:    cfg.Psw,
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
		discover.setTopo(string(subNode.Key), string(subNode.Value))
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
				switch event.Type {
				case clientv3.EventTypePut:
					discover.setTopo(key, value)
				case clientv3.EventTypeDelete:
					discover.rmTopo(key, value)
				}
			}
		}
	}
}

func (discover *Discover) setTopo(key, value string) {
	segment := strings.Split(string(key), "/")
	if l := len(segment); l < 3 {
		return
	}
	if leafname := segment[len(segment)-1]; leafname == "config" {

	} else if leafname == "name" {

	} else if segment[len(segment)-2] == "instances" {
		interfaceURI := strings.Join(segment[:len(segment)-3], "/")
		discover.regist(interfaceURI, value)
	}
}

func (discover *Discover) rmTopo(key, value string) {
	segment := strings.Split(string(key), "/")
	if l := len(segment); l < 3 {
		return
	}
	if leafname := segment[len(segment)-1]; leafname == "config" {
		// TO DO
	} else if leafname == "name" {
		// TO DO
	} else if segment[len(segment)-2] == "instances" {
		interfaceURI := strings.Join(segment[:len(segment)-3], "/")
		discover.unRegist(interfaceURI, segment[len(segment)-1])
	}
}

// 注册一个后台服务接口
func (discover *Discover) regist(URI string, svrAddr string) {
	if len(URI) == 0 {
		return
	}

	// 过滤掉监听到自己的状态变化产生的通知
	if discover.localListenAddr == svrAddr {
		return
	}

	discover.topoLocker.Lock()
	defer discover.topoLocker.Unlock()
	var topo *ConsistentHash
	var found bool

	if topo, found = discover.topology[URI]; !found {
		topo = NewConsistentHash()
		discover.topology[URI] = topo
		log.Infof("Regist a new service at direction %s, the addr is %s", URI, svrAddr)
	}

	// 用后台服务的地址作为key来生成hash节点
	log.Infof("AddNode: %s svrAddr:%s", URI, svrAddr)
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
			log.Infof("Remove backend serve %s, nodeHashKey %s remoteAddr %s.", URI, nodeHashKey, remoteAddr)
			// 清掉已经建立的连接
			if remoteAddr != "" {
				if connactor, found := discover.connholder[remoteAddr]; found {
					if connactor != nil {
						connactor.Close()
					}
					delete(discover.connholder, remoteAddr)
				}
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

// Quit Quit
func (discover *Discover) Quit() {
	discover.connLocker.Lock()
	defer discover.connLocker.Unlock()
	for k, c := range discover.connholder {
		if c != nil {
			delete(discover.connholder, k)
			c.Close()
		}
	}
}
