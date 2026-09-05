package postgres

import (
	"context"

	"go-enterprise-api/internal/domain/user"
)

type UserRepository struct {
	pool Querier
}

func NewUserRepository(pool Querier) *UserRepository {
	return &UserRepository{pool: pool}
}

func (r *UserRepository) Create(ctx context.Context, p user.CreateParams) (user.User, error) {
	const q = `
		INSERT INTO users (email, name, password_hash, provider, provider_id)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, email, name, password_hash, provider, provider_id, status, created_at, updated_at, deleted_at`
	u := user.User{}
	err := FromQuerier(ctx, r.pool).QueryRow(
		ctx, q, p.Email, p.Name, p.PasswordHash, p.Provider, p.ProviderID,
	).Scan(&u.ID, &u.Email, &u.Name, &u.PasswordHash, &u.Provider, &u.ProviderID, &u.Status, &u.CreatedAt, &u.UpdatedAt, &u.DeletedAt)
	if IsUniqueViolation(err) {
		return user.User{}, user.ErrEmailTaken
	}
	return u, err
}

func (r *UserRepository) GetByID(ctx context.Context, id int64) (user.User, error) {
	const q = `
		SELECT id, email, name, password_hash, provider, provider_id, status, created_at, updated_at, deleted_at
		FROM users
		WHERE id = $1 AND deleted_at IS NULL`
	u, err := scanUser(ctx, FromQuerier(ctx, r.pool).QueryRow(ctx, q, id))
	if IsNotFound(err) {
		return user.User{}, user.ErrNotFound
	}
	return u, err
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (user.User, error) {
	const q = `
		SELECT id, email, name, password_hash, provider, provider_id, status, created_at, updated_at, deleted_at
		FROM users
		WHERE email = $1 AND deleted_at IS NULL`
	u, err := scanUser(ctx, FromQuerier(ctx, r.pool).QueryRow(ctx, q, email))
	if IsNotFound(err) {
		return user.User{}, user.ErrNotFound
	}
	return u, err
}

func (r *UserRepository) List(ctx context.Context, includeDeleted bool) ([]user.User, error) {
	q := `
		SELECT id, email, name, password_hash, provider, provider_id, status, created_at, updated_at, deleted_at
		FROM users`
	if !includeDeleted {
		q += ` WHERE deleted_at IS NULL`
	}
	q += ` ORDER BY id`

	rows, err := FromQuerier(ctx, r.pool).Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := make([]user.User, 0)
	for rows.Next() {
		u, err := scanUserRow(ctx, rows)
		if err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func (r *UserRepository) Update(ctx context.Context, id int64, p user.UpdateParams) (user.User, error) {
	set := "updated_at = NOW()"
	args := []any{id}
	add := func(col string, v any) {
		args = append(args, v)
		set += ", " + col + " = $" + itoa(len(args))
	}
	if p.Name != nil {
		add("name", *p.Name)
	}
	if p.Status != nil {
		add("status", *p.Status)
	}

	q := `
		UPDATE users SET ` + set + `
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING id, email, name, password_hash, provider, provider_id, status, created_at, updated_at, deleted_at`
	u, err := scanUser(ctx, FromQuerier(ctx, r.pool).QueryRow(ctx, q, args...))
	if IsNotFound(err) {
		return user.User{}, user.ErrNotFound
	}
	return u, err
}

func (r *UserRepository) SoftDelete(ctx context.Context, id int64) error {
	const q = `UPDATE users SET deleted_at = NOW(), updated_at = NOW() WHERE id = $1 AND deleted_at IS NULL`
	tag, err := FromQuerier(ctx, r.pool).Exec(ctx, q, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return user.ErrNotFound
	}
	return nil
}

func (r *UserRepository) Restore(ctx context.Context, id int64) error {
	const q = `UPDATE users SET deleted_at = NULL, updated_at = NOW() WHERE id = $1`
	tag, err := FromQuerier(ctx, r.pool).Exec(ctx, q, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return user.ErrNotFound
	}
	return nil
}

func scanUser(ctx context.Context, row interface{ Scan(...any) error }) (user.User, error) {
	var u user.User
	err := row.Scan(&u.ID, &u.Email, &u.Name, &u.PasswordHash, &u.Provider, &u.ProviderID, &u.Status, &u.CreatedAt, &u.UpdatedAt, &u.DeletedAt)
	return u, err
}

func scanUserRow(ctx context.Context, r interface{ Scan(...any) error }) (user.User, error) {
	return scanUser(ctx, r)
}

func itoa(n int) string {
	const digits = "0123456789"
	return string(digits[n])
}

var _ user.Repository = (*UserRepository)(nil)
