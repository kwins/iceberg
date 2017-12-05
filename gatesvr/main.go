package main

import (
	"flag"
	"os"
	"path/filepath"

	"github.com/kwins/iceberg/frame"
	"github.com/kwins/iceberg/frame/config"
)

// Config 对应配置文件中的格式定义
type Config struct {
	IP   string         `json:"ip"`
	Port string         `json:"port"`
	Base config.BaseCfg `json:"baseCfg"`
}

var (
	cfgFile  = flag.String("config-path", "gatesvr_conf.json", "config file")
	logLevel = flag.String("level", "debug", "log level")
	logPath  = flag.String("log-path", "", "log path")
)

func main() {
	flag.Parse()
	// 设置进程的当前目录为程序所在的路径
	dir, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	os.Chdir(dir)

	// 读取并解析配置文件
	var cfg Config
	config.Parseconfig(*cfgFile, &cfg)

	var localListenAddr string
	if cfg.IP == "" {
		localListenAddr = frame.Netip() + ":" + cfg.Port
	} else {
		localListenAddr = cfg.IP + ":" + cfg.Port
	}

	s := NewGateWay()
	// 开启服务发现机制
	frame.DiscoverInstance().Start("gatesvr", &cfg.Base, []string{"/services"}, localListenAddr)
	s.ListenAndServe(localListenAddr)
}

func gracefulExit(s os.Signal) (isExit bool) {
	return true
}

func ignoreSignal(s os.Signal) (isExit bool) {
	return false
}
