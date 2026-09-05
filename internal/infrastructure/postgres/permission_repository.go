package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"go-enterprise-api/internal/domain/permission"
)

type PermissionRepository struct {
	pool Querier
}

func NewPermissionRepository(pool Querier) *PermissionRepository {
	return &PermissionRepository{pool: pool}
}

const permissionColumns = `id, name, slug, description, created_at, updated_at, deleted_at`

func (r *PermissionRepository) Create(ctx context.Context, p permission.CreateParams) (permission.Permission, error) {
	const q = `
		INSERT INTO permissions (name, slug, description)
		VALUES ($1, $2, $3)
		RETURNING ` + permissionColumns
	pm, err := scanPermission(ctx, FromQuerier(ctx, r.pool).QueryRow(ctx, q, p.Name, p.Slug, p.Description))
	if IsUniqueViolation(err) {
		return permission.Permission{}, permission.ErrSlugTaken
	}
	return pm, err
}

func (r *PermissionRepository) GetByID(ctx context.Context, id int64) (permission.Permission, error) {
	const q = `SELECT ` + permissionColumns + ` FROM permissions WHERE id = $1 AND deleted_at IS NULL`
	pm, err := scanPermission(ctx, FromQuerier(ctx, r.pool).QueryRow(ctx, q, id))
	if IsNotFound(err) {
		return permission.Permission{}, permission.ErrNotFound
	}
	return pm, err
}

func (r *PermissionRepository) GetBySlug(ctx context.Context, slug string) (permission.Permission, error) {
	const q = `SELECT ` + permissionColumns + ` FROM permissions WHERE slug = $1 AND deleted_at IS NULL`
	pm, err := scanPermission(ctx, FromQuerier(ctx, r.pool).QueryRow(ctx, q, slug))
	if IsNotFound(err) {
		return permission.Permission{}, permission.ErrNotFound
	}
	return pm, err
}

func (r *PermissionRepository) List(ctx context.Context, includeDeleted bool) ([]permission.Permission, error) {
	q := `SELECT ` + permissionColumns + ` FROM permissions`
	if !includeDeleted {
		q += ` WHERE deleted_at IS NULL`
	}
	q += ` ORDER BY id`

	rows, err := FromQuerier(ctx, r.pool).Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	perms := make([]permission.Permission, 0)
	for rows.Next() {
		var p permission.Permission
		if err := rows.Scan(&p.ID, &p.Name, &p.Slug, &p.Description, &p.CreatedAt, &p.UpdatedAt, &p.DeletedAt); err != nil {
			return nil, err
		}
		perms = append(perms, p)
	}
	return perms, rows.Err()
}

func (r *PermissionRepository) Update(ctx context.Context, id int64, p permission.CreateParams) (permission.Permission, error) {
	q := `
		UPDATE permissions
		SET name = $2, slug = $3, description = $4, updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING ` + permissionColumns
	pm, err := scanPermission(ctx, FromQuerier(ctx, r.pool).QueryRow(ctx, q, id, p.Name, p.Slug, p.Description))
	if IsNotFound(err) {
		return permission.Permission{}, permission.ErrNotFound
	}
	return pm, err
}

func (r *PermissionRepository) SoftDelete(ctx context.Context, id int64) error {
	const q = `UPDATE permissions SET deleted_at = NOW(), updated_at = NOW() WHERE id = $1 AND deleted_at IS NULL`
	err := softDeleteExec(ctx, FromQuerier(ctx, r.pool), q, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return permission.ErrNotFound
	}
	return err
}

func (r *PermissionRepository) Restore(ctx context.Context, id int64) error {
	const q = `UPDATE permissions SET deleted_at = NULL, updated_at = NOW() WHERE id = $1`
	err := restoreExec(ctx, FromQuerier(ctx, r.pool), q, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return permission.ErrNotFound
	}
	return err
}

func scanPermission(ctx context.Context, row interface{ Scan(...any) error }) (permission.Permission, error) {
	var p permission.Permission
	err := row.Scan(&p.ID, &p.Name, &p.Slug, &p.Description, &p.CreatedAt, &p.UpdatedAt, &p.DeletedAt)
	return p, err
}

var _ permission.Repository = (*PermissionRepository)(nil)
