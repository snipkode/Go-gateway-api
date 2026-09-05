package user

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"go-enterprise-api/internal/domain/audit"
	"go-enterprise-api/internal/domain/user"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrEmailRequired = errors.New("email is required")
	ErrEmailTaken    = errors.New("email already in use")
)

// UnitOfWork runs a closure inside a single DB transaction. Repositories
// called with the closure's context automatically join the transaction, so a
// data mutation and its audit log can be committed (or rolled back) together.
type UnitOfWork interface {
	Within(ctx context.Context, fn func(ctx context.Context) error) error
}

type Service struct {
	Users user.Repository
	Audit audit.Logger
	Tx    UnitOfWork
}

func NewService(users user.Repository, audit audit.Logger, tx UnitOfWork) *Service {
	return &Service{Users: users, Audit: audit, Tx: tx}
}

func (s *Service) Create(ctx context.Context, p user.CreateParams) (user.User, error) {
	if err := validateEmail(p.Email); err != nil {
		return user.User{}, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(p.PasswordHash), bcrypt.DefaultCost)
	if err != nil {
		return user.User{}, err
	}
	p.PasswordHash = string(hash)

	var created user.User
	err = s.Tx.Within(ctx, func(txCtx context.Context) error {
		var err error
		created, err = s.Users.Create(txCtx, p)
		if err != nil {
			return err
		}
		return s.Audit.Log(txCtx, audit.Entry{
			UserID:     created.ID,
			Action:     audit.ActionCreate,
			Resource:   "users",
			ResourceID: fmt.Sprintf("%d", created.ID),
			NewData:    map[string]any{"email": created.Email, "name": created.Name},
		})
	})
	if errors.Is(err, user.ErrEmailTaken) {
		return user.User{}, ErrEmailTaken
	}
	return created, err
}

func (s *Service) Get(ctx context.Context, id int64) (user.User, error) {
	return s.Users.GetByID(ctx, id)
}

func (s *Service) List(ctx context.Context) ([]user.User, error) {
	return s.Users.List(ctx, false)
}

func (s *Service) Update(ctx context.Context, id int64, p user.UpdateParams) (user.User, error) {
	var updated user.User
	err := s.Tx.Within(ctx, func(txCtx context.Context) error {
		prev, err := s.Users.GetByID(txCtx, id)
		if err != nil {
			return err
		}
		updated, err = s.Users.Update(txCtx, id, p)
		if err != nil {
			return err
		}
		return s.Audit.Log(txCtx, audit.Entry{
			UserID:     id,
			Action:     audit.ActionUpdate,
			Resource:   "users",
			ResourceID: fmt.Sprintf("%d", id),
			OldData:    map[string]any{"name": prev.Name, "status": prev.Status},
			NewData:    map[string]any{"name": updated.Name, "status": updated.Status},
		})
	})
	return updated, err
}

func (s *Service) SoftDelete(ctx context.Context, id int64) error {
	return s.Tx.Within(ctx, func(txCtx context.Context) error {
		if err := s.Users.SoftDelete(txCtx, id); err != nil {
			return err
		}
		return s.Audit.Log(txCtx, audit.Entry{
			UserID:     id,
			Action:     audit.ActionDelete,
			Resource:   "users",
			ResourceID: fmt.Sprintf("%d", id),
		})
	})
}

func (s *Service) Restore(ctx context.Context, id int64) error {
	return s.Tx.Within(ctx, func(txCtx context.Context) error {
		if err := s.Users.Restore(txCtx, id); err != nil {
			return err
		}
		return s.Audit.Log(txCtx, audit.Entry{
			UserID:     id,
			Action:     audit.ActionRestore,
			Resource:   "users",
			ResourceID: fmt.Sprintf("%d", id),
		})
	})
}

func validateEmail(email string) error {
	if strings.TrimSpace(email) == "" {
		return ErrEmailRequired
	}
	return nil
}
