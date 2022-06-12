package task

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

type QueueCodeType int

const (
	QueueCodeTypeSuccess QueueCodeType = iota
	QueueCodeTypeFullWaitTimeOut
	QueueCodeTypeQueueEmptyTimeOut
)

func (m QueueCodeType) Val() error {
	if m == QueueCodeTypeFullWaitTimeOut {
		return fmt.Errorf("系统操作频繁，请稍后再试")
	} else if m == QueueCodeTypeQueueEmptyTimeOut {
		return fmt.Errorf("操作任务为空，请稍后再试")
	}
	return nil
}

type Queue struct {
	queueLink chan interface{}
	timeout   time.Duration
}

func NewQueue(q int, timeout time.Duration) *Queue {
	return &Queue{
		queueLink: make(chan interface{}, q),
		timeout:   timeout,
	}
}

func (q *Queue) Link(item interface{}, fnTask func(item interface{}) interface{}, fnOver func(item interface{}, data interface{}, code QueueCodeType)) {
	go func(s interface{}) {
		select {
		case q.queueLink <- s:
		case <-time.After(q.timeout):
			fnOver(s, nil, QueueCodeTypeFullWaitTimeOut)
			return
		}
		data := fnTask(s)
		select {
		case item, ok := <-q.queueLink:
			if !ok {
				return
			}
			fnOver(item, data, QueueCodeTypeSuccess)
			return
		case <-time.After(q.timeout):
			fnOver(s, nil, QueueCodeTypeQueueEmptyTimeOut)
			return
		}
	}(item)
}

func (q *Queue) Close() {
	close(q.queueLink)
}

type Task struct {
	Name string
	Func func()
}

type ContainerTask struct {
	taks   []*Task
	mtask  map[string]int
	offset int
}

func ContainerTasks(tasks ...*Task) *ContainerTask {
	t := new(ContainerTask)
	t.taks = append(t.taks, tasks...)
	t.mtask = make(map[string]int, len(t.taks))
	for idx, task := range tasks {
		t.mtask[task.Name] = idx
	}
	t.offset = 0
	return t
}

func (t *ContainerTask) Next() {
	t.taks[t.offset].Func()
	t.offset++
}

func (t *ContainerTask) Offset() int {
	return t.offset
}

func (t *ContainerTask) SetOffset(o int) {
	t.offset = o
}

func (t *ContainerTask) SyncMore(o ...int) {
	for _, i := range o {
		t.taks[i].Func()
	}
}

func (t *ContainerTask) AsyncMore(o ...int) {
	for _, i := range o {
		x := i
		go t.taks[x].Func()
	}
}

func (t *ContainerTask) AsyncWaitMore(o ...int) {
	var w sync.WaitGroup
	w.Add(len(o))
	for _, i := range o {
		go func(idx int) {
			t.taks[idx].Func()
			w.Done()
		}(i)
	}
	w.Wait()
}

type CallbackFunc func(uuid string, val any) error

type Callback struct {
	uuid     string
	callFunc CallbackFunc
	send     chan any
}

func NewCallback(uuid string) *Callback {
	return &Callback{
		uuid: uuid,
		send: make(chan any),
	}
}

var (
	ErrCallbackClose   = errors.New("wait for close")
	ErrCallbackTimeout = errors.New("wait for timeout")
)

func (c *Callback) Wait(timeout time.Duration, f CallbackFunc) error {
	c.callFunc = f
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	select {
	case <-ctx.Done():
		return ErrCallbackTimeout
	case val, is := <-c.send:
		if !is {
			return ErrCallbackClose
		}
		if err := c.callFunc(c.uuid, val); err != nil {
			return err
		}
	}
	return nil
}

func (c *Callback) IsClose(err error) bool {
	return errors.Is(ErrCallbackClose, err)
}

func (c *Callback) IsTimeout(err error) bool {
	return errors.Is(ErrCallbackTimeout, err)
}

func (c *Callback) Done(val any) {
	c.send <- val
}

func (c *Callback) Close() {
	close(c.send)
}
