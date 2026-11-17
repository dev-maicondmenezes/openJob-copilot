# Navegação e responsividade (CP12)

Ver ADRs 0014 (breakpoints), 0015 (Dialog) e 0016 (trade-off do tablet).

## Estrutura visual do shell

O shell tem **quatro áreas**: sidebar, header, body (página de conteúdo) e
footer opcional. O desktop é composto por três elementos visuais separados:

1. **Sidebar** fixa à esquerda (identidade, navegação e Config no rodapé);
2. **Header** fixo no topo da área à direita da sidebar, ocupando toda a largura
   disponível, com o título `openJob`, a tagline "Seu assistente de busca de
   emprego, rodando local.", a busca e os ícones de notificações/atalhos;
3. **Página de conteúdo** (`.content`) — um cartão branco com raio/sombra que
   "flutua" sobre o fundo cinza (`--color-surface-soft`), com espaçamento do
   header e das bordas. Contém todo o conteúdo das rotas; dialogs, sheets,
   modais, drawers e toasts são overlays fixos renderizados fora dele.
4. **Footer** (`.app-footer`) — 4ª área do shell, renderizada pelo layout **após**
   o `<main>` quando uma página registra conteúdo via `PageFooter` (store
   `pageFooter`). No desktop, a barra de ações/paginação de `/jobs` usa este
   slot: mesma largura da página, cantos levemente arredondados e espaçamento
   médio, sem sobrepor o conteúdo. Em phone/tablet o footer não é usado: a
   navegação é por rolagem contínua (ver abaixo).

**Paginação × scroll contínuo:** apenas desktop usa paginação. Em phone/tablet,
listas carregam mais itens ao rolar (`DataView.onreachEnd`), evitando controles
de página em telas touch; a contagem simples fica no cabeçalho da lista.

No phone/tablet o header é sticky e a bottom nav é fixa; a página de conteúdo
mantém o mesmo padrão de cartão flutuante com margens menores.

## Três layouts, uma hierarquia

A hierarquia da sidebar desktop é (correções de UI/UX):

1. Início (`/`)
2. Vagas (`/jobs`)
3. Candidaturas (`/applications`)
4. Currículo (`/curriculum`)
5. Tarefas (`/tasks`, grupo secundário)
6. Configurações (`/settings`, grupo do rodapé)

O shell é preso ao viewport (`height: 100dvh`): header no topo, página no meio
e footer na base permanecem visíveis sem rolar a página; a lista de `/jobs`
tem scroll interno. O footer só aparece quando uma página o registra.

"Notificações" saiu da sidebar (o sino no cabeçalho continua disponível) e
"Atalhos" foi movido para o canto superior direito do cabeçalho. A bottom nav
mantém os 5 destinos originais, incluindo Config. Nenhuma seção nova foi criada.

| Dispositivo | Largura | Navegação | Conteúdo | Detalhe | Filtros |
|---|---|---|---|---|---|
| Smartphone | `< 768` | Bottom nav (inalterada) | 1 coluna, full-width | rota dedicada | Sheet |
| Tablet | `768–1023` | Bottom nav (touch targets ≥ 56px) | 1–2 colunas, `max-width: 840px` | rota dedicada | Sheet |
| Desktop | `≥ 1024` | Sidebar (rail opcional) | 2–3 colunas, `max-width: 1200px` | Drawer | inline |

## Drawer vs rota (regra do "voltar")

- **URL é a fonte da verdade.** `/jobs/123` sempre renderiza a página cheia,
  inclusive no desktop, vinda de deep link ou de "Abrir página completa" no drawer.
- No desktop, clicar em um item da lista abre um `Drawer` sobre a lista: `Escape`
  e clique fora fecham e **devolvem o foco ao disparador** (ver `lib/ui/focus.ts`).
  A URL da lista não é alterada pelo drawer — ele é uma camada de contexto.
- No phone/tablet, clicar em um item navega para `/jobs/[id]`: o botão "Voltar
  para vagas" e o histórico do navegador são o mecanismo de retorno.
- A mesma regra de foco vale para `Dialog` (Sheet/Modal): gravamos o elemento
  ativo no `mount`, fazemos focus trap e restauramos no `destroy`.

## Sidebar rail

- Estado persistido em `localStorage` (`openjob:sidebar-collapsed`).
- Alternar por `[` ou pelo botão de recolher no cabeçalho da sidebar.
- O conteúdo acompanha o espaço (`.app` ganha `margin-left` de 240px ou 72px),
  sem alterar `max-width` nem a hierarquia visual.
- `aria-current="page"` identifica o item ativo na sidebar e na bottom nav.

## Atalhos (desktop)

`g j`, `g a`, `g s`, `g n`, `g t` navegam; `?` abre ajuda; `/` foca a busca;
`[` alterna o rail; `Escape` fecha overlays. Implementados em
`frontend/src/lib/shortcuts.ts`, instalados uma vez pelo layout. Não disparam
com foco em `input`, `textarea`, `select` ou `contenteditable`.

## Notas de implementação nesta CP

- O roteamento do app é SPA (`ssr = false`), então o estado inicial de
  responsividade vem de `matchMedia` direto — evita um frame com a navegação
  errada durante a hidratação.
- `DataView` decide cards/tabela por breakpoint e renderiza **as mesmas colunas**
  nos dois caminhos; `Dialog` decide Sheet/Modal pelo mesmo critério.
- Nenhuma tela renderiza elementos "fantasma" (tabela no mobile, bottom nav no
  desktop): a decisão é feita por `viewport.isDesktop` ou por CSS de breakpoint.
