package rpc

import (
	"fmt"
	"strings"

	"google.golang.org/grpc/resolver"
)

type Discover interface {
	Get(key string) ([]resolver.Address, error)
	Put(key string, value string) (int64, error)
	Register(key string, value string) error
	Watch(key string, addrFunc func(addrs []resolver.Address))
	KeepAliveAsync() error
	Close()
}

var discoverContainer Discover

func SetDiscoverContainer(dis Discover) {
	discoverContainer = dis
}

func init() {
	resolver.Register(&discoverBuilder{})
}

type discoverBuilder struct{}

func (b *discoverBuilder) Build(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOptions) (
	resolver.Resolver, error) {
	prefix := "/" + target.URL.Scheme + "/" + strings.TrimPrefix(target.URL.Path, "/") + "/"
	addrs, err := discoverContainer.Get(prefix)
	if err != nil {
		return nil, err
	}
	cc.UpdateState(resolver.State{
		Addresses: addrs,
	})
	go discoverContainer.Watch(prefix, func(addrs []resolver.Address) {
		cc.UpdateState(resolver.State{
			Addresses: addrs,
		})
	})
	return &nopResolver{cc: cc}, nil
}

func (b *discoverBuilder) Scheme() string {
	return DiscoverScheme
}

func (b *discoverBuilder) Close() {
	discoverContainer.Close()
}

func DiscoverRegister(listenOn string, name string) error {
	val := figureOutListenOn(listenOn)
	key := fmt.Sprintf("/%v/%v/%v", DiscoverScheme, name, val)

	if err := discoverContainer.Register(key, val); err != nil {
		return err
	}

	go discoverContainer.KeepAliveAsync()
	return nil
}
