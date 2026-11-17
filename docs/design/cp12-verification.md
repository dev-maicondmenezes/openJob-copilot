# CP12 — verificação e desvios documentados

## Comandos executados (resultado)

| Comando | Resultado |
|---|---|
| `npm run check` | 0 erros, 0 avisos |
| `npm run lint:ui` | sem estilo ad-hoc em `src/routes/` |
| `npm run build` | SPA + PWA geradas e publicadas em `backend/web/dist` |
| `npx playwright test` (81 testes) | 81 passed |
| `go test ./...` (backend) | todos os pacotes ok |

Os 81 testes cobrem: 30 snapshots (10 rotas × 3 viewports), 30 auditorias axe
(10 rotas × 3 viewports) e 17 testes de interação (atalhos, viewport, deep link
e correções de UI/UX).

## Ajustes finais em `/jobs`

- Removido o card "Descoberta Automática de Vagas" e o título da lista
  "Vagas cadastradas"; subtítulo do `PageHeader` reescrito em duas frases sobre
  o mecanismo de busca.
- Card curto de palavras-chave com link (ícone) para `/settings/discovery`.
- Filtros com ícones quadrados no canto direito: "Limpar filtros" (quando
  ativo, com tooltip) antes do ícone de configuração de fontes
  (`/settings/sources`).
- Barra de rodapé (mesma largura da página, cantos levemente arredondados,
  após a lista, sem sobreposição): **Aprovar** antes de **Calcular score** à
  esquerda, **Descartar** à direita antes das setas e contador
  `x..y de total` depois das setas.
- Lista com coluna de checkbox, botão "Marcar todas" (ícone de verificação
  dupla) antes do ícone de fontes, e ações em lote sobre todas as vagas marcadas.
- Filtro de fontes com múltipla escolha; ordenação por uma coluna por vez
  (padrão: score descendente) com ícone de direção nos cabeçalhos/cards.
- Scroll interno na lista (55dvh), não na página.
- Botões "Buscar agora" e "Nova Vaga" alinhados horizontalmente; subtítulo
  reduzido à primeira frase.

## Revisão do design system (biblioteca)

- Barrel `frontend/src/lib/ui/index.ts` (alias `#ui`) como API pública única;
  guia de uso/extensão em `docs/design/library.md`.
- Utilitários duplicados em rotas (`.hint`, `.link`, `.form-actions`, `.chips`,
  `.meta-row`, `.truncate`, `.muted`) centralizados em
  `src/lib/styles/utilities.css`.
- Controles ad-hoc restantes substituídos por componentes: chip de remoção
  (`IconButton`), botão textual (`Button variant="link"`), item do feed
  (`Card onclick`) e campo de busca do header (`Input`).
- Padronização de overlays com token único `--overlay-backdrop` e micro-gap
  `--space-05`; lint de UI estendido para `src/lib/ui/**`.

## Navegação mobile/tablet (sem paginação)

- **Desktop (`≥ lg`):** mantém a guia de paginação no `PageFooter`, com o
  contador `x..y de total` e as ações em lote em `/jobs`.
- **Phone/tablet (`< lg`):** sem paginação — padrão de scroll contínuo, natural
  em telas touch. O `DataView` (e a lista de `/applications`) carrega mais
  itens ao se aproximar do fim da rolagem, com indicador "Carregando mais…".
- `/jobs`: contagem simples (`N vagas`) no cabeçalho da lista; quando há itens
  selecionados, uma barra flutuante de ações em lote aparece acima da bottom nav.
- `/applications`: cards compactos no phone (link/data no detalhe) e 2 colunas
  no tablet; a contagem aparece acima da lista.
- Testes cobrem ausência de paginação em 375×667 e carregamento progressivo
  por rolagem no tablet.

## Ajustes em `/applications` e largura global

- Cards do kanban reestruturados: título/empresa com truncamento, metadados em
  linha própria e ações no cabeçalho — nada vaza para fora do card. Colunas
  com marcador colorido por status, contador, estado vazio e drag and drop.
- Barra de filtros no padrão de `/jobs`: status (múltipla escolha), busca por
  título/empresa e "só com currículo adaptado"; versão mobile em `Dialog`.
- Tokens de largura ampliados para todas as telas (`--content-max-desktop`
  1920px e `--content-max-tablet` 1080px): o cartão de conteúdo aproveita a
  largura disponível após a sidebar, inclusive em desktop maximizado.

## Correções de UI/UX aplicadas (docs/plan/correções_ui-ux.txt)

- **Espaçamento:** cards de descoberta, cards de configuração, itens de lista
  (`DataView` — vagas/fontes/tarefas) e itens do popup de notificações passaram
  a usar tokens maiores (`--space-4`/`--space-6`).
- **`/jobs`:** formulário "Nova Vaga" virou `Dialog` acionado por botão
  secundário com ícone `+` no `PageHeader`; ações dos itens ficaram em uma linha
  com botões compactos e ícones 16px; "Aprovar" apenas emite toast (sem
  redirecionamento e sem recarregar a lista).
- **`/applications`:** cards compactos e clicáveis (abrem detalhe), com drag and
  drop entre colunas do kanban desktop via HTML5 DnD.
- **`/curriculum`:** seletor de vagas com status pendente (`new`/`pending`),
  resumo da vaga escolhida e JD pré-preenchida (editável).
- **`/settings`:** removidos os cards-links de seção — restam apenas as abas.
- **Notificações:** ações "Marcar todas como lidas" e "Limpar lidas" são
  botões-ícone com tooltip, alinhados à direita dos filtros; o popup usa o
  mesmo padrão.
- **Sidebar:** sem "Notificações"; "Config" no separador do rodapé; "Atalhos"
  movido para o canto superior direito do cabeçalho. Na bottom nav, Config
  continua como 5º destino.
- **Mobile:** cabeçalho sticky e bottom nav fixa com ícones 24px e alvos de
  toque de 64px.

## Mapeamento de rotas do plano → rotas reais

O plano foi escrito com nomes de rotas do spec; o código existente usa nomes
ligeiramente diferentes. **Nenhuma rota nova de funcionalidade foi criada** —
apenas a rota de detalhe de vaga `/jobs/[id]`, exigida pelo critério de deep link.

| Rota citada no plano | Rota real (mantida) |
|---|---|
| `/resume` | `/curriculum` |
| `/settings/ai` | `/settings` (card "Provedor de IA (BYOK)") |
| `/settings/telegram` | não existe no frontend (CP7 é backend) |
| `/settings/tasks` | `/settings/notifications` (card "Tarefas de IA (worker)") |
| `/jobs/[id]` | criada como camada de apresentação sobre `GET /api/jobs` |

## Desvios deliberados

- **`/tasks` detail:** o plano prevê drawer no desktop e rota no mobile/tablet.
  Como não existe rota de tarefa e o CP12 proíbe telas novas, usamos `Drawer`
  no desktop e o mesmo `Drawer` (que se comporta como sheet) no mobile/tablet.
- **Filtros em `/jobs`:** o plano cita filtros em Sheet/inline, mas a tela não
  os tinha. Foi adicionado um filtro **client-side** sobre dados já carregados
  (status, fonte e texto) — nenhuma chamada de API ou funcionalidade nova.
- **Busca global `/`:** o campo do topbar navega para `/jobs?q=…` e o filtro da
  lista consome o parâmetro. É um atalho de UI sobre dados existentes.
- **Ícones:** `@lucide/svelte` (mantido) em vez do `lucide-svelte` descontinuado
  citado no plano; centralizados em `lib/ui/Icon.svelte`. Ver ADR 0013.

## Itens de verificação manual pendentes

Estes critérios exigem execução humana e não são cobertos pelo CI:

- **Lighthouse PWA ≥ 90 no mobile** — manifest/service worker preservados
  (vite.config.ts inalterado nesse ponto), mas o score deve ser medido no
  dispositivo alvo.
- **Leitor de tela no drawer desktop** (NVDA/VoiceOver em `/jobs/[id]`): o
  focus trap e retorno de foco são testados por código, mas a experiência com
  leitor de tela precisa de validação manual.
- **Validação com usuários do tablet** — ADR 0016 registra o caminho de
  migração para navigation rail se a bottom nav se mostrar desconfortável.

## Baselines visuais

`frontend/tests/visual/routes.spec.ts-snapshots/*.png` foram gerados com
`--update-snapshots` e devem ser revisados manualmente a cada mudança. O CI
desabilita animações (CSS `data-test-animations="off"` + reduced motion) para
evitar flakiness.
