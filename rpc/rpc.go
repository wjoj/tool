package rpc

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"os"
	"strings"
	"time"

	"google.golang.org/grpc"
)

const (
	DirectScheme    = "direct"
	DiscoverScheme  = "discover"
	EndpointSepChar = ','
	subsetSize      = 30
)

type ClientGRPC struct {
	ServiceName  string
	NonBlock     bool
	BalancerName string
	ConTimeout   int64    `json:"conTimeout" yaml:"conTimeout"`
	Endpoints    []string `json:"endpoints" yaml:"endpoints"`
	Target       string
	Etcd         *ConfigEtcd
	Auth         *Auth
}

func (c *ClientGRPC) BuildTarget() (string, error) {
	if len(c.Target) != 0 {
		return c.Target, nil
	}
	if len(c.Endpoints) > 0 {
		return BuildDirectTarget(c.Endpoints), nil
	}
	if c.Etcd == nil {
		return "", errors.New("etcd is empty")
	}
	return BuildDiscoverTarget(c.Etcd.Endpoints, c.Etcd.serviceName), nil
}

func (c *ClientGRPC) Load(connFunc func(conn *grpc.ClientConn)) (err error) {
	if c.Etcd != nil {
		discoverContainer, err = NewEtcd(c.Etcd, c.ServiceName)
	}
	target, err := c.BuildTarget()
	var opts []grpc.DialOption
	if c.Auth != nil {
		if len(c.Auth.Token) == 0 || len(c.Auth.App) == 0 {
			return fmt.Errorf("token or app ")
		}
		opts = append(opts, grpc.WithPerRPCCredentials(c.Auth))
	}

	// opts = append(opts, grpc.WithTimeout(time.Second))
	if c.NonBlock {
		opts = append(opts, grpc.WithBlock()) //NonBlock //创建conn报错返回
	}
	if len(c.BalancerName) != 0 {
		opts = append(opts, grpc.WithBalancerName(c.BalancerName)) //获取注册平衡器 balancer.Register
	}

	// opts = append(opts, grpc.WithChainUnaryInterceptor())  //拦截器 路由追踪 监控 断路器 控制超时链接器
	// opts = append(opts, grpc.WithChainStreamInterceptor()) //流路由
	opts = append([]grpc.DialOption(nil), grpc.WithInsecure())
	timeCtx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	// target = "passthrough:127.0.0.1:9520"
	conn, err := grpc.DialContext(timeCtx, target, opts...)
	if err != nil {
		return
	}
	connFunc(conn)
	return
}

type ServiceRPC struct {
	Port              int
	ServiceName       string
	ConnectionTimeout int //s
	Etcd              *ConfigEtcd
	Auth              *Auth
}

func (c *ServiceRPC) Load(regiser func(srv *grpc.Server), errFunc func(err error)) {
	if c.Etcd != nil {
		disContainer, err := NewEtcd(c.Etcd, c.ServiceName)
		if err != nil {
			errFunc(err)
			return
		}
		discoverContainer = disContainer
		DiscoverRegister(figureOutListenOn(fmt.Sprintf("0.0.0.0:%v", c.Port)), c.ServiceName)
	}
	lis, err := net.Listen("tcp", fmt.Sprintf(":%v", c.Port))
	if err != nil {
		errFunc(err)
		return
	}
	s := grpc.NewServer(grpc.ConnectionTimeout(time.Duration(c.ConnectionTimeout) * time.Second))
	regiser(s)
	go func() {
		if err = s.Serve(lis); err != nil {
			errFunc(err)
			return
		}
	}()
}

func subset(set []string, sub int) []string {
	rand.Shuffle(len(set), func(i, j int) {
		set[i], set[j] = set[j], set[i]
	})
	if len(set) <= sub {
		return set
	}

	return set[:sub]
}

const (
	allEths  = "0.0.0.0"
	envPodIp = "POD_IP"
)

func isEthDown(f net.Flags) bool {
	return f&net.FlagUp != net.FlagUp
}

func isLoopback(f net.Flags) bool {
	return f&net.FlagLoopback == net.FlagLoopback
}

// InternalIp returns an internal ip.
func InternalIp() string {
	infs, err := net.Interfaces()
	if err != nil {
		return ""
	}

	for _, inf := range infs {
		if isEthDown(inf.Flags) || isLoopback(inf.Flags) {
			continue
		}

		addrs, err := inf.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
				if ipnet.IP.To4() != nil {
					return ipnet.IP.String()
				}
			}
		}
	}

	return ""
}

func figureOutListenOn(listenOn string) string {
	fields := strings.Split(listenOn, ":")
	if len(fields) == 0 {
		return listenOn
	}

	host := fields[0]
	if len(host) > 0 && host != allEths {
		return listenOn
	}

	ip := os.Getenv(envPodIp)
	if len(ip) == 0 {
		ip = InternalIp()
	}
	if len(ip) == 0 {
		return listenOn
	}

	return strings.Join(append([]string{ip}, fields[1:]...), ":")
}
