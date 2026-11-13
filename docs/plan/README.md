# Plano de Implementação — openJob

> Derivado de [`docs/context/spec.md`](../context/spec.md). Um arquivo por checkpoint, cada checkpoint é uma **tarefa individual**, executada e validada isoladamente antes de avançar.

## Como usar este plano

1. Execute os checkpoints **em ordem** (as dependências são explícitas em cada arquivo).
2. Cada checkpoint só é considerado concluído quando o **critério de verificação** é atendido em ambiente real (celular na rede local quando envolver UI/rede).
3. Cada arquivo contém: objetivo, pré-requisitos, tarefas, arquivos esperados, critério de verificação e o que registrar (testes/ADRs).
4. Checkpoints marcados como 🆕 são itens novos da spec revisada; ⚠️ é opcional e só entra após a validação pessoal.

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

**Regra de ouro (spec §13.8):** o Checkpoint 5 vem **antes** do 6 — o produto já é utilizável de verdade com entrada manual de vagas, sem esperar a descoberta automática.

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
- Hostname: `openjob.local` (config. via `OPENJOB_HOSTNAME`) · banco: `/data/openjob.db`
- Chave local: `/data/.master.key` (`0600`) · env: prefixo `OPENJOB_`
