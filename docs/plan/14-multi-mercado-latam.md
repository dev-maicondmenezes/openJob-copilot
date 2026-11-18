# Checkpoint 14  — Expansão multi-mercado: APIs, scraping ético e rate limiting defensivo

> **Princípio norteador :** o checkpoint mantém a arquitetura de `sources` + `capabilities` + registry de adapters do CP6. A mudança é que **fontes sem API passam a ser incluídas**, desde que implementadas com um **rate limiter persistente e conservador por fonte**, projetado para nunca exceder os limites observáveis da plataforma. O scraping deixa de ser "backlog" e passa a ser **cidadão de primeira classe**, com as mesmas garantias de isolamento de falhas e observabilidade das fontes com API.

---

## 1. Fontes qualificadas (APIs + scraping)

### 1.1 Fontes com API (inalteradas)

| Fonte | Tipo | Autenticação | Locale / Região |
|---|---|---|---|
| **Get on Board** | Pública (zero-auth) + Privada (Bearer) | Nenhuma (pública) | `es` / `pt` / `en`; LATAM tech |
| **Trampos** | API REST (JSON/XML) | Zero-auth (aparente) | `pt-BR` / BR |
| **Gupy** | Pública (zero-auth) + Privada (Bearer) | Nenhuma (pública) | `pt-BR` / BR |
| **getManfred** | Pública (zero-auth) | Nenhuma | `es` / ES + LATAM |

### 1.2 Fontes sem API — incluídas nesta revisão

| Fonte | Tipo de acesso | Locale / Região | Modalidade | Rate limit observado / conservador | Estratégia de proteção |
|---|---|---|---|---|---|
| **Programathor** | Scraping HTML | `pt-BR` / BR | Remoto, híbrido, presencial | Sem `Crawl-delay` público documentado; adotar **1 req a cada 2s** como piso seguro | Rate limiter por fonte + cache de página (TTL 6h) |
| **Computrabajo** | Scraping HTML (server-rendered) | `es-419` / 19 países LATAM | Remoto, híbrido, presencial | `robots.txt` desautoriza parâmetros de filtro (`by=`, `pubdate=`, `sal=`); paginação (`p=`) permitida. **1 req/s** para listagem; **1 req/s** para detalhe | Filtro client-side após fetch (sem usar parâmetros desautorizados) |
| **Bumeran** | Scraping HTML + API interna (`POST /api/avisos/searchV2`) | `es-419` / AR, PE, MX, VE | Remoto, híbrido, presencial | **1 req/s**; máximo de **1.200 vagas por execução**. `robots.txt` desautoriza apenas `recientes=` e `relevantes=`, não a API interna | Teto de itens por run + rate limiter |
| **Vagas.com.br** | Scraping HTML | `pt-BR` / BR | Remoto, híbrido, presencial | **2s entre requests** na listagem; **~16–20 requests de detalhe por IP por dia** antes de ban de ~24h. `robots.txt` desautoriza `/vagas/pesquisas` | **Limite diário rígido** de detalhes; listagem sem detalhe (card já traz título/empresa) |
| **Catho** | Scraping HTML | `pt-BR` / BR | Remoto, híbrido, presencial | Pausa entre páginas; respeitar `robots.txt`. **1 req/s** conservador | Rate limiter + pausa entre páginas |
| **InfoJobs** | Scraping HTML | `pt-BR` / BR (e ES) | Remoto, híbrido, presencial | **2s entre requests**; bloqueio suave observado após 5 páginas em alguns cenários. **1 req/s** funciona em ~75 requests | Rate limiter + backoff exponencial em 429/403 |

> **Nota sobre o `robots.txt`:** nenhuma das fontes acima proíbe explicitamente o acesso às páginas públicas de listagem de vagas. Onde há restrição (ex.: `pubdate=` no Computrabajo, `/vagas/pesquisas` no Vagas.com.br), o adapter **não utiliza** os caminhos ou parâmetros desautorizados. A coleta ocorre apenas sobre conteúdo público, sem login e sem acessar áreas restritas.

---

## 2. Modelo de rate limiting — a peça central da revisão

O problema não é apenas "qual o limite", mas **como garantir que o usuário possa clicar "Buscar agora" quantas vezes quiser sem que o app seja bloqueado**. A solução é um **rate limiter persistente e hierárquico por fonte**, com três camadas:

### 2.1 Camada 1 — Limite por requisição (ritmo)

Cada adapter declara em `capabilities`:

```json
"rate_limit": {
  "requests": 1,
  "period": "second",
  "strategy": "token_bucket"
}
```

O orquestrador usa `golang.org/x/time/rate` com um **bucket por fonte**. Requisições são enfileiradas e executadas no ritmo declarado, **independentemente de quantos cliques** o usuário deu. Se o usuário clicar 10 vezes, o orquestrador enfileira 10 execuções; o rate limiter as serializa no ritmo seguro.

| Fonte | `requests` | `period` | Estratégia |
|---|---|---|---|
| Programathor | 1 | 2s | Token bucket |
| Computrabajo | 1 | 1s | Token bucket |
| Bumeran | 1 | 1s | Token bucket |
| Vagas.com.br | 1 | 2s | Token bucket + **limite diário** (ver 2.2) |
| Catho | 1 | 1s | Token bucket |
| InfoJobs | 1 | 2s | Token bucket + backoff em 429 |

### 2.2 Camada 2 — Limite por execução (volume)

Além do ritmo, cada fonte tem um **teto de itens por execução de discovery** (`max_results_per_run`). Isso evita que uma única execução esgote a cota diária.

| Fonte | `max_results_per_run` | Justificativa |
|---|---|---|
| Computrabajo | 500 | Listagem completa de um país em ~14 min a 1 req/s; 500 é conservador |
| Bumeran | 1.200 | Limite observado na fonte |
| Vagas.com.br | 200 (listagem) + **0 detalhes** | Detalhes são limitados a ~20/dia; a listagem já traz título/empresa |
| Catho | 300 | Pausa entre páginas; volume moderado |
| InfoJobs | 200 | Evita bloqueio após 5 páginas |
| Programathor | 200 | Piso seguro para site menor |

### 2.3 Camada 3 — Limite diário persistente (a garantia final)

Para fontes com limite diário conhecido (**Vagas.com.br** e **InfoJobs**), o orquestrador mantém um **contador persistente em SQLite** (`source_daily_usage`):

| Fonte | Limite diário | Comportamento ao atingir |
|---|---|---|
| Vagas.com.br | **20 requisições de detalhe** por IP/dia | Adapter para de buscar detalhes; usa apenas os dados da listagem. O usuário vê "limite diário atingido — detalhes disponíveis amanhã". |
| InfoJobs | 75 requisições (conservador) | Adapter retorna o que já coletou e enfileira o restante para o dia seguinte. |

> **Contador persistente:** o uso é registrado em SQLite com `date` (dia local) e `source_id`. Mesmo que o app reinicie, o contador não zera. O usuário pode ver o consumo em `/settings/sources` (coluna "Uso hoje").

### 2.4 Backoff e circuit breaker

- **Backoff exponencial:** em `429` ou `403`, o adapter para imediatamente e agenda retry em 1h, 4h, 24h. O rate limiter **não** é contornado.
- **Circuit breaker por fonte:** se uma fonte falhar 3 vezes consecutivas (erro de parsing, bloqueio, timeout), ela é marcada como `degraded` e **não é executada** por 24h. O usuário é notificado no sino (CP10) e em `/settings/sources`.
- **Isolamento:** falha de uma fonte **nunca** derruba as demais (agregador tolerante do CP6).

---

## 3. Estratégia de scraping por fonte (sem violar `robots.txt`)

| Fonte | O que é coletado | O que **não** é coletado | Estratégia |
|---|---|---|---|
| **Programathor** | Listagem pública de vagas (título, empresa, URL, modalidade) | Páginas de detalhe além do necessário | Cache de listagem (TTL 6h); detalhe sob demanda |
| **Computrabajo** | Cards de listagem (título, empresa, local, salário se presente) | Parâmetros de filtro desautorizados (`by=`, `pubdate=`, `sal=`) | Filtro client-side após fetch; detalhe só quando o usuário clica "Saiba mais…" |
| **Bumeran** | API interna `searchV2` (título, empresa, modalidade, salário) | Páginas `recientes=` e `relevantes=` (desautorizadas) | Rate limiter 1 req/s; teto de 1.200 |
| **Vagas.com.br** | Listagem (`/vagas-de-?pagina=N`) — título, empresa | `/vagas/pesquisas` (desautorizado) | **Sem detalhe** na maioria dos casos; detalhe só com cota diária |
| **Catho** | Páginas públicas de busca | Páginas que exijam login | Pausa entre páginas; rate limiter 1 req/s |
| **InfoJobs** | Listagem e detalhe públicos | Áreas autenticadas | Rate limiter 2s; backoff em 429 |

---

## 4. Tarefas revisadas

### 4.1 Adapters (expansão)

- **Adapters de API (4):** Get on Board, Trampos, Gupy, getManfred — mantidos do CP14 original.
- **Adapters de scraping (6):** Programathor, Computrabajo, Bumeran, Vagas.com.br, Catho, InfoJobs — **novos**.
- **Interface comum** (CP6): todos implementam `Slug`, `Capabilities`, `Fetch`.
- **`Capabilities` estendidas:** cada adapter declara `rate_limit` (ritmo), `max_results_per_run` (volume), `requires_daily_quota` (bool), `daily_quota` (int), `robots_disallowed_paths` (lista de paths que o adapter **não** deve acessar).

### 4.2 Rate limiter central (`internal/discovery/ratelimit/`)

- **Token bucket por fonte** (`golang.org/x/time/rate`), configurado a partir de `capabilities.rate_limit`.
- **Contador diário persistente** (`source_daily_usage`), com reset à meia-noite local.
- **Backoff exponencial** em `429`/`403`, com estado por fonte.
- **Circuit breaker** com estado `degraded` persistido.

### 4.3 UI de configuração de fontes (`/settings/sources`)

- **Colunas adicionais:** "Tipo" (API / Scraping), "Rate limit" (ex.: "1 req/2s"), "Uso hoje" (ex.: "12/20"), "Saúde" (ok / degraded / bloqueada).
- **Badge de aviso** para fontes de scraping: "Esta fonte não oferece API. O openJob respeita limites conservadores para não bloquear seu acesso."
- **Contador diário visível:** o usuário vê quanto da cota diária já foi consumida.
- **Teste de fonte** (`POST /api/sources/{id}/test`): executa **1 requisição** e valida o parsing, sem consumir cota significativa.

### 4.4 Orquestração

- `POST /api/jobs/discover` enfileira a execução de todas as fontes habilitadas, **respeitando os rate limiters**.
- Se o usuário clicar "Buscar agora" 5 vezes seguidas, as 5 execuções são **enfileiradas**; o rate limiter as serializa. O usuário vê o progresso na fila de tarefas (CP10).
- **Prioridade:** fontes com API são executadas primeiro (mais rápidas); scraping entra depois, no ritmo permitido.

---

## 5. Critérios de verificação s

- **Rate limiting sob cliques repetidos:** clicar "Buscar agora" 10 vezes seguidas → as 10 execuções são enfileiradas; nenhuma fonte ultrapassa seu `rate_limit`; o usuário vê a fila em `/tasks`; nenhum bloqueio é disparado.
- **Contador diário (Vagas.com.br):** executar discovery repetidamente até consumir 20 detalhes → o adapter para de buscar detalhes e informa "limite diário atingido"; no dia seguinte, o contador zera e os detalhes voltam.
- **`robots.txt` respeitado:** verificar que o adapter do Computrabajo **não** envia `by=`, `pubdate=` ou `sal=`; que o adapter do Vagas.com.br **não** acessa `/vagas/pesquisas`; que o adapter do Bumeran **não** acessa `recientes=`/`relevantes=`.
- **Backoff:** simular `429` de uma fonte → o adapter para, agenda retry e não afeta as demais.
- **Circuit breaker:** simular 3 falhas consecutivas → a fonte fica `degraded` por 24h; o usuário é notificado.
- **Isolamento:** uma fonte de scraping falhando (HTML mudou) → as fontes de API continuam funcionando; o erro aparece na UI da fonte.
- **UI:** a coluna "Uso hoje" reflete o contador diário; o badge de scraping está visível.
- **Regressão:** as 4 fontes de API do CP14 original continuam funcionando.

---

## 6. ADRs e riscos s

- **ADR `0023-fontes-scraping-rate-limit-defensivo.md`** — substitui o ADR `0023-fontes-api-first-para-latam.md` do CP14 original. Decisão: incluir fontes sem API, desde que com rate limiter persistente, contador diário e respeito a `robots.txt`. Justificativa: a cobertura de mercado é incompleta sem essas fontes; o risco de bloqueio é mitigado por design, não por exclusão.
- **ADR `0025-rate-limiter-hierarquico.md`** — três camadas (ritmo + volume + cota diária) com persistência em SQLite. Justificativa: o clique repetido do usuário não pode violar os limites da fonte; a persistência garante que reinícios não zerem o contador.
- **Riscos anotados:**
  - **HTML instável:** mudanças de layout quebram o parsing. Mitigação: testes de contrato contra fixtures; circuit breaker isola a falha.
  - **Bloqueio por IP:** mesmo com rate limit, um IP residencial pode ser bloqueado se a fonte detectar padrão de scraping. Mitigação: o rate limiter é conservador (mais lento que o necessário); o usuário é avisado; não há tentativa de contornar bloqueios com proxies (isso violaria o princípio local-first e a ética de scraping).
  - **Cota diária do Vagas.com.br:** 20 detalhes/dia é um limite duro; o adapter prioriza a listagem (que já traz título/empresa) e usa detalhe apenas quando o usuário clica "Saiba mais…".

---

## 7. Fora de escopo 

- **Proxies rotativos / impersonation de browser** — o openJob é local-first e não tenta contornar bloqueios; usa apenas HTTP simples com User-Agent identificável.
- **Automação de candidatura** — mantida proibida (spec §3.3).
- **API privada da Get on Board / Gupy** — só faz sentido para integração com ATS (backlog).
- **Tradução automática de keywords** — opt-in futuro.
- **Conversão de moeda** — salários na moeda original.
- **Cobertura de todos os países LATAM** — as fontes de scraping cobrem BR e principais mercados ES; países menores ficam para expansão futura se necessário.

---
