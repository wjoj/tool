package limit

import (
	"math"
	"sync"
	"time"
)

//LeakyBucket 漏桶算法
type LeakyBucket struct {
	rate       float64 //固定每秒出水速率
	capacity   float64 //桶的容量
	water      float64 //桶中当前水量
	lastLeakMs int64   //桶上次漏水时间戳 ms
	lock       sync.Mutex
}

func NewLeakyBucket(r, c float64) *LeakyBucket {
	lb := new(LeakyBucket)
	lb.rate = r
	lb.capacity = c
	lb.water = 0
	lb.lastLeakMs = time.Now().UnixNano() / 1e6
	return lb
}

func (l *LeakyBucket) Allow() bool {
	l.lock.Lock()
	defer l.lock.Unlock()
	now := time.Now().UnixNano() / 1e6
	eclipse := float64((now - l.lastLeakMs)) * l.rate / 1000 //先执行漏水
	l.water = l.water - eclipse
	l.water = math.Max(0, l.water)
	l.lastLeakMs = now
	if (l.water + 1) < l.capacity {
		l.water++
		return true
	}
	return false
}
