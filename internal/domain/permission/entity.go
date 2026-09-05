package permission

import (
	"context"
	"errors"
	"time"
)

var (
	ErrNotFound  = errors.New("permission not found")
	ErrSlugTaken = errors.New("permission slug already in use")
)

type Permission struct {
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

type Repository interface {
	Create(ctx context.Context, p CreateParams) (Permission, error)
	GetByID(ctx context.Context, id int64) (Permission, error)
	GetBySlug(ctx context.Context, slug string) (Permission, error)
	List(ctx context.Context, includeDeleted bool) ([]Permission, error)
	Update(ctx context.Context, id int64, p CreateParams) (Permission, error)
	SoftDelete(ctx context.Context, id int64) error
	Restore(ctx context.Context, id int64) error
}
