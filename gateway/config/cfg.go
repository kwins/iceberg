package config

import "github.com/kwins/iceberg/frame/config"

// Config 对应配置文件中的格式定义
type Config struct {
	// 认证总开关
	Authorization bool            `json:"authorization"`
	IP            string          `json:"ip"`
	Port          string          `json:"port"`
	Base          config.BaseCfg  `json:"baseCfg"`
	Redis         config.RedisCfg `json:"redisCfg"`
	Mysql         config.MysqlCfg `json:"mysqlCfg"`
}
