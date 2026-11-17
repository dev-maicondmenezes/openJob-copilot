# Checkpoint 13 — Dark mode + Internacionalização (PT-BR, EN-US, ES-LATAM) (4–5 dias)

> Origem: spec §13 (UX) + backlog do CP12 (dark mode e i18n explicitamente adiados). Pré-requisito: **CP1–CP12 concluídos**. A fundação de tokens do CP12 é a base para o dark mode; a biblioteca `lib/ui` é o ponto único onde strings entram por props.

## 1. Objetivo

Fechar a experiência de UI em dois eixos, **sem mudança de fluxo funcional**:

1. **Dark mode completo** — tri-state `light` / `dark` / `system` (default `system`), aplicado via tokens do CP12, com persistência por dispositivo e *anti-flash-of-wrong-theme* (FOUC). Cobre Web (SPA + PWA) e respeita `prefers-reduced-motion`. O wizard TUI (CP11) **não** entra — herda o tema do terminal.
2. **Internacionalização em 3 locales** — **PT-BR** (primário), **EN-US** (fallback universal), **ES-419** (espanhol neutro da América Latina — **não** `es-ES`). Cobertura: UI web, notificações in-app (CP10), relatórios/alertas Telegram (CP7), erros de API, wizard TUI (CP11) e formatação de datas/números/moedas.
3. **Sem regressão visual**: os 3 viewports do CP12 (smartphone / tablet / desktop) funcionam nos 2 temas × 3 locales, mantendo a decisão de layout intacta.

> **Escolha de locale LATAM:** `es-419` (UN M.49, “Latin America and the Caribbean”) é o código canônico para espanhol neutro latino-americano. Todos os `es-MX`, `es-AR`, `es-CO`, `es-CL`, `es-PE` etc. **caem nele** por fallback. Não criamos variantes regionais nesta CP — cobertura máxima com um artefato. Regras de neutralidade no §4.2.

## 2. Pré-requisitos

- CP12 concluído — tokens em `frontend/src/lib/styles/tokens.css` e `lib/ui/` como fonte única de componentes.
- CP10 concluído — `notifications` (tabela) e event bus. **Será refatorada** (§4.3).
- CP7 concluído — `notify_log` e envio de Telegram.
- CP11 concluído — binário único; wizard TUI com `huh` v2.
- Nenhuma alteração de contrato de API além de campos **aditivos** (`code`/`params` em erros; `i18n_key`/`i18n_params` em notificações).

## 3. Modelo de configuração (novo)

### 3.1 Locale — fonte da verdade e cache

| Camada | Onde | Papel |
|---|---|---|
| Servidor | `settings.locale` (SQLite, CP3) | Fonte da verdade. Editável em `/settings/appearance`. Lido pelo backend nos envios de Telegram (CP7). |
| Cliente | `localStorage.openjob.locale` | Cache para apply instantâneo antes do primeiro render (evita flash de idioma). |
| Detecção inicial | `navigator.languages` | Resolve no primeiro acesso quando não há `settings.locale` nem `localStorage`. |

**Fallback chain:**

- `pt`, `pt-BR`, `pt-PT` → **`pt-BR`**
- `es-MX`, `es-AR`, `es-CO`, `es-CL`, `es-PE`, `es-*` → **`es-419`**
- `en`, `en-US`, `en-GB`, `en-*` → **`en-US`**
- qualquer outro → **`en-US`** (fallback universal)

> **Por que EN-US como fallback final e não PT-BR?** O produto é vendido como “vagas internacionais pagas em USD” — a audiência é internacional. Se o navegador está em japonês ou francês, EN é mais útil que PT. PT-BR continua sendo o default de mercado para `pt-*`.

### 3.2 Tema (dark mode) — por dispositivo

| Item | Valor |
|---|---|
| Estados | `light` \| `dark` \| `system` (default `system`) |
| Persistência | `localStorage.openjob.theme` (**por dispositivo**) |
| Backend | **Nenhum nesta CP** (backlog: sync cross-device) |
| Detecção `system` | `matchMedia('(prefers-color-scheme: dark)')` + **listener em tempo real** |
| Aplicação | atributo `data-theme="light\|dark"` no `<html>` |
| Anti-FOUC | `<script>` inline no `app.html` lê `localStorage` e seta `data-theme` **antes** do CSS |

> **Por que per-device?** Mobile e desktop podem ter preferências diferentes (usuário usa dark no celular à noite e light no desktop). Sincronizar via backend imporia a mesma escolha em todos — pior UX nesta fase.

## 4. Tarefas

### 4.1 Dark mode — tokens e componentes

- **Tokens:** adicionar bloco `[data-theme="dark"] { --color-surface: …; --color-text-primary: …; … }` em `tokens.css`. Manter todos os nomes de token já usados no CP12 — só os valores mudam.
- **Contraste WCAG AA:** validar ≥ 4.5:1 em texto e 3:1 em interativos nos 2 temas (axe cobre).
- **Componentes (`lib/ui/`):** revisar todos para garantir que **nenhum** use cor/valor hardcoded — só tokens. Lint do CP12 já proíbe fora de `lib/ui`, mas `lib/ui` precisa ser auditada manualmente.
- **`DataView` (tabela desktop):** header, zebra, hover, seleção, borda — todos os estados legíveis em dark.
- **`Dialog` (sheet/modal):** overlay `rgba(0,0,0,.6)` em light, `rgba(0,0,0,.8)` em dark; sombra do CP12 substituída por borda em dark (contraste).
- **Imagens e logos:**
  - SVGs com `currentColor` onde possível.
  - Logos de provedores LLM (OpenAI, Anthropic, Google) — usar variante dark oficial ou versão monocromática; **proibido** filtro CSS (`filter: invert`) por causar artefatos.
- **PWA:** `<meta name="theme-color" media="(prefers-color-scheme: dark)" content="#0b0b0c">` + variante light no `app.html`. Manifest permanece único.
- **`prefers-reduced-motion`:** transições de tema respeitam (sem animação quando o usuário pede).
- **Seletor de tema:** `/settings/appearance` com `SegmentedControl` **Light / Dark / System** (ícones sol / lua / monitor). Mudança instantânea, persiste em `localStorage`, sem reload.

### 4.2 i18n — infraestrutura

- **Biblioteca:** `@inlang/paraglide-js` — compile-time, gera funções tipadas por chave, tree-shake por locale (relevante porque o bundle é embutido no binário Go do CP11). Alternativa rejeitada: `svelte-i18n` (runtime, sem type-safety, bundle maior — discutido no ADR).
- **Estrutura de mensagens:** `frontend/messages/{pt-BR,en-US,es-419}.json` com chaves hierárquicas planas (`jobs.search.button`, `settings.appearance.theme.label`, `errors.job.not_found`, `notifications.application.status_changed.title`).
- **Helper único:** `frontend/src/lib/i18n/t.ts` reexporta `t` do paraglide. **Proibido** importar paraglide direto em `routes/` ou `lib/` fora de `lib/i18n/` (lint).
- **Glossário canônico** em `docs/i18n/glossary.md` — manter atualizado é **requisito de PR** que toca strings:

  | PT-BR | EN-US | ES-419 | Observação |
  |---|---|---|---|
  | Vagas | Jobs | Empleos | “Vacantes” aceito como sinônimo no glossário; UI usa “Empleos” |
  | Candidaturas | Applications | Postulaciones | “Aplicaciones” rejeitado (ambíguo = “apps”) |
  | Currículo | Resume | Currículum | “CV” aceito como forma curta |
  | Configurações | Settings | Configuración | |
  | Notificações | Notifications | Notificaciones | |
  | Tarefas | Tasks | Tareas | |
  | Salvar | Save | Guardar | “Salvar” rejeitado em ES-419 |
  | Arquivo | File | Archivo | |
  | Celular | Phone | Celular | “Móvil” rejeitado em ES-419 |

- **Regras de redação ES-419** (registradas no glossário, enforced em revisão):
  - `ustedes` (nunca `vosotros`).
  - `tú` (nunca `vos` — evita voseo regional).
  - “computadora”, “celular”, “aplicación”, “enlace”, “archivo”, “guardar”.
  - Sem regionalismos (“ordenador”, “móvil”, “fichero”, “app” como substantivo principal).
- **Validação de completude:** `npm run i18n:check` no CI — falha se algum locale tiver chave faltando ou chave extra em relação a `en-US` (canônico estrutural).
- **Pseudo-locale `en-XA`** (opcional, **build-only**, nunca em prod): acentua strings para QA detectar hardcoded text.

### 4.3 Refactor do backend e das notificações (CP7 + CP10)

**Problema atual:** Go devolve erros em inglês e o event bus do CP10 emite `Title`/`Body` já resolvidos.

**Solução fase 1 (esta CP):**

- **Erros de API (aditivo):**
  - Backend retorna `{ "code": "job.not_found", "params": {...}, "message": "job not found" }`.
  - Frontend mapeia `code` → `errors.<code>` do i18n. Se `code` desconhecido → exibe `message` (fallback seguro em EN).
  - Refatorar **apenas** os erros de domínio voltados ao usuário (jobs, applications, discovery, settings, sources, tasks, notifications). Erros internos/debug continuam `message` em EN.
- **Notificações in-app (CP10):**
  - Migration **aditiva** em `notifications`: `i18n_key TEXT NULL`, `i18n_params TEXT NULL (JSON)`.
  - Novas notificações gravam `type` (já existe) + `i18n_key` + `i18n_params`; **não** gravam `title`/`body` resolvidos.
  - Notificações antigas mantêm `title`/`body` (renderiza como fallback — **sem migração destrutiva**).
  - Render: `i18n_key ? t(i18n_key, i18n_params) : title + body`.
  - **Benefício:** trocar de locale **re-renderiza** todo o histórico corretamente.
- **Telegram (CP7):**
  - `settings.locale` é lido no momento do envio.
  - `notify_log` continua guardando o texto **resolvido** (é log/auditoria).
  - Relatório diário e notificação `status_change` traduzidos.

### 4.4 Locale — seleção e persistência

- Detecção inicial via `navigator.languages` com o fallback chain do §3.1.
- Aplicação: `<html lang="…">` + `setLocale()` do paraglide **antes** do primeiro render (sincronizado com `localStorage`).
- `/settings/appearance`:
  - `Select` — **Português (Brasil)** / **English (US)** / **Español (Latinoamérica)**.
  - Mudança aplica **instantaneamente** (sem reload) e atualiza `<html lang>`.
  - Persiste em `localStorage` + `PUT /api/settings/appearance { locale }`.
- Backend lê `settings.locale` em `GET /api/settings` e nos envios de Telegram.

### 4.5 Wizard TUI (CP11) localizado

- Detecção via `LANG` / `LC_ALL` do SO (`pt_BR.UTF-8`, `es_MX.UTF-8`, etc.); se não detectar, dropdown na 1ª tela com os 3 locales.
- Pacote `internal/i18n` no Go com **mapas estáticos** (`map[string]string` por locale) — sem dependência de paraglide (backend não usa JS).
- Traduzir: boas-vindas, transparência (permissões/limites — ADR 0012 do CP11), rede, IA, descoberta, Telegram, resumo final.
- Smoke test no CI: `LANG=pt_BR.UTF-8`, `LANG=es_MX.UTF-8`, default (sem `LANG`) — todas as telas renderizam.

### 4.6 Inventário e tradução de conteúdo

- Extrair **todas** as strings hardcoded de `frontend/src/routes/**` e `frontend/src/lib/**` (exceto `lib/ui/`, que recebe por props). Estimativa: **~250–400 chaves**.
- **Não traduzir:** nomes de provedores (OpenAI, Anthropic, Google), nomes de fontes (Remotive, Remote OK, Jobicy, WWR, Himalayas), slugs de adapter, URLs, headers HTTP, códigos de erro (`job.not_found`), chaves de log.
- PT-BR e EN-US traduzidos pelo autor; ES-419 requer **revisão por falante nativo** antes do release (registrar como gate manual no PR, não como teste automatizado).

### 4.7 Formatação localizada

`frontend/src/lib/i18n/format.ts` — wrappers sobre `Intl.*`:

| Função | Exemplo PT-BR | Exemplo ES-419 | Exemplo EN-US |
|---|---|---|---|
| `formatDate` | 15/11/2026 | 15/11/2026 | 11/15/2026 |
| `formatRelative` | há 2 minutos | hace 2 minutos | 2 minutes ago |
| `formatNumber` | 1.234,56 | 1.234,56 | 1,234.56 |
| `formatCurrency` | US$ 1.234,56 | US$ 1.234,56 | $1,234.56 |

- **Backend:** sempre UTC / ISO 8601 / valores numéricos puros. **Nunca** formatar no Go.
- Aplicar em: datas de vagas (CP5/CP6), `created_at` de notificações (CP10), tempos de tarefas (CP10), timestamps de `notify_log` (CP7), salários (se exibidos), tamanhos de arquivo.

### 4.8 Testes

- **Completude de chaves:** `npm run i18n:check` no CI.
- **Snapshot visual por locale × tema × viewport:**
  - **Não** rodar todas as combinações: 3 locales × 2 temas × 3 viewports = 18 por rota é inviável para todas as rotas.
  - **Estratégia:** em `/jobs`, `/applications`, `/settings`, `/notifications` → 3 locales × 2 temas **no viewport desktop** + 1 smoke por locale **no mobile** (2 temas). Cobre regressão sem explodir.
- **Textos longos:** pseudo-locale `en-XA` (dev) ou strings ES-419 reais (~20% maiores que EN) em botões, `SegmentedControl`, `BottomNav` labels, `Sidebar` — validar truncamento e quebra.
- **Dark mode:**
  - `matchMedia` listener troca tema em tempo real em modo `system`.
  - `data-theme` aplicado **antes** do CSS carregar (sem flash). Teste automatizado: abrir page com `theme=dark` em `localStorage` e verificar `document.documentElement.dataset.theme === 'dark'` no `DOMContentLoaded`.
  - A11y (`axe`): 0 violações sérias nos 2 temas × 3 viewports nas rotas core.
- **i18n de notificações:**
  - Criar notificação em pt-BR → trocar para es-419 → texto re-renderiza no feed.
  - Notificação antiga (sem `i18n_key`) continua renderizando `title`/`body` legados.
- **Backend:**
  - Erros com `code` retornam `params` corretos (teste por endpoint crítico: jobs, applications, sources).
  - Frontend mapeia `code` conhecido; `code` desconhecido exibe `message` (fallback).
- **Telegram:** relatório diário gerado em `settings.locale=pt-BR` e em `es-419` (teste de mensageria do CP7 atualizado).
- **Wizard:** smoke nos 3 locales (§4.5).

## 5. Critérios de verificação

- `/settings/appearance`: alternar Light / Dark / System → mudança instantânea, persiste após reload e após fechar/reabrir o navegador.
- Modo `system`: alterar preferência do SO → app muda **sem reload**.
- **Sem flash:** abrir com `theme=dark` em `localStorage` → nenhum frame em light (verificar via DevTools/Render).
- Navegador com `Accept-Language: es-MX` na primeira visita → app em **es-419** (`<html lang="es-419">`).
- `Accept-Language: pt-PT` → **pt-BR**. `Accept-Language: fr-FR` → **en-US**.
- `/settings/appearance`: trocar locale → toda a UI (incluindo histórico de notificações com `i18n_key`) re-renderiza **sem reload**.
- Deep link `/jobs/[id]` mantém o locale aplicado.
- Notificação de mudança para `interview` chega no Telegram **no idioma de `settings.locale`** (PT-BR e ES-419 testados).
- Snapshot visual do CP12 (3 viewports) passa em **dark mode** e nos **3 locales**.
- `npm run i18n:check` no CI: falha se falta chave em `es-419` ou `pt-BR`. Revisão de ES-419 por falante nativo registrada no PR (gate manual, não automatizado).
- Wizard rodado com `LANG=pt_BR.UTF-8`, `LANG=es_MX.UTF-8` e sem `LANG` → todas as telas traduzidas.
- `GET /api/jobs/999` retorna `{code:"job.not_found", params:{id:999}, message:"job not found"}`; frontend exibe “Vaga não encontrada” / “Empleo no encontrado” / “Job not found”.

## 6. Registrar

- **ADR `0017-i18n-paraglide.md`** — paraglide-js (compile-time) vs `svelte-i18n` (runtime). Justificativa: bundle menor (relevante para CP11 — binário embutido), type-safety de chaves, tree-shake por locale. Alternativa rejeitada: `svelte-i18n` (runtime, sem tipos, mais JS no binário).
- **ADR `0018-es-419-latam.md`** — `es-419` como locale LATAM canônico; todos os `es-*` regionais caem nele; glossário de neutralidade (`ustedes`, `tú`, `computadora`, `celular`, `guardar`). **Não** criar variantes regionais nesta CP. Justificativa: cobertura máxima com um artefato; variantes regionais são backlog pós-validação.
- **ADR `0019-dark-mode-tokens.md`** — dark mode via `[data-theme]` + CSS custom properties (sem `dark:` utility do Tailwind, sem CSS-in-JS); tema em `localStorage` (per-device, não sincronizado). Justificativa: alinhamento com CP12; per-device evita conflito entre celular/desktop.
- **ADR `0020-notifications-i18n-keys.md`** — notificações gravam `i18n_key` + `i18n_params`; render traduz no locale atual; notificações antigas usam `title`/`body`. Justificativa: permite trocar de idioma sem perder histórico; evita migração destrutiva.
- **ADR `0021-backend-error-codes.md`** — backend retorna `code` + `params` + `message` (EN fallback); frontend mapeia `code` → chave. Justificativa: i18n sem refactor massivo do backend; fallback seguro para codes desconhecidos.
- **Testes:** ver §4.8.
- **Riscos anotados:**
  - **Qualidade do ES-419:** se o autor não for falante nativo, revisão externa é **gate obrigatório** antes do release; registrar no PR. Termos regionais podem passar despercebidos em revisão não nativa.
  - **Strings longas em ES-419/PT-BR:** +15–25% vs EN; botões `icon-only` com `aria-label` traduzido e truncamento com `title` (tooltip) evitam overflow. Testar `BottomNav` (labels curtos no mobile) e `Sidebar` (labels completos no desktop).
  - **Logos de provedores em dark:** validar visualmente cada um (OpenAI/Anthropic/Google têm variantes oficiais); **proibido** filtro CSS (artefatos).
  - **Telegram em locale errado:** se `settings.locale` não setado, default `en-US`; documentar.
  - **Pseudo-locale `en-XA`:** garantir que só é incluído com build flag (`DEV_LOCALES=1`), nunca no binário de release do CP11.
  - **Cache reativo do paraglide:** `t()` precisa ser reativo ao trocar locale — não cachear em `const` de módulo; usar o padrão Svelte de `$derived` / reatividade.
  - **FOUC de locale:** o script inline anti-FOUC do **tema** já é necessário no CP12/CP13; adicionar `lang` no mesmo bloco para evitar flash de idioma (paraglide aplica no hydrate, mas `<html lang>` deve estar correto antes).
- **Backlog (não nesta CP):** variantes regionais `es-MX`/`es-AR`/`es-CO`; `pt-PT`; sync de tema entre dispositivos via backend; tradução do README (fica bilíngue EN+PT); i18n de logs do backend (logs ficam EN — *developer-facing*); mais idiomas; temas customizáveis além de light/dark/system.

## 7. Fora de escopo (deliberadamente adiado)

- **Mais idiomas** além de pt-BR, en-US, es-419.
- **Variantes regionais de espanhol** (`es-MX`, `es-AR`, etc.) — `es-419` cobre.
- **`pt-PT`** — `pt-BR` serve de fallback.
- **RTL** (árabe, hebraico) — fora do roadmap.
- **Sync de tema via backend** — per-device nesta CP.
- **i18n de logs do backend** — logs ficam EN (*developer-facing*).
- **Tradução do README e docs técnicos** — permanecem bilíngues EN+PT.
- **Temas customizáveis** (além de light/dark/system) e **fontes por locale** — backlog.
- **i18n do wizard além dos 3 locales** — mesmo escopo do app web.
- **Push de notificação com texto localizado no SO** — o CP10 não usa Web Push real (backlog); quando implementar, herda o locale de `settings.locale`.