package limit

import (
	"sync"
	"time"
)

//Counter 计数器 缺陷:
type Counter struct {
	rate  int
	begin time.Time
	cycle time.Duration
	count int
	lock  sync.Mutex
}

func NewCounter(r int, cycle time.Duration) *Counter {
	t := new(Counter)
	t.Set(r, cycle)
	return t
}

func (l *Counter) Allow() bool {
	l.lock.Lock()
	defer l.lock.Unlock()
	if l.count == l.rate-1 {
		now := time.Now()
		if now.Sub(l.begin) >= l.cycle {
			l.Reset(now)
			return true
		} else {
			return false
		}
	}
	l.count++
	return true
}

func (l *Counter) Set(r int, cycle time.Duration) {
	l.rate = r
	l.begin = time.Now()
	l.cycle = cycle
	l.count = 0
}

func (l *Counter) Reset(t time.Time) {
	l.begin = t
	l.count = 0
}
