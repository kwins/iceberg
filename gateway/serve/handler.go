package serve

import (
	"encoding/json"
	"net/http"

	"github.com/kwins/iceberg/frame"
	log "github.com/kwins/iceberg/frame/icelog"
)

// HandlePing LBS ping
func HandlePing(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("success"))
}

// HandleStatics 接口访问统计
func HandleStatics(w http.ResponseWriter, r *http.Request) {
	mt := frame.MeTables()
	if b, err := json.Marshal(mt); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else {
		w.Write(b)
	}
}

// HandleNotFound http 404
func HandleNotFound(w http.ResponseWriter, r *http.Request) {
	log.Infof("not found url:%s ip:%s", r.URL.Path, r.RemoteAddr)
	http.Error(w, errNotFounHTTPMethod, http.StatusNotFound)
}

// HandleIceberg iceberg 服务入口
func (gw *Gateway) HandleIceberg(w http.ResponseWriter, r *http.Request) {

	if task, err := resolveRequest(r); err != nil {
		log.Error(err.Error())
		http.Error(w, errRequestInvalide, http.StatusBadRequest)

	} else {
		log.Info(task.AsString())
		// 转发到具体服务
		resp, err := frame.DeliverTo(task)
		if err != nil {
			log.Warn(err.Error())
			if err == frame.ErrTimeout {
				http.Error(w, errGatewayTimeout, http.StatusGatewayTimeout)
			} else {
				http.Error(w, errInternalError, http.StatusInternalServerError)
			}
		} else {
			log.Info(resp.AsString())
			if len(resp.GetBody()) > 0 {
				for k, v := range resp.GetHeader() {
					w.Header().Set(k, v)
				}
				w.Write(resp.GetBody())
			} else if len(resp.GetErr()) > 0 {
				http.Error(w, string(resp.Err), http.StatusInternalServerError)
			} else {
				http.Error(w, errInternalError, http.StatusInternalServerError)
			}
		}
	}
}
