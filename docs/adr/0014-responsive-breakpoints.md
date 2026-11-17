# ADR 0014 — Breakpoints canônicos e tablet com bottom nav

**Status:** aceito (Checkpoint 12)
**Data:** 2026-11-16
**Relaciona:** docs/plan/12-design.md, docs/design/navigation.md, ADR 0016

## Contexto

A UI atual é smartphone-first. O pedido é explícito: o tablet deve manter o
**mesmo padrão** do smartphone (bottom nav preservada), enquanto o desktop ganha
sidebar. Sem uma regra documentada, cada módulo acabaria escolhendo seu próprio
ponto de quebra e a navegação mudaria de forma inconsistente.

## Decisão

Quatro breakpoints canônicos: `sm` 640, `md` 768, `lg` 1024, `xl` 1280.
Regra do app:

- `< md` → **smartphone** (referência canônica, inalterada);
- `md`–`< lg` → **tablet** (mesma bottom nav, touch targets e espaçamento
  expandidos, conteúdo centralizado em `max-width: 840px`, grids 2-col);
- `≥ lg` → **desktop** (sidebar persistente, `max-width: 1200px`, grids 2–3 col,
  tabelas e breadcrumb).

O smartphone **não regride**: tablet e desktop são progressivos. A sidebar
desktop contém exatamente os mesmos itens da bottom nav, mais atalhos para
Notificações e Tarefas — nenhuma seção nova. `DataView` e `Dialog` usam o mesmo
critério (`lg`) para trocar de representação, e ambos os caminhos são testados.

## Alternativas adiadas

- **Navigation rail no tablet (padrão Material)** — ver ADR 0016. Adiado até
  validação com usuários; hoje a regra "mesmo padrão do smartphone" é a prioridade.
- **Um breakpoint único (phone vs desktop)** — deixaria o iPad em paisagem sem
  tratamento e forçaria bottom nav em telas grandes.
- **Breakpoints por container query** — interessante no futuro, mas a navegação
  é global; `matchMedia` mantém uma única fonte de verdade para shell e conteúdo.

## Consequências

- Superfície de teste reduzida: 3 viewports canônicos (375×667, 768×1024,
  1440×900) cobrem as três classes. iPad landscape (≥ 1024) cai em desktop com
  sidebar, cobrindo o caso de "tablet grande" sem uma quarta categoria.
- Qualquer decisão de layout precisa citar um desses breakpoints; valores
  arbitrários inline são barrados pelo lint de UI (ADR 0013).
