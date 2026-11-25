# Arquitetura Fase 5 — B11: Infraestrutura determinística

**Versão:** Reformulação arquitetural\
**Release:** MVP\
**Objetivo:** substituir por algoritmos determinísticos as funcionalidades de processamento textual previstas no B11, preservando os contratos existentes e preparando a arquitetura para futuras extensões de Tailoring determinístico e Inteligência de documentos.

---

# 1. Objetivo e limites da Fase 5

A Fase 5 implementará uma infraestrutura compartilhada de processamento textual e três capacidades determinísticas:

1. **Keyword Extraction:** identificação de expressões relevantes no currículo.
2. **Resume Filtering:** seleção de vagas conforme critérios mínimos de correspondência com o currículo.
3. **Fit Scoring:** cálculo e explicação da relevância entre currículo e vaga.

As três capacidades compartilharão recursos linguísticos, normalização, versionamento e infraestrutura de busca, mas manterão responsabilidades independentes.

A arquitetura também deverá evitar dependências específicas de currículo ou vaga no núcleo de NLP, permitindo que novos consumidores sejam adicionados futuramente.

## 1.1. O que esta fase não fará

Continuam fora do escopo:

- Tailoring determinístico de currículos.
- Mapeamento semântico de evidências profissionais.
- Geração ou reescrita de experiências profissionais.
- Análise inteligente de documentos.
- Extração semântica de informações de documentos arbitrários.
- OCR e interpretação de documentos digitalizados.
- Geração de cartas de apresentação.
- Substituição do LLM nas funcionalidades de tailoring, carta e análise de currículo.
- Personalização comportamental e Learning to Rank.
- Novos fluxos ou alterações de UX não previstos no B11.

Esses itens deverão utilizar a infraestrutura entregue nesta fase quando forem implementados em fases posteriores.

## 1.2. Princípio arquitetural

A Fase 5 não deverá implementar uma plataforma de inteligência documental antecipadamente.

Deverá entregar uma infraestrutura pequena, reutilizável e desacoplada, com contratos que permitam adicionar novos motores sem modificar o núcleo de processamento.

**Implementar agora o que o MVP precisa. Preparar os contratos para o que virá depois.**

---

# 2. Arquitetura proposta

A arquitetura será organizada em quatro camadas:

1. NLP Core.
2. Motores determinísticos especializados.
3. Deterministic Service.
4. Adaptadores de integração com API, workers e discovery.

```
                 API / WORKERS / DISCOVERY
                            |
                            v
                  DETERMINISTIC SERVICE
                            |
              +-------------+-------------+
              |             |             |
              v             v             v
          KEYWORD        RANKING       FILTER
          ENGINE         ENGINE        ENGINE
              |             |             |
              +-------------+-------------+
                            |
                            v
                         NLP CORE
                            |
                            v
                 SHARED TEXT CONTRACTS
                            |
              +-------------+-------------+
              |             |             |
              v             v             v
          LANGUAGE      NORMALIZATION   TOKENIZATION
          RESOURCES     AND STEMMING    AND SECTIONS
```

O Ranking Engine também utilizará o índice global de vagas e suas interfaces de armazenamento.

## 2.1. Responsabilidades

| Componente            | Responsabilidade                              |
| --------------------- | --------------------------------------------- |
| NLP Core              | Processamento linguístico genérico            |
| Keyword Engine        | Extração de keywords por seção                |
| Ranking Engine        | Busca e pontuação BM25                        |
| Filter Engine         | Política de aceitação e rejeição de vagas     |
| Deterministic Service | Composição dos motores e contratos de serviço |
| Adapters              | Integração com API, workers e discovery       |

O NLP Core não deverá conhecer conceitos como currículo, experiência profissional, vaga, aderência ou evidência.

---

# 3. Contratos compartilhados de processamento textual

Esta é a principal preparação para a evolução futura.

Os contratos internos deverão representar documentos e trechos de texto de maneira genérica, sem assumir que todo documento será um currículo.

## 3.1. Representação de documento

Uma representação inicial poderá ser:

```
type TextDocument struct {
    ID             string
    OriginalText   string
    Language       string
    Sections       []TextSection
    Metadata       DocumentMetadata
    ResourceVersion string
}

type TextSection struct {
    ID          string
    Title       string
    Text        string
    StartOffset int
    EndOffset   int
    Tokens      []Token
}

type Token struct {
    Original    string
    Normalized  string
    Stem        string
    StartOffset int
    EndOffset   int
    Position    int
}

type DocumentMetadata struct {
    SourceType string
    SourceID   string
}
```

Os campos poderão ser ajustados durante a implementação, respeitando estas propriedades:

- Preservar o texto original.
- Preservar a representação normalizada.
- Preservar as posições dos tokens e trechos.
- Identificar idioma e versão dos recursos linguísticos.
- Permitir representar diferentes tipos de documentos.
- Não exigir que o documento seja um currículo ou uma vaga.

Os offsets deverão ter uma convenção documentada, preferencialmente posições em bytes UTF-8, para evitar ambiguidades entre consumidores.

## 3.2. Por que preservar offsets e proveniência?

Os motores atuais precisam comparar palavras e expressões.

No futuro, o Tailoring determinístico poderá precisar identificar exatamente qual trecho do currículo comprova determinada competência.

A Inteligência de documentos poderá precisar localizar a origem de uma informação em um documento processado.

Se o sistema descartar o texto original, as posições e a origem dos trechos, essas funcionalidades terão que reconstruir informações que poderiam ter sido preservadas desde o início.

Portanto, os contratos devem preservar esses dados agora.

**Importante:** preservar posições e proveniência não significa implementar o mapeamento de evidências nesta fase.

## 3.3. O que não deve entrar nos contratos do NLP Core

Não incluir:

- `ResumeExperience`.
- `JobRequirement`.
- `ProfessionalSkill`.
- `EvidenceMapping`.
- `TailoringSuggestion`.
- `DocumentUnderstandingResult`.

Esses são conceitos de domínios ou funcionalidades futuras.

O NLP Core deverá continuar sendo uma infraestrutura genérica.

---

# 4. NLP Core

O NLP Core será responsável pelo processamento linguístico compartilhado.

## 4.1. Responsabilidades

- Identificação de idioma.
- Normalização Unicode.
- Segmentação de frases.
- Tokenização.
- Aplicação de stopwords.
- Stemming.
- Preservação das formas originais.
- Representação normalizada.
- Preservação de offsets.
- Versionamento dos recursos utilizados.

A segmentação semântica de seções de currículo não deverá ser responsabilidade do NLP Core.

O processamento linguístico poderá receber seções já identificadas por um consumidor ou trabalhar com segmentos genéricos de texto.

A identificação de seções de currículo necessária ao Keyword Engine será responsabilidade do adaptador ou componente específico de currículo.

## 4.2. Bibliotecas e recursos candidatos

| Responsabilidade        | Opção                                   |
| ----------------------- | --------------------------------------- |
| Stopwords               | Stopwords ISO, snapshots versionados    |
| Identificação de idioma | Lingua Go                               |
| Stemming                | Snowball via go-stemmers                |
| Tokenização Unicode     | UAX #29                                 |
| Normalização            | Implementação compartilhada do NLP Core |

Os idiomas iniciais continuam sendo:

- PT-BR.
- EN-US.
- ES-419.

As listas externas de stopwords deverão ser importadas, versionadas e distribuídas com a aplicação.

Não deverá existir download de recursos linguísticos durante a inicialização do sistema.

As bibliotecas candidatas deverão passar por avaliação de licença, compatibilidade, manutenção e testes antes da adoção definitiva.

## 4.3. Versionamento

O NLP Core deverá disponibilizar a versão dos recursos utilizados no processamento.

A versão deverá considerar:

- Algoritmos.
- Bibliotecas.
- Stopwords.
- Configurações linguísticas.
- Regras de normalização.

Resultados determinísticos deverão ser reproduzíveis com a mesma entrada, configuração e versão dos recursos.

---

# 5. Keyword Engine

## 5.1. Objetivo

Substituir a chamada LLM do handler `extract_resume_keywords`, mantendo o comportamento externo da API.

O mecanismo deverá identificar palavras e expressões relevantes no currículo sem depender de vocabulários específicos de profissões.

## 5.2. Algoritmos

- YAKE como algoritmo principal.
- RAKE como fallback.

YAKE será executado por seção.

RAKE será acionado quando YAKE falhar tecnicamente, não produzir candidatos válidos ou não conseguir processar adequadamente uma seção.

Os scores brutos dos algoritmos não deverão ser misturados diretamente, pois possuem escalas e interpretações diferentes.

## 5.3. Fluxo

```
Curriculum Input
       |
       v
Curriculum Section Adapter
       |
       v
Generic Text Document
       |
       v
NLP Core
       |
       v
Keyword Extraction
       |
       +------> YAKE
       |           |
       |           v
       |       Valid Results?
       |           |
       |           +---- No ----> RAKE
       |           |
       +-----------+
                   |
                   v
          Consolidation
                   |
                   v
          Deduplication
                   |
                   v
                Top-N
```

## 5.4. Resultado

Exemplo ilustrativo:

```
{
  "keywords": [
    {
      "text": "gestão de projetos",
      "normalized": "gest projet",
      "section": "experience",
      "source": "yake",
      "rank": 1
    }
  ],
  "algorithm_version": "1.0"
}
```

Os identificadores de seção deverão respeitar a estrutura real do currículo.

Cada keyword deverá preservar sua forma original, sua representação normalizada, sua seção e o algoritmo que a produziu.

## 5.5. Critérios de aceite

- YAKE por seção.
- Fallback RAKE.
- Top-N configurável.
- Deduplicação de expressões.
- Resultados determinísticos.
- Nenhuma chamada de rede.
- Nenhuma dependência de vocabulário profissional.
- Contrato externo da API preservado.

---

# 6. Ranking Engine

## 6.1. Objetivo

Implementar o mecanismo compartilhado de busca e pontuação entre currículos e vagas.

O mecanismo deverá utilizar o corpus global de vagas deduplicadas e o índice FTS5.

## 6.2. Algoritmo

BM25 ponderado por campos.

Parâmetros iniciais:

- `k1 = 1.2`
- `b = 0.75`

Pesos experimentais:

| Campo da vaga     | Peso |
| ----------------- | ---- |
| Título            | 4,0  |
| Requisitos        | 3,0  |
| Responsabilidades | 2,0  |
| Descrição         | 1,0  |

Os pesos deverão ser avaliados com um conjunto de exemplos representativos.

## 6.3. Responsabilidades

- Construir consultas a partir do texto processado do currículo.
- Consultar o índice global de vagas.
- Executar BM25 ponderado.
- Retornar resultados ordenados.
- Preservar informações que permitam explicar a correspondência.
- Expor o score bruto.
- Identificar a versão do mecanismo e dos recursos utilizados.

## 6.4. Score bruto e score de interface

O score bruto do BM25 não deverá ser apresentado diretamente como percentual de aderência.

Uma transformação para `fit_score` de interface somente poderá ser utilizada quando existir calibração validada.

O resultado deverá diferenciar claramente:

- Pontuação bruta de ranking.
- Pontuação calibrada, quando disponível.
- Termos correspondentes.
- Informações utilizadas na explicação.

## 6.5. Corpus e índice

O índice deverá utilizar a coleção global de vagas deduplicadas.

Deverá suportar:

- Inserção incremental.
- Atualização de documentos.
- Deduplicação.
- Reconstrução do índice.
- Consistência entre corpus e índice.
- Versionamento da configuração de indexação.

A implementação deverá começar pela prova de compatibilidade do FTS5 com o driver SQLite utilizado atualmente.

---

# 7. Filter Engine

## 7.1. Objetivo

Substituir o mecanismo atual de filtragem por currículo, reutilizando o Ranking Engine.

O filtro não deverá implementar outro mecanismo independente de similaridade.

## 7.2. Separação de responsabilidades

| Funcionalidade | Responsabilidade                               |
| -------------- | ---------------------------------------------- |
| Ranking        | Ordenar vagas por relevância                   |
| Score Fit      | Expor a relevância calculada                   |
| Filtering      | Decidir se a vaga atende aos critérios mínimos |

O Filter Engine consumirá os resultados do Ranking Engine e aplicará políticas de aceitação.

## 7.3. Política inicial

```
type FilterPolicy struct {
    MinMatches int
    MinScore   *float64
}
```

O resultado poderá seguir:

```
type FilterResult struct {
    JobID        int64
    Accepted     bool
    MatchedTerms []string
    MissingTerms []string
    MatchCount   int
    RawBM25      float64
    Reason       string
}
```

Os contratos são ilustrativos e deverão ser compatibilizados com os tipos existentes.

## 7.4. Compatibilidade com min_matches

O comportamento atual de `min_matches` deverá ser auditado antes da migração.

Etapas:

1. Identificar a semântica atual.
2. Documentar como as correspondências são contadas.
3. Definir a representação normalizada das correspondências.
4. Criar testes de equivalência e regressão.
5. Manter uma política de compatibilidade.
6. Calibrar eventuais limiares BM25 com exemplos rotulados.

Não substituir `min_matches` silenciosamente por um limiar arbitrário.

Não implementar TF-IDF + cosseno em paralelo ao BM25 para a mesma finalidade.

## 7.5. Critérios de aceite

- Reutilização do Ranking Engine.
- Compatibilidade documentada com `min_matches`.
- Limiares configuráveis.
- Motivos claros de aceitação e rejeição.
- Testes de regressão do comportamento anterior.

---

# 8. Deterministic Service

O Deterministic Service será a fachada de composição dos motores.

Não deverá concentrar algoritmos nem regras específicas de cada funcionalidade.

## 8.1. Responsabilidades

- Disponibilizar operações de keywords.
- Disponibilizar operações de ranking.
- Disponibilizar operações de filtering.
- Compartilhar as dependências do NLP Core.
- Centralizar configuração e versionamento.
- Permitir injeção dos serviços nos consumidores.

Exemplo:

```
type Service struct {
    Keywords KeywordService
    Ranking  RankingService
    Filter   FilterService
}
```

Os handlers não deverão conhecer YAKE, RAKE, Snowball ou detalhes de SQL/FTS5.

Deverão solicitar operações por meio dos contratos de serviço.

## 8.2. Regras de dependência

- NLP Core não importa `store`.
- NLP Core não importa API, discovery ou currículo.
- Keywords não importa API, discovery ou score.
- Ranking depende de interfaces de busca.
- Filtering consome Ranking.
- API e workers utilizam a mesma fachada.
- Os contratos externos existentes permanecem preservados.

A arquitetura não deverá criar dependências circulares.

---

# 9. Organização de pacotes

Estrutura proposta:

```
backend/internal/
│
├── nlp/
│   ├── analyzer.go
│   ├── document.go
│   ├── section.go
│   ├── token.go
│   ├── language.go
│   ├── segmenter.go
│   ├── tokenizer.go
│   ├── normalizer.go
│   ├── stemmer.go
│   ├── stopwords.go
│   ├── resources.go
│   │
│   ├── providers/
│   │   ├── embedded.go
│   │   └── directory.go
│   │
│   └── assets/
│       ├── manifest.json
│       ├── stopwords/
│       └── licenses/
│
├── keywords/
│   ├── extractor.go
│   ├── yake.go
│   ├── rake.go
│   ├── consolidator.go
│   ├── result.go
│   └── *_test.go
│
├── ranking/
│   ├── service.go
│   ├── query_builder.go
│   ├── bm25.go
│   ├── result.go
│   ├── explanation.go
│   └── *_test.go
│
├── filtering/
│   ├── service.go
│   ├── policy.go
│   ├── result.go
│   └── *_test.go
│
├── score/
│   ├── service.go
│   ├── compatibility.go
│   └── score_test.go
│
├── deterministic/
│   ├── service.go
│   └── dependencies.go
│
└── curriculum/
    └── adapters/
        └── sections.go
```

O adaptador de currículo é responsável apenas por adequar o currículo existente aos contratos genéricos de texto.

Não deverá implementar interpretação semântica, mapeamento de evidências ou tailoring.

A estrutura poderá ser ajustada conforme os padrões já existentes no repositório. Não criar pacotes vazios apenas para reservar espaço para funcionalidades futuras.

---

# 10. Integração com o sistema existente

A migração deverá preservar os contratos externos.

## 10.1. Extração de keywords

```
API: extract_resume_keywords
             |
             v
Deterministic Service
             |
             v
Curriculum Adapter
             |
             v
NLP Core
             |
             v
Keyword Engine
             |
             v
Existing API Response
```

O `discovery/orchestrator.go` deverá reutilizar o mesmo mecanismo de extração para evitar divergência entre API e descoberta.

## 10.2. Score Fit

```
API / Worker: score_job
             |
             v
Deterministic Service
             |
             v
NLP Core
             |
             v
Resume Query Builder
             |
             v
Global Job Search Index
             |
             v
Weighted BM25
             |
             v
Explanation Builder
             |
             v
Existing Score Result
```

A implementação deverá preservar a estrutura de resposta atual.

## 10.3. Filtro por currículo

```
Discovery Orchestrator
          |
          v
Deterministic Service
          |
          v
NLP Core
          |
          v
Resume Query Builder
          |
          v
Ranking Engine
          |
          v
Filter Policy
          |
          v
Accepted / Rejected + Reason
```

---

# 11. Feature flags e rollback

A flag existente `score.deterministic` deverá ser preservada como controle principal, respeitando a compatibilidade com `/api/features`.

Poderão existir controles específicos por capacidade:

```
OPENJOB_FEATURE_SCORE_DETERMINISTIC=false
OPENJOB_FEATURE_KEYWORDS_DETERMINISTIC=false
OPENJOB_FEATURE_FILTER_DETERMINISTIC=false
```

Regras:

- A flag geral funciona como master switch.
- Flags específicas controlam a ativação individual.
- Nenhuma flag específica poderá habilitar uma funcionalidade se o master switch estiver desligado.
- A migração deverá permitir ativação gradual.
- O rollback deverá preservar o caminho legado enquanto ele for necessário.

O comportamento anterior deverá permanecer disponível durante a validação.

A remoção dos caminhos antigos só ocorrerá depois da aceitação da Fase 5.

---

# 12. Plano de implementação

O B11 será organizado em seis subfases, sem acrescentar funcionalidades ao escopo original.

| Subfase | Entrega        | Resultado                                 |
| ------- | -------------- | ----------------------------------------- |
| B11.1   | NLP Core       | Processamento linguístico compartilhado   |
| B11.2   | Keyword Engine | YAKE, RAKE e top-N por seção              |
| B11.3   | Search/Ranking | BM25 sobre vagas                          |
| B11.4   | Filter Engine  | Filtro com compatibilidade `min_matches`  |
| B11.5   | Integração     | Migração de API, workers e discovery      |
| B11.6   | Hardening      | Testes, benchmarks, ADR e validação final |

## B11.1 — NLP Core

- Validar bibliotecas e recursos.
- Implementar identificação de idioma.
- Implementar normalização Unicode.
- Implementar tokenização.
- Implementar stopwords e stemming.
- Preservar texto original e offsets.
- Implementar contratos genéricos.
- Versionar os recursos.

**Gate:** processamento reproduzível em PT-BR, EN-US e ES-419.

## B11.2 — Keyword Engine

- Implementar YAKE.
- Implementar fallback RAKE.
- Processar seções do currículo.
- Consolidar e deduplicar expressões.
- Implementar top-N configurável.
- Preservar o algoritmo de origem.

**Gate:** extração determinística, sem LLM e com contrato externo preservado.

## B11.3 — Search/Ranking

- Auditar compatibilidade do FTS5.
- Implementar ou ajustar migrations necessárias.
- Integrar corpus global deduplicado.
- Implementar índice incremental e reconstruível.
- Implementar consulta BM25 ponderada.
- Implementar explicabilidade.
- Separar score bruto de score calibrado.

**Gate:** ranking reproduzível, explicável e integrado ao corpus global.

## B11.4 — Filter Engine

- Implementar política de filtragem.
- Reutilizar Ranking Engine.
- Auditar `min_matches`.
- Implementar compatibilidade.
- Implementar testes de regressão.

**Gate:** filtragem determinística sem regressão funcional relevante.

## B11.5 — Integração

Arquivos prioritários:

| Arquivo                                      | Responsabilidade                 |
| -------------------------------------------- | -------------------------------- |
| `backend/internal/api/jobs.go`               | Score determinístico             |
| `backend/internal/api/*`                     | Endpoint de keywords e contratos |
| `backend/internal/tasks/handlers.go`         | Worker de scoring                |
| `backend/internal/discovery/orchestrator.go` | Extração e filtro                |
| `backend/internal/config/config.go`          | Flags e parâmetros               |
| `backend/internal/api/server.go`             | Injeção de serviços              |

A migração deverá ser gradual:

1. Implementar serviços isoladamente.
2. Testar os contratos.
3. Integrar a API.
4. Integrar os workers.
5. Integrar discovery.
6. Validar compatibilidade.
7. Habilitar as flags conforme os gates.

**Gate:** todos os consumidores utilizam a mesma infraestrutura determinística.

## B11.6 — Hardening

- Golden tests.
- Testes de regressão.
- Testes multilíngues.
- Testes de integração.
- Benchmarks.
- Validação das feature flags.
- Auditoria de dependências.
- Documentação das decisões arquiteturais.

**Gate:** `make test` e `make vet` aprovados, com os critérios de qualidade atendidos.

---

# 13. Matriz de testes

| Capacidade         | Testes essenciais                                          |
| ------------------ | ---------------------------------------------------------- |
| NLP                | Idiomas, stemming, Unicode e stopwords                     |
| Contratos          | Texto original, offsets e versionamento                    |
| Keywords           | YAKE, fallback RAKE, expressões e top-N                    |
| Keywords por seção | Seções diferentes não se contaminam                        |
| BM25               | Relevância, pesos por campo e determinismo                 |
| Filtering          | `min_matches`, limiares e regressão                        |
| Integração         | API, worker e discovery                                    |
| Rede               | Contador zero de chamadas LLM nas funcionalidades migradas |
| Performance        | Menos de 50 ms por vaga, conforme gate original            |
| Multilíngue        | PT-BR, EN-US e ES-419                                      |

## 13.1. Golden tests

```
testdata/
├── keywords/
│   ├── portuguese/
│   ├── english/
│   └── spanish/
│
├── filtering/
│   ├── relevant/
│   ├── partially_relevant/
│   └── unrelated/
│
└── scoring/
    ├── exact_match/
    ├── strong_match/
    ├── partial_match/
    └── weak_match/
```

As fixtures deverão representar diferentes áreas profissionais:

- Engenharia civil.
- Administração.
- Educação.
- Saúde.
- Contabilidade.
- Logística.
- Marketing.
- Computação.

Os resultados esperados deverão ser versionados junto aos algoritmos e recursos linguísticos.

---

# 14. Preparação para evolução pós-MVP

A arquitetura deverá permitir adicionar novos consumidores dos contratos compartilhados sem ampliar o B11.

As duas evoluções prioritárias são Tailoring determinístico e Inteligência de documentos.

## 14.1. Tailoring determinístico

**Não implementar na Fase 5.**

Futuramente, o Tailoring Engine poderá consumir:

- NLP Core.
- Representação genérica de documentos.
- Segmentos com texto original e offsets.
- Keyword Engine.
- Ranking Engine.
- Recursos de normalização.

A implementação futura poderá acrescentar componentes especializados:

| Componente futuro         | Responsabilidade                              |
| ------------------------- | --------------------------------------------- |
| Resume Analyzer           | Interpretar a estrutura do currículo          |
| Job Analyzer              | Identificar requisitos da vaga                |
| Evidence Mapping Engine   | Relacionar requisitos a evidências existentes |
| Tailoring Strategy Engine | Definir prioridades de adaptação              |
| Tailoring Generator       | Produzir alterações no currículo              |
| Validation Engine         | Verificar consistência e preservar fatos      |

A Fase 5 não implementará esses componentes.

A única preparação necessária agora será preservar contratos genéricos, texto original, offsets e versionamento.

## 14.2. Inteligência de documentos

**Não implementar na Fase 5.**

Futuramente, uma camada de inteligência documental poderá consumir os contratos do NLP Core e acrescentar:

- Adaptadores para diferentes formatos.
- Extração estruturada de conteúdo.
- Identificação de tipos de documento.
- Interpretação de estruturas documentais.
- Extração de entidades e informações.
- Proveniência das informações extraídas.
- Validação e classificação dos resultados.

A arquitetura atual deverá permitir que novos adaptadores forneçam documentos ao NLP Core sem modificar os motores de keywords, ranking e filtering.

A Fase 5 não implementará OCR, extração semântica genérica, classificação documental ou interpretação inteligente.

## 14.3. Limite da preparação arquitetural

| Preparar agora                          | Implementar futuramente           |
| --------------------------------------- | --------------------------------- |
| Contratos genéricos de texto            | Interpretadores especializados    |
| Preservação de texto original           | Mapeamento de evidências          |
| Offsets e proveniência básica           | Rastreabilidade semântica         |
| Identificação e versionamento de idioma | Análise documental avançada       |
| Segmentos e tokens reutilizáveis        | Tailoring determinístico          |
| Interfaces desacopladas                 | Motores especializados adicionais |

Não criar interfaces abstratas sem consumidor real, serviços vazios ou modelos de domínio antecipados.

A preparação deverá ocorrer nos contratos e nas fronteiras entre os componentes, não na implementação prematura das funcionalidades futuras.

---

# 15. Estimativa e riscos

A estimativa original de 4–6 dias úteis deve ser considerada uma referência, não um compromisso.

O B11 envolve:

- Compatibilidade do FTS5 com o driver atual.
- Corpus e índice global.
- Deduplicação e reconstrução do índice.
- NLP Core.
- Seleção e validação de bibliotecas.
- YAKE e RAKE.
- BM25 ponderado.
- Compatibilidade do filtro.
- Migração de consumidores.
- Feature flags e rollback.
- Golden tests multilíngues.

A estimativa final deverá ser atualizada após as provas técnicas iniciais.

## 15.1. Riscos

| Risco                                             | Mitigação                                      |
| ------------------------------------------------- | ---------------------------------------------- |
| Incompatibilidade do FTS5                         | Spike antes das migrations                     |
| Maturidade insuficiente das bibliotecas YAKE/RAKE | Avaliação técnica e alternativa documentada    |
| Alteração do ranking do feed                      | Golden tests e comparação                      |
| Divergência entre consumidores                    | Fachada compartilhada                          |
| Regressão de `min_matches`                        | Testes de compatibilidade                      |
| Recursos linguísticos inconsistentes              | Versionamento e snapshots                      |
| Crescimento de escopo                             | Funcionalidades futuras permanecem fora do B11 |

---

# 16. Entregáveis finais

| Entregável             | Resultado esperado                            |
| ---------------------- | --------------------------------------------- |
| NLP Core               | Processamento linguístico compartilhado       |
| NLP Resources          | Recursos externos versionados                 |
| Text Contracts         | Documento, seção, token e proveniência básica |
| Keyword Engine         | Extração determinística                       |
| Search Index           | Índice global incremental e reconstruível     |
| Ranking Engine         | BM25 ponderado e explicável                   |
| Filter Engine          | Filtro determinístico                         |
| Deterministic Service  | Fachada comum                                 |
| API/Worker Integration | Consumidores migrados                         |
| Golden Tests           | Validação generalista                         |
| Benchmarks             | Medição de desempenho                         |
| ADRs                   | Decisões arquiteturais documentadas           |
| Release Plan           | B11 atualizado e critérios cumpridos          |

---

# 17. Critério de conclusão da Fase 5

A Fase 5 estará concluída quando:

1. Keywords forem extraídas sem LLM.
2. Score Fit for calculado deterministicamente.
3. Filtro por currículo utilizar o mecanismo compartilhado.
4. Os contratos externos existentes forem preservados.
5. API, workers e discovery utilizarem a mesma infraestrutura.
6. O comportamento anterior puder ser recuperado por feature flags durante a validação.
7. Os resultados forem reproduzíveis e explicáveis.
8. Os testes multilíngues e de regressão forem aprovados.
9. O desempenho atender ao gate estabelecido.
10. `make test` e `make vet` forem aprovados.

**Resultado arquitetural esperado:** o OpenJob terá uma infraestrutura determinística compartilhada para processamento textual, keywords, ranking e filtragem, sem implementar antecipadamente Tailoring determinístico ou Inteligência de documentos.