.PHONY: build build-spa run dev dev-api dev-web test test-ui vet tidy clean \
	smoke docker-build docker-smoke snapshot

# Build identity injected into internal/version (CP11). Local builds default to
# "dev"; releases get the real values from GoReleaser / the Docker build args.
VERSION ?= dev
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
PKG     := github.com/maicondmenezes/openjob/internal/version
LDFLAGS := -s -w -X $(PKG).Version=$(VERSION) -X $(PKG).Commit=$(COMMIT) -X $(PKG).Date=$(DATE)
IMAGE   ?= openjob:local

# Dev server completo: Go com hot reload (air) + SPA com HMR (Vite), em paralelo.
# - SPA em http://localhost:5173 (proxy de /api para o Go, sem CORS)
# - API em http://localhost:8080
# Celular: use http://openjob.copilot:5173 (mesma rede Wi-Fi).
dev:
	@$(MAKE) -j2 dev-api dev-web

# dev-api roda o servidor Go com hot reload via air (instalado no primeiro uso).
dev-api:
	@air_path=$$(command -v air || echo "$(shell go env GOPATH)/bin/air"); \
	if [ ! -x "$$air_path" ]; then \
		echo "==> instalando air (github.com/air-verse/air)"; \
		(cd backend && go install github.com/air-verse/air@latest); \
	fi; \
	echo "==> air: Go com hot reload (:8080)"; \
	(cd backend && OPENJOB_DB_PATH=../data/openjob.db "$$air_path" -c .air.toml)

# dev-web roda o Vite dev server com HMR (:5173, /api via proxy).
dev-web:
	@echo "==> vite: SPA com HMR (:5173)"
	(cd frontend && npm run dev)

# build-spa builds the SvelteKit SPA (adapter-static + PWA) and publishes the
# output to backend/web/dist — the //go:embed target. Skipped when deps are
# already installed (npm install is only needed once).
build-spa:
	@if [ ! -d frontend/node_modules ]; then \
		echo "==> npm install (frontend)"; \
		(cd frontend && npm install --no-audit --no-fund); \
	fi
	@echo "==> npm run build (frontend -> backend/web/dist)"
	(cd frontend && npm run build)

# build compiles the static binary (CGO off — see docs/adr/0002) with the SPA
# embedded via //go:embed. SPA first, so the binary always ships fresh assets.
build: build-spa
	@echo "==> go build (backend)"
	(cd backend && CGO_ENABLED=0 go build -trimpath -ldflags="$(LDFLAGS)" -o ../bin/openjob ./cmd/openjob)

# run starts the server locally (without Docker) on :8080. DB and master key
# live in ./data (the /data default is the container volume path).
run: build
	OPENJOB_PORT=8080 OPENJOB_DB_PATH=./data/openjob.db ./bin/openjob serve

# smoke runs the binary smoke test (health, key permissions, restart keeps key).
smoke: build
	scripts/smoke-binary.sh bin/openjob

# docker-build builds the official image locally; docker-smoke then exercises it.
docker-build:
	docker build --build-arg VERSION=$(VERSION) --build-arg COMMIT=$(COMMIT) --build-arg DATE=$(DATE) -t $(IMAGE) .

docker-smoke: docker-build
	scripts/smoke-docker.sh $(IMAGE)

# snapshot builds every release artifact locally without publishing anything
# (needs goreleaser; the SPA must be built first so it is embedded).
snapshot: build-spa
	goreleaser release --snapshot --clean --skip=publish

test: build-spa
	cd backend && go test ./...

# Verificações de UI do CP12: tipos, lint de tokens e testes Playwright.
test-ui:
	cd frontend && npm run check && npm run i18n:check && npm run lint:ui && npm run test:e2e && npm run audit:a11y && npm run test:visual

vet:
	cd backend && go vet ./...

tidy:
	cd backend && go mod tidy

clean:
	rm -f bin/openjob
	rm -rf backend/tmp frontend/build frontend/.svelte-kit
	find backend/web/dist -mindepth 1 ! -name '.gitkeep' -delete
