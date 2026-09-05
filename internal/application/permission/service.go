package permission

import (
	"context"
	"errors"
	"fmt"

	"go-enterprise-api/internal/domain/audit"
	"go-enterprise-api/internal/domain/permission"
)

var ErrPermissionNotFound = errors.New("permission not found")

type UnitOfWork interface {
	Within(ctx context.Context, fn func(ctx context.Context) error) error
}

type Service struct {
	Permissions permission.Repository
	Audit       audit.Logger
	Tx          UnitOfWork
}

func NewService(permissions permission.Repository, audit audit.Logger, tx UnitOfWork) *Service {
	return &Service{Permissions: permissions, Audit: audit, Tx: tx}
}

func (s *Service) Create(ctx context.Context, p permission.CreateParams) (permission.Permission, error) {
	var created permission.Permission
	err := s.Tx.Within(ctx, func(txCtx context.Context) error {
		var err error
		created, err = s.Permissions.Create(txCtx, p)
		if err != nil {
			return err
		}
		return s.Audit.Log(txCtx, audit.Entry{
			Action:     audit.ActionCreate,
			Resource:   "permissions",
			ResourceID: fmt.Sprintf("%d", created.ID),
			NewData:    map[string]any{"name": created.Name, "slug": created.Slug},
		})
	})
	return created, err
}

func (s *Service) Get(ctx context.Context, id int64) (permission.Permission, error) {
	return s.Permissions.GetByID(ctx, id)
}

func (s *Service) List(ctx context.Context) ([]permission.Permission, error) {
	return s.Permissions.List(ctx, false)
}

func (s *Service) Update(ctx context.Context, id int64, p permission.CreateParams) (permission.Permission, error) {
	var updated permission.Permission
	err := s.Tx.Within(ctx, func(txCtx context.Context) error {
		var err error
		updated, err = s.Permissions.Update(txCtx, id, p)
		if err != nil {
			return err
		}
		return s.Audit.Log(txCtx, audit.Entry{
			Action:     audit.ActionUpdate,
			Resource:   "permissions",
			ResourceID: fmt.Sprintf("%d", id),
			NewData:    map[string]any{"name": updated.Name, "slug": updated.Slug},
		})
	})
	return updated, err
}

func (s *Service) SoftDelete(ctx context.Context, id int64) error {
	return s.Tx.Within(ctx, func(txCtx context.Context) error {
		if err := s.Permissions.SoftDelete(txCtx, id); err != nil {
			return err
		}
		return s.Audit.Log(txCtx, audit.Entry{
			Action:     audit.ActionDelete,
			Resource:   "permissions",
			ResourceID: fmt.Sprintf("%d", id),
		})
	})
}

func (s *Service) Restore(ctx context.Context, id int64) error {
	return s.Tx.Within(ctx, func(txCtx context.Context) error {
		if err := s.Permissions.Restore(txCtx, id); err != nil {
			return err
		}
		return s.Audit.Log(txCtx, audit.Entry{
			Action:     audit.ActionRestore,
			Resource:   "permissions",
			ResourceID: fmt.Sprintf("%d", id),
		})
	})
}
