package postgres

import (
	"context"
	"strings"

	"go-enterprise-api/internal/domain/audit"
)

type AuditRepository struct {
	pool Querier
}

func NewAuditRepository(pool Querier) *AuditRepository {
	return &AuditRepository{pool: pool}
}

func (r *AuditRepository) Insert(ctx context.Context, e audit.Entry) error {
	const q = `
		INSERT INTO audit_logs (
			user_id, action, resource, resource_id,
			method, path, ip_address, user_agent, request_id,
			old_data, new_data, metadata
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`

	_, err := FromQuerier(ctx, r.pool).Exec(ctx, q,
		nullableInt64(e.UserID),
		e.Action, e.Resource, nullableStr(e.ResourceID),
		nullableStr(e.Method), nullableStr(e.Path),
		nullableStr(e.IPAddress), nullableStr(e.UserAgent), nullableStr(e.RequestID),
		nullableJSON(e.OldData), nullableJSON(e.NewData), nullableJSON(e.Metadata),
	)
	return err
}

func (r *AuditRepository) Search(ctx context.Context, filter audit.SearchFilter) ([]audit.Entry, int64, error) {
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PageSize < 1 {
		filter.PageSize = 20
	}
	limit := filter.PageSize
	offset := (filter.Page - 1) * limit

	where := []string{}
	args := []any{}
	n := 0
	add := func(col string, op string, v any) {
		n++
		where = append(where, col+" "+op+" $"+itoa(n))
		args = append(args, v)
	}
	if filter.UserID > 0 {
		add("user_id", "=", filter.UserID)
	}
	if filter.Action != "" {
		add("action", "=", filter.Action)
	}
	if filter.Resource != "" {
		add("resource", "=", filter.Resource)
	}
	if filter.ResourceID != "" {
		add("resource_id", "=", filter.ResourceID)
	}
	if filter.RequestID != "" {
		add("request_id", "=", filter.RequestID)
	}

	cond := ""
	if len(where) > 0 {
		cond = " WHERE " + strings.Join(where, " AND ")
	}

	var total int64
	countQ := "SELECT COUNT(*) FROM audit_logs" + cond
	if err := FromQuerier(ctx, r.pool).QueryRow(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	listArgs := append(append([]any{}, args...), limit, offset)
	listQ := "SELECT user_id, action, resource, resource_id, method, path, ip_address, user_agent, request_id, old_data, new_data, metadata, created_at FROM audit_logs" +
		cond + " ORDER BY id DESC LIMIT $" + itoa(n+1) + " OFFSET $" + itoa(n+2)

	rows, err := FromQuerier(ctx, r.pool).Query(ctx, listQ, listArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	entries := make([]audit.Entry, 0)
	for rows.Next() {
		var e audit.Entry
		var userID *int64
		var resourceID, method, path, ip, userAgent, requestID *string
		if err := rows.Scan(
			&userID, &e.Action, &e.Resource, &resourceID,
			&method, &path, &ip, &userAgent, &requestID,
			&e.OldData, &e.NewData, &e.Metadata, &e.CreatedAt,
		); err != nil {
			return nil, 0, err
		}
		if userID != nil {
			e.UserID = *userID
		}
		e.ResourceID = deref(resourceID)
		e.Method = deref(method)
		e.Path = deref(path)
		e.IPAddress = deref(ip)
		e.UserAgent = deref(userAgent)
		e.RequestID = deref(requestID)
		entries = append(entries, e)
	}
	return entries, total, rows.Err()
}

func nullableInt64(v int64) any {
	if v == 0 {
		return nil
	}
	return v
}

func nullableStr(v string) any {
	if v == "" {
		return nil
	}
	return v
}

func nullableJSON(v map[string]any) any {
	if v == nil {
		return nil
	}
	return v
}

func deref(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

var _ audit.Repository = (*AuditRepository)(nil)
