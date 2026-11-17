# ADR 0017 — Paraglide JS para i18n compile-time

**Status:** aceito (Checkpoint 13)
**Data:** 2026-11-17
**Relaciona:** docs/plan/13-dark-mode-i18n.md, frontend/project.inlang/settings.json

## Contexto

A UI do openJob é embutida no binário Go do CP11 (`backend/web/dist`) e precisa
de tradução em três locales sem aumentar o runtime entregue ao navegador.
Strings viviam hardcoded em `src/routes/**` e `src/lib/**` (a biblioteca
`lib/ui/` já recebe texto por props). Precisávamos de: chaves tipadas, detecção
de chaves faltando no CI, tree-shaking e troca de idioma sem reload.

## Decisão

Usar **`@inlang/paraglide-js`** (v3) com projeto inlang em
`frontend/project.inlang/` e catálogos em
`frontend/messages/{pt-BR,en-US,es-419}.json`.

1. O plugin Vite compila cada mensagem em uma função ESM tipada
   (`frontend/src/paraglide/`), com `en-US` como locale canônico estrutural.
2. Todo acesso passa pelo helper único `src/lib/i18n/t.ts`. É proibido importar
   `@inlang/paraglide-js` ou `src/paraglide/*` fora de `src/lib/i18n/`
   (regra ESLint `no-restricted-imports`).
3. A reatividade é feita pelo estado `$state` em `locale.svelte.ts`: `t()` lê
   `localeState.current` no chamador, e chama a função gerada com
   `{ locale }` explícito. `setLocale(locale, { reload: false })` mantém o
   runtime do paraglide sincronizado sem document navigation.
4. `npm run i18n:check` falha se houver chave faltando/extra em qualquer
   locale ou placeholders divergentes em relação ao canônico.

## Alternativas rejeitadas

- **`svelte-i18n`**: runtime, dicionário carregado em execução, sem type-safety
  de chaves e bundle maior — custo relevante por ser embutido no binário.
- **Dicionário TS artesanal**: sem validação de completude nem pluralização ICU,
  e exigiria manter tipos à mão.
- **`Intl` direto com mapas locais**: resolve formatação, não resolve catálogo.

## Consequências

- Erro de digitação em chave é erro de compilação (quando a chave é literal) e
  falha no `i18n:check` (quando vem de `code`/`i18n_key` do backend).
- Mensagens com parâmetros usam ICU (`{name}`, `{count, plural, ...}`);
  formatação de datas/números/moedas fica em `lib/i18n/format.ts`.
- O diretório gerado é ignorado pelo Git; `npm run prepare` / `check` / build
  executam a compilação antes de qualquer consumo.
