# Checkpoint 2 — SPA + PWA (2–3 dias)

> Origem: spec §8, Checkpoint 2. Pré-requisito: [01-fundacao.md](01-fundacao.md).

## Objetivo

SvelteKit com `adapter-static` (modo SPA), build embutido no binário Go via `//go:embed`, navegável e instalável como PWA no celular.

## Pré-requisitos

- Checkpoint 1 concluído (binário Go servindo web/ via embed).

## Tarefas

1. **Projeto SvelteKit** em `frontend/`:
   - `svelte.config.js` com `adapter-static`, `fallback: 'index.html'`, `strict: false` (SPA puro, sem pré-render).
   - `vite.config.ts` com plugin PWA (`@vite-pwa/sveltekit`).
2. **Rotas base** (stubs visuais, sem API ainda):
   - `/` (dashboard placeholder), `/settings`, `/curriculum`, `/jobs`, `/applications`.
   - Navegação inferior (bottom nav) mobile-first.
3. **Integração de build**:
   - `pnpm build` copia output para `backend/web/dist/`.
   - Binário Go serve a SPA com fallback para `index.html` em rotas desconhecidas (deep links funcionam).
   - `make build` (ou script equivalente) orquestrando: build SPA → build Go.
4. **PWA**:
   - `manifest.webmanifest` (nome "openJob", ícones, `display: standalone`, theme color).
   - Service worker com cache offline básico (app shell).
5. **Mobile-first / iOS**:
   - `viewport-fit=cover` + `env(safe-area-inset-*)` no layout.
   - Teste de toque, scroll e áreas seguras no celular real.

## Critério de verificação

- No celular: navegar entre as 5 rotas, **recarregar em `/jobs`** (deep link) sem 404.
- **Instalar a PWA na tela inicial** do celular e abrir offline (app shell carrega).
- Mesma origem: SPA e API em `http://openjob.local:8080` — **sem CORS** (spec §3.5).

## Registrar

- ADR: `adapter-static` + fallback SPA (por que não SSR — local-first, sem necessidade de servidor de render).
- Nota sobre versionamento do cache do service worker (evitar stale assets no futuro).

## Fora de escopo

- Consumo real de API, dados, autenticação.
