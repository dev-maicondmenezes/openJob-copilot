# ADR 0039 — Ranking Engine: BM25 ponderado sobre corpus global e FTS5

**Status:** aceito (Fase 5 / B11.3)

**Data:** 2026-11-24
**Relaciona:** docs/plan/fase5_release_mvp.md §6, ADR 0037, ADR 0040

## Contexto

O score atual compara currículo e vaga de forma isolada (BM25 pairwise). Isso
ignora a raridade dos termos no conjunto de vagas e torna os scores sensíveis ao
tamanho de cada descrição. O plano pede um mecanismo compartilhado entre
currículos e vagas, usando o corpus global de vagas deduplicadas e um índice
FTS5.

## Decisão

1. **BM25 ponderado por campo.** `k1 = 1.2`, `b = 0.75`; pesos iniciais:
   título 4, requisitos 3, responsabilidades 2, descrição 1. Os pesos ficam em
   `search.Weights` e podem ser substituídos sem alterar o algoritmo.
2. **Índice FTS5 como corpus global.** A migração `0008_jobs_fts.sql` cria
   `jobs_fts` com `unicode61 remove_diacritics 2` e uma tabela de vocabulário.
   O índice armazena termos já normalizados/stemizados; a query é construída a
   partir dos stems do currículo, divididos em peças alfanuméricas válidas para o
   FTS5.
3. **Fallback pairwise.** Sem índice ou com corpus vazio, o ranking usa os
   próprios campos da vaga como corpus local. Isso mantém o endpoint síncrono
   funcional antes do rebuild e em testes.
4. **Score provisório e explícito.** O score 0–100 é uma normalização monotônica
   do BM25 bruto (`raw / ideal`), marcada com `Calibrated: false` até existir
   calibração com exemplos rotulados. `raw_bm25`, termos correspondidos e
   ausentes acompanham o resultado para explicabilidade. Toda resposta do
   caminho determinístico expõe `calibrated`, `mechanism_version`,
   `resource_version` e `index_version`, para que a UI não trate o valor como
   percentual validado.
5. **Reindexação idempotente e corpus deduplicado.** `Upsert`/`Delete`/`Rebuild`
   são operações do repositório; no startup o servidor reconstrói o índice a
   partir de `jobs` quando a flag determinística está ativa. O rebuild deduplica
   por URL e, para vagas sem URL, por conteúdo normalizado. Novas vagas são
   indexadas na inserção. `search.IndexVersion` versiona a configuração de
   indexação (tokenizador + contrato de campos).

## Alternativas rejeitadas

- **TF-IDF + cosseno em paralelo:** o plano proíbe dois mecanismos concorrentes
  para a mesma finalidade.
- **Índice em memória:** perderia o corpus entre reinícios e exigiria rebuild
  integral em cada boot sem ganho relevante.
- **Score calibrado sem dados rotulados:** daria falsa precisão; a flag
  `Calibrated` deixa a limitação explícita.

## Consequências

- Scores passam a refletir a raridade dos termos no mercado de vagas local.
- O rebuild no startup é O(n) e deve ser monitorado em bases grandes.
- Mudanças de pesos ou normalização exigem atualizar goldens de scoring.
