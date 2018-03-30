package frame

import (
	"github.com/kwins/iceberg/frame/protocol"
	"github.com/kwins/iceberg/frame/util"

	objectid "github.com/nobugtodebug/go-objectid"
	"github.com/opentracing/opentracing-go"
)

type methodHandler func(srv interface{}, ctx Context) error

// MethodDesc represents an RPC service's method specification.
type MethodDesc struct {
	// 是否允许外部无认证访问
	Allowed string

	// 方法名称
	MethodName string

	// 调起方法的句柄
	Handler methodHandler
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
func ReadyTask(fc Context,
	srvMethod, srvName, srvVersion string,
	in interface{}, opts ...CallOption) (*protocol.Proto, error) {

	c := defaultCallInfo()
	for _, o := range opts {
		if err := o.before(c); err != nil {
			return nil, err
		}
	}
	var task protocol.Proto
	if fc.Bizid() == "" {
		task.Bizid = objectid.New().String()
	} else {
		task.Bizid = fc.Bizid()
	}
	task.ServeMethod = srvMethod
	task.Format = c.format
	task.Header = make(map[string]string)
	for k := range c.header {
		task.Header[k] = c.header.Get(k)
	}
	task.Form = make(map[string]string)
	for k, v := range c.form {
		task.Form[k] = v
	}
	// 默认序列化方式为JSON
	if task.Format == protocol.RestfulFormat_FORMATNULL {
		task.Format = protocol.RestfulFormat_JSON
	}
	task.RequestID = GetInnerID()
	task.ServeURI = "/services/" + srvVersion + "/" + srvName
	task.Method = protocol.RestfulMethod_POST
	b, err := protocol.Pack(task.Format, in)
	if err != nil {
		return nil, err
	}
	// inject(fc, &task)
	task.Body = b
	return &task, nil
}

func inject(c Context, r *protocol.Proto) {
	if c.Request() != nil {
		r.TraceMap = c.Request().GetTraceMap()
	} else if tracer := opentracing.GlobalTracer(); tracer != nil {
		span := tracer.StartSpan(r.GetServeMethod())
		span.SetTag("Bizid", r.GetBizid())
		span.SetTag("Host name", util.GetHostname())
		tracer.Inject(span.Context(), opentracing.TextMap, r)
	}
}
