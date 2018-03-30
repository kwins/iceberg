package serve

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/kwins/iceberg/frame"
	"github.com/kwins/iceberg/frame/protocol"

	"github.com/nobugtodebug/go-objectid"
)

var defaultMemory = int64(32 >> 22)

func resolveRequest(r *http.Request) (*protocol.Proto, error) {
	// 准备Iceberg通用协议
	var task protocol.Proto
	businessID := r.Header.Get("bizid")
	if businessID == "" {
		businessID = objectid.New().String()
	}

	path := r.URL.Path
	// Tracer ID
	task.Bizid = businessID

	if i := strings.LastIndexByte(path, '/'); i == -1 {
		return nil, fmt.Errorf("error path:%s", path)
	} else {
		// URI
		task.ServeURI = path[:i]

		// Serve Method
		task.ServeMethod = strings.ToLower(path[i+1:])
	}

	// Inner ID
	task.RequestID = frame.GetInnerID()

	// HTTP Method
	n := protocol.RestfulMethod_value[strings.ToUpper(r.Method)]
	task.Method = protocol.RestfulMethod(n)
	task.RemoteAddr = r.RemoteAddr

	// 解析Header信息
	task.Header = make(map[string]string)
	for k := range r.Header {
		task.Header[k] = r.Header.Get(k)
	}

	// URL RAW Query data
	task.Form = make(map[string]string)
	q := r.URL.Query()
	for k := range q {
		task.Form[k] = q.Get(k)
	}
	// 解析Form，Body信息
	contentType := r.Header.Get(protocol.HeaderContentType)
	if strings.HasPrefix(contentType, protocol.MIMEApplicationJSON) {
		task.Format = protocol.RestfulFormat_JSON

	} else if strings.HasPrefix(contentType, protocol.MIMETextXML) ||
		strings.HasPrefix(contentType, protocol.MIMEApplicationXML) {
		task.Format = protocol.RestfulFormat_XML

	} else if strings.HasPrefix(contentType, protocol.MIMEApplicationProtobuf) {
		task.Format = protocol.RestfulFormat_PROTOBUF

	} else {
		if strings.HasPrefix(contentType, protocol.MIMEMultipartForm) {
			if err := r.ParseMultipartForm(defaultMemory); err != nil {
				return nil, err
			}
		} else if strings.HasPrefix(contentType, protocol.MIMEApplicationForm) {
			if err := r.ParseForm(); err != nil {
				return nil, err
			}
		}

		if len(r.Form) > 0 {
			for k := range r.Form {
				task.Form[k] = r.Form.Get(k)
			}
			task.Format = protocol.RestfulFormat_RAWQUERY
		} else {
			task.Format = protocol.RestfulFormat_FORMATNULL
		}
	}

	body, _ := ioutil.ReadAll(r.Body)
	task.Body = body

	return &task, nil
}
