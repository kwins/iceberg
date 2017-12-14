package frame

import (
	"context"

	"github.com/kwins/iceberg/frame/protocol"
	"github.com/opentracing/opentracing-go"
)

// SpanWithTask 讲Span注入Task
func SpanWithTask(ctx context.Context, task *protocol.Proto) opentracing.Span {
	if tracer := opentracing.GlobalTracer(); tracer != nil {
		var span opentracing.Span
		parentSpan := opentracing.SpanFromContext(ctx)
		if parentSpan != nil {
			spf := opentracing.ChildOf(parentSpan.Context())
			span = tracer.StartSpan(task.GetServeMethod(), spf)
		} else {
			span = tracer.StartSpan(task.GetServeMethod())
		}
		tracer.Inject(span.Context(), opentracing.TextMap, task)
		return span
	}
	return nil
}

// SpanFromTask 从Task中加工出Span
func SpanFromTask(task *protocol.Proto) opentracing.Span {
	if tracer := opentracing.GlobalTracer(); tracer != nil {
		var span opentracing.Span
		spanCtx, err := tracer.Extract(opentracing.TextMap, task)
		if err != nil {
			span = tracer.StartSpan(task.GetServeMethod())
		} else {
			spf := opentracing.FollowsFrom(spanCtx)
			span = tracer.StartSpan(task.GetServeMethod(), spf)
		}
		return span
	}
	return nil
}
