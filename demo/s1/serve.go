package main

import (
	"os"
	"time"

	hello "github.com/kwins/iceberg/demo/s1/pb"
	"github.com/kwins/iceberg/frame"
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
func (id *Hello) SayHello(c frame.Context) error {
	var res hello.HelloResponse
	res.Message = "welcome~~~"
	log.Info("SayHello receiver....", c.Bizid(), c.Header().Get("A"))
	return c.JSON(&res)
}

// GetExample HTTP GET With Query
func (id *Hello) GetExample(c frame.Context) error {
	log.Info("GetExample receiver....", c.Bizid(), " name=", c.FormValue("name"), " age=", c.FormValue("age"))
	return c.JSON2(0, "success", &hello.HelloResponse{Message: "GetExample hi~~~"})
}

// PostExample HTTP Post
func (id *Hello) PostExample(c frame.Context) error {
	log.Info("PostExample receiver....", c.Bizid())
	var req hello.HelloRequest
	if err := c.Bind(&req); err != nil {
		return err
	}
	log.Info("PostExample receiver....", req.String())
	return c.JSON(&hello.HelloResponse{Message: "PostExample hi~~~"})
}

// PostFormExample HTTP POST From
func (id *Hello) PostFormExample(c frame.Context) error {
	if c.GetString("B", "") != "B" {
		return c.JSON2(1, "params not illegal", nil)
	}
	c.Info("GetString:", c.GetString("B", "not get value B"))
	c.Info("GetInt:", c.GetInt("C", -1))
	c.Info("GetUint:", c.GetUint("D", uint64(0)))
	c.Info("GetFloat:", c.GetFloat("E", float64(0.0)))

	log.Info("PostFormExample receiver....", c.Bizid(), c.FormValues())
	return c.JSON2(0, "success", &hello.HelloResponse{Message: "PostFormExample hi~~~"})
}

// Timeout 超时GC测试
func (id *Hello) Timeout(c frame.Context) error {
	c.Info("receiver time out request....")
	time.Sleep(time.Second * 30)
	return c.String("success")
}

// Stop Stop
func (id *Hello) Stop(s os.Signal) bool {
	log.Infof("hi graceful exit.")
	return true
}
