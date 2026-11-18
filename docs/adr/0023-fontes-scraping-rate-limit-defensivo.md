# ADR 0023 — Incluir fontes sem API com rate limiting defensivo

**Status:** aceito (Checkpoint 14)
**Data:** 2026-11-17
**Relaciona:** docs/plan/14-multi-mercado-latam.md, ADR 0007, ADR 0025,
`internal/discovery/adapters_*.go`, `internal/discovery/ratelimit/`

## Contexto

O CP6 modelou fontes como `sources` + `capabilities` + registry de adapters,
mas só aceitava fontes com API pública. Quatro APIs (Get on Board, Trampos, Gupy
e getManfred) cobrem parte do mercado LATAM, porém deixam de fora sites com
grande volume de vagas (Programathor, Computrabajo, Bumeran, Vagas.com.br,
Catho e InfoJobs). Excluí-las por princípio reduz a cobertura de forma
significativa; incluí-las sem defesa pode bloquear o IP do usuário.

## Decisão

Fontes sem API passam a ser **cidadãs de primeira classe**, com as mesmas
garantias de isolamento e observabilidade das fontes com API, desde que cada
adapter cumpra um contrato defensivo:

1. **Declarar limites nas `capabilities`** — `rate_limit` (ritmo),
   `max_results_per_run` (volume), `requires_daily_quota` / `daily_quota` /
   `quota_unit` (cota diária) e `robots_disallowed_paths` (caminhos proibidos).
2. **Respeitar `robots.txt` por construção** — os adapters não montam URLs com
   parâmetros/caminhos desautorizados: Computrabajo não usa `by=`, `pubdate=`
   ou `sal=`; Bumeran não usa `recientes=`/`relevantes=`; Vagas.com.br não
   acessa `/vagas/pesquisas`. O filtro equivalente é aplicado no cliente, após
   o fetch.
3. **Usar apenas HTTP simples e User-Agent identificável**
   (`openJob/1.0 (+https://github.com/maicondmenezes/openjob; respectful crawler)`).
   Não há proxies rotativos, impersonation de browser, resolução de captcha ou
   login em áreas restritas.
4. **Passar pelo rate limiter central** (`internal/discovery/ratelimit`) — o
   adapter recebe um `RateLimiter` via `SourceConfig` e o usa em **toda**
   requisição, inclusive detalhes; nenhum adapter faz `http.Get` direto.
5. **Ser conservador por padrão** — o piso observado é arredondado para baixo
   (ex.: 1 req/2s no Programathor e no Vagas.com.br, 1 req/s nas demais) e o
   token bucket usa burst 1, impedindo duas requisições coladas mesmo quando a
   taxa configurada permitiria. O Programathor ainda mantém um cache de
   listagem de 6h, que evita refetch em execuções próximas.
6. **Isolar falhas** — erro de parsing, timeout ou bloqueio em uma fonte não
   interrompe as demais; o estado aparece em `/settings/sources` e o circuit
   breaker (ADR 0025) suspende a fonte problemática.

Fontes com API continuam sendo executadas primeiro; scraping entra depois, no
ritmo permitido. O usuário vê um aviso explícito na UI de que a fonte não
oferece API e que o openJob usa limites conservadores.

## Alternativas rejeitadas

- **Manter apenas fontes com API** (ADR 0023 original, `api-first`) — cobertura
  incompleta justamente nos mercados-alvo do CP14; o risco de bloqueio é
  tratado por design, não por exclusão.
- **Scraping sem limite, delegado ao usuário** — transfere ao usuário o custo
  de um bloqueio de IP e viola o princípio local-first.
- **Proxies rotativos / browser headless** — contornam bloqueios por meios que
  não queremos operar; aumentam o custo e contrariam a ética de scraping.
- **Um único adapter genérico de scraping** — HTML e contratos divergem demais;
  um adapter por fonte mantém o parsing testável e o blast radius pequeno.

## Consequências

- Toda fonte nova sem API precisa declarar limites e caminhos proibidos; a
  ausência de `rate_limit` é tratada como configuração inválida nos testes.
- Mudanças de HTML quebram o parsing: cada adapter tem teste de contrato com
  fixture e a falha fica isolada pelo circuit breaker.
- O rate limiter é mais lento que o necessário na maioria dos casos; esse custo
  é aceito em troca de não bloquear o acesso do usuário.
- Bloqueios residuais (429/403) são tratados com backoff persistente (ADR 0025),
  nunca contornados.
