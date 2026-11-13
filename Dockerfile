# syntax=docker/dockerfile:1

# openJob — multi-stage build (Checkpoint 2).
# Stage 1 (Node) builds the SvelteKit SPA (adapter-static → backend/web/dist,
# see frontend/scripts/copy-dist.mjs). Stage 2 produces a static Go binary with
# the SPA embedded via //go:embed, so the runtime stage ships only the binary.

# ---- Stage 1: SPA (SvelteKit + PWA) ----
FROM node:22-alpine AS spa
WORKDIR /app
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci --no-audit --no-fund
COPY frontend/ .
# npm run build = vite build + copy to ./backend/web/dist (relative to frontend/)
RUN npm run build

# ---- Stage 2: Go build (static binary, no CGO — keeps modernc.org/sqlite
# compatible and the image tiny; decision recorded in docs/adr/0002) ----
FROM golang:1.27-alpine AS build
WORKDIR /src
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ .
COPY --from=spa /app/backend/web/dist/ ./web/dist/
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/openjob ./cmd/openjob

# ---- Stage 3: runtime ----
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/openjob /openjob
EXPOSE 8080
USER nonroot
ENTRYPOINT ["/openjob"]
