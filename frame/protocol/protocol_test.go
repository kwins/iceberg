package protocol

import (
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/nobugtodebug/go-objectid"
)

func Test_Protocol(t *testing.T) {

	var trid int64
	task := Proto{
		Bizid:       objectid.New().String(),
		RequestID:   atomic.AddInt64(&trid, 1),
		ServeURI:    "sdfa",
		Method:      0,
		ServeMethod: "path",
		Body:        []byte("a=1&b=2&c=3"),
		Err:         nil,
	}

	buf, err := task.Serialize()
	if nil != err {
		return
	}

	var task1 Proto
	task1.UnSerialize(buf)
	t.Log(string(task1.GetBody()))
	t.Log(task1.String())
}

func TestPrintbaleBizid(t *testing.T) {
	t.Log(PrintableBizID("xxxxxxxxxxx"))
}

type NotifyResponse struct {
	Content []byte `protobuf:"bytes,1,opt,name=content" json:"content,omitempty" xml:"content,omitempty"`
}

func TestMarsharl(t *testing.T) {
	var v NotifyResponse
	v.Content = []byte("<a></a><b>a=b&v=d</b>")
	t.Log(string(marshal(&v)))
}

func TestStuctTag(t *testing.T) {
	var v NotifyResponse
	v.Content = []byte("<a></a><b>a=b&v=d</b>")
	ve := reflect.ValueOf(&v)
	typ := reflect.Indirect(ve).Type()
	t.Log(typ.Field(0).Tag)
}
