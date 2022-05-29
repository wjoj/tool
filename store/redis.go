package store

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

type ConfigRedis struct {
	Addrs        []string      `json:"addrs" yaml:"addrs"`
	IsCluster    bool          `json:"isCluster" yaml:"isCluster"`
	Username     string        `json:"username" yaml:"username"`
	Password     string        `json:"password" yaml:"password"`
	PoolSize     int           `json:"poolSize" yaml:"poolSize"`
	MinIdleConns int           `json:"minIdleConns" yaml:"minIdleConns"`
	MaxConnAge   time.Duration `json:"maxConnAge" yaml:"maxConnAge"`
	PoolTimeout  time.Duration `json:"poolTimeout" yaml:"poolTimeout"`
	IdleTimeout  time.Duration `json:"idleTimeout" yaml:"idleTimeout"`
}

func (c *ConfigRedis) String() string {
	msg := fmt.Sprintln("Redis info：")
	msg += fmt.Sprintln("\tAddrs：", c.Addrs)
	msg += fmt.Sprintln("\tIsCluster：", c.IsCluster)
	msg += fmt.Sprintln("\tUsername：", c.Username, " Password：", c.Password)
	msg += fmt.Sprintln("\tPoolSize：", c.PoolSize, " PoolTimeout：", c.PoolTimeout)
	msg += fmt.Sprintln("\tMinIdleConns：", c.MinIdleConns, " MaxConnAge：", c.MaxConnAge)
	msg += fmt.Sprintln("\tIdleTimeout：", c.IdleTimeout)
	return ""
}

func NewRedis(cfg *ConfigRedis) (RedisClient, error) {
	if len(cfg.Addrs) == 0 {
		return nil, fmt.Errorf("redis adds can't be empty")
	}
	var cli RedisClient
	if cfg.IsCluster {
		cli = redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:       cfg.Addrs,
			Username:    cfg.Username,
			Password:    cfg.Password,
			PoolSize:    cfg.PoolSize,
			MaxConnAge:  cfg.MaxConnAge,
			PoolTimeout: cfg.PoolTimeout,
			IdleTimeout: cfg.IdleTimeout,
		})
	} else {
		cli = redis.NewClient(&redis.Options{
			Addr:        cfg.Addrs[0],
			Username:    cfg.Username,
			Password:    cfg.Password,
			PoolSize:    cfg.PoolSize,
			MaxConnAge:  cfg.MaxConnAge,
			PoolTimeout: cfg.PoolTimeout,
			IdleTimeout: cfg.IdleTimeout,
		})
	}
	_, err := cli.Ping(context.Background()).Result()
	if err != nil {
		return nil, err
	}
	return cli, nil
}

var Redis RedisClient

func SetGlobalRedis(cfg *ConfigRedis) error {
	cli, err := NewRedis(cfg)
	if err != nil {
		return err
	}
	Redis = cli
	return nil
}

func GlobalRedis() RedisClient {
	return Redis
}
