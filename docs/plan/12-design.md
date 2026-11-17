# Checkpoint 12 — Consolidação do design system e responsividade multi-dispositivo (5–6 dias)

> Origem: spec §13 (UX) + feedback de uso (UI atual 100% smartphone-first). Pré-requisito: **CP1–CP11 concluídos**. Independente do CP8/CP9 — pode rodar em paralelo.

## 1. Objetivo

Consolidar a UI do openJob em três eixos, **sem mudar nenhum fluxo funcional**:

1. **Design system único** — todos os módulos (`/jobs`, `/applications`, `/resume`, `/settings/*`, `/notifications`, `/tasks`) passam a compartilhar tokens, componentes e padrões visuais. Elimina o *drift* atual (botões, espaçamentos, tipografia e estados inconsistentes entre telas).
2. **Responsividade por tipo de dispositivo** — três layouts explícitos:
   - **Smartphone (< 768px):** UI atual **inalterada** (bottom nav, single column).
   - **Tablet (768–1023px):** **mesmo padrão do smartphone** (bottom nav preservada), com espaçamento e *touch target* expandidos, conteúdo em `max-width` centralizado e grids de 2 colunas quando o conteúdo suportar.
   - **Desktop (≥ 1024px):** **sidebar lateral** persistente substitui a bottom nav; conteúdo em `max-width`, grids de 2–3 colunas, tabelas onde fizer sentido, navegação por teclado.
3. **Auditoria módulo-a-módulo** — inventário de todas as telas, mapeamento de desvios do design system e correção. Nenhuma tela nova; nenhuma funcionalidade nova.

> **Princípio:** smartphone é a referência canônica e **não regride**. Tablet e desktop são *progressivos* — adicionam conforto de uso, não removem padrões do mobile. Toda decisão de layout precisa ser explicável por um breakpoint documentado.

## 2. Pré-requisitos

- CP1–CP11 concluídos. Todas as rotas existem: `/`, `/jobs`, `/jobs/[id]`, `/applications`, `/resume`, `/settings` e sub-rotas (`/ai`, `/telegram`, `/sources`, `/discovery`, `/notifications`, `/tasks`), `/notifications`, `/tasks`.
- SPA SvelteKit + PWA do CP2 (ADR 0003), `//go:embed` do CP11 preservado.
- Sem mudança de backend neste CP (exceto correções de bugs visuais, registradas à parte).

## 3. Design system — fundação (novo)

Antes de mexer em telas, formalizar o que hoje está implícito.

### 3.1 Tokens (fonte única)

- **Cores semânticas:** `primary`, `primary-hover`, `on-primary`, `surface`, `surface-elevated`, `border`, `text-primary`, `text-secondary`, `text-muted`, `success`, `warning`, `error`, `info` — CSS custom properties em `frontend/src/lib/styles/tokens.css`, expostas ao Svelte e ao Tailwind (se usado).
- **Tipografia:** escala fixa (`h1` 28/1.25, `h2` 22/1.3, `h3` 18/1.35, `body` 15/1.5, `caption` 13/1.4, `mono`). Uma única família sans + mono.
- **Espaçamento:** escala 4/8/12/16/24/32/48/64 (base 4px). Proibido valor arbitrário inline.
- **Raio:** `sm` 6px, `md` 10px, `lg` 16px, `pill` 999px.
- **Sombras:** 3 níveis (`sm` cartão, `md` modal, `lg` sheet) — sem sombra no desktop onde a borda já separa.
- **Motion:** durações 120ms (micro), 200ms (transição), 320ms (sheet/route) + curvas padrão. Respeitar `prefers-reduced-motion`.
- **Breakpoints canônicos:** `sm` 640, `md` 768, `lg` 1024, `xl` 1280. **Regra do app:** `< md` = smartphone; `md`–`< lg` = tablet; `≥ lg` = desktop.

### 3.2 Biblioteca de componentes (`frontend/src/lib/ui/`)

- **Ação:** `Button` (variant primary/secondary/ghost/danger; size sm/md/lg; loading; icon-only com `aria-label` obrigatório), `IconButton`, `LinkButton`.
- **Formulário:** `Input`, `Textarea`, `Select`, `Combobox`, `Checkbox`, `RadioGroup`, `Switch`, `Field` (label + hint + error).
- **Feedback:** `Toast`, `Alert` (severity), `EmptyState`, `Spinner`, `Skeleton`, `ProgressBar`.
- **Estrutura:** `Card`, `Panel`, `PageHeader` (título + subtítulo + ações + breadcrumb opcional), `Section`, `Tabs`, `SegmentedControl`.
- **Overlay:** `Dialog` (API única, renderiza `Sheet` ou `Modal` — ver §6.4), `Drawer` (detalhe de job/task no desktop).
- **Dados:** `DataView` (ver §6.3).
- **Navegação:** `BottomNav` (smartphone/tablet) e `Sidebar` (desktop) — ver §4.

**Regra:** nenhum componente de tela cria botão/input/card "à mão". Se falta variante, adicionar ao componente base — **não duplicar**. Lint customizado (ESLint) proíbe classes utilitárias de estilo primário fora de `lib/ui/`, com escape hatch documentado.

### 3.3 Iconografia e acessibilidade

- Um único conjunto de ícones (`lucide-svelte`), tamanho 20/24, `aria-hidden` quando decorativo.
- **WCAG AA:** contraste ≥ 4.5:1 em texto e 3:1 em interativos; `:focus-visible` explícito; navegação completa por teclado (`Tab`/`Shift+Tab`, `Escape` fecha overlays, `Enter` ativa).
- **Desktop:** atalhos mínimos (`g j` → jobs, `g a` → applications, `g s` → settings, `g n` → notifications, `g t` → tasks, `?` → ajuda, `/` → busca). Ver §6.5.

## 4. Estratégia de responsividade (novo)

### 4.1 Layout por dispositivo

| | **Smartphone** (`< md` / < 768) | **Tablet** (`md`–`< lg` / 768–1023) | **Desktop** (`≥ lg` / ≥ 1024) |
|---|---|---|---|
| **Navegação primária** | Bottom nav (inalterada) | **Bottom nav** (mesmo padrão), espaçamento e touch target ≥ 48px | **Sidebar lateral** persistente, ícone + label; colapsável para *rail* (só ícone) em janelas estreitas |
| **Conteúdo** | 1 coluna, full-width | 1–2 colunas, `max-width` 840px centralizado | 2–3 colunas, `max-width` 1200px centralizado |
| **Listas** | Cards | Cards em grid 2-col quando faz sentido | Cards **ou** tabela (a critério do módulo — §5) |
| **Ações de página** | Botão primário full-width no rodapé do card | Botão primário à direita do `PageHeader` | `PageHeader` com ações à direita + atalhos |
| **Filtros** | `Sheet` (bottom) | `Sheet` (bottom) | Barra inline no topo da lista |
| **Detalhe (job, task)** | Rota dedicada | Rota dedicada | `Drawer` lateral (não perde contexto da lista) |
| **Modais** | `Sheet` (bottom) | `Sheet` (bottom) | `Modal` centralizado |
| **Breadcrumb** | Não | Não | Sim, no `PageHeader` |

### 4.2 Regras transversais

- **Mobile-first no CSS**, **desktop-first na arquitetura de informação** (a sidebar reflete a mesma hierarquia da bottom nav — não introduz seções novas).
- **Sem “layout fantasma”:** nenhuma tela renderiza elementos que só fazem sentido em desktop (ex.: tabela) no mobile, e vice-versa. `DataView` decide pelo breakpoint e **ambos os caminhos são testados** (§7).
- **`max-width` em desktop** obrigatório — evita linhas ilegíveis.
- **URLs não mudam** por causa de layout. `/jobs/[id]` continua rota no mobile; no desktop o mesmo conteúdo pode abrir em drawer sobre a lista — mas **deep link sempre funciona** (renderiza a rota cheia).
- **PWA (CP2) preservada:** manifest, ícones, service worker e instalação intactos.

## 5. Auditoria módulo-a-módulo

Para cada rota: (a) aplicar os componentes do §3; (b) garantir os três layouts do §4; (c) cobrir estados vazio/carregando/erro; (d) remover estilos ad-hoc. **Zero mudança funcional.**

| Rota | Estado atual | Ajustes obrigatórios |
|---|---|---|
| `/` (dashboard) | Cards empilhados | Grid 3-col desktop, 2-col tablet; `Card`/`Stat` padronizados |
| `/jobs` | Cards mobile; "Buscar agora" (CP6) no header | Filtros: `Sheet` no mobile/tablet, inline no desktop; cards → **tabela** no desktop (título, empresa, fonte, score, data); `EmptyState` "nenhuma vaga"; progresso do discovery (CP6/CP10) visível no header |
| `/jobs/[id]` | Página cheia | Mobile/tablet: rota normal. Desktop: **`Drawer`** lateral mantendo a lista ao fundo; deep link continua abrindo a página cheia |
| `/applications` | Pipeline em colunas (CP7) | Mobile: colunas empilhadas (scroll). Tablet: 2 colunas. Desktop: kanban horizontal com scroll vertical por coluna; ações rápidas viram `⋯` no card |
| `/resume` (CP4) | Formulário longo | `PageHeader` consistente; grupos em `Section`; ações "Salvar"/"Reextrair keywords" no padrão |
| `/settings` (hub) | Lista de links | `Card` de seção com ícone + descrição + chevron; grid 2-col no desktop |
| `/settings/sources` (CP6) | CRUD | Desktop: **tabela** (Nome, Adapter, Enabled, Auth, Rate limit, Última execução, Ações). Mobile/tablet: cards. Formulário de criar/editar em `Dialog`. `POST .../test` com feedback inline padronizado |
| `/settings/discovery` (CP6) | Form + contador 5/5 | `Field` padronizados; preview de keywords do currículo em `Panel`; validação inline |
| `/settings/notifications` (CP10) | Toggles | `Switch` + `Field`; agrupamento em `Section` |
| `/settings/tasks` (CP10) | Sliders numéricos | `Input type=number` + hint com faixa (slider é impreciso no mobile) |
| `/settings/telegram` (CP7) | Token + chat_id | Token sempre mascarado com "revelar" temporário; `POST /api/notify/test` com `Alert` |
| `/settings/ai` (CP3) | BYOK | Padronizar com `/settings/telegram` (mesma estrutura de card e teste) |
| `/notifications` (CP10) | Feed | Filtros por `type`/`severity`: `SegmentedControl` no mobile/tablet, inline no desktop; agrupamento por dia com `Section` |
| `/tasks` (CP10) | Abas + lista | `Tabs`; linha em `Card` (mobile/tablet) → `Table` (desktop); barra de progresso consistente; detalhe em `Drawer` (desktop) / rota (mobile/tablet) |

> O **wizard (CP11) é TUI** e **não** entra nesta auditoria.

## 6. Tarefas

### 6.1 Fundação do design system
- Criar `frontend/src/lib/styles/tokens.css` + `tokens.ts` + documentação em `docs/design/tokens.md`.
- Criar `frontend/src/lib/ui/` com os componentes do §3.2 e uma **página de referência** em `/dev/ui` (protegida por `OPENJOB_DEV_UI=1`, não exposta no build de produção) — “storybook” mínimo.
- Lint: ESLint proibindo classes utilitárias primárias fora de `lib/ui/` (escape hatch documentado).
- `npm run lint:ui` no CI.

### 6.2 Layout shell (navegação)
- Refatorar `frontend/src/routes/+layout.svelte` para:
  - `< md`: bottom nav atual (sem regressão visual).
  - `md`–`< lg`: bottom nav com espaçamento expandido (mesma marcação, CSS ajustado).
  - `≥ lg`: `Sidebar.svelte` com ícone + label; `PageHeader` com breadcrumb; conteúdo com `max-width`.
- **Sidebar:** mesmos itens da bottom nav (Início, Vagas, Candidaturas, Currículo, Configurações) + atalho para Notificações e Tarefas. Colapsável para *rail* via botão ou `[`. Estado em `localStorage`. `aria-current="page"` no ativo.
- **Bottom nav:** revisar ícones (consistência `lucide`), `aria-label` por item, `aria-current` no ativo.
- **`PageHeader`:** `title`, `subtitle`, `actions` (slot), `breadcrumb` opcional. Mobile: título + ações em linha abaixo; tablet: título + ações à direita; desktop: título + breadcrumb + ações.

### 6.3 `DataView` (tabela vs cards)
- Componente que recebe `columns`, `rows` e `renderers` e decide:
  - `< lg` → `CardList` (mesmo conteúdo, sem colunas explícitas).
  - `≥ lg` → `Table` (com `thead`, sorting opcional, sticky header).
- Primeiros consumidores: `/jobs`, `/settings/sources`. Depois `/tasks` e `/notifications`.
- **Nenhuma perda de informação** na troca: toda coluna da tabela aparece como `Field` no card.

### 6.4 `Dialog` (modal vs sheet)
- API única; renderiza `Sheet` (slide-up) em `< lg` e `Modal` (centralizado) em `≥ lg`.
- Foco gerenciado (focus trap), `Escape` fecha, clique fora fecha (configurável), retorno de foco ao disparador.
- Consumidores: criar/editar fonte (CP6), confirmar exclusão, detalhe de task/job (desktop).

### 6.5 Atalhos de teclado (desktop)
- `frontend/src/lib/shortcuts.ts` com: `g j`, `g a`, `g s`, `g n`, `g t`, `?`, `/`, `[`, `Escape`.
- Não disparam com foco em `input`/`textarea`/`select`/contenteditable.
- Ajuda acessível via `?` e item “Atalhos” no rodapé da sidebar.

### 6.6 Acessibilidade (checklist aplicado)
- Auditoria com axe por rota × viewport canônico.
- Corrigir: contraste, `aria-label` em botões só-ícone, ordem de foco, headings hierárquicos, landmarks (`<nav>`, `<main>`, `<header>`).
- Respeitar `prefers-reduced-motion`. **Dark mode fica para backlog**, mas os tokens já preparam.

### 6.7 Ordem de execução sugerida (redução de risco)
1. **Fundação (≈ 2 dias):** tokens, `lib/ui/`, layout shell, `DataView`, `Dialog`. Baseline de screenshots do mobile capturado **antes** do refactor.
2. **Auditoria módulo-a-módulo (≈ 2–3 dias):** aplicar §5 na ordem `/jobs` → `/settings/*` → `/applications` → `/notifications` → `/tasks` → `/resume` → `/`.
3. **Polimento + testes (≈ 1 dia):** atalhos, a11y, snapshots, ajustes.

## 7. Critérios de verificação

- **Smartphone não regride:** screenshots 375×667 de todas as rotas batem com o baseline pré-refactor — diferenças só onde a correção de inconsistência foi aprovada em revisão.
- **Tablet (768×1024):** bottom nav presente (mesmo padrão), conteúdo em `max-width` centralizado, grids de 2 colunas em `/jobs` e `/settings`.
- **Desktop (1440×900):** sidebar presente com ícone + label; bottom nav **ausente**; `/jobs` e `/settings/sources` renderizam tabela; detalhe abre em `Drawer` sem perder a lista; `max-width` respeitado.
- **Redimensionamento 375 → 768 → 1024 → 1440 → volta:** sem perda de estado de formulário, sem *flicker* de navegação, sem duplicação de elementos.
- **Deep link:** `http://localhost:8080/jobs/123` no desktop abre a página cheia (não só drawer) — a URL é a fonte da verdade.
- **Atalhos:** `g j`, `g a`, `g s` navegam; `?` abre ajuda; `/` foca busca; atalhos **não** disparam em campos de texto.
- **Acessibilidade:** `npm run audit:a11y` sem violações sérias em `/`, `/jobs`, `/applications`, `/settings`, `/settings/sources`, `/notifications`, `/tasks` nos 3 viewports.
- **Lint de UI:** PR falha se usar cor/spacing fora dos tokens em `routes/` sem escape hatch.
- **PWA:** Lighthouse PWA ≥ 90 no mobile.
- **Nenhuma regressão funcional:** testes dos CPs 1–11 continuam verdes; contratos de API inalterados.

## 8. Registrar

- **ADR `0013-design-system-tokens.md`** — formalizar tokens em CSS custom properties + componentes em `lib/ui/`, com lint proibindo estilo ad-hoc em `routes/`. Justificativa: elimina drift entre módulos; permite evolução (ex.: dark mode) sem refactor. Alternativa rejeitada: adotar biblioteca pronta (Shoelace/Flowbite) — aumenta superfície de dependência e conflita com a PWA leve do CP2.
- **ADR `0014-responsive-breakpoints.md`** — três breakpoints (`md` = phone→tablet, `lg` = tablet→desktop) e **tablet mantém bottom nav**. Justificativa: respeita o pedido de “mesmo padrão” no tablet; reduz superfície de teste; iPad landscape (≥1024) já cai em desktop com sidebar, cobrindo o caso “tablet grande”. Alternativa adiada: *navigation rail* no tablet (padrão Material) — testar com usuários antes de trocar.
- **ADR `0015-dialog-modal-vs-sheet.md`** — um único `Dialog` que renderiza `Sheet` (< lg) ou `Modal` (≥ lg). Justificativa: consistência de comportamento e foco; uma API só.
- **ADR `0016-tablet-navigation-tradeoff.md`** (curto, anexado ao 0014) — documentar explicitamente o trade-off “bottom nav no tablet vs rail” para orientar validação pós-CP8/CP9.
- **Testes:**
  - **Snapshot visual (Playwright)** por rota × viewport canônico (375×667, 768×1024, 1440×900) — baseline em `frontend/tests/visual/`. Review manual obrigatório quando snapshot muda.
  - **A11y automatizada** (axe-core) por rota × viewport.
  - **Atalhos:** teclas disparam navegação; não disparam em inputs.
  - **Troca de viewport** com formulário preenchido (ex.: `/settings/discovery` com 3 keywords) — estado preservado.
  - **Deep link** em `/jobs/[id]` abre a página no desktop (sem exigir clique na lista).
- **Riscos anotados:**
  - **Flakiness de snapshot:** fixar viewport, `prefers-reduced-motion` e desabilitar animações no CI (`animation: none` via CSS de teste).
  - **Drawer (desktop) vs rota (mobile):** risco de divergência de “voltar”. Regra: no drawer, `Escape` e clique fora fecham e retornam à lista; URL sempre funciona. Documentar em `docs/design/navigation.md`.
  - **Tablet bottom nav vs expectativa de usuário (iPad):** ADR 0014 registra a decisão e o caminho de migração (rail) se validação indicar.
  - **Lint de UI rígido demais:** escape hatch documentado evita bloquear exceções legítimas (ex.: `style` inline para `max-width` calculado).
  - **Acessibilidade no drawer:** focus trap precisa devolver foco corretamente — testar com leitor de tela (NVDA/VoiceOver) ao menos manualmente em `/jobs/[id]` no desktop.
- **Backlog (não nesta CP):** dark mode; navigation rail no tablet; atalhos customizáveis; Storybook dedicado (a página `/dev/ui` cobre o essencial); i18n da UI.

## 9. Fora de escopo (deliberadamente adiado)

- **Dark mode** — tokens preparam, mas implementação fica para depois.
- **Navigation rail no tablet** — ADR 0014 adia até validação com usuários.
- **Redesign de fluxos** — este CP é sobre aderência e responsividade, **não** repensar UX de nenhum módulo.
- **Wizard TUI (CP11)** — permanece intacto; não entra na auditoria de UI web.
- **Novas telas ou funcionalidades** — zero.
- **i18n, temas customizáveis, white-label** — pós-validação (spec §13.9).