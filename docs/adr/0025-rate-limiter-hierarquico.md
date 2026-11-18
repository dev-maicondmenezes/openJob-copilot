# ADR 0025 — Rate limiter hierárquico com estado persistente

**Status:** aceito (Checkpoint 14)
**Data:** 2026-11-17
**Relaciona:** docs/plan/14-multi-mercado-latam.md, ADR 0023, ADR 0010,
`internal/discovery/ratelimit/`, `internal/store/migrations/0005_sources_multi_market.sql`

## Contexto

O botão “Buscar agora” pode ser clicado repetidamente. Sem serialização, cada
clique dispara uma execução concorrente contra as mesmas fontes e o conjunto de
requisições ultrapassa qualquer limite observável. Além disso, um limite apenas
em memória zera ao reiniciar o app — exatamente quando um bloqueio de 24h
deveria continuar valendo.

## Decisão

Adotar um limiter **hierárquico de três camadas**, por fonte, com o estado de
longo prazo em SQLite:

### Camada 1 — Ritmo (token bucket em memória)

- Um `rate.Limiter` por fonte (`golang.org/x/time/rate`), derivado de
  `capabilities.rate_limit` (`requests`, `period`, `strategy`).
- **Burst fixo em 1**: mesmo que a taxa permita rajadas, duas requisições nunca
  saem coladas.
- O adapter recebe um `SourceWaiter` em `SourceConfig` e chama `Wait(ctx)`
  antes de cada request. O bucket é compartilhado entre execuções, então cliques
  repetidos são serializados globalmente, não por run.
- A execução de discovery em si também é serializada: `POST /api/jobs/discover`
  apenas enfileira uma tarefa `discover_sources` (fila do CP10) e o handler usa
  um mutex; cliques repetidos viram fila, não erro nem concorrência.

### Camada 2 — Volume (por execução)

- `capabilities.max_results_per_run` limita quantos itens uma execução pode
  produzir. O adapter para de paginar ao atingir o teto.
- Caso especial do Vagas.com.br: `max_details_per_run` limita a **2** o número
  de páginas de detalhe por execução, além das `20` permitidas por dia.

### Camada 3 — Cota diária + circuit breaker (SQLite)

Tabelas persistidas pela migração `0005_sources_multi_market.sql`:

- `source_daily_usage (source_id, day, used, details_used, updated_at)` —
  contador por dia **local**; a virada de meia-noite não exige job: o novo dia
  simplesmente não tem linha. `quota_unit` decide qual coluna conta
  (`requests` para InfoJobs, `details` para Vagas.com.br). Quando a cota de
  **requests** esgota, a fonte é pulada com status `quota` e mensagem “limite
  diário atingido”; quando a cota é de **details**, a listagem (gratuita)
  continua rodando e apenas o enriquecimento de detalhes é desligado, com a
  mensagem “limite diário de detalhes atingido”.
- `source_health (source_id, status, consecutive_failures, backoff_until,
  degraded_until, last_error, last_success_at, updated_at)` —
  - **429/403**: backoff de 1h, depois 4h, depois 24h (escalada pelo número de
    falhas consecutivas);
  - **3 falhas consecutivas** (qualquer causa): `degraded` por 24h; a fonte não
    é executada e um evento `source.degraded` alimenta o sino (CP10);
  - **sucesso**: zera a sequência e fecha backoff/circuito;
  - estados expirados são curados preguiçosamente em `Check`, preservando a
    sequência de backoff e reiniciando a de degraded.

Instantes são gravados em UTC (e relidos com `parseUTC`), garantindo que um
`backoff_until` continue válido após reinício, independentemente do fuso local.

## Alternativas rejeitadas

- **Só limite em memória** — reinício zera contador diário e esquece bloqueios;
  o usuário poderia estourar a cota do dia sem perceber.
- **Rejeitar cliques concorrentes com 409** — o requisito é enfileirar, não
  falhar; a fila de tarefas do CP10 já oferece progresso visível em `/tasks`.
- **Contador diário em UTC** — a cota do provedor é observada por dia local do
  usuário; usar UTC mudaria a virada para horários estranhos.
- **Limite global único** — fontes têm limites distintos; um bucket global
  desperdiçaria folga das fontes rápidas e não protegeria as lentas.

## Consequências

- O estado de rate limiting é auditável em `/settings/sources` (tipo, rate
  limit, uso hoje, saúde) e no endpoint `/api/sources`.
- `POST /api/sources/{id}/test` também passa pelo limiter e não consome cota de
  detalhes, evitando que um teste manual esgote a cota diária.
- Testes usam `NewWithClock` para tornar determinísticos o dia e as janelas de
  backoff/degraded.
- A migração é aditiva: as migrações existentes ficam intactas e as 10 novas
  fontes são semeadas com seus limites.
