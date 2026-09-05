# Deployment — Makefile automation

The repository ships a `Makefile` that drives both deployment targets:

| runtime   | purpose                                  | entrypoint    |
| --------- | ---------------------------------------- | ------------- |
| **Docker** | full local/dev stack via Docker Compose  | `make deploy` |
| **Kubernetes** | cluster deployment via kustomize    | `make k8s-apply` |

Both paths build the same artifacts:

- **Frontend** micro-frontend console (shell + 4 micro apps) → `frontend/dist`
- **Go API** image (`go-enterprise-api`)
- **Gateway** image (`go-enterprise-gateway`) — nginx + baked console dist
- **Mock IdP** image (`mock-idp`) — local OIDC provider

`make help` prints every target with a short description.

---

## 0. Prerequisites

| tool            | version/notes                                   |
| --------------- | ----------------------------------------------- |
| `make`          | any POSIX make                                  |
| `docker` + compose v2 | required by `make deploy` / `make k8s-build` |
| `node` + `npm`  | Node 18+ for the console build                  |
| `kubectl`       | only for the Kubernetes targets                 |
| `kind`/`minikube` (or a cluster) | only for `make k8s-apply` |

Kubernetes images are expected to exist **in the local Docker daemon**; the
manifests use `imagePullPolicy: IfNotPresent`, so `kind`/`minikube` pick them
up without a registry.

---

## 1. Docker Compose deployment

```bash
make deploy
```

`deploy` chains: `setup` (create `.env` from `.env.example`) → `install`
(npm) → `build-frontend` → `build` (images) → `up` (start stack).

Resulting services:

| service   | exposed    | notes                                     |
| --------- | ---------- | ----------------------------------------- |
| gateway   | `:18080`   | only public entry point                   |
| api       | (internal) | reached only via the gateway              |
| mock-idp  | `:9090`    | OIDC SSO demo                             |
| postgres  | `:15432`   |                                     |
| redis     | `:16379`   |                                     |

Verify:

```bash
make health
# gateway/healthz  ✔
# console /admin/   ✔
# swagger          ✔
```

Access the management console at http://localhost:18080/admin/
(`admin@example.com` / `admin123`).

### Day-2 operations (Docker)

| command          | effect                                                        |
| ---------------- | ------------------------------------------------------------- |
| `make redeploy`  | rebuild + recreate `api gateway mock-idp` (after changes)     |
| `make logs`      | tail API + gateway logs                                       |
| `make ps`        | service status                                                |
| `make restart`   | restart all services                                          |
| `make down`      | stop containers (volumes kept)                                |
| `make setup`     | create `.env` (never overwrites)                              |
| `make token`     | print a fresh `APP_GATEWAY_TOKEN` for `.env`                  |

> **File edits to Go code** need only `make redeploy`. **Generated gateway
> configs** are hot-reloaded by the gateway entrypoint — but after Go-side
> template changes, re-render them once via `POST /api/v1/gateway/publish`.
> If you ever `rm -rf frontend/dist`, recreate the gateway container too, or
> the read-only bind mount keeps the old inode:
> `docker compose -f deployments/docker-compose.yml up -d --force-recreate gateway`.

---

## 2. Kubernetes deployment

Targets a **kind**/**minikube** cluster or any k8s with `kubectl` in PATH.

```bash
# 1) local + secrets
make setup                          # .env from .env.example
make k8s-secret                     # project Secret from .env

# 2) build images                    (api + mock-idp via compose, gateway via its own Dockerfile)
make k8s-build

# 3) deploy
make k8s-apply
```

`k8s-apply` runs `kubectl apply -k deployments/kubernetes` then waits for the
gateway rollout. The kustomize app creates:

| resource       | kind                       | notes                                            |
| -------------- | -------------------------- | ------------------------------------------------ |
| `go-enterprise`| Namespace                  |                                                  |
| `app-config`   | ConfigMap                  | non-secret runtime env                           |
| `app-secret`   | Secret                     | from `.env` (`k8s-secret`)                       |
| `postgres`     | StatefulSet + PVC + Service| storage 1Gi                                     |
| `redis`        | Deployment + Service       |                                                  |
| `mock-idp`     | Deployment + Service       |                                                  |
| `gateway`      | Deployment + Service       | `NodePort 30080`; nginx + api **in one Pod**     |

The gateway Pod runs **nginx and the Go API as sidecar containers sharing an
`emptyDir`** registry volume — the same hot-reload behaviour as Docker
Compose without a shared PVC. nginx uses `APP_UPSTREAM_API=127.0.0.1:8080`
since the API shares its network namespace.

Access the console: `http://<cluster-node>:30080/admin/` (on kind, ask kind to
forward the node port with `kind create cluster --config ...` or use a port
forward).

### Day-2 operations (Kubernetes)

| command        | effect                                              |
| -------------- | --------------------------------------------------- |
| `make k8s-logs`| tail both gateway pod containers (`-f`)             |
| `make k8s-delete` | remove all deployed resources (incl. namespace)  |
| `make k8s-push`   | push images to `${IMG_PREFIX}` registry         |

### Tagging & registry push

Image names are deliberately fixed so Compose, the Makefile and the manifests
agree:

```
go-enterprise-api:latest   (compose api service)
go-enterprise-gateway:latest (deployments/nginx/Dockerfile)
mock-idp:latest            (compose mock-idp service)
```

To publish to a registry instead of a local cluster, override the Makefile
variables:

```bash
make k8s-build IMG_PREFIX=ghcr.io/snipkode VERSION=v1.0.0
make k8s-push  IMG_PREFIX=ghcr.io/snipkode VERSION=v1.0.0
```

then change the `image:` references (or add a kustomize
`images:` transformer) in `deployments/kubernetes/*.yaml`.

---

## 3. Configuration reference

| source          | consumed by                       |
| --------------- | --------------------------------- |
| `deployments/docker-compose.yml` | compose services, env, volumes |
| `deployments/kubernetes/configmap.yaml` | non-secret runtime env |
| `deployments/kubernetes/secret.yaml` (overwritten by `make k8s-secret`) | secrets |
| `.env` (repo root) | compose `env_file`, `make k8s-secret` |

All variables are documented in [docs/ENVIRONMENT.md](ENVIRONMENT.md).

Key values:

- `APP_GATEWAY_TOKEN` — shared secret between nginx and the API; keep it
  unique per environment (`make token`).
- `GATEWAY_CONFIG_DIR` — registry volume path (`/app/gateway-configs` in
  compose, `/app/gateway-configs` in k8s; empty disables registry/monitoring).
- `GATEWAY_AUTH_UPSTREAM` — where nginx sends `auth_request` JWT checks
  (`http://api:8080` compose, `http://localhost:8080` k8s).
- `APP_UPSTREAM_API` — nginx → API upstream target
  (`api:8080` compose default, `127.0.0.1:8080` in the k8s gateway Pod).

---

## 4. Troubleshooting

| symptom                                  | fix                                                               |
| ---------------------------------------- | ----------------------------------------------------------------- |
| console 403/500 after `rm -rf dist`      | recreate the gateway container (stale bind-mount inode)           |
| nginx `host not found in upstream "api"` | start `api` first / ensure compose DNS; set `APP_UPSTREAM_API`    |
| `ImagePullBackOff` on k8s                | images not in local daemon → run `make k8s-build` first           |
| stale gateway configs                    | `POST /api/v1/gateway/publish` to re-render generated includes    |
| `make` errors with `unknown shorthand flag '-g'` | `IMG_PREFIX` has trailing whitespace → keep the comment off the value line |
| Swagger disabled                        | set `APP_SWAGGER_ENABLED=true` (dev only)                          |