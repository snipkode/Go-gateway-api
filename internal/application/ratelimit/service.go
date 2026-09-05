package ratelimit

import (
	"context"
	"fmt"
	"strings"

	"go-enterprise-api/internal/domain/audit"
	"go-enterprise-api/internal/domain/ratelimit"
)

type UnitOfWork interface {
	Within(ctx context.Context, fn func(ctx context.Context) error) error
}

type Service struct {
	Rules ratelimit.Repository
	Audit audit.Logger
	Tx    UnitOfWork
}

func NewService(rules ratelimit.Repository, audit audit.Logger, tx UnitOfWork) *Service {
	return &Service{Rules: rules, Audit: audit, Tx: tx}
}

func (s *Service) List(ctx context.Context) ([]ratelimit.Rule, error) {
	return s.Rules.List(ctx, false)
}

func (s *Service) Get(ctx context.Context, id int64) (ratelimit.Rule, error) {
	return s.Rules.GetByID(ctx, id)
}

func (s *Service) Create(ctx context.Context, p ratelimit.CreateParams) (ratelimit.Rule, error) {
	p.Scope = strings.ToLower(strings.TrimSpace(p.Scope))
	var created ratelimit.Rule
	err := s.Tx.Within(ctx, func(txCtx context.Context) error {
		var err error
		created, err = s.Rules.Create(txCtx, p)
		if err != nil {
			return err
		}
		return s.Audit.Log(txCtx, audit.Entry{
			Action:     audit.ActionRateLimitCreated,
			Resource:   "rate_limit_rules",
			ResourceID: fmt.Sprintf("%d", created.ID),
			NewData: map[string]any{
				"scope": created.Scope, "identifier": created.Identifier,
				"requests": created.Requests, "window_seconds": created.WindowSeconds,
			},
		})
	})
	return created, err
}

func (s *Service) Update(ctx context.Context, id int64, p ratelimit.UpdateParams) (ratelimit.Rule, error) {
	var updated ratelimit.Rule
	err := s.Tx.Within(ctx, func(txCtx context.Context) error {
		prev, err := s.Rules.GetByID(txCtx, id)
		if err != nil {
			return err
		}
		updated, err = s.Rules.Update(txCtx, id, p)
		if err != nil {
			return err
		}
		return s.Audit.Log(txCtx, audit.Entry{
			Action:     audit.ActionRateLimitUpdated,
			Resource:   "rate_limit_rules",
			ResourceID: fmt.Sprintf("%d", id),
			OldData:    map[string]any{"requests": prev.Requests, "window_seconds": prev.WindowSeconds},
			NewData:    map[string]any{"requests": updated.Requests, "window_seconds": updated.WindowSeconds},
		})
	})
	return updated, err
}

func (s *Service) SoftDelete(ctx context.Context, id int64) error {
	return s.Tx.Within(ctx, func(txCtx context.Context) error {
		if err := s.Rules.SoftDelete(txCtx, id); err != nil {
			return err
		}
		return s.Audit.Log(txCtx, audit.Entry{
			Action:     audit.ActionRateLimitDeleted,
			Resource:   "rate_limit_rules",
			ResourceID: fmt.Sprintf("%d", id),
		})
	})
}

func (s *Service) Restore(ctx context.Context, id int64) error {
	return s.Tx.Within(ctx, func(txCtx context.Context) error {
		if err := s.Rules.Restore(txCtx, id); err != nil {
			return err
		}
		return s.Audit.Log(txCtx, audit.Entry{
			Action:     audit.ActionRestore,
			Resource:   "rate_limit_rules",
			ResourceID: fmt.Sprintf("%d", id),
		})
	})
}
