# Checkpoint 10 — Notificações in-app + tarefas de IA assíncronas (3–4 dias)

> Origem: spec §3.4 (notificações) + §3.2/§3.3 (tarefas de IA: tailoring, scoring, extração de keywords). Pré-requisito: [07-telegram-tracking.md](07-telegram-tracking.md) (notificações externas e pipeline de candidaturas funcionando) e [06-descoberta-automatica.md](06-descoberta-automatica.md) (scoring em batch e filtro de currículo — que passam a ser consumidores da fila criada aqui). Independente do CP8/CP9.

## 1. Objetivo

Fechar o ciclo de feedback *dentro* do app, sem depender do Telegram para saber o que aconteceu, e dar **controle explícito** sobre as tarefas de IA que hoje rodam “escondidas” (scoring em batch do CP5/CP6, extração de keywords do CP6, tailoring do CP4).

Duas entregas complementares:

1. **Central de notificações in-app** — feed persistido, sino no header com badge de não lidas, e notificações básicas dos fluxos de candidatura (mudança de status, novas vagas descobertas, resultado de execução de discovery, conclusão/falha de tarefa de IA, erros de fonte). Complementa — **não substitui** — o Telegram do CP7.
2. **Fila de tarefas de IA assíncronas** — infraestrutura única (`tasks`) com *workers* em background, estados explícitos (queued/running/done/failed/cancelled), progresso, *retry* e *cancel*, com UI de gestão. Migra o scoring em batch e a extração de keywords para essa fila, e prepara o terreno para o tailoring virar tarefa de background em vez de request síncrono.

> **Princípio de design:** notificações e tarefas são desacopladas por um **event bus interno** (in-process). Quem produz eventos (discovery, pipeline de candidatura, worker de IA) publica; quem consome (in-app, Telegram do CP7) reage. Isso evita que o CP7 e o CP10 dupliquem lógica de formatação/dispatch e permite adicionar canais futuros (e-mail, push) sem tocar nos produtores.

## 2. Pré-requisitos

- CP7 concluído — `applications` com `status`, `notify_log` com `type`, mecanismo AES-GCM do CP3 reutilizado.
- CP6 concluído — scoring em batch e extração de keywords já existem; serão **migrados** para a fila (com testes de regressão garantindo comportamento idêntico).
- CP4 concluído — currículo base disponível; tailoring pode virar consumidor da fila (opcional nesta checkpoint, ver §4.4).
- Padrão de progresso assíncrono já estabelecido no CP6 (`POST /api/jobs/discover` + SSE ou polling em `GET /api/jobs/discover/{run_id}`) — reutilizar.

## 3. Modelo de dados (novo)

### 3.1 Tabela `notifications`

| Coluna | Tipo | Descrição |
|---|---|---|
| `id` | INTEGER PK | Auto-incremento |
| `type` | TEXT | Enum: `application_status`, `discovery_result`, `task_completed`, `task_failed`, `source_error`, `test` |
| `severity` | TEXT | `info` \| `success` \| `warning` \| `error` (dirige cor/ícone na UI) |
| `title` | TEXT | Título curto (ex.: “Candidatura em entrevista”) |
| `body` | TEXT | Texto curto (truncado em ~500 chars; detalhes via `link`) |
| `link` | TEXT (nullable) | Deep link interno da SPA (ex.: `/applications/42`, `/jobs?source=remoteok`) |
| `metadata` | TEXT (JSON, nullable) | Payload livre para auditoria (ex.: `{"application_id":42,"from":"sent","to":"interview"}`) |
| `read_at` | DATETIME (nullable) | `NULL` = não lida (badge do sino conta isso) |
| `created_at` | DATETIME | Ordenação do feed |

**Índices:** `(read_at, created_at DESC)` para o feed; `(type, created_at DESC)` para filtros na UI.

**Retenção:** política de *housekeeping* (ex.: apagar notificações `read_at` com mais de 90 dias) configurável em `settings`; default 90 dias. Evita crescimento indefinido do SQLite local.

### 3.2 Tabela `tasks` (fila de IA)

| Coluna | Tipo | Descrição |
|---|---|---|
| `id` | TEXT PK (ULID) | Ordenável por tempo |
| `type` | TEXT | Enum extensível: `score_job`, `extract_resume_keywords`, `tailor_resume` (este último opcional nesta CP; ver §4.4) |
| `status` | TEXT | `queued` \| `running` \| `done` \| `failed` \| `cancelled` |
| `payload` | TEXT (JSON) | Entrada da tarefa (ex.: `{"job_id":123}`) |
| `result` | TEXT (JSON, nullable) | Saída persistida (ex.: score calculado, keywords extraídas) |
| `error` | TEXT (nullable) | Mensagem legível do último erro |
| `progress` | INTEGER | 0–100 (para tarefas multi-etapa; `score_job` fica 0→100) |
| `attempts` | INTEGER | Contador de tentativas (default 0) |
| `max_attempts` | INTEGER | Default 3 (retry com backoff exponencial) |
| `priority` | INTEGER | Menor = mais prioritário (ex.: tailoring do usuário < scoring em batch) |
| `created_at` / `started_at` / `finished_at` | DATETIME | Auditoria e SLA |

**Índices:** `(status, priority, created_at)` para o *claim* atômico do worker; `(type, status)` para métricas na UI.

**Claim atômico:** `UPDATE tasks SET status='running', started_at=?, attempts=attempts+1 WHERE id = (SELECT id FROM tasks WHERE status='queued' ORDER BY priority, created_at LIMIT 1) RETURNING *` — SQLite serializa, então é seguro com *worker pool* concorrente; um único `UPDATE … RETURNING` evita corrida.

### 3.3 Configuração (`settings`)

- `notifications.{in_app_enabled:bool, retention_days:int, browser_notifications:bool}` (default `true`, `90`, `false`).
- `tasks.{max_concurrency:int, worker_poll_ms:int}` (default `2`, `500`).
- **Sem UI de tipos de notificação por enquanto** — os tipos são emitidos pelos produtores; a UI filtra por `type`/`severity` no feed. Granularidade fica para productização.

## 4. Tarefas

### 4.1 Event bus interno (`internal/events/`)

- Interface mínima:
  ```go
  type Event struct {
      Kind     string          // "application.status_changed", "discovery.completed", "task.completed", ...
      Severity string          // info|success|warning|error
      Title    string
      Body     string
      Link     string          // opcional
      Metadata map[string]any  // opcional
  }
  type Bus interface { Publish(ctx context.Context, e Event) }
  ```
- Implementação in-process (`chan Event` + fan-out para subscribers registrados). Sem fila externa — o app é local-first.
- **Subscribers:**
  - `InAppNotifier` (CP10) → grava em `notifications`.
  - `TelegramNotifier` (CP7, **refatorado**) → mapeia `Kind` para o formato atual, mantendo `notify_log`. A mudança é só de origem: antes chamado diretamente pelos fluxos; agora reage ao bus. Testes do CP7 continuam passando.
- **Produtores migrados:** `PATCH /api/applications/:id` (mudança para `interview`/`rejected`), `POST /api/jobs/discover` (conclusão/falha por fonte), worker de tarefas (`task.completed`/`task.failed`). Emissores são *best-effort*: falha do bus não quebra o fluxo principal (mesmo princípio do CP7).

### 4.2 Worker de tarefas (`internal/tasks/`)

- **Worker pool:** `max_concurrency` goroutines (default 2) + ticker de `worker_poll_ms`. Configurável em `settings.tasks`.
- **Registro de handlers** por `type` (padrão idêntico ao registry de adapters do CP6):
  ```go
  type Handler interface {
      Type() string
      Run(ctx context.Context, payload json.RawMessage, progress func(int)) (json.RawMessage, error)
  }
  ```
- **Rate limiting preservado:** o handler `score_job` reutiliza o `golang.org/x/time/rate` do CP6 (mesma instância por provedor LLM), para não estourar o BYOK. Rate limit é responsabilidade do handler, **não** do worker — permite que tarefas sem custo de LLM (ex.: extração heurística) rodem em paralelo.
- **Retry:** falha → `attempts < max_attempts` reenfileira com backoff exponencial (1s, 4s, 16s…); esgotou → `failed` + evento `task.failed`.
- **Cancel:** `POST /api/tasks/{id}/cancel` — `queued` → `cancelled` direto; `running` → `ctx.Cancel()` propagado ao handler (handlers **devem** checar `ctx.Err()` entre etapas).
- **Restart seguro:** no boot, `UPDATE tasks SET status='queued' WHERE status='running'` (tarefas órfãs voltam para a fila); `attempts` já foi incrementado no claim, então o backoff natural protege contra *loop* infinito em handler quebrado.
- **Handlers implementados nesta CP:**
  - `score_job` — **migração** do batch do CP6; mesmo `rate.Limiter`, mesmas validações. Discovery passa a **enfileirar** em vez de rodar inline; teste de regressão garante mesmo score para o mesmo payload.
  - `extract_resume_keywords` — **migração** do endpoint `/api/settings/discovery/resume-keywords/refresh` do CP6; endpoint passa a enfileirar e a UI acompanha via task (com fallback síncrono opcional para o usuário não esperar quando a fila está vazia — flag `?sync=true`, útil em testes).
- **Handlers fora de escopo (ficam para o CP11 se o tailoring não couber no CP4):** `tailor_resume`. A **estrutura** já suporta; só não implementamos o handler agora (ver §6).

### 4.3 Endpoints

**Notificações:**
- `GET /api/notifications?unread=1&type=…&limit=50&cursor=…` — feed paginado por cursor (`created_at,id`).
- `POST /api/notifications/{id}/read` — marca uma como lida.
- `POST /api/notifications/read-all` — marca todas como lidas.
- `DELETE /api/notifications/{id}` — remove uma (o usuário pode limpar o feed).
- `GET /api/notifications/unread-count` — leve, para o badge do sino (polling curto na UI; 30s).

**Tarefas:**
- `GET /api/tasks?status=…&type=…&limit=50&cursor=…` — lista para a UI.
- `GET /api/tasks/{id}` — detalhe (payload, result, error, attempts, progress).
- `POST /api/tasks/{id}/cancel` — cancelamento (ver §4.2).
- `POST /api/tasks/{id}/retry` — reenfileira `failed`/`cancelled` (reseta `attempts`, `status='queued'`).
- `GET /api/tasks/stream` — **SSE** com mudanças de estado (`queued`→`running`→`done`/`failed`/`cancelled`) e `progress`. Fallback: polling em `GET /api/tasks?since=…`. Reutiliza o padrão do CP6.

**Configurações:**
- `GET/PUT /api/settings/notifications` — `in_app_enabled`, `retention_days`, `browser_notifications`.
- `GET/PUT /api/settings/tasks` — `max_concurrency`, `worker_poll_ms` (com validação de faixa: 1–8 e 100–5000).

### 4.4 UI

**Sino no header (layout SPA):**
- Badge com contagem de não lidas (`GET /api/notifications/unread-count`, polling 30s; pausa quando aba em background via Page Visibility API).
- Painel dropdown com as últimas 10 notificações, agrupadas por dia, com “Marcar todas como lidas” e “Ver todas” (→ `/notifications`).
- Item clicável: navega para `link` e marca como lida.
- **Notificações do navegador (Web Notifications API):** opt-in em `/settings/notifications`; dispara apenas quando a aba **não** está visível (evita ruído). Sem service worker push nesta CP — é notificação local enquanto o app está aberto.

**Página `/notifications`:**
- Feed completo com filtros por `type` e `severity`, “somente não lidas”, e botão de limpar lidas.
- Empty state explicando o que gera notificação (link para `/settings/notifications`).

**Página `/tasks`:**
- Lista com abas por `status` (Queued / Running / Done / Failed / Cancelled) + contadores.
- Linha: tipo (ícone), resumo (ex.: “Score da vaga: Backend Go @ Acme”), progresso (barra quando `running`), tempo decorrido, ações (Cancelar / Retry / Ver detalhe).
- Atualiza em tempo real via SSE; fallback polling 2s.
- Detalhe (drawer/modal): payload, result (formatado), error, histórico de tentativas.
- Ações em massa opcionais: “Cancelar todas as queued” e “Retry todas as failed” (respeitando `max_concurrency`).

**Integração com telas existentes:**
- `/jobs` — o botão “Buscar agora” do CP6 exibe também “X vagas na fila de scoring” lendo `GET /api/tasks?type=score_job&status=queued,running`; link para `/tasks`.
- `/applications` — mudança de status já gera evento → aparece no sino e, se o usuário estiver na tela, um *toast* confirma a ação (o toast é client-side, não persiste em `notifications`; persistir é do servidor via bus).

### 4.5 Refatoração do CP7 (Telegram) para o bus

- `internal/notify/` (CP7) mantém o cliente HTTP e o `notify_log`, mas a **chamada direta** é substituída por uma **subscription no bus**.
- Mapeamento `Kind → type` de `notify_log` preservado: `application.status_changed` → `status_change`; `discovery.completed`/`task.*` → **não** vão para o Telegram por padrão (Telegram é para o que “acontece com sua candidatura”, não para ruído de infra). Ajuste configurável em `settings.telegram.notify_kinds` (default: só `application.status_changed` + `test` + o relatório diário do CP7).
- Teste de regressão do CP7 continua passando: mesma mensagem chega ao Telegram para a mesma mudança de status.

### 4.6 Observabilidade mínima

- `GET /api/health` já existe; adicionar `tasks: {queued, running, failed_last_24h}` e `notifications: {unread}` no payload (útil para *scripts* e para a própria UI de status).
- Log estruturado (`slog`) no worker: `task_id`, `type`, `attempts`, `duration`, `result` (sem logar payload sensível — o handler `score_job` loga só `job_id`).

## 5. Critérios de verificação

- Mudar uma candidatura para `interview` → sino recebe badge **e** (se o usuário estiver em outra tela) *toast* aparece; Telegram continua recebendo a notificação (CP7 sem regressão).
- Rodar “Buscar agora” no CP6 → concluído o discovery, aparece notificação `discovery_result` com a contagem; as vagas novas aparecem em `/tasks` como `score_job` `queued`/`running`, e o score final é idêntico ao comportamento anterior do CP6 (teste de regressão com payload fixo).
- Reextrair keywords do currículo → vira tarefa em `/tasks` (progresso 0→100); resultado persiste em `settings.resume_keywords`; opção `?sync=true` continua funcionando para testes.
- Cancelar tarefa `running` → handler recebe `ctx.Cancel()` e a tarefa vai para `cancelled` em até `worker_poll_ms + tempo do handler respeitar ctx`; sem vazamento de goroutine (checar `goleak` no teste).
- Forçar falha do handler (ex.: LLM retorna 500) → retry com backoff até `max_attempts`; depois `failed` com `error` legível, notificação `task_failed` no sino.
- Reiniciar o container com tarefas `running` → voltam a `queued` no boot e são reprocessadas (sem tarefa órfã).
- Desativar `in_app_enabled` → feed não grava novas, mas Telegram continua; UI mostra aviso de que está desligado.
- Marcar todas como lidas → badge zera; feed continua listando com `read_at` preenchido.
- SSE de `/api/tasks/stream` reconecta após queda de rede (retry do `EventSource` com *backoff*, validado manualmente).

## 6. Registrar

- **ADR `0009-event-bus-notifications.md`** — bus in-process como ponto único de fan-out (in-app + Telegram); justificativa: desacoplar produtores de canais, permitir adicionar e-mail/push no futuro sem tocar nos fluxos, e remover a chamada direta do Telegram dos handlers de candidatura. Alternativa rejeitada: cada fluxo chamar `InAppNotifier` + `TelegramNotifier` explicitamente (duplicação e ordem frágil).
- **ADR `0010-async-task-queue-sqlite.md`** — fila em SQLite com claim atômico via `UPDATE … RETURNING`, worker pool, retry/backoff, cancel por contexto; migração de `score_job` e `extract_resume_keywords` do CP6. Alternativa rejeitada: fila externa (Redis/Asynq) — contraria o “local-first em Docker” e adiciona um serviço a mais. Alternativa rejeitada: `goroutine` por request — sem durabilidade (perde tudo no restart) e sem visibilidade na UI.
- **Testes:**
  - Deduplicação de notificação por `type+metadata.id` em janelas curtas (evita spam de `source_error` repetido a cada tick do CP6).
  - Claim atômico sob concorrência (N workers, M tarefas, nenhuma processada duas vezes — `go test -race`).
  - Retry/backoff e `max_attempts`.
  - Cancel de `running` propaga `ctx.Err()` (handler de teste que dorme e checa ctx).
  - Regressão do CP7: Telegram continua recebendo `application.status_changed` via bus.
  - Retenção de notificações: housekeeping apaga `read_at < now-90d` e mantém não lidas.
- **Riscos anotados:**
  - SSE mantém conexão aberta — em redes móveis com suspensão de aba, o `EventSource` cai; fallback de polling em `/api/tasks` precisa ser testado no Android (limitação do `.local` já documentada no README).
  - `max_concurrency > 1` com `score_job` dividindo o mesmo `rate.Limiter` — testar que o rate limit **global** do provedor LLM é respeitado mesmo com 2 workers.
  - Fila crescente se o LLM estiver fora do ar: UI mostra backlog; considerar (backlog) um *pause/resume* global da fila por provedor.
  - Notificações do navegador exigem HTTPS ou `localhost` — em LAN via `openjob.copilot` (HTTP) o Chrome pode bloquear; documentar que é “best-effort” e degrada para o sino in-app.
- **Backlog (não nesta CP):** push via service worker (PWA) com Web Push real; canais adicionais (e-mail, Slack); *pause/resume* global da fila; priorização dinâmica de `score_job` por `matched_resume_terms` (hoje já é ordem de enfileiramento do CP6).

## 7. Fora de escopo (deliberadamente adiado)

- **Push via service worker (Web Push / VAPID)** — PWA já existe (CP2), mas push real exige infra de push e HTTPS confiável; fica para depois do CP8/CP9 e da validação mobile.
- **Handler `tailor_resume`** — a fila já o suporta (registry de handlers), mas a integração com o fluxo de tailoring do CP4 é feita lá, não aqui. Se o CP4 já entrega tailoring síncrono aceitável, migrar para a fila é um *follow-up* de baixo risco.
- **Multi-tenant, canais externos além do Telegram** (spec §13.9 — pós-validação).
- **UI de configuração de tipos de notificação por canal** (granularidade fina) — o default por `Kind` já cobre o uso.
- **Webhooks de saída** (notificar sistemas externos) — fora do escopo do copiloto local.
- **Automação de candidatura** — mantida proibida (spec §3.3).