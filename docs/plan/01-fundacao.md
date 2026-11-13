# Checkpoint 1 — Fundação (2–3 dias)

> Origem: spec §8, Checkpoint 1. Arquivo de referência: [`README.md`](README.md).

## Objetivo

Backend Go + `chi` servindo "Hello World" via `//go:embed`, containerizado em Docker multi-stage, acessível pelo **celular** via `http://openjob.copilot:8080` na mesma rede Wi-Fi.

## Pré-requisitos

- Go 1.22+ e Node 20+ instalados no host.
- Docker + Docker Compose funcionais.
- Celular na mesma rede Wi-Fi do notebook.

## Tarefas

1. **Estrutura do monorepo** (spec §6):
   - `backend/cmd/openjob/main.go`
   - `backend/internal/api/` (handlers chi)
   - `backend/internal/tenant/` (vazio, com `.gitkeep` — terreno para multi-tenancy)
   - `frontend/` (vazio por enquanto)
   - `Dockerfile`, `docker-compose.yml`
2. **Servidor HTTP**:
   - `go.mod` com `module github.com/maicondmenezes/openjob`.
   - Router `chi` com rota `GET /api/health` → `{"status":"ok"}` e `GET /` servindo placeholder via `//go:embed` (inicialmente um `index.html` mínimo em `backend/web/dist/`).
   - Config via env: `OPENJOB_PORT` (default `8080`), `OPENJOB_DB_PATH` (default `/data/openjob.db`), `OPENJOB_HOSTNAME` (default `openjob.copilot`).
3. **Dockerfile multi-stage**:
   - Estágio Node: build da SPA (placeholder) → `backend/web/dist`.
   - Estágio Go: binário estático (CGO desabilitado) com assets embutidos.
   - Runtime mínimo (`gcr.io/distroless/static` ou `alpine`), porta `8080`, volume `./data:/data`.
4. **`docker-compose.yml`**:
   - Serviço `openjob` + sidecar `skhmt/mdns` com `network_mode: host` e `HOST=openjob.copilot`.
   - Volume `./data` persistente.
5. **Script de setup DNS local** (`scripts/setup-dns.sh`): caminho mDNS (Linux nativo) e alternativa dnsmasq no host; documentar os dois no README (spec §12).
6. **README inicial** (EN/PT): como subir, como acessar `openjob.copilot`, aviso sobre `data/.master.key` (papel documentado desde já, spec §3.2/§4).

## Critério de verificação

- `docker compose up` sobe o container sem erro.
- `curl http://localhost:8080/api/health` → `{"status":"ok"}`.
- **Abrir `http://openjob.copilot:8080` no celular na mesma rede Wi-Fi** e ver a página.
- Fallback documentado: acessar por IP local se mDNS falhar no Mac/Windows.

## Registrar

- ADR: **mDNS mantido** (por que `openjob.copilot` e não IP — uso no sofá pelo celular é caso de uso central).
- Decisão sobre `modernc.org/sqlite` vs `mattn/go-sqlite3` (afeta CGO/estático) — antecipar aqui para não travar o CP3.

## Fora de escopo

- Banco de dados, IA, UI real (tudo em checkpoints seguintes).
