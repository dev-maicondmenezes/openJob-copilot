# ADR 0019 — Dark mode via `[data-theme]` e CSS custom properties

**Status:** aceito (Checkpoint 13)
**Data:** 2026-11-17
**Relaciona:** ADR 0013, docs/plan/13-dark-mode-i18n.md

## Contexto

O CP12 criou tokens semânticos em `tokens.css` e uma biblioteca `lib/ui` que
consome apenas esses tokens. Faltava o tema escuro sem introduzir Tailwind
`dark:`, CSS-in-JS ou duplicação de componentes.

## Decisão

1. **Estado tri-state**: `light | dark | system` (default `system`), persistido
   em `localStorage.openjob.theme`, por dispositivo. Nenhum campo no backend.
2. **Aplicação**: atributo `data-theme="light|dark"` no `<html>`, com bloco
   `[data-theme='dark']` em `tokens.css` redefinindo **somente os valores** dos
   tokens semânticos. Nomes nunca mudam; componentes não são alterados.
3. **Anti-FOUC**: script inline em `src/app.html` lê `localStorage` e aplica
   `data-theme`, `color-scheme` e `<html lang>` antes de `%sveltekit.head%`.
4. **Modo system**: `matchMedia('(prefers-color-scheme: dark)')` com listener
   `change` em tempo real (`theme.svelte.ts`), sem reload.
5. **Seletor**: `/settings/appearance` com `SegmentedControl`
   Sol/Lua/Monitor; troca instantânea e persiste localmente.
6. **Contraste**: alvo WCAG AA (≥ 4.5:1 texto, ≥ 3:1 interativos), validado
   por axe nos dois temas. No escuro sombras viram `none` e bordas assumem a
   separação visual; overlay `rgba(0,0,0,.8)`.

## Alternativas rejeitadas

- **Utility `dark:` do Tailwind**: exigiria duplicar classes em cada componente
  e quebraria a regra de token único do CP12.
- **CSS-in-JS / context provider**: runtime extra e hidratação complexa em uma
  SPA embutida no binário Go.
- **Sync do tema via backend**: mobile e desktop podem querer preferências
  diferentes; sincronizar imporia a mesma escolha em todos os dispositivos.

## Consequências

- Todo valor hardcoded de cor em `lib/ui` é um bug (auditado manualmente;
  MarkdownView e aliases legados foram migrados para tokens semânticos).
- `prefers-reduced-motion` já zera transições em `app.css`, cobrindo a troca
  de tema.
- Sync cross-device de tema fica em backlog.
