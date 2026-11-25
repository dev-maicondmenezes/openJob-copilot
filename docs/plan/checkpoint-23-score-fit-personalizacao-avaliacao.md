# Checkpoint 23 — Score Fit, Parte 2: personalização individual e avaliação

> **Estimativa inicial:** a definir após auditoria dos eventos e projeções entregues pelo CP22, do pipeline F2/CP17, dos contratos do feed F1/CP16 e do ranking-base F5/B11. Não fixar prazo antes de confirmar a qualidade dos dados, a disponibilidade do contexto histórico e o corpus de avaliação.
> **Status:** proposto.
> **Pré-requisitos:** CP18 concluído nos pontos de identidade, autenticação, autorização e ownership; F5/B11 com candidate generation e score BM25 estáveis e versionados; F1/CP16 com feed integrado ao ranking-base; F2/CP17 com pipeline e transições persistidas; CP22 concluído e aprovado, com event log, projeções, contexto de exposição e isolamento por usuário; SQLite WAL/FTS5 e migrations atuais verdes.
> **Responsabilidade:** construir perfil comportamental individual e reproduzível, aplicar reranking determinístico depois do score-base BM25 e avaliar a qualidade da personalização sem confundir engajamento com aderência ou resultado profissional.
> **Relação com CP18:** identidade, sessão, autorização e ownership pertencem ao CP18. Todo perfil, feature, relatório e histórico comportamental deve ser derivado do usuário autenticado no backend.
> **Relação com CP22:** o CP22 é a fonte dos eventos de exposição, atenção, feedback e transições de pipeline. O CP23 consome seus eventos/projeções; não recria event log, captura de impressão ou controles de feedback.
> **Relação com F5/B11 — Score Fit Parte 1:** candidate generation, score BM25, corpus, analyzer e estatísticas globais permanecem independentes. O CP23 aplica personalização somente depois do ranking-base.
> **Relação com F1/CP16 e F2/CP17:** o feed apresenta o ranking resultante e o pipeline continua sendo a fonte da verdade para candidaturas e etapas. O CP23 não cria um segundo feed nem um segundo pipeline.
> **Fora deste checkpoint:** collaborative filtering, Learning to Rank, LambdaMART, BPR, embeddings treinados, exploração probabilística avançada e modelos de previsão de contratação.

## 1. Objetivo

Entregar personalização comportamental individual para o Score Fit — Parte 2. O sistema deverá utilizar sinais históricos de cada pessoa para reordenar vagas já recuperadas pelo mecanismo existente, sem alterar a geração de candidatos ou o score-base BM25 da Parte 1.

A personalização deverá considerar preferências positivas e negativas, similaridade entre características das vagas e vagas avaliadas, localização e modalidade quando disponíveis, atualidade, ocultações e contexto do pipeline. Os sinais precisam manter semânticas distintas: exposição não é interesse; dwell não é preferência explícita; salvar não é gostar; candidatura é uma ação/comprometimento; entrevista, oferta e contratação são resultados contextuais.

O perfil deve evoluir de forma determinística, com deduplicação lógica, limitação de ações repetidas sobre a mesma vaga e decaimento temporal calibrado por avaliação. A confiança da personalização deve depender da quantidade e qualidade das observações. No cold start ou quando os dados forem insuficientes, o resultado deve permanecer próximo do ranking-base BM25.

O Evaluation Engine deverá comparar versões do ranking com métricas offline e operacionais, controlar viés de posição e separar qualidade textual, preferência declarada/observada e resultados do pipeline. O sistema não deve otimizar somente cliques ou tempo de permanência.

**Princípio central:** personalização é uma camada individual de reranking e avaliação; não altera o matching textual global, não transfere sinais entre usuários e não transforma correlação comportamental em fato sobre a qualidade objetiva de uma vaga.

## 2. Escopo e decisões arquiteturais

### 2.1 Responsabilidades

- **CP18:** autenticação, identidade, sessão, autorização e ownership dos dados individuais.
- **F5/B11 — Score Fit Parte 1:** candidate generation, normalização linguística, keywords, BM25 ponderado, score-base e versões do corpus/analyzer/ranking-base.
- **CP22 — Timeline e captura comportamental:** event log, impressões, dwell, feedback explícito, sessões e eventos originados por transições reais do pipeline. O CP23 consome esses dados.
- **F1/CP16:** feed, cards, detalhes, paginação/navegação e apresentação do ranking.
- **F2/CP17:** candidaturas, pipeline e estados/transições oficiais. O CP23 interpreta sinais derivados desses eventos sem alterar a propriedade do domínio.
- **CP23:** derivação de features individuais, perfil de preferências, confiança, decaimento, reranking determinístico, explicações, Evaluation Engine, controles de privacidade e testes.
- **CP11:** instalação, bootstrap técnico, configuração da instância e operação do executável.

### 2.2 Fronteira entre score-base e personalização

- O score BM25 continua medindo aderência textual entre currículo e vaga, de acordo com a Parte 1.
- Candidate generation e score-base são etapas independentes da personalização.
- O CP23 recebe a lista de candidatos e seus scores-base; só então calcula a ordem personalizada.
- O CP23 não modifica corpus global, estatísticas BM25, keywords globais, pesos do analyzer ou score-base.
- O resultado deve preservar separadamente score-base, contribuição/score personalizado, versão do perfil e razões de reranking.
- Quando a personalização estiver desabilitada, indisponível ou sem confiança suficiente, o feed deve usar a ordenação-base ou uma aproximação controlada dela.
- Não recalcular o score-base para representar preferência individual nem persistir preferências pessoais em registros globais de vagas.

### 2.3 Determinismo, explicabilidade e versionamento

- Para o mesmo conjunto de candidatos, eventos, estado do pipeline e versões de regras/features, a derivação do perfil e a ordenação devem ser reproduzíveis.
- A ordenação não pode depender de aleatoriedade, de chamadas LLM ou de serviços externos.
- Cada versão de perfil, extrator de features, regras de agregação, decaimento, reranker e configuração deve ser identificável.
- As razões de reranking devem apontar fatores efetivamente utilizados, sem expor dados privados de outro usuário.
- Não fixar pesos, limiares, meia-vida ou coeficientes definitivos antes da avaliação offline.
- Configurações iniciais devem ser versionadas e calibráveis por experimento offline, não alteradas silenciosamente em produção.
- Reprocessamento de eventos deve produzir o mesmo perfil para a mesma janela de dados e versões de regras.

### 2.4 Semântica dos sinais

Manter dimensões diferentes no perfil e na avaliação:

1. **Exposição:** impressão qualificada e posição da vaga no feed. A exposição informa que o usuário teve oportunidade de ver a vaga; não informa interesse.
2. **Atenção implícita:** abertura de detalhes e dwell ativo/visível. É um sinal fraco de atenção, não uma preferência explícita.
3. **Preferência explícita:** like e dislike. São sinais direcionais e distintos; remoção/desfazer deve ser tratada conforme o estado atual e o histórico do CP22.
4. **Intenção/comprometimento:** salvar, incluir no pipeline e registrar candidatura. Cada ação mantém sua própria semântica; salvar ou candidatar-se não equivale automaticamente a like.
5. **Resultado contextual:** retirada, rejeição da empresa, entrevista, oferta e contratação. Esses eventos descrevem o percurso da candidatura; não são rótulos universais de preferência.

Não interpretar ausência de interação como dislike. Não converter automaticamente reprovação da empresa em avaliação negativa da vaga. Não tratar uma contratação como preferência permanente por qualquer vaga com atributos superficialmente semelhantes.

### 2.5 Arquitetura local-first

- Manter perfis, features, agregações e relatórios no SQLite local.
- Reutilizar WAL, migrations, índices e transações curtas existentes.
- Não introduzir Kafka, Redis, MongoDB, fila externa, serviço de analytics ou outro banco exclusivamente para personalização.
- Não reconstruir todo o perfil a cada abertura da timeline; atualizar incrementalmente ou em lotes controlados.
- Garantir idempotência na aplicação de eventos e na atualização de features.
- Degradar graciosamente para o ranking-base em caso de erro de leitura, perfil ausente ou serviço de personalização indisponível.

### 2.6 Privacidade e controle do usuário

- Todos os sinais e perfis pertencem exclusivamente ao usuário autenticado.
- O backend não aceita `user_id` livre do frontend como fonte de identidade.
- Sinais individuais não podem compor perfil, ranking ou estatística de outro usuário.
- O usuário deve poder exportar seu histórico comportamental e perfil derivado, limpar o histórico e desabilitar a personalização.
- Desabilitar personalização interrompe o uso de features individuais no reranking; não deve apagar automaticamente o histórico, salvo escolha explícita de limpeza.
- A limpeza deve remover ou invalidar os dados pessoais derivados e os eventos dentro do escopo escolhido, sem excluir vagas globais nem modificar dados de outros usuários.
- Não enviar histórico, features ou preferências para serviços externos.

## 3. Modelo de domínio e contratos

A estrutura final deve seguir as convenções reais do projeto e ser formalizada em migrations/contratos versionados. Os nomes abaixo são conceituais e podem ser ajustados à implementação existente desde que a semântica, ownership e capacidade de auditoria sejam preservados.

### 3.1 `user_preference_profiles`

Representa a versão consolidada do perfil comportamental de um usuário.

Campos mínimos esperados:

- identificador do perfil e `user_id` proprietário;
- versão do perfil e versão do schema;
- versão do extrator/normalizador de features;
- versão das regras de agregação e decaimento;
- estado de ativação da personalização;
- janela temporal ou referência de eventos considerados;
- quantidade/qualidade resumida de observações por dimensão;
- timestamps de criação e última atualização;
- estado de processamento/reconstrução, quando aplicável.

O perfil deve ser reconstruível a partir dos eventos elegíveis do CP22 e das versões identificadas das regras. Não armazenar no perfil uma cópia desnecessária do conteúdo integral de vagas, currículos ou payloads dos eventos.

### 3.2 `user_preference_features`

Representa características individuais derivadas das vagas e dos sinais do usuário.

Campos mínimos esperados:

- usuário proprietário e chave/tipo da feature;
- dimensão de origem, como preferência positiva, preferência negativa, atenção, intenção/comprometimento ou resultado contextual;
- força/sinal agregado positivo e negativo, sem colapsar dimensões de semântica diferente;
- quantidade de observações válidas e quantidade efetiva após deduplicação/limites;
- última observação e informação necessária ao decaimento;
- confiança ou qualidade da evidência;
- versão do extrator e das regras de agregação;
- timestamps de atualização.

As features devem derivar de características disponíveis no conteúdo das vagas e nos metadados confiáveis. Não depender de listas manuais fixas de profissões, tecnologias ou empresas. A taxonomia, normalização e extração devem reutilizar NLP Core e componentes existentes quando apropriado.

Exemplos de atributos podem incluir área, família de cargo, competências, senioridade explicitamente descrita, localização, modalidade e outros campos realmente disponíveis. A ausência de um atributo não deve ser interpretada como preferência negativa.

### 3.3 Origem e linhagem das features

Para permitir auditoria e reprocessamento, cada feature deve ser vinculável às classes de eventos e às vagas que contribuíram para sua derivação, por identificadores e versões suficientes. Evitar duplicar o event log do CP22. Quando for necessário armazenar agregados intermediários, registrar a janela, regra e origem utilizadas.

Uma feature derivada de `job.liked` deve permanecer distinguível de uma feature apoiada apenas em dwell ou em candidatura. A interface de explicação pode resumir o fator sem revelar payloads ou histórico desnecessário.

### 3.4 Contrato de entrada do reranker

O serviço de personalização recebe, no mínimo:

- usuário autenticado/contexto de ownership;
- candidatos já gerados pela Parte 1;
- identificador de cada vaga;
- score-base BM25 e posição/ordem-base;
- versão do corpus, analyzer e ranking-base;
- atributos normalizados da vaga que estejam realmente disponíveis;
- perfil individual ativo e suas versões;
- estado de ocultação/feedback/pipeline necessário à decisão.

O reranker não busca candidatos fora do conjunto recebido e não recalcula BM25. Dados ausentes são representados como indisponíveis, nunca preenchidos com suposições.

### 3.5 Contrato de saída do reranker

Para cada candidato, disponibilizar:

- ID da vaga e score-base original;
- score/ordem personalizada, se a personalização foi aplicada;
- posição de origem e posição final;
- identificador/versão do perfil utilizado;
- versão do reranker e configuração aplicada;
- fatores considerados e razões concisas de inclusão, redução ou manutenção da posição;
- indicação de fallback/cold start, quando aplicável.

O contrato deve preservar a rastreabilidade entre saída e entrada. Não sobrescrever score-base ou metadados históricos.

### 3.6 Contexto histórico de exposição

O CP23 deve estender os contratos do CP22 apenas quando necessário para registrar:

- versão do reranker e da configuração;
- versão do perfil comportamental;
- score personalizado efetivamente exibido;
- posição final e fatores/razões de reranking;
- indicação de personalização ativa, fallback ou cold start.

Reutilizar `feed_sessions`, `feed_impressions` e `interaction_events` do CP22. Não criar segunda tabela de impressões nem substituir contexto histórico por valores atuais.

## 4. Tarefas de implementação

### 4.1 Auditoria dos sinais e contratos do CP22

- [ ] Auditar o event log e as projeções entregues pelo CP22 antes de implementar agregações.
- [ ] Confirmar semântica, versionamento, timestamps, idempotência e ownership dos eventos de exposição, atenção, feedback e pipeline.
- [ ] Verificar disponibilidade de posição, `search_run_id`, score-base, versão do corpus/analyzer/ranking e vínculo vaga–impressão.
- [ ] Mapear quais atributos de cada vaga estão disponíveis e qual componente os normaliza.
- [ ] Identificar eventos duplicados, tardios, inválidos, compensatórios ou sem vínculo confiável e definir seu tratamento.
- [ ] Formalizar a matriz sinal → dimensão → regra de elegibilidade → limite/deduplicação → efeito no perfil.
- [ ] Não criar agregação enquanto os contratos de eventos necessários não estiverem validados.

**Aceite:** existe uma matriz versionada dos sinais utilizáveis, com semântica, origem, ownership, limitações e regras de consumo; eventos insuficientes ou inválidos não entram silenciosamente no perfil.

### 4.2 Modelo de perfil e features individuais

- [ ] Criar migration versionada para `user_preference_profiles` ou estender estrutura equivalente existente.
- [ ] Criar migration versionada para `user_preference_features` ou estender estrutura equivalente existente.
- [ ] Aplicar ownership e constraints por usuário, incluindo unicidade adequada para perfil ativo e chave de feature.
- [ ] Versionar schema, extrator, normalização e regras de agregação.
- [ ] Distinguir dimensões positivas, negativas, atenção, comprometimento e resultado contextual.
- [ ] Preservar contagens/qualidade de observações para cálculo de confiança.
- [ ] Criar índices para consulta do perfil ativo e features por usuário/tipo/chave.
- [ ] Definir estratégia de reconstrução e verificação de perfil sem duplicar o event log.
- [ ] Testar criação em banco novo, upgrade, rollback e isolamento entre usuários.

**Aceite:** perfil e features têm ownership, versionamento, reconstrução auditável e não misturam dimensões semânticas diferentes.

### 4.3 Derivação de características das vagas

- [ ] Derivar características a partir do conteúdo/metadados reais das vagas, reutilizando NLP Core, normalização e taxonomias existentes.
- [ ] Evitar listas manuais fixas de profissões, tecnologias, empresas ou cargos como fonte primária de classificação.
- [ ] Registrar origem e versão da extração/normalização usada para cada característica.
- [ ] Tratar título, área, skills, senioridade, localização e modalidade somente quando houver evidência/dado disponível.
- [ ] Preservar texto/valor original quando houver normalização.
- [ ] Diferenciar atributo explicitamente presente de valor normalizado e de inferência incerta.
- [ ] Não converter ausência de atributo em sinal negativo.
- [ ] Não extrair ou derivar atributos pessoais sensíveis desnecessários à recomendação.

**Aceite:** características utilizadas pelo perfil são provenientes de dados rastreáveis das vagas, com versão e qualidade; falta de informação não vira preferência presumida.

### 4.4 Construção do perfil a partir de eventos

- [ ] Construir sinais a partir de like, dislike, save, abertura, dwell, inclusão no pipeline, candidatura e resultados observados, respeitando a semântica do CP22.
- [ ] Manter exposição, atenção, preferência explícita, comprometimento e resultado como dimensões independentes.
- [ ] Não tratar impressão como preferência nem ausência de clique como dislike.
- [ ] Limitar a influência de ações repetidas sobre a mesma vaga, evitando que recarregamentos, impressões repetidas ou eventos duplicados dominem o perfil.
- [ ] Aplicar deduplicação lógica por evento/vaga/ação e idempotência no processamento.
- [ ] Definir regras explícitas para eventos compensatórios (`unliked`, `undisliked`, `unsaved`) e estado atual de feedback.
- [ ] Aplicar decaimento temporal às observações conforme regra versionada.
- [ ] Calibrar meia-vida e impacto do decaimento durante avaliação; não fixar valores definitivos sem corpus/protocolo.
- [ ] Considerar quantidade e qualidade das observações na confiança do perfil.
- [ ] Atualizar perfil incrementalmente ou em lotes, sem reconstruí-lo integralmente a cada abertura do feed.
- [ ] Permitir reconstrução determinística por usuário para auditoria e recuperação.

**Aceite:** a mesma sequência de eventos elegíveis e as mesmas versões de regras geram o mesmo perfil; eventos repetidos não aumentam influência indevidamente; confiança reflete quantidade e qualidade de observações.

### 4.5 Interpretação do pipeline e dos resultados

- [ ] Tratar inclusão no pipeline e candidatura submetida como intenção/ação, não como preferência explícita equivalente a like.
- [ ] Tratar entrevista, oferta e contratação como resultados contextuais, sem transformá-los em preferências universais.
- [ ] Tratar desistência/retirada como sinal contextual, considerando o motivo quando existir dado explícito e confiável.
- [ ] Não transformar rejeição da empresa em dislike automático da vaga ou de suas características.
- [ ] Não inferir que o usuário não gosta de uma área apenas porque uma candidatura foi encerrada sem sucesso.
- [ ] Evitar que candidaturas em andamento ganhem prioridade indefinidamente no feed.
- [ ] Avaliar, com base na UX existente, se candidaturas em andamento devem ser destacadas em área própria do pipeline em vez de permanecerem artificialmente no topo do ranking de descoberta.
- [ ] Garantir que os eventos consumidos correspondam a transições realmente persistidas no CP17.

**Aceite:** eventos de pipeline influenciam somente as dimensões e regras explicitamente definidas; reprovação externa não vira preferência negativa e candidaturas ativas não dominam indefinidamente o feed.

### 4.6 Re-ranking determinístico pós-BM25

- [ ] Implementar serviço/fachada de reranking que recebe candidatos e score-base prontos, sem alterar candidate generation.
- [ ] Aplicar personalização somente depois da ordenação/score-base BM25.
- [ ] Considerar sinais positivos e negativos, similaridade com vagas avaliadas, localização/modalidade quando disponíveis, atualidade, ocultações e contexto do pipeline.
- [ ] Manter fatores separados, configuráveis, versionados e explicáveis.
- [ ] Aplicar confiança baseada na quantidade e qualidade das observações; quanto menor a confiança, menor a influência do reranking.
- [ ] No cold start, manter ordenação próxima do BM25, sem fabricar preferências.
- [ ] Não fixar pesos/coeficientes/limiares como definitivos antes da avaliação offline.
- [ ] Não deixar feedback de uma vaga alterar o score-base ou as estatísticas globais de outras vagas.
- [ ] Preservar ocultações conforme a semântica do CP22, sem tratá-las como dislike.
- [ ] Aplicar diversidade de empregadores/oportunidades e cobertura de preferências sem ocultar arbitrariamente candidatos relevantes.
- [ ] Definir desempate determinístico e estável, evitando oscilações sem mudança de entrada/versão.
- [ ] Registrar score-base, score personalizado, posição inicial/final, versão do perfil e razões de reranking.
- [ ] Permitir fallback imediato para o ranking-base quando o perfil estiver indisponível, desativado ou falhar.

**Aceite:** reranking é determinístico, explicável, pós-BM25 e reversível; cold start preserva o ranking-base como principal sinal; diversidade e ocultações não contaminam o mecanismo global.

### 4.7 Evaluation Engine e protocolo de avaliação

- [ ] Definir corpus de referência, unidade de análise, janelas temporais e protocolo de comparação antes de calibrar pesos.
- [ ] Registrar versão do ranking, perfil, corpus, analyzer, posição, score-base, score personalizado e motivos de reranking.
- [ ] Medir `Qualified Impression Rate`, `Details Open Rate`, `Like Rate`, `Dislike Rate`, `Save Rate`, `Pipeline Conversion`, `Application Rate`, `Interview Progression`, `NDCG@K` e `Coverage`.
- [ ] Definir denominadores, elegibilidade, janela e tratamento de ausência de dados para cada métrica operacional; evitar comparar taxas calculadas sobre populações diferentes.
- [ ] Implementar métricas offline `Precision@K`, `Recall@K`, `NDCG@K` e `MRR` sobre julgamentos/dados de referência adequados.
- [ ] Separar qualidade textual/aderência, preferência do usuário, engajamento e resultados observados.
- [ ] Controlar viés de posição registrando exposição real e posição; não avaliar apenas cliques produzidos pelo próprio ranking.
- [ ] Separar temporalmente os eventos usados para construir o perfil dos eventos usados para avaliação.
- [ ] Evitar vazamento temporal entre treino/derivação do perfil e conjunto de avaliação, mesmo sem modelos de ML.
- [ ] Não otimizar exclusivamente clique ou dwell; engajamento não equivale automaticamente a qualidade do matching.
- [ ] Comparar ranking-base e ranking personalizado sobre a mesma população elegível e contexto comparável.
- [ ] Gerar relatórios reproduzíveis por versão, com limitações, tamanho da amostra e cobertura.
- [ ] Não declarar melhoria quando a amostra for insuficiente ou o viés de exposição impedir conclusão confiável.

**Aceite:** cada versão pode ser comparada com protocolo repetível; métricas são definidas e interpretáveis; viés de posição e separação temporal são considerados; engajamento não substitui avaliação de qualidade.

### 4.8 API, feed e interface de controle

- [ ] Integrar o reranker ao fluxo atual de consulta do feed por serviço/fachada, sem introduzir regras de personalização em handlers ou componentes.
- [ ] Preservar contratos atuais do feed quando possível; versionar alterações necessárias.
- [ ] Estender o contexto de `feed_impressions` do CP22 para registrar versão do reranker, versão do perfil, score personalizado e razões de reranking efetivamente exibidas.
- [ ] Garantir que a posição e os scores históricos não sejam recalculados retroativamente após atualização do perfil.
- [ ] Expor estado de personalização ativa/desativada e fallback de forma compreensível.
- [ ] Permitir desabilitar a personalização comportamental sem desativar o score-base BM25.
- [ ] Explicar, em nível útil e sem excesso de detalhes, os principais fatores que influenciaram a ordenação.
- [ ] Não expor informações privadas ou sinais de outros usuários nas explicações.
- [ ] Reutilizar a interface de feedback e os componentes do feed já entregues no CP22/F1.
- [ ] Manter a navegação utilizável quando o serviço de personalização estiver indisponível.
- [ ] Localizar textos em `pt-BR`, `en-US` e `es-419`; garantir acessibilidade por teclado e leitor de tela.

**Aceite:** feed utiliza o reranker após BM25, registra contexto histórico correto, oferece controle para desativar personalização e continua funcional com fallback.

### 4.9 Privacidade, exportação, limpeza e operação

- [ ] Manter eventos, perfil e features no SQLite local.
- [ ] Implementar exportação do histórico comportamental e do perfil derivado de um usuário autenticado, em formato documentado e consistente.
- [ ] Implementar limpeza do histórico comportamental do usuário, incluindo remoção/invalidação de features e perfil derivados.
- [ ] Definir e documentar o alcance da limpeza: eventos do CP22, agregados do CP23, snapshots pessoais e dados não pessoais/globais que não podem ser removidos por engano.
- [ ] Permitir desabilitar personalização sem obrigar a apagar o histórico.
- [ ] Impedir acesso cruzado entre membros da mesma instância em APIs, relatórios, exportação, limpeza e jobs.
- [ ] Não usar sinais individuais para construir perfil de outro usuário nem para alterar estatísticas globais BM25.
- [ ] Não introduzir fila, banco ou serviço externo para o volume previsto.
- [ ] Manter transações curtas, escritas idempotentes, índices úteis e atualização incremental/loteada.
- [ ] Não reconstruir perfil completo a cada abertura da timeline.
- [ ] Definir limites e recuperação para reprocessamento, falhas parciais e reconstrução de perfil.
- [ ] Garantir degradação graciosa quando telemetria, perfil ou personalização estiver indisponível.

**Aceite:** exportação, limpeza e desativação são funcionais e isoladas por usuário; falhas não comprometem o feed nem o ranking-base; não há dependência de infraestrutura externa.

### 4.10 Testes, calibração e gate

- [ ] Criar fixtures de eventos do CP22 cobrindo like/dislike, save/unsave, hide, dwell, exposição, pipeline e resultados.
- [ ] Testar eventos duplicados, fora de ordem, tardios, compensatórios e inválidos.
- [ ] Testar limitação de ações repetidas sobre a mesma vaga e deduplicação lógica.
- [ ] Testar decaimento temporal, confiança e cold start com parâmetros/versões controlados.
- [ ] Criar golden tests para agregação de features, perfil e ordenação personalizada.
- [ ] Testar que a mesma entrada e versões geram o mesmo perfil, scores, razões e ordenação.
- [ ] Testar que BM25 e candidate generation permanecem idênticos antes/depois da personalização.
- [ ] Testar que dados de um usuário não afetam perfil, reranking, relatório ou saída de outro.
- [ ] Testar pipeline: inclusão/candidatura como intenção, entrevista/oferta/contratação como resultado contextual e rejeição da empresa sem dislike automático.
- [ ] Testar diversidade, cobertura, ocultações e desempate determinístico.
- [ ] Testar exportação, limpeza, desativação e reativação da personalização.
- [ ] Testar fallback quando perfil ausente, desatualizado, corrompido ou indisponível.
- [ ] Testar migrations em banco novo e upgrade com dados de CP22.
- [ ] Medir custo de atualização/consulta com volume representativo, sem fixar metas arbitrárias antes do baseline.
- [ ] Gerar relatório comparativo entre BM25-base e reranking em corpus temporalmente separado.
- [ ] Executar `make test`, `make vet`, `make test-ui` e testes de migration/ownership.

**Gate CP23:** perfil individual reproduzível e isolado; cold start preserva BM25 como principal sinal; reranking determinístico, explicável e posterior ao score-base; diversidade/ocultações sem contaminação global; avaliação offline com separação temporal e controle de viés de exposição; exportação, limpeza e desativação validadas; nenhuma dependência de modelo de ML.

## 5. API e contratos de serviço esperados

Os nomes abaixo são conceituais e devem ser adaptados à convenção existente. Não criar endpoints duplicados se os serviços do projeto já oferecerem os contratos equivalentes.

### 5.1 Serviços internos

- `PreferenceProfileService`: construir, consultar, atualizar, reconstruir e versionar o perfil do usuário autenticado.
- `PreferenceFeatureExtractor`: derivar características de vagas usando componentes existentes e registrar origem/versão.
- `PreferenceAggregator`: consumir eventos elegíveis, aplicar deduplicação, limites, confiança e decaimento.
- `PersonalizedReranker`: receber candidatos ordenados por BM25 e devolver ordenação personalizada com scores e razões.
- `RankingEvaluationService`: calcular métricas e relatórios comparáveis por versão.
- `BehavioralDataService`: exportar, limpar e controlar os dados comportamentais do próprio usuário.

### 5.2 Contratos HTTP conceituais

- `GET /api/preferences/status` — estado da personalização e versão do perfil do usuário autenticado.
- `PUT /api/preferences/settings` — ativar/desativar personalização e preferências de processamento permitidas.
- `POST /api/preferences/rebuild` — solicitar reconstrução idempotente do próprio perfil, com validação de limites e execução controlada.
- `GET /api/preferences/export` — exportar perfil e histórico comportamental do próprio usuário.
- `DELETE /api/preferences/history` — limpar histórico/perfil derivados dentro do escopo explicitamente documentado.

O ranking do feed deve consumir o serviço internamente. Não aceitar usuário, score-base ou features pessoais arbitrárias fornecidas pelo frontend. O backend deriva a identidade da sessão, carrega os candidatos/score-base confiáveis e valida os estados usados no reranking.

### 5.3 Segurança dos contratos

- Requisições sem sessão válida retornam `401`.
- Tentativas de acesso cruzado retornam `403` ou resposta equivalente que não revele a existência de dados de outro usuário.
- Exportação e limpeza exigem ownership validado no backend e não aceitam `user_id` livre.
- Jobs de reconstrução carregam contexto explícito do proprietário e não executam com identidade global implícita.
- Erros não devem retornar payloads privados, eventos integrais ou detalhes de implementação.

## 6. Critérios de verificação e aceite

- [ ] CP23 só inicia integração final após o gate do CP22 e a validação dos eventos consumidos.
- [ ] O perfil comportamental pertence a um usuário e não pode ser consultado ou reconstruído por outro.
- [ ] Features são derivadas de atributos reais das vagas e recursos NLP versionados, sem depender de listas manuais fixas.
- [ ] Exposição, atenção, preferência explícita, comprometimento e resultado permanecem dimensões diferentes.
- [ ] Repetições da mesma ação/vaga têm influência limitada e eventos duplicados não alteram novamente o perfil.
- [ ] Decaimento temporal e confiança são versionados e calibrados por avaliação, sem coeficientes definitivos arbitrários.
- [ ] Eventos de pipeline são interpretados como intenção/resultado contextual, sem transformar rejeição da empresa em dislike automático.
- [ ] Candidate generation e score-base BM25 permanecem inalterados.
- [ ] Personalização atua exclusivamente depois do score-base e não modifica estatísticas globais.
- [ ] Cold start e personalização desativada mantêm ordenação próxima/do ranking-base conforme política documentada.
- [ ] Reranking é determinístico para as mesmas entradas e versões.
- [ ] Cada mudança de posição possui razões e versões rastreáveis.
- [ ] Diversidade, cobertura e ocultações são aplicadas sem contaminar o BM25 global.
- [ ] Evaluation Engine calcula métricas operacionais e offline com definições/denominadores documentados.
- [ ] Avaliação controla viés de posição e separa temporalmente derivação do perfil e avaliação.
- [ ] Relatórios separam qualidade textual, preferência, engajamento e resultados observados.
- [ ] Usuário consegue exportar e limpar seu histórico comportamental e desabilitar personalização.
- [ ] Falha de personalização não bloqueia feed, pipeline ou score-base.
- [ ] Migrations, autorização, isolamento, testes de regressão e gates gerais passam.

## 7. Testes obrigatórios

### 7.1 Backend e persistência

- [ ] Criação, consulta, atualização e reconstrução de perfil por usuário.
- [ ] Rejeição de acesso sem sessão e de qualquer tentativa de acesso cruzado.
- [ ] Idempotência de processamento dos eventos do CP22.
- [ ] Deduplicação e limite de influência para ações repetidas sobre a mesma vaga.
- [ ] Aplicação correta de eventos compensatórios e estados atuais de feedback.
- [ ] Decaimento temporal versionado e reproduzível.
- [ ] Confiança proporcional à quantidade/qualidade de observações, conforme regra definida.
- [ ] Atualização incremental/loteada sem reconstrução integral a cada acesso ao feed.
- [ ] Exportação e limpeza sem afetar vagas globais ou dados de outros usuários.
- [ ] Desativação e reativação de personalização sem perda inesperada do histórico.
- [ ] Reconstrução determinística a partir dos eventos elegíveis.
- [ ] Migrations em banco novo e upgrade de banco com dados de CP22.

### 7.2 Re-ranking

- [ ] Candidate generation e score BM25 idênticos com personalização ligada/desligada.
- [ ] Cold start próximo ao ranking-base.
- [ ] Preferências positivas e negativas influenciam somente o usuário correspondente.
- [ ] Similaridade usa características reais das vagas e trata atributos ausentes como indisponíveis.
- [ ] Localização/modalidade só influenciam quando disponíveis e sustentadas por dados.
- [ ] Ocultação não é convertida em dislike.
- [ ] Inclusão no pipeline/candidatura não é confundida com like.
- [ ] Rejeição da empresa não cria sinal negativo automático.
- [ ] Entrevista, oferta, contratação e desistência mantêm semântica contextual.
- [ ] Candidaturas em andamento não dominam indefinidamente a descoberta.
- [ ] Diversidade e cobertura são determinísticas e explicáveis.
- [ ] Empates e desempates produzem ordenação estável.
- [ ] Perfil ausente, corrompido ou indisponível aciona fallback sem erro visível impeditivo.

### 7.3 Evaluation Engine

- [ ] `Precision@K`, `Recall@K`, `NDCG@K` e `MRR` calculados em corpus de referência documentado.
- [ ] Métricas operacionais usam denominadores e janelas definidos.
- [ ] Separação temporal evita que eventos de avaliação influenciem o perfil avaliado.
- [ ] Controle/relato de viés de posição utiliza dados reais de exposição do CP22.
- [ ] Comparações usam população e conjunto elegível compatíveis entre ranking-base e personalizado.
- [ ] Relatórios apresentam tamanho de amostra, cobertura, versão e limitações.
- [ ] Resultados não concluem melhoria com evidência insuficiente.

### 7.4 E2E, UI e acessibilidade

- [ ] Feed com personalização ativa e perfil com observações suficientes.
- [ ] Feed em cold start sem preferências registradas.
- [ ] Feed com personalização desabilitada.
- [ ] Feed após falha/indisponibilidade do serviço de perfil.
- [ ] Exibição de explicações sem revelar dados de outro usuário.
- [ ] Exportação e limpeza do próprio histórico.
- [ ] Isolamento entre dois usuários simultâneos na mesma instância.
- [ ] Acessibilidade por teclado, leitor de tela e resoluções suportadas.
- [ ] i18n `pt-BR`, `en-US` e `es-419`.
- [ ] `make test`, `make vet` e `make test-ui` verdes.

## 8. Entregáveis

- Modelo versionado de perfil individual e features comportamentais.
- Serviço de extração/normalização de características das vagas reutilizando o NLP Core.
- Agregador determinístico de sinais com deduplicação, limites, confiança e decaimento temporal.
- Serviço de reranking pós-BM25, com fallback, diversidade, cobertura e razões explicáveis.
- Integração do reranker ao feed existente e registro de contexto histórico por impressão.
- Evaluation Engine com métricas offline/operacionais e relatórios comparáveis por versão.
- Controles de ativação/desativação, exportação e limpeza do histórico do usuário.
- Migrations, índices, testes de ownership e reconstrução.
- Golden tests, corpus/protocolo de avaliação e documentação de calibração.
- ADRs e documentação de operação, privacidade e rollback.

## 9. Registrar

- ADR de perfil comportamental, dimensões de sinais e regras de agregação.
- ADR de reranking pós-BM25, fallback/cold start, diversidade e explicabilidade.
- ADR do protocolo de avaliação, métricas, separação temporal e controle de viés de posição.
- Registro de versões de extrator, features, decaimento, perfil, reranker e configurações.
- Matriz de semântica dos eventos do CP22 e sua influência permitida no perfil.
- Decisão documentada sobre exportação, limpeza e desativação da personalização.
- Registro do corpus de referência, critérios de qualidade e resultados de calibração.
- Plano de rollback para desabilitar reranking e voltar ao ranking-base sem apagar o histórico.

## 10. Dependências, riscos e mitigação

| Risco/dependência | Impacto | Mitigação |
|---|---|---|
| CP22 não entrega eventos confiáveis ou versionados | Perfil incorreto e avaliação inválida | Bloquear integração final até o gate do CP22; corrigir contratos na origem em vez de compensar inconsistências silenciosamente. |
| Poucas observações por usuário | Personalização instável | Cold start próximo ao BM25, confiança graduada e fallback até existir evidência suficiente. |
| Repetição de ações sobre a mesma vaga | Perfil dominado por um único item | Deduplicação lógica, limites de influência e testes de repetição. |
| Dwell confundido com preferência | Perfil enviesado por atenção passiva | Manter atenção separada de like/dislike e calibrar impacto com avaliação. |
| Eventos de pipeline interpretados incorretamente | Preferências falsas | Manter intenção, resultado e feedback explícito separados; nunca converter rejeição externa automaticamente em dislike. |
| Atributos de vagas inconsistentes | Features ruidosas | Reutilizar NLP Core, versionar extração, registrar origem e tratar ausência como desconhecida. |
| Pesos/decay definidos sem evidência | Reranking arbitrário | Calibrar em corpus temporalmente separado; não fixar parâmetros definitivos antes da avaliação. |
| Viés de exposição/posição | Métricas superestimam o ranking atual | Registrar impressões qualificadas, posições e conjuntos apresentados no CP22; comparar com protocolo documentado. |
| Personalização prejudica diversidade ou descoberta | Feed estreito e repetitivo | Aplicar diversidade/cobertura, medir efeitos e manter rollback para BM25. |
| Perfil individual vaza entre usuários | Violação de privacidade | Ownership no backend, testes negativos, exportação/limpeza por proprietário e nenhuma agregação cruzada. |
| Reprocessamento caro | Lentidão no feed | Atualização incremental/loteada, índices e reconstrução fora do caminho crítico. |
| Serviço de personalização indisponível | Feed deixa de funcionar | Fallback para ranking-base e degradação graciosa. |

## 11. Fora de escopo

- Collaborative filtering.
- Learning to Rank com LambdaMART.
- BPR e embeddings treinados.
- Exploração probabilística avançada ou bandits.
- Modelos de previsão de contratação ou sucesso profissional.
- Alteração do candidate generation ou do score-base BM25 da Parte 1.
- Recriação do event log, das impressões, da timeline ou dos controles de feedback (CP22).
- Recriação do pipeline de candidaturas (F2/CP17).
- Envio de eventos/perfis comportamentais a serviços externos.
- Introdução de banco, broker ou infraestrutura de analytics externa para o volume previsto.
- Mudanças no NLP Core ou criação de taxonomia paralela sem decisão arquitetural específica.

Essas evoluções exigem volume, representatividade, governança e protocolos de avaliação que não fazem parte desta entrega.

## 12. Sequência de execução e gates internos

A implementação deve ocorrer em etapas internas, mas o resultado é um único checkpoint.

1. **Gate A — Auditoria e contratos:** CP22, eventos, pipeline, dados de vaga e contexto de ranking validados.
2. **Gate B — Perfil e features:** migrations, ownership, extração de características, agregação, confiança, deduplicação e decaimento reproduzíveis.
3. **Gate C — Reranking:** pós-BM25, cold start, diversidade, explicações, versionamento e fallback aprovados.
4. **Gate D — Avaliação e privacidade:** métricas offline/operacionais, viés de exposição, separação temporal, exportação, limpeza e desativação aprovados.
5. **Gate final CP23:** testes completos, documentação, ADRs, rollback e aceite do ranking personalizado sem regressão do BM25/feed.

Não habilitar reranking por padrão antes de concluir os gates e demonstrar comportamento correto em cold start e fallback. A ativação deve ser controlável por configuração/feature flag compatível com a infraestrutura de release.

## 13. Critérios transversais de qualidade

- [ ] `make test`, `make vet` e `make test-ui` verdes.
- [ ] Migrations testadas em banco novo e upgrade, sem perda de dados.
- [ ] Testes negativos de autorização e isolamento por usuário.
- [ ] Golden tests para extração de features, agregação, confiança, decaimento e ordenação.
- [ ] Nenhum evento, perfil ou conteúdo privado em logs.
- [ ] Relatórios reproduzíveis com versões, amostras, denominadores e limitações.
- [ ] Nenhuma chamada externa ou modelo de ML obrigatório para a personalização.
- [ ] Rollback para ranking-base disponível e testado.
- [ ] Acessibilidade, i18n e regressão visual verificadas nas alterações de UI.
- [ ] Atualizar ADRs, `docs/plan/README.md`, backlog e documentação de release ao concluir.

**Gate final do CP23:** perfil individual reproduzível e isolado; sinais com semântica preservada; cold start próximo ao BM25; reranking determinístico, explicável e pós-score-base; diversidade e ocultações aplicadas sem contaminação global; avaliação temporalmente separada e consciente do viés de exposição; privacidade, exportação, limpeza e desativação validadas; e fallback comprovado para o ranking-base. Nenhum modelo de ML ou infraestrutura externa é necessário para aceitar esta entrega.
