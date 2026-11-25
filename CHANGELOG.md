# Changelog

Todas as mudanças notáveis do openJob são registradas aqui.
O formato segue o [Keep a Changelog](https://keepachangelog.com/pt-BR/1.1.0/)
e o projeto usa [Versionamento Semântico](https://semver.org/lang/pt-BR/).

## [Unreleased]

### Corrigido

- Dockerfile: removida a instrução `VOLUME`, incompatível com o deploy no
  Railway (#4).

### Alterado

- Documentação do MVP consolidada: plano de release/F7, backlog, distribuição,
  notas da versão e planos pós-MVP (CP18–CP23).

## [0.1.0] - 2026-11-25

Primeira versão publicável do MVP.

### Adicionado

- **Feed vertical imersivo (CP16):** `/jobs` como home, lista preservada em
  `/jobs/list` e detalhe em `/jobs/[id]` com Markdown sanitizado.
- **Pipeline por e-mail (CP17/B12):** sincronização Gmail/IMAP, matching em
  cascata, pipeline com 6 colunas, tags `phase:` e transições monotônicas.
- **Currículo:** cópia rápida do adaptado (B10), erros amigáveis no
  upload/tailoring (B8) e exportação em PDF, TXT e MD (B9).
- **UX de IA:** onboarding BYOK (B1), catálogo de presets (B3), free tier (B4) e
  múltiplos perfis com troca do ativo (B2).
- **Backend determinístico (B11):** score BM25, keywords YAKE/RAKE e filtro por
  currículo sem LLM, com flags de rollback.
- **Distribuição (CP11):** binário único, pacote `.deb`, imagem Docker
  multi-arquitetura e release automatizada por tag `vX.Y.Z`.
- **Erros amigáveis de provedor (B6):** causa provável + ação sugerida, sem
  expor payload, stack ou segredo.
- Dashboard (CP15), dark mode/i18n PT-BR, EN-US e ES-419 (CP13) e expansão
  multi-mercado (CP14).

### Alterado

- Flags de release (`ui.feed`, `email.sync`, `curriculum.export`) habilitadas
  por padrão, com rollback por `false`/`0`.
- ADR da imagem Docker renumerado de `0036` para `0043` para não colidir com o
  ADR de perfis LLM.

### Corrigido

- Migrações validadas em banco novo e no upgrade `0007 → 0008`, preservando
  dados.
- CI de UI: `test:e2e` restrito ao projeto `e2e`; snapshots visuais executados
  na imagem Playwright fixa.

### Segurança

- Markdown de vagas sanitizado contra XSS.
- Chaves BYOK cifradas com AES-GCM e redaction de segredos em logs e erros.

### Decisões de escopo

- LinkedIn OAuth (B7) e OAuth de provedores de IA (B5) ficaram **no-go no MVP**;
  upload manual de currículo e BYOK seguem como caminhos suportados.

[Unreleased]: https://github.com/dev-maicondmenezes/openJob-copilot/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/dev-maicondmenezes/openJob-copilot/releases/tag/v0.1.0
