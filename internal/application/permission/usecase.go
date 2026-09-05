package permission

import (
	"context"

	"go-enterprise-api/internal/domain/permission"
)

type UseCase interface {
	Create(ctx context.Context, p permission.CreateParams) (permission.Permission, error)
	Get(ctx context.Context, id int64) (permission.Permission, error)
	List(ctx context.Context) ([]permission.Permission, error)
	Update(ctx context.Context, id int64, p permission.CreateParams) (permission.Permission, error)
	SoftDelete(ctx context.Context, id int64) error
	Restore(ctx context.Context, id int64) error
}
