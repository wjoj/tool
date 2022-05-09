package metric

import (
	"fmt"
	"sync"
	"time"
)

type (
	RollingPolicy struct {
		mu     sync.RWMutex
		size   int
		window *Window
		offset int

		bucketDuration time.Duration
		lastAppendTime time.Time
	}

	RollingPolicyOpts struct {
		BucketDuration time.Duration
	}

	RollingPolicyOptsFunc func(opt *RollingPolicyOpts)
)

func WithRollingPolicyOptsBucketDuration(t time.Duration) RollingPolicyOptsFunc {
	return func(opt *RollingPolicyOpts) {
		opt.BucketDuration = t
	}
}

func NewRollingPolicy(window *Window, ops ...RollingPolicyOptsFunc) *RollingPolicy {
	cfg := &RollingPolicyOpts{}

	for _, op := range ops {
		op(cfg)
	}

	return &RollingPolicy{
		size:           window.Size(),
		window:         window,
		offset:         0,
		bucketDuration: cfg.BucketDuration,
		lastAppendTime: time.Now(),
	}

}

func (r *RollingPolicy) timespan() int {
	v := int(time.Since(r.lastAppendTime) / r.bucketDuration)
	//if v > -1 { // maybe time backwards
	//	return v
	//}
	//return r.size
	if v >= 0 && v < r.size {
		return v
	}
	return r.size
}

func (r *RollingPolicy) add(append func(offset int, val float64), val float64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	timespan := r.timespan()
	if timespan <= 0 {
		append(r.offset, val)
		return
	}

	fmt.Println("timespan: ", timespan)
	fmt.Println("lastAppendTime: ", r.lastAppendTime)

	r.lastAppendTime = r.lastAppendTime.Add(
		time.Duration(timespan * int(r.bucketDuration)),
	)

	fmt.Println("lastAppendTime: ", r.lastAppendTime)

	offset := r.offset

	// reset the expired buckets
	s := offset + 1
	if timespan > r.size {
		timespan = r.size
	}

	e, e1 := s+timespan, 0
	if e > r.size {
		e1 = e - r.size
		e = r.size
	}

	for i := s; i < e; i++ {
		r.window.ResetBucket(i)
		offset = i
	}

	for i := 0; i < e1; i++ {
		r.window.ResetBucket(i)
		offset = i
	}

	r.offset = offset

	append(r.offset, val)

}

func (r *RollingPolicy) Append(val float64) {
	r.add(r.window.AppendBucketPoint, val)
}

func (r *RollingPolicy) Add(val float64) {
	r.add(r.window.AddBucketPoint, val)
}

// Reduce 从当前timespan下一位开始迭代
func (r *RollingPolicy) Reduce(fn func(iterator Iterator) float64) (val float64) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	timespan := r.timespan()
	if count := r.size - timespan; count > 0 {
		offset := r.offset + timespan + 1
		if offset >= r.size {
			offset = offset - r.size
		}
		val = fn(r.window.Iterator(offset, count))
	}

	return val
}
