# openJob — frontend

SPA (SvelteKit 3, `adapter-static` em modo SPA) + PWA. Os assets do build são
embutidos no binário Go via `//go:embed` (ver `docs/adr/0003-adapter-static-spa.md`).

## Desenvolvimento

```sh
npm install
npm run dev -- --open   # dev server com HMR
npm run dev:ui          # OPENJOB_DEV_UI=1 → página de referência em /dev/ui
```

## Build de produção

```sh
npm run build           # gera frontend/build/ (SPA + manifest + service worker)
npm run preview         # serve o build localmente
```

## Verificações

```sh
npm run check           # svelte-check (tipos)
npm run lint:ui         # proíbe cor/spacing/raio ad-hoc em src/routes/
npm run test:e2e        # Playwright: atalhos, viewport, deep link
npm run test:visual     # snapshots por rota × viewport canônico (Docker)
npm run audit:a11y      # axe-core por rota × viewport canônico
```

> **Snapshots visuais rodam em Docker**, na imagem oficial do Playwright pinada
> pela versão de `@playwright/test` (`scripts/visual-test.sh`). Assim o Debian
> local e o Ubuntu do CI renderizam com o mesmo Chromium/fontes — sem isso os
> anti-aliasing e quebras de linha divergem e todos os baselines falham.
>
> Para atualizar baselines use `npm run test:visual -- --update-snapshots`
> (sempre na mesma imagem) e revise o diff manualmente (CP12 §8).

## Estrutura

- `src/routes/` — rotas da aplicação (`/`, `/jobs`, `/jobs/[id]`, `/applications`,
  `/curriculum`, `/settings/*`, `/notifications`, `/tasks`, `/dev/ui`)
- `src/lib/styles/tokens.css` / `tokens.ts` — fonte única de tokens (ADR 0013)
- `src/lib/ui/` — biblioteca de componentes: botões, formulários, feedback,
  estrutura, navegação (`BottomNav`/`Sidebar`), overlays (`Dialog`/`Drawer`) e
  `DataView` (cards < lg / tabela ≥ lg)
- `src/lib/components/` — componentes de domínio (sino, toaster, diff, detalhe de vaga)
- `src/lib/shortcuts.ts` — atalhos de teclado desktop
- `static/` — favicon e ícones PWA (`scripts/generate-icons.mjs` regenera os PNGs)
- `tests/` — snapshots visuais, axe e testes de interação do Playwright

## Responsividade (CP12)

- `< 768px`: bottom nav inalterada, 1 coluna.
- `768–1023px`: bottom nav com touch targets expandidos, `max-width: 840px`.
- `≥ 1024px`: sidebar com ícone + label (rail opcional via `[`),
  `max-width: 1200px`, tabelas/drawers.

Detalhes em `docs/design/navigation.md` e `docs/design/tokens.md`.
