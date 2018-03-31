package serve

import (
	"net"
	"net/http"
	"os"

	gcfg "github.com/kwins/iceberg/gateway/config"

	"github.com/kwins/iceberg/frame"
	log "github.com/kwins/iceberg/frame/icelog"
)

var root = "/services"

// Gateway 网关服务
type Gateway struct {
	cfg        gcfg.Config
	listenAddr string
	rt         *Router
}

// NewGateway 网关
func NewGateway(cfg gcfg.Config) *Gateway {
	gw := new(Gateway)
	gw.cfg = cfg

	gw.listenAddr = frame.Netip() + ":" + gw.cfg.Port
	frame.Instance().Start("Gateway", &gw.cfg.Base, []string{root}, gw.listenAddr)

	gw.rt = NewRouter(gw.HandleIceberg, HandleNotFound)
	gw.rt.Add("/ping", HandlePing)
	gw.rt.Add("/statistics", HandleStatics)

	log.Debugf("gateway init with cfg=%v", gw.cfg)
	return gw
}

// ListenAndServe listen and serve
func (gw *Gateway) ListenAndServe() {
	l, err := net.Listen("tcp", gw.listenAddr)
	if err != nil {
		panic(err.Error())
	}
	http.Serve(l, gw)
}

// Stop Stop
func (gw *Gateway) Stop(s os.Signal) bool {
	log.Infof("gateway graceful exit.")
	return true
}

// ServeHTTP implement http ServeHTTP
// 服务入口，转发到来的所有请求到具体服务
func (gw *Gateway) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	gw.rt.Hanlder(r.URL.Path)(w, r)
}
