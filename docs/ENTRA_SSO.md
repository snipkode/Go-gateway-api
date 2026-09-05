# Microsoft Entra (OIDC) SSO

The API implements the full **OIDC authorization-code flow** against Microsoft
Entra ID using `github.com/coreos/go-oidc/v3` and
`golang.org/x/oauth2`. It is production-oriented (ID token signature and
claims verified, single-use CSRF state, JIT provisioning) and also runs against
a bundled **local mock IdP** so the flow is testable without Azure.

## Flow

```
Browser                                API (/auth/entra/login)
  │  1. GET  (no auth)                    │
  │ ◄─────────────────────────────────────┤  mint single-use CSRF state (in-memory,
  │                                       │  TTL 15m), 302 → IdP /authorize
  │                                      IdP
  │  2. 302 ── ID allows_login ─────────► authorize (user consents)
  │ ◄──── 302 ?code=…&state=… ────────────┘  (browser leg)
  │                                      API (/auth/entra/callback)
  │  3. GET ?code&state ► verify state (consume, single-use)
  │                       exchange code: client_id + client_secret
  │                       validate id_token: issuer, audience, RS256 signature
  │                                       (JWKS from discovery), exp, nonce
  │                       → profile {sub, email, email_verified, name}
  │  ◄──── LoginResult JSON (or 302 <frontend>#access_token=…) ────┘
  │  4. use access_token for /api/v1/*
```

Full chain: `login → state + authorize URL → IdP code → token exchange →
id_token verification → JIT provisioning → same JWT + Redis session as a
normal login`.

## JIT provisioning (Just-In-Time)

On a successful exchange the use case (`EntraLogin` in
`internal/application/auth/`):

1. Looks up the user by `provider = 'entra'` and `provider_id = token.sub`.
2. **Found** → the existing account is authenticated (roles intact).
3. **Not found** → creates the user with:
   - `provider = 'entra'`, `provider_id = sub`
   - `name`/`email` from the token claims
   - the role slotted in `OAUTH_DEFAULT_ROLE` (default `viewer`)
   - an unusable password hash (`!`), so password login can never hijack it
4. Issues `access_token` + Redis session, writes an audit `LOGIN` entry.

### Conflict handling

| Situation                                              | Behaviour                                              |
| ------------------------------------------------------ | ------------------------------------------------------ |
| New SSO user, email already used by a **local** user   | `409 sso_conflict` — never auto-merges                  |
| New SSO user, email used by a **different** entra sub  | `409 sso_conflict` — identity confusion guard           |
| Local login attempted on an SSO account                | rejected (empty password hash)                          |
| SSO not configured (`OAUTH_CLIENT_ID` empty, no issuer)| `503 sso_not_configured`                                |

## Security model

- **CSRF**: every login mints a one-time `state` (`entra/state.go`, TTL 15m);
  the callback verifies **and consumes** it. Replays of a used state fail.
- **Signature + claims**: the RS256 `id_token` is verified against the IdP's
  published JWKS from OIDC discovery; issuer (`iss`) and audience (`aud`) are
  checked by `go-oidc`; `exp` is enforced.
- **Server-side session**: SSO issues the JWT + Redis session pair like a
  normal login, so logout invalidates it and RBAC applies equally.
- **PKCE/nonce**: `go-oidc` validates the claims; for SPA/`OAUTH_FRONTEND_URL`
  deployments use a confidential client on the API side (no client secret in
  the browser).
- **In-memory state store** is per-instance: run a single API replica, or swap
  the `StateStore` interface for a Redis-backed one if you scale horizontally.

> The in-memory state store is intentionally single-instance. If you scale the
> API horizontally, implement a shared store (e.g. Redis with TTL) behind the
> same interface in `internal/infrastructure/entra/state.go`.

## Configuration

All values come from environment variables (see
[ENVIRONMENT.md](ENVIRONMENT.md)).

| Variable                | Purpose                                                            |
| ----------------------- | ------------------------------------------------------------------ |
| `OAUTH_CLIENT_ID`       | Entra app (client) ID                                              |
| `OAUTH_CLIENT_SECRET`   | Entra client secret                                                |
| `OAUTH_TENANT_ID`       | Tenant ID (used to build the default issuer)                       |
| `OAUTH_ISSUER`          | Full issuer URL; overrides tenant-based default (e.g. custom IdP)  |
| `OAUTH_REDIRECT_URL`    | Public callback URL — must match the registered redirect URI       |
| `OAUTH_FRONTEND_URL`    | If set, callback 302s to `<url>#access_token=…` (SPA flow)         |
| `OAUTH_DEFAULT_ROLE`    | Role slug assigned on first sign-in (must be seeded)               |

Provider initialisation: issuer = `OAUTH_ISSUER` if set, otherwise
`https://login.microsoftonline.com/<OAUTH_TENANT_ID>/v2.0`. The provider is
started when **both** `OAUTH_CLIENT_ID` is set **and** (`OAUTH_TENANT_ID` or
`OAUTH_ISSUER`) is set; otherwise SSO endpoints answer `503 sso_not_configured`
and the log line `entra oidc enabled` never appears.

---

## Setup against a real Microsoft tenant

1. **Register an app** — Azure portal → Microsoft Entra ID → App registrations →
   New registration (single tenant).
2. **Redirect URI** — under *Authentication → Web* add the public callback,
   e.g. `https://api.yourdomain.com/api/v1/auth/entra/callback` (and
   `http://localhost:18080/...` for local). For an SPA, this is the value of
   `OAUTH_REDIRECT_URL` verbatim.
3. **Client secret** — *Certificates & secrets → New client secret*; put it in
   `OAUTH_CLIENT_SECRET`.
4. **Environment** — set at least:

   ```env
   OAUTH_CLIENT_ID=<application client id>
   OAUTH_CLIENT_SECRET=<secret value>
   OAUTH_TENANT_ID=<directory tenant id>
   OAUTH_REDIRECT_URL=https://api.yourdomain.com/api/v1/auth/entra/callback
   OAUTH_DEFAULT_ROLE=viewer
   # SPA mode: uncomment to honour a login redirect instead of JSON
   # OAUTH_FRONTEND_URL=https://app.yourdomain.com
   ```

5. **Restart**: the provider runs discovery on startup. Confirm the log line
   `entra oidc enabled` with issuer `https://login.microsoftonline.com/<tenant>/v2.0`.
6. **First login** creates the user; verify:

   ```bash
   curl -s -L http://localhost:18080/api/v1/auth/entra/login
   # gives a JWT for the SSO identity; /auth/me shows roles incl. OAUTH_DEFAULT_ROLE
   ```

> `OAUTH_TENANT_ID` also accepts `common`, `organizations`, or `consumers` as
> the tenant value in the default issuer URL.

---

## Local demo mode (no Azure)

A tiny, dependency-free OIDC IdP ships in `deployments/mock-idp/` (RS256,
discovery + JWKS + authorize + token). The compose file points the API at it
out of the box:

```env
OAUTH_ISSUER=http://mock-idp:9090
OAUTH_CLIENT_ID=mock-client
OAUTH_CLIENT_SECRET=mock-secret
OAUTH_TENANT_ID=mock-tenant
OAUTH_DEFAULT_ROLE=viewer
```

### Try the full browser flow

```bash
docker compose -f deployments/docker-compose.yml up --build

# Browser leg: /login 302s to the mock; the mock 302s back with a code;
# the callback exchanges it, verifies the id_token, provisions the user.
# --resolve maps mock-idp:9090 to your machine because the mock answers on 9090.
curl -s -L --resolve mock-idp:9090:127.0.0.1 http://localhost:18080/api/v1/auth/entra/login
# → {"data":{"access_token":"…","user":{"email":"sso-user@example.com","roles":["viewer"],…}}}
```

Run it again — the same `sub` is returned, so the **existing** account is
reused (no duplicates, roles intact).

### Details of the mock

- Discovery: `/.well-known/openid-configuration`, edition of `http://mock-idp:9090`
- Keys: `/jwks` (RSA RS256), tokens signed at startup
- `authorize` requires `response_type=code` and a registered `redirect_uri`
- `token` requires client basic auth (`mock-client` / `mock-secret`)
- Identity: `email` from `login_hint` (default `sso-user@example.com`); the
  `sub` is derived from the email, so it is stable across logins — mirroring
  how Entra keeps a stable `sub` per user.

### Switching between mock and production

The demo values live in `deployments/docker-compose.yml` under the `api`
service (the `OAUTH_*` env block). For production, remove/override those with
real tenant values, set `OAUTH_ISSUER=https://login.microsoftonline.com/<tenant>/v2.0`
(or leave it unset and use `OAUTH_TENANT_ID`), and confirm the `entra oidc
enabled` startup log. Nothing else changes — the app code is identical.