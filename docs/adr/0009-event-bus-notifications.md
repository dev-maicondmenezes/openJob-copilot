# ADR 0009 — Event bus in-process para notificações

**Status:** aceito (Checkpoint 10)
**Data:** 2026-11-16
**Relaciona:** docs/plan/10-notificacoes-in-app.md

## Contexto

Antes do CP10, cada fluxo que precisava notificar o usuário (discovery, pipeline de
candidaturas) chamaria diretamente cada canal de saída: o notificador in-app (novo
neste checkpoint) e, quando o CP7 existir, o Telegram. Copiar essa chamada direta em
cada produtor duplica a lógica de formatação/dispatch e torna frágil a ordem e a
completude das notificações — e adicionar um canal futuro (e-mail, push) exigiria
tocar em todos os fluxos produtores.

## Decisão

Um **event bus in-process** (`internal/events`) é o ponto único de fan-out: os
produtores publicam eventos de domínio (`application.status_changed`,
`discovery.completed`, `discovery.failed`, `source.fetch_failed`, `task.completed`,
`task.failed`) e os canais assinam o bus:

- `internal/notifier` — grava em `notifications` (CP10);
- Telegram (CP7, futuro) — assina e mantém `notify_log`; enquanto o CP7 não
  existir, o bus tem somente o subscriber in-app.

Propriedades deliberadas:

- **Best-effort:** `Publish` nunca bloqueia nem propaga erro; buffer cheio de um
  subscriber descarta o evento (feed persistido + SSE com fallback de polling
  toleram a perda de um evento transitório).
- **Dedup por `DedupKey`:** eventos repetidos em janela curta (ex.: `source_error`
  da mesma fonte a cada tick do CP6) viram uma única notificação.
- **Desligável por canal:** `settings.notifications.in_app_enabled` silencia o
  canal in-app sem afetar produtores nem outros canais.

## Alternativas rejeitadas

- **Cada fluxo chamar `InAppNotifier` + `TelegramNotifier` explicitamente** —
  duplicação de formatação, ordem frágil, impossível adicionar canal sem tocar nos
  fluxos.
- **Fila externa (Redis/NATS) para eventos** — contraria o local-first em um
  binário único com SQLite (ver ADR 0010 para a mesma lógica na fila de tarefas).

## Consequências

- Produtores só dependem de `events.Bus` (leaf package, sem dependências).
- SSE de `/api/tasks/stream` e o sino consomem o mesmo fluxo de eventos.
- Falha do subscriber in-app não quebra o fluxo principal (mesmo princípio do CP7).
