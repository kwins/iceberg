package frame

import (
	"context"
	"net/http"

	"github.com/kwins/iceberg/frame/protocol"
	"github.com/kwins/iceberg/frame/util"

	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/common/log"
)

// SpanWithTask 将Span注入Task
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
		span.SetTag("Bizid", task.GetBizid())
		span.SetTag("Host name", util.GetHostname())
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
		span.SetTag("Bizid", task.GetBizid())
		span.SetTag("Host name", util.GetHostname())
		return span
	}
	return nil
}

// SpanFromHTTPHeader 从HTTP请求中Extract Tracer信息 然后 Inject to Task
func SpanFromHTTPHeader(header http.Header, task *protocol.Proto) opentracing.Span {
	if tracer := opentracing.GlobalTracer(); tracer != nil {
		var span opentracing.Span
		wireContext, err := tracer.Extract(
			opentracing.TextMap,
			opentracing.HTTPHeadersCarrier(header),
		)
		if err != nil {
			span = opentracing.StartSpan("server-http")
		} else {
			spf := opentracing.FollowsFrom(wireContext)
			span = tracer.StartSpan("serve-http", spf)
		}
		span.SetTag("Bizid", task.GetBizid())
		span.SetTag("Host name", util.GetHostname())
		if err := tracer.Inject(span.Context(), opentracing.TextMap, task); err != nil {
			log.Error(err.Error())
			return nil
		}
		return span
	}
	return nil
}

// SpanFromContext 从Context中获取Span，然后Inject Request Header
func SpanFromContext(ctx context.Context, h http.Header) opentracing.Span {
	if tracer := opentracing.GlobalTracer(); tracer != nil {
		var span opentracing.Span
		if span = opentracing.SpanFromContext(ctx); span == nil {
			span = opentracing.StartSpan("run")
		}
		if err := tracer.Inject(span.Context(), opentracing.TextMap, h); err != nil {
			return nil
		}
		return span
	}
	return nil
}
