package metric

import (
	"fmt"
	"time"
)

type (
	RollingCounter interface {
		Metric
		Aggregation
		TimeSpan() int
		Reduce(func(iterator Iterator) float64) float64
	}

	RollingCounterOpts struct {
		Size           int
		BucketDuration time.Duration
	}

	// rollingCounter implement RollingCounter
	rollingCounter struct {
		policy *RollingPolicy
	}

	RollingCounterOptsFn func(opt *RollingCounterOpts)
)

func WithRollingCounterOptsSize(size int) RollingCounterOptsFn {
	return func(opt *RollingCounterOpts) {
		opt.Size = size
	}
}
func WithRollingCounterOptsBucketDuration(bucketDuration time.Duration) RollingCounterOptsFn {
	return func(opt *RollingCounterOpts) {
		opt.BucketDuration = bucketDuration
	}
}

func NewRollingCounter(fns ...RollingCounterOptsFn) RollingCounter {

	cfg := &RollingCounterOpts{}

	for _, fn := range fns {
		fn(cfg)
	}

	policy := NewRollingPolicy(
		NewWindow(WithWindowSize(cfg.Size)),
		WithRollingPolicyOptsBucketDuration(cfg.BucketDuration),
	)

	return &rollingCounter{policy: policy}

}

func (r *rollingCounter) Add(val int64) {
	if val < 0 {
		panic(fmt.Errorf("stat/metric: cannot decrease in value. val: %d", val))
	}
	r.policy.Add(float64(val))
}

func (r *rollingCounter) Reduce(f func(Iterator) float64) float64 {
	return r.policy.Reduce(f)
}

func (r *rollingCounter) Avg() float64 {
	return r.policy.Reduce(Avg)
}

func (r *rollingCounter) Min() float64 {
	return r.policy.Reduce(Min)
}

func (r *rollingCounter) Max() float64 {
	return r.policy.Reduce(Max)
}

func (r *rollingCounter) Sum() float64 {
	return r.policy.Reduce(Sum)
}

func (r *rollingCounter) Value() int64 {
	return int64(r.Sum())
}

func (r *rollingCounter) TimeSpan() int {
	r.policy.mu.RLock()
	defer r.policy.mu.RUnlock()
	return r.policy.timespan()
}
