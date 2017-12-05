package util

// TaskSchedule 任务调度器
type TaskSchedule struct {
	s *tasksSlot // 环形队列的槽
	// taskHoler *taskHolder // 环形队列所有的定时任务
}

// NewTaskSchedule new task shcedule
func NewTaskSchedule() *TaskSchedule {
	schedule := new(TaskSchedule)
	schedule.s = newTasksSlot()
	// schedule.taskHoler = newTaskHolder()
	schedule.s.loop()
	return schedule
}

// CancelTask cancel task
// func (schedule *TaskSchedule) CancelTask(bizid int64) {
// 	schedule.taskHoler.cancel(bizid)
// }

// AddTask add task
func (schedule *TaskSchedule) AddTask(sencond int, bizid int64, t *Task) error {
	t.seqid = bizid
	count := schedule.s.current + sencond
	t.cycleNum = count / schedule.s.count
	index := count % schedule.s.count
	// schedule.taskHoler.add(bizid, t)
	return schedule.s.addByIndex(index, t)
}
