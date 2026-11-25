# ADR 0040 — Filtering sobre o Ranking Engine com compatibilidade de min_matches

**Status:** aceito (Fase 5 / B11.4)

**Data:** 2026-11-24
**Relaciona:** docs/plan/fase5_release_mvp.md §7, ADR 0039

## Contexto

O filtro de currículo atual conta quantas palavras-chave do currículo aparecem
na vaga e compara com `min_matches`. O plano exige reutilizar o Ranking Engine,
manter a semântica de `min_matches`, oferecer limiares configuráveis e motivos
claros de aceitação/rejeição, sem substituir silenciosamente a contagem por um
limiar BM25 arbitrário.

## Decisão

1. **Filtering não calcula similaridade própria.** Ele consome o score do
   Ranking Engine e decide aceitar/rejeitar com base em uma `Policy`.
2. **`min_matches` preservado.** `FilterWithKeywords` conta palavras-chave
   distintas do currículo cujos termos normalizados aparecem na vaga — a mesma
   semântica do filtro legado. `MatchMode: "all"` exige todas as palavras-chave;
   o padrão `"any"` usa apenas `MinMatches`.
3. **Score é critério adicional opcional.** `MinScore` permite um piso de
   aderência sem substituir a contagem. Sem `MinScore`, o comportamento é o
   legado.
4. **Resultado explicável.** `Result` carrega `accepted`, `matched_terms`,
   `missing_terms`, `match_count`, `raw_bm25`, `score` e um `reason` em texto
   claro. O discovery usa `Accepted`/`MatchedTerms` para as colunas existentes.
5. **Fallback controlado.** Se o motor determinístico falhar, o orquestrador
   registra o erro e usa o filtro legado, preservando disponibilidade.

## Alternativas rejeitadas

- **Trocar `min_matches` por um limiar BM25:** mudaria decisões existentes sem
  base calibrada e violaria a compatibilidade exigida.
- **Filtro com similaridade própria:** duplicaria o ranking e produziria scores
  divergentes entre feed e filtro.
- **Rejeição sem motivo:** dificultaria depuração e a UI de explicações.

## Consequências

- Goldens de filtering cobrem relevante, parcialmente relevante e não
  relacionado em áreas profissionais distintas, em pt-BR, inglês e espanhol.
- `TestFilterEquivalenceWithLegacy` compara aceitação e contagem com o
  `FilterResume` legado em casos de token exato; `TestFilterInflectionImprovement`
  documenta a melhoria intencional de stemming (formas flexionadas).
- Ajustar `MinMatches`/`MinScore` é configuração, não mudança de código.
- O filtro determinístico é ativado por flag e pode ser revertido sem migração.
