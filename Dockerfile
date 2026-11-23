# syntax=docker/dockerfile:1

# openJob — official image (CP11, docs/adr/0036-imagem-docker-oficial.md).
#
# Headless server only: no wizard, no interactive bootstrap, no browser, no LAN
# discovery. Every operational setting is an environment variable; database and
# master key live in the /data volume. First-use setup happens in the web app.
#
# Stages 1–2 run on the *build* host's native architecture and cross-compile for
# $TARGETARCH (pure Go, CGO off — docs/adr/0002), so linux/arm64 images don't
# need slow emulation to build.

# ---- Stage 1: SPA (SvelteKit + PWA) ----
FROM --platform=$BUILDPLATFORM node:22-alpine AS spa
WORKDIR /app/frontend
COPY frontend/package.json frontend/package-lock.json ./
# `npm ci` runs the package's `prepare` script (paraglide i18n compile), which
# reads the inlang project and the message catalogs: they must exist *before*
# the install, not only after `COPY frontend/ .`.
COPY frontend/project.inlang ./project.inlang
COPY frontend/messages ./messages
RUN npm ci --no-audit --no-fund
COPY frontend/ .
# npm run build = vite build + copy to ./backend/web/dist (relative to frontend/)
RUN npm run build

# ---- Stage 2: Go build (static binary, SPA embedded via //go:embed) ----
FROM --platform=$BUILDPLATFORM golang:1.27-alpine AS build
# GOPROXY é configurável no build (rede local lenta → use um mirror, ex.:
#   docker compose build --build-arg GOPROXY=https://goproxy.cn,direct)
ARG GOPROXY=https://proxy.golang.org,direct
ENV GOPROXY=$GOPROXY
ARG TARGETOS
ARG TARGETARCH
ARG VERSION=dev
ARG COMMIT=none
ARG DATE=unknown
WORKDIR /src
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ .
COPY --from=spa /app/backend/web/dist/ ./web/dist/
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -trimpath \
    -ldflags="-s -w \
      -X github.com/maicondmenezes/openjob/internal/version.Version=${VERSION} \
      -X github.com/maicondmenezes/openjob/internal/version.Commit=${COMMIT} \
      -X github.com/maicondmenezes/openjob/internal/version.Date=${DATE}" \
    -o /out/openjob ./cmd/openjob
# Empty data dir to be copied with the right owner: a fresh named volume
# inherits ownership from the image path, so the non-root user can write to it.
RUN mkdir -p /out/data

# ---- Stage 3: minimal runtime (no shell, no toolchain, no Node) ----
FROM gcr.io/distroless/static-debian12:nonroot
ARG VERSION=dev
ARG COMMIT=none
ARG DATE=unknown
ARG SOURCE=https://github.com/dev-maicondmenezes/openJob-copilot
# TODO(owner): replace NOASSERTION with the project's SPDX license id once a
# LICENSE file is added to the repository.
ARG LICENSE=NOASSERTION

LABEL org.opencontainers.image.title="openJob" \
      org.opencontainers.image.description="Local-first job application copilot (Go + SvelteKit, SQLite, BYOK)" \
      org.opencontainers.image.source="${SOURCE}" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.revision="${COMMIT}" \
      org.opencontainers.image.created="${DATE}" \
      org.opencontainers.image.licenses="${LICENSE}"

COPY --from=build /out/openjob /openjob
COPY --from=build --chown=65532:65532 /out/data /data

# Operational configuration — the complete list is in docs/distribution.md.
# OPENJOB_FEATURE_* flags (default off) are also read from the environment.
ENV OPENJOB_PORT=8080 \
    OPENJOB_DB_PATH=/data/openjob.db \
    OPENJOB_HOSTNAME=openjob.copilot \
    OPENJOB_CONTAINER=true

WORKDIR /data
VOLUME ["/data"]
EXPOSE 8080

# Numeric non-root user (distroless "nonroot"): also satisfies runAsNonRoot.
USER 65532:65532

# The runtime has no curl/wget: the binary probes /api/health itself and reads
# the same OPENJOB_PORT as the server.
HEALTHCHECK --interval=30s --timeout=5s --start-period=20s --retries=3 \
    CMD ["/openjob", "healthcheck"]

ENTRYPOINT ["/openjob"]
CMD ["serve"]
