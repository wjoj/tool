package limit

import (
	"errors"
	"fmt"
	"math"
	"sync/atomic"
	"time"

	"github.com/wjoj/tool/base/metric"
	"github.com/wjoj/tool/system"
)

/**
bbr := NewBBR(
		WithWindow(time.Second*5),
		WithBuckets(50),
		WithCpuThreshold(100),
	)
	f, err := bbr.Allow()
		f.Pass()
**/
var (
	cpu                  int64
	decay                      = 0.95
	initTime                   = time.Now()
	defaultWindow              = time.Second * 10
	defaultWindowBucket        = 100
	defaultCpuThreshold  int64 = 900
	ErrServiceOverloaded       = errors.New("service overloaded")
)

type Config struct {
	Enabled      bool
	Window       time.Duration
	WinBucket    int
	Rule         string
	Debug        bool
	CPUThreshold int64 // [0~1000]
}

func CpuProc() {
	ticker := time.NewTicker(time.Millisecond * 250)
	defer func() {
		ticker.Stop()
		if err := recover(); err != nil {
			fmt.Println("cpuProc fail, e:", err)
			go CpuProc()
		}
	}()

	// EMA algorithm: https://blog.csdn.net/m0_38106113/article/details/81542863
	for range ticker.C {
		stat := &system.Stat{}
		system.LoadStat(stat)
		preCpu := atomic.LoadInt64(&cpu)

		// cpu = cpuᵗ⁻¹ * decay + cpuᵗ * (1 - decay)
		curCpu := int64(float64(preCpu)*decay + float64(stat.Usage)*(1.0-decay))

		atomic.StoreInt64(&cpu, curCpu)
		fmt.Printf("old-self-cpu: %v, now-self-cpu:%v \n", preCpu, curCpu)
	}

}

type (
	// Shedder is the interface that wraps the Allow method.
	Shedder interface {
		// Allow returns the Promise if allowed, otherwise ErrServiceOverloaded.
		Allow() (Promise, error)
	}

	// A Promise interface is returned by Shedder.Allow to let callers tell
	// whether the processing request is successful or not.
	Promise interface {
		// Pass lets the caller tell that the call is successful.
		Pass()
		// Fail lets the caller tell that the call is failed.
		Fail()
	}

	cpuGetter func() int64

	// CounterCache is used to cache maxPASS and minRt result.
	// Value of current bucket is not counted in real time.
	// Cache time is equal to a bucket duration.
	CounterCache struct {
		val  int64
		time time.Time
	}

	// Stat is a snapshot for bbr status
	Stat struct {
		Cpu         int64
		InFlight    int64
		MaxInFlight int64
		MinRt       int64
		MaxPass     int64
	}

	// implement Promise
	promise struct {
		bbr       *BBR
		initTime  time.Time
		sinceTime time.Duration // since init time
	}

	// ShedderOption lets caller customize the Shedder.
	ShedderOption func(opts *Config)
)

// Fail
// add response time to rT counter
// decr req count to inFlight
func (p *promise) Fail() {
	rt := int64((time.Since(initTime) - p.sinceTime) / time.Millisecond)
	p.bbr.rtStat.Add(rt)
	atomic.AddInt64(&p.bbr.inFlight, -1)
}

// Pass
// add response time to counter
// decr req count to inFlight
// add pass to pass counter
func (p *promise) Pass() {
	rt := int64((time.Since(initTime) - p.sinceTime) / time.Millisecond)
	p.bbr.rtStat.Add(rt)
	atomic.AddInt64(&p.bbr.inFlight, -1)

	p.bbr.passStat.Add(1)
	fmt.Printf("------%#v, \n", p.bbr.Stat())
}

// BBR implement Shedder
type BBR struct {
	cpu             cpuGetter
	passStat        metric.RollingCounter
	rtStat          metric.RollingCounter
	inFlight        int64 // 当前正在处理的请求总数
	winBucketPerSec int64
	bucketDuration  time.Duration
	winSize         int
	conf            *Config
	preDrop         atomic.Value
	maxPassCache    atomic.Value // 最大请求数缓存, 缓存一个bucketDuration周期
	minRtCache      atomic.Value // 最小请求时间缓存, 缓存一个bucketDuration周期
}

func (b *BBR) timespan(lastTime time.Time) int {
	v := int(time.Since(lastTime) / b.bucketDuration)
	if v > -1 {
		return v
	}
	return b.winSize
}

// maxPass 单个采样窗口在一个采样周期中的最大的请求数,
// 默认的采样窗口是10s, 采样bucket数量100
func (b *BBR) maxPass() int64 {
	maxPassCache := b.maxPassCache.Load()
	if maxPassCache != nil {
		ps := maxPassCache.(*CounterCache)
		if b.timespan(ps.time) < 1 {
			return ps.val
		}
	}

	rawMaxPass := int64(b.passStat.Reduce(func(iterator metric.Iterator) float64 {
		var result = 1.0
		for i := 1; iterator.Next() && i < b.conf.WinBucket; i++ {
			bucket := iterator.Bucket()
			count := 0.0
			for _, point := range bucket.Points {
				count += point
			}
			result = math.Max(result, count)
		}
		return result
	}))

	if rawMaxPass == 0 {
		rawMaxPass = 1
	}

	b.maxPassCache.Store(&CounterCache{
		val:  rawMaxPass,
		time: time.Now(),
	})

	return rawMaxPass
}

// minRT 单个采样窗口中最小的响应时间
func (b *BBR) minRT() int64 {
	minRtCache := b.minRtCache.Load()
	if minRtCache != nil {
		rc := minRtCache.(*CounterCache)
		if b.timespan(rc.time) < 1 {
			return rc.val
		}
	}

	rawMinRt := int64(math.Ceil(b.rtStat.Reduce(func(iterator metric.Iterator) float64 {
		var res = math.MaxFloat64

		for i := 1; iterator.Next() && i < b.conf.WinBucket; i++ {
			bucket := iterator.Bucket()
			if len(bucket.Points) == 0 {
				continue
			}

			total := 0.0
			for _, point := range bucket.Points {
				total += point
			}
			avg := total / float64(bucket.Count)
			res = math.Min(res, avg)
		}

		return res

	})))

	if rawMinRt <= 0 {
		rawMinRt = 1
	}

	b.minRtCache.Store(&CounterCache{
		val:  rawMinRt,
		time: time.Now(),
	})

	return rawMinRt
}

// current window max flight
func (b *BBR) maxFlight() int64 {
	// winBucketPerSec: 每秒内的采样数量,
	// 计算方式:
	// int64(time.Second)/(int64(conf.Window)/int64(conf.WinBucket)),
	// conf.Window默认值10s, conf.WinBucket默认值100.
	// 简化下公式: 1/(10/100) = 10, 所以每秒内的采样数就是10
	// maxQPS = maxPass * winBucketPerSec
	// minRT = min average response time in milliseconds
	// maxQPS * minRT / milliseconds_per_second
	return int64(
		math.Floor(
			float64(
				b.maxPass()*b.minRT()*b.winBucketPerSec)/1e3 + 0.5,
		),
	)

}

// Cooling time: 1s
func (b *BBR) shouldDrop() bool {
	// not overload
	if b.cpu() < b.conf.CPUThreshold {
		preDropTime, _ := b.preDrop.Load().(time.Duration)
		// didn't drop before
		if preDropTime == 0 {
			return false
		}

		// in cooling time duration, 1s
		// should not drop
		if time.Since(initTime)-preDropTime <= time.Second {
			inFlight := atomic.LoadInt64(&b.inFlight)
			return inFlight > 1 && inFlight > b.maxFlight()
		}

		// store this drop time as pre drop time
		b.preDrop.Store(time.Duration(0))
		return false
	}

	// overload case
	inFlight := atomic.LoadInt64(&b.inFlight)
	shouldDrop := inFlight > 1 && inFlight > b.maxFlight()

	if shouldDrop {
		preDropTime, _ := b.preDrop.Load().(time.Duration)
		if preDropTime != 0 {
			return shouldDrop
		}
		b.preDrop.Store(time.Since(initTime))
	}

	return shouldDrop
}

func (b *BBR) Stat() Stat {
	return Stat{
		Cpu:         b.cpu(),
		InFlight:    atomic.LoadInt64(&b.inFlight),
		MaxInFlight: b.maxFlight(),
		MinRt:       b.minRT(),
		MaxPass:     b.maxPass(),
	}
}

func (b *BBR) Allow() (Promise, error) {

	if b.shouldDrop() {
		return nil, ErrServiceOverloaded
	}

	// total req incr
	atomic.AddInt64(&b.inFlight, 1)

	return &promise{
		bbr:       b,
		initTime:  initTime,
		sinceTime: time.Since(initTime),
	}, nil

}

// WithBuckets customizes the Shedder with given number of buckets.
func WithBuckets(buckets int) ShedderOption {
	return func(opts *Config) {
		opts.WinBucket = buckets
	}
}

// WithCpuThreshold customizes the Shedder with given cpu threshold.
func WithCpuThreshold(threshold int64) ShedderOption {
	return func(opts *Config) {
		opts.CPUThreshold = threshold
	}
}

// WithWindow customizes the Shedder with given
func WithWindow(window time.Duration) ShedderOption {
	return func(opts *Config) {
		opts.Window = window
	}
}

func NewBBR(options ...ShedderOption) Shedder {

	cfg := &Config{
		Window:       defaultWindow,
		WinBucket:    defaultWindowBucket,
		CPUThreshold: defaultCpuThreshold,
	}

	for _, option := range options {
		option(cfg)
	}

	cpuF := func() int64 {
		return atomic.LoadInt64(&cpu)
	}

	size := cfg.WinBucket
	bucketDuration := cfg.Window / time.Duration(cfg.WinBucket)
	passStat := metric.NewRollingCounter(
		metric.WithRollingCounterOptsBucketDuration(bucketDuration),
		metric.WithRollingCounterOptsSize(size),
	)

	rtStat := metric.NewRollingCounter(
		metric.WithRollingCounterOptsBucketDuration(bucketDuration),
		metric.WithRollingCounterOptsSize(size),
	)

	shedder := &BBR{
		cpu:             cpuF,
		passStat:        passStat,
		rtStat:          rtStat,
		winBucketPerSec: int64(time.Second) / (int64(cfg.Window) / int64(cfg.WinBucket)),
		bucketDuration:  bucketDuration,
		winSize:         cfg.WinBucket,
		conf:            cfg,
	}

	return shedder

}
