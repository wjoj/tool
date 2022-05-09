package metric

import "sync/atomic"

var _ Metric = &counter{}

type Counter interface {
	Metric
}

type counter struct {
	val int64
}

func (c *counter) Add(delta int64) {
	atomic.AddInt64(&c.val, delta)
}

func (c *counter) Value() int64 {
	return atomic.LoadInt64(&c.val)
}

func NewCounter() *counter {
	return &counter{}
}
