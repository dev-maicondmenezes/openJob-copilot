# openJob — Copiloto de Candidaturas

> Local-first job application copilot (Go + SvelteKit, Docker, SQLite).
> Spec: [`docs/context/spec.md`](docs/context/spec.md) · Plan: [`docs/plan/`](docs/plan/) · Backlog: [`docs/backlog.md`](docs/backlog.md)

**Status: Checkpoint 3 (SQLite + BYOK) — done.**

---

## English

### Overview

openJob automates your application flow for international remote jobs (paid in USD). It runs **locally in Docker** on your laptop, accessible from your phone browser via a friendly LAN URL (`openjob.copilot`, not an IP). It is a **copilot**: it finds jobs, tailors your résumé with AI (BYOK), scores fit and notifies you — **you** keep final approval. LinkedIn automation (Easy Apply) is **not** supported; LinkedIn is read-only/public discovery only.

### Requirements

- Docker + Docker Compose
- (For local dev) Go 1.22+, Node 22+

### Quick start

Three official ways to run it — all built from the same `vX.Y.Z` tag. Details
(install, update, remove, backup, env vars, firewall):
[`docs/distribution.md`](docs/distribution.md). First-time setup (AI provider,
resume, sources) always happens **in the web app**.

**1. Binary or `.deb`** (no Docker, no Go) — download from the GitHub Releases:

```bash
sudo apt install ./openjob_X.Y.Z_amd64.deb   # Debian/Ubuntu (or run the binary directly)
openjob          # prepare, start, open the browser
openjob serve    # server only (SSH / headless)
```

**2. Docker Hub** (`maicondmenezes/openjob`, `linux/amd64` + `linux/arm64`):

```bash
docker run -d --name openjob -p 8080:8080 -v openjob-data:/data maicondmenezes/openjob:X.Y.Z
```

**3. Docker Compose** (published image + named volume):

```bash
docker compose up -d
```

- App: <http://localhost:8080>
- Health check: `GET /api/health` → `{"status":"ok", ...}`
- Building from source with Compose: `docker compose -f docker-compose.yml -f docker-compose.build.yml up -d --build`

### Local DNS (openjob.copilot)

Paths tested and documented in [`docs/adr/0001-mdns-local-hostname.md`](docs/adr/0001-mdns-local-hostname.md):

1. **Native mDNS on Linux (recommended on Linux hosts/VMs):** publish the alias in the single file `/etc/avahi/hosts` and restart avahi — machine hostname stays untouched:
   ```bash
   echo '192.168.15.X openjob.copilot' | sudo tee -a /etc/avahi/hosts
   sudo systemctl restart avahi-daemon
   ```
   Also add the same line to the VM's own `/etc/hosts` for guaranteed local resolution.
2. **mDNS sidecar (Mac/Windows guests without avahi):** the `mdns` service in `docker-compose.yml` announces `openjob.copilot`. Set your LAN IP in `.env` (copy `.env.example`). Keep **one** mDNS responder per network.
3. **Hosts-file alias — the project's canonical local name (desktop use):** `openjob.copilot`, resolved via hosts-file entries only (no extra service):
   - Linux/VM: `echo '192.168.15.X openjob.copilot' | sudo tee -a /etc/hosts`
   - Windows host (admin PowerShell): `Add-Content C:\Windows\System32\drivers\etc\hosts "192.168.15.X openjob.copilot"` then `ipconfig /flushdns`
   - Set `OPENJOB_HOSTNAME=openjob.copilot` in `.env` so the app announces/logs the right name.
4. **Android/Chrome** cannot resolve `.local` (Chromium limitation); mobile access will be a dedicated native Android app after validation (CP8+). Meanwhile, Android can use the magic domain `http://192.168.15.X.nip.io:8080` or the raw IP.

> If neither path works on your network, `http://<your-lan-ip>:8080` always works as a fallback.

### Configuration (environment variables)

| Variable | Default | Purpose |
|---|---|---|
| `OPENJOB_PORT` | `8080` | HTTP listen port |
| `OPENJOB_DB_PATH` | `/data/openjob.db` | SQLite database path |
| `OPENJOB_HOSTNAME` | `openjob.copilot` | LAN hostname announced via mDNS |
| `OPENJOB_FEATURE_UI_FEED` | `false` | Vertical feed as home (CP16) |
| `OPENJOB_FEATURE_EMAIL_SYNC` | `false` | Email tracking + `in_process` pipeline (CP17) |
| `OPENJOB_FEATURE_CURRICULUM_EXPORT` | `false` | Resume export (B10) |
| `OPENJOB_FEATURE_SCORE_DETERMINISTIC` | `true` | Deterministic scoring, no AI (B11) |
| `OPENJOB_FEATURE_KEYWORDS_DETERMINISTIC` | `true` | Deterministic keyword extraction (B11) |
| `OPENJOB_FEATURE_FILTER_DETERMINISTIC` | `true` | Deterministic resume filter (B11) |

UI/email/export flags accept `true`/`1` (and `false`/`0`); anything else stays
off. Deterministic flags default on and can be set to `false`/`0` for rollback.
They are exposed by `GET /api/features` so the SPA can hide unfinished UI until
the matching phase ships.

### Local data & secrets — ⚠️ read this

- Everything persists in the data directory: a Docker volume (`/data`), `./data/` with `make run`/the build override, or your per-user directory for the binary/`.deb` (`~/.local/share/openjob` on Linux).
- **`data/.master.key`** (created from Checkpoint 3 on) is the AES-GCM key that encrypts your API keys at rest. **It is the vault's secret: back it up together with the DB, never commit it** (`.gitignore` already excludes `data/`).
- If the master key is lost, stored API keys become unrecoverable (explicit error) — just re-enter your key in `/settings`. See [docs/adr/0004-byok-local-master-key.md](docs/adr/0004-byok-local-master-key.md).

### Development (without Docker)

```bash
make build   # SPA build (npm) -> backend/web/dist -> static binary -> bin/openjob
make run     # serves on :8080 (openjob serve)
make test    # go test ./...
make dev     # hot reload: air (Go, :8080) + Vite HMR (SPA, :5173) in parallel
```

The SPA (SvelteKit + PWA) lives in `frontend/`; `npm run build` there compiles
it and publishes the output to `backend/web/dist`, which is embedded into the
Go binary via `//go:embed` (see `docs/adr/0003-adapter-static-spa.md`).
Deep links (e.g. reloading `/jobs`) are served by the SPA fallback.

#### Hot reload

- **SPA** — Vite dev server with HMR (`make dev-web` / `npm run dev`):
  edits reload instantly; `/api` requests are proxied to `:8080` (no CORS).
- **Go** — [air](https://github.com/air-verse/air) watches `.go` files,
  rebuilds and restarts the server (`make dev-api`; installed on first run).
- `make dev` runs both. Use the SPA at `http://localhost:5173`
  (from the phone on the same Wi-Fi: `http://openjob.copilot:5173`).
  The embedded-bundle path (`make build && make run`) is for testing the real
  production assets — the dev server does not use `backend/web/dist`.

### Project layout

```
backend/          # Go module github.com/maicondmenezes/openjob
├── cmd/openjob/  # entrypoint
├── internal/     # api, config, store (SQLite+migrations+crypto), llm, tenant (reserved)
└── web/          # SPA bundle embedded with //go:embed
frontend/         # SvelteKit SPA (Checkpoint 2)
scripts/          # local DNS setup helpers
data/             # SQLite DB + master key (gitignored)
```

---

## Português

### Visão geral

O openJob automatiza o fluxo de candidaturas a vagas internacionais remotas (pagas em USD). Ele roda **localmente em Docker** no seu notebook, acessível pelo navegador do celular via URL amigável na rede local (`openjob.copilot`, não IP). É um **copiloto**: busca vagas, adapta o currículo com IA (BYOK), calcula score de fit e notifica — a **aprovação final é sempre sua**. Automação no LinkedIn (Easy Apply) é **proibida**; integração somente leitura/pública.

### Requisitos

- Docker + Docker Compose
- (Para desenvolvimento local) Go 1.22+, Node 22+

### Início rápido

Três caminhos oficiais — todos gerados da mesma tag `vX.Y.Z`. Detalhes
(instalação, atualização, remoção, backup, variáveis, firewall):
[`docs/distribution.md`](docs/distribution.md). O primeiro uso (provedor de IA,
currículo, fontes) acontece sempre **na webapp**.

**1. Binário ou `.deb`** (sem Docker, sem Go) — baixe nas Releases do GitHub:

```bash
sudo apt install ./openjob_X.Y.Z_amd64.deb   # Debian/Ubuntu (ou execute o binário direto)
openjob          # prepara, sobe e abre o navegador
openjob serve    # só o servidor (SSH / headless)
```

**2. Docker Hub** (`maicondmenezes/openjob`, `linux/amd64` + `linux/arm64`):

```bash
docker run -d --name openjob -p 8080:8080 -v openjob-data:/data maicondmenezes/openjob:X.Y.Z
```

**3. Docker Compose** (imagem publicada + volume nomeado):

```bash
docker compose up -d
```

- App: <http://localhost:8080>
- Health check: `GET /api/health` → `{"status":"ok", ...}`
- Build do código-fonte com Compose: `docker compose -f docker-compose.yml -f docker-compose.build.yml up -d --build`

### DNS local (openjob.copilot)

Caminhos testados e documentados em [`docs/adr/0001-mdns-local-hostname.md`](docs/adr/0001-mdns-local-hostname.md):

1. **mDNS nativo no Linux (recomendado em hosts/VMs Linux):** publique o alias no **arquivo único** `/etc/avahi/hosts` e reinicie o avahi — o hostname da máquina fica intacto:
   ```bash
   echo '192.168.15.X openjob.copilot' | sudo tee -a /etc/avahi/hosts
   sudo systemctl restart avahi-daemon
   ```
   Adicione a mesma linha ao `/etc/hosts` da própria VM para resolução local garantida.
2. **Sidecar mDNS (Mac/Windows sem avahi no guest):** o serviço `mdns` do `docker-compose.yml` anuncia `openjob.copilot`. Configure o IP da LAN no `.env` (copie `.env.example`). Mantenha **um único** respondente mDNS por rede.
3. **Alias por hosts-file — nome canônico local (uso desktop):** `openjob.copilot`, resolvido apenas por entradas de hosts (sem serviço extra):
   - Linux/VM: `echo '192.168.15.X openjob.copilot' | sudo tee -a /etc/hosts`
   - Windows host (PowerShell como admin): `Add-Content C:\Windows\System32\drivers\etc\hosts "192.168.15.X openjob.copilot"` e depois `ipconfig /flushdns`
   - Configure `OPENJOB_HOSTNAME=openjob.copilot` no `.env` para o app anunciar/logar o nome certo.
4. **Android/Chrome** não resolve `.local` (limitação do Chromium); o acesso mobile será um app Android nativo dedicado após a validação (CP8+). Enquanto isso, Android pode usar o domínio mágico `http://192.168.15.X.nip.io:8080` ou o IP direto.

> Se nenhum caminho funcionar na sua rede, `http://<ip-local>:8080` sempre funciona.

### Configuração (variáveis de ambiente)

| Variável | Padrão | Uso |
|---|---|---|
| `OPENJOB_PORT` | `8080` | Porta HTTP |
| `OPENJOB_DB_PATH` | `/data/openjob.db` | Caminho do banco SQLite |
| `OPENJOB_HOSTNAME` | `openjob.copilot` | Hostname anunciado via mDNS |
| `OPENJOB_FEATURE_UI_FEED` | `false` | Feed vertical como home (CP16) |
| `OPENJOB_FEATURE_EMAIL_SYNC` | `false` | Tracking de e-mail + pipeline `in_process` (CP17) |
| `OPENJOB_FEATURE_CURRICULUM_EXPORT` | `false` | Exportação de currículo (B10) |
| `OPENJOB_FEATURE_SCORE_DETERMINISTIC` | `true` | Scoring determinístico, sem IA (B11) |
| `OPENJOB_FEATURE_KEYWORDS_DETERMINISTIC` | `true` | Extração determinística de keywords (B11) |
| `OPENJOB_FEATURE_FILTER_DETERMINISTIC` | `true` | Filtro determinístico de currículo (B11) |

Flags de UI/e-mail/exportação aceitam `true`/`1` (e `false`/`0`); qualquer outro
valor fica desligado. As flags determinísticas vêm ligadas e podem ser definidas
como `false`/`0` para rollback. Elas são expostas por `GET /api/features` para a
SPA esconder UI inacabada até a fase correspondente entrar no ar.

### Dados locais e segredos — ⚠️ leia

- Tudo persiste no diretório de dados: volume Docker (`/data`), `./data/` com `make run`/o override de build, ou o diretório do usuário para binário/`.deb` (`~/.local/share/openjob` no Linux).
- **`data/.master.key`** (a partir do Checkpoint 3) é a chave AES-GCM que cifra suas chaves de API em repouso. **É o segredo do seu cofre local: faça backup dele junto com o banco, nunca o commite** (o `.gitignore` já exclui `data/`).
- Se a master key for perdida, as chaves cifradas ficam irrecuperáveis (erro explícito) — basta reconfigurar a API key em `/settings`. Ver [docs/adr/0004-byok-local-master-key.md](docs/adr/0004-byok-local-master-key.md).

### Desenvolvimento (sem Docker)

```bash
make build   # build da SPA (npm) -> backend/web/dist -> binário estático -> bin/openjob
make run     # sobe na porta 8080 (openjob serve)
make test    # go test ./...
make dev     # hot reload: air (Go, :8080) + Vite HMR (SPA, :5173) em paralelo
```

A SPA (SvelteKit + PWA) fica em `frontend/`; o `npm run build` compila e
publica o output em `backend/web/dist`, embutido no binário Go via `//go:embed`
(ver `docs/adr/0003-adapter-static-spa.md`). Deep links (ex.: recarregar `/jobs`)
são servidos pelo fallback da SPA.

#### Hot reload

- **SPA** — dev server do Vite com HMR (`make dev-web` / `npm run dev`):
  edições recarregam na hora; `/api` é repassado por proxy para `:8080` (sem CORS).
- **Go** — [air](https://github.com/air-verse/air) assiste os `.go`,
  rebuilda e reinicia o servidor (`make dev-api`; instala no primeiro uso).
- `make dev` sobe os dois. Use a SPA em `http://localhost:5173`
  (pelo celular na mesma Wi-Fi: `http://openjob.copilot:5173`).
  O caminho embutido (`make build && make run`) serve para testar os assets
  reais de produção — o dev server não usa `backend/web/dist`.

### Roadmap

| Checkpoint | Descrição | Status |
|---|---|---|
| 1 | Fundação (Go + chi + Docker + mDNS) | ✅ este release |
| 2 | SPA SvelteKit + PWA | ✅ este release — [plano](docs/plan/02-spa-pwa.md) |
| 3 | SQLite + BYOK | ✅ este release — [plano](docs/plan/03-sqlite-byok.md) |
| 4–9 | Currículo/tailoring, vagas, discovery, Telegram, validação, ATS | ✅ este release — [plano](docs/plan/README.md) |
| 10 | Notificações in-app + fila assíncrona de tarefas | ✅ este release — [plano](docs/plan/10-notificacoes-in-app.md) |
| 11 | Distribuição: binário único, `.deb`, imagem Docker Hub e CI/CD por tag | ✅ este release — [plano](docs/plan/11-package-release.md) |
| 12 | Design system + responsividade multi-dispositivo | ✅ este release — [plano](docs/plan/12-design.md) |

Melhorias pós-checkpoints (ex.: configuração de IA mais amigável, presets de provedores, multi-modelos): [`docs/backlog.md`](docs/backlog.md).
