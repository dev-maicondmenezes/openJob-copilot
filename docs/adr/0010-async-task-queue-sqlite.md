# ADR 0010 — Fila de tarefas assíncronas em SQLite

**Status:** aceito (Checkpoint 10)
**Data:** 2026-11-16
**Relaciona:** docs/plan/10-notificacoes-in-app.md

## Contexto

O CP6 executava scoring em batch e extração de keywords inline nos fluxos
(discovery e endpoint de settings). Isso significa: sem progresso visível, sem
retry controlado, sem cancelamento e perda total do trabalho em restart do
processo. O tailoring (CP4) também roda dentro do request HTTP.

## Decisão

Fila durável em SQLite (`tasks`) consumida por um **worker pool** em goroutines
(`internal/tasks`), com:

- **Claim atômico** — `UPDATE tasks SET status='running' … WHERE id = (SELECT …
  LIMIT 1) RETURNING *`: com o pool serializado em uma única conexão, nenhum
  worker pega a mesma tarefa duas vezes.
- **Retry com backoff exponencial** (1s, 4s, 16s…, teto 1min) até
  `max_attempts` (3); a coluna `next_run_at` (adição ao schema do plano)
  materializa o atraso — o claim só pega tarefas elegíveis.
- **Cancelamento por contexto** — `POST /api/tasks/{id}/cancel` cancela direto se
  `queued`, ou propaga `ctx.Cancel()` ao handler em execução; handlers checam
  `ctx.Err()` entre etapas.
- **Recuperação no boot** — `running` órfãs de um crash voltam a `queued`
  (`RequeueRunningTasks`); como o claim incrementa `attempts`, o backoff natural
  protege contra loop infinito de handler quebrado.
- **Handlers registrados por tipo** (registry, padrão dos adapters do CP6) —
  `score_job` e `extract_resume_keywords` migrados do CP6; `tailor_resume`
  suportado pela estrutura sem handler nesta checkpoint.
- **Rate limit no handler, não no pool** — o handler `score_job` compartilha um
  `golang.org/x/time/rate.Limiter` por provedor de LLM, respeitando o limite
  global do BYOK mesmo com `max_concurrency > 1`; handlers sem custo de LLM
  rodam em paralelo.
- **Observabilidade** — eventos `task.*` no bus (ADR 0009) alimentam a UI
  (`/tasks` via SSE/polling), `/api/health` e notificações.

## Alternativas rejeitadas

- **Fila externa (Redis + Asynq)** — contraria o local-first em Docker: um
  serviço a mais para um app single-user, e a durabilidade do SQLite já existe.
- **Goroutine por request** — sem durabilidade (perde tudo no restart), sem
  visibilidade de fila/progresso, sem retry nem cancelamento.

## Consequências

- Discovery enfileira scoring (`ScoreEnqueuer`) em vez de rodar inline; teste de
  regressão garante score idêntico para o mesmo payload.
- Extração de keywords vira tarefa com fallback síncrono (`?sync=true`) para
  testes.
- UI lista/filtra/cancela/retry via `/api/tasks`, com SSE em `/api/tasks/stream`.
- Config em `settings` (`tasks.config`): `max_concurrency` (1–8, ajustável a
  quente) e `worker_poll_ms` (100–5000).
