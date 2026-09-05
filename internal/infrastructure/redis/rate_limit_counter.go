package redis

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go-enterprise-api/internal/domain/ratelimit"

	"github.com/redis/go-redis/v9"
)

const rateLimitKey = "ratelimit:%s"

// RateLimitCounter implements a fixed-window counter in Redis. On every
// request it increments the key and, when the key does not exist, sets the
// TTL equal to the rule's window. The counter is atomic (INCR) and therefore
// safe under concurrency.
type RateLimitCounter struct {
	rdb *redis.Client
}

func NewRateLimitCounter(c *Client) *RateLimitCounter {
	return &RateLimitCounter{rdb: c.RDB()}
}

// Consume increments the counter for key and returns the new count. The
// first increment in a window sets the key's TTL, so counters self-expire.
func (c *RateLimitCounter) Consume(ctx context.Context, key string, window time.Duration) (int64, error) {
	redisKey := fmt.Sprintf(rateLimitKey, key)
	count, err := c.rdb.Incr(ctx, redisKey).Result()
	if err != nil {
		return 0, err
	}
	if count == 1 {
		_ = c.rdb.Expire(ctx, redisKey, window).Err()
	}
	return count, nil
}

func (c *RateLimitCounter) Reset(ctx context.Context, key string) error {
	return c.rdb.Del(ctx, fmt.Sprintf(rateLimitKey, key)).Err()
}

func (c *RateLimitCounter) RetryAfter(ctx context.Context, key string, window time.Duration) (time.Duration, error) {
	ttl, err := c.rdb.TTL(ctx, fmt.Sprintf(rateLimitKey, key)).Result()
	if errors.Is(err, redis.Nil) || ttl < 0 {
		return window, nil
	}
	if err != nil {
		return window, err
	}
	return ttl, nil
}

var _ ratelimit.Counter = (*RateLimitCounter)(nil)
