package main

import (
	"context"

	hello "github.com/kwins/iceberg/demo/s1/pb"
	"github.com/kwins/iceberg/frame/config"
	log "github.com/kwins/iceberg/frame/icelog"
)

// Config 配置
type Config struct {
	IP   string         `json:"ip"`
	Etcd config.EtcdCfg `json:"etcdCfg"`
	Port string         `json:"prot"`
}

// Hello 对象
type Hello struct {
	transmitid int64
}

// SayHello handel message 01
func (id *Hello) SayHello(ctx context.Context, in *hello.HelloRequest) (*hello.HelloResponse, error) {
	// 开启zipkin 可以使用下面
	// span := opentracing.SpanFromContext(ctx)
	// span.SetTag("SayHello-foo", "bar")
	// span.SetTag("SayHello-time", time.Now().Format(frame.Normalformat))
	var res hello.HelloResponse
	res.Message = "welcome~~~"
	log.Info("SayHello receiver....", ctx.Value("bizid"))
	return &res, nil
}
