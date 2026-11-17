# Checkpoint 15 — Dashboard de estatísticas e ações na home (4–5 dias)

> Origem: placeholder existente na home (CP2) + dados produzidos pelos CPs 5, 6, 7, 10, 14. Pré-requisito: **CP10, CP12, CP13 e CP14 concluídos**. Substitui o placeholder por um painel acionável; **não** introduz fluxos novos — apenas agrega e apresenta o que já existe.

## 1. Objetivo

Transformar a home em um **dashboard útil** que responde, em até 5 segundos de leitura, a três perguntas:

1. **“O que precisa da minha atenção agora?”** — ações pendentes, tarefas falhando, fontes com erro, configurações incompletas.
2. **“Estou progredindo?”** — funil de candidaturas (descobertas → com fit → candidaturas → entrevistas → ofertas) e atividade recente.
3. **“De onde vêm as oportunidades?”** — performance por fonte (descobertas, conversão, saúde).

**Princípio norteador:** *métricas acionáveis > métricas de vaidade*. Nenhum número aparece sem responder “e daí?”. Cada bloco tem um caminho claro de ação (link para `/jobs`, `/applications`, `/tasks`, `/settings/*`). O layout respeita a fundação do CP12 (responsivo por dispositivo), o dark mode e o i18n do CP13, e usa as fontes multilíngues do CP14.

> **Substituição de placeholder:** o card genérico atual da home é removido. Ele é reaproveitado como **estado de carregamento** (*skeleton*) da primeira carga do dashboard.

## 2. Pré-requisitos

- **CP5** — `jobs` com `score_fit`, `created_at`, `source`.
- **CP6** — `sources` com `enabled`, `experimental`, `capabilities`; `discovery_runs` com contagens por fonte.
- **CP7** — `applications` com `status` (`pending|sent|interview|rejected|offer`) e `created_at`, `updated_at`.
- **CP10** — `tasks` (fila com estados) e `notifications` (feed persistido); event bus disponível.
- **CP12** — tokens, `lib/ui/` (`Card`, `PageHeader`, `Section`, `Stat`, `Alert`, `EmptyState`, `Skeleton`, `DataView`).
- **CP13** — i18n (pt-BR, en-US, es-419) e `Intl` para formatação de data/número.
- **CP14** — `jobs.modality`, `salary_currency`, `locale`; `sources.locale`, `regions`.

## 3. Métricas — o que entra e por quê

Cada bloco justifica sua presença com uma **ação associada**. Se a métrica não leva a uma ação, fica fora (§7).

### 3.1 Precisa de atenção (bloco de topo)

Cards horizontais (scroll no mobile, grid no desktop), cada um com **contagem + link de ação**:

| Alerta | Origem do dado | Ação associada |
|---|---|---|
| Candidaturas **pendentes de envio** | `applications.status = 'pending'` | → `/applications` |
| Tarefas **falhando** (últimas 24h) | `tasks.status = 'failed' AND finished_at > now-24h` | → `/tasks?status=failed` |
| Fontes com **erro na última execução** | `discovery_runs.errors` por fonte (última execução) | → `/settings/sources` |
| **Notificações não lidas** | `notifications.read_at IS NULL` | → `/notifications` |
| **Última descoberta** há mais de 7 dias | `MAX(discovery_runs.finished_at)` | → `/jobs` (botão “Buscar agora”) |
| **Onboarding incompleto** (IA, Telegram, fontes, keywords) | `settings` + `sources` | → item específico em `/settings/*` |

**Regras:**
- Alerta só aparece quando o estado existe (é derivado, não persistido).
- **Sem “dispensar”** nesta CP — o alerta some quando a causa é resolvida. *Snooze* de 24h fica no backlog (§7) para evitar esconder problemas reais.
- Se **nenhum** alerta existir, o bloco some e um *empty state* positivo aparece: “Tudo em dia.” (com ícone e cor `success`).

### 3.2 Progresso — funil de candidaturas

Barra visual de funil (não gráfico de pizza; funil comunica ordem):

```
Vagas descobertas  →  Com fit  →  Candidaturas  →  Enviadas  →  Entrevistas  →  Ofertas
      1.247            892           41             28            6             1
```

- **Vagas descobertas:** `COUNT(jobs)`.
- **Com fit:** `COUNT(jobs WHERE score_fit IS NOT NULL)`.
- **Candidaturas:** `COUNT(applications)`.
- **Enviadas/Entrevistas/Ofertas:** `COUNT(applications WHERE status = ...)`.
- **Taxas exibidas abaixo do funil** (pequenas, em `caption`):
  - Conversão descoberta → candidatura: `apps / jobs`.
  - Taxa de resposta: `(interview + rejected) / sent`.
  - Taxa de entrevista: `interview / sent`.

**Ação:** cada estágio é clicável → `/jobs` (com filtro de fit) ou `/applications` (com filtro de status).

**Justificativa:** é o “pulso” do produto. O usuário vê em um número se está preso em alguma etapa (ex.: muitas descobertas, poucas candidaturas = precisa investir em tailoring).

### 3.3 Atividade recente (últimos 7 dias, configurável)

Dois *sparklines* (barras por dia, sem eixo Y):

- **Vagas descobertas por dia** — última `period` (default 7d).
- **Candidaturas criadas por dia** — mesma janela.

Abaixo, uma linha de contexto:
- “Última descoberta: há 3h · 47 novas vagas · fontes: Gupy, Get on Board.”
- “Última candidatura: há 2 dias · Backend Go @ Acme.”

**Ação:** link para `/jobs` (rodar busca) ou `/applications` (ver pipeline).

**Seletor de período:** `7d` (default) · `30d` · `all`. Muda os sparklines e o bloco de fontes (§3.4) simultaneamente.

**Justificativa:** mostra tendência sem exigir leitura de log. Se o usuário sumiu por uma semana, vê que a atividade caiu e pode agir.

### 3.4 Performance por fonte (período selecionado)

Tabela compacta (mobile: lista; desktop: `DataView` do CP12) com **top 5 fontes**:

| Fonte | Descobertas | Com fit | Candidaturas | Conversão | Saúde |
|---|---|---|---|---|---|
| Gupy | 412 | 287 | 18 | 4,4% | ✅ |
| Get on Board | 203 | 156 | 9 | 4,4% | ✅ |
| Remote OK | 89 | 61 | 4 | 4,5% | ⚠️ (janela de 100) |
| Trampos | 34 | 12 | 1 | 2,9% | 🔴 (erro na última) |
| getManfred | 41 | 30 | 2 | 4,9% | ✅ |

- **Saúde** deriva de `discovery_runs.errors` + `sources.experimental` (CP6/CP14). Fontes `experimental` mostram ícone de aviso.
- Ordenação default: por descobertas no período. Colunas ordenáveis.
- **Fontes com 0 descobertas no período** aparecem em cinza, no fim.

**Ação:** clicar na linha → `/settings/sources` (editar/testar) ou `/jobs?source=<slug>` (ver vagas).

**Justificativa:** responde “qual fonte vale a pena manter habilitada?” e “qual está quebrada?”. Sem isso, o CRUD de fontes do CP6/CP14 fica às cegas.

### 3.5 Fit da IA (compacto, secundário)

- **Fit médio** das últimas 20 vagas com score (número + tendência vs. período anterior).
- **Histograma** de 5 buckets (`0–20`, `20–40`, `40–60`, `60–80`, `80–100`) em barras verticais — sem eixo.
- **Top 3 vagas com fit alto ainda não aplicadas** (atalho para ação imediata).

**Ação:** cada vaga do top 3 → `/jobs/[id]`. Link “ver todas com fit ≥ 70” → `/jobs?min_score=70`.

**Justificativa:** é o diferencial do produto (curadoria por IA). Se o fit médio estiver baixo, o usuário sabe que precisa ajustar keywords (CP6) ou o currículo (CP4).

### 3.6 Fora da home (não duplicar)

- Notificações detalhadas → `/notifications` (CP10).
- Fila de tarefas completa → `/tasks` (CP10).
- Vagas → `/jobs`. Pipeline → `/applications`.

A home **não** replica telas; apenas resume e linka.

## 4. Modelo de dados — sem novas tabelas

Todas as métricas são **derivadas** de tabelas existentes. Não há migração de schema; apenas **índices** para acelerar as queries (criados via migration aditiva):

| Índice | Tabela | Uso |
|---|---|---|
| `idx_jobs_created_at` | `jobs` | Sparkline e filtro por período |
| `idx_jobs_source_score` | `jobs(source, score_fit)` | Performance por fonte |
| `idx_applications_status_created` | `applications(status, created_at)` | Funil e sparkline |
| `idx_tasks_status_finished` | `tasks(status, finished_at)` | Alertas de falha |
| `idx_notifications_unread` | `notifications(read_at)` (já existe do CP10) | Badge |

**Cache em memória (não persistido):**
- O handler `GET /api/dashboard` calcula as agregações e guarda o resultado em cache **por até 30 s**, chaveado por `(period, locale)`.
- **Invalidação**: o event bus do CP10 publica eventos (`job.discovered`, `application.status_changed`, `task.completed`, `task.failed`, `source.error`) → o dashboard invalida o cache. Sem TTL arbitrário longo, sem polling agressivo.
- Volume esperado (single-user local, < 100k vagas) → queries diretas são rápidas; cache é *belt-and-suspenders*, não obrigatório para correção.

## 5. Endpoint único

`GET /api/dashboard?period=7d|30d|all` (default `7d`) — retorna um único JSON com os blocos:

```json
{
  "attention":      { "items": [ { "kind": "pending_applications", "count": 3, "link": "/applications" }, ... ] },
  "funnel":         { "discovered": 1247, "with_fit": 892, "applications": 41, "sent": 28, "interview": 6, "offer": 1,
                      "rates": { "discovery_to_app": 0.033, "response": 0.25, "interview": 0.21 } },
  "activity":       { "period": "7d", "jobs_per_day": [ {...}, ... ], "apps_per_day": [ {...}, ... ],
                      "last_discovery": { "at": "...", "new_jobs": 47, "sources": ["gupy","getonbrd"] },
                      "last_application": { "at": "...", "job_title": "Backend Go @ Acme", "job_id": 42 } },
  "sources":        { "top": [ { "slug": "gupy", "label": "Gupy", "discovered": 412, "with_fit": 287,
                                 "applications": 18, "conversion": 0.044, "health": "ok" }, ... ] },
  "fit":            { "avg_last_20": 68.3, "trend_vs_previous": +2.1,
                      "histogram": [ {"bucket": "0-20", "count": 3}, ..., {"bucket": "80-100", "count": 41} ],
                      "top_unapplied": [ { "job_id": 123, "title": "...", "company": "...", "score": 91 } ] }
}
```

- Todos os campos numéricos são **puros** (sem formatação) — a formatação (`Intl`) fica no cliente (CP13).
- Blocos vazios retornam `null` (ex.: `fit.top_unapplied = []`) em vez de erro.
- Sem paginação — o payload é pequeno (< 10 KB).
- Sem `POST` — dashboard é **somente leitura**.

## 6. UI — layout e comportamento

### 6.1 Estrutura (ordem vertical, respeitando o CP12)

1. **`PageHeader`** — “Início” + subtítulo contextual (“Bom dia. 3 itens precisam da sua atenção.”) + seletor de período (`SegmentedControl`: 7d / 30d / tudo).
2. **Precisa de atenção** — cards horizontais (mobile: scroll-x com *snap*; tablet/desktop: grid).
3. **Progresso (funil)** — barra horizontal com 6 estágios + taxas abaixo.
4. **Atividade** — dois sparklines lado a lado (mobile: empilhados).
5. **Performance por fonte** — `DataView` (cards no mobile, tabela no desktop — CP12).
6. **Fit da IA** — fit médio + histograma + top 3 não aplicadas.

**Grid responsivo:**
- `< md` (smartphone): 1 coluna; bloco “atenção” em scroll horizontal.
- `md`–`< lg` (tablet): 2 colunas para “atividade” e “fontes”; “atenção” continua full-width.
- `≥ lg` (desktop): grid 3 colunas; “atenção” full-width no topo, funil e fit lado a lado, atividade e fontes abaixo.

### 6.2 Estados

- **Carregando:** o **placeholder atual** (que será removido) é reaproveitado como *skeleton* — cards cinza com shimmer, sem “pular” de layout na chegada dos dados.
- **Vazio (novo usuário):**
  - Sem vagas descobertas → *EmptyState* com CTA “Fazer a primeira busca” → `/jobs`.
  - Sem candidaturas → *EmptyState* “Crie sua primeira candidatura a partir de uma vaga” → `/jobs`.
  - Sem chave de IA → card de alerta com link para `/settings/ai`.
  - Sem fontes habilitadas → alerta com link para `/settings/sources`.
- **Erro parcial:** se um bloco falhar (ex.: `fit` com erro de query), os demais renderizam; o bloco com erro mostra mensagem inline com “Tentar novamente”.
- **Offline/local-first:** sem dependência externa; funciona mesmo sem internet (exceto fontes).

### 6.3 Comportamento

- **Atualização:** *fetch* ao montar e ao voltar o foco da aba (`visibilitychange`); sem polling contínuo. Botão discreto de “Atualizar” no `PageHeader` para o usuário forçar.
- **Live update via event bus:** quando uma tarefa termina (CP10) ou uma candidatura muda de status, o dashboard **invalida e re-fetcha** o bloco afetado (se o usuário estiver na home).
- **Seletor de período** persiste em `localStorage` (default `7d`).
- **Acessibilidade:** cada número é lido com rótulo (`aria-label`), sparklines têm `role="img"` com resumo textual, cores nunca são o único indicador (ícone + cor em saúde de fonte).
- **i18n:** números via `Intl.NumberFormat`, datas via `Intl.DateTimeFormat`, textos via chaves do CP13. Verbo do “Bom dia” segue o horário local do dispositivo.
- **Dark mode:** todos os gráficos usam tokens do CP12 (sem cor hardcoded); sparklines funcionam nos 2 temas.

### 6.4 Stack de gráficos (decisão)

- **Sem biblioteca de gráficos** (uPlot, Chart.js, LayerCake). Os gráficos do dashboard são simples — sparkline de barras, barra de funil, histograma — e cabem em **SVG inline via Svelte**, com tokens do CP12. Justificativa: bundle menor (relevante para CP11 — binário embutido), controle total de acessibilidade, zero manutenção de dependência pesada. ADR dedicado (§8).

## 7. Critérios de verificação

- **Novo usuário (banco vazio):** home mostra *empty states* com CTAs funcionais; nenhum erro de query; nenhum bloco “quebrado”.
- **Usuário com dados:** os 5 blocos renderizam números **consistentes** com as telas de origem (spot-check: `COUNT` de `/jobs`, `/applications`, `/tasks`, `/notifications` bate com o dashboard).
- **Seletor de período:** trocar 7d → 30d → tudo atualiza sparklines e bloco de fontes; o funil e o fit (que são totais) **não** mudam com o período (funil é *lifetime*; sparkline é período).
- **Precisa de atenção:** criar uma candidatura `pending` → alerta aparece; marcar como `sent` → alerta some. Falhar uma tarefa → alerta aparece em até 30s (invalidação via bus).
- **Saúde por fonte:** forçar erro em uma fonte (CP6) → a linha fica vermelha na próxima atualização; fonte `experimental:true` (CP14, ex.: Trampos) mostra aviso, não vermelho.
- **Fit médio:** os buckets do histograma somam o total de vagas com `score_fit` (invariante testável).
- **Performance:** `GET /api/dashboard` responde em < 200 ms (p95) com 10k vagas + 200 aplicações em ambiente de teste; com cache quente, < 20 ms.
- **Responsividade (CP12):** 375×667 (smartphone, scroll-x no bloco de atenção), 768×1024 (tablet, grid 2 col), 1440×900 (desktop, grid 3 col) — sem quebra, sem overflow.
- **Dark mode (CP13):** todos os 6 blocos legíveis nos 2 temas; axe sem violações sérias.
- **i18n (CP13):** trocar locale → todos os textos traduzem **sem reload**; números e datas reformatados pelo `Intl`.
- **Acessibilidade:** navegação por teclado em todos os links/cards; `aria-label` em sparklines; contraste AA nos 2 temas.
- **Sem regressão:** testes dos CPs 1–14 continuam verdes; nenhuma tabela alterada (apenas índices).

## 8. Registrar

- **ADR `0025-dashboard-actionable-over-vanity.md`** — toda métrica exibida precisa de uma **ação associada**; métricas sem ação (tempo total no app, número de notificações já vistas, tamanho do currículo em palavras) ficam fora. Justificativa: dashboards de vaidade inflam a UI e não mudam comportamento; a home precisa ser **curta e acionável**.
- **ADR `0026-dashboard-no-chart-library.md`** — gráficos do dashboard em **SVG inline via Svelte** com tokens do CP12, sem biblioteca externa. Justificativa: os gráficos são triviais (sparkline, funil, histograma); uma lib adiciona bundle ao binário (CP11) e complexidade de tema (CP13) sem ganho. Alternativa rejeitada: LayerCake/Chart.js.
- **ADR `0027-dashboard-cache-invalidation.md`** — cache em memória (30 s) + invalidação por eventos do bus (CP10); sem tabela de snapshots. Justificativa: single-user local, volume baixo; snapshots persistidos são complexidade desnecessária. Alternativa adiada: `dashboard_snapshots` (se performance virar problema em datasets anômalos).
- **Testes:**
  - Agregações com *fixtures* cobrindo: usuário novo (vazio), usuário com dados, fonte com erro, tarefa falhando.
  - Invariante do histograma de fit (soma = vagas com `score_fit`).
  - Invalidação do cache via bus (publicar `application.status_changed` → próximo `GET` reflete o novo status).
  - Seletor de período: 7d/30d/all retornam janelas corretas (sem *off-by-one* em virada de dia).
  - Estados de carregamento/vazio/erro (snapshot visual por viewport — CP12).
  - Snapshot visual por tema × locale × viewport (matriz mínima do CP13).
- **Riscos anotados:**
  - **Métricas enganosas em datasets pequenos:** com < 5 candidaturas, taxas como “taxa de resposta = 0%” desanimam sem significância. Mitigação: exibir taxas só a partir de um mínimo (ex.: `sent >= 5`); abaixo disso, mostrar “—” com tooltip “dados insuficientes”.
  - **Fonte com nome ausente** (slug sem label): exibir o slug como fallback — nunca em branco.
  - **Cache stale em caso de erro:** se a invalidação falhar (bug no bus), o usuário vê dados antigos por até 30 s; mitigado pelo TTL curto e pelo botão “Atualizar”.
  - **Timezone do usuário vs. `created_at` UTC:** sparklines agrupam por dia **no fuso local** do navegador, não em UTC — senão a barra de “hoje” some em fusos negativos. Backend retorna timestamps em UTC; cliente agrupa.
  - **Onboarding checklist permanente:** se o usuário nunca configurar IA, o alerta fica para sempre. Backlog: “snooze de 7 dias” por alerta (§9).
  - **Bundle size:** gráficos inline crescem o componente da home; manter `< 15 KB` gzip para o chunk da home (medir no CI, como no CP11).
- **Backlog (não nesta CP):** *snooze* de alertas; comparação com período anterior mais rica (YoY); exportação CSV/PDF do dashboard; notificação “resumo semanal” no Telegram (CP7) usando os mesmos números; filtros por fonte/locale no dashboard (hoje é global); widget “tempo economizado” (estimativa de horas).

## 9. Fora de escopo (deliberadamente adiado)

- **Métricas de vaidade** — tempo de uso, número total de notificações, palavras no currículo.
- **Gráficos avançados** (linha temporal longa, heatmap, sankey do funil) — os 3 gráficos simples cobrem o uso.
- **Personalização do dashboard** (arrastar blocos, ocultar métricas) — backlog pós-validação.
- **Multi-tenant / comparação entre usuários** — o app é local-first single-user (CP11).
- **Integração com Google Analytics / telemetria** — local-first (CP11) e privacidade (ADR do CP11) proíbem.
- **Backfill histórico** — o dashboard olha para frente a partir do momento em que o dado existe; usuários antigos não ganham “dados de antes” que nunca foram coletados.