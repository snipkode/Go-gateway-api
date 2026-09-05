# Swagger / OpenAPI documentation

The HTTP API exposes an OpenAPI 2.0 spec generated from Go comments via
[swag](https://github.com/swaggo/swag). The UI is served at
`/swagger/index.html` and the machine-readable specs at `/swagger/doc.json`
and `/swagger/swagger.json`.

| Path                        | Content                        |
| --------------------------- | ------------------------------ |
| `/swagger/index.html`       | Swagger UI (browser)           |
| `/swagger/doc.json`         | OpenAPI 2.0 spec (JSON)        |
| `/swagger/swagger.json`     | Spec (JSON, may be versioned)  |
| `/admin/docs`               | Swagger UI embedded in the management console |

> `/swagger/*` is a dev convenience. It is disabled in production by setting
> `APP_SWAGGER_ENABLED=false` — the paths then answer `404`.

## Enabling / disabling

```bash
# compose override or shell env
APP_SWAGGER_ENABLED=true   # serve
APP_SWAGGER_ENABLED=false  # hide in production
```

## How handlers are annotated

Handlers carry swag doc-comments on the exported method:

```go
// List godoc
// @Summary      List registered gateway APIs
// @Description  Returns the APIs exposed through the gateway.
// @Tags         Gateway
// @Produce      json
// @Success      200 {array} gatewayapi.GatewayAPI
// @Failure      401 {object} httpapi.ErrorResponse
// @Router       /gateway/apis [get]
// @Security     BearerAuth
```

The Go API docs live in `docs/` (generated: `docs/docs.go`,
`docs/swagger.json`, `docs/swagger.yaml`) and are **compiled into the binary**,
so the endpoint works without mounting any files.

## Regenerating after handler changes

Requires the `swag` binary (v1.16.x used in this repo):

```bash
# from the repository root
swag init -g cmd/api/main.go -o docs --parseDependency --parseInternal
```

Notes:

- `--parseDependency` resolves types from imported packages (e.g. domain
  entities and infrastructure models).
- After regenerating, rebuild the API image: migrations and docs are baked
  into the image (`docker compose build api && docker compose up -d api`).

## Console integration

The management console's **Docs** tab loads `/swagger/index.html` in an
iframe and lists the dynamically registered gateway routes beside it — see
[docs/GATEWAY.md](GATEWAY.md).