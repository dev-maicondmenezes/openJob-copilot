# ADR 0020 — Notificações gravam `i18n_key` + `i18n_params`

**Status:** aceito (Checkpoint 13)
**Data:** 2026-11-17
**Relaciona:** docs/plan/13-dark-mode-i18n.md (CP10, §4.3)

## Contexto

O event bus do CP10 emitia `Title`/`Body` já resolvidos em português e o
notifier persistia esses textos na tabela `notifications`. Com a troca de
locale, o histórico ficaria preso no idioma em que foi criado.

## Decisão

Migration aditiva `0004_notifications_i18n.sql` adiciona `i18n_key TEXT` e
`i18n_params TEXT` (JSON) à tabela `notifications`.

- Notificações **novas** gravam `type`, `severity`, `i18n_key` (ex.:
  `notifications.application.status_changed`) e `i18n_params` (JSON com
  `job_label`, `from`, `to`). `title`/`body` ficam vazios.
- Notificações **antigas** mantêm `title`/`body` resolvidos e são renderizadas
  como fallback — nenhuma migração destrutiva.
- O payload da API expõe `i18n_key`/`i18n_params` (nulos quando ausentes).
- O frontend resolve `i18n_key ? t(i18n_key, params) : title + body` em
  `lib/i18n/notifications.ts`, usado pelo feed e pelo sino.

`notify_log` (CP7) continua auditing o texto resolvido no momento do envio; não é
afetado.

## Alternativas rejeitadas

- **Reescrever `title`/`body` a cada troca de locale**: migração destrutiva e
  perda do texto original.
- **Traduzir no backend com Accept-Language**: a preferência é por dispositivo
  no cliente; o backend não conhece o locale corrente da aba.
- **Gravar apenas `type` e reconstruir tudo no frontend**: perde parâmetros
  (ex.: nome da vaga) que só o produtor do evento tem.

## Consequências

- Trocar de idioma re-renderiza todo o histórico corretamente.
- Testes do CP10 que verificavam `title`/`body` de eventos novos agora
  devem verificar `i18n_key`/`i18n_params` (os campos antigos seguem
  preenchidos internamente apenas para compatibilidade de teste, não persistidos).
