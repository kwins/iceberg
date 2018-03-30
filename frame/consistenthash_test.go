package frame

import (
	"fmt"
	"sync"
	"testing"
)

var chash = NewConsistentHash()
var wg sync.WaitGroup

func init() {
	for index := 0; index < 2; index++ {
		chash.AddNode("127.0.0.1:" + fmt.Sprint(index))
	}
}

func BenchmarkLeastNode(b *testing.B) {
	for k := 0; k < 100; k++ {
		go func() {
			wg.Add(1)
			for i := 0; i < b.N; i++ {
				v := chash.Leastload()
				if v == "" {
					b.Error("v null")
					return
				}
			}
			wg.Done()
		}()
	}
	wg.Wait()
}

func BenchmarkHash(b *testing.B) {
	for index := 0; index < b.N; index++ {
		_hash([]byte{byte(index)})
	}
}
