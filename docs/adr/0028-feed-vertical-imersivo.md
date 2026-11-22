# ADR 0028 — Feed vertical imersivo como home principal do openJob

**Status:** aceito (Checkpoint 16 / Fase 1 MVP)  
**Data:** 2026-11-22  
**Relaciona:** docs/plan/16-infinite-timeline-design.md, docs/plan/release-mvp.md, ADR 0013, ADR 0014, ADR 0016  

## Contexto

A experiência inicial do openJob baseava-se em uma lista tradicional de cards ou tabela (CP12), herdando padrões de dashboards desktop. Em uso mobile-first, listas longas geram fadiga de decisão, rolagem excessiva e perda de foco em cada oportunidade.

Produtos modernos de consumo de conteúdo comprovaram a eficácia do padrão de feed vertical de tela cheia (um item por viewport com scroll-snap), proporcionando atenção dedicada a cada vaga e triagem ágil (candidatar, pontuar ou descartar com um toque).

## Decisão

Adotar o padrão de **feed vertical de tela cheia** como a nova home principal do app em `/jobs` (com `/` redirecionando para `/jobs`), mantendo:

1. **Scroll-snap nativo:** container com `scroll-snap-type: y mandatory` e itens com `scroll-snap-align: start`, sem bibliotecas externas pesadas.
2. **Janela deslizante de reciclagem:** o DOM mantém apenas o card ativo e seus vizinhos imediatos (máximo de ~3 cards simultâneos montados), utilizando placeholders transparentes para manter o scroll nativo íntegro.
3. **Acessibilidade total:** container com `role="feed"`, cards com `role="article"`, atributo `inert` em cards fora da tela para evitar navegação por foco invisível, suporte às teclas `↑`/`↓` no desktop e anúncio de posição em `aria-live="polite"`.
4. **Preservação da lista tradicional:** a visualização em lista/tabela é preservada em `/jobs/list`, com alternância fluida entre "Feed" e "Lista". O dashboard do CP15 é alocado em `/dashboard`.

## Alternativas rejeitadas

- **Manter apenas a lista tradicional:** não resolve o problema de sobrecarga cognitiva e fadiga de rolagem em smartphones.
- **Substituir completamente a lista:** prejudicaria usuários desktop avançados que realizam seleção em lote e ordenação tabular.
- **Carrosséis JS pesados (Swiper, Slick):** aumentam o bundle, quebram a acessibilidade nativa e introduzem jank em dispositivos móveis modestos.

## Consequências

- Experiência mobile-first imersiva e de alto engajamento.
- Desempenho garantido pela janela deslizante (número constante de nós no DOM independente da quantidade de vagas).
- Total retrocompatibilidade com as ferramentas de triagem e rotas existentes.
