# Checkpoint 16 — Reformulação da UX para feed vertical imersivo (estilo TikTok/Reels) (6–8 dias)

> Origem: pedido de evolução da UX após o CP12 (design system e responsividade). Pré-requisito: **CP12 e CP13 concluídos** (tokens, componentes base, i18n, dark mode). Este checkpoint **não substitui** o CP12 — ele **reaproveita** a fundação de design system e adapta a arquitetura de informação para um padrão de consumo mais imersivo e adequado ao uso mobile-first do openJob.

## 1. Objetivo

Reformular a experiência principal do openJob para um **feed vertical de tela cheia**, no qual cada vaga ocupa 100% do viewport e o usuário navega por gesto de scroll (ou swipe), com:

- **Uma vaga por vez** em tela cheia, com informações essenciais visíveis de imediato (título, empresa, fonte, fit score, modalidade, salário).
- **Botão discreto de atualização** (“Buscar agora”) integrado ao feed, sem sair da tela.
- **Descrição renderizada em Markdown** de forma segura, com um **teaser** curto e um botão **“Saiba mais…”** que abre uma **tela dedicada com a descrição completa e scroll vertical**.
- **Navegação por scroll-snap nativo** (sem bibliotecas de carrossel pesadas), com suporte a teclado, leitores de tela e gestos.
- **Reaproveitamento de todas as funcionalidades existentes** (filtros do CP6, scoring do CP10, pipeline do CP7, notificações) em uma camada de apresentação mais focada.

**Princípio:** o feed é a **nova home** do app. As telas de lista tradicionais (`/jobs` como lista) deixam de ser o ponto de entrada principal, mas **continuam existindo** como modo de “gerenciamento” e para deep links. O usuário pode alternar entre “Feed” e “Lista” sem perder contexto.

## 2. Fundamentação — o que a pesquisa sobre feeds imersivos revela

O padrão de feed vertical de tela cheia é bem documentado em produtos como TikTok, Reels e Shorts, e em implementações de “media feed” para web. Os pontos críticos que orientam este CP:

| Dimensão | Padrão observado | Implicação para o openJob |
|---|---|---|
| **Estrutura** | Lista reciclável com *snap* de tela cheia; apenas o item visível é “ativo” | Um card por viewport, com `scroll-snap-type: y mandatory` nativo. Sem renderizar todos os cards de uma vez (virtualização ou janela de pré-carregamento). |
| **Conteúdo** | O item em si é uma “página” curta; o conteúdo longo vai para um **“read-more sheet”** ou rota dedicada | Descrição completa em rota separada (`/jobs/[id]`), com teaser no card do feed. |
| **Navegação** | Scroll/swipe é a interação primária; setas e botões são alternativas de acessibilidade | Scroll nativo + botões “anterior/próximo” no desktop + teclas `↑`/`↓`; gesto de swipe no mobile. |
| **Acessibilidade** | Itens fora da tela devem ser `inert`; apenas o item ativo é focável; teclado e leitor de tela precisam de suporte explícito | `inert` nos cards não visíveis; `role="feed"` no container; navegação por teclado; anúncio de “item X de Y” para leitores de tela. |
| **Performance** | Pré-carregar alguns itens à frente; virtualizar para limitar o DOM; evitar renderizar todos os itens de uma lista infinita | Feed carrega em “páginas” (cursor/keyset); mantém no DOM apenas o item atual + vizinhos (janela deslizante). |
| **Atualização** | Pull-to-refresh ou botão explícito no topo do feed; feedback visual de progresso | Botão “Buscar agora” no topo do feed, com estado de loading e contagem de novas vagas. |
| **Compartilhamento** | URL sincronizada com o item em view; deep link abre o item correto | `/jobs/[id]` continua funcionando; o feed pode sincronizar o item atual com `?job=<id>` (opcional, mas recomendado para deep link). |

## 3. Arquitetura de informação reformulada

### 3.1 Nova hierarquia de navegação

| Antes (CP12) | Depois (CP16) |
|---|---|
| Home (`/`) com dashboard | **Feed (`/jobs`)** como home principal. O dashboard (CP15) permanece acessível em `/dashboard`, mas não é mais a primeira tela. |
| `/jobs` como lista de cards | `/jobs` passa a ser o **feed imersivo**; a **lista tradicional** vira uma visualização secundária em `/jobs/list` (ou um toggle “Feed / Lista” no topo). |
| Detalhe da vaga em `/jobs/[id]` (página cheia) | `/jobs/[id]` continua existindo como **página de detalhe completa** (com scroll vertical), aberta pelo botão “Saiba mais…”. No feed, o item é um resumo. |
| Sidebar/bottom nav com 5 itens | A navegação primária muda: **Feed** vira o item central; os demais itens (Candidaturas, Currículo, Configurações, Notificações) permanecem, mas o feed é o “lar”. No mobile, a bottom nav pode ter o feed como primeiro item. |

### 3.2 Rotas

| Rota | Função |
|---|---|
| `/jobs` (ou `/`) | **Feed vertical de vagas** (tela cheia, um item por viewport) |
| `/jobs/list` | Lista tradicional (cards/tabela do CP12), para quem prefere visão geral |
| `/jobs/[id]` | Detalhe completo da vaga (descrição em Markdown, scroll vertical) |
| `/dashboard` | Dashboard do CP15 (movido para rota própria) |
| `/applications`, `/resume`, `/settings/*`, `/notifications`, `/tasks` | Inalterados (mas com navegação adaptada ao novo fluxo) |

> **Deep link:** `/jobs/123` abre a página de detalhe completa. O feed pode sincronizar o item atual com `?job=123` para permitir que o usuário compartilhe ou recarregue na mesma posição.

## 4. Componentes e padrões de UI

### 4.1 `JobFeed` (container do feed)

- Container com `height: 100dvh` (dynamic viewport height) e `overflow-y: auto`; cada item tem `scroll-snap-align: start` e `height: 100dvh`.
- `scroll-snap-type: y mandatory` no container (nativo, sem JS para o snap).
- **Janela deslizante:** renderiza no máximo 3 itens (anterior, atual, próximo) para limitar o DOM; ao mudar o índice ativo, descarta os distantes e carrega mais da API se necessário (cursor/keyset).
- **Carregamento infinito:** quando o usuário se aproxima do fim da página carregada, dispara `fetchNextPage` (via `IntersectionObserver` ou `scroll` com debounce).
- **Estado de carregamento:** *skeleton* de tela cheia (reaproveitando o placeholder do CP15) enquanto a primeira página não chega.
- **Estado vazio:** *EmptyState* com CTA “Buscar agora” (se nenhuma vaga ainda) ou “Ajustar filtros” (se filtros muito restritivos).

### 4.2 `JobFeedCard` (item do feed)

Cada card ocupa 100% do viewport e exibe, em camadas:

1. **Cabeçalho do card (topo):**
   - Título da vaga (grande, legível).
   - Empresa · Localização/Região (se disponível).
   - Badges: `fit score` (ex.: “92% fit”), modalidade (`remoto`, `híbrido`, `presencial`), tipo de contrato (`CLT`, `PJ`, `freelance`), faixa salarial (se disponível).
   - Fonte (ex.: “Gupy”, “Get on Board”) com link para a vaga original (atribuição obrigatória quando `attribution_required`).

2. **Corpo do card (meio):**
   - **Teaser da descrição:** primeiros ~3–4 parágrafos ou ~200 caracteres, renderizados em **Markdown seguro** (ver §5), com truncamento e gradiente de fade.
   - Botão **“Saiba mais…”** (discreto, mas visível) que navega para `/jobs/[id]`.

3. **Rodapé do card (base):**
   - Ações rápidas: **Salvar/Candidatar** (se aplicável no contexto do CP7), **Compartilhar** (link para `/jobs/[id]`), **Fechar/Descartar** (para o feed não mostrar novamente).
   - Indicador de posição no feed (ex.: “3 de 120”) — apenas no desktop ou para leitores de tela.

> **Não** incluir formulários ou campos de edição no card. O feed é para **consumo e triagem**, não para configuração. Ações que exigem contexto (ex.: editar candidatura) ficam no detalhe ou nas telas de gerenciamento.

### 4.3 Botão de atualização (“Buscar agora”)

- Posicionado no **topo do feed**, como um botão flutuante discreto (ícone de “refresh” + texto curto) ou integrado a uma barra superior mínima.
- Comportamento:
  - Clique → dispara `POST /api/jobs/discover` (CP6) e mostra estado de loading no próprio botão.
  - Durante a execução, o feed não é bloqueado; novas vagas são **inseridas no topo** (ou no fim, conforme a ordenação) e o usuário é notificado com um *toast*: “12 novas vagas adicionadas ao feed.”
  - Se nenhuma vaga nova for encontrada, *toast* informativo: “Nenhuma vaga nova no momento.”
- **Pull-to-refresh (mobile):** opcional, como alternativa ao botão; implementado com `touch` events e threshold de ~80px, mas o botão continua sendo o mecanismo primário para desktop e para acessibilidade.

### 4.4 Página de detalhe (`/jobs/[id]`)

- Rota cheia com scroll vertical normal (sem snap).
- Renderiza a **descrição completa em Markdown** (ver §5), com tipografia confortável para leitura longa.
- Mantém o cabeçalho com título, empresa, badges e ações (candidatar, salvar, compartilhar).
- Botão “Voltar ao feed” no topo (ou botão voltar do navegador).
- **Link para a vaga original** (fonte) sempre visível — atribuição obrigatória.

## 5. Renderização de descrição em Markdown com segurança

- **Pipeline de renderização:**
  1. O backend (ou o adapter de cada fonte) **normaliza** a descrição para Markdown **ou** HTML sanitizado. O ideal é Markdown, mas se a fonte fornece HTML, ele deve ser sanitizado no servidor antes de ser armazenado (remover `script`, `on*`, `javascript:` etc.).
  2. No cliente, usar uma biblioteca de Markdown **com sanitização embutida** (ex.: `@humanspeak/svelte-markdown`, que aplica políticas de URL e atributos antes de renderizar).
  3. **Nunca** usar `{@html}` com Markdown não sanitizado. Se for inevitável usar `{@html}`, a sanitização deve ocorrer no servidor e o conteúdo deve ser tratado como confiável apenas após a sanitização.
- **Markdown suportado:** headings, listas, negrito/itálico, links (com `rel="noopener noreferrer"`), blocos de código (se relevante), quebras de linha. **Não** renderizar HTML embutido não sanitizado.
- **Performance:** renderizar o Markdown **no cliente** apenas para o item ativo do feed (teaser) e para a página de detalhe. O feed não precisa renderizar Markdown completo de todos os itens.

## 6. Navegação e acessibilidade

### 6.1 Navegação por dispositivo

| Dispositivo | Navegação no feed |
|---|---|
| **Smartphone** | Scroll vertical nativo (snap). Swipe para cima/baixo. Botão flutuante “Buscar agora”. Bottom nav permanece, mas o feed ocupa a área acima dela. |
| **Tablet** | Mesmo padrão do smartphone (snap vertical), com área de toque maior nos botões e tipografia ligeiramente maior. |
| **Desktop** | Scroll com roda do mouse (snap), **setas ↑/↓** para navegar entre itens, **botões “Anterior/Próximo”** visíveis nas laterais ou no topo. Sidebar permanece (CP12), mas o feed ocupa a área principal. |

### 6.2 Acessibilidade (requisitos não negociáveis)

- **`role="feed"`** no container; cada item com `role="article"` e `aria-label` descritivo.
- **Apenas o item ativo é focável:** usar `inert` nos itens fora da tela para que o `Tab` não alcance controles invisíveis.
- **Anúncio de posição:** um `aria-live="polite"` oculto anuncia “Item 3 de 120” quando o item ativo muda.
- **Teclado:** `↑`/`↓` navegam; `Enter` no botão “Saiba mais…” abre o detalhe; `Esc` fecha o detalhe (se aberto como overlay) ou volta ao feed.
- **Leitores de tela:** o feed não deve “prender” o foco; o usuário pode navegar para fora do feed usando a navegação normal.
- **Movimento reduzido:** respeitar `prefers-reduced-motion`; se ativado, o snap pode ser instantâneo (sem animação de scroll) e o botão “Buscar agora” não pulsa.
- **Contraste e tamanho de toque:** os badges e botões devem atender ao WCAG AA do CP12; área de toque mínima de 44×44px.

## 7. Integração com funcionalidades existentes

| Funcionalidade | Como se integra ao feed |
|---|---|
| **Descoberta (CP6/CP14)** | O feed consome a mesma API de vagas; o botão “Buscar agora” dispara discovery e atualiza o feed. |
| **Filtros (CP6)** | Os filtros ativos (keywords, região, modalidade) são aplicados no feed; um botão “Filtros” no topo abre o painel de filtros (Sheet no mobile, inline no desktop). |
| **Scoring (CP10)** | O `fit score` é exibido no card; vagas sem score ainda aparecem (com placeholder), mas o feed prioriza vagas com score. |
| **Candidaturas (CP7)** | Ação “Candidatar” no card cria uma entrada em `applications` com status `pending`; a ação “Salvar” é um estado separado. |
| **Notificações (CP10)** | Notificações de novas vagas ou de conclusão de scoring aparecem no sino; o feed pode ser atualizado manualmente. |
| **Tarefas (CP10)** | O botão “Buscar agora” pode mostrar o progresso da tarefa de scoring associada (se houver). |
| **Dashboard (CP15)** | Movido para `/dashboard`; acessível pela navegação, mas não é mais a home. |

## 8. Critérios de verificação

- **Home = feed:** abrir o app leva a `/jobs` com um card ocupando 100% do viewport; a barra de navegação (bottom nav ou sidebar) permanece visível.
- **Scroll-snap:** rolar para cima/baixo encaixa um card por vez, sem “meio card” visível; funciona em 375×667 (smartphone) e 1440×900 (desktop).
- **“Saiba mais…”:** o botão abre `/jobs/[id]` com a descrição completa em Markdown, rolável verticalmente, e um botão “Voltar ao feed”.
- **Atualização:** clicar em “Buscar agora” dispara o discovery (verificável via log/API), mostra loading no botão e, ao concluir, insere novas vagas no feed (ou informa que não há novas).
- **Acessibilidade:**
  - Apenas o card ativo é alcançável por `Tab`.
  - `↑`/`↓` navegam entre cards no desktop.
  - Leitor de tela anuncia a posição (“Item 3 de 120”).
  - `prefers-reduced-motion` desativa animações de snap.
- **Performance:**
  - O DOM contém no máximo ~3–5 cards simultaneamente, independentemente do tamanho da lista.
  - O feed carrega a primeira página em < 1s em rede local.
- **Markdown seguro:**
  - Uma descrição com `<script>` ou `javascript:` não executa nada (teste com payload malicioso).
  - A renderização do teaser e da página de detalhe usa a mesma biblioteca sanitizada.
- **Deep link:** `/jobs/123` abre a página de detalhe mesmo que o usuário nunca tenha passado pelo feed.
- **Navegação:** a lista tradicional (`/jobs/list`) continua funcionando e é acessível por um toggle no feed ou pela navegação.
- **Não regressão:** todos os fluxos dos CPs anteriores (candidaturas, configurações, notificações, tarefas) continuam acessíveis e funcionais.

## 9. Registrar

- **ADR `0028-feed-vertical-imersivo.md`** — adotar o padrão de feed vertical de tela cheia como home do app, com scroll-snap nativo e janela deslizante de itens. Justificativa: reduz fadiga de navegação em listas longas, aumenta foco por vaga e se alinha ao uso mobile-first. Alternativa rejeitada: manter apenas a lista tradicional (não resolve o problema de sobrecarga de informação).
- **ADR `0029-read-more-sheet-e-rota-detalhe.md`** — descrição completa em rota dedicada (`/jobs/[id]`), com teaser no feed. Justificativa: o card do feed deve ser escaneável; o detalhe é para leitura profunda. Alternativa rejeitada: expandir o card no próprio feed (quebra o snap e a performance).
- **ADR `0030-markdown-sanitizado.md`** — usar Markdown sanitizado no cliente com biblioteca que aplica políticas de URL/atributos antes da renderização. Justificativa: segurança contra XSS sem sacrificar a formatação. Alternativa rejeitada: `{@html}` com HTML não sanitizado.
- **ADR `0031-atualizacao-do-feed.md`** — botão “Buscar agora” como mecanismo primário de atualização, com pull-to-refresh opcional no mobile. Justificativa: o botão é acessível (teclado, leitor de tela) e explícito; o gesto é um enhancement.
- **Testes:**
  - Snapshot visual do feed em 375×667, 768×1024 e 1440×900.
  - Navegação por teclado e `inert` (teste com `@testing-library` ou Playwright).
  - Renderização de Markdown com payloads maliciosos (XSS).
  - Performance: contagem de nós no DOM após rolar 20 itens (deve permanecer constante).
  - Integração: “Buscar agora” dispara discovery e atualiza o feed.
- **Riscos anotados:**
  - **`scroll-snap` e leitores de tela:** o snap nativo pode causar problemas de navegação com VoiceOver em algumas combinações de `:root` + imagens grandes. Mitigação: aplicar o snap a um container interno (não ao `html`), testar com NVDA/VoiceOver e oferecer um modo “lista” como alternativa.
  - **Virtualização e acessibilidade:** ao reciclar nós, o foco pode ser perdido. Mitigação: manter o item ativo sempre montado e gerenciar o foco explicitamente ao navegar.
  - **Markdown de fontes não confiáveis:** mesmo sanitizado, o Markdown pode conter links para domínios maliciosos. Mitigação: exibir um aviso ao sair para links externos ou usar `rel="noopener noreferrer"` e um interstitial (backlog).
  - **Performance em dispositivos antigos:** o snap e a janela deslizante podem ser custosos em smartphones antigos. Mitigação: testar em dispositivo real e degradar para lista se necessário.

## 10. Fora de escopo (deliberadamente adiado)

- **Feed personalizado por algoritmo de recomendação** — o feed segue a ordem de descoberta/scoring existente; personalização pesada fica para backlog.
- **Autoplay de vídeos ou mídia** — as vagas são textuais; não há mídia rica.
- **Comentários ou interação social no feed** — fora do escopo do copiloto.
- **Edição inline no feed** — ações de edição continuam nas telas de gerenciamento.
- **Substituição completa da lista tradicional** — a lista (`/jobs/list`) permanece como modo alternativo.

---

## 11. Status de conclusão

**Status (2026-11-22): concluído (Fase 1 do MVP).**
Todos os critérios de verificação e gates foram atendidos com sucesso:
- Feed vertical de tela cheia ativo em `/jobs` e `/` com scroll-snap nativo e janela deslizante.
- Lista tradicional em `/jobs/list` preservada.
- Detalhe em `/jobs/[id]` com Markdown sanitizado e botão de retorno.
- Botão "Buscar agora" com feedback não bloqueante e toasts de status.
- ADRs 0028–0031 registrados.
- Suíte completa verde: tipos (`svelte-check`), i18n (`i18n-check`), lint de UI (`lint-ui`), e2e (`playwright test`), acessibilidade (`axe-core`) e visual (`test:visual`).
