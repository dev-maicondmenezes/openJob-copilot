# openJob — frontend

SPA (SvelteKit 3, `adapter-static` em modo SPA) + PWA. Os assets do build são
embutidos no binário Go via `//go:embed` (ver `docs/adr/0003-adapter-static-spa.md`).

## Desenvolvimento

```sh
npm install
npm run dev -- --open   # dev server com HMR
```

## Build de produção

```sh
npm run build           # gera frontend/build/ (SPA + manifest + service worker)
npm run preview         # serve o build localmente
```

## Verificações

```sh
npm run check           # svelte-check (tipos)
```

## Estrutura

- `src/routes/` — rotas `/`, `/settings`, `/curriculum`, `/jobs`, `/applications` (stubs visuais no CP2)
- `src/lib/components/` — `BottomNav` (navegação inferior mobile-first), `PageStub`
- `static/` — favicon e ícones PWA (`scripts/generate-icons.mjs` regenera os PNGs)
- `vite.config.ts` — adapter-static (fallback `index.html`) + `@vite-pwa/sveltekit`

## Rotas (CP2)

Bottom nav com 5 destinos; sem consumo de API ainda (fora de escopo até CP3+).
Deep links (ex.: recarregar `/jobs`) funcionam via fallback do `index.html`.
