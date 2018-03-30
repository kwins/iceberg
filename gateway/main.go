package main

import (
	"flag"
	_ "net/http/pprof"
	"os"
	"path/filepath"
	"syscall"

	"github.com/kwins/iceberg/frame"
	"github.com/kwins/iceberg/frame/config"
	gwcfg "github.com/kwins/iceberg/gateway/config"
	"github.com/kwins/iceberg/gateway/serve"
)

var (
	cfgFile  = flag.String("config-path", "gw.json", "config file")
	logLevel = flag.String("level", "debug", "log level")
	logPath  = flag.String("log-path", "", "log path")
)

func main() {
	flag.Parse()
	// 设置进程的当前目录为程序所在的路径
	dir, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	os.Chdir(dir)

	// 读取并解析配置文件
	var cfg gwcfg.Config
	config.Parseconfig(*cfgFile, &cfg)

	s := serve.NewGateway(cfg)
	sh := frame.NewSignalHandler()
	sh.Register(syscall.SIGTERM, s)
	sh.Register(syscall.SIGQUIT, s)
	sh.Register(syscall.SIGINT, s)
	sh.Start()
	s.ListenAndServe()
}
