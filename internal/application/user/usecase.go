package user

import (
	"context"

	"go-enterprise-api/internal/domain/user"
)

type UseCase interface {
	Create(ctx context.Context, p user.CreateParams) (user.User, error)
	Get(ctx context.Context, id int64) (user.User, error)
	List(ctx context.Context) ([]user.User, error)
	Update(ctx context.Context, id int64, p user.UpdateParams) (user.User, error)
	SoftDelete(ctx context.Context, id int64) error
	Restore(ctx context.Context, id int64) error
}
