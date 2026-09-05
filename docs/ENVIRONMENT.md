# Environment variables

All settings are read from the environment (see `internal/config/config.go`).
The repository ships a ready-to-copy template: `.env.example`. Non-empty values
override the compiled defaults.

> Secrets (`APP_GATEWAY_TOKEN`, `APP_JWT_SECRET`, `OAUTH_CLIENT_SECRET`,
> DB/Redis passwords) must come from a secret manager in production. Never
> commit a real `.env`.

## Application

| Variable                | Default                    | Description                                                                 |
| ----------------------- | -------------------------- | --------------------------------------------------------------------------- |
| `APP_ENV`               | `development`              | Log/behaviour switches (`development`, `staging`, `production`).            |
| `APP_PORT`              | `8080`                     | HTTP port the API listens on (internal; only nginx needs to reach it).      |
| `APP_ALLOWED_ORIGINS`   | `*`                        | Comma-separated CORS allow-list for the browser console/dev SPA.            |
| `APP_LOGGER_LEVEL`      | `debug`                    | `debug` / `info` / `warn` / `error`. Structured JSON logs.                  |
| `APP_GATEWAY_TOKEN`     | `""`                       | Shared secret the nginx gateway sends via `X-Gateway-Token` on every proxied request. Must match nginx. |
| `APP_SWAGGER_ENABLED`   | `true`                     | Mount `/swagger/*` (OpenAPI + swagger-ui). Set `false` in production.       |
| `APP_JWT_SECRET`        | `change-me-in-production`  | HS256 signing key. **Must change in production.**                            |
| `APP_JWT_TTL`           | `15m`                      | Access-token lifetime (Go duration, e.g. `15m`, `1h`).                       |
| `APP_SESSION_TTL`       | `15m`                      | Server-side session TTL in Redis (bounded by `APP_JWT_TTL`).                |

## PostgreSQL

| Variable             | Default          | Description                                            |
| -------------------- | ---------------- | ------------------------------------------------------ |
| `POSTGRES_HOST`      | `localhost`      | Host (or docker service name `postgres`).              |
| `POSTGRES_PORT`      | `5432`           |                                                        |
| `POSTGRES_USER`      | `app`            |                                                        |
| `POSTGRES_PASSWORD`  | `app`            |                                                        |
| `POSTGRES_DB`        | `go_enterprise`  | Database name; schema auto-migrates on API startup.    |
| `POSTGRES_SSLMODE`   | `disable`        | `disable` for docker-network/local, `require` for remote managed Postgres. |

## Redis

| Variable         | Default | Description                          |
| ---------------- | ------- | ------------------------------------ |
| `REDIS_HOST`     | `localhost` | Host (or service name `redis`).   |
| `REDIS_PORT`     | `6379`  |                                      |
| `REDIS_PASSWORD` | `""`    | Empty for the bundled container.     |
| `REDIS_DB`       | `0`     | Logical DB index.                    |

Redis-backed state: `sessions:*`, `ratelimit:*`, dynamic rule cache, and the
gateway per-API daily counters `gwstats:<apiID>:<YYYYMMDD>`.

## Microsoft Entra (OIDC) SSO

Leave `OAUTH_CLIENT_ID`/`OAUTH_TENANT_ID` empty to keep SSO disabled (the
Entra endpoints then answer `503`).

| Variable                | Default | Description                                                             |
| ----------------------- | ------- | ----------------------------------------------------------------------- |
| `OAUTH_CLIENT_ID`       | `""`    | Entra (or mock IdP) application client id.                              |
| `OAUTH_CLIENT_SECRET`   | `""`    | Client secret, sent server-side during the code exchange.               |
| `OAUTH_TENANT_ID`       | `""`    | Tenant id, used to build the discovery/JWKS URLs for the common issuer. |
| `OAUTH_ISSUER`          | `""`    | Full issuer URL. Set it to a real tenant (`https://login.microsoftonline.com/<tenant>/v2.0`) or to the local mock (`http://mock-idp:9090`). If empty, the issuer is derived from `OAUTH_TENANT_ID`. |
| `OAUTH_REDIRECT_URL`    | `""`    | Public callback URL; must equal the redirect URI registered in the app. |
| `OAUTH_FRONTEND_URL`    | `""`    | When set, the callback 302-redirects to `<url>#access_token=<jwt>` instead of returning JSON (SPA flow). |
| `OAUTH_DEFAULT_ROLE`    | `viewer`| Role slug assigned to SSO users on first (JIT) sign-in.                 |

See [ENTRA_SSO.md](ENTRA_SSO.md) for the full design + mock IdP mode.

## API Gateway registry

Wired only in Docker (see `deployments/docker-compose.yml`). Used by the
management console and the nginx hot-reload publisher.

| Variable                | Default            | Description                                                          |
| ----------------------- | ------------------ | -------------------------------------------------------------------- |
| `GATEWAY_CONFIG_DIR`    | `""`               | API-side mount of the shared registry volume. Empty disables publishing and monitoring (safe for no-Docker dev). |
| `GATEWAY_AUTH_UPSTREAM` | `http://api:8080`  | Where nginx's per-API `auth_request` subrequests reach this API's `GET /internal/auth`. |

## Docker compose

`deployments/docker-compose.yml` pulls the values from the repository-root
`.env` and adds container-only overrides (`POSTGRES_HOST=postgres`,
`REDIS_HOST=redis`, mock IdP OAuth settings, gateway env). It also injects
`APP_GATEWAY_TOKEN` into nginx via `deployments/nginx/entrypoint.sh`.

The nginx image is configured at runtime from these env vars:

| Variable | Description |
| -------- | ----------- |
| `APP_GATEWAY_TOKEN` | Value threaded into `gateway-token.conf` (`$gateway_token`), used both for the main `/` proxy `X-Gateway-Token` header and by the generated per-API auth subrequests. |