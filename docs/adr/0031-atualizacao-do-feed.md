# ADR 0031 — Atualização do feed com botão "Buscar agora" e feedback assíncrono

**Status:** aceito (Checkpoint 16 / Fase 1 MVP)  
**Data:** 2026-11-22  
**Relaciona:** docs/plan/16-infinite-timeline-design.md, docs/plan/release-mvp.md, ADR 0028, ADR 0025, ADR 0010  

## Contexto

A atualização do catálogo de vagas envolve a orquestração assíncrona de múltiplas fontes externas, raspagem defensiva e rate limiting (CP14/ADR 0025). O processo pode demandar vários segundos. O feed não pode travar a tela nem forçar recarga completa de página, garantindo continuidade do consumo.

## Decisão

1. **Botão de atualização explícito:** o botão "Buscar agora" é integrado de forma visível e discreta no topo do feed como mecanismo primário de disparo.
2. **Operação não bloqueante:** ao ser acionado, dispara `POST /api/jobs/discover` e exibe indicador de progresso no próprio botão. A navegação no feed permanece 100% responsiva.
3. **Feedback pontual via toast:** ao concluir a busca, novas vagas são carregadas e inseridas no feed, notificando o usuário com um toast informativo ("X novas vagas adicionadas ao feed" ou "Nenhuma vaga nova no momento").
4. **Acessibilidade garantida:** o mecanismo é operável por teclado, foco acessível e leitores de tela. Gestos de pull-to-refresh em dispositivos móveis atuam como enhancement opcional, sem substituir o botão primário.

## Alternativas rejeitadas

- **Recarregar a página inteira:** causa perda da posição atual de leitura e degrada a sensação de aplicativo SPA/PWA fluido.
- **Apenas pull-to-refresh:** incompatível com navegação por teclado no desktop e inacessível para leitores de tela.

## Consequências

- Feedback claro e transparente de atualização.
- Integração perfeita com a fila de tarefas do CP10 e o rate limiter do CP14.
