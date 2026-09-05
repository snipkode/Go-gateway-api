package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"go-enterprise-api/internal/domain/ratelimit"
)

const (
	rulesCacheKey = "ratelimit:rules"
	rulesCacheTTL = 30 * time.Second
)

// RateLimitEvaluator loads active rate limit rules from the source of truth
// (PostgreSQL) and caches them in Redis, then enforces the strictest rule
// that applies to the request subject. Admins can change rules at runtime via
// the API — no redeploy needed; the cache refreshes every few seconds.
type RateLimitEvaluator struct {
	rules    ratelimit.Repository
	rdb      *goredis.Client
	counter  ratelimit.Counter
	cache    cache
	cacheTTL time.Duration
	minRate  float64 // lower bound sanity guard (per second)
}

type cache interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value any, ttl time.Duration) error
}

var _ cache = (*redisCacheAdapter)(nil)

type redisCacheAdapter struct{ rdb *goredis.Client }

func (a *redisCacheAdapter) Get(ctx context.Context, key string) (string, error) {
	return a.rdb.Get(ctx, key).Result()
}
func (a *redisCacheAdapter) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	return a.rdb.Set(ctx, key, value, ttl).Err()
}

func NewRateLimitEvaluator(rules ratelimit.Repository, c *Client, counter ratelimit.Counter) *RateLimitEvaluator {
	return &RateLimitEvaluator{
		rules:    rules,
		rdb:      c.RDB(),
		counter:  counter,
		cache:    &redisCacheAdapter{rdb: c.RDB()},
		cacheTTL: rulesCacheTTL,
		minRate:  0,
	}
}

// Evaluate computes which rate limit rule is the strictest one that matches
// the subject, then increments the appropriate counter and reports whether
// the request may proceed.
//
// Layering (from slowest to most specific) is respected: the rule with the
// smallest requests/second wins, so a per-route login rule of 10/min defeats
// a global 10000/min limit — matching "yang paling ketat berlaku".
func (e *RateLimitEvaluator) Evaluate(ctx context.Context, s ratelimit.Subject) (*ratelimit.Decision, error) {
	rules, err := e.loadRules(ctx)
	if err != nil {
		return nil, err
	}

	type candidate struct {
		rule ratelimit.Rule
		key  string
	}
	var candidates []candidate

	for _, r := range rules {
		if !r.Enabled || r.Requests <= 0 || r.WindowSeconds <= 0 {
			continue
		}
		key, ok := counterKeyForScope(r.Scope, r.Identifier, s)
		if !ok {
			continue
		}
		candidates = append(candidates, candidate{rule: r, key: key})
	}

	if len(candidates) == 0 {
		return nil, nil // no rule applies: unlimited
	}

	// Strictest = smallest requests per second.
	sort.Slice(candidates, func(i, j int) bool {
		return rate(candidates[i].rule) < rate(candidates[j].rule)
	})

	chosen := candidates[0]
	window := time.Duration(chosen.rule.WindowSeconds) * time.Second
	count, err := e.counter.Consume(ctx, chosen.key, window)
	if err != nil {
		return nil, err
	}

	rejected := count > chosen.rule.Requests
	remaining := chosen.rule.Requests - count
	if remaining < 0 {
		remaining = 0
	}
	retryAfter := time.Duration(0)
	if rejected {
		retryAfter, _ = e.counter.RetryAfter(ctx, chosen.key, window)
	}

	return &ratelimit.Decision{
		Rule:       chosen.rule,
		Remaining:  remaining,
		RetryAfter: retryAfter,
		AppliedKey: chosen.key,
		Rejected:   rejected,
	}, nil
}

// loadRules returns active rules, using the Redis cache on the fast path.
func (e *RateLimitEvaluator) loadRules(ctx context.Context) ([]ratelimit.Rule, error) {
	if raw, err := e.cache.Get(ctx, rulesCacheKey); err == nil && raw != "" {
		var rules []ratelimit.Rule
		if json.Unmarshal([]byte(raw), &rules) == nil {
			return rules, nil
		}
	}
	rules, err := e.rules.List(ctx, false)
	if err != nil {
		return nil, err
	}
	if payload, err := json.Marshal(rules); err == nil {
		_ = e.cache.Set(ctx, rulesCacheKey, payload, e.cacheTTL)
	}
	return rules, nil
}

// counterKeyForScope produces a stable Redis key per identity (scope) and
// tells whether the rule applies to the subject.
func counterKeyForScope(scope, identifier string, s ratelimit.Subject) (string, bool) {
	switch scope {
	case ratelimit.ScopeGlobal:
		return "global", true
	case ratelimit.ScopeIP:
		if s.IP == "" {
			return "", false
		}
		return "ip:" + s.IP, true
	case ratelimit.ScopeUser:
		if s.UserID == 0 {
			return "", false
		}
		return fmt.Sprintf("user:%d", s.UserID), true
	case ratelimit.ScopeRole:
		for _, rl := range s.Roles {
			if rl == identifier {
				return "role:" + identifier, true
			}
		}
		return "", false
	case ratelimit.ScopeRoute:
		if s.Route == "" || (identifier != "" && s.Route != identifier) {
			return "", false
		}
		if identifier == "" {
			return "route:" + s.Route, true
		}
		return "route:" + identifier, true
	case ratelimit.ScopeAPIKey:
		if s.APIKey == "" {
			return "", false
		}
		return "apikey:" + identifier, true
	}
	return "", false
}

func rate(r ratelimit.Rule) float64 {
	if r.WindowSeconds <= 0 {
		return math.MaxFloat64
	}
	return float64(r.Requests) / float64(r.WindowSeconds)
}

var _ ratelimit.Evaluator = (*RateLimitEvaluator)(nil)
