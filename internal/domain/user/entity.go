package user

import (
	"context"
	"errors"
	"time"
)

// Sentinel domain errors so application and infrastructure layers can agree
// on business failures without importing each other.
var (
	ErrNotFound    = errors.New("user not found")
	ErrEmailTaken  = errors.New("email already in use")
	ErrInvalidCred = errors.New("invalid credentials")
)

type User struct {
	ID           int64      `json:"id"`
	Email        string     `json:"email"`
	Name         string     `json:"name"`
	PasswordHash string     `json:"-"`
	Provider     string     `json:"provider"`
	ProviderID   string     `json:"provider_id"`
	Status       string     `json:"status"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	DeletedAt    *time.Time `json:"deleted_at,omitempty"`
}

func (u *User) IsDeleted() bool {
	return u.DeletedAt != nil
}

func (u *User) IsActive() bool {
	return u.DeletedAt == nil && u.Status == "active"
}

type CreateParams struct {
	Email        string
	Name         string
	PasswordHash string
	Provider     string
	ProviderID   string
}

type UpdateParams struct {
	Name   *string
	Status *string
}

type Repository interface {
	Create(ctx context.Context, p CreateParams) (User, error)
	GetByID(ctx context.Context, id int64) (User, error)
	GetByEmail(ctx context.Context, email string) (User, error)
	List(ctx context.Context, includeDeleted bool) ([]User, error)
	Update(ctx context.Context, id int64, p UpdateParams) (User, error)
	SoftDelete(ctx context.Context, id int64) error
	Restore(ctx context.Context, id int64) error
}
