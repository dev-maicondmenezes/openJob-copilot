# Design tokens do openJob (CP12)

Fonte única de verdade: `frontend/src/lib/styles/tokens.css`. Espelho TypeScript:
`frontend/src/lib/styles/tokens.ts`. Regra ESLint: `scripts/eslint-ui-plugin.mjs`
(usada por `npm run lint:ui`). ADR: `docs/adr/0013-design-system-tokens.md`.

## Papéis semânticos

Nunca use uma cor "bonita" diretamente. Use um papel:

| Papel | Token CSS | Uso típico |
|---|---|---|
| Primária | `--color-primary`, `--color-primary-hover`, `--color-on-primary` | Ações principais, links, estado ativo |
| Superfície | `--color-surface`, `--color-surface-elevated`, `--color-surface-soft` | Cards, painéis, fundo da página |
| Borda | `--color-border` | Divisórias, contornos de inputs |
| Texto | `--color-text-primary`, `--color-text-secondary`, `--color-text-muted` | Hierarquia de leitura |
| Feedback | `--color-success`, `--color-warning`, `--color-error`, `--color-info` (+ `-soft`, `-border`) | Alertas, badges, validação |

Os aliases antigos (`--accent`, `--text`, `--surface`, `--radius`, …) permanecem
**somente** como compatibilidade transitória. Código novo usa apenas `--color-*`.

## Tipografia

| Nível | Token | Tamanho / entrelinha |
|---|---|---|
| `h1` | `--text-h1`, `--leading-h1` | 28px / 1.25 |
| `h2` | `--text-h2`, `--leading-h2` | 22px / 1.3 |
| `h3` | `--text-h3`, `--leading-h3` | 18px / 1.35 |
| `body` | `--text-body`, `--leading-body` | 15px / 1.5 |
| `caption` | `--text-caption`, `--leading-caption` | 13px / 1.4 |
| `mono` | `--font-mono`, `--text-mono` | monoespaçada 13px |

Família única: `--font-sans` (UI) e `--font-mono` (código/dados).

## Espaçamento, raio, sombra, motion

- **Escala base 4px:** `--space-05` (2, meio passo para micro-gaps de controls),
  `--space-1` (4), `--space-2` (8), `--space-3` (12), `--space-4` (16),
  `--space-6` (24), `--space-8` (32), `--space-12` (48), `--space-16` (64).
  Valores arbitrários inline são proibidos.
- **Backdrop de overlay:** `--overlay-backdrop`, usado por Modal, Sheet e Drawer.
- **Raio:** `--radius-sm` 6, `--radius-md` 10, `--radius-lg` 16,
  `--radius-pill` 999.
- **Sombra:** `--shadow-sm` (card), `--shadow-md` (modal/menu),
  `--shadow-lg` (sheet/drawer). No desktop (`≥ lg`) cards usam borda, sem sombra.
- **Motion:** `--motion-micro` 120ms, `--motion-transition` 200ms,
  `--motion-route` 320ms; curvas `--ease-standard` e `--ease-out`.
  `prefers-reduced-motion: reduce` zera animações globalmente.

## Breakpoints canônicos

| Nome | Largura | Classe do app |
|---|---|---|
| `sm` | 640px | — |
| `md` | 768px | phone → tablet |
| `lg` | 1024px | tablet → desktop |
| `xl` | 1280px | — |

Largura do cartão de conteúdo no desktop: `--content-max-desktop` (1920px).
Em monitores menores que isso, a página ocupa toda a área após a sidebar.

Regra do app: `< md` = smartphone, `md`–`< lg` = tablet, `≥ lg` = desktop.
Os valores estão em `BREAKPOINTS`/`MEDIA` (`tokens.ts`) e são replicados nos
media queries de `app.css` (CSS não pode usar custom properties em `@media`).
Ver ADR 0014 e `navigation.md`.

## Componentes (`src/lib/ui/`)

Nenhuma tela cria botão, input, card ou diálogo "à mão". Famílias:

- **Ação:** `Button`, `IconButton` (aria-label obrigatório por tipo), `LinkButton`
- **Formulário:** `Field`, `Input`, `Textarea`, `Select`, `Combobox`, `Checkbox`,
  `Switch`, `RadioGroup`
- **Feedback:** `Alert`, `Badge`, `Toast`, `EmptyState`, `Spinner`, `Skeleton`, `ProgressBar`
- **Estrutura:** `Card`, `Panel`, `PageHeader`, `Section`, `Tabs`, `SegmentedControl`
- **Navegação:** `BottomNav`, `Sidebar`, `Icon`, `Menu`
- **Overlay:** `Dialog` (Sheet < lg / Modal ≥ lg), `Drawer`
- **Dados:** `DataView` (cards < lg / tabela ≥ lg)

## Página de referência

`/dev/ui` é um "storybook mínimo", disponível apenas quando a variável de
ambiente `OPENJOB_DEV_UI=1` está presente no dev/build (`npm run dev:ui`).
Em produção sem essa variável a página mostra apenas um aviso; o conteúdo não
é registrado no bundle quando a flag é omitida pelo `define` do Vite.

```bash
cd frontend
OPENJOB_DEV_UI=1 npm run dev     # ou: npm run dev:ui
# abra http://localhost:5173/dev/ui
```

## Lint de UI

```bash
npm run lint:ui        # detector Node + regra ESLint ui/no-ad-hoc-style
```

Falha o CI quando `src/routes/**/*.svelte` **ou** `src/lib/ui/**/*.svelte` contém:

- cor literal (`#hex`, `rgb()`, `hsla()`, nomes de cor);
- `padding`/`margin`/`gap` com `px`/`rem`/`em` literais;
- `border-radius` com valor literal.

**Escape hatch documentado:** `/* ui-lint-ignore */` na linha anterior (ou na
própria linha) e `/* ui-lint-disable */` no início de um bloco `<style>` para
exceções legítimas (ex.: largura calculada, pixel de gráfico). Exceptions devem
ser justificadas no PR — o objetivo é manter o drift visível, não bloquear trabalho.
