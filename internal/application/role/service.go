package role

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"go-enterprise-api/internal/domain/audit"
	"go-enterprise-api/internal/domain/role"
)

var (
	ErrSlugRequired  = errors.New("slug is required")
	ErrRoleNotFound  = errors.New("role not found")
	ErrAssignedUsers = errors.New("role still has assigned users")
)

type UnitOfWork interface {
	Within(ctx context.Context, fn func(ctx context.Context) error) error
}

type Service struct {
	Roles role.Repository
	Audit audit.Logger
	Tx    UnitOfWork
}

func NewService(roles role.Repository, audit audit.Logger, tx UnitOfWork) *Service {
	return &Service{Roles: roles, Audit: audit, Tx: tx}
}

func (s *Service) Create(ctx context.Context, p role.CreateParams) (role.Role, error) {
	p.Slug = strings.ToLower(strings.TrimSpace(p.Slug))
	if p.Slug == "" {
		return role.Role{}, ErrSlugRequired
	}
	var created role.Role
	err := s.Tx.Within(ctx, func(txCtx context.Context) error {
		var err error
		created, err = s.Roles.Create(txCtx, p)
		if err != nil {
			return err
		}
		return s.Audit.Log(txCtx, audit.Entry{
			Action:     audit.ActionCreate,
			Resource:   "roles",
			ResourceID: fmt.Sprintf("%d", created.ID),
			NewData:    map[string]any{"name": created.Name, "slug": created.Slug},
		})
	})
	return created, err
}

func (s *Service) Get(ctx context.Context, id int64) (role.Role, error) {
	return s.Roles.GetByID(ctx, id)
}

func (s *Service) List(ctx context.Context) ([]role.Role, error) {
	return s.Roles.List(ctx, false)
}

func (s *Service) Update(ctx context.Context, id int64, p role.UpdateParams) (role.Role, error) {
	var updated role.Role
	err := s.Tx.Within(ctx, func(txCtx context.Context) error {
		prev, err := s.Roles.GetByID(txCtx, id)
		if err != nil {
			return err
		}
		updated, err = s.Roles.Update(txCtx, id, p)
		if err != nil {
			return err
		}
		return s.Audit.Log(txCtx, audit.Entry{
			Action:     audit.ActionUpdate,
			Resource:   "roles",
			ResourceID: fmt.Sprintf("%d", id),
			OldData:    map[string]any{"name": prev.Name},
			NewData:    map[string]any{"name": updated.Name},
		})
	})
	return updated, err
}

func (s *Service) SoftDelete(ctx context.Context, id int64) error {
	users, err := s.Roles.UsersByRole(ctx, id)
	if err != nil {
		return err
	}
	if len(users) > 0 {
		return ErrAssignedUsers
	}
	return s.Tx.Within(ctx, func(txCtx context.Context) error {
		if err := s.Roles.SoftDelete(txCtx, id); err != nil {
			return err
		}
		return s.Audit.Log(txCtx, audit.Entry{
			Action:     audit.ActionDelete,
			Resource:   "roles",
			ResourceID: fmt.Sprintf("%d", id),
		})
	})
}

func (s *Service) Restore(ctx context.Context, id int64) error {
	return s.Tx.Within(ctx, func(txCtx context.Context) error {
		if err := s.Roles.Restore(txCtx, id); err != nil {
			return err
		}
		return s.Audit.Log(txCtx, audit.Entry{
			Action:     audit.ActionRestore,
			Resource:   "roles",
			ResourceID: fmt.Sprintf("%d", id),
		})
	})
}

func (s *Service) AssignPermission(ctx context.Context, roleID, permissionID int64) error {
	return s.Tx.Within(ctx, func(txCtx context.Context) error {
		if err := s.Roles.AssignPermission(txCtx, role.AssignPermissionParams{
			RoleID:       roleID,
			PermissionID: permissionID,
		}); err != nil {
			return err
		}
		return s.Audit.Log(txCtx, audit.Entry{
			Action:     audit.ActionPermissionGranted,
			Resource:   "roles",
			ResourceID: fmt.Sprintf("%d", roleID),
			Metadata:   map[string]any{"permission_id": permissionID},
		})
	})
}

func (s *Service) RemovePermission(ctx context.Context, roleID, permissionID int64) error {
	return s.Tx.Within(ctx, func(txCtx context.Context) error {
		if err := s.Roles.RemovePermission(txCtx, roleID, permissionID); err != nil {
			return err
		}
		return s.Audit.Log(txCtx, audit.Entry{
			Action:     audit.ActionPermissionRevoked,
			Resource:   "roles",
			ResourceID: fmt.Sprintf("%d", roleID),
			Metadata:   map[string]any{"permission_id": permissionID},
		})
	})
}
