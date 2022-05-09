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

	"github.com/wjoj/tool/monitoring"
	"github.com/wjoj/tool/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

func serviceConfig(balancing string, srvName string) string {
	if len(balancing) == 0 {
		balancing = "round_robin"
	}
	return fmt.Sprintf(`{
		"loadBalancingPolicy": "%v",
		"healthCheckConfig": {
			"serviceName": "%s"
		}
	}`, balancing, srvName)
}

const (
	DirectScheme    = "direct"   //
	DiscoverScheme  = "discover" //
	K8sScheme       = "k8s"
	EndpointSepChar = ','
	subsetSize      = 30
)

type ConfigClient struct {
	ServiceName  string
	NonBlock     bool
	BalancerName string   ///round_robin pick_first
	ConTimeout   int64    `json:"conTimeout" yaml:"conTimeout"`
	Endpoints    []string `json:"endpoints" yaml:"endpoints"`
	Target       string
	Etcd         *ConfigEtcd
	Auth         *Auth
	Trace        *trace.TracerCfg
	Prom         *monitoring.ConfigPrometheus
}

func (c *ConfigClient) BuildTarget() (string, error) {
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

func (c *ConfigClient) Start(connFunc func(conn *grpc.ClientConn)) (err error) {
	if c.Etcd != nil {
		discoverContainer, err = NewEtcd(c.Etcd, c.ServiceName)
	}
	target, err := c.BuildTarget()
	var opts []grpc.DialOption
	opts = append([]grpc.DialOption(nil), grpc.WithTransportCredentials(insecure.NewCredentials())) //grpc.WithInsecure()

	if c.Auth != nil {
		if len(c.Auth.Token) == 0 || len(c.Auth.App) == 0 {
			return fmt.Errorf("token or app is empty")
		}
		opts = append(opts, grpc.WithPerRPCCredentials(c.Auth))
	}

	if c.NonBlock {
		opts = append(opts, grpc.WithBlock()) //NonBlock //创建conn报错返回
	}
	if len(c.BalancerName) == 0 {
		c.BalancerName = "round_robin"
	}
	if c.ConTimeout == 0 {
		c.ConTimeout = 1
	}

	opts = append(opts, grpc.WithDefaultServiceConfig(serviceConfig(c.BalancerName, c.ServiceName))) //获取注册平衡器 balancer.Register
	clientInterceptors := []grpc.UnaryClientInterceptor{}
	streamInterceptors := []grpc.StreamClientInterceptor{}
	if c.Trace != nil {
		tracer, err := trace.NewTracer(c.Trace, c.ServiceName)
		if err != nil {
			return err
		}
		clientInterceptors = append(clientInterceptors, trace.TracerGrpcClientUnaryInterceptor(tracer))
		streamInterceptors = append(streamInterceptors, trace.TracerGrpcStreamClientUnaryInterceptor(tracer))
	}
	if c.Prom != nil {
		if c.Prom.Namespace == "" {
			c.Prom.Namespace = "rpc-client"
		}
		clientInterceptors = append(clientInterceptors, monitoring.UnaryRPCClientPrometheusInterceptor(monitoring.RPCPrometheusStart(c.Prom)))
	}
	if len(clientInterceptors) != 0 {
		opts = append(opts, grpc.WithChainUnaryInterceptor(clientInterceptors...)) //拦截器 路由追踪 监控 断路器 控制超时链接器
	}
	if len(streamInterceptors) != 0 {
		opts = append(opts, grpc.WithChainStreamInterceptor(streamInterceptors...)) //流路由
	}

	timeCtx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(c.ConTimeout))
	defer cancel()
	// target = "passthrough:127.0.0.1:9520"
	conn, err := grpc.DialContext(timeCtx, target, opts...)
	if err != nil {
		return
	}
	connFunc(conn)
	return
}

type ConfigService struct {
	Port              int         `json:"port" yaml:"port"`
	ServiceName       string      `json:"serviceName" yaml:"serviceName"`
	ConnectionTimeout int         `json:"connectionTimeout" yaml:"connectionTimeout"` //s
	Etcd              *ConfigEtcd `json:"etcd" yaml:"etcd"`
	Auth              *Auth       `json:"auth" yaml:"auth"`
	Trace             *trace.TracerCfg
	Prom              *monitoring.ConfigPrometheus
}

func (c *ConfigService) Start(regiser func(srv *grpc.Server), errFunc func(err error)) {
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

	unaryInterceptors := []grpc.UnaryServerInterceptor{}
	streamInterceptors := []grpc.StreamServerInterceptor{}
	if c.Trace != nil {
		tracer, err := trace.NewTracer(c.Trace, c.ServiceName)
		if err != nil {
			errFunc(err)
			return
		}
		unaryInterceptors = append(unaryInterceptors, trace.TracerGrpcServerUnaryInterceptor(tracer))
		streamInterceptors = append(streamInterceptors, trace.TracerGrpcStreamServerUnaryInterceptor(tracer))
	}
	if c.Prom != nil {
		if c.Prom.Namespace == "" {
			c.Prom.Namespace = "rpc-server"
		}
		unaryInterceptors = append(unaryInterceptors, monitoring.UnaryRPCServerPrometheusInterceptor(monitoring.RPCPrometheusStart(c.Prom)))
	}
	options := []grpc.ServerOption{
		grpc.ConnectionTimeout(time.Duration(c.ConnectionTimeout) * time.Second),
	}
	if len(unaryInterceptors) != 0 {
		options = append(options, grpc.ChainUnaryInterceptor(unaryInterceptors...))
	}
	if len(streamInterceptors) != 0 {
		options = append(options, grpc.ChainStreamInterceptor(streamInterceptors...))
	}

	s := grpc.NewServer(options...)

	healthcheck := health.NewServer()
	healthcheck.SetServingStatus(c.ServiceName, healthpb.HealthCheckResponse_SERVING)
	healthpb.RegisterHealthServer(s, healthcheck)

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
