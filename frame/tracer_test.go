package frame

import (
	"context"
	"testing"

	"github.com/kwins/iceberg/frame/protocol"
	"github.com/nobugtodebug/go-objectid"
	zipkin "github.com/openzipkin/zipkin-go-opentracing"
)

func TestTracer(t *testing.T) {

	s := DiscoverInstance()
	s.StartZipkinTrace("http://localhost:94111", "127.0.0.1", "gateway")

	var task protocol.Proto
	task.Bizid = objectid.New().String()
	task.Body = []byte("test")
	task.Format = protocol.RestfulFormat_RAWQUERY
	task.Method = protocol.RestfulMethod_POST
	task.RequestID = 1
	task.ServeMethod = "Test"
	task.ServeURI = "/services/test"

	span := SpanWithTask(context.TODO(), &task)
	pp := span.Context().(zipkin.SpanContext)

	span1 := SpanFromTask(&task)
	span1.SetTag("serverSide", "here")

	pp1 := span1.Context().(zipkin.SpanContext)

	if *pp1.ParentSpanID != pp.SpanID {
		t.Errorf("%v:%v", pp1.ParentSpanID, pp.SpanID)
	}
}
