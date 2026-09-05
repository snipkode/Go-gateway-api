package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Client wraps the go-redis client so the rest of the app depends on a small
// surface instead of the concrete library.
type Client struct {
	rdb *redis.Client
}

func NewClient(ctx context.Context, addr, password string, db int) (*Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     password,
		DB:           db,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	})
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("ping redis: %w", err)
	}
	return &Client{rdb: rdb}, nil
}

func (c *Client) Close() error       { return c.rdb.Close() }
func (c *Client) RDB() *redis.Client { return c.rdb }
