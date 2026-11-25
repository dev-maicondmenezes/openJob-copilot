# ADR 0038 — Keyword Engine: YAKE por seção com fallback RAKE

**Status:** aceito (Fase 5 / B11.2)

**Data:** 2026-11-24
**Relaciona:** docs/plan/fase5_release_mvp.md §5, ADR 0037

## Contexto

A extração atual de palavras-chave é uma heurística simples sobre o currículo
inteiro. Ela não preserva a seção de origem, não deduplica expressões e mistura
termos de experiência, habilidades e formação. O plano exige YAKE por seção,
fallback RAKE, Top-N configurável, deduplicação e determinismo, mantendo o
contrato externo da API.

## Decisão

1. **YAKE por seção.** Cada seção é analisada isoladamente para que termos de
   seções diferentes não se contaminem. O ranking é feito dentro da seção e a
   consolidação usa apenas a posição relativa (`rank`) e a prioridade da fonte.
2. **Fallback RAKE.** Quando uma seção não produz candidatos suficientes no
   YAKE (texto curto ou sem padrões), o RAKE gera candidatos para aquela seção.
   Os escores brutos dos dois algoritmos nunca são misturados: a consolidação
   compara apenas ranks e prioridade.
3. **Deduplicação por forma normalizada.** A chave é a sequência de stems
   normalizados; a primeira ocorrência preserva a forma original e a seção de
   origem. Expressões idênticas em seções diferentes aparecem uma vez.
4. **Top-N configurável.** O limite é aplicado após a consolidação global,
   mantendo a ordem determinística.
5. **Resultado rico.** Cada keyword carrega `text`, `normalized`, `section`,
   `source` (`yake`/`rake`) e `rank`. `ExtractKeywordStrings` existe apenas para
   consumidores legados que precisam de `[]string`.

## Alternativas rejeitadas

- **TF-IDF global:** depende de corpus e não oferece o fallback solicitado.
- **YAKE sobre o documento inteiro:** perde proveniência por seção e mistura
  contextos.
- **Misturar escores YAKE e RAKE:** escalas incompatíveis, resultados não
  comparáveis e difíceis de explicar.

## Consequências

- A API existente (`count`, `keywords`, `curriculum`, `at`) continua válida; o
  handler passa a expor `algorithm_version` quando o motor determinístico está
  ativo.
- Goldens por idioma (pt/en/es) travam a qualidade e o determinismo.
- Ajustes de pesos do YAKE exigem regenerar goldens de forma intencional.
