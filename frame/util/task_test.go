package util

import (
	"sync"
)

var wg sync.WaitGroup

// func TestTasks(t *testing.T) {
// 	var ts = new(tasks)

// 	var t1 = new(Task)
// 	t1.seqid = 1
// 	ts.pushFront(t1)

// 	var t2 = new(Task)
// 	t2.seqid = 2
// 	ts.pushFront(t2)

// 	var t3 = new(Task)
// 	t3.seqid = 3
// 	ts.pushFront(t3)

// 	var t4 = new(Task)
// 	t4.seqid = 4
// 	ts.pushFront(t4)

// 	var t5 = new(Task)
// 	t5.seqid = 5
// 	ts.pushFront(t5)

// 	var t6 = new(Task)
// 	t6.seqid = 6
// 	ts.pushFront(t6)

// 	t.Log("------------before------------")
// 	for idx := ts.next; idx != nil; idx = idx.next {
// 		t.Log(idx.seqid)
// 	}

// 	ts.remove(t4)
// 	t.Log("------------after------------")
// 	for idx := ts.next; idx != nil; idx = idx.next {
// 		t.Log(idx.seqid)
// 	}

// }
// func TestCTicker(t *testing.T) {
// 	ticker := NewTaskSchedule()
// 	for index := 0; index < 100; index++ {
// 		wg.Add(1)
// 		var task Task
// 		var i = index
// 		task.handler = func() error {
// 			fmt.Println("index:", i)
// 			wg.Done()
// 			return nil
// 		}
// 		ticker.AddTask(index+1, int64(index), &task)
// 	}
// 	ticker.CancelTask(10)
// 	wg.Wait()
// }

// func TestChanClose(t *testing.T) {
// 	var ch = make(chan int)
// 	wg.Add(1)
// 	go func() {
// 		ch <- 1
// 		wg.Done()
// 	}()
// 	<-ch
// 	close(ch)
// 	wg.Wait()
// 	t.Log("close success")
// }

// func TestChan(t *testing.T) {
// 	var ch = make(map[int]chan string)
// 	ch[1] = make(chan string)
// 	chCopy := ch[1]
// 	fmt.Printf("%v | %v\n", ch[1], chCopy)
// 	go func() {
// 		ch[1] <- "hello"
// 	}()
// 	<-ch[1]
// 	delete(ch, 1)

// 	go func() {
// 		chCopy <- "money"
// 	}()
// 	out := <-chCopy
// 	close(chCopy)

// 	fmt.Printf("%v | %v\n", ch[1], chCopy)
// 	t.Log(out)
// }

// func TestTimeBefore(t *testing.T) {
// 	var a = "2017-07-28 15:45:22"
// 	b, _ := time.ParseInLocation("2006-01-02 15:04:05", a, time.Local)
// 	t.Log(b.Before(time.Now()))
// }
