package ratelimit

import (
	"context"

	"go-enterprise-api/internal/domain/ratelimit"
)

type UseCase interface {
	List(ctx context.Context) ([]ratelimit.Rule, error)
	Get(ctx context.Context, id int64) (ratelimit.Rule, error)
	Create(ctx context.Context, p ratelimit.CreateParams) (ratelimit.Rule, error)
	Update(ctx context.Context, id int64, p ratelimit.UpdateParams) (ratelimit.Rule, error)
	SoftDelete(ctx context.Context, id int64) error
	Restore(ctx context.Context, id int64) error
}
