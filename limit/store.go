package limit

import (
	"context"

	"github.com/go-redis/redis/v8"
)

type Store interface {
	Eval(ctx context.Context, script string, keys []string, args ...interface{}) (interface{}, error)
	IsErrNil(err error) bool
	Ping() bool
}

type Client struct {
	*redis.Client
}

func (c *Client) Eval(ctx context.Context, script string, keys []string, args ...interface{}) (interface{}, error) {
	res := c.Client.Eval(ctx, script, keys, args...)
	return res.Val(), res.Err()
}

func (c *Client) IsErrNil(err error) bool {
	return err == redis.Nil
}

func (c *Client) Ping() bool {
	return c.Client.Ping(context.Background()).Err() == nil
}

type ClusterClient struct {
	*redis.ClusterClient
}

func (c *ClusterClient) Eval(ctx context.Context, script string, keys []string, args ...interface{}) (interface{}, error) {
	res := c.ClusterClient.Eval(ctx, script, keys, args...)
	return res.Val(), res.Err()
}

func (c *ClusterClient) IsErrNil(err error) bool {
	return err == redis.Nil
}

func (c *ClusterClient) Ping() bool {
	return c.ClusterClient.Ping(context.Background()).Err() == nil
}

func NewRedis(r *redis.Client) *Client {
	return &Client{
		r,
	}
}

func NewClusterClient(r *redis.ClusterClient) *ClusterClient {
	return &ClusterClient{
		r,
	}
}
