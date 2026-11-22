# Plano de Implementação — openJob

> Derivado de [`docs/context/spec.md`](../context/spec.md). Um arquivo por checkpoint, cada checkpoint é uma **tarefa individual**, executada e validada isoladamente antes de avançar.

## Como usar este plano

1. Execute os checkpoints **em ordem** (as dependências são explícitas em cada arquivo).
2. Cada checkpoint só é considerado concluído quando o **critério de verificação** é atendido em ambiente real (celular na rede local quando envolver UI/rede).
3. Cada arquivo contém: objetivo, pré-requisitos, tarefas, arquivos esperados, critério de verificação e o que registrar (testes/ADRs).
4. Checkpoints marcados como 🆕 são itens novos da spec revisada; ⚠️ é opcional e só entra após a validação pessoal.
5. Para o fechamento do MVP (CP16 + CP17 + backlog restante), siga o [release-mvp.md](release-mvp.md): fases auto-contidas, com gate de teste por fase.

## Sequência e dependências

| # | Arquivo | Checkpoint | Duração | Depende de |
|---|---|---|---|---|
| 1 | [01-fundacao.md](01-fundacao.md) | Fundação (Go + chi + Docker + mDNS) | 2–3 dias | — |
| 2 | [02-spa-pwa.md](02-spa-pwa.md) | SPA SvelteKit + PWA embutida | 2–3 dias | CP1 |
| 3 | [03-sqlite-byok.md](03-sqlite-byok.md) | SQLite + criptografia BYOK | 3–4 dias | CP1, CP2 |
| 4 | [04-curriculo-tailoring.md](04-curriculo-tailoring.md) | Currículo + adaptação via IA | 3–4 dias | CP3 |
| 5 | [05-vagas-manual-scoring.md](05-vagas-manual-scoring.md) 🆕 | Entrada manual de vagas + scoring + aprovação | 1–2 dias | CP3, CP4 |
| 6 | [06-descoberta-automatica.md](06-descoberta-automatica.md) | Scrapers Remotive/WWR/RemoteOK + ticker | 3–4 dias | CP5 |
| 7 | [07-telegram-tracking.md](07-telegram-tracking.md) | Telegram + relatório diário + tracking de status | 2–3 dias | CP5 |
| 8 | [08-validacao-pessoal.md](08-validacao-pessoal.md) 🆕 | Validação pessoal com dados reais | 2–4 semanas | CP1–CP7 |
| 9 | [09-ats-assistido.md](09-ats-assistido.md) ⚠️ | ATS assistido (opcional) | — | CP8 + dor real comprovada |
| 10 | [10-notificacoes-in-app.md](10-notificacoes-in-app.md) | Notificações in-app + tarefas de IA assíncronas | 3–4 dias | CP6, CP7 |
| 11 | [11-package-release.md](11-package-release.md) | Binário único, wizard de instalação e release | 4–5 dias | CP1–CP10 |
| 12 | [12-design.md](12-design.md) | Design system e responsividade multi-dispositivo | 5–6 dias | CP1–CP11 |
| 13 | [13-dark-mode-i18n.md](13-dark-mode-i18n.md) | Dark mode + i18n (PT-BR, EN-US, ES-419) | 4–5 dias | CP1–CP12 |
| 14 | [14-multi-mercado-latam.md](14-multi-mercado-latam.md) | Expansão multi-mercado (APIs + scraping ético) | — | CP6, CP10 |
| 15 | [15-dashboard.md](15-dashboard.md) | Dashboard de estatísticas e ações na home | 4–5 dias | CP10, CP12, CP13, CP14 |
| 16 | [16-infinite-timeline-design.md](16-infinite-timeline-design.md) | Feed vertical imersivo (estilo TikTok/Reels) | 6–8 dias | CP12, CP13 |
| 17 | [17-email-tracking-pipeline.md](17-email-tracking-pipeline.md) 🆕 | Automação de e-mail, pipeline "em processo" e tags de fase | 6–8 dias | CP7, CP10, CP12 |

**Regra de ouro (spec §13.8):** o Checkpoint 5 vem **antes** do 6 — o produto já é utilizável de verdade com entrada manual de vagas, sem esperar a descoberta automática.

> O CP17 nasceu de uma sessão de chat indexada (`docs/context/6c7eea4c-71d0-49e7-9f5b-5c374f9b73c6.json`, msgs. 34–35) e está registrado no backlog como item promovido a checkpoint.

## Artefatos transversais (mantidos desde o início)

Conforme spec §10, preservar em todos os checkpoints:

1. **Interface `Repository`** em `internal/store/` — troca SQLite→Postgres sem tocar em handlers.
2. **Abstração `LLMClient`** em `internal/llm/` — troca de SDK sem acoplamento a provedor.
3. **Migrations versionadas** embutidas (`//go:embed`) — nunca "drop table".
4. **ADRs curtos** em `docs/adr/` — um por decisão (BYOK chave local, sem tenant_id, mDNS, cliente LLM OpenAI-compatible).
5. **Testes nos pontos críticos** — criptografia BYOK, parsing de JD, deduplicação de vagas.
6. **`internal/tenant/` vazio** — terreno pronto para multi-tenancy pós-validação.

## Convenções de nomenclatura (spec §2)

- Módulo Go: `github.com/maicondmenezes/openjob` · binário: `openjob` · container: `openjob`
- Hostname: `openjob.copilot` (config. via `OPENJOB_HOSTNAME`) · banco: `/data/openjob.db`
- Chave local: `/data/.master.key` (`0600`) · env: prefixo `OPENJOB_`
