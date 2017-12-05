package frame

import (
	"context"

	"github.com/kwins/iceberg/frame/protocol"
	objectid "github.com/nobugtodebug/go-objectid"
)

type methodHandler func(srv interface{}, ctx context.Context, fromat protocol.RestfulFormat, in []byte) ([]byte, error)

// MethodDesc represents an RPC service's method specification.
type MethodDesc struct {
	MethodName string
	Handler    methodHandler
}

// ServiceDesc 服务描述
type ServiceDesc struct {
	Version     string
	ServiceName string
	// The pointer to the service interface. Used to check whether the user
	// provided implementation satisfies the interface requirements.
	HandlerType interface{}
	Methods     []MethodDesc
	Metadata    interface{}
	ServiceURI  []string
}

// ReadyTask 准备请求的任务
func ReadyTask(ctx context.Context, srvMethod string, srvName string, in interface{}) (*protocol.Proto, error) {
	var task protocol.Proto
	if bizid, ok := ctx.Value("bizid").(string); !ok {
		task.Bizid = objectid.New().String()
	} else {
		task.Bizid = bizid
	}

	format, ok := ctx.Value("format").(protocol.RestfulFormat)
	if !ok {
		format = protocol.RestfulFormat_JSON
	}
	b, err := protocol.Pack(format, in)
	if err != nil {
		return nil, err
	}
	task.Body = b
	task.ServeMethod = srvMethod
	task.Format = format
	task.RequestID = GetInnerID()
	task.ServeURI = "/services/v1/" + srvName
	task.Method = protocol.RestfulMethod_POST
	return &task, nil
}
