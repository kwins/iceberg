package frame

import (
	"testing"
	"time"
)

// func BenchmarkPutRequest(b *testing.B) {
// 	var h = NewHolder()
// 	for i := 0; i < b.N; i++ {
// 		h.Put(int64(i), make(chan *protocol.Proto))
// 	}
// }

func TestShapeTime(t *testing.T) {
	ShapingTime(time.Now(), 10, 2)
	time.Sleep(time.Millisecond * 100)
	ShapingTime(time.Now(), 10, 2)
	time.Sleep(time.Millisecond * 100)
	ShapingTime(time.Now(), 10, 2)
	time.Sleep(time.Millisecond * 100)
	ShapingTime(time.Now(), 10, 2)
}
