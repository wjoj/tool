package rpc

import (
	"context"
	"fmt"
	"testing"

	demosrv "github.com/wjoj/tool/rpc/demorpc"
	demorpc "github.com/wjoj/tool/rpc/demorpc/pb"
	"google.golang.org/grpc"
)

func TestCli(t *testing.T) {
	cfg := &ConfigClient{
		ServiceName:  "demo",
		NonBlock:     true,
		BalancerName: "round_robin",
		ConTimeout:   5,
		Endpoints: []string{
			"127.0.0.1:5202",
			"127.0.0.1:5200",
			"127.0.0.1:5201",
		},
	}
	err := cfg.Start(func(conn *grpc.ClientConn) {
		cli := demorpc.NewHelloClient(conn)
		for i := 0; i < 10; i++ {
			req, err := cli.Info(context.Background(), &demorpc.Request{
				Name: fmt.Sprintf("i:%+v", i),
			})
			if err != nil {
				fmt.Printf("\nrpc client req err:%v", err)
			} else {
				fmt.Printf("\nrpc client req:%+v", req)
			}
		}
	})
	fmt.Printf("\ngrpc client err:%v", err)
}

func TestSrv(t *testing.T) {
	adds := []int{5200, 5201, 5202}
	for _, add := range adds {
		cfg := &ConfigService{
			Port:              add,
			ServiceName:       "demo",
			ConnectionTimeout: 5,
		}
		cfg.Start(func(srv *grpc.Server) {
			demorpc.RegisterHelloServer(srv, &demosrv.Demo{
				Addr: fmt.Sprintf("%v", add),
			})
		}, func(err error) {
			fmt.Printf("\ngrpc service error:%v", err)
		})
	}
	fmt.Printf("\n已启动")
	select {}
}
