package util

/*
优化后新增循环定时队列，延迟消息可以使用此结构
*/
import (
	"sync"
)

// Task task
type Task struct {
	seqid    int64                   //
	cycleNum int                     // 第几圈执行此定时任务
	handler  func(interface{}) error // 执行任务的函数,参数个自定义
	params   interface{}             // 参数
	next     *Task                   // next
	pre      *Task                   // pre
}

// SetHandler 设置任务函数
func (tk *Task) SetHandler(h func(interface{}) error) {
	if nil == h {
		tk.handler = func(interface{}) error {
			return nil
		}
	}
	tk.handler = h
}

type tasks struct {
	locker sync.RWMutex
	next   *Task // Slot中任务集合
	l      int   // task length
}

// 双向链表
func newTasks() *tasks {
	t := new(tasks)
	t.next = nil
	t.l = 0
	return t
}

func (tks *tasks) remove(t *Task) {
	tks.locker.Lock()
	if t.next != nil {
		t.next.pre = t.pre
	}
	if t.pre != nil {
		t.pre.next = t.next
	} else {
		tks.next = t.next
	}
	tks.l--
	tks.locker.Unlock()
}

func (tks *tasks) pushFront(t *Task) {
	tks.locker.Lock()
	if tks.next != nil {
		tks.next.pre = t
	}
	t.next = tks.next
	t.pre = nil
	tks.next = t
	tks.l++
	tks.locker.Unlock()
}

// func (tks *tasks) drop() {
// 	tks.locker.Lock()
// 	for idx := tks.next; idx != nil; idx = idx.next {
// 		idx.cancel = true
// 	}
// 	tks.locker.Unlock()
// }

/*
taskHolder 维护所有任务，方便取消任务
*/
type taskHolder struct {
	locker sync.RWMutex
	bulk   map[int64]*Task
}

func newTaskHolder() *taskHolder {
	h := new(taskHolder)
	h.bulk = make(map[int64]*Task)
	return h
}

// Get Get
func (holder *taskHolder) get(sequenceid int64) *Task {
	holder.locker.Lock()
	defer holder.locker.Unlock()
	if v, ok := holder.bulk[sequenceid]; ok {
		delete(holder.bulk, sequenceid)
		return v
	}
	return nil
}

// add task
func (holder *taskHolder) add(sequenceid int64, t *Task) {
	holder.locker.Lock()
	holder.bulk[sequenceid] = t
	holder.locker.Unlock()
}

// delete 删除
func (holder *taskHolder) delete(sequenceid int64) {
	holder.locker.Lock()
	delete(holder.bulk, sequenceid)
	holder.locker.Unlock()
}

// func (holder *taskHolder) cancel(sequenceid int64) {
// 	if t := holder.get(sequenceid); t != nil {
// 		t.cancel = true
// 	}
// }
