package lock

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis"
)

type RedisClient interface {
	SetNX(key string, value interface{}, expiration time.Duration) *redis.BoolCmd
	Expire(key string, expiration time.Duration) *redis.BoolCmd
	Subscribe(channels ...string) *redis.PubSub
	Publish(channel string, message interface{}) *redis.IntCmd
	Eval(script string, keys []string, args ...interface{}) *redis.Cmd
	EvalSha(sha1 string, keys []string, args ...interface{}) *redis.Cmd
	ScriptExists(hashes ...string) *redis.BoolSliceCmd
	ScriptLoad(script string) *redis.StringCmd
}

func NewRedis() *redis.Client {
	red := redis.NewClient(&redis.Options{
		Addr:     "127.0.0.1:6379",
		Password: "",
		DB:       0,
	})
	// clis := redis.NewClusterClient(&redis.ClusterOptions{})

	_, err := red.Ping().Result()
	if err != nil {

	}
	return red
}

type RedisLock struct {
	cli        RedisClient // RedisClient
	key        string
	value      string
	expiration time.Duration
	cancelFunc context.CancelFunc
}

func NewRedisLock(cli *redis.Client, key string, value string, expiration time.Duration) *RedisLock {
	return &RedisLock{
		cli:        cli,
		key:        key,
		value:      value,
		expiration: expiration,
	}
}

func (r *RedisLock) contract(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(r.expiration / 2)
		for {
			select {
			case <-ticker.C:
				r.cli.Expire(r.key, r.expiration)
			case <-ctx.Done():
				ticker.Stop()
				return
			}
		}
	}()
}

func (r *RedisLock) TryLock() (bool, error) {
	is, err := r.cli.SetNX(r.key, r.value, r.expiration).Result()
	if err != nil {
		return false, err
	}
	if is {
		ctx, cancel := context.WithCancel(context.Background())
		r.cancelFunc = cancel
		r.contract(ctx)
	}
	return is, nil
}

func (r *RedisLock) subscribe() error {
	sub := r.cli.Subscribe(subPubTopic(r.key))
	_, err := sub.Receive()
	if err != nil {
		return err
	}
	<-sub.Channel()
	return nil
}

func (r *RedisLock) subscribeWithTimeout(d time.Duration) error {
	timeNow := time.Now()
	pubSub := r.cli.Subscribe(subPubTopic(r.key))
	_, err := pubSub.ReceiveTimeout(d)
	if err != nil {
		return err
	}
	deltaTime := time.Since(timeNow) - d
	select {
	case <-pubSub.Channel():
		return nil
	case <-time.After(deltaTime):
		return fmt.Errorf("timeout")
	}
}

func (r *RedisLock) publish() error {
	err := r.cli.Publish(subPubTopic(r.key), "release").Err()
	if err != nil {
		return err
	}
	return nil
}

func (r *RedisLock) Lock() error {
	for {
		is, err := r.TryLock()
		if err != nil {
			return err
		}
		if is {
			return nil
		}
		if err := r.subscribe(); err != nil {
			return err
		}
	}
}

func (r *RedisLock) LockWithTimeout(d time.Duration) error {
	now := time.Now()
	for {
		is, err := r.TryLock()
		if err != nil {
			return err
		}
		if is {
			return nil
		}
		if err := r.subscribeWithTimeout(d - time.Since(now)); err != nil {
			return err
		}
	}
}

func (r *RedisLock) LockSpin(spin int) error {
	for i := 0; i < spin; i++ {
		success, err := r.TryLock()
		if err != nil {
			return err
		}
		if success {
			return nil
		}
		time.Sleep(time.Millisecond * 100)
	}
	return fmt.Errorf("max spin reached")
}

func (r *RedisLock) Unlock() error {
	script := redis.NewScript(fmt.Sprintf(`if redis.call("get", KEYS[1]) == "%v" then return redis.call("del", KEYS[1]) else return 0 end`, r.value))
	res, err := script.Run(r.cli, []string{r.key}).Result()
	if err != nil {
		return err
	}

	if tmp, ok := res.(int64); ok {
		if tmp == 1 {
			if r.cancelFunc != nil {
				r.cancelFunc()
			}
			r.publish()
			return nil
		}
	}
	return fmt.Errorf("Unlock script fail: %v", r.key)
}

func subPubTopic(key string) string {
	return "{lock}_" + key
}
