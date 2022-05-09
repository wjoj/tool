package util

import (
	"container/list"
	"fmt"
	"time"
)

const drainWorkers = 8

type (
	Execute func(key, value interface{})

	TimingWheel struct {
		interval  time.Duration
		ticker    Ticker
		slots     []*list.List
		timers    *SafeMap
		tickedPos int
		numSlots  int
		execute   Execute

		setChan    chan timingEntry
		moveChan   chan baseEntry
		removeChan chan interface{}
		drainChan  chan func(key, value interface{})
		stopChan   chan struct{}
	}

	baseEntry struct {
		delay time.Duration
		key   interface{}
	}

	timingEntry struct {
		baseEntry
		value   interface{}
		circle  int
		diff    int
		removed bool
	}

	positionEntry struct {
		pos  int
		item *timingEntry
	}

	timingTask struct {
		key, value interface{}
	}
)

func NewTimingWheel(interval time.Duration, numSlots int, execute Execute) (*TimingWheel, error) {
	if interval <= 0 || numSlots <= 0 || execute == nil {
		return nil, fmt.Errorf("invalid param, interval: %v, numSlots: %d, execute: %p", interval, numSlots, execute)
	}

	return newTimingWheel(interval, numSlots, execute, NewTicker(interval))
}

func newTimingWheel(interval time.Duration, numSlots int, execute Execute, ticker Ticker) (*TimingWheel, error) {
	tw := &TimingWheel{
		interval:  interval,
		ticker:    ticker,
		slots:     make([]*list.List, numSlots),
		timers:    NewSafeMap(),
		tickedPos: numSlots - 1,
		numSlots:  numSlots,
		execute:   execute,

		setChan:    make(chan timingEntry),
		moveChan:   make(chan baseEntry),
		removeChan: make(chan interface{}),
		drainChan:  make(chan func(key, value interface{})),
		stopChan:   make(chan struct{}),
	}

	tw.initSlots()
	go tw.run()

	return tw, nil
}

// Drain 执行所有任务
func (tw *TimingWheel) Drain(fn func(key, value interface{})) {
	tw.drainChan <- fn
}

// MoveTimer 在时间轮上根据指定时间移动指定任务
func (tw *TimingWheel) MoveTimer(key interface{}, delay time.Duration) {
	if delay <= 0 || key == nil {
		return
	}

	tw.moveChan <- baseEntry{
		delay: delay,
		key:   key,
	}
}

// RemoveTimer 移除时间轮上的指定任务
func (tw *TimingWheel) RemoveTimer(key interface{}) {
	if key == nil {
		return
	}

	tw.removeChan <- key
}

// SetTimer 在时间轮上新增任务
func (tw *TimingWheel) SetTimer(key, value interface{}, delay time.Duration) {
	if delay <= 0 || key == nil {
		return
	}

	tw.setChan <- timingEntry{
		baseEntry: baseEntry{
			delay: delay,
			key:   key,
		},
		value: value,
	}
}

// Stop 停止时间轮任务轮询
func (tw TimingWheel) Stop() {
	close(tw.stopChan)
}

func (tw *TimingWheel) initSlots() {
	for i := range tw.slots {
		tw.slots[i] = list.New()
	}
}

// run chan通信, 多事件轮询
func (tw *TimingWheel) run() {
	for {
		select {
		case <-tw.ticker.Chan():
			tw.onTick()
		case task := <-tw.setChan:
			tw.setTask(&task)
		case key := <-tw.removeChan:
			tw.removeTask(key)
		case task := <-tw.moveChan:
			tw.moveTask(task)
		case fn := <-tw.drainChan:
			tw.drainAll(fn)
		case <-tw.stopChan:
			tw.ticker.Stop()
			return
		}
	}
}

// 清空所有任务, 能执行优先执行
func (tw *TimingWheel) drainAll(fn func(key, value interface{})) {
	workers := make(chan struct{}, drainWorkers)

	for _, slot := range tw.slots {
		for e := slot.Front(); e != nil; {
			task := e.Value.(*timingEntry)
			next := e.Next()
			slot.Remove(e)
			e = next
			if !task.removed {
				workers <- struct{}{}
				GoSave(func() {
					defer func() {
						<-workers
					}()
					fn(task.key, task.value)
				})
			}
		}
	}
}

func (tw *TimingWheel) removeTask(key interface{}) {
	val, ok := tw.timers.Get(key)
	if !ok {
		return
	}

	timer := val.(*positionEntry)
	timer.item.removed = true
	tw.timers.Del(key)
}

func (tw *TimingWheel) setTask(task *timingEntry) {
	// 轮盘最小时间滚动刻度
	if task.delay < tw.interval {
		task.delay = tw.interval
	}

	if val, ok := tw.timers.Get(task.key); ok {
		// 有相同任务
		entry := val.(*positionEntry)
		entry.item.value = task.value
		tw.moveTask(task.baseEntry)
	} else {
		// 全新添加任务
		pos, circle := tw.getPosAndCircle(task.delay)
		task.circle = circle
		tw.slots[pos].PushBack(task)
		tw.setTimerPosition(pos, task)
	}
}

// 移动任务
// 是否能够获取当前任务
//	不能: 退出
//	能:
//		1) 任务时间小于轮盘时间刻度: 执行任务
//		2) 多层时间轮任务: 层级-1, 计算层级差值
//		3) 标记删除旧任务, 设置新任务
func (tw *TimingWheel) moveTask(task baseEntry) {
	val, ok := tw.timers.Get(task.key)
	if !ok {
		return
	}

	timer := val.(*positionEntry)
	if task.delay < tw.interval {
		GoSave(func() {
			tw.execute(timer.item.key, timer.item.value)
		})
		return
	}

	pos, circle := tw.getPosAndCircle(task.delay)
	if pos >= timer.pos {
		// 新任务延后执行
		timer.item.circle = circle
		timer.item.diff = pos - timer.pos
	} else if circle > 0 {
		// 多层级任务
		circle--
		timer.item.circle = circle
		timer.item.diff = tw.numSlots + pos - timer.pos
	} else {
		// 标记为删除任务
		timer.item.removed = true
		newItem := &timingEntry{
			baseEntry: task,
			value:     timer.item.value,
		}
		tw.slots[pos].PushBack(newItem)
		tw.setTimerPosition(pos, newItem)
	}

}

func (tw *TimingWheel) getPosAndCircle(d time.Duration) (pos, circle int) {
	steps := int(d / tw.interval)
	pos = (tw.tickedPos + steps) % tw.numSlots
	circle = (steps - 1) / tw.numSlots

	return pos, circle
}

func (tw *TimingWheel) onTick() {
	tw.tickedPos = (tw.tickedPos + 1) % tw.numSlots
	taskList := tw.slots[tw.tickedPos]

	tw.scanAndRun(taskList)
}

func (tw *TimingWheel) scanAndRun(taskList *list.List) {
	tasks := tw.scanTask(taskList)
	tw.runTask(tasks)
}

// scanTask 轮询所有任务, 执行/删除/移动...
func (tw *TimingWheel) scanTask(taskList *list.List) []timingTask {
	var tasks []timingTask

	for e := taskList.Front(); e != nil; {
		task := e.Value.(*timingEntry)
		if task.removed {
			// 标记为可移除任务
			next := e.Next()
			taskList.Remove(e)
			e = next
			continue
		} else if task.circle > 0 {
			// 多层时间轮任务, 层级 -1
			task.circle--
			e = e.Next()
			continue
		} else if task.diff > 0 {
			// 多层时间轮任务到最底层, 通过diff判断差值, 放入当前轮盘的任务链表中
			next := e.Next()
			taskList.Remove(e)

			pos := (tw.tickedPos + task.diff) % tw.numSlots
			tw.slots[pos].PushBack(task)
			tw.setTimerPosition(pos, task)
			task.diff = 0
			e = next
			continue
		}

		tasks = append(tasks, timingTask{
			key:   task.key,
			value: task.value,
		})
		next := e.Next()
		taskList.Remove(e)
		tw.timers.Del(task.key)
		e = next
	}

	return tasks
}

func (tw *TimingWheel) runTask(tasks []timingTask) {
	if len(tasks) == 0 {
		return
	}

	go func() {
		for i := range tasks {
			GoSave(func() {
				tw.execute(tasks[i].key, tasks[i].value)
			})
		}
	}()
}

func (tw *TimingWheel) setTimerPosition(pos int, task *timingEntry) {
	if v, ok := tw.timers.Get(task.key); ok {
		timer := v.(*positionEntry)
		timer.item = task
		timer.pos = pos
	} else {
		tw.timers.Set(task.key, &positionEntry{
			pos:  pos,
			item: task,
		})
	}
}
