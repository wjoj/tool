package tool

import (
	"fmt"
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
		case item := <-q.queueLink:
			fnOver(item, data, QueueCodeTypeSuccess)
			return
		case <-time.After(q.timeout):
			fnOver(s, nil, QueueCodeTypeQueueEmptyTimeOut)
			return
		}
	}(item)
}
