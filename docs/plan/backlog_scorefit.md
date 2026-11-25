# **Plano de implementação — Parte 2 do OpenJob**



Backlog futuro

**Parte 2: Timeline comportamental, personalização e avaliação contínua do ranking.**



Esta etapa será executada depois da conclusão da Parte 1, quando o OpenJob já tiver:

- Coleção global de vagas deduplicadas.
- SQLite WAL e FTS5 operacionais.
- Normalização linguística determinística.
- Keywords extraídas do currículo sem LLM.
- BM25 ponderado integrado à API e aos workers.
- Versões identificáveis do corpus, analisador e ranking.

O objetivo da Parte 2 será transformar o mecanismo de busca em um sistema de recomendação que aprende progressivamente com as preferências de cada usuário.

Vou manter as duas fases que definimos:

1. Timeline comportamental e coleta de sinais de relevância.
2. Personalização por usuário e avaliação contínua da qualidade do ranking.

A arquitetura continuará utilizando SQLite. Com o cenário de um usuário normalmente e até quatro simultâneos, não há justificativa para introduzir Kafka, Redis, MongoDB ou outro banco exclusivamente para capturar interações.

# **1. Decisões arquiteturais**

Antes de detalhar as implementações, existem algumas decisões que considero fundamentais.

## **1.1. Separar exposição, interesse e resultado**

Nem toda interação representa a mesma coisa.

Uma vaga visualizada, uma vaga favoritada e uma vaga que resultou em contratação são evidências diferentes.

**Exposição**

Oportunidade apresentada

A vaga apareceu na timeline e teve oportunidade real de ser vista.

**Interesse implícito**

Comportamento observado

Tempo ativo no card, abertura dos detalhes e leitura da descrição.

**Interesse explícito**

Preferência declarada

Like, dislike, salvar ou remover dos favoritos.

**Comprometimento**

Ação concreta

Inclusão no pipeline, candidatura e progressão no processo seletivo.

**Resultado**

Desfecho observado

Entrevista, proposta, contratação, reprovação ou desistência.

Essas categorias não devem ser transformadas automaticamente em uma única nota.

Por exemplo, a ausência de clique não significa desinteresse. O usuário pode não ter visto a vaga, pode não ter tido tempo ou simplesmente não ter chegado até ela.

O registro de exposição e posição é especialmente importante porque interações são influenciadas pela ordenação anterior do próprio sistema. Esse fenômeno é conhecido como *position bias*.&#x20;

[image](https://www.google.com/s2/favicons?domain=https://www.microsoft.com\&sz=32)

Microsoft Research+1



## **1.2. Separar o registro dos eventos do estado atual**

Minha recomendação é adotar dois modelos complementares:

- **Event log:** histórico imutável das interações.
- **Projeções de estado:** representação atual das preferências e do relacionamento do usuário com cada vaga.

Assim, quando o usuário clicar duas vezes no coração, por exemplo, não teremos dois likes ativos. Teremos o histórico dos cliques e um estado atual consistente.

O histórico poderá ser reproduzido para reconstruir estados e recalcular os sinais de personalização quando o algoritmo evoluir.

## **1.3. Não usar ML nesta etapa**

Esse ponto é importante.

Com normalmente um usuário e no máximo quatro pessoas da família, teremos uma quantidade pequena de dados e usuários muito correlacionados.

Não considero apropriado introduzir collaborative filtering, redes neurais ou treinamento de Learning to Rank agora.

Uma revisão sobre avaliação de sistemas de recomendação destaca justamente as dificuldades de obter dados representativos e o volume necessário para certas abordagens colaborativas.&#x20;

[image](https://www.google.com/s2/favicons?domain=https://onlinelibrary.wiley.com\&sz=32)

Wiley Online Library+1



Começaremos com personalização determinística, explicável e individual.

# **Fase 4 — Timeline e captura de sinais comportamentais**

## **Objetivo**

Transformar a timeline em uma fonte confiável de dados sobre como cada usuário interage com as oportunidades que recebe.

A implementação terá três componentes principais:

1. Instrumentação do frontend.
2. Persistência dos eventos no backend.
3. Projeções e estados de interação no SQLite.

## **2.1. Instrumentação da timeline**

A captura deverá utilizar os recursos nativos do navegador, sem bibliotecas pesadas de analytics.

Recomendo:

- `IntersectionObserver` para identificar cards visíveis.
- `document.visibilityState` para detectar abas ocultas.
- `focus` e `blur` para verificar atividade da janela.
- Relógio monotônico para calcular durações.
- Envio agrupado de eventos ao backend.

### **Definição de impressão qualificada**

Um card não deve ser contabilizado como impressão simplesmente porque foi carregado no DOM.

Minha proposta inicial:

**Condições para registrar uma impressão**

Pelo menos 50% da área do card visível.

Visibilidade mantida por pelo menos 1 segundo.

Aba do navegador visível e janela ativa.

**Classificação:** impressão qualificada.

Os valores serão parâmetros configuráveis e deverão passar por validação com o comportamento real da timeline.

### **Tempo de permanência**

Não recomendo enviar um evento a cada segundo de permanência.

O frontend acumulará o tempo ativo por card e enviará um resumo associado à impressão.

Faixas iniciais para análise:

| Tempo ativo         | Interpretação inicial |
| ------------------- | --------------------- |
| 1–3 segundos        | Exposição breve       |
| 3–10 segundos       | Leitura rápida        |
| 10–30 segundos      | Possível interesse    |
| Mais de 30 segundos | Leitura prolongada    |

Essas faixas não serão convertidas diretamente em recompensas. Permanecer na tela não significa necessariamente interesse.

Também precisamos distinguir:

- Tempo no card.
- Tempo nos detalhes da vaga.
- Tempo com a página ativa.
- Tempo com o conteúdo efetivamente visível.

Não coletaremos movimentos de mouse, histórico geral de navegação ou teclas digitadas.

### **Abertura dos detalhes**

A abertura será um sinal independente do tempo de permanência.

Uma pessoa pode abrir a vaga, ler rapidamente os requisitos e sair. Isso deve ser registrado como uma interação diferente de simplesmente deixar o card na tela.

## **2.2. Modelo de eventos**

Minha proposta é criar um log imutável com eventos tipados.

### **Eventos de exposição**

```
feed.impression
feed.card_dwell
job.details_opened
job.details_dwell
```

### **Eventos de preferência**

```
job.liked
job.unliked
job.disliked
job.undisliked
job.saved
job.unsaved
job.hidden
```

### **Eventos do pipeline**

```
pipeline.job_added
pipeline.application_submitted
pipeline.stage_changed
pipeline.application_withdrawn
pipeline.employer_rejected
pipeline.offer_received
pipeline.hired
```

Os eventos do pipeline deverão refletir os estados que realmente existirem no OpenJob. Não vamos criar um segundo fluxo de estados independente do pipeline atual.

## **2.3. Modelo de dados**

O banco principal continuará sendo SQLite.

Eu criaria as seguintes tabelas.

**A.&#x20;**`feed_sessions`

Uma sessão de navegação na timeline.

```
feed_sessions
├── id
├── user_id
├── started_at
├── ended_at
├── candidate_generation_version
└── ranking_version
```

**B.&#x20;**`feed_impressions`

Cada oportunidade qualificada como efetivamente exposta.

```
feed_impressions
├── id
├── session_id
├── user_id
├── job_id
├── search_run_id
├── displayed_position
├── candidate_set_id
├── base_bm25_score
├── displayed_score
├── visible_duration_ms
├── details_opened
├── details_duration_ms
├── analyzer_version
├── corpus_generation
├── ranking_version
└── displayed_at
```

Esse registro permitirá reconstruir o que foi apresentado, em qual posição e com qual versão do mecanismo.

**C.&#x20;**`interaction_events`

O histórico imutável.

```
interaction_events
├── id
├── event_uuid
├── user_id
├── job_id
├── impression_id
├── session_id
├── event_type
├── occurred_at
├── received_at
├── schema_version
└── payload_json
```

O `event_uuid` terá uma restrição de unicidade, garantindo idempotência quando o frontend tentar reenviar um evento.

**D.&#x20;**`user_job_feedback`

Estado atual da interação.

```
user_job_feedback
├── user_id
├── job_id
├── liked
├── disliked
├── saved
├── hidden
├── dislike_reason
└── updated_at
```

Restrição:

```
UNIQUE(user_id, job_id)
```

Like e dislike serão mutuamente exclusivos. Salvar será independente, pois guardar uma vaga não significa necessariamente gostar dela.

**E. Pipeline**

O pipeline existente deverá continuar como fonte da verdade dos estágios.

A alteração será registrar cada transição em `interaction_events`, e não duplicar seu estado em outra tabela.

### **Índices importantes**

```
CREATE INDEX idx_interaction_user_time
ON interaction_events(user_id, occurred_at);

CREATE INDEX idx_interaction_job
ON interaction_events(job_id);

CREATE INDEX idx_impressions_user_position
ON feed_impressions(user_id, displayed_position);

CREATE INDEX idx_feedback_user
ON user_job_feedback(user_id, liked, disliked);
```

Os índices definitivos serão ajustados com base nas consultas reais.

## **2.4. Comunicação frontend/backend**

Não recomendo uma requisição HTTP para cada evento.

Criaria um endpoint de ingestão em lote:

```
POST /api/feed/events
```

Payload conceitual:

```
{
  "session_id": "session-id",
  "events": [
    {
      "event_id": "unique-event-id",
      "type": "feed.impression",
      "job_id": 123,
      "impression_id": "impression-id",
      "occurred_at": "2026-11-24T15:30:00Z"
    },
    {
      "event_id": "another-event-id",
      "type": "job.details_opened",
      "job_id": 123,
      "impression_id": "impression-id",
      "occurred_at": "2026-11-24T15:30:05Z"
    }
  ]
}
```

O backend será responsável por:

- Validar o usuário pela sessão autenticada.
- Verificar o vínculo da impressão com usuário e vaga.
- Validar os tipos de eventos.
- Impedir duplicação.
- Limitar valores de duração.
- Atualizar as projeções de estado na mesma transação.

O frontend acumulará eventos e fará flush:

- Ao alcançar um limite de eventos.
- Em intervalos controlados.
- Quando a aba deixar de estar visível.
- Ao encerrar a sessão de timeline.

O `sendBeacon` poderá ser utilizado como mecanismo complementar de envio, com fallback apropriado. Não dependeremos exclusivamente de `beforeunload`.

## **2.5. Controles da timeline**

A interface terá três ações distintas.

**Gostei**

Preferência positiva explícita.

**Não gostei**

Preferência negativa, opcionalmente acompanhada do motivo.

**Salvar**

Guardar a oportunidade para consulta posterior.

Para o dislike, sugiro oferecer motivos opcionais:

- Área de atuação.
- Remuneração.
- Localização.
- Modalidade de trabalho.
- Senioridade.
- Requisitos incompatíveis.
- Não tenho interesse.
- Outro motivo.

Uma ação adicional, `Ocultar vaga`, será diferente de dislike. Ocultar significa que o usuário não deseja mais ver aquela oportunidade, enquanto dislike serve como sinal para a personalização.

### **Arquivos previstos da Fase 4**

```
backend/
├── internal/
│   ├── store/
│   │   ├── migrations/
│   │   │   ├── 00XX_feed_sessions.sql
│   │   │   ├── 00XX_feed_impressions.sql
│   │   │   ├── 00XX_interaction_events.sql
│   │   │   └── 00XX_user_job_feedback.sql
│   │   ├── interaction_events.go
│   │   ├── feed_impressions.go
│   │   └── user_job_feedback.go
│   │
│   ├── engagement/
│   │   ├── event.go
│   │   ├── collector.go
│   │   ├── validator.go
│   │   ├── service.go
│   │   └── service_test.go
│   │
│   └── api/
│       ├── feed_events.go
│       ├── feed_events_test.go
│       └── server.go
│
└── web/
    └── [componentes reais da timeline a localizar]
```

O pacote `internal/events` já existente no projeto deverá ser auditado antes. Caso ele represente apenas eventos internos da aplicação, não o transformaremos automaticamente em armazenamento de comportamento do usuário. Os dois conceitos terão responsabilidades separadas.

### **Critérios de aceite da Fase 4**

- Impressões qualificadas, não apenas cards carregados.
- Tempo ativo corretamente contabilizado.
- Eventos enviados em lote.
- Eventos duplicados não geram duplicação de estado.
- Like, dislike e save independentes.
- Descurtidas e remoções devidamente registradas.
- Mudanças no pipeline geram eventos.
- Histórico preservado.
- Posição e versão do ranking registradas.
- Nenhuma interação ausente é interpretada automaticamente como dislike.
- A captura não prejudica a performance da timeline.

# **Fase 5 — Personalização por usuário e avaliação contínua**

## **Objetivo**

Utilizar os sinais acumulados na Fase 4 para melhorar a ordenação das oportunidades de cada usuário, sem alterar o funcionamento básico do BM25.

A arquitetura deverá ter três estágios:

**1. Candidate generation**

Vagas encontradas a partir das keywords e do perfil

**2. Scoring**

Relevância textual calculada pelo BM25

**3. Personalized reranking**

Preferências, interações, diversidade e atualidade

**Timeline personalizada**

Ordem final das oportunidades para cada usuário

Essa separação segue uma arquitetura amplamente utilizada em sistemas de recomendação: geração de candidatos, scoring e reordenação. A camada final pode incorporar preferências e diversidade sem contaminar o mecanismo inicial de relevância.&#x20;

[image](https://www.google.com/s2/favicons?domain=https://developers.google.com\&sz=32)

developers.google.com



## **3.1. Perfil individual de preferências**

Cada usuário terá um perfil derivado das próprias interações.

Por exemplo:

```
user_preference_profiles
├── user_id
├── profile_version
├── analyzer_version
├── created_at
└── updated_at
```

E uma estrutura para as características aprendidas:

```
user_preference_features
├── user_id
├── feature_type
├── feature_key
├── positive_weight
├── negative_weight
├── observation_count
├── last_observed_at
└── profile_version
```

As características serão genéricas e derivadas do conteúdo das vagas.

Não existirão listas manuais de profissões ou de tecnologias.

### **Como os sinais influenciarão o perfil**

| Sinal                  | Papel na personalização                      |
| ---------------------- | -------------------------------------------- |
| Like                   | Forte preferência positiva                   |
| Dislike                | Forte preferência negativa                   |
| Save                   | Interesse moderado                           |
| Details opened         | Interesse implícito fraco                    |
| Dwell time             | Evidência complementar e limitada            |
| Pipeline added         | Intenção concreta                            |
| Application submitted  | Comprometimento concreto                     |
| Avanço para entrevista | Resultado positivo observado                 |
| Proposta / contratação | Resultado positivo relevante, mas contextual |

Ações repetidas sobre a mesma vaga não deverão inflar indefinidamente a preferência.

Por isso, serão utilizados pesos limitados, deduplicação lógica e decaimento temporal.

## **3.2. Decaimento temporal**

Preferências podem mudar.

Uma pessoa pode buscar oportunidades de gestão durante alguns meses e posteriormente passar a procurar vagas técnicas.

O sistema deverá dar maior importância aos sinais recentes.

Sugiro avaliar decaimento exponencial:

\\[ w(t)=w_0 \times 2^{-\frac{t}{h}} \\]

Onde:

- \\(w_0\\) é o peso original do sinal.
- \\(t\\) é o tempo transcorrido.
- \\(h\\) é o período de meia-vida.

A meia-vida será configurável e calibrada na avaliação. Não definirei um período fixo antes de observar o comportamento real.

## **3.3. Re-ranking determinístico**

O BM25 continuará calculando a relevância de conteúdo.

A personalização atuará sobre o resultado, considerando:

- Preferência positiva.
- Preferência negativa.
- Similaridade com oportunidades que receberam feedback.
- Localização e modalidade, caso sejam preferências já disponíveis.
- Atualidade da vaga.
- Vagas já ocultadas.
- Oportunidades que já estão em andamento no pipeline.

Uma estrutura conceitual:

\\[ S\_{final}=S\_{base}+\alpha C\_{personalização} +\beta C\_{preferências} +\gamma C\_{atualidade} \\]

Os fatores serão separados, explicáveis e configuráveis.

Não fixaria esses coeficientes antes de uma bateria de avaliação.

Também aplicaria um fator de confiança baseado na quantidade e qualidade das observações: quando um usuário ainda tiver poucos sinais, o ranking permanecerá próximo do BM25 original.

Isso evita que um único like altere radicalmente toda a timeline.

## **3.4. Tratamento especial do pipeline**

É importante diferenciar intenção, preferência e resultado.

Minha proposta:

| Situação                | Tratamento                                    |
| ----------------------- | --------------------------------------------- |
| Adicionou ao pipeline   | Forte intenção                                |
| Iniciou candidatura     | Intenção muito forte                          |
| Candidatou-se           | Ação concreta                                 |
| Avançou para entrevista | Resultado positivo                            |
| Recebeu proposta        | Resultado positivo forte                      |
| Contratado              | Resultado positivo importante                 |
| Desistiu                | Sinal contextual                              |
| Reprovado pela empresa  | Resultado do processo, não dislike automático |

Uma reprovação pode ter ocorrido por fatores externos ao candidato. Não será utilizada para concluir automaticamente que a vaga tinha baixa aderência.

Também não quero aumentar indefinidamente o ranking de vagas já candidatas. Ações do pipeline deverão influenciar o aprendizado de preferências, mas a interface poderá priorizar uma área própria para oportunidades em andamento.

## **3.5. Diversidade da timeline**

A personalização excessiva cria um problema: o OpenJob pode apresentar repetidamente oportunidades muito semelhantes.

Para evitar isso, a reordenação poderá considerar:

- Diversidade de empregadores.
- Diversidade de oportunidades.
- Atualidade.
- Repetição de vagas.
- Cobertura das preferências do usuário.

Esses fatores serão usados no reranking, não no BM25.

O objetivo não é fazer a timeline mostrar somente o que o usuário já demonstrou gostar, mas também oferecer oportunidades potencialmente relevantes que ele ainda não conhece.

## **3.6. Avaliação contínua do ranking**

Esta é uma parte fundamental da Fase 5.

O OpenJob precisa conseguir responder:

> A versão nova do ranking realmente está apresentando oportunidades melhores do que a versão anterior?

Para isso, registraremos:

- Versão do ranking.
- Versão do perfil de preferências.
- Versão do corpus.
- Posição da vaga.
- Score base.
- Score personalizado.
- Motivos que alteraram a posição.

### **Indicadores de avaliação**

| Indicador                 | O que mede                              |
| ------------------------- | --------------------------------------- |
| Qualified Impression Rate | Percentual de impressões qualificadas   |
| Details Open Rate         | Abertura de detalhes por impressão      |
| Like Rate                 | Likes por impressão                     |
| Dislike Rate              | Dislikes por impressão                  |
| Save Rate                 | Vagas salvas por impressão              |
| Pipeline Conversion       | Vagas adicionadas ao pipeline           |
| Application Rate          | Candidaturas por oportunidades expostas |
| Interview Progression     | Progressão para entrevista              |
| NDCG\@K                   | Qualidade da ordenação                  |
| Coverage                  | Diversidade da coleção apresentada      |

Não devemos otimizar somente cliques ou tempo de tela. Esses indicadores representam engajamento, não necessariamente satisfação ou qualidade do matching.

### **Como será avaliada a qualidade**

Eu dividiria a avaliação em três perspectivas.

**A. Qualidade textual**

Comparar a ordenação BM25 com julgamentos de relevância independentes.

**B. Preferência do usuário**

Verificar likes, dislikes, saves e inclusão no pipeline, sempre considerando as exposições.

**C. Resultado observado**

Analisar candidaturas, entrevistas, propostas e contratações separadamente.

Os dados comportamentais terão viés de exposição, então não podemos avaliar uma política nova apenas observando os cliques que ela mesma produziu. Precisamos preservar contexto, posição e política de exibição.&#x20;

[image](https://www.google.com/s2/favicons?domain=https://onlinelibrary.wiley.com\&sz=32)

Wiley Online Library+2



### **Métricas de ranking**

Para medir a qualidade da ordem, utilizaria:

- `Precision@K`
- `Recall@K`
- `NDCG@K`
- `MRR`

A NDCG é particularmente interessante porque permite atribuir diferentes níveis de relevância aos resultados, em vez de tratar todos os cliques da mesma forma.

A avaliação offline deverá considerar recorte temporal: treinar ou ajustar usando interações anteriores e avaliar em eventos posteriores. Não misturar registros futuros com o histórico usado para gerar o perfil.

# **4. Arquivos previstos para a implementação**

A proposta é manter as responsabilidades separadas.

```
backend/
├── internal/
│   ├── store/
│   │   ├── migrations/
│   │   │   ├── 00XX_feed_sessions.sql
│   │   │   ├── 00XX_feed_impressions.sql
│   │   │   ├── 00XX_interaction_events.sql
│   │   │   ├── 00XX_user_job_feedback.sql
│   │   │   └── 00XX_user_preference_features.sql
│   │   │
│   │   ├── interaction_events.go
│   │   ├── feed_impressions.go
│   │   ├── user_job_feedback.go
│   │   └── user_preference_features.go
│   │
│   ├── engagement/
│   │   ├── event.go
│   │   ├── collector.go
│   │   ├── validator.go
│   │   ├── service.go
│   │   ├── projector.go
│   │   └── *_test.go
│   │
│   ├── personalization/
│   │   ├── profile.go
│   │   ├── feature_builder.go
│   │   ├── preference_model.go
│   │   ├── reranker.go
│   │   ├── weights.go
│   │   ├── diversity.go
│   │   └── *_test.go
│   │
│   ├── evaluation/
│   │   ├── dataset.go
│   │   ├── metrics.go
│   │   ├── ndcg.go
│   │   ├── precision.go
│   │   ├── recall.go
│   │   ├── mrr.go
│   │   ├── report.go
│   │   └── *_test.go
│   │
│   └── api/
│       ├── feed_events.go
│       ├── feed.go
│       └── server.go
│
└── web/
    └── [estrutura atual da timeline]
        ├── impression tracking
        ├── dwell tracking
        ├── feedback controls
        └── event batching
```

Os nomes são uma estrutura-alvo. Antes da implementação, os arquivos existentes de timeline, pipeline, eventos e API serão auditados para reaproveitar serviços e não criar duplicações.

### **Responsabilidade de cada módulo**

| Módulo            | Responsabilidade                              |
| ----------------- | --------------------------------------------- |
| `engagement`      | Registrar e validar eventos                   |
| `store`           | Persistência e consultas                      |
| `personalization` | Construir perfis e reordenar vagas            |
| `evaluation`      | Medir qualidade e gerar relatórios            |
| `api`             | Expor ingestão de eventos e timeline          |
| `web`             | Capturar comportamento e apresentar controles |

# **5. Segurança, privacidade e operação**

Como o OpenJob roda localmente, o histórico comportamental continuará armazenado no notebook do usuário.

Ainda assim, a captura deve ser transparente.

Requisitos:

- Não armazenar conteúdo digitado ou navegação fora do OpenJob.
- Não registrar cada movimento de scroll.
- Permitir limpar o histórico comportamental de um usuário.
- Separar dados de diferentes usuários.
- Não expor eventos de um membro da família a outro.
- Não utilizar eventos individuais para treinar preferências de outro usuário.
- Permitir desabilitar a personalização comportamental.
- Manter os eventos necessários para reconstrução do estado.
- Documentar os tipos de sinais coletados.

O frontend não poderá informar `user_id` livremente no corpo da requisição e esperar que o backend confie nele. A identidade será obtida pela sessão autenticada.

### **Operação e performance**

A arquitetura não precisará de uma fila externa.

Para o volume esperado:

- Eventos enviados em lote.
- Transações SQLite curtas.
- Escrita idempotente.
- Índices por usuário, vaga e data.
- Projeções incrementais.
- Atualização do perfil em lotes ou de forma incremental.
- Nenhuma reconstrução completa do perfil a cada abertura da timeline.

A timeline deverá continuar responsiva mesmo quando o registro de eventos estiver indisponível temporariamente. A perda de uma métrica de dwell não poderá impedir a exibição das vagas.

# **6. Sequência de implementação**

Esta é a ordem recomendada para executar a Parte 2.

1. **Auditar timeline e pipeline existentes**

   Identificar componentes de UI, estados atuais das vagas, autenticação e transições do pipeline.
2. **Criar persistência de eventos**

   Migrations, event log, idempotência, sessões e impressões.
3. **Instrumentar frontend**

   IntersectionObserver, tempo ativo, abertura de detalhes e envio agrupado.
4. **Implementar os controles de preferência**

   Like, dislike, save, hide e motivos de feedback.
5. **Integrar pipeline ao histórico**

   Registrar transições e resultados reais sem duplicar os estados existentes.
6. **Construir perfil de preferências**

   Extrair características de vagas positivas e negativas, com recência e confiança.
7. **Implementar o re-ranker**

   Combinar o BM25 com sinais comportamentais de forma independente e explicável.
8. **Construir avaliação contínua**

   Métricas de relevância, engajamento, conversão e comparações de versões.
9. **Validar e documentar**

   Testar replay de eventos, consistência de perfis, privacidade, performance e regressões.

# **7. Gates de qualidade da Parte 2**

| Gate           | Critério de aceite                            |
| -------------- | --------------------------------------------- |
| Instrumentação | Exposições reais, não simples renderizações   |
| Dwell time     | Somente tempo ativo e visível                 |
| Ingestão       | Eventos idempotentes e em lote                |
| Preferências   | Like, dislike, save e hide independentes      |
| Histórico      | Eventos reproduzíveis e estado reconstruível  |
| Pipeline       | Transições registradas corretamente           |
| Personalização | Perfil isolado por usuário                    |
| Cold start     | BM25 continua predominando sem histórico      |
| Re-ranking     | Score base preservado e explicável            |
| Diversidade    | Timeline não fica presa a uma única categoria |
| Avaliação      | Métricas offline com separação temporal       |
| Privacidade    | Exportação, limpeza e desativação dos sinais  |

## **8. O que continuará fora do escopo**

Mesmo depois da Parte 2, eu manteria algumas evoluções em um backlog posterior:

| Evolução                              | Motivo                                                   |
| ------------------------------------- | -------------------------------------------------------- |
| Collaborative filtering               | Poucos usuários para aprender relações entre pessoas     |
| Learning to Rank com LambdaMART       | Necessita conjunto representativo de exemplos rotulados  |
| BPR e modelos de embeddings treinados | Necessitam histórico adequado de interações              |
| Exploração probabilística avançada    | Exige volume e avaliação cuidadosa de riscos             |
| Modelos de previsão de contratação    | Contratação depende de fatores que o OpenJob não observa |

O histórico criado nesta Parte 2 será a infraestrutura de dados necessária para avaliar essas alternativas futuramente.

# **9. Entregáveis finais da Parte 2**

| Entregável              | Resultado                          |
| ----------------------- | ---------------------------------- |
| Event log               | Histórico comportamental confiável |
| Feed impressions        | Registro contextual das exposições |
| Telemetria              | Dwell time e leitura de detalhes   |
| UI feedback             | Like, dislike, save e hide         |
| Pipeline events         | Histórico de progressão            |
| User preference profile | Perfil individual derivado         |
| Personalized reranker   | Ranking ajustado ao usuário        |
| Evaluation engine       | Métricas e relatórios              |
| Versioning              | Reprodutibilidade de resultados    |
| Documentação            | ADR e critérios de avaliação       |

## **10. Resultado esperado**

Ao final, o OpenJob deverá conseguir responder três perguntas diferentes:

1. **A vaga combina com o currículo?** — respondida pelo BM25 da Parte 1.
2. **O usuário demonstra interesse nesse tipo de oportunidade?** — respondida pelos sinais comportamentais da Parte 2.
3. **A ordenação está realmente melhorando?** — respondida pela camada de avaliação.

Essa distinção é essencial. Uma vaga pode ter alta aderência ao currículo e pouco interesse do usuário. Também pode despertar interesse, mas ter baixa aderência textual.

A plataforma precisa conhecer ambas as coisas sem confundi-las.

## **Decisão arquitetural da Parte 2**

**SQLite + Event Log + Projeções de estado + Perfil individual + Personalized Reranker + Evaluation Engine.**

Sem um novo banco de dados, sem treinamento de ML nesta fase e sem misturar comportamento do usuário às estatísticas globais de BM25.

**Nenhum código foi alterado.** Este é o planejamento do backlog da Parte 2, condicionado à conclusão da Parte 1.

Quando a Parte 1 estiver concluída, o próximo passo será transformar este backlog em tarefas técnicas menores, com dependências, migrations, arquivos, testes e critérios de aceite. Quer que eu te lembre de retomar esse planejamento nessa ocasião?

### **Referências técnicas**

- [Google — Recommendation Systems Overview](https://developers.google.com/machine-learning/recommendation/overview/types?utm_source=chatgpt.com): separação entre geração de candidatos, scoring e reranking.
- [Microsoft Research — Effects of Position Bias on Click-Based Recommender Evaluation](https://www.microsoft.com/en-us/research/publication/effects-of-position-bias-on-click-based-recommender-evaluation/?utm_source=chatgpt.com): viés de posição em interações.
- [Wiley — Offline Recommender System Evaluation: Challenges and New Directions](https://onlinelibrary.wiley.com/doi/full/10.1002/aaai.12051?utm_source=chatgpt.com): limitações e vieses na avaliação de recomendações.