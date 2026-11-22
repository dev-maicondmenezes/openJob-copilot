# ADR 0029 — Descrição completa em rota dedicada (/jobs/[id]) com teaser no feed

**Status:** aceito (Checkpoint 16 / Fase 1 MVP)  
**Data:** 2026-11-22  
**Relaciona:** docs/plan/16-infinite-timeline-design.md, docs/plan/release-mvp.md, ADR 0028, ADR 0030  

## Contexto

Descrições de vagas (JDs) possuem tamanhos muito variáveis, frequentemente ultrapassando dezenas de parágrafos. Exibir todo o texto dentro do card de tela cheia quebra a cadência de scroll-snap, esconde botões de ação e prejudica a triagem rápida. Por outro lado, modais sobrepostos em telas pequenas costumam apresentar problemas de viewport height (`100vh` vs `100dvh`) e perda de histórico de navegação.

## Decisão

Separar o consumo superficial do aprofundado:

1. **Teaser no card do feed:** cada item exibe apenas os primeiros ~280 caracteres renderizados em Markdown seguro com gradiente de fade inferior e um link/botão "Saiba mais…".
2. **Rota dedicada de detalhe (`/jobs/[id]`):** a descrição completa é apresentada em página inteira com scroll vertical livre, formatação Markdown completa e segura, informações de cabeçalho (empresa, pontuação, modalidade, fonte com link original obrigatório) e botão "Voltar ao feed".
3. **Deep linking e compartilhamento:** qualquer vaga pode ser diretamente acessada ou compartilhada pelo link canônico `/jobs/[id]`.

## Alternativas rejeitadas

- **Expandir o card inline no feed:** altera dinamicamente a altura do container, quebra o alinhamento do snap e causa repaints custosos.
- **Exibir apenas em modal dialog:** dificulta o retorno pelo botão "Voltar" nativo do navegador/dispositivo móvel e impede o compartilhamento direto de links.

## Consequências

- O feed permanece focado, ágil e estável no snap.
- Usuários interessados têm um ambiente limpo e dedicado para leitura aprofundada da JD.
- Suporte nativo a histórico do navegador e deep links.
