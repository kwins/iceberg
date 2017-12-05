package util

import (
	"errors"
	"sync"
	"time"
)

var (
	errIndexOutofRange = errors.New("index out of range")
	errTaskNil         = errors.New("task nil")
)

type tasksSlot struct {
	locker   sync.RWMutex
	slots    []*tasks      //
	count    int           // 槽总数
	current  int           // 当前Slot index
	duration time.Duration // 槽移动单位时间
}

func newTasksSlot() *tasksSlot {
	s := new(tasksSlot)
	s.count = 3600
	s.current = 1
	s.duration = time.Second
	s.slots = make([]*tasks, s.count+1)
	for k := range s.slots {
		s.slots[k] = newTasks()
	}
	return s
}

func (slot *tasksSlot) loop() {
	nt := time.NewTicker(slot.duration)
	go func() {
		for {
			<-nt.C
			// 防止任务过多，一秒内没执行完
			// 串行操作task链表
			go slot.nextSlot()
		}
	}()
}

func (slot *tasksSlot) nextSlot() {
	slot.locker.RLock()
	ts := slot.slots[slot.current]
	slot.locker.RUnlock()
	for e := ts.next; e != nil; e = e.next {
		if e.cycleNum == 0 {
			go e.handler(e.params)
			ts.remove(e)
		} else {
			e.cycleNum--
		}
	}
	// 移动到下一个slot
	slot.current++
	if slot.current == slot.count {
		slot.current = 1
	}
}

func (slot *tasksSlot) addByIndex(index int, t *Task) error {
	if index > slot.count {
		return errIndexOutofRange
	}
	if nil == t {
		return errTaskNil
	}
	slot.locker.Lock()
	slot.slots[index].pushFront(t)
	slot.locker.Unlock()
	return nil
}
