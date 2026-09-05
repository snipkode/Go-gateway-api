package audit

import (
	"context"

	"go-enterprise-api/internal/domain/audit"
)

type Service struct {
	Repo audit.Repository
}

func NewService(repo audit.Repository) *Service {
	return &Service{Repo: repo}
}

// Log is best-effort at the application boundary; callers (middleware,
// login flows) must never fail the main request because audit logging failed.
// For atomic use cases, the Logger is shared inside a DB transaction instead.
func (s *Service) Log(ctx context.Context, e audit.Entry) error {
	return s.Repo.Insert(ctx, e)
}

func (s *Service) Search(ctx context.Context, filter audit.SearchFilter) ([]audit.Entry, int64, error) {
	return s.Repo.Search(ctx, filter)
}
