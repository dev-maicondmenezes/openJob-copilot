# ADR 0037 — NLP Core com contratos genéricos, offsets e versionamento

**Status:** aceito (Fase 5 / B11.1)

**Data:** 2026-11-24
**Relaciona:** docs/plan/fase5_release_mvp.md §3–§5, ADR 0038

## Contexto

Keywords, ranking, filtering e o futuro tailoring precisam da mesma
normalização, tokenização, stemming e segmentação. Implementar cada consumidor
com sua própria heurística duplicaria regras, produziria resultados divergentes e
dificultaria testes multilíngues. Também é necessário preservar o texto original
e a proveniência para futuras edições de currículo.

## Decisão

1. **Um único pacote `internal/nlp`.** Ele concentra normalização Unicode,
   detecção de idioma (pt/en/es), tokenização, stopwords, stemming e segmentação
   por seções Markdown. Nenhum consumidor implementa essas etapas localmente.
2. **Contratos genéricos, não específicos de currículo.** `TextDocument` contém
   `OriginalText`, metadados de origem e `[]TextSection`. O adaptador de
   currículo (`internal/curriculum/adapters`) converte Markdown em documento
   genérico com identificadores de seção estáveis (`summary`, `experience`,
   `skills`, ...).
3. **Offsets explícitos.** Offsets de seção são bytes relativos a
   `OriginalText`; offsets de token são bytes relativos ao `TextSection.Text`.
   Isso permite reconstruir qualquer trecho original sem heurísticas.
4. **Recursos embarcados e versionados.** Stopwords e metadados ficam em
   `internal/nlp/assets` com `manifest.json` e licenças. `Resources.Version()`
   é propagado em todos os resultados para auditoria e invalidação de cache.
5. **Determinismo sem rede.** Nenhuma etapa do core faz chamadas externas ou usa
   vocabulário profissional específico.

## Alternativas rejeitadas

- **Normalização dentro de cada motor:** divergência de stems entre keywords e
  ranking, além de testes duplicados.
- **Trabalhar com strings sem offsets:** impossibilita tailoring e explicações
  que apontem para o texto original.
- **Baixar stopwords em runtime:** quebra determinismo, offline-first e
  reprodutibilidade de resultados.

## Consequências

- Novos consumidores (tailoring, inteligência de documentos) reutilizam o core
  sem alterar os motores existentes.
- Mudanças de recursos linguísticos exigem atualizar o manifesto e os goldens.
- O contrato público da API não muda: os offsets são internos ao backend.
