# ADR 0015 — Dialog único com Sheet e Modal por breakpoint

**Status:** aceito (Checkpoint 12)
**Data:** 2026-11-16
**Relaciona:** docs/plan/12-design.md §6.4, docs/design/navigation.md

## Contexto

Antes do CP12, cada módulo implementava seu próprio overlay: o CRUD de fontes
usava um modal centralizado com backdrop, a lista de tarefas usava um drawer
inferior, e o sino de notificações um painel absoluto. Cada um tinha seu próprio
comportamento de `Escape`, clique fora e foco — ou não tinha. Isso gera
inconsistência e falhas de acessibilidade (foco perdido, nenhum focus trap).

## Decisão

Uma única API `Dialog` (`src/lib/ui/Dialog.svelte`) renderiza:

- `Sheet` (slide-up, `border-radius` superior) em `< lg` (phone e tablet);
- `Modal` (centralizado, `--shadow-md`) em `≥ lg` (desktop).

`Drawer` é um overlay irmão, usado quando o contexto da lista deve permanecer
visível no desktop (detalhe de job/tarefa). Em telas menores ele se comporta como
sheet — mesma hierarquia, mesma implementação.

Comportamento comum a todos (action `trapFocus`):

- foco no primeiro elemento focusable ao abrir;
- `Tab`/`Shift+Tab` cicla dentro do overlay;
- `Escape` fecha (listener de captura, antes dos atalhos globais);
- clique fora fecha (configurável via `dismissible`);
- foco devolvido ao disparador ao fechar;
- `role="dialog"`, `aria-modal="true"`, `aria-labelledby`.

Consumidores: criar/editar fonte, filtros de vagas, ajuda de atalhos e detalhes.

## Alternativas rejeitadas

- **Componentes separados por dispositivo com APIs diferentes** — duplicação de
  lógica de foco e maior chance de divergência entre caminhos.
- **Sempre modal centralizado** — ruim no mobile (teclado cobre o conteúdo,
  alvos de toque distantes do polegar).
- **Sempre sheet** — desperdício de espaço no desktop e quebra da expectativa
  de "modal" em fluxos de configuração.

## Consequências

- Novos overlays devem usar `Dialog`/`Drawer`; overlays ad-hoc em `routes/` são
  revisados como dívida de design system.
- Testes de interação verificam `Escape`, retorno de foco e ausência de
  duplicação de overlays após redimensionamento.
