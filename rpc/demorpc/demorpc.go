package demorpc

import (
	"context"

	demorpc "github.com/wjoj/tool/rpc/demorpc/pb"
)

type Demo struct {
	Addr string
}

func (s *Demo) Info(ctx context.Context, req *demorpc.Request) (*demorpc.Response, error) {
	return &demorpc.Response{
		Code: 1,
		Data: &demorpc.Info{
			Name: req.Name + "  (" + s.Addr + ")",
		},
	}, nil
}
