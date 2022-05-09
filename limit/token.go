package limit

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	xrate "golang.org/x/time/rate"
)

const (
	tokenLimiter = `local rate = tonumber(ARGV[1])
local cap = tonumber(ARGV[2]) -- capacity
local now = tonumber(ARGV[3]) -- now timestamp
local requested = tonumber(ARGV[4]) -- needed token
local fill_time = cap/rate
local ttl = math.floor(fill_time*2) -- give more time for more stability
-- KEYS[1]: token key
local last_tokens = tonumber(redis.call("get",KEYS[1]))
if last_tokens == nil then
	last_tokens = cap
end
-- KEYS[2]: token refreshed timestamp
local last_refreshed = tonumber(redis.call("get",KEYS[2]))
if last_refreshed == nil then
	last_refreshed = 0
end
local delta = math.max(0, now-last_refreshed)
local left = math.min(cap,last_tokens+delta*rate)
local new_tokens = left
local allowed = left >= requested
if allowed then
  new_tokens = left - requested
end
redis.call("setex", KEYS[1], ttl, new_tokens)
redis.call("setex", KEYS[2], ttl, now)
return allowed`
)

const (
	tokenFormat     = "{%s}.token-limiter"
	timestampFormat = "{%s}.token-limiter.ts"
	heartbeat       = time.Millisecond * 100
	remoteAliveNo   = iota
	remoteAliveYes
)

//令牌桶算法
type TokenBucket struct {
	rate         int64 //固定的token放入速率, r/s
	capacity     int64 //桶的容量
	tokens       int64 //桶中当前token数量
	lastTokenSec int64 //桶上次放token的时间戳 s
	lock         sync.Mutex
}

func NewTokenBucket(r, c int64) *TokenBucket {
	l := new(TokenBucket)
	l.rate = r
	l.capacity = c
	l.tokens = 0
	l.lastTokenSec = time.Now().Unix()
	return l
}

func (l *TokenBucket) Allow() bool {
	l.lock.Lock()
	defer l.lock.Unlock()
	now := time.Now().Unix()
	l.tokens = l.tokens + (now-l.lastTokenSec)*int64(l.rate) //先添加令牌
	if l.tokens > l.capacity {
		l.tokens = l.capacity
	}
	l.lastTokenSec = now
	if l.tokens > 0 { //还有令牌，领取令牌
		l.tokens--
		return true
	}
	return false //没有令牌,则拒绝
}

type TokenBucketStore struct {
	rate int // generate token number each second
	cap  int // at most token to store

	local          *xrate.Limiter // limiter in process
	remote         Store          // for distributed situation, can use redis
	tokenKey       string
	tsKey          string // timestamp key, tag get token time
	remoteMu       sync.Mutex
	remoteAlive    uint32 // ping remote server is alive or not
	monitorStarted bool
}

func NewTokenBucketRedis(store Store, key string, rate, cap int) *TokenBucketStore {
	return &TokenBucketStore{
		rate:        rate,
		cap:         cap,
		local:       xrate.NewLimiter(xrate.Every(time.Second/time.Duration(rate)), cap),
		remote:      store,
		tokenKey:    fmt.Sprintf(tokenFormat, key),
		tsKey:       fmt.Sprintf(timestampFormat, key),
		remoteAlive: 1,
	}
}

func (l *TokenBucketStore) Allow() bool {
	return l.AllowN(time.Now(), 1)
}

func (l *TokenBucketStore) AllowN(now time.Time, n int) bool {
	return l.reserveN(now, n)
}
func (l *TokenBucketStore) reserveN(now time.Time, n int) bool {
	if atomic.LoadUint32(&l.remoteAlive) == remoteAliveNo {
		return l.local.AllowN(now, n)
	}
	resp, err := l.remote.Eval(
		context.Background(),
		tokenLimiter,
		[]string{
			l.tokenKey,
			l.tsKey,
		}, []string{
			strconv.Itoa(l.rate),
			strconv.Itoa(l.cap),
			strconv.FormatInt(now.Unix(), 10),
			strconv.Itoa(n),
		})

	if l.remote.IsErrNil(err) {
		return false
	} else if err != nil {
		l.monitor()
		return l.local.AllowN(now, n)
	}

	code, ok := resp.(int64)
	if !ok {
		l.monitor()
		return l.local.AllowN(now, n)
	}
	return code == 1

}

func (l *TokenBucketStore) monitor() {
	l.remoteMu.Lock()
	defer l.remoteMu.Unlock()

	if l.monitorStarted {
		return
	}

	l.monitorStarted = true
	atomic.StoreUint32(&l.remoteAlive, remoteAliveNo)

	go l.ping()
}

func (l *TokenBucketStore) ping() {
	ticker := time.NewTicker(heartbeat)
	defer func() {
		ticker.Stop()
		l.remoteMu.Lock()
		l.monitorStarted = false
		l.remoteMu.Unlock()
	}()

	for range ticker.C {
		if l.remote.Ping() {
			atomic.StoreUint32(&l.remoteAlive, remoteAliveYes)
			return
		}
	}
}
