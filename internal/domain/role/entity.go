package role

import (
	"context"
	"errors"
	"time"
)

var (
	ErrNotFound  = errors.New("role not found")
	ErrSlugTaken = errors.New("role slug already in use")
)

type Role struct {
	ID          int64      `json:"id"`
	Name        string     `json:"name"`
	Slug        string     `json:"slug"`
	Description string     `json:"description"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty"`
}

type CreateParams struct {
	Name        string
	Slug        string
	Description string
}

type UpdateParams struct {
	Name        *string
	Description *string
}

// Permission assignment types
type AssignPermissionParams struct {
	RoleID       int64
	PermissionID int64
}

type Repository interface {
	Create(ctx context.Context, p CreateParams) (Role, error)
	GetByID(ctx context.Context, id int64) (Role, error)
	GetBySlug(ctx context.Context, slug string) (Role, error)
	List(ctx context.Context, includeDeleted bool) ([]Role, error)
	Update(ctx context.Context, id int64, p UpdateParams) (Role, error)
	SoftDelete(ctx context.Context, id int64) error
	Restore(ctx context.Context, id int64) error

	AssignPermission(ctx context.Context, p AssignPermissionParams) error
	RemovePermission(ctx context.Context, roleID, permissionID int64) error
	Permissions(ctx context.Context, roleID int64) ([]int64, error)
	UsersByRole(ctx context.Context, roleID int64) ([]int64, error)

	// AssignRoleToUser links a role to a user (used by SSO/JIT provisioning).
	AssignRoleToUser(ctx context.Context, userID, roleID int64) error

	PermissionsByUser(ctx context.Context, userID int64) ([]string, error)
	RoleSlugsByUser(ctx context.Context, userID int64) ([]string, error)
}
