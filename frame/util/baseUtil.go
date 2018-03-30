package util

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"fmt"
	"os"
	"time"

	"github.com/kwins/iceberg/frame/config"

	"github.com/garyburd/redigo/redis"
)

// MD5 md5
func MD5(data []byte) string {
	h := md5.New()
	h.Write(data)
	return fmt.Sprintf("%x", h.Sum(nil))
}

// Sha1WithByte Sha1WithByte
func Sha1WithByte(data []byte) []byte {
	h := sha1.New()
	h.Write(data)
	return h.Sum(nil)
}

// Sha256WithByte Sha256
func Sha256WithByte(data []byte) []byte {
	h := sha256.New()
	h.Write(data)
	return h.Sum(nil)
}

// Sha256WithString Sha256
func Sha256WithString(data string) []byte {
	h := sha256.New()
	h.Write([]byte(data))
	return h.Sum(nil)
}

// NewRedisPool new redis pool
func NewRedisPool(cfg *config.RedisCfg) *redis.Pool {
	return &redis.Pool{
		MaxIdle:     3,
		IdleTimeout: 300 * time.Second,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", cfg.Addr)
			if err != nil {
				return nil, err
			}
			if len(cfg.Psw) > 0 {
				if _, err := c.Do("AUTH", cfg.Psw); err != nil {
					c.Close()
					return nil, err
				}
			}
			return c, err
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
	}
}

var hostName string

// GetHostname 获取服务器主机名称
func GetHostname() string {
	if hostName == "" {
		var err error
		hostName, err = os.Hostname()
		if err != nil {
			hostName = "unknow"
		}
		return hostName
	}
	return hostName
}
