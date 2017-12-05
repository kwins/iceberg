package config

import (
	"encoding/json"
	"os"
	"time"
)

// MysqlCfg mysql config
type MysqlCfg struct {
	Host struct {
		Read  string `json:"read"`
		Write string `json:"write"`
	} `json:"Host"`
	Port   int    `json:"Port"`
	User   string `json:"User"`
	Psw    string `json:"Psw"`
	DbName string `json:"DbName"`
}

// RedisCfg config
type RedisCfg struct {
	Addr string `json:"Addr"`
	Psw  string `json:"Psw"`
	DBNo int    `json:"DBNo"`
}

// BaseCfg 服务基础配置
type BaseCfg struct {
	Etcd   EtcdCfg   `json:"etcdCfg"`
	Zipkin ZipkinCfg `json:"zipkinCfg"`
	Staff  StaffCfg  `json:"staffCfg"`
}

// ZipkinCfg Zipkin配置
type ZipkinCfg struct {
	EndPoints string `json:"endpoints"`
}

// StaffCfg 服务监控人员
type StaffCfg struct {
	Name        string `json:"name" yaml:"name"`     // 服务名称
	Email       string `json:"email" yaml:"email"`   // 通知邮箱
	MobilePhone string `json:"mobile" yaml:"mobile"` // 通知手机号 ，暂未实现
}

// EtcdCfg 对应配置文件中关于etcd配置内容
type EtcdCfg struct {
	EndPoints []string      `json:"endpoints" yaml:"endpoints"`
	User      string        `json:"user" yaml:"user"`
	Psw       string        `json:"psw" yaml:"psw"`
	Timeout   time.Duration `json:"timeout" yaml:"timeout"`
}

// Parseconfig parse json config
// out must be pointer
func Parseconfig(filepath string, out interface{}) {
	file, err := os.Open(filepath)
	if err != nil {
		panic(err)
	}
	decoder := json.NewDecoder(file)
	err = decoder.Decode(out)
	if err != nil {
		panic(err)
	}
}
