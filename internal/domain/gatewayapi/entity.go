package gatewayapi

import (
	"context"
	"errors"
	"time"
)

var (
	ErrNotFound         = errors.New("gateway api not found")
	ErrBasePathTaken    = errors.New("base path already in use")
	ErrInvalidBasePath  = errors.New("invalid base path")
	ErrInvalidUpstream  = errors.New("invalid upstream url")
	ErrInvalidRateLimit = errors.New("rate limit must be between 1 and 100000")
)

// GatewayAPI is a self-service registration of an upstream API that the Nginx
// gateway exposes at BasePath with optional JWT protection + per-IP rate limit.
type GatewayAPI struct {
	ID           int64      `json:"id"`
	Name         string     `json:"name"`
	BasePath     string     `json:"base_path"`
	Upstream     string     `json:"upstream"`
	Methods      []string   `json:"methods"`
	RequiresAuth bool       `json:"requires_auth"`
	RateLimitRPM int        `json:"rate_limit_rpm"`
	IsActive     bool       `json:"is_active"`
	Status       string     `json:"status"`
	LastChecked  *time.Time `json:"last_checked_at,omitempty"`
	Note         string     `json:"note,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	DeletedAt    *time.Time `json:"deleted_at,omitempty"`
}

type CreateParams struct {
	Name         string
	BasePath     string
	Upstream     string
	Methods      []string
	RequiresAuth bool
	RateLimitRPM int
	IsActive     bool
	Note         string
}

type UpdateParams struct {
	Name         *string
	BasePath     *string
	Upstream     *string
	Methods      *[]string
	RequiresAuth *bool
	RateLimitRPM *int
	IsActive     *bool
	Note         *string
}

func (p CreateParams) Normalized() CreateParams {
	if len(p.Methods) == 0 {
		p.Methods = []string{"GET"}
	}
	if p.RateLimitRPM <= 0 {
		p.RateLimitRPM = 60
	}
	return p
}

type Repository interface {
	Create(ctx context.Context, p CreateParams) (GatewayAPI, error)
	GetByID(ctx context.Context, id int64) (GatewayAPI, error)
	List(ctx context.Context, includeDeleted bool) ([]GatewayAPI, error)
	ListActive(ctx context.Context) ([]GatewayAPI, error)
	Update(ctx context.Context, id int64, p UpdateParams) (GatewayAPI, error)
	SoftDelete(ctx context.Context, id int64) error
	Restore(ctx context.Context, id int64) error
	UpdateStatus(ctx context.Context, id int64, status string, checkedAt time.Time) error
}
