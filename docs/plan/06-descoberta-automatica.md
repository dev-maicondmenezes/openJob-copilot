# Checkpoint 6 — Descoberta automática (3–4 dias)

> Origem: spec §8, Checkpoint 6, §3.3 Fase 2. Pré-requisito: [05-vagas-manual-scoring.md](05-vagas-manual-scoring.md).

## Objetivo

Descoberta automática de vagas em fontes públicas remotas (Remotive, We Work Remotely, Remote OK), sob demanda e via ticker opcional, com deduplicação e scoring em batch rate-limited.

## Pré-requisitos

- Checkpoint 5 concluído (pipeline manual funcionando; `jobs` com UNIQUE em `url`).

## Tarefas

1. **Scrapers** (`internal/scraper/`):
   - Um adapter por fonte com interface comum (`Source` → lista de vagas normalizadas), implementando Remotive, WWR e RemoteOK via `gocolly/colly` **ou** `net/http + goquery` (decidir e documentar; prefira APIs JSON públicas quando existirem — Remotive e RemoteOK têm).
   - Normalização: título, empresa, URL canônica, descrição (limpar HTML), `source`.
   - Não automatizar LinkedIn; candidatura redirecionada ao ATS da empresa (Greenhouse/Lever/Workday) — apenas descobrir/linkar, nada de automação (spec §3.3).
2. **Orquestração**:
   - `POST /api/jobs/discover` — dispara scraping sob demanda (botão "Buscar agora"), assíncrono com feedback de progresso/contagem.
   - **`time.Ticker`** opcional a cada 6h, gerenciado pelo servidor Go, **desabilitado por padrão** (env/config, ex. `OPENJOB_DISCOVERY_INTERVAL`); sem cron do SO.
3. **Persistência e dedup**:
   - `INSERT` com UNIQUE em `url` → `INSERT OR IGNORE` (ou upsert) para não duplicar; registrar contagem de novas.
   - **Teste crítico de deduplicação** (spec §10.5): mesma vaga duas vezes → uma linha.
4. **Scoring em batch**:
   - Após discovery, enfileirar vagas novas para scoring com **rate limiting** (ex. `golang.org/x/time/rate`, ~1 req/s ou config) para não estourar a API do provedor.
   - Batch resumível/limitado (ex. máx. N por execução) para custo previsível.
5. **UI** (`/jobs`):
   - Botão **"Buscar agora"** com estado de loading e resultado ("X novas vagas, Y com score").
   - Fontes de scraping configuradas via `config.yaml`/env — **sem UI de habilitar/desabilitar plataformas** (fica para productização, spec §3.5/§6).

## Critério de verificação

- Clicar "Buscar agora" → ver **30–80 vagas novas**, com `score_fit` calculado (progressivo), ordenadas por score.
- Rodar discovery duas vezes seguidas → segunda não duplica vagas.
- Desabilitar ticker por padrão: reiniciar container sem config → nenhum scraping automático roda.
- Ticker habilitado → segunda execução automática funciona (observar log).

## Registrar

- ADR: **scraper local** sob demanda/ticker, sem cron do SO (spec §6).
- Teste de deduplicação em suíte.
- Risco anotado: APIs/HTML das fontes mudam — tratar falha de uma fonte sem derrubar as outras (agregador tolerante).
