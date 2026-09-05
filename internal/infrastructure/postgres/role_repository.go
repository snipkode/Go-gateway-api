package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"go-enterprise-api/internal/domain/role"
)

type RoleRepository struct {
	pool Querier
}

func NewRoleRepository(pool Querier) *RoleRepository {
	return &RoleRepository{pool: pool}
}

const roleColumns = `id, name, slug, description, created_at, updated_at, deleted_at`

func (r *RoleRepository) Create(ctx context.Context, p role.CreateParams) (role.Role, error) {
	const q = `
		INSERT INTO roles (name, slug, description)
		VALUES ($1, $2, $3)
		RETURNING ` + roleColumns
	rl, err := scanRole(ctx, FromQuerier(ctx, r.pool).QueryRow(ctx, q, p.Name, p.Slug, p.Description))
	if IsUniqueViolation(err) {
		return role.Role{}, role.ErrSlugTaken
	}
	return rl, err
}

func (r *RoleRepository) GetByID(ctx context.Context, id int64) (role.Role, error) {
	const q = `SELECT ` + roleColumns + ` FROM roles WHERE id = $1 AND deleted_at IS NULL`
	rl, err := scanRole(ctx, FromQuerier(ctx, r.pool).QueryRow(ctx, q, id))
	if IsNotFound(err) {
		return role.Role{}, role.ErrNotFound
	}
	return rl, err
}

func (r *RoleRepository) GetBySlug(ctx context.Context, slug string) (role.Role, error) {
	const q = `SELECT ` + roleColumns + ` FROM roles WHERE slug = $1 AND deleted_at IS NULL`
	rl, err := scanRole(ctx, FromQuerier(ctx, r.pool).QueryRow(ctx, q, slug))
	if IsNotFound(err) {
		return role.Role{}, role.ErrNotFound
	}
	return rl, err
}

func (r *RoleRepository) List(ctx context.Context, includeDeleted bool) ([]role.Role, error) {
	q := `SELECT ` + roleColumns + ` FROM roles`
	if !includeDeleted {
		q += ` WHERE deleted_at IS NULL`
	}
	q += ` ORDER BY id`

	rows, err := FromQuerier(ctx, r.pool).Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	roles := make([]role.Role, 0)
	for rows.Next() {
		var rl role.Role
		if err := rows.Scan(&rl.ID, &rl.Name, &rl.Slug, &rl.Description, &rl.CreatedAt, &rl.UpdatedAt, &rl.DeletedAt); err != nil {
			return nil, err
		}
		roles = append(roles, rl)
	}
	return roles, rows.Err()
}

func (r *RoleRepository) Update(ctx context.Context, id int64, p role.UpdateParams) (role.Role, error) {
	set := "updated_at = NOW()"
	args := []any{id}
	add := func(col string, v any) {
		args = append(args, v)
		set += ", " + col + " = $" + itoa(len(args))
	}
	if p.Name != nil {
		add("name", *p.Name)
	}
	if p.Description != nil {
		add("description", *p.Description)
	}

	q := `UPDATE roles SET ` + set + ` WHERE id = $1 AND deleted_at IS NULL RETURNING ` + roleColumns
	rl, err := scanRole(ctx, FromQuerier(ctx, r.pool).QueryRow(ctx, q, args...))
	if IsNotFound(err) {
		return role.Role{}, role.ErrNotFound
	}
	return rl, err
}

func (r *RoleRepository) SoftDelete(ctx context.Context, id int64) error {
	const q = `UPDATE roles SET deleted_at = NOW(), updated_at = NOW() WHERE id = $1 AND deleted_at IS NULL`
	err := softDeleteExec(ctx, FromQuerier(ctx, r.pool), q, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return role.ErrNotFound
	}
	return err
}

func (r *RoleRepository) Restore(ctx context.Context, id int64) error {
	const q = `UPDATE roles SET deleted_at = NULL, updated_at = NOW() WHERE id = $1`
	err := restoreExec(ctx, FromQuerier(ctx, r.pool), q, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return role.ErrNotFound
	}
	return err
}

func (r *RoleRepository) AssignRoleToUser(ctx context.Context, userID, roleID int64) error {
	const q = `
		INSERT INTO user_roles (user_id, role_id)
		VALUES ($1, $2)
		ON CONFLICT (user_id, role_id) DO UPDATE SET deleted_at = NULL`
	_, err := FromQuerier(ctx, r.pool).Exec(ctx, q, userID, roleID)
	return err
}

func (r *RoleRepository) AssignPermission(ctx context.Context, p role.AssignPermissionParams) error {
	const q = `
		INSERT INTO role_permissions (role_id, permission_id)
		VALUES ($1, $2)
		ON CONFLICT (role_id, permission_id) DO NOTHING`
	_, err := FromQuerier(ctx, r.pool).Exec(ctx, q, p.RoleID, p.PermissionID)
	return err
}

func (r *RoleRepository) RemovePermission(ctx context.Context, roleID, permissionID int64) error {
	const q = `
		UPDATE role_permissions SET deleted_at = NOW()
		WHERE role_id = $1 AND permission_id = $2 AND deleted_at IS NULL`
	_, err := FromQuerier(ctx, r.pool).Exec(ctx, q, roleID, permissionID)
	return err
}

func (r *RoleRepository) Permissions(ctx context.Context, roleID int64) ([]int64, error) {
	const q = `
		SELECT rp.permission_id
		FROM role_permissions rp
		WHERE rp.role_id = $1 AND rp.deleted_at IS NULL`
	rows, err := FromQuerier(ctx, r.pool).Query(ctx, q, roleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ids := make([]int64, 0)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (r *RoleRepository) UsersByRole(ctx context.Context, roleID int64) ([]int64, error) {
	const q = `
		SELECT ur.user_id
		FROM user_roles ur
		WHERE ur.role_id = $1 AND ur.deleted_at IS NULL`
	rows, err := FromQuerier(ctx, r.pool).Query(ctx, q, roleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ids := make([]int64, 0)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (r *RoleRepository) PermissionsByUser(ctx context.Context, userID int64) ([]string, error) {
	const q = `
		SELECT DISTINCT p.slug
		FROM permissions p
		JOIN role_permissions rp ON rp.permission_id = p.id AND rp.deleted_at IS NULL
		JOIN roles r           ON rp.role_id = r.id AND r.deleted_at IS NULL
		JOIN user_roles ur     ON ur.role_id = r.id AND ur.deleted_at IS NULL
		WHERE ur.user_id = $1 AND p.deleted_at IS NULL`
	rows, err := FromQuerier(ctx, r.pool).Query(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	slugs := make([]string, 0)
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		slugs = append(slugs, s)
	}
	return slugs, rows.Err()
}

func (r *RoleRepository) RoleSlugsByUser(ctx context.Context, userID int64) ([]string, error) {
	const q = `
		SELECT r.slug
		FROM roles r
		JOIN user_roles ur ON ur.role_id = r.id AND ur.deleted_at IS NULL
		WHERE ur.user_id = $1 AND r.deleted_at IS NULL
		ORDER BY r.id`
	rows, err := FromQuerier(ctx, r.pool).Query(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	slugs := make([]string, 0)
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		slugs = append(slugs, s)
	}
	return slugs, rows.Err()
}

func scanRole(ctx context.Context, row interface{ Scan(...any) error }) (role.Role, error) {
	var r role.Role
	err := row.Scan(&r.ID, &r.Name, &r.Slug, &r.Description, &r.CreatedAt, &r.UpdatedAt, &r.DeletedAt)
	return r, err
}

func softDeleteExec(ctx context.Context, q Querier, query string, id int64) error {
	tag, err := q.Exec(ctx, query, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func restoreExec(ctx context.Context, q Querier, query string, id int64) error {
	tag, err := q.Exec(ctx, query, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

var _ role.Repository = (*RoleRepository)(nil)
