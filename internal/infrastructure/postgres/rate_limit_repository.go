package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"go-enterprise-api/internal/domain/ratelimit"
)

type RateLimitRepository struct {
	pool Querier
}

func NewRateLimitRepository(pool Querier) *RateLimitRepository {
	return &RateLimitRepository{pool: pool}
}

const ruleColumns = `id, name, scope, identifier, requests, window_seconds, enabled, priority, created_at, updated_at, deleted_at`

func (r *RateLimitRepository) List(ctx context.Context, includeDeleted bool) ([]ratelimit.Rule, error) {
	q := `SELECT ` + ruleColumns + ` FROM rate_limit_rules`
	if !includeDeleted {
		q += ` WHERE deleted_at IS NULL`
	}
	q += ` ORDER BY priority DESC, id`

	rows, err := FromQuerier(ctx, r.pool).Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	rules := make([]ratelimit.Rule, 0)
	for rows.Next() {
		var rl ratelimit.Rule
		if err := rows.Scan(
			&rl.ID, &rl.Name, &rl.Scope, &rl.Identifier,
			&rl.Requests, &rl.WindowSeconds, &rl.Enabled, &rl.Priority,
			&rl.CreatedAt, &rl.UpdatedAt, &rl.DeletedAt,
		); err != nil {
			return nil, err
		}
		rules = append(rules, rl)
	}
	return rules, rows.Err()
}

func (r *RateLimitRepository) GetByID(ctx context.Context, id int64) (ratelimit.Rule, error) {
	const q = `SELECT ` + ruleColumns + ` FROM rate_limit_rules WHERE id = $1 AND deleted_at IS NULL`
	rl, err := scanRule(ctx, FromQuerier(ctx, r.pool).QueryRow(ctx, q, id))
	if IsNotFound(err) {
		return ratelimit.Rule{}, ratelimit.ErrNotFound
	}
	return rl, err
}

func (r *RateLimitRepository) Create(ctx context.Context, p ratelimit.CreateParams) (ratelimit.Rule, error) {
	const q = `
		INSERT INTO rate_limit_rules (name, scope, identifier, requests, window_seconds, enabled, priority)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING ` + ruleColumns
	return scanRule(ctx, FromQuerier(ctx, r.pool).QueryRow(
		ctx, q,
		p.Name, p.Scope, p.Identifier, p.Requests, p.WindowSeconds, p.Enabled, p.Priority,
	))
}

func (r *RateLimitRepository) Update(ctx context.Context, id int64, p ratelimit.UpdateParams) (ratelimit.Rule, error) {
	set := "updated_at = NOW()"
	args := []any{id}
	add := func(col string, v any) {
		args = append(args, v)
		set += ", " + col + " = $" + itoa(len(args))
	}
	if p.Name != nil {
		add("name", *p.Name)
	}
	if p.Requests != nil {
		add("requests", *p.Requests)
	}
	if p.WindowSeconds != nil {
		add("window_seconds", *p.WindowSeconds)
	}
	if p.Enabled != nil {
		add("enabled", *p.Enabled)
	}
	if p.Priority != nil {
		add("priority", *p.Priority)
	}

	q := `UPDATE rate_limit_rules SET ` + set + ` WHERE id = $1 AND deleted_at IS NULL RETURNING ` + ruleColumns
	rl, err := scanRule(ctx, FromQuerier(ctx, r.pool).QueryRow(ctx, q, args...))
	if IsNotFound(err) {
		return ratelimit.Rule{}, ratelimit.ErrNotFound
	}
	return rl, err
}

func (r *RateLimitRepository) SoftDelete(ctx context.Context, id int64) error {
	const q = `UPDATE rate_limit_rules SET deleted_at = NOW(), updated_at = NOW() WHERE id = $1 AND deleted_at IS NULL`
	err := softDeleteExec(ctx, FromQuerier(ctx, r.pool), q, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return ratelimit.ErrNotFound
	}
	return err
}

func (r *RateLimitRepository) Restore(ctx context.Context, id int64) error {
	const q = `UPDATE rate_limit_rules SET deleted_at = NULL, updated_at = NOW() WHERE id = $1`
	err := restoreExec(ctx, FromQuerier(ctx, r.pool), q, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return ratelimit.ErrNotFound
	}
	return err
}

func scanRule(ctx context.Context, row interface{ Scan(...any) error }) (ratelimit.Rule, error) {
	var r ratelimit.Rule
	err := row.Scan(
		&r.ID, &r.Name, &r.Scope, &r.Identifier,
		&r.Requests, &r.WindowSeconds, &r.Enabled, &r.Priority,
		&r.CreatedAt, &r.UpdatedAt, &r.DeletedAt,
	)
	return r, err
}

var _ ratelimit.Repository = (*RateLimitRepository)(nil)
