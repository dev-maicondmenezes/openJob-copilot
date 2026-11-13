// SPA pura: sem SSR e sem pré-render — o adapter-static gera apenas o fallback
// `index.html` e o cliente hidrata as rotas (ver docs/adr/0003-adapter-static-spa.md).
export const ssr = false;
export const prerender = false;
