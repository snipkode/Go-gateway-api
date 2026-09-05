package postgres

import (
	"context"
	"time"

	"go-enterprise-api/internal/domain/gatewayapi"
)

type GatewayAPIRepository struct {
	pool Querier
}

func NewGatewayAPIRepository(pool Querier) *GatewayAPIRepository {
	return &GatewayAPIRepository{pool: pool}
}

const gatewayAPIColumns = `id, name, base_path, upstream, methods, requires_auth,
	rate_limit_rpm, is_active, status, last_checked_at, note, created_at, updated_at, deleted_at`

func (r *GatewayAPIRepository) Create(ctx context.Context, p gatewayapi.CreateParams) (gatewayapi.GatewayAPI, error) {
	const q = `
		INSERT INTO gateway_apis (name, base_path, upstream, methods, requires_auth, rate_limit_rpm, is_active, note)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING ` + gatewayAPIColumns
	api, err := scanGatewayAPI(ctx, FromQuerier(ctx, r.pool).QueryRow(
		ctx, q, p.Name, p.BasePath, p.Upstream, p.Methods, p.RequiresAuth, p.RateLimitRPM, p.IsActive, p.Note,
	))
	if IsUniqueViolation(err) {
		return gatewayapi.GatewayAPI{}, gatewayapi.ErrBasePathTaken
	}
	return api, err
}

func (r *GatewayAPIRepository) GetByID(ctx context.Context, id int64) (gatewayapi.GatewayAPI, error) {
	const q = `SELECT ` + gatewayAPIColumns + ` FROM gateway_apis WHERE id = $1 AND deleted_at IS NULL`
	api, err := scanGatewayAPI(ctx, FromQuerier(ctx, r.pool).QueryRow(ctx, q, id))
	if IsNotFound(err) {
		return gatewayapi.GatewayAPI{}, gatewayapi.ErrNotFound
	}
	return api, err
}

func (r *GatewayAPIRepository) List(ctx context.Context, includeDeleted bool) ([]gatewayapi.GatewayAPI, error) {
	q := `SELECT ` + gatewayAPIColumns + ` FROM gateway_apis`
	if !includeDeleted {
		q += ` WHERE deleted_at IS NULL`
	}
	q += ` ORDER BY id`

	rows, err := FromQuerier(ctx, r.pool).Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	apis := make([]gatewayapi.GatewayAPI, 0)
	for rows.Next() {
		api, err := scanGatewayAPIRow(ctx, rows)
		if err != nil {
			return nil, err
		}
		apis = append(apis, api)
	}
	return apis, rows.Err()
}

func (r *GatewayAPIRepository) ListActive(ctx context.Context) ([]gatewayapi.GatewayAPI, error) {
	const q = `SELECT ` + gatewayAPIColumns + ` FROM gateway_apis WHERE deleted_at IS NULL AND is_active ORDER BY id`
	rows, err := FromQuerier(ctx, r.pool).Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	apis := make([]gatewayapi.GatewayAPI, 0)
	for rows.Next() {
		api, err := scanGatewayAPIRow(ctx, rows)
		if err != nil {
			return nil, err
		}
		apis = append(apis, api)
	}
	return apis, rows.Err()
}

func (r *GatewayAPIRepository) Update(ctx context.Context, id int64, p gatewayapi.UpdateParams) (gatewayapi.GatewayAPI, error) {
	set := "updated_at = NOW()"
	args := []any{id}
	add := func(col string, v any) {
		args = append(args, v)
		set += ", " + col + " = $" + itoa(len(args))
	}
	if p.Name != nil {
		add("name", *p.Name)
	}
	if p.BasePath != nil {
		add("base_path", *p.BasePath)
	}
	if p.Upstream != nil {
		add("upstream", *p.Upstream)
	}
	if p.Methods != nil {
		add("methods", *p.Methods)
	}
	if p.RequiresAuth != nil {
		add("requires_auth", *p.RequiresAuth)
	}
	if p.RateLimitRPM != nil {
		add("rate_limit_rpm", *p.RateLimitRPM)
	}
	if p.IsActive != nil {
		add("is_active", *p.IsActive)
	}
	if p.Note != nil {
		add("note", *p.Note)
	}

	q := `UPDATE gateway_apis SET ` + set + `
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING ` + gatewayAPIColumns
	api, err := scanGatewayAPI(ctx, FromQuerier(ctx, r.pool).QueryRow(ctx, q, args...))
	if IsUniqueViolation(err) {
		return gatewayapi.GatewayAPI{}, gatewayapi.ErrBasePathTaken
	}
	if IsNotFound(err) {
		return gatewayapi.GatewayAPI{}, gatewayapi.ErrNotFound
	}
	return api, err
}

func (r *GatewayAPIRepository) SoftDelete(ctx context.Context, id int64) error {
	const q = `UPDATE gateway_apis SET deleted_at = NOW(), updated_at = NOW(), is_active = FALSE WHERE id = $1 AND deleted_at IS NULL`
	tag, err := FromQuerier(ctx, r.pool).Exec(ctx, q, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return gatewayapi.ErrNotFound
	}
	return nil
}

func (r *GatewayAPIRepository) Restore(ctx context.Context, id int64) error {
	const q = `UPDATE gateway_apis SET deleted_at = NULL, updated_at = NOW() WHERE id = $1`
	tag, err := FromQuerier(ctx, r.pool).Exec(ctx, q, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return gatewayapi.ErrNotFound
	}
	return nil
}

func (r *GatewayAPIRepository) UpdateStatus(ctx context.Context, id int64, status string, checkedAt time.Time) error {
	const q = `UPDATE gateway_apis SET status = $2, last_checked_at = $3, updated_at = NOW() WHERE id = $1`
	_, err := FromQuerier(ctx, r.pool).Exec(ctx, q, id, status, checkedAt)
	return err
}

func scanGatewayAPI(ctx context.Context, row interface{ Scan(...any) error }) (gatewayapi.GatewayAPI, error) {
	var api gatewayapi.GatewayAPI
	err := row.Scan(
		&api.ID, &api.Name, &api.BasePath, &api.Upstream, &api.Methods, &api.RequiresAuth,
		&api.RateLimitRPM, &api.IsActive, &api.Status, &api.LastChecked, &api.Note,
		&api.CreatedAt, &api.UpdatedAt, &api.DeletedAt,
	)
	return api, err
}

func scanGatewayAPIRow(ctx context.Context, r interface{ Scan(...any) error }) (gatewayapi.GatewayAPI, error) {
	return scanGatewayAPI(ctx, r)
}

var _ gatewayapi.Repository = (*GatewayAPIRepository)(nil)
