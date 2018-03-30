package frame

import (
	"errors"
)

// 定义外部响应错误类型
var (
	ErrBlocking       = errors.New("通道阻塞")
	ErrClosed         = errors.New("连接关闭")
	ErrTimeout        = errors.New("请求超时")
	ErrMethodNotFound = errors.New("资源不存在")
)
