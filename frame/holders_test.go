package frame

import (
	"testing"
)

func BenchmarkPutRequest(b *testing.B) {
	d := NewDispatcher()

	for i := 0; i < b.N; i++ {
		d.Put(int64(i))
	}
}

// var req = make(map[int]chan *protocol.Proto)

// func BenchmarkPutRequest(b *testing.B) {

// 	for i := 0; i < b.N; i++ {
// 		var ch = make(chan *protocol.Proto)
// 		req[i] = ch
// 	}
// }

// func TestProtoList(t *testing.T) {
// 	l := newProtoList()

// 	for index := 0; index < 20; index++ {
// 		var rid requestID
// 		rid.value = int64(index)
// 		rid.pre = nil
// 		rid.next = nil
// 		l.pushFront(&rid)
// 	}
// 	for e := l.next; e != nil; e = e.next {
// 		fmt.Println(e.value)
// 	}
// }
