# ADR 0003 — `adapter-static` com fallback SPA (sem SSR)

- **Status:** aceito (Checkpoint 2)
- **Data:** 2026-11-13

## Contexto

O openJob é local-first, servido pelo próprio binário Go na rede local
(`openjob.copilot:8080`). O frontend precisa:

- navegar entre as rotas `/`, `/settings`, `/curriculum`, `/jobs`, `/applications`;
- funcionar como deep link (recarregar `/jobs` no celular sem 404);
- ser instalável como PWA e abrir offline (app shell);
- ser embutido no binário Go via `//go:embed` — logo, precisa ser **100% estático**.

## Decisão

`@sveltejs/adapter-static` em modo SPA puro, sem pré-render:

- `fallback: 'index.html'` — qualquer rota desconhecida serve o shell, e o
  cliente resolve a rota (deep links funcionam no servidor Go e offline);
- `strict: false` — não exige pré-render das rotas;
- `ssr = false` e `prerender = false` globais (`src/routes/+layout.ts`).

## Alternativas descartadas

- **SSR / adapter-node**: exigiria um processo Node ao lado do Go — contra o
  objetivo "um binário, zero dependências". Local-first não tem necessidade de
  render no servidor: não há SEO, nem dados por request no primeiro paint.
- **Pré-render estático de todas as rotas**: funcionaria hoje, mas cada rota
  nova com dados dinâmicos exigiria `prerender = false` individual; o fallback
  SPA cobre todos os casos com uma única regra.

## Detalhe de implementação (desvio do plano)

O plano (escrito para SvelteKit 2) manda criar `svelte.config.js`. No
**SvelteKit 3** (versão atual, instalada em `frontend/`), o arquivo
`svelte.config.js` não é mais carregado: as opções do Kit — incluindo o
adapter — são passadas direto ao plugin `sveltekit()` no `vite.config.ts`.
Mantivemos a semântica do plano (adapter-static, fallback, strict=false), só
mudou o local da configuração.

Integração PWA: `@vite-pwa/sveltekit` v2 exporta `withPwa(config, pwaOptions)`
(antes era o plugin `SvelteKitPWA`). O service worker é gerado em `generateSW`
(opção `workbox` da v1 está deprecada).

## Service worker e versionamento de cache

- `registerType: 'autoUpdate'` + `skipWaiting`/`clientsClaim`: quando um novo
  deploy chega, o SW novo assume e as páginas abertas recarregam — evita
  assets stale rodando junto do shell novo.
- O pré-cache usa os assets com hash de conteúdo (`_app/immutable/*`), então
  cada build invalida o cache antigo automaticamente. O `navigateFallback: '/'`
  garante que deep links funcionem offline.
- Risco conhecido: páginas abertas durante o update podem misturar shell novo
  com chunk antigo (race do `autoUpdate`). Se isso aparecer em produção,
  migrar para `prompt` (banner "atualização disponível") no futuro.
