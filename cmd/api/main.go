// @title Go Enterprise API
// @version 1.0
// @description Enterprise boilerplate: Dynamic Rate Limit, Soft Delete, Audit Logs, RBAC, JWT + Redis sessions, Microsoft Entra (OIDC). All requests pass through the Nginx gateway (X-Gateway-Token handshake) — public endpoints are also rate-limited per IP at the gateway.
// @contact.name GitOps
// @host localhost:18080
// @schemes http https
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Provide a JWT access token obtained from /api/v1/auth/login or the Entra OIDC flow as "Bearer <token>".
package main

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	auditapp "go-enterprise-api/internal/application/audit"
	authapp "go-enterprise-api/internal/application/auth"
	gwapp "go-enterprise-api/internal/application/gateway"
	permapp "go-enterprise-api/internal/application/permission"
	rlapp "go-enterprise-api/internal/application/ratelimit"
	roleapp "go-enterprise-api/internal/application/role"
	userapp "go-enterprise-api/internal/application/user"
	"go-enterprise-api/internal/config"
	"go-enterprise-api/internal/infrastructure/entra"
	"go-enterprise-api/internal/infrastructure/gatewayconfig"
	"go-enterprise-api/internal/infrastructure/gatewaymonitor"
	"go-enterprise-api/internal/infrastructure/jwt"
	"go-enterprise-api/internal/infrastructure/postgres"
	"go-enterprise-api/internal/infrastructure/redis"
	apphttp "go-enterprise-api/internal/interface/http"
	"go-enterprise-api/internal/interface/http/handler"

	_ "go-enterprise-api/docs"
)

func main() {
	cfg := config.Load()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := run(ctx, cfg, logger); err != nil {
		logger.Error("fatal", slog.String("error", err.Error()))
		os.Exit(1)
	}
}

func run(ctx context.Context, cfg *config.Config, logger *slog.Logger) error {
	// ── Infrastructure ─────────────────────────────────────────────────
	pgPool, err := postgres.NewPool(ctx, cfg.Postgres.DSN())
	if err != nil {
		return err
	}
	defer pgPool.Close()

	if err := postgres.Migrate(ctx, pgPool, migrationsDir(cfg)); err != nil {
		return err
	}

	rdb, err := redis.NewClient(ctx, cfg.Redis.Addr(), cfg.Redis.Password, cfg.Redis.DB)
	if err != nil {
		return err
	}
	defer rdb.Close()

	// ── Repositories ───────────────────────────────────────────────────
	users := postgres.NewUserRepository(pgPool)
	roles := postgres.NewRoleRepository(pgPool)
	permissions := postgres.NewPermissionRepository(pgPool)
	rateRules := postgres.NewRateLimitRepository(pgPool)
	auditRepo := postgres.NewAuditRepository(pgPool)
	gwAPIs := postgres.NewGatewayAPIRepository(pgPool)

	sessions := redis.NewSessionRepository(rdb, cfg.JWT.TTL)
	counter := redis.NewRateLimitCounter(rdb)
	evaluator := redis.NewRateLimitEvaluator(rateRules, rdb, counter)

	auditLogger := auditapp.NewService(auditRepo)
	uow := postgres.NewUnitOfWork(pgPool)
	tokens := jwt.NewTokenService(cfg.JWT.Secret)

	// Microsoft Entra (OIDC) adapter. Discovery runs at boot; when the tenant
	// or client id are missing the flow stays disabled (handlers answer 503).
	var ssoProvider entra.Provider
	if p, err := entra.NewProvider(ctx, entra.Config{
		ClientID:     cfg.OAuth.ClientID,
		ClientSecret: cfg.OAuth.ClientSecret,
		TenantID:     cfg.OAuth.TenantID,
		Issuer:       cfg.OAuth.Issuer,
		RedirectURL:  cfg.OAuth.RedirectURL,
		Timeout:      10 * time.Second,
	}); err != nil {
		if !errors.Is(err, entra.ErrNotConfigured) {
			logger.Warn("entra provider init", slog.String("error", err.Error()))
		}
	} else {
		ssoProvider = p
		logger.Info("entra oidc enabled", slog.String("issuer", p.Issuer()))
	}
	stateStore := entra.NewStateStore(15 * time.Minute)

	// ── Application / use cases ───────────────────────────────────────
	authSvc := authapp.NewService(
		users, roles, sessions, tokens, auditLogger,
		cfg.JWT.TTL,
		cfg.OAuth.DefaultRole,
	)
	userSvc := userapp.NewService(users, auditLogger, uow)
	roleSvc := roleapp.NewService(roles, auditLogger, uow)
	permSvc := permapp.NewService(permissions, auditLogger, uow)
	rateSvc := rlapp.NewService(rateRules, auditLogger, uow)
	auditSvc := auditapp.NewService(auditRepo)

	// ── Gateway registry: generated nginx config + monitoring ──────────
	gwRenderer := &gatewayconfig.Renderer{
		BaseDir:      cfg.Gateway.ConfigDir,
		AuthUpstream: cfg.Gateway.AuthUpstream,
	}
	var gwPublisher gwapp.ConfigPublisher
	if cfg.Gateway.ConfigDir != "" {
		gwPublisher = gatewayconfig.NewPublisher(gwRenderer, gwAPIs)
	}
	gwSvc := gwapp.NewService(gwAPIs, auditLogger, uow, gwRenderer, gwPublisher)

	var gwAggregator *gatewaymonitor.Aggregator
	if cfg.Gateway.ConfigDir != "" {
		registerCtx, cancelRegister := context.WithCancel(ctx)
		defer cancelRegister()
		gwAggregator = gatewaymonitor.NewAggregator(rdb.RDB(), filepath.Join(cfg.Gateway.ConfigDir, "logs"), 100)
		go gwAggregator.Run(registerCtx)
		go gatewaymonitor.NewHealthChecker(gwAPIs, 30*time.Second, 5*time.Second).Run(registerCtx)
	}

	// ── HTTP handlers ──────────────────────────────────────────────────
	health := &handler.HealthHandler{
		Ready: func() error {
			return pgPool.Ping(context.Background())
		},
	}

	deps := apphttp.Dependencies{
		Logger:         logger,
		TokenSvc:       tokens,
		Sessions:       sessions,
		Roles:          roles,
		GatewayToken:   cfg.GatewayToken,
		SwaggerEnabled: cfg.SwaggerEnabled,

		AuthHandler: &handler.AuthHandler{
			Base:        handler.Base{Logger: logger},
			Auth:        authSvc,
			Entra:       ssoProvider,
			States:      stateStore,
			FrontendURL: cfg.OAuth.FrontendURL,
		},
		UserHandler: &handler.UserHandler{
			Base:  handler.Base{Logger: logger},
			Users: userSvc,
		},
		RoleHandler: &handler.RoleHandler{
			Base:  handler.Base{Logger: logger},
			Roles: roleSvc,
		},
		PermissionHandler: &handler.PermissionHandler{
			Base:        handler.Base{Logger: logger},
			Permissions: permSvc,
		},
		RateLimitHandler: &handler.RateLimitHandler{
			Base:  handler.Base{Logger: logger},
			Rules: rateSvc,
		},
		AuditHandler: &handler.AuditHandler{
			Base:  handler.Base{Logger: logger},
			Audit: auditSvc,
		},
		GatewayHandler: &handler.GatewayHandler{
			Base:    handler.Base{Logger: logger},
			Gateway: gwSvc,
			Monitor: gwAggregator,
		},
		HealthHandler: health,

		RateLimitEvaluator: evaluator,
	}

	srv := apphttp.NewRouter(deps, cfg.AllowedOrigins)
	srv.Addr = net.JoinHostPort("", strconv.Itoa(cfg.Port))

	// ── Serve ──────────────────────────────────────────────────────────
	errCh := make(chan error, 1)
	go func() {
		logger.Info("api listening", slog.String("addr", srv.Addr))
		errCh <- srv.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		logger.Info("shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}

func migrationsDir(_ *config.Config) string {
	if dir := os.Getenv("APP_MIGRATIONS_PATH"); dir != "" {
		return dir
	}
	return "./migrations"
}
