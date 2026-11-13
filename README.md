# openJob — Copiloto de Candidaturas

> Local-first job application copilot (Go + SvelteKit, Docker, SQLite).
> Spec: [`docs/context/spec.md`](docs/context/spec.md) · Plan: [`docs/plan/`](docs/plan/)

**Status: Checkpoint 1 (Foundation) — done.**

---

## English

### Overview

openJob automates your application flow for international remote jobs (paid in USD). It runs **locally in Docker** on your laptop, accessible from your phone browser via a friendly LAN URL (`openjob.local`, not an IP). It is a **copilot**: it finds jobs, tailors your résumé with AI (BYOK), scores fit and notifies you — **you** keep final approval. LinkedIn automation (Easy Apply) is **not** supported; LinkedIn is read-only/public discovery only.

### Requirements

- Docker + Docker Compose
- (For local dev) Go 1.22+, Node 22+

### Quick start

```bash
docker compose up -d
```

- App: <http://localhost:8080>
- Health check: `GET /api/health` → `{"status":"ok", ...}`

### Local DNS (openjob.local)

Two supported paths (see `scripts/setup-dns.sh`):

1. **mDNS sidecar (default, native on Linux):** the `mdns` service in `docker-compose.yml` announces `openjob.local` on your LAN. Set your LAN IP in `.env` (copy `.env.example`), then open <http://openjob.local:8080> from your phone on the same Wi-Fi. On macOS/Windows the host's Bonjour usually resolves `.local` automatically.
2. **dnsmasq on the host (fallback):** `./scripts/setup-dns.sh dnsmasq` prints the exact host-record config to add.

> Note: if neither path works on your network, `http://<your-lan-ip>:8080` always works as a fallback.

### Configuration (environment variables)

| Variable | Default | Purpose |
|---|---|---|
| `OPENJOB_PORT` | `8080` | HTTP listen port |
| `OPENJOB_DB_PATH` | `/data/openjob.db` | SQLite database path |
| `OPENJOB_HOSTNAME` | `openjob.local` | LAN hostname announced via mDNS |

### Local data & secrets — ⚠️ read this

- Everything persists in `./data/` (mounted to `/data` in the container).
- **`data/.master.key`** (created from Checkpoint 3 on) is the AES-GCM key that encrypts your API keys at rest. **It is the vault's secret: back it up together with the DB, never commit it** (`.gitignore` already excludes `data/`).

### Development (without Docker)

```bash
make build   # static binary -> bin/openjob
make run     # serves on :8080
make test    # go test ./...
```

### Project layout

```
backend/          # Go module github.com/maicondmenezes/openjob
├── cmd/openjob/  # entrypoint
├── internal/     # api, config, tenant (reserved for future multi-tenancy)
└── web/          # SPA bundle embedded with //go:embed
frontend/         # SvelteKit SPA (Checkpoint 2)
scripts/          # local DNS setup helpers
data/             # SQLite DB + master key (gitignored)
```

---

## Português

### Visão geral

O openJob automatiza o fluxo de candidaturas a vagas internacionais remotas (pagas em USD). Ele roda **localmente em Docker** no seu notebook, acessível pelo navegador do celular via URL amigável na rede local (`openjob.local`, não IP). É um **copiloto**: busca vagas, adapta o currículo com IA (BYOK), calcula score de fit e notifica — a **aprovação final é sempre sua**. Automação no LinkedIn (Easy Apply) é **proibida**; integração somente leitura/pública.

### Requisitos

- Docker + Docker Compose
- (Para desenvolvimento local) Go 1.22+, Node 22+

### Início rápido

```bash
docker compose up -d
```

- App: <http://localhost:8080>
- Health check: `GET /api/health` → `{"status":"ok", ...}`

### DNS local (openjob.local)

Dois caminhos suportados (veja `scripts/setup-dns.sh`):

1. **Sidecar mDNS (padrão; nativo no Linux):** o serviço `mdns` do `docker-compose.yml` anuncia `openjob.local` na rede. Configure o IP da LAN no `.env` (copie `.env.example`) e abra <http://openjob.local:8080> pelo celular no mesmo Wi-Fi. No Mac/Windows o Bonjour do host geralmente já resolve `.local`.
2. **dnsmasq no host (fallback):** `./scripts/setup-dns.sh dnsmasq` imprime a configuração exata do `host-record`.

> Se nenhum caminho funcionar na sua rede, `http://<ip-local>:8080` sempre funciona.

### Configuração (variáveis de ambiente)

| Variável | Padrão | Uso |
|---|---|---|
| `OPENJOB_PORT` | `8080` | Porta HTTP |
| `OPENJOB_DB_PATH` | `/data/openjob.db` | Caminho do banco SQLite |
| `OPENJOB_HOSTNAME` | `openjob.local` | Hostname anunciado via mDNS |

### Dados locais e segredos — ⚠️ leia

- Tudo persiste em `./data/` (montado em `/data` no container).
- **`data/.master.key`** (a partir do Checkpoint 3) é a chave AES-GCM que cifra suas chaves de API em repouso. **É o segredo do seu cofre local: faça backup dele junto com o banco, nunca o commite** (o `.gitignore` já exclui `data/`).

### Desenvolvimento (sem Docker)

```bash
make build   # binário estático -> bin/openjob
make run     # sobe na porta 8080
make test    # go test ./...
```

### Roadmap

| Checkpoint | Descrição | Status |
|---|---|---|
| 1 | Fundação (Go + chi + Docker + mDNS) | ✅ este release |
| 2 | SPA SvelteKit + PWA | ⏳ [plano](docs/plan/02-spa-pwa.md) |
| 3 | SQLite + BYOK | ⏳ [plano](docs/plan/03-sqlite-byok.md) |
| 4–9 | Currículo/tailoring, vagas, discovery, Telegram, validação, ATS | ⏳ [plano](docs/plan/README.md) |
