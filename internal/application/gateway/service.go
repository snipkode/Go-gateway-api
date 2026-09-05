package gateway

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"strings"

	"go-enterprise-api/internal/domain/audit"
	"go-enterprise-api/internal/domain/gatewayapi"
	"go-enterprise-api/internal/infrastructure/gatewayconfig"
)

var (
	ErrNameRequired  = fmt.Errorf("name is required")
	ErrEmptyUpdate   = fmt.Errorf("nothing to update")
	ErrInvalidMethod = fmt.Errorf("invalid http method")
	ErrConfigPublish = fmt.Errorf("gateway config publish failed")
)

// UnitOfWork mirrors the pattern used by the other application services.
type UnitOfWork interface {
	Within(ctx context.Context, fn func(ctx context.Context) error) error
}

// ConfigPublisher applies the registry to the gateway (shared volume + reload).
type ConfigPublisher interface {
	Publish(ctx context.Context) error
}

type Service struct {
	Repo      gatewayapi.Repository
	Audit     audit.Logger
	Tx        UnitOfWork
	Renderer  *gatewayconfig.Renderer
	Publisher ConfigPublisher
}

// UseCase is the surface consumed by the HTTP layer.
type UseCase interface {
	Create(ctx context.Context, p gatewayapi.CreateParams) (gatewayapi.GatewayAPI, error)
	Get(ctx context.Context, id int64) (gatewayapi.GatewayAPI, error)
	List(ctx context.Context) ([]gatewayapi.GatewayAPI, error)
	Update(ctx context.Context, id int64, p gatewayapi.UpdateParams) (gatewayapi.GatewayAPI, error)
	SoftDelete(ctx context.Context, id int64) error
	Restore(ctx context.Context, id int64) error
	Preview(ctx context.Context, id int64) (string, error)
	Publish(ctx context.Context) error
}

func NewService(repo gatewayapi.Repository, audit audit.Logger, tx UnitOfWork, r *gatewayconfig.Renderer, pub ConfigPublisher) *Service {
	return &Service{Repo: repo, Audit: audit, Tx: tx, Renderer: r, Publisher: pub}
}

func (s *Service) Create(ctx context.Context, p gatewayapi.CreateParams) (gatewayapi.GatewayAPI, error) {
	p = normalize(p)
	if err := validate(p); err != nil {
		return gatewayapi.GatewayAPI{}, err
	}

	var created gatewayapi.GatewayAPI
	err := s.Tx.Within(ctx, func(txCtx context.Context) error {
		var err error
		created, err = s.Repo.Create(txCtx, p)
		if err != nil {
			return err
		}
		return s.Audit.Log(txCtx, audit.Entry{
			Action:     audit.ActionCreate,
			Resource:   "gateway_apis",
			ResourceID: fmt.Sprintf("%d", created.ID),
			NewData:    gatewaySnapshot(created),
		})
	})
	if err != nil {
		return gatewayapi.GatewayAPI{}, err
	}
	s.publishBestEffort(ctx, created.ID)
	return created, nil
}

func (s *Service) Get(ctx context.Context, id int64) (gatewayapi.GatewayAPI, error) {
	return s.Repo.GetByID(ctx, id)
}

func (s *Service) List(ctx context.Context) ([]gatewayapi.GatewayAPI, error) {
	return s.Repo.List(ctx, false)
}

func (s *Service) Update(ctx context.Context, id int64, p gatewayapi.UpdateParams) (gatewayapi.GatewayAPI, error) {
	if p.Name == nil && p.BasePath == nil && p.Upstream == nil && p.Methods == nil &&
		p.RequiresAuth == nil && p.RateLimitRPM == nil && p.IsActive == nil && p.Note == nil {
		return gatewayapi.GatewayAPI{}, ErrEmptyUpdate
	}

	var merged gatewayapi.GatewayAPI
	err := s.Tx.Within(ctx, func(txCtx context.Context) error {
		prev, err := s.Repo.GetByID(txCtx, id)
		if err != nil {
			return err
		}
		params := applyUpdate(prev, p)
		if err := validate(params); err != nil {
			return err
		}
		merged, err = s.Repo.Update(txCtx, id, p)
		if err != nil {
			return err
		}
		return s.Audit.Log(txCtx, audit.Entry{
			Action:     audit.ActionUpdate,
			Resource:   "gateway_apis",
			ResourceID: fmt.Sprintf("%d", id),
			OldData:    gatewaySnapshot(prev),
			NewData:    gatewaySnapshot(merged),
		})
	})
	if err != nil {
		return gatewayapi.GatewayAPI{}, err
	}
	s.publishBestEffort(ctx, merged.ID)
	return merged, nil
}

func (s *Service) SoftDelete(ctx context.Context, id int64) error {
	if err := s.Tx.Within(ctx, func(txCtx context.Context) error {
		if err := s.Repo.SoftDelete(txCtx, id); err != nil {
			return err
		}
		return s.Audit.Log(txCtx, audit.Entry{
			Action:     audit.ActionDelete,
			Resource:   "gateway_apis",
			ResourceID: fmt.Sprintf("%d", id),
		})
	}); err != nil {
		return err
	}
	s.publishBestEffort(ctx, id)
	return nil
}

func (s *Service) Restore(ctx context.Context, id int64) error {
	if err := s.Tx.Within(ctx, func(txCtx context.Context) error {
		if err := s.Repo.Restore(txCtx, id); err != nil {
			return err
		}
		return s.Audit.Log(txCtx, audit.Entry{
			Action:     audit.ActionRestore,
			Resource:   "gateway_apis",
			ResourceID: fmt.Sprintf("%d", id),
		})
	}); err != nil {
		return err
	}
	s.publishBestEffort(ctx, id)
	return nil
}

// Preview simulates the generated nginx location block without publishing it.
func (s *Service) Preview(ctx context.Context, id int64) (string, error) {
	api, err := s.Repo.GetByID(ctx, id)
	if err != nil {
		return "", err
	}
	return s.Renderer.RenderLocation(api)
}

// Publish (re)applies the whole registry to the gateway and reloads it.
func (s *Service) Publish(ctx context.Context) error {
	if s.Publisher == nil {
		return ErrConfigPublish
	}
	if err := s.Publisher.Publish(ctx); err != nil {
		return err
	}
	return s.Audit.Log(ctx, audit.Entry{
		Action:   "GATEWAY_CONFIG_PUBLISHED",
		Resource: "gateway_apis",
		Metadata: map[string]any{"all": true},
	})
}

func (s *Service) publishBestEffort(ctx context.Context, id int64) {
	if s.Publisher == nil {
		return
	}
	if err := s.Publisher.Publish(ctx); err != nil {
		slog.Warn("gateway config publish failed",
			"api_id", id, "error", err.Error())
	}
}

func normalize(p gatewayapi.CreateParams) gatewayapi.CreateParams {
	p.Name = strings.TrimSpace(p.Name)
	p.BasePath = strings.TrimSpace(p.BasePath)
	p.Upstream = strings.TrimSpace(p.Upstream)
	p.Note = strings.TrimSpace(p.Note)
	methods := make([]string, 0, len(p.Methods))
	seen := map[string]bool{}
	for _, m := range p.Methods {
		m = strings.ToUpper(strings.TrimSpace(m))
		if m == "" || seen[m] {
			continue
		}
		seen[m] = true
		methods = append(methods, m)
	}
	p.Methods = methods
	p = p.Normalized()
	return p
}

func validate(p gatewayapi.CreateParams) error {
	if strings.TrimSpace(p.Name) == "" {
		return ErrNameRequired
	}
	if err := gatewayconfig.ValidateBasePath(p.BasePath); err != nil {
		return err
	}
	if p.RateLimitRPM < 1 || p.RateLimitRPM > 100000 {
		return gatewayapi.ErrInvalidRateLimit
	}
	u, err := url.Parse(p.Upstream)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return gatewayapi.ErrInvalidUpstream
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return gatewayapi.ErrInvalidUpstream
	}
	for _, m := range p.Methods {
		if len(m) > 16 {
			return ErrInvalidMethod
		}
	}
	return nil
}

func applyUpdate(prev gatewayapi.GatewayAPI, p gatewayapi.UpdateParams) gatewayapi.CreateParams {
	out := gatewayapi.CreateParams{
		Name: prev.Name, BasePath: prev.BasePath, Upstream: prev.Upstream,
		Methods: prev.Methods, RequiresAuth: prev.RequiresAuth,
		RateLimitRPM: prev.RateLimitRPM, IsActive: prev.IsActive, Note: prev.Note,
	}
	if p.Name != nil {
		out.Name = strings.TrimSpace(*p.Name)
	}
	if p.BasePath != nil {
		out.BasePath = strings.TrimSpace(*p.BasePath)
	}
	if p.Upstream != nil {
		out.Upstream = strings.TrimSpace(*p.Upstream)
	}
	if p.Methods != nil {
		out.Methods = *p.Methods
		out = normalize(out)
	}
	if p.RequiresAuth != nil {
		out.RequiresAuth = *p.RequiresAuth
	}
	if p.RateLimitRPM != nil {
		out.RateLimitRPM = *p.RateLimitRPM
	}
	if p.IsActive != nil {
		out.IsActive = *p.IsActive
	}
	if p.Note != nil {
		out.Note = *p.Note
	}
	return normalize(out)
}

func gatewaySnapshot(a gatewayapi.GatewayAPI) map[string]any {
	return map[string]any{
		"name": a.Name, "base_path": a.BasePath, "upstream": a.Upstream,
		"methods": a.Methods, "requires_auth": a.RequiresAuth,
		"rate_limit_rpm": a.RateLimitRPM, "is_active": a.IsActive, "note": a.Note,
	}
}
