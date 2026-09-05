package ratelimit

import (
	"context"
	"errors"
	"time"
)

var ErrNotFound = errors.New("rate limit rule not found")

const (
	ScopeGlobal = "global"
	ScopeIP     = "ip"
	ScopeUser   = "user"
	ScopeRole   = "role"
	ScopeRoute  = "route"
	ScopeAPIKey = "api_key"
)

type Rule struct {
	ID            int64      `json:"id"`
	Name          string     `json:"name"`
	Scope         string     `json:"scope"`
	Identifier    string     `json:"identifier"`
	Requests      int64      `json:"requests"`
	WindowSeconds int64      `json:"window_seconds"`
	Enabled       bool       `json:"enabled"`
	Priority      int        `json:"priority"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	DeletedAt     *time.Time `json:"deleted_at,omitempty"`
}

type CreateParams struct {
	Name          string
	Scope         string
	Identifier    string
	Requests      int64
	WindowSeconds int64
	Enabled       bool
	Priority      int
}

type UpdateParams struct {
	Name          *string
	Requests      *int64
	WindowSeconds *int64
	Enabled       *bool
	Priority      *int
}

// Repository is the source of truth for rules (PostgreSQL).
type Repository interface {
	List(ctx context.Context, includeDeleted bool) ([]Rule, error)
	GetByID(ctx context.Context, id int64) (Rule, error)
	Create(ctx context.Context, p CreateParams) (Rule, error)
	Update(ctx context.Context, id int64, p UpdateParams) (Rule, error)
	SoftDelete(ctx context.Context, id int64) error
	Restore(ctx context.Context, id int64) error
}

// Counter is the fast-path token storage (Redis).
type Counter interface {
	// Consume increments the counter for key and returns the new count. The
	// caller compares it against the limit: count <= limit passes, count > limit
	// is rejected. The first increment sets the window TTL.
	Consume(ctx context.Context, key string, window time.Duration) (int64, error)
	Reset(ctx context.Context, key string) error
	RetryAfter(ctx context.Context, key string, window time.Duration) (time.Duration, error)
}

// Subject describes what "the requester is" so rules can be matched and
// the strictest applicable rule enforced.
type Subject struct {
	IP     string
	UserID int64
	Roles  []string
	Route  string // e.g. "POST:/auth/login"
	APIKey string
}

// Evaluator loads rules and applies the strictest matching one.
type Evaluator interface {
	// Evaluate returns the effective limit for the subject, or nil when no
	// rule applies (no rate limiting).
	Evaluate(ctx context.Context, s Subject) (*Decision, error)
}

type Decision struct {
	Rule       Rule
	Remaining  int64
	RetryAfter time.Duration
	AppliedKey string
	Rejected   bool
}
