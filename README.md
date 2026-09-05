# Go Enterprise API

Enterprise boilerplate for a Go backend (Vue.js frontend not included), built
around **Clean Architecture**. Core enterprise concerns are first-class:
**Dynamic Rate Limit**, **Soft Delete**, **Audit Logging**, **RBAC**, a
**Nginx API Gateway**, and **Microsoft Entra (OIDC) SSO** — with a
**PostgreSQL source of truth** and **Redis** for cache/session/rate-limit
counters.

```
Browser/Vue.js → Nginx Gateway → Go API (GatewayToken → RequestID → Recovery → CORS → Dynamic Rate Limit → Auth → RBAC)
                                 → Use Case → Repository/PostgreSQL + Audit (same transaction) → Redis
```

## Blueprint

| Concern                    | Implementation                                                                  |
| -------------------------- | ------------------------------------------------------------------------------- |
| Architecture               | Clean / DDD-style layering: domain → application → infrastructure → interface    |
| Dynamic Rate Limit         | `rate_limit_rules` in PostgreSQL, cached in Redis (30s TTL), no redeploy to change |
| Rate limit scopes          | `global`, `ip`, `user`, `role`, `route`, `api_key` — strictest rule wins        |
| Soft Delete                | `deleted_at` on every entity; partial unique indexes re-enable reuse (e.g. email) |
| Audit Logging              | `audit_logs` (WHO/WHAT/DATA/WHEN/FROM); written in the same DB transaction as the mutation |
| RBAC                       | users → user_roles → roles → role_permissions → permissions (dynamic, data-driven) |
| Auth                       | JWT (HS256) + Redis session (login/logout, `ViewerToken`)                        |
| SSO                        | Microsoft Entra (OIDC) via `go-oidc`; CSRF state store; JIT user provisioning with a default role |
| Gateway                    | Nginx reverse proxy: per-IP rate limits, WAF-ish request filtering, security headers, shared-secret handshake |
| Docs                        | Swagger/OpenAPI 2.0 (swag annotations → `docs/`), served at `/swagger/`          |
| Observability              | structured JSON logs, request id, healthz / readyz, graceful shutdown             |

## Documentation

| Guide                          | Covers                                                                                     |
| ------------------------------ | ------------------------------------------------------------------------------------------ |
| [docs/ENTRA_SSO.md](docs/ENTRA_SSO.md) | Microsoft Entra integration + local mock IdP demo mode, security design              |
| [docs/SWAGGER.md](docs/SWAGGER.md)     | Swagger UI, how to annotate handlers, regenerate docs, toggle in prod                 |
| [docs/GATEWAY.md](docs/GATEWAY.md)     | Nginx gateway: rate limits, WAF-ish filters, headers, `X-Gateway-Token` handshake     |
| [docs/ENVIRONMENT.md](docs/ENVIRONMENT.md) | Every environment variable explained                                             |

---

## Getting started (Docker)

```bash
cp .env.example .env
# optional: generate a strong shared secret for the gateway handshake
python3 -c "import secrets;print(secrets.token_urlsafe(32))"
docker compose -f deployments/docker-compose.yml up --build
```

| Access point          | URL                          | Notes                                      |
| --------------------- | ---------------------------- | ------------------------------------------ |
| API gateway           | http://localhost:18080       | only public entry point                    |
| Swagger UI            | http://localhost:18080/swagger/index.html | OpenAPI 2.0 (disable in prod)   |
| Mock IdP (SSO demo)   | http://localhost:9090/.well-known/openid-configuration | local OIDC provider    |
| Postgres              | localhost:15432 (`app`/`app`) | db `go_enterprise`                        |
| Redis                 | localhost:16379              |                                            |

The Nginx **gateway** terminates the only public port and proxies to the Go
API, which lives on the internal Docker network with **no published host
port**. Migrations run automatically on API startup. A bootstrap admin is
seeded:

```
email:    admin@example.com
password: admin123
```

> Change the password/hash in `migrations/009_bootstrap_admin.sql` for anything
> other than local development, and prefer Entra/OIDC provisioning in
> production.

## Try it

```bash
# 1. Local login → token
TOKEN=$(curl -s -X POST http://localhost:18080/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"admin@example.com","password":"admin123"}' \
  | python3 -c "import sys,json;print(json.load(sys.stdin)['data']['access_token'])")

# 2. RBAC: admin can write; logged-in reads work
curl -s http://localhost:18080/api/v1/users -H "Authorization: Bearer $TOKEN"

# 3. SSO: full OIDC code flow against the bundled mock IdP
#    (state + authorize → code → token → id_token → JIT user + session)
curl -s -L --resolve mock-idp:9090:127.0.0.1 http://localhost:18080/api/v1/auth/entra/login

# 4. Dynamic rate limit: change the viewer rule to 1000 req/min live, no redeploy
curl -s -X PUT http://localhost:18080/api/v1/rate-limits/4 \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"name":"Viewer Role","scope":"role","identifier":"viewer","requests":1000,"window_seconds":60}'

# 5. Audit trail (old → new data captured on every mutation)
curl -s "http://localhost:18080/api/v1/audit-logs?resource=users" -H "Authorization: Bearer $TOKEN"

# 6. Gateway defenses (through the proxy)
curl -s -o /dev/null -w "sqlmap UA → %{http_code}\n" -A "sqlmap/1.7" http://localhost:18080/api/v1/users
curl -s -o /dev/null -w "SQLi payload → %{http_code}\n" "http://localhost:18080/api/v1/auth/login?q=UNION%20SELECT"
```

## Endpoints

| Method | Path                              | Permission(s)          |
| ------ | --------------------------------- | ---------------------- |
| GET    | `/healthz`, `/readyz`             | public (probes)        |
| GET    | `/swagger/index.html`             | public (disable in prod) |
| POST   | `/api/v1/auth/login`              | public (route-limited) |
| POST   | `/api/v1/auth/logout`             | auth                   |
| GET    | `/api/v1/auth/me`                 | auth                   |
| GET    | `/api/v1/auth/entra/login`        | public; 302 to IdP (SSO) |
| GET    | `/api/v1/auth/entra/callback`     | public; OIDC callback (SSO) |
| CRUD   | `/api/v1/users[/{id}[/restore]]`  | user:read/create/update/delete/restore |
| CRUD   | `/api/v1/roles[/{id}[/restore]]`  | role:read / role:write |
| POST/DELETE | `/api/v1/roles/{id}/permissions[{/permissionId}]` | permission:assign |
| CRUD   | `/api/v1/permissions[/{id}[/restore]]` | role:read / permission:assign |
| CRUD   | `/api/v1/rate-limits[/{id}[/restore]]` | ratelimit:read / ratelimit:write |
| GET/POST | `/api/v1/audit-logs`             | audit:read             |

Soft-delete endpoints return 204; `POST .../{id}/restore` clears `deleted_at`.
Hard delete is deliberately not exposed outside maintenance/retention jobs.

## How it works

### Auth + sessions
`POST /auth/login` verifies the password (bcrypt) and returns a short-lived
`access_token` (JWT HS256) plus a server-side session stored in Redis
(`sessions:*`, TTL `APP_SESSION_TTL`). Every protected request validates the
JWT signature, claims, and that the session still exists (`ViewerToken`
middleware) — logging out (or deleting the session) invalidates tokens.

### Microsoft Entra (OIDC) SSO
`GET /auth/entra/login` mints a single-use CSRF state, redirects the browser to
the IdP, and the callback verifies the state, exchanges the code with client
credentials, validates the RS256 `id_token` against the IdP's published JWKS,
then **just-in-time provisions** the user (default role `OAUTH_DEFAULT_ROLE`)
and issues the same JWT + Redis session as a normal login. Works with the
bundled mock IdP out of the box and with a real Microsoft tenant — see
[docs/ENTRA_SSO.md](docs/ENTRA_SSO.md).

### Dynamic rate limit
Rules live in `rate_limit_rules` (PostgreSQL) and are cached in Redis under
`ratelimit:rules`. The middleware builds a `Subject` (IP, user, roles, route)
and the evaluator enforces the **strictest** matching rule. Counters are plain
fixed-window Redis counters (`ratelimit:<scope>:<id>`, TTL = window). Changing
a rule via `PUT /rate-limits/{id}` is live within the cache TTL.

### Soft delete + uniqueness
Every table has `deleted_at TIMESTAMPTZ NULL`. Deletes are
`UPDATE ... SET deleted_at = NOW()`; repositories always filter
`AND deleted_at IS NULL`. Unique columns use partial indexes so a soft-deleted
row no longer blocks a new one.

### Audit inside transactions
`internal/infrastructure/postgres/unit_of_work.go` runs mutations and audit
inserts in one DB transaction — a failed audit rolls back the change. The
`accountable` fields (who, what, data, when, and the change itself) are
recorded per entry. Login/logout are logged outside transactions (no atomic
data mutation to protect).

## Local development (no Docker)

Requirements: PostgreSQL, Redis, Go 1.25+.

```bash
cp .env.example .env
# point POSTGRES_HOST / REDIS_HOST at your local instances
# for SSO, either run the mock (go run ./deployments/mock-idp) or use a real tenant
APP_MIGRATIONS_PATH=./migrations go run ./cmd/api
```

## Project layout

```
cmd/api/                         entry point, wiring a.k.a. composition root
docs/                            generated Swagger spec (doc.go, swagger.json/yaml)
internal/
  domain/                        entities + repository/use-case interfaces
    user role permission session ratelimit audit
  application/                   use-case implementations (transaction + audit)
    auth user role permission ratelimit audit
  infrastructure/                adapters: postgres, redis, jwt, entra
  interface/http/                handlers, middleware, router
  interface/httpapi/             shared HTTP helpers (response, context)
  config/                        environment config
migrations/                      001..009 SQL migrations + seed
deployments/docker-compose.yml   postgres + redis + api + gateway + mock-idp
deployments/nginx/nginx.conf     reverse proxy + gateway security (rate limit, filters, headers)
deployments/nginx/entrypoint.sh  injects X-Gateway-Token from APP_GATEWAY_TOKEN
deployments/mock-idp/            dependency-free local OIDC IdP (RS256) for dev/demo
```

## Design principles

- Business config (roles, permissions, rate limits) is **data in PostgreSQL**,
  never hardcoded — cached in Redis, editable at runtime.
- Soft delete, audit, and rate limiting are cross-cutting **concerns by
  default**.
- Dependencies point inward: interface → application → domain; infrastructure
  implements domain interfaces (Dependency Inversion).
- Repositories auto-join a transaction via context (`FromQuerier(ctx, pool)`),
  which is how UPDATE + AUDIT stay atomic.