# Checkpoint 7 — Telegram + tracking (2–3 dias)

> Origem: spec §8, Checkpoint 7, §3.4. Pré-requisito: [05-vagas-manual-scoring.md](05-vagas-manual-scoring.md) (independente do CP6 — pode rodar em paralelo com ele).

## Objetivo

Notificações via Telegram Bot API: mensagem de teste, relatório diário às 8h (configurável) e notificações imediatas em mudança de status para `interview`/`rejected`. Tracking completo do pipeline de candidaturas.

## Pré-requisitos

- Checkpoint 5 concluído (applications existem para serem trackeadas).

## Tarefas

1. **Notificador** (`internal/notify/`):
   - Cliente Telegram Bot API via HTTP POST: `https://api.telegram.org/bot<TOKEN>/sendMessage` (sem webhooks — spec §4).
   - `type` de mensagem registrado em `notify_log` (`daily_report`, `status_change`, `test`).
   - Falha de envio → log + registro em `notify_log`, **não quebra** o fluxo do app.
2. **Endpoints**:
   - `POST /api/notify/test` — mensagem de teste.
   - `GET/POST /api/settings/telegram` — token + `chat_id` (token cifrado como BYOK, via mesmo mecanismo do CP3).
   - `PATCH /api/applications/:id` — atualiza `status` (`pending|sent|interview|rejected|offer`); ao virar `interview` ou `rejected` → **notificação imediata**.
   - `GET /api/applications` — lista para o pipeline (com dados do job).
3. **Relatório diário**:
   - Scheduler via `robfig/cron` ou `time.Ticker` às 8h (**configurável**, ex. `OPENJOB_DAILY_REPORT_TIME`).
   - Conteúdo: novas vagas encontradas, candidaturas pendentes de aprovação, enviadas, em andamento (spec §3.4).
   - Só envia se token + chat_id configurados.
4. **UI**:
   - `/settings`: campos token (mascarado) + chat_id + botão "Enviar teste"; instrução de como obter chat_id (ex. `@userinfobot`).
   - `/applications`: pipeline em colunas/estágios, com seletor de status por candidatura (ações rápidas mobile).

## Critério de verificação

- Receber **mensagem de teste** no Telegram.
- Mudar uma candidatura para `interview` → notificação imediata chega.
- **No dia seguinte, receber o relatório diário** às 8h (ou horário configurado no teste).
- Sem config → relatório não envia e não gera erro aparente na UI.

## Registrar

- ADR: Telegram mantido (alertas com o laptop fechado — spec §6).
- Teste de mensageria: formato do relatório, truncamento de textos longos.
- Cifrar token do Telegram com o mesmo mecanismo AES-GCM do BYOK (não duplicar lógica).

## Fora de escopo

- Multi-tenant, licenciamento, UI de plataformas (pós-validação apenas — spec §13.9).
