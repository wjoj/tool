package store

import (
	"errors"
	"time"

	"github.com/pangudashu/memcache"
)

type MemcacheConfig struct {
	Nodes []struct {
		Address  string        `json:"address" yaml:"address"`
		Weight   int           `json:"weight" yaml:"weight"`
		MaxConn  int           `json:"maxConn" yaml:"maxConn"`
		InitConn int           `json:"initConn" yaml:"initConn"`
		IdleTime time.Duration `json:"idleTime" yaml:"idleTime"`
	} `json:"nodes" yaml:"nodes"`
	RemoveBadNode bool          `json:"removeBadNode" yaml:"removeBadNode"`
	DialTimeout   time.Duration `json:"dialTimeout" yaml:"dialTimeout"`
	ReadTimeout   time.Duration `json:"readTimeout" yaml:"readTimeout"`
	WriteTimeout  time.Duration `json:"writeTimeout" yaml:"writeTimeout"`
}

func NewMemcache(cfg *MemcacheConfig) (*memcache.Memcache, error) {
	if len(cfg.Nodes) == 0 {
		return nil, errors.New("node information cannot be empty")
	}
	srcs := make([]*memcache.Server, len(cfg.Nodes))
	for _, cfg := range cfg.Nodes {
		srcs = append(srcs, &memcache.Server{
			Address:  cfg.Address,
			Weight:   cfg.Weight,
			MaxConn:  cfg.MaxConn,
			InitConn: cfg.InitConn,
			IdleTime: cfg.IdleTime,
		})
	}
	mc, err := memcache.NewMemcache(srcs)
	if err != nil {
		return nil, err
	}
	mc.SetRemoveBadServer(cfg.RemoveBadNode)
	mc.SetTimeout(cfg.DialTimeout, cfg.ReadTimeout, cfg.WriteTimeout)
	return mc, nil

}
