package frame

import (
	"testing"

	"github.com/kwins/iceberg/frame/protocol"
	"github.com/nobugtodebug/go-objectid"
	"github.com/opentracing/opentracing-go"
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

	span := opentracing.StartSpan("gateway")

	pp := span.Context().(zipkin.SpanContext)
	println(">>>>>1>>>>>>>>TraceID:", pp.TraceID,
		"SpanID:", pp.SpanID,
		" Sampled:", pp.Sampled,
		" Baggage:", pp.Baggage,
		" ParentSpanID:", pp.ParentSpanID,
		" Flags:", pp.Flags)

	if err := opentracing.GlobalTracer().Inject(span.Context(), opentracing.TextMap, &task); err != nil {
		t.Error(err.Error())
	} else {
		t.Log(task.String())
	}

	println("============================ fake send to server =============================")

	wireContext, err := opentracing.GlobalTracer().Extract(opentracing.TextMap, &task)
	if err != nil {
		t.Error(err.Error())
	}

	spf := opentracing.FollowsFrom(wireContext)
	span2 := opentracing.GlobalTracer().StartSpan("server1", spf)

	span2.SetTag("serverSide", "here")
	pp = span2.Context().(zipkin.SpanContext)
	println(">>>>>2>>>>>>>>TraceID:", pp.TraceID,
		"SpanID:", pp.SpanID,
		" Sampled:", pp.Sampled,
		" Baggage:", pp.Baggage,
		" ParentSpanID:", *pp.ParentSpanID,
		" Flags:", pp.Flags)
}
