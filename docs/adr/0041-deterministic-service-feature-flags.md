# ADR 0041 — Deterministic Service como fachada e feature flags de rollback

**Status:** aceito (Fase 5 / B11.5)

**Data:** 2026-11-24
**Relaciona:** docs/plan/fase5_release_mvp.md §8–§12, ADR 0037–0040

## Contexto

Os motores determinísticos precisam ser injetados em consumidores heterogêneos
(API síncrona, worker de tarefas, discovery) sem duplicar wiring nem acoplar
cada consumidor a detalhes de NLP, keywords, ranking e filtering. A migração
também precisa ser reversível: os caminhos legados (LLM e heurísticas) devem
continuar funcionando.

## Decisão

1. **Fachada de composição.** `internal/deterministic.Service` constrói e
   compartilha as dependências (analyzer, keyword extractor, ranking, filtering)
   e expõe operações de keywords, score, filtro, indexação e busca. Ela não
   implementa algoritmos nem regras de negócio específicas.
2. **Injeção nos consumidores.** API, handlers de tarefas e discovery recebem a
   fachada via setters opcionais. Quando ela é `nil` ou a flag está desligada, o
   comportamento legado permanece idêntico.
3. **Flags em cascata.** `OPENJOB_FEATURE_SCORE_DETERMINISTIC` é o master
   switch. `OPENJOB_FEATURE_KEYWORDS_DETERMINISTIC` e
   `OPENJOB_FEATURE_FILTER_DETERMINISTIC` só têm efeito com o master ligado
   (`Features.KeywordsEnabled()` / `FilterEnabled()`). O endpoint
   `/api/features` expõe os três estados efetivos.
4. **Rollback sem migração.** Desligar as flags volta aos caminhos anteriores
   (LLM para score, heurística para keywords, `FilterResume` para filtro) sem
   alterar schema ou dados persistidos.
5. **Índice sob demanda.** Quando o score determinístico está ativo, o servidor
   reconstrói o índice FTS5 no startup e indexa novas vagas; com a flag
   desligada, o índice não é usado.

## Alternativas rejeitadas

- **Substituir os caminhos legados de uma vez:** impede rollback e mistura
  migração de comportamento com mudança de arquitetura.
- **Cada consumidor montar os próprios motores:** wiring duplicado, risco de
  configurações divergentes e testes frágeis.
- **Flag única para tudo:** impossibilita habilitar keywords/filtro antes do
  score ou isolar regressões.

## Consequências

- Todos os consumidores usam a mesma infraestrutura quando as flags estão
  ativas, atendendo ao gate do B11.5.
- O custo de manter dois caminhos existe durante a fase de validação; a remoção
  dos legados fica para um release posterior.
- Testes de integração verificam que o caminho determinístico não chama LLM.
