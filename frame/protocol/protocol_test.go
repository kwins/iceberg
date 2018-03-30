package protocol

import (
	"sync/atomic"
	"testing"

	log "github.com/kwins/iceberg/frame/icelog"

	"github.com/nobugtodebug/go-objectid"
)

func TestProto(t *testing.T) {

	var trid int64
	src := Proto{
		Bizid:       objectid.New().String(),
		RequestID:   atomic.AddInt64(&trid, 1),
		ServeURI:    "sdfa",
		Method:      1,
		ServeMethod: "path",
		Body:        []byte("a=1&b=2&c=3"),
		Err:         nil,
	}

	buf, err := src.Serialize()
	if nil != err {
		return
	}

	var dst Proto
	if err := dst.UnSerialize(buf); err != nil {
		t.Error(err.Error())
	} else {
		t.Log(dst.String())
	}
}
func BenchmarkSeriAndUnSerilize(b *testing.B) {
	for index := 0; index < b.N; index++ {
		var trid int64
		src := Proto{
			Bizid:       objectid.New().String(),
			RequestID:   atomic.AddInt64(&trid, 1),
			ServeURI:    "sdfa",
			Method:      1,
			ServeMethod: "path",
			Body:        []byte("a=1&b=2&c=3"),
			Err:         nil,
		}

		buf, err := src.Serialize()
		if nil != err {
			return
		}

		var dst Proto
		if err := dst.UnSerialize(buf); err != nil {
			b.Error(err.Error())
		}
	}
}

func TestLog(t *testing.T) {
	task := Proto{
		Bizid:       objectid.New().String(),
		RequestID:   1,
		ServeURI:    "sdfa",
		Method:      1,
		ServeMethod: "path",
		Body:        []byte("a=1&b=2&c=3"),
		Err:         nil,
	}
	log.Info(task.PrintableBizID(), " raw request ", task.String(), "123.12.3.43")
}
