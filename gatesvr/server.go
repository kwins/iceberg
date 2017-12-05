package main

import (
	"encoding/json"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/kwins/iceberg/frame"
	log "github.com/kwins/iceberg/frame/icelog"
	"github.com/kwins/iceberg/frame/protocol"
	"github.com/nobugtodebug/go-objectid"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
)

var pingOK = []byte("OK")
var pingpath = "/ping"

var errGatewayTimeout = `{"err_code":504,"err_msg":"Gateway Timeout"}`
var errInternalServerError = `{"err_code":500,"err_msg":"Internal Server Error"}`

// GateWay 是服务层向外统一提供接口的地方。外层的请求都通过GateWay中转
type GateWay struct {
	listenAddr string
}

// NewGateWay 新建GateWay服务
func NewGateWay() *GateWay {
	instanceGateSvr := new(GateWay)
	return instanceGateSvr
}

// ListenAndServe listen and serve
func (gatesvr *GateWay) ListenAndServe(listenAddr string) {
	gatesvr.listenAddr = listenAddr
	lis, err := net.Listen("tcp", listenAddr)
	if err != nil {
		panic(err.Error())
	}
	http.Serve(lis, gatesvr)
}

// ServeHTTP implement http ServeHTTP
// 服务入口，转发到来的所有请求到具体服务
func (gatesvr *GateWay) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	if r.RequestURI == pingpath {
		w.Write(pingOK)
		return
	}

	businessID := r.Header.Get("business_id")
	if businessID == "" {
		businessID = objectid.New().String()
	}

	srvPath, srvMethod, srvRawQuery := splitPathAndMethod(r.RequestURI)

	// 准备Iceberg通用协议
	var task protocol.Proto
	task.Bizid = businessID
	task.ServeURI = srvPath
	task.ServeMethod = srvMethod
	task.RequestID = frame.GetInnerID()
	task.Method = protocol.RestfulMethod(protocol.RestfulMethod_value[strings.ToUpper(r.Method)])

	switch task.Method {
	case protocol.RestfulMethod_GET:
		task.Body = []byte(srvRawQuery)
		task.Format = protocol.RestfulFormat_RAWQUERY
	default:
		body, _ := ioutil.ReadAll(r.Body)
		task.Body = body
		if contentType := r.Header.Get("Content-Type"); strings.Contains(contentType, "json") {
			task.Format = protocol.RestfulFormat_JSON
		} else if strings.Contains(contentType, "xml") {
			task.Format = protocol.RestfulFormat_XML
		} else {
			task.Format = protocol.RestfulFormat_RAWQUERY
		}
	}

	log.Info(task.PrintableBizID(), " raw request ", task.String())

	// tracing start
	if tracer := opentracing.GlobalTracer(); tracer != nil {
		var span opentracing.Span
		wireContext, err := tracer.Extract(
			opentracing.TextMap,
			opentracing.HTTPHeadersCarrier(r.Header),
		)
		if err != nil {
			span = opentracing.StartSpan("ServeHTTP")
		} else {
			span = tracer.StartSpan("ServeHTTP", ext.RPCServerOption(wireContext))
		}
		if err := tracer.Inject(span.Context(),
			opentracing.TextMap, &task); err != nil {
			log.Error(err.Error())
		}
		defer span.Finish()
	}
	// 转发到具体服务
	resp, err := frame.DeliverTo(&task)
	if err != nil {
		if err == frame.ErrTimeout {
			http.Error(w, errGatewayTimeout, http.StatusGatewayTimeout)
		} else {
			b, _ := json.Marshal(&protocol.ErrInfo{ErrCode: http.StatusInternalServerError, ErrInfo: err.Error()})
			http.Error(w, string(b), http.StatusInternalServerError)
		}
	} else {
		log.Info(task.PrintableBizID(), " raw response ", resp.String())
		if len(resp.GetBody()) > 0 {
			w.Write(resp.GetBody())
		} else if len(resp.GetErr()) > 0 {
			http.Error(w, string(resp.Err), http.StatusInternalServerError)
		} else {
			http.Error(w, errInternalServerError, http.StatusInternalServerError)
		}
	}
}

func splitPathAndMethod(requestURI string) (string, string, string) {
	reqURI, err := url.ParseRequestURI(requestURI)
	if err != nil {
		return "", "", ""
	}
	if index := strings.LastIndexByte(reqURI.Path, '/'); index != -1 && index != len(reqURI.Path) {
		return reqURI.Path[:index], reqURI.Path[index+1:], reqURI.RawQuery
	}
	return "", "", ""
}
