package protocol

import (
	"errors"
)

var errNotRawType = errors.New("not raw type")

// Raw 处理原始报文
type Raw interface {
	Get() []byte
	Set([]byte) error
}
