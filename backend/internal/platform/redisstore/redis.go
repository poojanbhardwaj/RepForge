package redisstore

import (
	"context"

	"github.com/redis/go-redis/v9"
)

type Client struct {
	client *redis.Client
}

func New(url string) (*Client, error) {
	options, err := redis.ParseURL(url)
	if err != nil {
		return nil, err
	}
	return &Client{client: redis.NewClient(options)}, nil
}

func (c *Client) Ping(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}

func (c *Client) Close() error {
	return c.client.Close()
}
