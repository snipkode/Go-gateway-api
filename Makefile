# ── Go Enterprise API · deployment automation ───────────────────────────────
# Targets for local dev (Docker Compose) and Kubernetes (kind/minikube/cluster).
#
#   make deploy           # docker: full first-time bring-up
#   make redeploy         # docker: rebuild + recreate app containers
#   make k8s-build k8s-apply   # kubernetes: build images + apply manifests
#
COMPOSE_FILE := deployments/docker-compose.yml
KUBE_DIR     := deployments/kubernetes
NS           := go-enterprise
IMG_PREFIX   ?= go-enterprise   # set to a registry (e.g. ghcr.io/<user>) for k8s-push
VERSION      ?= latest

.DEFAULT_GOAL := help

help:            ## List targets
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}'

# ── Setup ───────────────────────────────────────────────────────────────────
.PHONY: setup token install
setup:           ## Create .env from .env.example (never overwrites)
	@test -f .env || cp .env.example .env
	@echo ".env ready"

token:           ## Print a fresh gateway-token value
	python3 -c "import secrets;print(secrets.token_urlsafe(32))"

install:         ## Install frontend dependencies
	cd frontend && npm install

# ── Frontend ────────────────────────────────────────────────────────────────
.PHONY: build-frontend
build-frontend:  ## Build the console shell + all micro-frontends into dist/
	cd frontend && npm run build

# ── Docker Compose ──────────────────────────────────────────────────────────
.PHONY: build up down restart logs ps redeploy deploy health
build:           ## Build all images
	docker compose -f $(COMPOSE_FILE) build

up: build        ## Bring the whole stack up (builds + starts)
	docker compose -f $(COMPOSE_FILE) up -d

down:            ## Stop and remove containers (volumes kept)
	docker compose -f $(COMPOSE_FILE) down

restart:         ## Restart all services
	docker compose -f $(COMPOSE_FILE) restart

logs:            ## Tail the API + gateway logs
	docker compose -f $(COMPOSE_FILE) logs -f api gateway

ps:              ## Show running services
	docker compose -f $(COMPOSE_FILE) ps

redeploy:        ## Rebuild + recreate the app containers (after code/config changes)
	docker compose -f $(COMPOSE_FILE) up -d --build --force-recreate api gateway mock-idp

health:          ## Gateway + API smoke checks
	@curl -sf -o /dev/null http://localhost:18080/healthz && echo "gateway/healthz  ✔"
	@curl -sf -o /dev/null http://localhost:18080/admin/ && echo "console /admin/   ✔"
	@curl -sf -o /dev/null http://localhost:18080/swagger/index.html && echo "swagger          ✔"

deploy: setup install build-frontend up ## Full local deploy (docker)
	@echo "→ http://localhost:18080"

# ── Go toolchain ────────────────────────────────────────────────────────────
.PHONY: fmt vet tidy swag
fmt:             ## gofmt + go vet
	gofmt -l $$(find . -name '*.go' -not -path './vendor/*'); go vet ./...
tidy:            ## go mod tidy
	go mod tidy && cd deployments/mock-idp && go mod tidy
swag:            ## Regenerate OpenAPI docs
	swag init -g cmd/api/main.go -o docs --parseDependency --parseInternal

# ── Kubernetes ──────────────────────────────────────────────────────────────
.PHONY: k8s-secret k8s-build k8s-push k8s-apply k8s-delete k8s-logs
k8s-secret:      ## Create/replace the app Secret from .env (run after setup)
	kubectl delete secret app-secret -n $(NS) --ignore-not-found 2>/dev/null || true
	kubectl create secret generic app-secret -n $(NS) --from-env-file=.env --dry-run=client -o yaml | kubectl apply -f -

k8s-build: build-frontend ## Build all images locally (api, gateway, mock-idp)
	docker compose -f $(COMPOSE_FILE) build api mock-idp
	docker build -f deployments/nginx/Dockerfile -t $(IMG_PREFIX)-gateway:$(VERSION) .

k8s-push:        ## Push images (set IMG_PREFIX to a real registry first)
	@test "$(IMG_PREFIX)" != "go-enterprise" || (echo "set IMG_PREFIX to your registry (e.g. ghcr.io/<user>)" && exit 1)
	docker tag $(IMG_PREFIX)-api:latest      $(IMG_PREFIX)-api:$(VERSION)
	docker tag $(IMG_PREFIX)-gateway:$(VERSION) $(IMG_PREFIX)-gateway:$(VERSION)
	docker tag $(IMG_PREFIX)-mock-idp:latest $(IMG_PREFIX)-mock-idp:$(VERSION)
	docker push $(IMG_PREFIX)-api:$(VERSION)
	docker push $(IMG_PREFIX)-gateway:$(VERSION)
	docker push $(IMG_PREFIX)-mock-idp:$(VERSION)

k8s-apply:       ## Deploy everything (creates namespace, secrets, workloads)
	kubectl create namespace $(NS) --dry-run=client -o yaml | kubectl apply -f -
	kubectl apply -k $(KUBE_DIR)
	kubectl rollout status deployment/gateway -n $(NS) --timeout=180s

k8s-delete:      ## Remove all deployed resources
	kubectl delete -k $(KUBE_DIR)
	kubectl delete namespace $(NS) --ignore-not-found

k8s-logs:        ## Tail gateway pod logs (both containers)
	kubectl logs -l app=gateway -n $(NS) --all-containers=true -f