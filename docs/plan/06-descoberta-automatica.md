# Checkpoint 6 — Descoberta automática com curadoria (revisado)

> Origem: spec §8, Checkpoint 6, §3.3 Fase 2. Pré-requisito: [05-vagas-manual-scoring.md](05-vagas-manual-scoring.md) **e** Checkpoint 4 (currículo/base de perfil disponível para o filtro). Esforço estimado: **5–6 dias** (antes 3–4).

## 1. Objetivo

Descoberta automática de vagas em fontes públicas remotas, sob demanda e via ticker opcional, com:

- **Filtro por palavras-chave** (máximo **5 termos**) aplicado na chamada às fontes que suportam busca e/ou no cliente para as que não suportam.
- **Filtro por currículo do usuário**: as vagas retornadas são pré-filtradas por aderência ao perfil/currículo (evitando gasto de LLM em vagas irrelevantes).
- **CRUD de fontes** em `/settings/sources`, com **particularidades generalizadas** por fonte (capacidades declarativas em vez de `if source == "remotive"` espalhado pelo código).
- Deduplicação e scoring em batch *rate-limited*, mantidos do escopo original.

> **Mudança de escopo em relação ao CP6 original:** o checkpoint anterior previa "fontes configuradas via `config.yaml`/env — sem UI". Esta revisão **antecipa a UI de gestão de fontes** para o CP6, e a torna o mecanismo primário de configuração. O `config.yaml`/env passa a ser apenas *seed* inicial (bootstrap).

## 2. Pré-requisitos

- CP5 concluído (pipeline manual funcionando; `jobs` com UNIQUE em `url`).
- CP4 concluído (currículo base / perfil do usuário persistido; ver [04](../plan/04-curriculo-tailoring.md)).
- ADR 0006 (parsing HTML com `golang.org/x/net/html`) mantido — os adapters de scraping usam o mesmo parser.

## 3. Modelo de dados (novo)

### 3.1 Tabela `sources`

Cada fonte é uma linha no banco, com **capacidades declarativas** que o orquestrador lê para decidir *como* chamar o adapter. Adicionar uma fonte = registrar um adapter no código (Go) + inserir/editar uma linha aqui pela UI. O orquestrador não tem conhecimento específico de nenhuma fonte.

| Coluna | Tipo | Descrição |
|---|---|---|
| `id` | TEXT (slug) PK | Identificador estável (ex.: `remotive`, `remoteok`, `himalayas`) |
| `name` | TEXT | Nome exibido na UI |
| `adapter` | TEXT | Slug do adapter registrado no código (validação: precisa existir no registry) |
| `enabled` | INTEGER (bool) | Fonte ativa para discovery |
| `priority` | INTEGER | Ordem de execução (menor = primeiro) |
| `auth_type` | TEXT | `none` \| `api_key` (enum extensível no futuro para `oauth`) |
| `auth_config` | BLOB/TEXT | JSON cifrado com AES-GCM (mesma master key do BYOK) quando `auth_type != none`; vazio caso contrário |
| `capabilities` | TEXT (JSON) | Ver §3.2 |
| `created_at` / `updated_at` | DATETIME | Auditoria |

**Índice único** em `adapter` (uma linha por adapter — evita duplicar a mesma fonte). `id` é o slug estável; o `adapter` referencia o código.

### 3.2 `capabilities` (JSON) — particularidades generalizadas

Sempre que o orquestrador precisa saber "essa fonte aceita busca?", "tem rate limit?", "exige atribuição?", ele consulta este JSON. Nada de `switch source` no código de orquestração.

| Chave | Tipo | Uso pelo orquestrador |
|---|---|---|
| `supports_keyword_search` | bool | Se `true`, os até 5 termos são enviados como parâmetro de busca ao adapter. Se `false`, o orquestrador aplica o filtro de palavras-chave **client-side** após o fetch. |
| `supports_pagination` | bool | Se `true`, o adapter aceita `cursor`/`page` e o orquestrador pagina até `max_results_per_run`. |
| `max_results_per_run` | int | Teto de vagas por execução desta fonte (custo previsível). |
| `rate_limit` | `{requests:int, period:"day"\|"hour"\|"minute"}` | Aplicado pelo orquestrador via `golang.org/x/time/rate`. Ex.: Remotive = `{4, "day"}`; Himalayas = `{60, "minute"}` com chave. |
| `attribution_required` | bool | Se `true`, a UI/notificação deve exibir link para a fonte (Remote OK exige backlink; Jobicy exige atribuição). |
| `experimental` | bool | Fontes sem contrato público (Working Nomads). UI marca com aviso; falhas não contam no health geral. |
| `recent_window_hours` | int | Janela de "recentes" que a fonte entrega (ex.: Remote OK = 100 mais recentes; Jobicy = 7 dias = 168h). Usado só para log/observabilidade. |

> **Por que JSON e não colunas?** As particularidades variam muito entre fontes e evoluem. Colunas fixas forçariam migrações a cada nova fonte. JSONB-like (TEXT + validação em Go) mantém o schema estável e a UI pode renderizar o formulário dinamicamente a partir de um *schema* declarado pelo adapter.

### 3.3 Configuração global de discovery (`settings` ou tabela dedicada)

Persistida em `settings` (chave-valor JSON), editável em `/settings/discovery`:

- `keywords`: array de até **5 strings** (validação dura no backend: `len(keywords) <= 5`; termos vazios removidos; trim; dedup case-insensitive).
- `resume_filter`: `{enabled:bool, min_matches:int, match_mode:"any"|"all", source_of_keywords:"resume"|"manual"}`.
- `discovery_interval`: string (ex.: `"6h"`, `"off"`); default `off`.

### 3.4 Extração de keywords do currículo

Para não acoplar o filtro a uma chamada de LLM em cada discovery, as keywords do currículo são **extraídas uma vez** (na conclusão do CP4 ou sob demanda em `/settings/discovery`) e cacheadas em `settings.resume_keywords` (JSON array). Estratégia:

1. **Heurística local primeiro** (sem LLM): tokenização + stopwords + TF-IDF sobre o texto do currículo → top N termos (N configurável, default 20). Custo zero.
2. **Opcional (se o usuário quiser)**: usar o LLM já configurado (BYOK) para extrair skills/competências em JSON estruturado. Resultado cacheado; só reexecuta quando o currículo muda (hash do conteúdo).

O filtro por currículo compara as keywords do job (título + descrição limpa) com `resume_keywords`, respeitando `min_matches` e `match_mode`. **Roda antes do scoring LLM** para cortar custo.

## 4. Tarefas

### 4.1 Registro de adapters (`internal/scraper/`)

- Interface comum:
  ```go
  type Adapter interface {
      Slug() string                 // "remotive", "remoteok", ...
      Capabilities() Capabilities   // declaradas em código; UI mostra o schema
      Fetch(ctx context.Context, cfg SourceConfig, keywords []string) ([]Job, error)
  }
  ```
- `SourceConfig` carrega `auth_config` já decifrado + `capabilities` da linha do banco.
- **Registry** em `internal/scraper/registry.go`: mapa `slug → Adapter`. O CRUD valida que o `adapter` da linha existe no registry antes de salvar (impede fonte "fantasma").
- Implementar adapters para: **Himalayas** (JSON, search + paginação), **Remotive** (JSON, sem paginação, rate limit 4/dia), **Remote OK** (JSON, janela fixa 100, sem keyword search), **Jobicy** (JSON, paginação por cursor, últimos 7 dias) e **We Work Remotely** (RSS via `encoding/xml`). Working Nomads é opcional/`experimental`.
- Normalização: título, empresa, URL canônica, descrição (limpar HTML com `x/net/html` — ADR 0006), `source`, `external_id` quando a fonte fornecer.
- **Não** automatizar LinkedIn; candidatura sempre redirecionada ao ATS da empresa (spec §3.3).

### 4.2 CRUD de fontes — API + UI

**API** (`/api/sources`):
- `GET /api/sources` — lista com `capabilities` expandidas + status (última execução, última contagem, último erro).
- `POST /api/sources` — cria (valida `adapter` no registry; valida `auth_type` e campos obrigatórios de `auth_config` conforme o adapter).
- `PATCH /api/sources/{id}` — atualiza (inclui toggle `enabled`).
- `DELETE /api/sources/{id}` — remove (soft-delete ou hard? → **hard**, mas com confirmação na UI; vagas já descobertas mantêm `source` histórico).
- `POST /api/sources/{id}/test` — executa um fetch mínimo (1 página/1 item) e retorna amostra + latência + erro, para validar credenciais.

**UI** (`/settings/sources`):
- Tabela com colunas: Nome, Adapter, Enabled (toggle), Auth, Rate limit, Prioridade, Última execução, Ações.
- Botão **"Adicionar fonte"** → formulário gerado a partir de um *schema* declarado pelo adapter (via `GET /api/sources/adapters` que lista adapters + schema de config). Campos:
  - Nome, Adapter (dropdown dos registrados), Enabled, Prioridade.
  - Autenticação: `auth_type` (none / api_key) + campo de credencial (se aplicável), cifrado com a master key.
  - Capacidades: renderizadas do schema do adapter, **read-only por padrão** (o adapter é a fonte da verdade); o usuário só pode ajustar `max_results_per_run` e `enabled`, se o adapter permitir override. Isso evita configurações inconsistentes.
- Ações por linha: editar, testar, desabilitar, remover.
- Fontes `experimental:true` exibem *badge* de aviso.

### 4.3 Configuração de discovery — API + UI

**API** (`/api/settings/discovery`):
- `GET` / `PUT` com validação:
  - `keywords`: máx. 5 (rejeita >5 com `400`), trim, dedup.
  - `resume_filter`: validação de `min_matches` (1–10) e `match_mode`.
  - `discovery_interval`: parse de duração (`off`, `1h`, `6h`, `24h`).
- `POST /api/settings/discovery/resume-keywords/refresh` — reextrai keywords do currículo (heurística local por default; LLM se o usuário marcar).

**UI** (`/settings/discovery`):
- Campo de **até 5 palavras-chave** com contador visual ("3/5") e validação client-side.
- Seção **"Filtro por currículo"**: toggle, `min_matches`, `match_mode`, botão "Reextrair palavras do currículo" com preview das keywords extraídas e origem (heurística/LLM).
- Seção **"Ticker"**: intervalo (off/1h/6h/24h) + aviso de que o default é off.

### 4.4 Orquestração

- `POST /api/jobs/discover` — dispara sob demanda ("Buscar agora"), assíncrono com progresso via SSE ou polling (`GET /api/jobs/discover/{run_id}`).
- Fluxo do orquestrador (genérico, dirigido pelas `capabilities`):
  1. Carrega `discovery_config` + `sources` habilitadas ordenadas por `priority`.
  2. Para cada fonte: aplica `rate_limit`; se `supports_keyword_search`, passa `keywords`; senão, fetch completo.
  3. Normaliza e acumula.
  4. **Filtro de keywords client-side** (para fontes sem search, e como *safety net* para as demais — garante consistência).
  5. **Filtro por currículo** (se habilitado) — corta antes do scoring.
  6. Dedup por `url` (UNIQUE + `INSERT OR IGNORE`), conta novas.
  7. Enfileira novas para scoring em batch.
  8. Registra em `discovery_runs`: run_id, timestamps, por fonte (fetched, filtered_keyword, filtered_resume, new, scored, errors).
- **`time.Ticker`** opcional gerenciado pelo servidor Go, **desabilitado por padrão**; intervalo lido de `discovery_config`. Sem cron do SO. Se `off`, não inicia ticker.
- **Agregador tolerante**: falha de uma fonte (401, timeout, HTML mudou) não derruba as outras; erro vai para `discovery_runs.errors` e para a UI (badge na fonte).

### 4.5 Filtros (novos)

**Filtro de palavras-chave** (`internal/discovery/keywordfilter.go`):
- Case-insensitive, match por substring de palavra (evitar "go" casar "google" → usar boundary de palavra).
- Aplicado sobre título + descrição limpa.
- `match_mode`: `any` (default, ≥1 termo) ou `all` (todos os termos).

**Filtro por currículo** (`internal/discovery/resumefilter.go`):
- Compara `settings.resume_keywords` com tokens do job (título + descrição).
- `min_matches` (default 3) e `match_mode` (`any`/`all`).
- Se `resume_keywords` estiver vazio, filtro é *no-op* (não bloqueia nada) e a UI avisa.
- Sem chamada LLM neste passo.

### 4.6 Persistência e dedup

- `INSERT OR IGNORE` com UNIQUE em `url` (mantido do original).
- Colunas novas em `jobs` (se ainda não existirem): `source` (já previsto), `external_id` (nullable), `matched_keywords` (JSON array, nullable — auditoria de por que a vaga passou nos filtros), `matched_resume_terms` (JSON array, nullable).
- **Teste crítico de deduplicação** (spec §10.5): mesma vaga duas vezes → uma linha.
- Teste de filtros: keyword filter com `any`/`all`; resume filter com `min_matches`.

### 4.7 Scoring em batch

- Mantido do original: fila com `golang.org/x/time/rate`, batch resumível/limitado (`max N por execução`), custo previsível.
- Ordem: prioriza vagas com maior `matched_resume_terms` (proxy de fit sem LLM) para que o batch corte o topo primeiro.

### 4.8 UI (`/jobs`)

- Botão **"Buscar agora"** com loading e resultado: `"X novas vagas, Y após filtro de currículo, Z com score"`.
- Exibe as keywords ativas e o status do filtro de currículo (com link para `/settings/discovery`).
- Badge de atribuição nas vagas de fontes com `attribution_required:true`.
- **Sem** UI de habilitar/desabilitar fonte diretamente em `/jobs` — isso vive em `/settings/sources` (decisão de UX para centralizar configuração).

## 5. Critérios de verificação

- Clicar "Buscar agora" com 5 keywords configuradas → fontes com `supports_keyword_search` recebem os termos; fontes sem suporte filtram client-side; resultado exibe contagem por estágio (fetched → keyword → resume → new → scored).
- Configurar 6 keywords → API rejeita (`400`) e UI bloqueia o 6º termo.
- Habilitar filtro de currículo com `min_matches=3` → vagas com <3 matches não são persistidas nem enfileiradas para scoring; `discovery_runs` registra a contagem cortada.
- `/settings/sources`: criar fonte com adapter inexistente → `400`; testar credencial inválida → `POST .../test` retorna erro claro; toggle `enabled` reflete na próxima execução de discovery.
- Rodar discovery duas vezes seguidas → segunda não duplica vagas (teste automatizado).
- Desabilitar ticker por padrão: reiniciar container sem config → nenhum scraping automático roda.
- Ticker habilitado (`6h`) → segunda execução automática funciona (observar log e `discovery_runs`).
- Fonte `experimental:true` falhando → não derruba as demais; erro aparece na UI da fonte.

## 6. Registrar

- **ADR novo: `0007-sources-crud-capabilities.md`** — decisão de modelar fontes como linhas no banco com `capabilities` JSON + registry de adapters em código, em vez de config estática. Justificativa: extensibilidade sem migração de schema, UI orientada a schema, orquestrador agnóstico de fonte.
- **ADR novo: `0008-resume-keyword-filter.md`** — extração local (heurística) por default e LLM opcional cacheado; filtro pré-LLM para reduzir custo.
- **ADR de scraper local** (já previsto no original): sob demanda/ticker, sem cron do SO.
- **Testes**: deduplicação (§10.5), keyword filter (`any`/`all`/boundary), resume filter (`min_matches`/`match_mode`), validação de `/api/sources` (adapter inexistente, auth_type inválido), rejeição de >5 keywords.
- **Riscos anotados**:
  - APIs/HTML das fontes mudam — agregador tolerante (mantido).
  - Working Nomads sem contrato público — marcar `experimental`, não contar no health.
  - `resume_keywords` desatualizadas se o usuário editar o currículo sem reextrair — botão "Reextrair" na UI + reextração automática quando o hash do currículo mudar.
  - Credenciais por fonte cifradas com a master key: backup do `data/.master.key` torna-se ainda mais crítico (já documentado no README).

## 7. Fora de escopo (deliberadamente adiado)

- OAuth para fontes (nenhuma das fontes do CP6 exige; `auth_type` já é enum extensível).
- UI de habilitar/desabilitar fontes dentro de `/jobs` (fica só em `/settings/sources`).
- Productização multi-tenant das fontes (spec §3.5/§6) — o CRUD aqui é *single-tenant local*.
- Automação de candidatura em qualquer ATS (mantido proibido, spec §3.3).
