# Biblioteca de design system do openJob

A biblioteca vive em `frontend/src/lib/ui/` e tem um único ponto de entrada:
`frontend/src/lib/ui/index.ts` (alias `#ui`). Tokens e utilitários ficam em
`frontend/src/lib/styles/`.

```
src/lib/
├── styles/
│   ├── tokens.css        # fonte única: cor, tipo, espaço, raio, sombra, motion
│   ├── tokens.ts         # espelho tipado (BREAKPOINTS, SPACING, LAYOUT…)
│   └── utilities.css     # classes utilitárias (.hint, .link, .grid, .chips…)
└── ui/
    ├── index.ts                 # barrel / API pública
    ├── Button.svelte …          # ações, formulário, feedback, estrutura
    ├── Dialog/Sheet/Modal/...   # overlays com foco gerenciado
    ├── DataView/Pagination      # dados, tabela↔cards e paginação
    ├── Sidebar/BottomNav/...    # navegação do shell
    ├── PageFooter/footer.svelte.ts  # 4ª área do layout
    ├── media.svelte.ts          # viewport reativo (matchMedia)
    ├── nav.ts / focus.ts / data.ts
    └── sidebar.svelte.ts / footer.svelte.ts
```

## Uso

```svelte
<script lang="ts">
	import { Button, Card, Field, Input, Badge, DataView } from '#ui';
</script>
```

Também é possível importar por arquivo (`#lib/ui/Button.svelte`) — o barrel
existe para padronizar a API pública e facilitar descoberta/extensão.

## Convenções (revisão de design)

1. **Um componente por responsabilidade.** Se uma tela precisa de uma variação,
   adicione a variante ao componente base (`compact`, `iconOnly`, `selectable`…)
   — nunca crie um botão/card paralelo.
2. **Tokens em todo lugar.** Nenhuma cor, espaçamento, raio, tipografia ou
   backdrop literal. O lint cobre `src/routes/**` **e** `src/lib/ui/**`.
3. **Nomes de propriedades consistentes:** `label` (acessível/tooltip),
   `variant` (tom), `size` (densidade), `disabled`, `loading`, `class`.
4. **Acessibilidade por padrão:** `aria-label` obrigatório em ícones solo
   (tipado), `role`/`aria-*` em overlays, focus trap e retorno de foco.
5. **Sem duplicação de estilo:** utilitários comuns (`.hint`, `.muted`, `.link`,
   `.form-actions`, `.chips`, `.meta-row`, `.truncate`) ficam em
   `styles/utilities.css`; estados específicos pertencem ao componente.

## Extensão

- Novo token: adicione em `tokens.css` **e** `tokens.ts`, documente em
  `docs/design/tokens.md`.
- Novo componente: crie em `lib/ui/`, exporte no `index.ts`, use apenas tokens e
  adicione ao `/dev/ui` (referência visual).
- Nova variante: prefira prop no componente existente; se criou um padrão novo
  de interação, documente aqui e em `navigation.md` quando afetar o shell.

## Verificação

```bash
npm run check      # tipos (svelte-check)
npm run lint:ui    # tokens + ESLint em routes/ e lib/ui/
npm run dev:ui     # /dev/ui com OPENJOB_DEV_UI=1
npm run test:e2e / audit:a11y / test:visual
```

O escape hatch `ui-lint-ignore` / `ui-lint-disable` continua disponível para
exceções justificadas (ex.: cálculo de largura), mas revisões devem preferir
novo token ou variante do componente.
