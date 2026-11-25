# ADR 0042 — Auditoria das dependências linguísticas e do índice

**Status:** aceito (Fase 5 / B11.6)

**Data:** 2026-11-24
**Relaciona:** docs/plan/fase5_release_mvp.md §10, §13, ADR 0037, ADR 0039

## Contexto

O B11.6 exige auditoria de dependências. A infraestrutura determinística
introduziu stemming e passou a usar FTS5, então é preciso registrar o que foi
adicionado, a licença, o custo de runtime e a ausência de chamadas de rede.

## Decisão

1. **Stemming: `github.com/freeeve/go-stemmers` (MIT).** Dependência pura Go,
   sem CGO, sem rede e sem estado global. Cobre os idiomas suportados pelo MVP
   (português, inglês e espanhol). O módulo é pequeno e não traz dependências
   transitivas relevantes.
2. **Índice: `modernc.org/sqlite` (já existente).** O driver pure-Go compila com
   FTS5; a compatibilidade é provada pela migração `0008_jobs_fts.sql` e pelos
   testes de índice (`ranking`/`store`). Nenhum driver CGO foi adicionado.
3. **Recursos linguísticos embarcados.** Stopwords pt/en/es vêm do projeto
   stopwords-iso; a licença está em `internal/nlp/assets/licenses` e o
   `manifest.json` versiona os recursos. Nada é baixado em runtime.
4. **Sem vocabulário profissional proprietário.** YAKE/RAKE usam apenas o texto
   do documento e os recursos genéricos acima.
5. **Reprodutibilidade.** `go.mod`/`go.sum` fixam as versões; o build usa
   `CGO_ENABLED=0`. A auditoria deve ser refeita ao adicionar qualquer nova
   dependência de NLP, tokenização ou armazenamento de índice.

## Alternativas rejeitadas

- **Stemmer caseiro:** maior risco de regressão linguística e custo de
  manutenção; o pacote auditado é pequeno e testado.
- **Snowball via CGO:** quebraria o build estático e a premissa de binário
  único.
- **Baixar recursos em runtime:** quebra offline-first, determinismo e
  reprodutibilidade.

## Consequências

- O binário continua estático, offline e sem CGO.
- Atualizações de recursos linguísticos exigem regenerar goldens e revisar o
  manifesto.
- Novas dependências passam por esta auditoria antes de entrar no B11.
