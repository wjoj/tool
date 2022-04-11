package etcd

import (
	"context"
	"crypto/tls"
	"fmt"
	"time"

	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc/resolver"
)

type Account struct {
	User string
	Pass string
}

type EtcdClient struct {
	cli *clientv3.Client
}

func EtcdDialClient(endpoints []string, account *Account, tls *tls.Config) (*EtcdClient, error) {
	cfg := clientv3.Config{
		Endpoints:            endpoints,
		AutoSyncInterval:     time.Minute,
		DialTimeout:          5 * time.Second,
		DialKeepAliveTime:    5 * time.Second,
		DialKeepAliveTimeout: 5 * time.Second,
		RejectOldCluster:     true,
	}
	if account != nil {
		cfg.Username = account.User
		cfg.Password = account.Pass
	}
	if tls != nil {
		cfg.TLS = tls
	}
	cli, err := clientv3.New(cfg)
	if err != nil {
		return nil, err
	}
	return &EtcdClient{
		cli: cli,
	}, nil
}

func (c *EtcdClient) Watch(key string, addrFunc func(addr []resolver.Address)) {
	ctx, cancel := context.WithTimeout(c.cli.Ctx(), 3*time.Second)
	val, err := c.cli.Get(ctx, key, clientv3.WithPrefix())
	cancel()
	if err != nil {
		return
	}
	var addrList []resolver.Address
	for i := range val.Kvs {
		addrList = append(addrList, resolver.Address{
			Addr: string(val.Kvs[i].Value),
		})
	}
	addrFunc(addrList)
	rch := c.cli.Watch(clientv3.WithRequireLeader(c.cli.Ctx()), key, clientv3.WithPrefix())
	for ch := range rch {
		for _, ev := range ch.Events {
			addr := string(ev.Kv.Value)
			switch ev.Type {
			case mvccpb.PUT:
				addrList = append(addrList, resolver.Address{Addr: addr})
			case mvccpb.DELETE:
				for i, ad := range addrList {
					if ad.Addr == addr {
						addrList = append(addrList[:i], addrList[i+1:]...)
					}
				}

			}
		}
		addrFunc(addrList)
	}
}

func (c *EtcdClient) Put(key string, value string) (clientv3.LeaseID, error) {
	resp, err := c.cli.Grant(c.cli.Ctx(), 10)
	if err != nil {
		return clientv3.NoLease, err
	}

	lease := resp.ID
	_, err = c.cli.Put(c.cli.Ctx(), key, value, clientv3.WithLease(lease))
	return lease, err
}

func (c *EtcdClient) KeepAliveAsync(lease clientv3.LeaseID) error {
	ch, err := c.cli.KeepAlive(c.cli.Ctx(), lease)
	if err != nil {
		return err
	}
	for {
		isExit := false
		select {
		case _, ok := <-ch:
			fmt.Printf("\nKeepAlive:%v", ok)
			if !ok {
				c.cli.Revoke(c.cli.Ctx(), clientv3.LeaseID(lease))
				isExit = true
			}
		}
		if isExit {
			break
		}
	}
	return nil
}
