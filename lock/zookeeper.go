package lock

import (
	"github.com/samuel/go-zookeeper/zk"
)

type ZookeeperLock struct {
	lock *zk.Lock
}

func NewZookeeperLock(c *zk.Conn, key string) *ZookeeperLock {
	l := zk.NewLock(c, "/"+key, zk.WorldACL(zk.PermAll))
	return &ZookeeperLock{
		lock: l,
	}
}

func (z *ZookeeperLock) Lock() error {
	return z.lock.Lock()
}

func (z *ZookeeperLock) Unlock() error {
	return z.lock.Unlock()
}
