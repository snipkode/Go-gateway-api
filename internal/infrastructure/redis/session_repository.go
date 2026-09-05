package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"go-enterprise-api/internal/domain/session"

	"github.com/redis/go-redis/v9"
)

const sessionKey = "session:%s"

type SessionRepository struct {
	rdb *redis.Client
	ttl time.Duration
}

func NewSessionRepository(c *Client, ttl time.Duration) *SessionRepository {
	return &SessionRepository{rdb: c.RDB(), ttl: ttl}
}

func (r *SessionRepository) Create(ctx context.Context, s session.Session) error {
	payload, err := json.Marshal(s)
	if err != nil {
		return err
	}
	ttl := r.ttl
	if !s.ExpiresAt.IsZero() {
		ttl = time.Until(s.ExpiresAt)
		if ttl <= 0 {
			ttl = r.ttl
		}
	}
	return r.rdb.Set(ctx, fmt.Sprintf(sessionKey, s.SessionID), payload, ttl).Err()
}

func (r *SessionRepository) Get(ctx context.Context, sessionID string) (session.Session, error) {
	raw, err := r.rdb.Get(ctx, fmt.Sprintf(sessionKey, sessionID)).Bytes()
	if errors.Is(err, redis.Nil) {
		return session.Session{}, session.ErrNotFound
	}
	if err != nil {
		return session.Session{}, err
	}
	var s session.Session
	if err := json.Unmarshal(raw, &s); err != nil {
		return session.Session{}, err
	}
	return s, nil
}

func (r *SessionRepository) UserID(ctx context.Context, sessionID string) (int64, error) {
	s, err := r.Get(ctx, sessionID)
	if err != nil {
		return 0, err
	}
	return s.UserID, nil
}

func (r *SessionRepository) Delete(ctx context.Context, sessionID string) error {
	return r.rdb.Del(ctx, fmt.Sprintf(sessionKey, sessionID)).Err()
}

func (r *SessionRepository) DeleteByUser(ctx context.Context, userID int64) error {
	iter := r.rdb.Scan(ctx, 0, "session:*", 100).Iterator()
	for iter.Next(ctx) {
		raw, err := r.rdb.Get(ctx, iter.Val()).Bytes()
		if err != nil {
			continue
		}
		var s session.Session
		if json.Unmarshal(raw, &s) == nil && s.UserID == userID {
			_ = r.rdb.Del(ctx, iter.Val()).Err()
		}
	}
	return iter.Err()
}

func (r *SessionRepository) Touch(ctx context.Context, sessionID string) error {
	key := fmt.Sprintf(sessionKey, sessionID)
	return r.rdb.Expire(ctx, key, r.ttl).Err()
}

var _ session.Repository = (*SessionRepository)(nil)
