package frame

import (
	"os"
	"os/signal"
	"syscall"

	log "github.com/kwins/iceberg/frame/icelog"
)

var (
	defaultServerSignal = &defaultSingal{}
	defaultIgnoreSignal = &ignoreSigal{}
)

// Singal 优雅退出信号
type Singal interface {
	Stop(s os.Signal) bool
}

type defaultSingal struct{}

func (ds *defaultSingal) Stop(s os.Signal) bool {
	log.Warnf("Receive signal %s; Exit now.", s.String())
	return true
}

type ignoreSigal struct{}

func (ig *ignoreSigal) Stop(s os.Signal) bool {
	log.Warnf("Receive signal %s; Exit now.", s.String())
	return true
}

// SignalHandler 信号处理类，管理程序要处理的信号
type SignalHandler struct {
	handlerMap map[os.Signal]Singal
}

// NewSignalHandler 创建信号处理对象
func NewSignalHandler() *SignalHandler {
	sh := new(SignalHandler)
	sh.handlerMap = make(map[os.Signal]Singal)
	// 默认要忽略的信号
	sh.Register(syscall.SIGWINCH, defaultIgnoreSignal)
	sh.Register(syscall.SIGCHLD, defaultIgnoreSignal)
	sh.Register(syscall.SIGCONT, defaultIgnoreSignal)
	sh.Register(syscall.SIGURG, defaultIgnoreSignal)
	sh.Register(syscall.SIGPIPE, defaultIgnoreSignal)
	return sh
}

// Register 注册感兴趣的信号及该信号的回调函数
func (shr *SignalHandler) Register(s os.Signal, h Singal) {
	if _, exist := shr.handlerMap[s]; !exist {
		if h == nil {
			shr.handlerMap[s] = defaultServerSignal
		} else {
			shr.handlerMap[s] = h
		}
	}
}

// UnRegister 解除已注册的信号及回调函数
func (shr *SignalHandler) UnRegister(s os.Signal, h Singal) {
	if _, exist := shr.handlerMap[s]; exist {
		delete(shr.handlerMap, s)
	}
}

// Start 开始对信号的拦截
func (shr *SignalHandler) Start() {
	sc := make(chan os.Signal, 1)
	signal.Notify(sc) // 接收所有信号

	go func() {
		for {
			s := <-sc
			shr.handle(s)
		}
	}()
}

func (shr *SignalHandler) handle(s os.Signal) {
	if _, exist := shr.handlerMap[s]; exist {
		if shr.handlerMap[s].Stop(s) {
			Instance().quit()
			os.Exit(0)
		}
	} else {
		log.Errorf("Not found signal(%s)'s handler, exit.", s.String())
		os.Exit(0)
	}
}

func (shr *SignalHandler) ignore(s os.Signal) (isExit bool) {
	log.Debug("ignore:", s.String())
	return false
}
