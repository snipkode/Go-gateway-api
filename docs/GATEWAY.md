# API Gateway, dynamic registry, monitoring & management console

Nginx is the only public entry point. It terminates TLS/HTTP on `:18080`,
applies gateway-level defenses, proxies `/api/*` to the Go API, serves the
management console SPA at `/admin/`, and dynamically exposes user-registered
upstream APIs.

```
                 ┌──────────────────────────────────────────────────────────┐
 Browser ───────▶│ nginx (public :18080)                                    │
                 │  · gateway defenses (rate limit / WAF / headers)         │
                 │  · /api/*  ──▶ Go API  (internal :8080, no host port)    │
                 │  · /admin/ ──▶ Vue shell (dist)                          │
                 │  · /<base_path>/ ──▶ user upstream  (auth_request?)      │
                 │  · watch_registry hot-reloads generated includes         │
                 └───────┬───────────────────────────────▲──────────────────┘
                         │  shared volume "gwregistry"    │  X-Gateway-Token
                         │  (generated nginx conf + JSONL │  GET /internal/auth
                         │   access logs)                 │
                 ┌───────▼───────────────────────────────┴──────────────────┐
                 │ Go API  · registry CRUD · publisher · monitor            │
                 │ PostgreSQL (gateway_apis, audit) · Redis (gwstats:*)     │
                 └──────────────────────────────────────────────────────────┘
```

## Architecture & request flow

1. Every request passes the gateway middleware-first:
   `GatewayToken → RequestID → Recovery → CORS → Dynamic Rate Limit → Auth → RBAC → handler`.
   Auth + RBAC run **only** on `protected("permission:slug", handler)` routes.
2. The gateway forwards the shared secret as `X-Gateway-Token` (server-level
   `proxy_set_header`); a request without it is rejected with `403` by the API.
3. Static-protected locations (`/api/`, `/swagger/`, `/api/v1/auth/login`,
   `/api/v1/auth/entra/`, `/healthz`) are compiled into
   `deployments/nginx/nginx.conf`.
4. User-registered APIs are **generated config**, written by the Go API to the
   shared volume and hot-reloaded by nginx — no rebuild, no restart, no
   container exec.

## Static gateway defenses (`deployments/nginx/nginx.conf`)

- **Per-IP rate limits** with `limit_req_zone` (`zone_login` 10r/m strict on
  login, `zone_auth` 50r/m on Entra endpoints, `zone_api` 200r/m elsewhere).
  Status: `429` with a JSON body matching the API error shape.
- **WAF-ish filter** `$is_suspicious` (map on `$request_uri`): SQLi `UNION
  SELECT`, `sleep(`, `concat(`, `into …outfile`, XSS tokens, `../`, `%00`,
  `/etc/passwd`. `$is_bad_agent` blocks scanner UAs (sqlmap, nikto, dirbuster,
  python-requests, …). Matches are denied with `403`.
- **Security headers**: `X-Content-Type-Options`, `X-Frame-Options: DENY`,
  `X-XSS-Protection`, `Referrer-Policy`, `Permissions-Policy`.
- **Handshake**: `conf.d/gateway-token.conf` (written by `entrypoint.sh` from
  `APP_GATEWAY_TOKEN`) sets `$gateway_token` + the `X-Gateway-Token` header.
- Logs: standard `gateway` format + per-API JSONL format (`gateway_json`) used
  by the monitor.

## Dynamic registry (registered APIs)

The registry is normal PostgreSQL data — `migrations/010_gateway_apis.sql`
creates `gateway_apis` plus the `apigateway:read` / `apigateway:manage`
permissions (granted to the `admin` role by the migration).

### REST API

| Method   | Path                                      | Permission        | Description                    |
| -------- | ----------------------------------------- | ----------------- | ------------------------------ |
| GET      | `/api/v1/gateway/apis`                    | `apigateway:read` | List (includes health status)  |
| POST     | `/api/v1/gateway/apis`                    | `apigateway:manage` | Register an API            |
| GET      | `/api/v1/gateway/apis/{id}`               | `apigateway:read` | Detail                       |
| PUT      | `/api/v1/gateway/apis/{id}`               | `apigateway:manage` | Update (re-publishes)     |
| DELETE   | `/api/v1/gateway/apis/{id}`               | `apigateway:manage` | Soft delete (removes config) |
| POST     | `/api/v1/gateway/apis/{id}/restore`       | `apigateway:manage` | Restore                   |
| GET      | `/api/v1/gateway/apis/{id}/preview`       | `apigateway:read` | Rendered nginx location text |
| GET      | `/api/v1/gateway/apis/{id}/stats`         | `apigateway:read` | Live traffic + health    |
| POST     | `/api/v1/gateway/publish`                 | `apigateway:manage` | Re-render whole registry     |
| GET      | `/internal/auth`                          | any valid session | nginx `auth_request` target   |

Mutation requests are audited (`audit_logs`, resource `gateway_api`) inside the
same DB transaction. Mutations that leave the registry dirty trigger a
best-effort publish automatically.

### Fields

| JSON field     | Rules                                                            |
| -------------- | ---------------------------------------------------------------- |
| `name`         | required, short label                                            |
| `base_path`    | required, single segment `^/[a-zA-Z0-9._~-]+$`, globally unique |
| `upstream`     | required `http(s)://host[:port]` origin (path not used)          |
| `methods`      | array, e.g. `["GET","POST"]`                                     |
| `requires_auth`| if true nginx calls `/internal/auth` first                       |
| `rate_limit_rpm`| integer > 0, requests/minute per IP (also the zone name)         |
| `is_active`    | false → location not generated                                    |
| `note`         | optional free text                                               |

### Example — register an upstream and verify exposure

```bash
TOKEN=$(curl -s -X POST http://localhost:18080/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"admin@example.com","password":"admin123"}' \
  | python3 -c "import sys,json;print(json.load(sys.stdin)['data']['access_token'])")

# expose the bundled mock IdP at /idp, JWT-protected, 120 req/min
curl -s -X POST http://localhost:18080/api/v1/gateway/apis \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"name":"OIDC IdP","base_path":"/idp","upstream":"http://mock-idp:9090",\
       "methods":["GET"],"requires_auth":true,"rate_limit_rpm":120,"is_active":true}'

# protected: 401 for anonymous, then proxied with a valid token
curl -s -o /dev/null -w "%{http_code}\n" http://localhost:18080/idp/.well-known/openid-configuration   # 401
curl -sL -H "Authorization: Bearer $TOKEN" http://localhost:18080/idp/.well-known/openid-configuration # 200/302
```

## Generated nginx config

The API renders each registered API into `locations/reg_<id>.conf` and the
rate-limit zones into `zones.conf` on the shared volume:

```
# zones.conf  (http context — defines shared rate-limit zones)
limit_req_zone $binary_remote_addr zone=reg_1_120:2m rate=120r/m;
```

```
# locations/reg_1.conf  — location for /idp
location = /_gw_auth_1 {                      # internal auth_request subrequest
    internal;
    proxy_set_header Authorization $http_authorization;
    proxy_set_header X-Gateway-Token $gateway_token;   # explicit (does not inherit)
    proxy_pass http://api:8080/internal/auth;
}
location ^~ /idp/ {
    auth_request /_gw_auth_1;                 # 401/403 blocks; error_page 403=@denied
    limit_req zone=reg_1_120 burst=120 nodelay;
    if ($request_method !~ ^GET$) { return 405; }
    rewrite ^/idp$ / break;                    # normalize (no double slash)
    rewrite ^/idp(/.*)$ $1 break;
    proxy_pass http://mock-idp:9090;
    access_log /etc/nginx/registry/logs/reg_1.json gateway_json;
}
```

Details worth knowing:

- `auth_request` subrequests get their **own** `proxy_set_header`, which stops
  server-level header inheritance in nginx — so `X-Gateway-Token` must be set
  explicitly on the subrequest (it references `$gateway_token` defined
  server-wide by the entrypoint).
- A location that returns `403`/`401` from `auth_request` is surfaced through
  the generic `error_page 403 = @denied` JSON body — that is how the
  gateway's own deny message appears.
- The zone name embeds `rate_limit_rpm`, so changing the limit creates a fresh
  nginx zone (nginx cannot mutate zone params live) — simply republish.

## Hot reload

`deployments/nginx/entrypoint.sh` watches the registry directory in a loop:
`watch_registry()` sha1sums `reg_*.conf` + `zones.conf` every 1s and runs
`nginx -s reload` when the digest changes. The entrypoint also creates the
registry dirs and an empty `zones.conf` so a clean first boot doesn't fail on
the `include` directive.

## Monitoring

Two goroutines run in the API when `GATEWAY_CONFIG_DIR` is set:

- **Health checker** — probes `GET <upstream>/healthz` every 30s; any HTTP
  response counts as healthy, a transport error marks the API unhealthy. The
  status is persisted to `gateway_apis.status` and surfaced in `GET /apis`.
- **Aggregator** — tails `logs/reg_*.json` (nginx JSONL) every 2s, pushes
  today's counters to Redis hash `gwstats:<id>:<YYYYMMDD>` (`count`, `2xx`,
  `4xx`, `5xx`, `errors`, `rt_total_ms`) and keeps the last 100 entries in an
  in-memory ring. `GET /apis/{id}/stats` reads Redis for the totals and the
  ring for recent entries.

## Shared registry volume

```
gwregistry (named volume)
 ├─ zones.conf              http-level include (rate-limit zones)
 ├─ locations/reg_1.conf    server-level include (per-API blocks)
 └─ logs/reg_1.json         per-API JSONL access logs (gateway_json format)
```
Mounted at `/etc/nginx/registry` in nginx and `/app/gateway-configs` in the
API. The API writes configs and reads logs; nginx reads configs and writes
logs. `GATEWAY_AUTH_UPSTREAM` tells nginx where the API's internal auth
endpoint lives.

## Management console (`/admin`)

A Vue 3 SPA — **Apple-style mobile-first UI**, built with a **micro frontend
architecture**. It lets operators, without touching nginx:

- **Dashboard** — summary cards + quick navigation.
- **Registry** — register/edit/delete/restore exposed APIs (maps 1:1 to the
  REST API above).
- **Simulate** — pick an API and preview the *exact* nginx location nginx will
  apply, then `Apply & reload` (POST `/publish`).
- **Monitoring** — live counters + recent requests per API (5s refresh).
- **Docs** — Swagger UI iframe + list of dynamically exposed routes.

### Micro frontends

Each feature is an independently built ESM module and stylesheet:

```
frontend/
  vite.config.js                 shell build  → dist/
  vite.mf.{registry,simulator,monitoring,docs}.config.js  → dist/mf/<name>/{index.js,style.css}
  src/micro/<name>/              one self-contained feature app per micro
  src/views/MicroHost.vue        runtime loader (dynamic import + css injection)
  public/mf-manifest.json        name → js/css URLs served at /admin/mf-manifest.json
```

The shell renders inside `/admin/`; `MicroHost` fetches the manifest and
composes the right micro via `import(m.js)` plus one-time style injection.
Micros can be built, versioned, and deployed independently — swap a row in the
manifest (or point at another CDN) to hot-swap features.

### Build

```bash
cd frontend && npm install
npm run build      # shell + all four micros
```

`dist/` is mounted into the gateway container at `/usr/share/nginx/html`
(bind mount; recreate the gateway if you `rm -rf dist` on the host, because
the mount follows the old directory inode).

## Security notes

- The Go API has **no published host port**; only nginx exposes `:18080`.
- Registered upstreams are reachable **only** through the gateway, so they
  inherit its rate limits, WAF filters, headers, and (optionally) JWT checks.
- `auth_request` returns `401` for anonymous and `403 = @denied` when the API
  rejects the shared-secret handshake (misconfigured `APP_GATEWAY_TOKEN`).
- Rotate `APP_GATEWAY_TOKEN` in both `APP_GATEWAY_TOKEN` and nginx
  (`deployments/nginx/entrypoint.sh` reads it from the same env).
- Set `APP_SWAGGER_ENABLED=false` and harden the admin role/credentials
  (`migrations/009_bootstrap_admin.sql`) outside local dev.