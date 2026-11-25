# **Plano de implementação — Fase 4 do Release MVP**

Escopo definido

**Parte 1: armazenamento, NLP, descoberta determinística e ranking BM25.**



O objetivo desta etapa é entregar o novo mecanismo de relevância funcionando com as vagas reais coletadas pelo OpenJob, sem depender do histórico comportamental dos usuários.

A implementação será dividida em três fases, na ordem correta de dependência:

1. SQLite WAL, FTS5, deduplicação global e indexação das vagas.
2. Normalização linguística, recursos NLP versionados e extração determinística de keywords.
3. BM25 com campos ponderados e comparação entre currículo e vaga.

## **1. Decisões arquiteturais**

Antes de detalhar os arquivos, estas são as decisões que orientam toda a implementação.

| Componente                | Decisão                                                    |
| ------------------------- | ---------------------------------------------------------- |
| Banco principal           | SQLite, mantendo o driver atual inicialmente               |
| Concorrência              | WAL, transações curtas e pool controlado                   |
| Busca textual             | SQLite FTS5                                                |
| Ranking                   | BM25 do FTS5 com pesos por campo                           |
| Corpus estatístico        | Coleção global de vagas únicas                             |
| Normalização linguística  | Bibliotecas NLP externas e versão fixada                   |
| Stopwords                 | Dados externos versionados, sem listas embutidas no código |
| Stemming                  | Snowball                                                   |
| Identificação de idioma   | Lingua                                                     |
| Keywords do currículo     | YAKE determinístico, sujeito a validação                   |
| Persistência dos recursos | Arquivos versionados, carregados localmente                |
| Atualização do índice     | Incremental, com reconstrução controlada                   |

**Uma mudança importante:** não será necessário construir um corpus genérico de livros, artigos ou textos da internet para calcular o IDF. O próprio conjunto global de vagas será o corpus documental do BM25.

Isso fará o ranking refletir a distribuição de termos nas oportunidades efetivamente coletadas pelo OpenJob.

### **Recursos NLP: o que importar**

Não precisamos desenvolver listas linguísticas manualmente.

**Stopwords — Stopwords ISO**

Fonte de dados candidata

[stopwords-iso](https://github.com/stopwords-iso/stopwords-iso?utm_source=chatgpt.com)

Coleção com listas para vários idiomas, incluindo português, inglês e espanhol. Possui licença MIT e documentação das fontes utilizadas.&#x20;

[image](https://www.google.com/s2/favicons?domain=https://github.com\&sz=32)

GitHub+1

**Decisão:** importar snapshots das listas, preservar créditos, versão e checksum. Não buscar atualizações automaticamente durante a execução do OpenJob.

Atenção: é uma coleção comunitária, não um padrão linguístico oficial. As listas devem passar por testes e revisão, principalmente para preservar negações.

**Identificação de idioma — Lingua Go**

[pemistahl/lingua-go](https://github.com/pemistahl/lingua-go?utm_source=chatgpt.com)

Suporta 75 idiomas, funciona offline e utiliza métodos estatísticos e baseados em regras. O projeto informa capacidade de classificação inclusive em textos curtos.&#x20;

[image](https://www.google.com/s2/favicons?domain=https://github.com\&sz=32)

GitHub+1

**Decisão:** avaliar carregamento restrito aos idiomas usados inicialmente pelo OpenJob, especialmente português, inglês e espanhol.

**Stemming — Snowball**

[freeeve/go-stemmers](https://github.com/freeeve/go-stemmers?utm_source=chatgpt.com)

Implementa 18 algoritmos Snowball, incluindo português, inglês e espanhol, com testes de conformidade em relação à implementação de referência.&#x20;

[image](https://www.google.com/s2/favicons?domain=https://github.com\&sz=32)

GitHub+1


**Decisão:** manter como candidato preferencial, fixar versão e executar os testes de conformidade e performance. O projeto tem uma comunidade pequena, então não será tratado como dependência inquestionável.

**Keywords — YAKE**

[phrozen/yake](https://github.com/phrozen/yake?utm_source=chatgpt.com)

Implementação Go de extração não supervisionada de palavras-chave, baseada em características estatísticas do próprio documento. Não precisa de corpus global nem de vocabulário profissional.&#x20;

[image](https://www.google.com/s2/favicons?domain=https://github.com\&sz=32)

phrozen/yake · GitHub

**Decisão:** fazer uma prova de qualidade antes de incorporá-la. Apesar de apresentar testes de validação cruzada, o repositório tem adoção pública ainda muito pequena. Não assumiria sua maturidade apenas pela documentação.

A regra geral será: **bibliotecas fornecem os algoritmos; arquivos externos fornecem os dados; o código do OpenJob fornece as interfaces e a orquestração.**

As versões utilizadas ficarão fixadas. Atualizações acontecerão por uma tarefa explícita de manutenção, com testes e revisão de licença.

# **Fase 1 — SQLite WAL, FTS5, deduplicação e indexação**

## **Objetivo**

Transformar o armazenamento de vagas em uma coleção global pesquisável e estatisticamente consistente.

### **1.1. Preparação do SQLite**

Primeiro, será necessário verificar se a versão atual de `modernc.org/sqlite` está compilada com suporte ao FTS5.

A documentação oficial exige que o SQLite seja compilado com `SQLITE_ENABLE_FTS5`, ou que a extensão seja carregada separadamente.&#x20;

[image](https://www.google.com/s2/favicons?domain=https://www.sqlite.org\&sz=32)

sqlite.org+1



Não devemos presumir que o driver atual possui o recurso habilitado.

**Critério técnico obrigatório:** executar um teste de integração com:

```
CREATE VIRTUAL TABLE fts_smoke_test
USING fts5(content);
```

Caso não funcione, avaliar uma versão/build compatível do driver antes de implementar as migrações de produção.

Não recomendo trocar o driver ou introduzir CGO sem antes verificar essa compatibilidade, pois o OpenJob hoje utiliza SQLite sem CGO.

### **Arquivos envolvidos**

| Arquivo/local                       | Trabalho                                           |
| ----------------------------------- | -------------------------------------------------- |
| `backend/go.mod`                    | Verificar compatibilidade do driver e dependências |
| `backend/internal/store/`           | Configuração central do banco                      |
| `backend/internal/config/config.go` | Configurações de banco, quando necessário          |
| `backend/internal/store/*_test.go`  | Testes WAL, FTS5 e concorrência                    |

### **Critérios de aceite**

- WAL habilitado.
- Foreign keys habilitadas.
- Busy timeout configurado.
- FTS5 disponível no driver utilizado.
- Múltiplas leituras concorrentes.
- Escritas serializadas corretamente.
- Backup e restauração preservando o índice.

## **1.2. Deduplicação global das vagas**

A tabela de vagas deve representar a vaga em si, não a descoberta individual feita por cada usuário.

Modelo proposto:

```
jobs
├── id
├── canonical_key
├── canonical_url
├── title
├── company
├── location
├── requirements
├── responsibilities
├── description
├── content_hash
├── language
├── created_at
├── updated_at
└── collected_at
```

Uma tabela complementar manterá a procedência:

```
job_sources
├── id
├── job_id
├── source_name
├── source_url
├── source_external_id
├── first_seen_at
└── last_seen_at
```

E outra manterá a associação com as buscas dos usuários:

```
user_job_discoveries
├── user_id
├── job_id
├── search_run_id
├── discovered_at
└── discovery_keywords
```

A regra é:

- A mesma oportunidade descoberta por diferentes usuários continua sendo uma única vaga global.
- Fontes diferentes que apontam para a mesma oportunidade são preservadas.
- Cada usuário mantém sua própria associação de descoberta.

A deduplicação terá duas estratégias:

1. URL canônica normalizada, quando disponível.
2. Fingerprint de conteúdo, como alternativa conservadora.

Não usaremos apenas título e empresa, pois vagas diferentes podem compartilhar esses dados.

A migração deverá analisar e consolidar os dados existentes sem perder associações, histórico de descoberta ou referências utilizadas por outros módulos.

## **1.3. Índice FTS5**

Não recomendo indexar toda a vaga como uma única coluna.

O índice deverá possuir campos independentes:

```
jobs_fts
├── title
├── requirements
├── responsibilities
└── description
```

Pesos iniciais para experimentação:

| Campo             | Peso inicial |
| ----------------- | ------------ |
| Título            | 4,0          |
| Requisitos        | 3,0          |
| Responsabilidades | 2,0          |
| Descrição geral   | 1,0          |

O FTS5 suporta esses pesos diretamente em sua função `bm25()`.&#x20;

[image](https://www.google.com/s2/favicons?domain=https://www2.sqlite.org\&sz=32)

www2.sqlite.org+1

### **Estrutura de persistência do índice**

Recomendo uma tabela intermediária:

```
job_search_documents
├── job_id
├── title_normalized
├── requirements_normalized
├── responsibilities_normalized
├── description_normalized
├── analyzer_version
├── content_hash
└── indexed_at
```

A tabela virtual FTS5 usará esses documentos como conteúdo externo.

Isso permite:

- Preservar o texto original das vagas.
- Indexar somente o texto normalizado.
- Reprocessar vagas quando a versão NLP mudar.
- Recriar o índice sem modificar a origem dos dados.

Triggers manterão o índice sincronizado com `job_search_documents`, dentro das transações de escrita. O FTS5 documenta esse padrão de external content e os cuidados de sincronização.&#x20;

[image](https://www.google.com/s2/favicons?domain=https://sqlite.org\&sz=32)

sqlite.org+1

### **Arquivos propostos**

```
backend/internal/store/
├── migrations/
│   ├── 00XX_global_job_dedup.sql
│   ├── 00XX_job_search_documents.sql
│   └── 00XX_jobs_fts.sql
│
├── jobs.go
├── job_sources.go
├── job_discoveries.go
└── search_index.go
```

Os números das migrations serão definidos conforme a sequência atual do projeto.

### **Critérios de aceite da Fase 1**

- Duplicatas consolidadas sem perda de relacionamentos.
- Vagas existentes indexadas durante a migração.
- Novas vagas indexadas incrementalmente.
- Alterações de descrição atualizam o índice.
- Exclusões removem o documento correspondente.
- Reindexação completa pode ser executada mais de uma vez sem duplicar documentos.
- O ranking considera todas as vagas globais distintas.

# **Fase 2 — Normalização linguística, corpus versionado e keywords determinísticas**

## **Objetivo**

Criar uma camada linguística única, reaproveitável tanto pela descoberta quanto pelo ranking.

Essa camada não poderá conhecer profissões, competências técnicas ou setores econômicos.

## **2.1. Diretório de recursos NLP**

Estrutura sugerida:

```
backend/internal/nlp/
├── analyzer.go
├── language.go
├── normalizer.go
├── stemmer.go
├── stopwords.go
├── resources.go
├── keyword_extractor.go
│
├── assets/
│   ├── manifest.json
│   ├── licenses/
│   │   ├── stopwords-iso.txt
│   │   ├── go-stemmers.txt
│   │   └── lingua-go.txt
│   │
│   ├── stopwords/
│   │   ├── pt.txt
│   │   ├── en.txt
│   │   └── es.txt
│   │
│   └── testdata/
│       ├── pt.jsonl
│       ├── en.jsonl
│       └── es.jsonl
│
└── *_test.go
```

As listas serão importadas das fontes externas, mas distribuídas com o aplicativo em snapshots versionados.

Um arquivo `manifest.json` registrará:

```
{
  "version": "1.0.0",
  "languages": ["pt", "en", "es"],
  "sources": [],
  "checksums": {},
  "normalizer_version": "1"
}
```

Não haverá dependência de internet para executar o aplicativo.

### **Como trocar o fornecedor dos recursos**

Será criado um contrato:

```
type ResourceProvider interface {
    Load(ctx context.Context) (*ResourceSet, error)
}
```

Implementações:

```
EmbeddedResourceProvider
DirectoryResourceProvider
```

A origem será selecionada na inicialização.

Assim, trocar Stopwords ISO por outra coleção não exigirá mudanças no algoritmo de comparação.

## **2.2. Pipeline de normalização**

Todos os textos passarão pelo mesmo processamento:

```
Texto original
     ↓
Normalização Unicode
     ↓
Identificação do idioma
     ↓
Tokenização
     ↓
Stopwords
     ↓
Stemming
     ↓
Tokens normalizados
```

O processamento será idêntico na indexação das vagas e na criação das consultas a partir do currículo.

Regras importantes:

- Idioma identificado individualmente por documento.
- Preservação de negações.
- Nenhum dicionário específico de computação ou recrutamento.
- Nenhuma substituição automática de palavras por supostos sinônimos.
- Tratamento explícito de textos muito curtos ou idioma desconhecido.
- Mesma versão de normalização para documentos e consultas.

**Não utilizar o Porter do FTS5 como substituto do Snowball**, pois ele é focado em inglês. A normalização deverá ocorrer previamente em Go, e o FTS5 receberá os tokens já processados, utilizando seu tokenizador Unicode.&#x20;

[image](https://www.google.com/s2/favicons?domain=https://www2.sqlite.org\&sz=32)

www2.sqlite.org+1



## **2.3. Versionamento e reconstrução do corpus**

Aqui, corpus significa a coleção global de vagas únicas indexadas.

Serão adicionadas informações de versão do índice:

```
search_index_metadata
├── generation
├── document_count
├── analyzer_version
├── stopwords_version
├── stemmer_version
├── indexed_at
└── last_rebuild_at
```

Quando uma versão do analisador mudar:

1. O OpenJob identifica a incompatibilidade.
2. Gera novamente os documentos normalizados.
3. Reconstrói o índice FTS5.
4. Confirma a integridade da coleção.
5. Atualiza a geração ativa.

Isso é fundamental porque uma mudança na normalização dos termos exige reindexar os documentos e processar as consultas usando as mesmas regras.

## **2.4. Extração determinística das keywords do currículo**

O método atual baseado em LLM será substituído por um extrator determinístico.

Proponho a seguinte interface:

```
type KeywordExtractor interface {
    Extract(ctx context.Context, text string, limit int) ([]Keyword, error)
}
```

O candidato principal é YAKE, por trabalhar com um único documento e utilizar características estatísticas locais, sem depender de um corpus global ou vocabulário profissional.&#x20;

[image](https://www.google.com/s2/favicons?domain=https://github.com\&sz=32)

phrozen/yake · GitHub



Entretanto, a biblioteca Go encontrada ainda tem pouca adoção pública. Por isso, a implementação terá uma etapa de comparação com a implementação de referência e um conjunto de testes próprio.

O extrator deverá:

- Identificar palavras e expressões relevantes.
- Agrupar variações morfológicas.
- Remover duplicações próximas.
- Preservar a forma original da expressão.
- Retornar uma pontuação interna de importância.
- Ser determinístico para os mesmos textos e versões de recursos.

O limite atual de cinco keywords continuará sendo respeitado na descoberta, mas deverá ser configurável.

**Importante:** as keywords extraídas não serão consideradas evidência de domínio ou proficiência. Elas serão apenas termos que ajudam a descobrir oportunidades.

### **Arquivos a integrar**

| Arquivo existente                            | Mudança planejada                              |
| -------------------------------------------- | ---------------------------------------------- |
| `backend/internal/discovery/resumefilter.go` | Substituir extração heurística pela camada NLP |
| `backend/internal/discovery/orchestrator.go` | Usar o novo extrator                           |
| `backend/internal/config/config.go`          | Configuração de recursos e versões             |
| `backend/internal/config/config_test.go`     | Testar configurações e valores padrão          |
| `backend/internal/api/server.go`             | Carregar e injetar os recursos no startup      |

### **Critérios de aceite da Fase 2**

- Nenhuma lista de stopwords hardcoded nos algoritmos.
- Recursos NLP com proveniência, licença e checksum.
- Aplicação funciona offline.
- Stemming consistente na indexação e consulta.
- Idiomas desconhecidos têm fallback controlado.
- Extração determinística sem LLM.
- Golden tests em português, inglês e espanhol.
- Testes com textos de diferentes áreas profissionais.
- Reindexação completa ao mudar a versão do analisador.

# **Fase 3 — BM25 ponderado e comparação currículo-vaga**

## **Objetivo**

Substituir o cálculo artesanal atual pelo ranking BM25 baseado na coleção real de vagas indexadas.

### **3.1. Separação entre descoberta e relevância**

Esta distinção será mantida durante toda a implementação.

| Elemento                              | Responsabilidade                  |
| ------------------------------------- | --------------------------------- |
| Keywords manuais do usuário           | Definir intenção de descoberta    |
| Keywords determinísticas do currículo | Ampliar a coleta de vagas         |
| Currículo completo                    | Fonte da consulta de relevância   |
| Vagas coletadas                       | Documentos candidatos ao ranking  |
| Corpus global de vagas                | Estatísticas utilizadas pelo BM25 |

A busca recupera as oportunidades usando os parâmetros definidos pelo usuário.

O ranking compara essas oportunidades com o currículo completo, usando a coleção global como base estatística.

Isso evita atribuir uma nota alta somente porque uma vaga foi coletada usando a mesma keyword que o usuário digitou.

## **3.2. Fluxo de scoring**

**Currículo completo**

**Analyzer NLP**

Idioma, normalização e extração de termos

**Consulta FTS5**

Termos e expressões relevantes do currículo

**BM25 com pesos por campo**

Título, requisitos, responsabilidades e descrição

**Ranking de vagas**

Score técnico, termos correspondentes e lacunas

A consulta será construída de maneira segura, evitando interpretar caracteres do currículo como comandos da sintaxe MATCH do FTS5.

O mecanismo deverá tratar consultas curtas, termos repetidos, expressões com mais de uma palavra e currículos extensos.

## **3.3. BM25 com pesos**

A função do FTS5 permite ponderar os campos diretamente.

Exemplo conceitual:

```
SELECT
    rowid,
    bm25(
        jobs_fts,
        4.0,
        3.0,
        2.0,
        1.0
    ) AS relevance
FROM jobs_fts
WHERE jobs_fts MATCH ?
ORDER BY relevance ASC
LIMIT ?;
```

A ordenação é crescente porque o FTS5 retorna valores numericamente menores para correspondências melhores.&#x20;

[image](https://www.google.com/s2/favicons?domain=https://www2.sqlite.org\&sz=32)

www2.sqlite.org



### **Um cuidado sobre o score de 0 a 100**

O valor bruto do BM25 não é um percentual de aderência.

Portanto, teremos dois resultados distintos:

```
raw_bm25
    Pontuação original de relevância para ordenação.

fit_score
    Representação calibrada para a interface,
    caso seja possível validá-la com exemplos rotulados.
```

Não quero simplesmente aplicar uma transformação matemática arbitrária ao BM25 para continuar exibindo um percentual aparentemente preciso.

O conjunto de exemplos avaliados manualmente será utilizado para estabelecer os critérios de calibração e verificar se a pontuação apresentada corresponde à percepção de aderência.

Se a calibração não tiver sustentação suficiente, o produto deverá preservar o score de relevância bruto e exibir uma classificação de aderência, sem alegar que a nota representa uma probabilidade.

## **3.4. Explicabilidade**

A função deverá retornar informações suficientes para explicar o resultado:

```
type RankingResult struct {
    JobID              int64
    RawBM25            float64
    FitScore           *int
    MatchingTerms      []string
    MissingTerms       []string
    MatchedFields      []string
    AnalyzerVersion    string
    CorpusGeneration   int64
}
```

Esses campos poderão ser apresentados posteriormente na interface, sem precisar reprocessar os textos.

Não serão adicionados fatores comportamentais neste momento.

## **3.5. Arquivos previstos**

```
backend/internal/ranking/
├── service.go
├── bm25.go
├── query_builder.go
├── result.go
├── calibration.go
├── explain.go
└── *_test.go
```

Integrações:

| Arquivo                                      | Alteração                                        |
| -------------------------------------------- | ------------------------------------------------ |
| `backend/internal/api/jobs.go`               | Usar o serviço determinístico                    |
| `backend/internal/tasks/handlers.go`         | Aplicar a mesma regra no worker                  |
| `backend/internal/api/server.go`             | Injetar o serviço                                |
| `backend/internal/score/`                    | Preservar compatibilidade ou migrar consumidores |
| `backend/internal/discovery/orchestrator.go` | Entregar currículo e vagas ao ranking            |

O serviço de ranking será único. API e workers não deverão implementar versões diferentes do cálculo.

### **Critérios de aceite da Fase 3**

- BM25 calculado sobre vagas únicas globais.
- Pesos por campo testados.
- Currículo completo utilizado como consulta.
- Resultado determinístico.
- Ranking consistente entre API e workers.
- Explicabilidade dos resultados.
- Golden tests com diferentes profissões e setores.
- Benchmarks de indexação, consulta e ranking.
- Compatibilidade com os limites e fluxos atuais de scoring.
- Nenhuma dependência de LLM no cálculo determinístico.

# **4. Sequência geral de trabalho**

Esta é a ordem que eu usaria para executar a Parte 1:

1. **Auditoria e prova de compatibilidade do FTS5**

   Validar a versão real do SQLite e o comportamento do WAL no driver existente.
2. **Migrations e deduplicação global**

   Consolidar vagas, fontes e associações de usuários sem perder dados.
3. **Índice FTS5 incremental**

   Criar a indexação por campos, sincronização e reconstrução.
4. **Camada NLP externa**

   Importar e versionar stopwords, configurar identificação de idioma e stemming.
5. **Keywords determinísticas**

   Validar YAKE e substituir o extrator LLM no fluxo de descoberta.
6. **Ranking BM25 ponderado**

   Construir consulta pelo currículo, executar ranking e gerar explicabilidade.
7. **Integração e validação final**

   Unificar API e workers, validar score, performance e regressões.
8. **Documentação do release**

   Registrar ADR, migrations, versões NLP, comandos operacionais e critérios de aceite.

# **5. Gates de qualidade do release**

Não considero essa entrega pronta apenas porque os testes unitários passaram.

| Gate              | Critério                                                               |
| ----------------- | ---------------------------------------------------------------------- |
| Banco             | WAL, FTS5, backup e recuperação validados                              |
| Deduplicação      | Nenhuma duplicação global indevida                                     |
| Indexação         | Inserção, atualização, exclusão e rebuild                              |
| NLP               | Recursos externos versionados e auditáveis                             |
| Idiomas           | PT, EN e ES testados                                                   |
| Keywords          | Resultado determinístico sem LLM                                       |
| Ranking           | BM25 ponderado validado em diferentes áreas                            |
| Reprodutibilidade | Mesma versão de documentos, recursos e consulta produz o mesmo ranking |
| Performance       | Benchmark com volume representativo de vagas                           |
| Integração        | API e worker utilizam o mesmo serviço                                  |

Eu criaria um conjunto de dados de avaliação com vagas de pelo menos estas áreas: computação, administração, engenharia, educação, saúde, finanças, logística e marketing.

Isso é necessário para verificar se o mecanismo está realmente generalista.

Também incluiria currículos e vagas com diferentes idiomas, tamanhos e estilos de escrita.

# **6. O que fica explicitamente fora desta entrega**

**Parte 2 — Backlog futuro**

Não implementar agora

**Fase 4 — Timeline comportamental**

Impressões, dwell time, abertura de detalhes, likes, dislikes e eventos do pipeline.

**Fase 5 — Personalização e qualidade contínua**

Preferências aprendidas, sinais de comportamento, Learning to Rank, avaliação de modelos e métricas de qualidade de recomendação.

A arquitetura da Parte 1 deverá deixar espaço para esses módulos, mas não precisaremos modificar agora toda a estrutura de eventos, interface de timeline ou pipeline seletivo.

## **7. Entregáveis da Parte 1**

Ao final, espero ter estes resultados no repositório:

| Entregável        | Resultado esperado                           |
| ----------------- | -------------------------------------------- |
| Migrations SQLite | Modelo global de vagas e índice FTS5         |
| NLP resources     | Recursos linguísticos externos e versionados |
| Analyzer          | Pipeline único de normalização               |
| Keyword extractor | Extração determinística                      |
| Search index      | Índice incremental e reconstruível           |
| Ranking service   | BM25 ponderado e explicável                  |
| Golden tests      | Validação generalista                        |
| Benchmarks        | Medição de desempenho                        |
| ADR               | Justificativas das decisões arquiteturais    |
| Release plan      | Parte 1 documentada e critérios cumpridos    |

## **Minha recomendação de execução**

Eu começaria pela **prova de compatibilidade do FTS5 com o driver atual**, antes de alterar qualquer migration. É o único ponto que pode exigir uma decisão estrutural sobre a camada de banco.

Em paralelo, manteria a seleção dos recursos NLP como uma pequena prova técnica: Stopwords ISO + Lingua + Snowball + avaliação do YAKE Go. Os recursos serão fixados e auditados antes de entrarem na versão final.

**O resultado esperado da Parte 1 é um mecanismo determinístico de descoberta e ranking baseado nas vagas reais do OpenJob, com corpus dinâmico, normalização generalista e pontuação explicável.**