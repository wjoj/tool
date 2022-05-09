package rpc

import (
	"fmt"
	"log"
	"math"
	"math/rand"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/wjoj/tool/util"

	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/balancer/base"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/resolver"
	"google.golang.org/grpc/status"
)

const (
	BalancerNameP2CEWMA = "p2c_EWMA"

	decayTime   = int64(time.Second * 10)
	logInterval = time.Minute

	initSuccess     = 1000
	throttleSuccess = initSuccess / 2
	penalty         = int64(math.MaxInt32)
	forcePick       = int64(time.Second)
	pickTimes       = 3
)

var (
	logger   = log.New(os.Stdout, fmt.Sprintf("[DEBUG][pkg=p2c][%s] ", time.Now().Format(time.StampMilli)), log.Lshortfile)
	initTime = time.Now().AddDate(-1, -1, -1)
)

func newBuilder() balancer.Builder {
	return base.NewBalancerBuilder(BalancerNameP2CEWMA, &p2cPickBuilder{}, base.Config{HealthCheck: true})
}

func init() {
	balancer.Register(newBuilder())
}

type p2cPickBuilder struct {
}

func (p *p2cPickBuilder) Build(info base.PickerBuildInfo) balancer.Picker {
	logger.Printf("p2cPickBuilder: Build called with info: %v", info)
	if len(info.ReadySCs) == 0 {
		return base.NewErrPicker(balancer.ErrNoSubConnAvailable)
	}

	var conns []*p2cSubConn
	for conn, connInfo := range info.ReadySCs {
		conns = append(conns, &p2cSubConn{
			addr:        connInfo.Address,
			conn:        conn,
			successEWMA: initSuccess,
		})
	}

	return &p2cPicker{
		conns: conns,
		r:     rand.New(rand.NewSource(time.Now().UnixNano())),
		stamp: util.NewAtomicDuration(),
	}

}

type p2cPicker struct {
	conns []*p2cSubConn
	r     *rand.Rand
	stamp *util.AtomicDuration
	lock  sync.Mutex
}

func (p *p2cPicker) Pick(balancer.PickInfo) (balancer.PickResult, error) {
	p.lock.Lock()
	defer p.lock.Unlock()

	var chosen *p2cSubConn
	switch len(p.conns) {
	case 0: // 没有节点, 直接报错
		return balancer.PickResult{}, balancer.ErrNoSubConnAvailable
	case 1: // 一个节点, 直接返回
		chosen = p.choose(p.conns[0], nil)
	case 2: // 两个节点, 返回负载最低的节点
		chosen = p.choose(p.conns[0], p.conns[1])
	default: // 多个节点, 最多经常三次计算, 选择合适的节点
		var node1, node2 *p2cSubConn
		// 三次随机选择节点
		for i := 0; i < pickTimes; i++ {
			a := p.r.Intn(len(p.conns))
			b := p.r.Intn(len(p.conns) - 1)
			if b >= a {
				// 防止出现相同节点
				b++
			}
			node1 = p.conns[a]
			node2 = p.conns[b]

			// 选出一次符合要求的节点则停止
			if node1.healthy() && node2.healthy() {
				break
			}
		}

		chosen = p.choose(node1, node2)
	}

	atomic.AddInt64(&chosen.inFlight, 1)
	atomic.AddInt64(&chosen.requests, 1)

	return balancer.PickResult{
		SubConn: chosen.conn,
		Done:    p.buildDoneFunc(chosen),
	}, nil
}

func (p *p2cPicker) buildDoneFunc(c *p2cSubConn) func(info balancer.DoneInfo) {
	start := int64(time.Since(initTime))
	return func(info balancer.DoneInfo) {
		// 正在处理的请求数-1
		atomic.AddInt64(&c.inFlight, -1)

		// 计算相对时间
		now := time.Since(initTime)
		last := atomic.SwapInt64(&c.lastLag, int64(now))
		td := int64(now) - last
		if td < 0 {
			td = 0
		}

		// 牛顿冷却定律的衰减模型, 确定ewma中的β值, β = 1/e^(k*△t)
		beta := math.Exp(float64(-td) / float64(decayTime))
		lag := int64(now) - start
		if lag < 0 {
			lag = 0
		}
		olag := atomic.LoadUint64(&c.lagEWMA)
		if olag == 0 {
			beta = 0
		}

		// 指数加权平均算法 vt = vt-1 * β + vt * (1 - β)
		// 存储当前lagEWMA
		atomic.StoreUint64(&c.lagEWMA, uint64(float64(olag)*beta+float64(lag)*(1-beta)))

		success := initSuccess
		if info.Err != nil {
			switch status.Code(info.Err) {
			case codes.DeadlineExceeded, codes.Internal, codes.Unavailable, codes.DataLoss:
				success = 0
			}
		}

		oldSuccess := atomic.LoadUint64(&c.successEWMA)
		// 指数加权平均算法 vt = vt-1 * β + vt * (1 - β)
		// 存储当前successEWMA
		atomic.StoreUint64(&c.successEWMA, uint64(float64(oldSuccess)*beta+float64(success)*(1-beta)))

		stamp := p.stamp.Load()
		if now-stamp >= logInterval {
			if p.stamp.CompareAndSwap(stamp, now) {
				p.logStats()
			}
		}
	}
}

func (p *p2cPicker) logStats() {
	var stats []string

	p.lock.Lock()
	defer p.lock.Unlock()

	for _, conn := range p.conns {
		stats = append(stats, fmt.Sprintf("conn: %s, load: %d, reqs: %d",
			conn.addr.Addr, conn.load(), atomic.SwapInt64(&conn.requests, 0)))
	}

	logger.Printf("%s", strings.Join(stats, "; "))
}

func (p *p2cPicker) choose(c1, c2 *p2cSubConn) *p2cSubConn {
	start := int64(time.Since(initTime))
	if c2 == nil {
		atomic.StoreInt64(&c1.pickTime, start)
		return c1
	}

	// 优先选择负载低的
	if c1.load() > c2.load() {
		c1, c2 = c2, c1
	}

	// 选择响应快的
	// 如果在超时时间内节点没有被选中过, 则选择该节点
	pick := atomic.LoadInt64(&c2.pickTime)
	if start-pick > forcePick && atomic.CompareAndSwapInt64(&c2.pickTime, pick, start) {
		return c2
	}

	atomic.StoreInt64(&c1.pickTime, start)
	return c1
}

type p2cSubConn struct {
	addr resolver.Address
	conn balancer.SubConn

	lagEWMA uint64 // 请求耗时, 计算后的ewma

	inFlight    int64  // 节点拥塞度, 正在处理的请求
	successEWMA uint64 // 一段时间内此连接的健康状态, 计算后的ewma
	requests    int64  // 请求量

	lastLag  int64 // 上一次请求耗时, 用于计算ewma
	pickTime int64 // 上一次选择的时间时间戳
}

func (s *p2cSubConn) healthy() bool {
	return atomic.LoadUint64(&s.successEWMA) > throttleSuccess
}

// load = lagEWMA * inFlight
func (s *p2cSubConn) load() int64 {
	lag := int64(math.Sqrt(float64(atomic.LoadUint64(&s.lagEWMA) + 1)))
	load := lag * (atomic.LoadInt64(&s.inFlight) + 1)
	if load == 0 {
		// penalty是初始化没有数据时的惩罚值, 在没有被选过的情况下, 会强制选择一次
		return penalty
	}

	return load
}
