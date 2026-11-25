# Checkpoint 22 — Score Fit, Parte 2: timeline e captura comportamental

> **Estimativa inicial:** a definir após auditoria do feed, dos eventos existentes, do pipeline, dos contratos da F5/B11 e do pacote `internal/events`. Não fixar prazo antes de identificar componentes reutilizáveis, lacunas de instrumentação e riscos de persistência.
> **Status:** proposto.
> **Pré-requisitos:** CP18 concluído nos pontos de identidade, autenticação, autorização e ownership; F1/CP16 com feed e navegação estabilizados; F2/CP17 com pipeline e transições de candidatura disponíveis; F5/B11 com coleção de vagas, score-base, versões do corpus/analyzer/ranking e contratos de feed estabilizados; SQLite WAL/FTS5 operacionais; migrations e suítes atuais verdes.
> **Responsabilidade:** registrar exposição real a vagas, duração de visualização, abertura de detalhes, preferências explícitas e transições reais do pipeline, mantendo histórico idempotente e projeções atuais por usuário.
> **Relação com CP18:** identidade, sessão e ownership pertencem ao CP18. O CP22 deriva o usuário da sessão autenticada e não aceita `user_id` livre do frontend.
> **Relação com F1/CP16:** o feed fornece os cards, detalhes, navegação e pontos de interação a instrumentar. O CP22 deve evoluir a experiência existente, sem criar um segundo feed.
> **Relação com F2/CP17:** o pipeline continua sendo a fonte da verdade para candidaturas e etapas. O CP22 registra eventos correspondentes às transições reais, sem duplicar o estado do pipeline.
> **Relação com F5/B11:** o score BM25 e o ranking-base permanecem inalterados. O CP22 registra a posição, os scores e as versões usados na exposição para permitir avaliação posterior.
> **Relação com CP23:** o CP23 consumirá os eventos e projeções confiáveis produzidos aqui para calcular preferências e aplicar reranking individual. O CP22 não implementa perfil de preferências nem personalização.

## 1. Objetivo

Entregar a infraestrutura de timeline comportamental do Score Fit — Parte 2, registrando de forma confiável como cada usuário é exposto às vagas e quais ações realiza no feed e no pipeline.

A captura deve distinguir exposição efetiva de simples renderização no DOM, medir duração somente enquanto o conteúdo estiver visível e a janela estiver ativa, separar tempo no card de tempo nos detalhes e registrar ações explícitas sem confundir `like`, `dislike`, `save` e `hide`.

Os eventos serão persistidos em SQLite como histórico imutável, com ingestão em lote, idempotência e atualização transacional de projeções. O histórico deve permitir reconstruir o estado atual de feedback e fornecer contexto suficiente para avaliar versões futuras do ranking, incluindo posição, conjunto de candidatos, score-base, score exibido e versões do mecanismo.

**Princípio central:** exposição, interesse implícito, preferência explícita, comprometimento e resultado são sinais diferentes. Ausência de interação não significa rejeição; salvar não significa gostar; ocultar não significa não gostar; e uma transição do pipeline deve refletir um fato real registrado pelo próprio pipeline.

## 2. Escopo e decisões arquiteturais

### 2.1 Responsabilidades

- **CP18:** identidade, sessão, autorização e ownership. Toda leitura e escrita comportamental é vinculada ao usuário autenticado pelo backend.
- **F1/CP16:** feed, cards, detalhes, navegação, layout e componentes de interação existentes. O CP22 adiciona instrumentação e controles necessários sem recriar a experiência principal.
- **F2/CP17:** candidaturas, pipeline e transições monotônicas. O CP22 observa e registra eventos originados por mudanças reais nesse domínio; não passa a ser dono do pipeline.
- **F5/B11 — Score Fit Parte 1:** coleção global de vagas, normalização, keywords, BM25 ponderado e score-base. O CP22 captura o contexto do ranking, sem alterar algoritmo, pesos ou estatísticas globais.
- **CP22:** captura de exposição e interação, sessões/impressões do feed, event log, projeção de feedback atual, endpoint de ingestão em lote, controles de feedback e testes do ciclo completo.
- **CP23 — personalização e avaliação:** consumirá eventos/projeções do CP22 para construir perfil individual, aplicar reranking e medir resultados. Não deve ser implementado dentro deste checkpoint.
- **CP18 e migrations existentes:** fornecem o modelo de usuário e a base SQLite. O CP22 estende o schema com migrations versionadas, sem introduzir outro banco.
- **CP11:** continua responsável por instalação, bootstrap técnico e operação do executável. O CP22 utiliza a infraestrutura atual.

### 2.2 Limites da captura comportamental

O checkpoint registra apenas interações pertinentes ao uso do OpenJob:

1. **Exposição:** uma vaga efetivamente visível no feed, qualificada por proporção visível e duração mínima.
2. **Atenção implícita:** duração ativa no card e nos detalhes, enquanto a aba e a janela estiverem visíveis/ativas.
3. **Preferência explícita:** ações de gostar, não gostar, salvar e ocultar, mantidas como dimensões distintas.
4. **Comprometimento:** inclusão no pipeline e candidatura, observadas a partir dos eventos reais do CP17.
5. **Resultado:** transições relevantes do pipeline, como retirada, rejeição da empresa, oferta e contratação, sem inferir preferência a partir delas.

Não coletar movimentos de mouse, teclas digitadas, navegação fora do OpenJob ou sinais não previstos nesta finalidade. Não interpretar ausência de clique, tempo curto ou abandono do card como `dislike`.

### 2.3 Arquitetura local-first e persistência

- Manter SQLite como banco de eventos e projeções, reutilizando a configuração WAL e os padrões de migrations atuais.
- Não introduzir Kafka, Redis, MongoDB, fila externa ou outro banco exclusivamente para registrar interações.
- Manter o event log imutável e as projeções como representações atualizáveis do estado atual.
- Atualizar evento e projeção na mesma transação quando a operação produzir ambos.
- Fazer ingestão idempotente por `event_uuid`; reenvio de lote não pode duplicar eventos nem aplicar duas vezes a mesma ação.
- Criar índices conforme os padrões de consulta por usuário/data, vaga, sessão/impressão, posição e estado de feedback.
- Auditar `internal/events` antes de criar pacotes ou estruturas com responsabilidade duplicada. Estender abstrações existentes quando apropriado.
- Manter operações curtas e em lote. Falha de telemetria não pode bloquear a exibição, navegação ou uso básico do feed.

### 2.4 Sem alteração do score-base

- O score BM25 da Parte 1 continua sendo a medida textual de aderência entre currículo e vaga.
- O CP22 registra o score-base e, quando aplicável, o score efetivamente exibido, junto das versões do analyzer, corpus e ranking.
- As ações individuais não modificam corpus, estatísticas globais, pesos BM25 ou ranking-base.
- O CP22 não ordena candidatos por preferência pessoal e não cria perfil de usuário.
- O registro de posição e contexto de exposição é necessário para que o CP23 e a avaliação posterior possam considerar viés de posição e comparar versões.

### 2.5 Privacidade, segurança e confiança dos dados

- Toda entidade individual deve ter ownership do usuário autenticado, inclusive sessões, impressões, eventos e projeções.
- O backend nunca confia em `user_id` enviado no payload; deriva o usuário da sessão CP18.
- Validar tipo de evento, vínculo entre usuário/vaga/impressão, timestamps, duração, tamanho do payload e quantidade de eventos por lote.
- Tratar payload recebido do navegador como dado não confiável. Não aceitar campos que permitam alterar score, versão do ranking, proprietário ou estado do pipeline sem validação do servidor.
- Não registrar conteúdo privado de vagas, documentos, currículos, tokens ou payloads integrais nos logs técnicos.
- Não enviar eventos ou perfis comportamentais para serviços externos.
- Não compartilhar sinais ou feedback de um usuário com outro membro da mesma instância.

## 3. Modelo de domínio e contratos

A estrutura final deve seguir os padrões reais do projeto e ser formalizada em migrations e contratos versionados. Os nomes abaixo são os identificadores conceituais já definidos no backlog; podem ser ajustados à convenção do repositório desde que a semântica seja preservada.

### 3.1 `feed_sessions`

Representa uma sessão de navegação no feed, pertencente a um usuário.

Campos mínimos esperados:

- identificador da sessão;
- usuário proprietário;
- início e fim, com timestamps;
- identificador da execução de busca/geração (`search_run_id`), quando disponível;
- versão do corpus, analyzer e ranking utilizados;
- metadados necessários para relacionar impressões e eventos.

A sessão não deve ser confundida com a sessão de autenticação do CP18. Sua finalidade é agrupar uma sequência de exposições no feed para análise comportamental.

### 3.2 `feed_impressions`

Representa a exposição de uma vaga em uma posição específica de uma sessão do feed.

Campos mínimos esperados:

- identificador da impressão;
- usuário proprietário, sessão e vaga;
- posição no conjunto apresentado;
- referência ao `search_run_id` ou conjunto de candidatos, quando disponível;
- score-base e score exibido no momento da exposição;
- indicação de impressão qualificada e parâmetros/versão usados para qualificá-la;
- duração ativa e visível no card;
- indicação de abertura dos detalhes e duração ativa/visível nos detalhes;
- timestamps relevantes;
- versões do corpus, analyzer e ranking.

Uma vaga pode gerar mais de uma impressão legítima em sessões ou exposições diferentes. A identidade da impressão deve permitir relacionar os eventos de atenção e interação à exposição correspondente sem duplicar a mesma impressão por reenvio de lote.

### 3.3 `interaction_events`

Histórico imutável de eventos comportamentais e eventos do pipeline observados pelo Score Fit.

Campos mínimos esperados:

- `event_uuid` único para idempotência;
- usuário proprietário;
- vaga relacionada, quando aplicável;
- sessão/impressão relacionada, quando aplicável;
- tipo do evento;
- horário de ocorrência e horário de recebimento;
- versão do schema;
- payload validado e limitado ao necessário.

Eventos aceitos inicialmente:

**Exposição e atenção**

- `feed.impression`
- `feed.card_dwell`
- `job.details_opened`
- `job.details_dwell`

**Preferências explícitas**

- `job.liked`
- `job.unliked`
- `job.disliked`
- `job.undisliked`
- `job.saved`
- `job.unsaved`
- `job.hidden`

**Pipeline e resultados**

- `pipeline.job_added`
- `pipeline.application_submitted`
- `pipeline.stage_changed`
- `pipeline.application_withdrawn`
- `pipeline.employer_rejected`
- `pipeline.offer_received`
- `pipeline.hired`

O catálogo de eventos deve ser validado no backend. Eventos desconhecidos ou payloads incompatíveis devem ser rejeitados de forma controlada, sem interromper o processamento dos demais eventos válidos do lote conforme a política de atomicidade definida na API.

### 3.4 `user_job_feedback`

Projeção do estado atual de feedback de cada usuário para cada vaga.

- Deve possuir `UNIQUE(user_id, job_id)`.
- `like` e `dislike` são mutuamente exclusivos.
- `save` é independente de `like`/`dislike`.
- `hide` é distinto de `dislike`.
- Remover ou desfazer uma ação atualiza a projeção, mas não apaga o evento histórico correspondente.
- A projeção pode ser reconstruída a partir do event log, conforme regras/versionamento documentados.

O contrato final deve permitir consultar eficientemente o estado atual sem percorrer todo o histórico a cada abertura do feed.

### 3.5 Semântica de estados e eventos

- `job.liked` ativa o estado de gostei e desativa eventual dislike atual, respeitando a regra de exclusividade.
- `job.unliked` remove o estado de gostei sem apagar o histórico.
- `job.disliked` ativa o estado de não gostei e desativa eventual like atual.
- `job.undisliked` remove o estado de não gostei sem apagar o histórico.
- `job.saved` e `job.unsaved` controlam o estado de salvo independentemente de like/dislike.
- `job.hidden` registra a ação de ocultar; não deve ser reinterpretada como dislike. Caso o produto ofereça desfazer ocultação, seu contrato deve ser documentado sem apagar o evento original.
- `pipeline.*` reflete uma ação/transição efetivamente persistida no pipeline. Não inferir eventos de pipeline apenas por exposição, feedback ou navegação.

### 3.6 Contexto de ranking e exposição

Para cada impressão qualificada, preservar, quando disponível:

- posição da vaga no conjunto apresentado;
- identificador da execução de busca (`search_run_id`);
- conjunto/versão de candidatos apresentados;
- score-base BM25;
- score efetivamente exibido;
- versão do corpus;
- versão do analyzer;
- versão do ranking.

Não substituir valores históricos por versões atuais do ranking. O registro deve descrever o contexto em que a vaga foi mostrada ao usuário.

## 4. Tarefas de implementação

### 4.1 Auditoria do feed, pipeline e eventos existentes

- [ ] Auditar componentes e estados do feed F1/CP16 para identificar pontos corretos de instrumentação de cards, detalhes, navegação e feedback.
- [ ] Auditar a implementação do pipeline F2/CP17 e identificar transições reais que podem emitir eventos comportamentais.
- [ ] Auditar o pacote `internal/events`, contratos e mecanismos existentes antes de criar novas estruturas.
- [ ] Inventariar eventos já emitidos, tabelas, stores, workers e endpoints relacionados; evitar duplicação.
- [ ] Confirmar como a versão do corpus, analyzer, ranking, score-base, posição e conjunto de candidatos chegam ao frontend.
- [ ] Documentar gaps e decisões de compatibilidade antes das migrations.

**Aceite:** existe uma matriz de eventos e pontos de instrumentação que identifica origem, payload, proprietário, persistência e consumidor de cada evento.

### 4.2 Instrumentação de exposição real

- [ ] Usar `IntersectionObserver` para determinar a visibilidade efetiva dos cards.
- [ ] Registrar impressão qualificada quando pelo menos 50% do card estiver visível por pelo menos 1 segundo, com aba visível e janela ativa.
- [ ] Manter limiar de visibilidade e duração configuráveis, identificando a versão/parâmetros aplicados à impressão.
- [ ] Não registrar exposição somente porque o card foi renderizado ou inserido no DOM.
- [ ] Evitar criar diversas impressões para o mesmo card durante uma exposição contínua; definir regra estável para encerrar e iniciar uma nova impressão.
- [ ] Associar cada exposição à sessão, vaga, posição e contexto de ranking disponíveis.
- [ ] Registrar `feed.impression` somente quando as condições de qualificação forem satisfeitas.

**Aceite:** uma impressão representa exposição qualificada e não simples renderização; limiares e contexto são reproduzíveis e testáveis.

### 4.3 Medição de tempo ativo e visível

- [ ] Usar `document.visibilityState`, eventos `focus` e `blur` para evitar contabilizar tempo com aba oculta ou janela inativa.
- [ ] Usar relógio monotônico para medir duração, evitando distorção por ajuste do relógio do sistema.
- [ ] Separar tempo no card, tempo nos detalhes e tempo com conteúdo efetivamente visível.
- [ ] Pausar a contagem quando o conteúdo deixar de atender às condições de visibilidade/atividade.
- [ ] Retomar a contagem apenas quando as condições voltarem a ser atendidas.
- [ ] Acumular duração localmente e enviar resumo por impressão; não emitir um evento a cada segundo.
- [ ] Registrar abertura de detalhes como sinal independente (`job.details_opened`), além da duração (`job.details_dwell`).
- [ ] Definir limites e validações para durações impossíveis, negativas ou incompatíveis com a janela temporal da impressão.
- [ ] Não interpretar dwell curto ou igual a zero como dislike.

**Aceite:** dwell representa somente tempo ativo e visível, separa card/detalhes e não depende de eventos emitidos continuamente.

### 4.4 Event log e projeções SQLite

- [ ] Criar migration versionada para `feed_sessions` ou estender estrutura equivalente já existente.
- [ ] Criar migration versionada para `feed_impressions` ou estender estrutura equivalente já existente.
- [ ] Criar/estender `interaction_events` como event log imutável com `event_uuid` único, usuário, vaga, sessão/impressão, tipo, timestamps, versão de schema e payload validado.
- [ ] Criar/estender `user_job_feedback` como projeção do estado atual, com `UNIQUE(user_id, job_id)`.
- [ ] Aplicar constraints e índices adequados às consultas por usuário, data, vaga, posição, sessão e estado de feedback.
- [ ] Atualizar projeções na mesma transação da ingestão do evento correspondente.
- [ ] Aplicar idempotência por `event_uuid` antes de alterar projeções.
- [ ] Manter histórico imutável: desfazer feedback gera evento compensatório e altera somente a projeção.
- [ ] Preservar like/dislike mutuamente exclusivos e save/hide independentes.
- [ ] Manter o pipeline como fonte da verdade; não duplicar estados de candidatura em `user_job_feedback` ou em uma segunda tabela de pipeline.
- [ ] Definir estratégia de reconstrução/verificação das projeções a partir do histórico.
- [ ] Testar migrations em banco novo e upgrade com dados preexistentes.

**Aceite:** event log e projeções são consistentes, idempotentes, transacionais e isolados por usuário; projeções podem ser verificadas/reconstruídas a partir do histórico.

### 4.5 Emissão de eventos do pipeline

- [ ] Emitir `pipeline.job_added` quando uma vaga for efetivamente adicionada ao pipeline.
- [ ] Emitir `pipeline.application_submitted` quando a candidatura for registrada como enviada, conforme o estado real do CP17.
- [ ] Emitir `pipeline.stage_changed` em transições persistidas de etapa.
- [ ] Emitir `pipeline.application_withdrawn`, `pipeline.employer_rejected`, `pipeline.offer_received` e `pipeline.hired` apenas quando a transição correspondente ocorrer de fato.
- [ ] Preservar a referência à candidatura/vaga e ao usuário proprietário.
- [ ] Evitar eventos duplicados quando a transição for repetida, reprocessada ou tentada novamente.
- [ ] Não emitir evento de resultado a partir de inferência comportamental ou de ausência de atividade.
- [ ] Não alterar a semântica, as regras ou a fonte de verdade do pipeline para atender à telemetria.

**Aceite:** eventos do pipeline correspondem a transições reais, são idempotentes e não duplicam o domínio de candidaturas.

### 4.6 Endpoint de ingestão em lote

- [ ] Criar `POST /api/feed/events` seguindo os padrões de autenticação e resposta existentes.
- [ ] Derivar usuário da sessão autenticada CP18; rejeitar payload sem sessão válida.
- [ ] Não aceitar `user_id` do cliente como autoridade. Se o campo aparecer por compatibilidade, ignorá-lo ou rejeitá-lo conforme o contrato documentado.
- [ ] Validar tipos, versão do schema, `event_uuid`, relação entre impressão/sessão/vaga, timestamps, duração e tamanho de payload.
- [ ] Aplicar limites de quantidade de eventos e tamanho máximo por lote.
- [ ] Aplicar idempotência por evento e transação consistente para event log/projeções.
- [ ] Definir explicitamente a política de lote parcialmente inválido: rejeição total ou retorno por item, sem deixar eventos/projeções em estado inconsistente.
- [ ] Retornar confirmação suficiente para o frontend descartar eventos aceitos e tratar itens rejeitados.
- [ ] Garantir que erros não revelem dados de outro usuário nem payloads privados.
- [ ] Testar autorização cruzada, replay, lote duplicado, lote fora de ordem e payload excessivo.

**Aceite:** endpoint autenticado e limitado aceita lotes válidos de forma idempotente, rejeita eventos inválidos e nunca permite atribuição de dados a outro usuário.

### 4.7 Flush, resiliência e degradação graciosa no frontend

- [ ] Fazer flush ao atingir limite de lote, em intervalos controlados, ao ocultar a aba e ao encerrar a sessão do feed.
- [ ] Usar `sendBeacon` como mecanismo complementar para flush em transições de página/encerramento, com fallback compatível.
- [ ] Não depender exclusivamente de `beforeunload` para persistir eventos.
- [ ] Tratar falhas de rede sem bloquear feed, navegação, abertura de detalhes ou ações do usuário.
- [ ] Manter fila temporária limitada para eventos ainda não confirmados, com política de retry e descarte documentada.
- [ ] Garantir que reenvio após falha não duplique eventos nem altere projeções duas vezes.
- [ ] Limitar memória e volume acumulado no frontend; não manter fila ilimitada.
- [ ] Exibir erros apenas quando uma ação do usuário exigir feedback; falha silenciosa de telemetria não deve produzir toasts intrusivos.
- [ ] Não prometer captura de eventos que o navegador não conseguiu enviar; documentar limites de encerramento abrupto.

**Aceite:** perda eventual de telemetria não degrada a experiência principal, e qualquer reenvio possível é idempotente.

### 4.8 Controles de feedback na timeline

- [ ] Disponibilizar ações distintas de Gostei, Não gostei, Salvar e Ocultar no contexto adequado do feed/detalhes.
- [ ] Permitir desfazer/remover feedback compatível com o estado atual, preservando o histórico de eventos.
- [ ] Garantir que like e dislike não permaneçam ativos simultaneamente.
- [ ] Garantir que salvar seja independente de gostar/não gostar.
- [ ] Garantir que ocultar não seja sinônimo de não gostar.
- [ ] Oferecer motivos opcionais de dislike: área, remuneração, localização, modalidade, senioridade, requisitos incompatíveis, falta de interesse ou outro.
- [ ] Não exigir motivo para registrar dislike.
- [ ] Persistir estado atual por usuário/vaga e refletir o resultado após refresh.
- [ ] Preservar acessibilidade, rótulos, foco, responsividade e consistência com o design system existente.

**Aceite:** ações têm semântica distinta, podem ser desfeitas quando aplicável e persistem sem apagar o histórico.

### 4.9 Integração, migração e operação

- [ ] Integrar a instrumentação ao feed atual sem criar rota, store ou experiência paralela.
- [ ] Manter compatibilidade com eventos e contratos existentes durante a migração, quando houver consumidores ativos.
- [ ] Versionar schema e payloads para permitir evolução controlada.
- [ ] Garantir que workers e tarefas assíncronas preservem o usuário proprietário quando processarem eventos.
- [ ] Definir logs técnicos mínimos, métricas operacionais agregadas e diagnóstico sem expor conteúdo privado.
- [ ] Documentar comportamento quando a persistência de eventos estiver indisponível.
- [ ] Não habilitar personalização/re-ranking nesta entrega; a saída do CP22 é a base de dados para o CP23.

**Aceite:** feed e pipeline seguem funcionais, contratos permanecem versionados e o CP23 consegue consumir histórico e projeções sem depender de detalhes do frontend.

## 5. Critérios de verificação e aceite

- [ ] Impressões representam exposição efetiva, não apenas renderização de componentes.
- [ ] O critério inicial de qualificação é pelo menos 50% do card visível por pelo menos 1 segundo, com aba visível e janela ativa; parâmetros são configuráveis e versionados.
- [ ] Dwell considera somente tempo ativo/visível e separa card e detalhes.
- [ ] O relógio monotônico é utilizado para duração; timestamps de ocorrência e recebimento são preservados.
- [ ] Abertura de detalhes é registrada como sinal independente da duração nos detalhes.
- [ ] Eventos de exposição, atenção, feedback e pipeline possuem tipos e semânticas distintos.
- [ ] Ausência de clique, dwell curto ou abandono não é convertido em dislike.
- [ ] Like/dislike são mutuamente exclusivos; save é independente; hide não equivale a dislike.
- [ ] Desfazer feedback atualiza a projeção e mantém o histórico imutável.
- [ ] `interaction_events` possui `event_uuid` único e ingestão idempotente.
- [ ] `user_job_feedback` possui unicidade por usuário/vaga e pode ser reconciliado com o event log.
- [ ] `feed_sessions` e `feed_impressions` preservam usuário, vaga, posição, contexto e versões disponíveis.
- [ ] O contexto registra `search_run_id`, score-base, score exibido, corpus, analyzer e ranking quando disponíveis.
- [ ] Eventos de pipeline correspondem a transições reais do CP17 e não duplicam sua fonte da verdade.
- [ ] `POST /api/feed/events` deriva identidade da sessão, valida payload e aplica limites/idempotência.
- [ ] Dois usuários simultâneos não conseguem consultar ou alterar sessões, impressões, eventos ou feedback um do outro.
- [ ] Flush em lote e `sendBeacon` são complementares; não dependem apenas de `beforeunload`.
- [ ] Falhas de telemetria não impedem a navegação nem o uso do feed.
- [ ] Migrations funcionam em banco novo e upgrade sem perda de dados.
- [ ] `make test`, `make vet`, `make test-ui` e testes específicos de migration/ownership passam.

**Gate final do CP22:** exposição real e dwell ativo são capturados com semântica clara; histórico e projeções são idempotentes e isolados por usuário; feedbacks são distintos e reversíveis quando aplicável; eventos do pipeline refletem transições verdadeiras; contexto e versões do ranking são preservados; falhas de telemetria não degradam o feed; e os contratos ficam estáveis para o CP23.

## 6. Testes obrigatórios

### 6.1 Backend e persistência

- [ ] Criação/leitura/encerramento de sessões de feed por usuário.
- [ ] Criação de impressões qualificadas e não qualificadas.
- [ ] Rejeição de impressão, vaga, sessão ou feedback pertencente a outro usuário.
- [ ] Unicidade de `event_uuid` e reenvio idempotente do mesmo evento.
- [ ] Atualização transacional de event log e projeção.
- [ ] Reconstrução/reconciliação de `user_job_feedback` a partir do histórico.
- [ ] Exclusividade de like/dislike e independência de save/hide.
- [ ] Eventos compensatórios para unliked/undisliked/unsaved sem apagar o histórico.
- [ ] Validação de timestamps, duração, tipos, schema e payload excessivo.
- [ ] Ingestão de lotes válidos, duplicados, parcialmente inválidos e fora de ordem, conforme a política definida.
- [ ] Emissão de eventos de pipeline em transições reais e ausência de duplicação em retry.
- [ ] Rejeição de requisição sem sessão ou com tentativa de atribuir `user_id` arbitrário.
- [ ] Migrations em banco novo e upgrade a partir do schema suportado.

### 6.2 Frontend, instrumentação e E2E

- [ ] Card renderizado fora da viewport não gera impressão.
- [ ] Card com menos de 50% de visibilidade não gera impressão qualificada.
- [ ] Card visível por menos de 1 segundo não gera impressão qualificada.
- [ ] Card com pelo menos 50% de visibilidade por pelo menos 1 segundo, aba visível e janela ativa gera uma impressão.
- [ ] Aba oculta, janela inativa ou perda de foco pausa a contagem de dwell conforme a regra implementada.
- [ ] Retorno à aba/janela ativa retoma a contagem sem incluir o intervalo inativo.
- [ ] Tempo no card e tempo nos detalhes são registrados separadamente.
- [ ] Abertura de detalhes gera evento independente.
- [ ] Scroll, virtualização e re-renderização do feed não criam impressões duplicadas durante exposição contínua.
- [ ] Like, unlike, dislike, undislike, save, unsave e hide mantêm os estados corretos após refresh.
- [ ] Motivos opcionais de dislike são registrados sem obrigatoriedade.
- [ ] Flush por limite, intervalo, visibilidade/encerramento e `sendBeacon` são testados.
- [ ] Falha de rede e retry não bloqueiam o feed nem duplicam eventos.
- [ ] Regressão visual, responsividade, teclado, leitores de tela e foco dos novos controles.

### 6.3 Privacidade e isolamento

- [ ] Usuário A não lista nem consulta sessões, impressões, eventos ou feedback de usuário B.
- [ ] IDs de sessão, impressão, vaga ou evento manipulados não permitem acesso cruzado.
- [ ] Payload com `user_id` de terceiro não altera proprietário do evento.
- [ ] Nenhum documento, currículo, token ou conteúdo privado aparece em logs.
- [ ] Nenhum evento individual é enviado a serviço externo.
- [ ] Eventos de navegação fora do OpenJob, movimentos de mouse e teclas digitadas não são coletados.

## 7. Entregáveis

- Migrations SQLite versionadas para sessões, impressões, event log e projeção de feedback, reutilizando estruturas existentes quando apropriado.
- Contratos versionados dos eventos de exposição, atenção, preferências e pipeline.
- Instrumentação do feed com `IntersectionObserver`, visibilidade da aba/janela e medição monotônica de dwell.
- Endpoint autenticado `POST /api/feed/events` com ingestão em lote, validação, limites e idempotência.
- Projeção `user_job_feedback` consistente com o event log.
- Controles de Gostei, Não gostei, Salvar e Ocultar integrados à timeline existente.
- Emissão de eventos de transição do pipeline integrada ao CP17, sem duplicar sua fonte da verdade.
- Testes backend, migrations, ownership, E2E, acessibilidade e regressão.
- Documentação do catálogo de eventos, payloads, regras de qualificação, limites de captura e operação.

## 8. Registrar

- ADR de timeline comportamental e separação entre event log e projeções.
- ADR de critérios de impressão qualificada e medição de dwell ativo/visível.
- Contrato/versionamento do catálogo de eventos e payloads.
- Decisão documentada sobre atomicidade de lotes parcialmente inválidos, limites, retries e descarte.
- Matriz de ownership e índices das tabelas comportamentais.
- Decisão de integração entre eventos do pipeline e `internal/events`, preservando o CP17 como fonte da verdade.
- Protocolo de reconstrução/reconciliação de `user_job_feedback`.
- Registro de limites conhecidos de envio no encerramento de página/navegador.

## 9. Dependências, riscos e mitigação

| Risco/dependência | Impacto | Mitigação |
|---|---|---|
| Feed virtualizado ou re-renderizado | Impressões duplicadas ou não registradas | Instrumentar ciclo de visibilidade com identidade estável da impressão e testes de scroll/virtualização. |
| Aba em segundo plano ou janela inativa | Dwell artificialmente elevado | Combinar `IntersectionObserver`, `document.visibilityState`, `focus`/`blur` e relógio monotônico. |
| Navegador encerra sem flush | Perda de eventos recentes | Flush em lote, `sendBeacon` complementar, fallback e limites explícitos; não depender apenas de `beforeunload`. |
| Reenvio de lote após falha | Eventos ou feedback duplicados | `event_uuid` único, idempotência no backend e transação de projeção. |
| Payload adulterado pelo cliente | Atribuição cruzada ou métricas inválidas | Derivar usuário da sessão, validar relacionamentos, tipos, timestamps, duração e limites. |
| Duplicação de eventos do pipeline | Histórico divergente do CP17 | Emitir a partir das transições persistidas e manter CP17 como fonte da verdade. |
| Crescimento do histórico | Aumento de armazenamento e consultas lentas | Ingestão em lote, índices orientados a consultas reais e avaliação com volume representativo antes de definir política operacional. |
| Interação interpretada de forma incorreta | Preferências contaminadas no CP23 | Manter exposição, dwell, preferência, comprometimento e resultado como categorias distintas; ausência de ação nunca é dislike. |
| `internal/events` já possui função equivalente | Duplicação de infraestrutura | Auditar antes de implementar e estender contratos existentes quando possível. |
| Falha da telemetria afeta o feed | Degradação da função principal | Processamento desacoplado, fila temporária limitada e degradação graciosa. |

## 10. Fora de escopo

- Criar perfil de preferências individuais ou vetor de características do usuário (CP23).
- Aplicar personalização, reranking ou alterar posição de vagas por sinais comportamentais (CP23).
- Alterar o score-base BM25, os pesos globais, a coleção de vagas ou as estatísticas do ranking da Parte 1.
- Implementar Evaluation Engine, métricas offline de ranking ou experimentos comparativos de personalização (CP23).
- Collaborative filtering, Learning to Rank, LambdaMART, BPR, embeddings treinados ou previsão de contratação.
- Coletar movimentos de mouse, teclas digitadas, navegação externa ou telemetria de terceiros.
- Reimplementar feed, detalhes de vagas ou pipeline de candidaturas.
- Introduzir Kafka, Redis, MongoDB ou infraestrutura externa de eventos para o volume previsto.
- Implementar autenticação, sessões de usuário ou ownership geral (CP18).
- Implementar Inteligência de documentos ou Tailoring determinístico (CP20/CP21).

## 11. Integração com o próximo checkpoint

O CP22 deve encerrar com contratos de eventos e projeções estáveis, sem antecipar a lógica de personalização.

O **CP23 — Score Fit, Parte 2: personalização individual e avaliação** dependerá de:

- histórico de eventos idempotente e isolado por usuário;
- impressões qualificadas com posição e contexto do ranking;
- feedback explícito com semântica distinta para like, dislike, save e hide;
- eventos do pipeline consistentes com transições reais;
- versões identificáveis de corpus, analyzer e ranking;
- projeção atual de feedback reconstruível a partir do histórico.

O CP23 continuará preservando o BM25 como score-base e atuará depois dele, como reranking individual. Nenhuma funcionalidade de personalização será considerada entregue apenas pela instrumentação do CP22.

**Gate final do CP22:** o OpenJob captura exposição real, atenção ativa, preferências explícitas e transições do pipeline com histórico idempotente, contexto de ranking e isolamento multiusuário. O feed permanece funcional mesmo quando a telemetria falha. A entrega fornece ao CP23 dados confiáveis sem implementar ou alterar o ranking-base.
