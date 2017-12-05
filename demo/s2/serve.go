package main

import (
	"context"

	hello "github.com/kwins/iceberg/demo/s1/pb"
	hi "github.com/kwins/iceberg/demo/s2/pb"
	log "github.com/kwins/iceberg/frame/icelog"
)

// Hi 对象
type Hi struct {
}

// SayHi handel message 01
func (id *Hi) SayHi(ctx context.Context, in *hi.HiRequest) (*hi.HiResponse, error) {
	// 开启zipkin
	// span := opentracing.SpanFromContext(ctx)
	// span.SetTag("SayHi-foo", "bar")
	// span.SetTag("SayHi-time", time.Now().Format(frame.Normalformat))

	var res hi.HiResponse

	var foo hello.HelloRequest
	foo.Name = "quinn-wang"
	resp, err := hello.SayHello(ctx, &foo)
	if err != nil {
		res.Message = "hi say hello to hello fail!!"
	} else {
		res.Message = resp.Message
	}

	log.Info("SayHi call SayHello ....", ctx.Value("bizid"))
	return &res, nil
}
