# Checkpoint 14 — Expansão multi-mercado via APIs públicas e privadas (PT-BR e ES-419)

> **Princípio norteador:** este checkpoint segue a diretriz da CP6 revisada. **Só entram fontes com API pública e/ou privada.** Fontes que exigem scraping de HTML (Programathor, Computrabajo, Bumeran) **ficam fora do escopo** e são registradas como backlog no §7. A arquitetura de `sources` + `capabilities` + registry de adapters da CP6 é preservada; o que muda é apenas o **conjunto de adapters implementados** e a **generalização dos campos regionais**.

---

## 1. Fontes com API — lista qualificada

| Fonte | Tipo de API | Documentação | Autenticação | Locale / Região | Observações críticas |
|---|---|---|---|---|---|
| **Get on Board** | **Pública** (zero-auth) + **Privada** (Bearer) | OpenAPI 3.0.3 em `https://www.getonbrd.com/api-doc.html` | Pública: nenhuma. Privada: `Authorization: Bearer <api_key>` | `es` / `pt` / `en` (param `lang`); LATAM tech | API pública expõe categorias, empresas, vagas publicadas e busca (`/api/v0/search/jobs`). Paginação `per_page` máx. 120. Sandbox disponível. |
| **Trampos** | **API REST** (JSON/XML) | Sem documentação pública formal; uso evidenciado em testes técnicos e projetos open-source | Aparentemente **zero-auth** para listagem pública | `pt-BR` / BR | Paginação `?page=N`; listagem completa (~270 vagas) sem filtro textual. A busca textual (`tr=`) é “literal demais” — o filtro precisa ser feito no cliente (keyword filter da CP6). **Risco:** API sem contrato público estável; tratar como fonte de prioridade média e monitorar quebras. |
| **Gupy** | **Pública** (zero-auth) + **Privada** (Bearer) | `developers.gupy.io` (API privada); endpoint público: `employability-portal.gupy.io/api/v1/jobs` | Pública: nenhuma. Privada: Bearer Token gerado em Setup → Configurações Avançadas | `pt-BR` / BR | API pública busca **board-wide** (todas as empresas), keyed by keyword. É a fonte com maior cobertura de empresas BR. Para o openJob, o **endpoint público é suficiente** para discovery. |
| **getManfred** | **Pública** (zero-auth) | Sem página de doc dedicada; feed evidenciado em projetos open-source | Nenhuma | `es` (tech) / Espanha + LATAM | Feed **board-wide** em uma única chamada: `GET https://www.getmanfred.com/api/v2/public/offers`. A API **rejeita chamadas sem `lang`** (EN ou ES). Sem paginação — o catálogo inteiro vem de uma vez. Cobertura forte de vagas em espanhol com faixa salarial publicada. |

> **Por que estas e não outras?** Programathor, Computrabajo e Bumeran **não têm API pública documentada** — dependem de scraping de HTML ou de serviços pagos de terceiros (Apify). Incluí-las violaria a diretriz da CP6 revisada. A **Get on Board** é a única com API pública + privada formalmente documentada e com OpenAPI; **Trampos** e **Gupy** têm APIs REST públicas usadas pela comunidade; **getManfred** é um feed JSON público zero-auth.

---

## 2. Modelo de dados — generalização (aditivo à CP6)

### 2.1 Expansão de `sources`

| Coluna nova | Tipo | Descrição |
|---|---|---|
| `locale` | TEXT | `pt-BR`, `es-419`, `en-US` |
| `regions` | TEXT (JSON) | `["BR"]`, `["LATAM"]`, `["ES"]` |
| `supported_modalities` | TEXT (JSON) | `["remote","hybrid","onsite"]` |
| `supported_contract_types` | TEXT (JSON) | `["clt","pj","freelance","full-time"]` |
| `supported_currencies` | TEXT (JSON) | `["BRL"]`, `["MXN","ARS","CLP"]`, `["EUR"]` |

O orquestrador filtra `sources` por `regions` e `locale` **antes** de executar discovery, evitando chamadas desnecessárias. As `capabilities` da CP6 (`supports_keyword_search`, `rate_limit`, `max_results_per_run`, `attribution_required`) continuam valendo e são declaradas por cada adapter.

### 2.2 Expansão de `jobs` (aditivo)

| Coluna nova | Tipo | Descrição |
|---|---|---|
| `modality` | TEXT | `remote` \| `hybrid` \| `onsite` |
| `contract_type` | TEXT | `clt`, `pj`, `freelance`, `full-time`, `other` |
| `salary_min` / `salary_max` | REAL (nullable) | Valores na moeda original (sem conversão) |
| `salary_currency` | TEXT (nullable) | ISO 4217: `BRL`, `MXN`, `ARS`, `CLP`, `EUR`, `USD` |
| `salary_period` | TEXT (nullable) | `month`, `year`, `hour` |

> **Sem conversão de moeda nesta CP.** O `score_fit` continua baseado em texto; a UI exibe o salário com `Intl.NumberFormat` da CP13.

---

## 3. Tarefas

### 3.1 Adapters — detalhamento por fonte

Cada adapter implementa a interface da CP6 (`Slug`, `Capabilities`, `Fetch`) e declara seu schema de configuração (para a UI de `/settings/sources` renderizar dinamicamente).

**Get on Board:**
- **Pública (discovery):** `GET https://www.getonbrd.com/api/v0/categories/{slug}/jobs?page=N&per_page=120&lang={lang}`. Também há `GET /api/v0/search/jobs` para busca textual.
- **Privada (não usada nesta CP):** requer Bearer token de empresa; só faz sentido se o openJob evoluir para integração com ATS (backlog).
- **`capabilities`:** `supports_keyword_search: true` (via `/search/jobs`); `supports_pagination: true`; `max_results_per_run: 120`; `rate_limit: {60, "minute"}` (conservador); `attribution_required: true`.

**Trampos:**
- **Endpoint:** API REST em JSON ou XML, paginação `?page=N`. A listagem completa (~270 vagas) é obtida **sem termo de busca**; a busca textual (`tr=`) é “literal demais” e tende a perder resultados.
- **`capabilities`:** `supports_keyword_search: false` → o **keyword filter da CP6 roda no cliente**; `supports_pagination: true`; `max_results_per_run: 300`; `rate_limit: {60, "minute"}`; `attribution_required: true`.
- **Risco:** ausência de contrato público → marcar `experimental: true` e tratar falha de forma isolada (agregador tolerante da CP6).

**Gupy:**
- **Endpoint público:** `GET https://employability-portal.gupy.io/api/v1/jobs?keyword={termo}` — busca board-wide, sem autenticação. É a fonte de maior cobertura de empresas BR.
- **`capabilities`:** `supports_keyword_search: true`; `supports_pagination: true` (via offset/limit, validar no adapter); `max_results_per_run: 200`; `rate_limit: {60, "minute"}`; `attribution_required: false` (API pública sem exigência explícita).

**getManfred:**
- **Endpoint:** `GET https://www.getmanfred.com/api/v2/public/offers?lang=ES` (ou `lang=EN`). Retorna o catálogo inteiro em uma chamada, sem paginação.
- **`capabilities`:** `supports_keyword_search: false` → keyword filter no cliente; `supports_pagination: false`; `max_results_per_run: null` (sem teto — o feed é único); `rate_limit: {10, "minute"}` (conservador); `attribution_required: true` (boa prática).

### 3.2 Orquestração multi-mercado (inalterada conceitualmente)

- `POST /api/jobs/discover` aceita `regions`, `locales` e `modalities` opcionais; o orquestrador filtra `sources` habilitadas antes de executar.
- Para fontes com `supports_keyword_search: false` (Trampos, getManfred), o **keyword filter da CP6** é aplicado client-side após o fetch.
- O **filtro de currículo da CP6** (pré-LLM) continua igual, operando no idioma da vaga. O usuário pode cadastrar `resume_keywords` em PT e ES.
- A **fila de tasks da CP10** (`score_job`) é reutilizada sem mudanças; o payload passa a incluir `modality` e `contract_type` para o score considerar fit de modalidade.

### 3.3 UI de configuração de fontes (`/settings/sources`)

Expansão do CRUD da CP6 com:
- Filtros por **Região**, **Locale** e **Modalidade** no topo da lista.
- Colunas adicionais: Locale, Regiões, Modalidades, Contratos, Moedas.
- Formulário gerado a partir do **schema declarado pelo adapter** (como `capabilities` na CP6), incluindo `locale`, `regions`, `supported_modalities`, `supported_contract_types`, `supported_currencies`.
- `POST /api/sources/{id}/test` retorna amostra de vagas **no idioma/região da fonte**, com modalidade e salário (se disponível), para validação antes de habilitar.
- Fontes `experimental: true` (Trampos) exibem *badge* de aviso.

### 3.4 Filtros adaptados a múltiplos idiomas

- **Keyword filter (CP6):** sem tradução automática. O usuário cadastra termos em PT (`desenvolvedor backend`) e ES (`desarrollador backend`). A UI de `/settings/discovery` exibe aviso explícito.
- **Resume filter (CP6):** as `resume_keywords` podem ser extraídas em múltiplos idiomas se o currículo tiver seções em PT e EN. O `min_matches` e `match_mode` continuam iguais.

---

## 4. Critérios de verificação

- Em `/settings/sources`, habilitar **Get on Board**, **Gupy**, **getManfred** e **Trampos** (experimental). Rodar “Buscar agora” → vagas das quatro fontes aparecem, com `locale`, `regions`, `modality` e `salary_currency` corretos.
- Vagas da **Gupy** têm `locale = "pt-BR"` e `salary_currency = "BRL"` (quando informado); vagas da **Get on Board** em CLP/MXN/ARS têm a moeda correta; vagas da **getManfred** têm `salary_currency = "EUR"` ou `"USD"` (conforme a fonte). Se a fonte não informar salário, `salary_currency` é `NULL` (sem inferência).
- Filtrar discovery por `regions = ["BR"]` → apenas **Gupy** e **Trampos** são executadas; Get on Board (LATAM) e getManfred (ES) não são chamadas.
- Cadastrar palavras-chave em PT (`desenvolvedor backend`) e ES (`desarrollador backend`). Rodar discovery com **Get on Board** (`supports_keyword_search: true`) → os termos são enviados na chamada. Rodar com **getManfred** (`supports_keyword_search: false`) → o filtro roda no cliente e vagas com “desarrollador backend” passam.
- **Trampos** (experimental) falhando (API mudou, timeout) → as demais fontes continuam; erro aparece na UI da fonte e no sino de notificações (CP10), sem derrubar o discovery.
- Deduplicação continua funcionando entre fontes (mesma URL não duplica, mesmo que uma fonte seja PT e outra ES).
- UI de `/settings/sources` exibe filtros de região/locale e colunas de modalidade/moeda; o formulário de criação renderiza os campos a partir do schema do adapter.
- Nenhuma regressão nos fluxos da CP6 (vagas remotas em EN continuam funcionando) e da CP13 (i18n da UI).

---

## 5. Registrar

- **ADR `0022-multi-market-source-model.md`** — generalização de `sources` e `jobs` com `locale`, `regions`, `supported_modalities`, `supported_currencies`, `modality`, `contract_type`, `salary_*`. Justificativa: manter a arquitetura declarativa da CP6; a UI se adapta ao schema do adapter. Alternativa rejeitada: criar um segundo pipeline só para vagas locais.
- **ADR `0023-fontes-api-first-para-latam.md`** — decisão de **rejeitar fontes sem API** (Programathor, Computrabajo, Bumeran) nesta CP, mantendo a diretriz da CP6 revisada. Justificativa: APIs são mais estáveis que HTML; o custo de manutenção de scrapers frágeis não se paga nesta fase. Scraping fica como backlog se a cobertura de APIs for insuficiente.
- **ADR `0024-keywords-multilingues-sem-traducao-automatica.md`** — filtro de palavras-chave não traduz automaticamente; o usuário cadastra termos em PT, EN e ES. Justificativa: evitar custo de LLM por discovery e latência; manter previsibilidade. Tradução automática fica como backlog (opt-in).
- **Testes:**
  - Normalização de `modality` e `contract_type` para PT e ES.
  - Extração de salário com moedas locais (BRL, MXN, ARS, CLP, EUR) e `NULL` quando ausente.
  - Deduplicação entre fontes de locales diferentes.
  - Keyword filter com termos em ES e PT (boundary de palavra) para fontes com `supports_keyword_search: false` (Trampos, getManfred).
  - CRUD de fontes com novos campos declarativos.
  - Regressão da CP6: vagas remotas em EN continuam passando.
- **Riscos anotados:**
  - **Trampos sem contrato público:** API sem documentação formal; tratar como `experimental`, com testes de contrato contra fixture e monitoramento de falhas. Se quebrar, isolar (agregador tolerante) e notificar.
  - **getManfred sem paginação:** o catálogo inteiro vem em uma chamada; se crescer demais, pode estourar timeout ou memória. Monitorar tamanho da resposta e adicionar `max_results_per_run` se necessário.
  - **Gupy privada vs pública:** o openJob usa apenas a **API pública zero-auth** para discovery. A API privada (Bearer) exigiria credenciais de empresa — fora do escopo.
  - **Termos regionais de contrato:** “PJ” e “CLT” são específicos do BR; mapeamento documentado no glossário (CP13).
  - **Sem conversão de moeda:** salários exibidos na moeda original; `score_fit` não usa salário.

---

## 6. Fora de escopo (deliberadamente adiado)

- **Fontes sem API pública** (Programathor, Computrabajo, Bumeran, Vagas.com.br) — backlog; exigiriam scraping de HTML (`golang.org/x/net/html`) e violariam a diretriz “API-first” desta CP.
- **API privada da Get on Board** — só faz sentido para integração com ATS (backlog).
- **Tradução automática de keywords** — opt-in futuro; nesta CP o usuário cadastra termos equivalentes.
- **Conversão de moeda** — salários na moeda original.
- **Cobertura de todos os países LATAM** — Get on Board cobre LATAM tech; getManfred cobre ES/LATAM; Gupy cobre BR. Países sem API pública ficam para expansão futura.
- **Automação de candidatura** — mantida proibida (spec §3.3).
