package role

import (
	"context"

	"go-enterprise-api/internal/domain/role"
)

type UseCase interface {
	Create(ctx context.Context, p role.CreateParams) (role.Role, error)
	Get(ctx context.Context, id int64) (role.Role, error)
	List(ctx context.Context) ([]role.Role, error)
	Update(ctx context.Context, id int64, p role.UpdateParams) (role.Role, error)
	SoftDelete(ctx context.Context, id int64) error
	Restore(ctx context.Context, id int64) error

	AssignPermission(ctx context.Context, roleID, permissionID int64) error
	RemovePermission(ctx context.Context, roleID, permissionID int64) error
}
