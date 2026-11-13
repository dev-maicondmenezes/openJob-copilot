# syntax=docker/dockerfile:1

# openJob — multi-stage build (Checkpoint 1).
# Stage 1 (Node) will build the real SvelteKit SPA in Checkpoint 2; for now it
# stages the placeholder assets. Stage 2 produces a static Go binary with the
# SPA embedded via //go:embed, so the runtime stage ships only the binary.

# ---- Stage 1: SPA assets (placeholder until Checkpoint 2) ----
FROM node:22-alpine AS spa
WORKDIR /app
# CP2 replaces this COPY with the SvelteKit build:
#   COPY frontend/ ./ && RUN npm ci && RUN npm run build -> dist
COPY backend/web/dist/ /out/dist/

# ---- Stage 2: Go build (static binary, no CGO — keeps modernc.org/sqlite
# compatible and the image tiny; decision recorded in docs/adr/0002) ----
FROM golang:1.27-alpine AS build
WORKDIR /src
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ .
COPY --from=spa /out/dist/ ./web/dist/
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/openjob ./cmd/openjob

# ---- Stage 3: runtime ----
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/openjob /openjob
EXPOSE 8080
USER nonroot
ENTRYPOINT ["/openjob"]
