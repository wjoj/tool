package rpc

import "context"

const (
	appKey   = "app"
	tokenKey = "token"
)

type Auth struct {
	App, Token string
}

func (a *Auth) GetRequestMetadata(ctx context.Context, uri ...string) (map[string]string, error) {
	return map[string]string{
		appKey:   a.App,
		tokenKey: a.Token,
	}, nil
}

func (a *Auth) RequireTransportSecurity() bool {
	return false
}
