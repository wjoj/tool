package cache

import (
	"context"
	"sync"
	"time"

	"github.com/wjoj/tool/util"
	"golang.org/x/sync/singleflight"
)

const (
	defaultName         = "proc"
	timingWheelSlots    = 300
	timingWheelInterval = time.Second
)

type (
	Local struct {
		name   string
		lock   sync.Mutex
		data   map[string]interface{}
		expire time.Duration

		lru         Lru
		timingWheel *util.TimingWheel
		sf          *singleflight.Group
		stat        *util.Stat
	}

	Option func(cache *Local)
)

func WithName(name string) Option {
	return func(cache *Local) {
		cache.name = name
	}
}

func WithLimit(limit int) Option {
	return func(cache *Local) {
		cache.lru = NewLru(limit, cache.onEvict)
	}
}

func NewLocal(expire time.Duration, opts ...Option) (cache *Local, err error) {
	cache = &Local{
		data:   make(map[string]interface{}),
		expire: expire,
		lru:    NewNoneLru(),
		sf:     &singleflight.Group{},
	}

	for _, opt := range opts {
		opt(cache)
	}

	if len(cache.name) == 0 {
		cache.name = defaultName
	}

	cache.stat = util.NewStat(cache.name, cache.size)

	var tw *util.TimingWheel
	tw, err = util.NewTimingWheel(timingWheelInterval, timingWheelSlots, func(key, value interface{}) {
		v, ok := key.(string)
		if !ok {
			return
		}

		cache.Del(context.Background(), v)
	})
	if err != nil {
		return nil, err
	}

	cache.timingWheel = tw
	return cache, nil
}

func (c *Local) Take(ctx context.Context, key string, fetch func(ctx context.Context) (interface{}, error)) (interface{}, error) {
	if val, ok := c.doGet(ctx, key); ok {
		c.stat.Hit()
		return val, nil
	}

	var fresh bool
	val, err, _ := c.sf.Do(key, func() (interface{}, error) {
		// double check
		if val, ok := c.doGet(ctx, key); ok {
			c.stat.Hit()
			return val, nil
		}

		v, err := fetch(ctx)
		if err != nil {
			return nil, err
		}

		fresh = true
		c.Set(ctx, key, v)
		return v, nil
	})
	if err != nil {
		return nil, err
	}

	if fresh {
		c.stat.Miss()
		return val, nil
	}

	c.stat.Hit()
	return val, nil
}

func (c *Local) Set(ctx context.Context, key string, value interface{}) {
	c.lock.Lock()
	_, ok := c.data[key]
	c.data[key] = value
	c.lru.Add(key)
	c.lock.Unlock()

	if ok {
		c.timingWheel.MoveTimer(key, c.expire)
	} else {
		c.timingWheel.SetTimer(key, value, c.expire)
	}
}

func (c *Local) Del(ctx context.Context, key string) {
	c.lock.Lock()
	delete(c.data, key)
	c.lru.Remove(key)
	c.lock.Unlock()

	// using chan
	c.timingWheel.RemoveTimer(key)
}

func (c *Local) Get(ctx context.Context, key string) (value interface{}, ok bool) {
	value, ok = c.doGet(ctx, key)
	if ok {
		c.stat.Hit()
	} else {
		c.stat.Miss()
	}

	return value, ok
}

func (c *Local) doGet(ctx context.Context, key string) (value interface{}, ok bool) {
	c.lock.Lock()
	defer c.lock.Unlock()

	value, ok = c.data[key]
	if ok {
		c.lru.Add(key)
	}

	return value, ok
}

func (c *Local) onEvict(key string) {
	// already locked
	delete(c.data, key)
	c.timingWheel.RemoveTimer(key)
}

func (c *Local) size() int {
	c.lock.Lock()
	defer c.lock.Unlock()
	return len(c.data)
}
