package rpc

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"sync"
	"time"

	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"

	"google.golang.org/grpc/resolver"
)

type Account struct {
	User string
	Pass string
}

type ConfigEtcd struct {
	Endpoints          []string
	UserName           string
	Pass               string `json:",optional"`
	CertFile           string `json:",optional"`
	CertKeyFile        string `json:",optional=CertFile"`
	CACertFile         string `json:",optional=CertFile"`
	InsecureSkipVerify bool   `json:",optional"`
	serviceName        string
}

type ClientEtcd struct {
	cli   *clientv3.Client
	adds  []resolver.Address
	lock  sync.Mutex
	lease clientv3.LeaseID
	key   string
	value string
}

func NewEtcd(cfg *ConfigEtcd, serviceName string) (Discover, error) {
	cf := clientv3.Config{
		Endpoints:            cfg.Endpoints,
		AutoSyncInterval:     time.Minute,
		DialTimeout:          5 * time.Second,
		DialKeepAliveTime:    5 * time.Second,
		DialKeepAliveTimeout: 5 * time.Second,
		RejectOldCluster:     true,
	}
	cfg.serviceName = serviceName
	if len(cfg.UserName) != 0 && len(cfg.Pass) != 0 {
		cf.Username = cfg.UserName
		cf.Password = cfg.Pass
	}
	if len(cfg.CACertFile) != 0 && len(cfg.CertKeyFile) != 0 && len(cfg.CertFile) != 0 {
		cert, err := tls.LoadX509KeyPair(cfg.CACertFile, cfg.CertKeyFile)
		if err != nil {
			return nil, err
		}

		caData, err := ioutil.ReadFile(cfg.CertFile)
		if err != nil {
			return nil, err
		}

		pool := x509.NewCertPool()
		pool.AppendCertsFromPEM(caData)
		cf.TLS = &tls.Config{
			Certificates:       []tls.Certificate{cert},
			RootCAs:            pool,
			InsecureSkipVerify: cfg.InsecureSkipVerify,
		}
	}
	cli, err := clientv3.New(cf)
	if err != nil {
		return nil, err
	}

	return &ClientEtcd{
		cli: cli,
	}, nil
}

func (c *ClientEtcd) Get(key string) ([]resolver.Address, error) {
	ctx, cancel := context.WithTimeout(c.cli.Ctx(), 3*time.Second)
	val, err := c.cli.Get(ctx, key, clientv3.WithPrefix())
	cancel()
	if err != nil {
		return nil, err
	}
	var addrList []resolver.Address
	for i := range val.Kvs {
		addrList = append(addrList, resolver.Address{
			Addr: string(val.Kvs[i].Value),
		})
	}
	c.add(addrList)
	return addrList, nil
}

func (c *ClientEtcd) Put(key string, value string) (int64, error) {
	resp, err := c.cli.Grant(c.cli.Ctx(), 10)
	if err != nil {
		return int64(clientv3.NoLease), err
	}
	lease := resp.ID
	_, err = c.cli.Put(c.cli.Ctx(), key, value, clientv3.WithLease(lease))
	if err != nil {
		return 0, err
	}
	c.lease = lease
	return int64(lease), nil
}

func (c *ClientEtcd) Watch(key string, addrFunc func(addrs []resolver.Address)) {
	rch := c.cli.Watch(clientv3.WithRequireLeader(c.cli.Ctx()), key, clientv3.WithPrefix())
	for ch := range rch {
		var addrList []resolver.Address
		var rmList []resolver.Address
		for _, ev := range ch.Events {
			addr := string(ev.Kv.Value)
			switch ev.Type {
			case mvccpb.PUT:
				addrList = append(addrList, resolver.Address{Addr: addr})
			case mvccpb.DELETE:
				rmList = append(rmList, resolver.Address{Addr: addr})
			}
		}
		if len(addrList) != 0 {
			c.add(addrList)
		}
		if len(rmList) != 0 {
			c.remove(rmList)
		}
		addrFunc(c.get())
	}
}

func (c *ClientEtcd) Register(key string, value string) error {
	_, err := c.Put(key, value)
	if err != nil {
		return err
	}
	c.key = key
	c.value = value
	return nil
}

func (c *ClientEtcd) Reconnect() {
	if err := c.Register(c.key, c.value); err == nil {
		go c.KeepAliveAsync()
	} else {

	}

}

func (c *ClientEtcd) KeepAliveAsync() error {
	ch, err := c.cli.KeepAlive(c.cli.Ctx(), c.lease)
	if err != nil {
		return err
	}
	for {
		isExit := false
		select {
		case _, ok := <-ch:
			fmt.Printf("\nKeepAlive:%v", ok)
			if !ok {
				c.Reconnect()
				isExit = true
			}
		}
		if isExit {
			break
		}
	}
	return nil
}

func (c *ClientEtcd) Close() {
	if c.cli == nil {
		return
	}
	c.cli.Close()
}

func (c *ClientEtcd) add(addrs []resolver.Address) {
	c.lock.Lock()
	defer c.lock.Unlock()
	for _, add := range addrs {
		eq := false
		for _, a := range c.adds {
			if add.Addr == a.Addr {
				eq = true
				break
			}
		}
		if !eq {
			c.adds = append(c.adds, add)
		}
	}
}

func (c *ClientEtcd) remove(addrs []resolver.Address) {
	c.lock.Lock()
	defer c.lock.Unlock()

	nadds := []resolver.Address{}
	for _, a := range c.adds {
		eq := false
		for _, add := range addrs {
			if add.Addr == a.Addr {
				eq = true
				break
			}
		}
		if !eq {
			nadds = append(nadds, a)
		}
	}
	c.adds = nadds
}

func (c *ClientEtcd) get() []resolver.Address {
	c.lock.Lock()
	defer c.lock.Unlock()
	return c.adds
}
