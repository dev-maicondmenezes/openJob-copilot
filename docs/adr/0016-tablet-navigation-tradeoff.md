# ADR 0016 — Trade-off: bottom nav no tablet vs navigation rail

**Status:** aceito (Checkpoint 12) — decisão provisória, reavaliar após validação
**Data:** 2026-11-16
**Relaciona:** ADR 0014 (anexo curto), docs/plan/12-design.md §4

## Contexto

No intervalo `md`–`< lg` (768–1023px) há duas opções razoáveis: manter a bottom
nav do smartphone (pedido explícito do plano: "mesmo padrão") ou adotar um
*navigation rail* lateral, padrão Material para tablet. O rail melhora o uso em
paisagem e libera altura vertical, mas introduz um terceiro padrão de navegação
para testar e manter.

## Decisão

Manter a **bottom nav no tablet**, com touch targets expandidos (≥ 56px),
espaçamento maior e conteúdo centralizado. A sidebar aparece somente em `≥ lg`.
O iPad em paisagem (geralmente 1024px+) já cai em desktop com sidebar, cobrindo
o caso de "tablet grande" sem criar uma categoria separada.

## Trade-off registrado

- **Ganho:** uma única marcação de navegação para phone+tablet; menos testes,
  menos risco de regressão visual no smartphone (referência canônica).
- **Custo:** em tablet paisagem estreito (ex.: 800×600), a bottom nav ocupa
  altura valiosa e o polegar alcança os extremos com menos conforto.

## Caminho de migração (se a validação indicar)

1. Instrumentar uso real no tablet (toques por item da bottom nav, rotação).
2. Prototipar rail `md`–`< lg` em `/dev/ui` sem remover a bottom nav (feature flag).
3. Validar com usuários; se aprovado, promover a decisão e atualizar ADR 0014,
   `navigation.md`, snapshots e os testes de viewport.

Até lá, qualquer proposta de rail deve ser registrada como exceção documentada,
não como mudança silenciosa de layout.
