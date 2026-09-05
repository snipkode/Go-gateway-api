package audit

import (
	"context"

	"go-enterprise-api/internal/domain/audit"
)

type UseCase interface {
	Log(ctx context.Context, e audit.Entry) error
	Search(ctx context.Context, filter audit.SearchFilter) ([]audit.Entry, int64, error)
}
