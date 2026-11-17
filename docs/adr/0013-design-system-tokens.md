# ADR 0013 — Tokens de design e componentes em lib/ui

**Status:** aceito (Checkpoint 12)
**Data:** 2026-11-16
**Relaciona:** docs/plan/12-design.md, docs/design/tokens.md

## Contexto

Até o CP11 cada tela Svelte definia seus próprios botões, inputs, cards, cores e
espaçamentos dentro de `<style>` escopado. O resultado é *drift*: o mesmo botão
primário tem quatro alturas diferentes, a cor de erro é `#b91c1c` em um módulo e
`#dc2626` em outro, e uma mudança de marca exigiria editar uma dezena de arquivos.
A UI é 100% smartphone-first e não há abstração que permita crescentes
variações de dispositivo sem duplicar markup.

## Decisão

1. **Tokens em CSS custom properties** em `frontend/src/lib/styles/tokens.css`,
   com espelho tipado em `tokens.ts`: cores semânticas, escala tipográfica fixa,
   espaçamento base 4px, raio, sombra, motion e breakpoints.
2. **Biblioteca em `frontend/src/lib/ui/`** com API única por família (ação,
   formulário, feedback, estrutura, overlay, dados, navegação). Telas em
   `src/routes/` consomem componentes; não criam botão/input/card "à mão".
3. **Lint de UI** (`npm run lint:ui`): detector Node + regra ESLint
   `ui/no-ad-hoc-style` proíbe cor/espaçamento/raio literais em `routes/` e
   `lib/ui/`, com escape hatch documentado (`ui-lint-ignore` / `ui-lint-disable`).
   Utilitários compartilhados ficam em `lib/styles/utilities.css` e a API
   pública da biblioteca é o barrel `lib/ui/index.ts` (alias `#ui`).
4. **Página de referência** `/dev/ui` protegida por `OPENJOB_DEV_UI=1`,
   fora do bundle de produção quando a flag não é definida.

Tokens preparam dark mode (troca-se apenas o bloco `:root`) sem implementá-lo nesta CP.

## Alternativas rejeitadas

- **Adotar biblioteca pronta (Shoelace/Flowbite/Tailwind UI)** — aumenta a
  superfície de dependências e conflita com a PWA leve do CP2 (ADR 0003). Os
  componentes necessários são poucos e o domínio é estável.
- **Manter estilos escopados por tela e apenas "padronizar à mão"** — não resolve
  o drift: não há ponto único de mudança nem verificação automática.
- **CSS-in-JS com tema via JS** — adicionaria runtime e hidratação complexa a uma
  SPA cujo diferencial é simplicidade e embed do binário Go (CP11).

Ícones: usamos `@lucide/svelte` (sucessor mantido do `lucide-svelte` citado no
plano), centralizado no componente `Icon.svelte`, com tamanhos 16/20/24 e
`aria-hidden` quando decorativo.

## Consequências

- Mudança visual de uma tela passa a ser, na maioria dos casos, mudança de token
  ou de componente base — com impacto previsível e revisável por snapshot.
- Todo novo componente de tela precisa passar pelo lint e pela auditoria de a11y.
- O escape hatch existe para exceções legítimas, mas precisa de justificativa no PR:
  o objetivo é tornar o drift visível, não impedir trabalho.
