# Checkpoint 21 — Tailoring determinístico

> **Estimativa inicial:** a definir após auditoria do fluxo atual de Tailoring, dos contratos da F3, do NLP Core da F5/B11, dos resultados do CP20 e dos perfis/provedores de IA da F4. Não fixar prazo antes de identificar o que pode ser reutilizado e medir os principais riscos.
> **Status:** proposto.
> **Pré-requisitos:** CP18 concluído nos pontos de autenticação, autorização e ownership; CP20 concluído e com contratos canônicos estáveis; F3 disponível para auditoria e integração; F4 disponível para reutilização dos perfis de IA; F5/B11 acessível por contratos; migrations e suítes atuais verdes.
> **Responsabilidade:** analisar currículo e vaga, mapear evidências, construir estratégia reproduzível, gerar uma proposta controlada e validar sua fidelidade antes da aprovação do usuário.
> **Relação com CP20:** o CP21 consome documentos canônicos, entidades e referências de origem. Não implementa parsers, OCR, classificação documental genérica ou infraestrutura própria de ingestão.
> **Relação com CP18:** identidade, sessão e ownership são fornecidos pelo CP18. O CP21 deriva o proprietário da sessão autenticada e não aceita `user_id` livre do frontend.
> **Relação com F3/F4/F5:** reutiliza a experiência atual de Tailoring, cópia, exportação, Markdown e erros da F3; os perfis/provedores de IA da F4; e o NLP Core da F5/B11. Não recria essas capacidades.
> **Relação com CP22/CP23:** o Score Fit — Parte 2 permanece separado. O CP21 não registra sinais comportamentais, altera o BM25 ou implementa personalização do ranking.

## 1. Objetivo

Entregar um fluxo completo de Tailoring que adapte o currículo-base de um usuário a uma vaga por meio de análise estruturada, correspondência rastreável de evidências, estratégia determinística, geração controlada e validação factual.

O sistema deve identificar quais requisitos da vaga encontram suporte no currículo, quais possuem relação indireta, quais não possuem evidência e quais dependem de confirmação. A estratégia deve selecionar, priorizar e ordenar conteúdo por regras versionadas e explicáveis. A geração poderá reformular linguagem e organização, inclusive com uso controlado dos provedores de IA já configurados na F4, mas não poderá introduzir fatos sem suporte documental.

O currículo-base, a proposta gerada, os resultados da validação e a versão aprovada devem permanecer separados. Nenhuma execução pode substituir o currículo-base nem transformar uma proposta ainda não validada em versão aprovada sem ação explícita do usuário.

**Princípio central:** a análise, o mapeamento de evidências, a estratégia, a ordenação e a validação são determinísticos para as mesmas entradas e versões de regras. A redação natural pode variar quando utilizar LLM, mas toda afirmação factual precisa estar ancorada em evidência ou ficar pendente de confirmação.

## 2. Escopo e decisões arquiteturais

### 2.1 Responsabilidades

- **CP18:** identidade, sessão, autorização, perfil profissional, currículo-base e ownership. O CP21 consome os dados autorizados do usuário e não implementa autenticação nem gestão geral de contas.
- **CP20 — Inteligência de documentos:** fornece documento canônico, estrutura, entidades, trechos e referências de origem. O CP21 consome esses contratos e não cria parsers, OCR ou uma representação documental paralela.
- **CP21:** Resume Analyzer, Job Analyzer, Evidence Mapping Engine, Tailoring Strategy Engine, Tailoring Generator controlado, Validation Engine, fachada de serviço, integração de API/UI e testes do fluxo completo.
- **F3:** fornece o fluxo atual de Tailoring, upload/conversão, cópia, exportação, renderização Markdown e catálogo amigável de erros. O CP21 deve auditar e evoluir a implementação existente, preservando funcionalidades aceitas e evitando uma segunda experiência concorrente.
- **F4:** fornece BYOK, presets, free tier, perfis de IA e configuração/teste de provedores. O CP21 consome os perfis existentes e não cria outro cadastro de provedores, armazenamento de API keys ou fluxo OAuth.
- **F5/B11:** fornece NLP Core, normalização linguística, tokenização, recursos de idioma, keywords e contratos genéricos. O CP21 reutiliza esses recursos quando aplicáveis e não cria outro tokenizer, catálogo de stopwords ou motor NLP.
- **CP19:** pode oferecer a entrada guiada para currículo-base, mas não implementa regras de análise, evidência, estratégia ou geração do Tailoring.
- **CP22 — Score Fit: captura comportamental:** registra exposições e interações. O CP21 não altera o event log nem depende de sinais comportamentais para decidir quais evidências existem no currículo.
- **CP23 — Score Fit: personalização e avaliação:** mantém o reranking individual separado do score-base. O CP21 não modifica BM25 nem o ranking global de vagas.
- **CP11:** continua responsável por instalação, bootstrap técnico, diretórios e operação do executável. O CP21 utiliza os mecanismos existentes.

### 2.2 Limites entre determinismo e geração de linguagem

As responsabilidades devem ser separadas explicitamente:

1. **Análise do currículo:** estrutura fatos e referências presentes no currículo-base.
2. **Análise da vaga:** estrutura requisitos e responsabilidades presentes na descrição da vaga.
3. **Evidence Mapping:** relaciona requisitos e evidências, sem transformar semelhança lexical em comprovação.
4. **Estratégia:** escolhe e ordena conteúdo por regras determinísticas, versionadas e explicáveis.
5. **Geração:** reformula e organiza somente evidências e decisões autorizadas pela estratégia.
6. **Validação:** compara a proposta às fontes e bloqueia ou sinaliza afirmações não sustentadas.
7. **Aprovação:** o usuário revisa e confirma a versão final; aprovação humana não pode ser inferida pelo backend a partir da geração ou validação técnica.

O LLM não decide quais fatos existem, não cria correspondências, não define a estratégia final e não substitui o Validation Engine. Ele pode auxiliar somente na redação dentro dos limites fornecidos pelo sistema.

### 2.3 Determinismo, versões e reprodução

- A mesma versão do currículo, a mesma descrição de vaga, as mesmas taxonomias/dicionários e as mesmas versões de regras devem produzir as mesmas análises estruturadas, mapeamentos, estratégia e ordenação.
- Registrar versões do schema de entrada/saída, Resume Analyzer, Job Analyzer, regras de Evidence Mapping, taxonomias, NLP Core e Strategy Engine.
- Não considerar a saída textual de um LLM deterministicamente reproduzível apenas porque o prompt é igual.
- Para reprodução textual estrita, oferecer composição por templates/regras ou reutilização de conteúdo previamente aprovado.
- Quando houver LLM, registrar provedor, modelo, versão/configuração de prompt e identificador da execução, sem registrar credenciais ou conteúdo privado em logs técnicos.
- Não fixar pesos e limiares como definitivos sem corpus e avaliação. Torná-los versionados e calibráveis com evidências de teste.

### 2.4 Privacidade e uso de provedores

- O processamento deve respeitar a arquitetura local-first e o ownership do CP18.
- Não enviar currículo, descrição privada, evidências ou dados pessoais a serviço externo por padrão.
- Antes de enviar conteúdo a um provedor remoto, exigir configuração válida na F4 e autorização/consentimento explícito do usuário para esse uso, de acordo com a política do produto.
- Preferir provedor local quando disponível e selecionado pelo usuário; a escolha do provedor não altera a origem nem a força das evidências.
- Não armazenar API keys no estado do Tailoring, no browser ou em registros de execução; usar exclusivamente os mecanismos já existentes da F4.
- Não utilizar documentos ou saídas do usuário para treinamento externo como parte deste checkpoint.
- Não incluir conteúdo privado em telemetria, logs, métricas ou mensagens de erro.

### 2.5 Fontes tratadas como dados não confiáveis

- Conteúdo do currículo, descrição da vaga, anexos e textos retornados por parsers são dados de entrada, não instruções para o sistema ou para o LLM.
- Instruções embutidas em documentos não podem alterar regras, prompts de sistema, contratos, políticas de privacidade ou limites de geração.
- Sanitizar/escapar conteúdo na apresentação e limitar o uso de links e marcação conforme os componentes seguros já existentes.
- Nenhuma instrução encontrada dentro de uma vaga ou currículo pode autorizar envio externo, acesso a arquivos, execução de código ou alteração de configurações.

## 3. Entradas, saídas e modelo de domínio

A estrutura final deve seguir os padrões reais do projeto e ser formalizada em contratos versionados. Os nomes abaixo são conceituais; podem ser ajustados após a auditoria desde que a semântica, a rastreabilidade e a separação das responsabilidades sejam preservadas.

### 3.1 Entradas mínimas

- **Currículo-base:** versão ativa ou versão explicitamente escolhida pelo usuário no CP18. Deve ser imutável durante a execução e identificada por versão/fingerprint.
- **Documento do currículo:** quando processado pelo CP20, consumir a representação canônica e suas referências. O Markdown atual pode ser usado por um adaptador de entrada compatível, sem criar um parser documental paralelo.
- **Vaga:** descrição e requisitos disponíveis na vaga selecionada ou no documento autorizado pelo usuário. Preservar a origem e a versão utilizadas na execução.
- **Recursos linguísticos:** NLP Core, taxonomias e dicionários aprovados, todos identificados por versão.
- **Regras de Tailoring:** pesos, limiares, regras de seleção, ordenação e composição, versionados e explicáveis.
- **Preferências de saída:** idioma, formato e opções de apresentação suportadas pelo produto. Essas preferências não podem modificar fatos nem a evidência disponível.

### 3.2 Entidades/contratos conceituais

- `TailoringRun`: identidade da execução, usuário, referências de entrada, versões de regras/motores, estado, timestamps e resultado.
- `ResumeAnalysis`: fatos e estruturas reconhecidos no currículo, com origem e distinção entre declaração explícita, normalização e incerteza.
- `JobAnalysis`: requisitos, responsabilidades, qualificações e demais elementos identificados na vaga, cada qual com trecho-fonte e classificação de certeza.
- `EvidenceLink`: relação entre um requisito e uma ou mais evidências do currículo, com tipo de correspondência, justificativa, confiança quando aplicável e referências de origem.
- `TailoringStrategy`: conteúdo selecionado, excluído, resumido ou reordenado, prioridades, razões e versão das regras.
- `TailoringBlock`: bloco da proposta, texto, ordem, origem, evidências associadas e estado de validação.
- `ValidationFinding`: achado do Validation Engine, categoria, severidade, trecho afetado, evidências consultadas, justificativa e ação requerida.
- `TailoringDraft`: proposta gerada, distinta do currículo-base e da versão aprovada.
- `ApprovedTailoring`: versão explicitamente aprovada pelo usuário, preservada com referência à proposta e validação que lhe deram origem.

### 3.3 Regras obrigatórias dos contratos

- [ ] Versionar schemas de análise, mapeamento, estratégia, proposta e validação.
- [ ] Preservar IDs/referências de currículo, vaga, documento e execução usados como entrada.
- [ ] Preservar o vínculo dos fatos do currículo e dos requisitos da vaga com os `SourceReference` do CP20, quando disponíveis.
- [ ] Manter conteúdo original separado de texto normalizado, interpretação e texto gerado.
- [ ] Diferenciar campos ausentes, desconhecidos, não aplicáveis e explicitamente declarados.
- [ ] Não representar baixa confiança como fato confirmado.
- [ ] Exigir que cada `TailoringBlock` factual indique suas evidências de origem ou permaneça pendente de confirmação.
- [ ] Registrar as versões de analisadores, taxonomias, regras e configuração de geração relevantes para reproduzir a decisão.
- [ ] Preservar currículo-base, proposta e versão aprovada como entidades/estados distintos.
- [ ] Permitir comparação entre execuções sem sobrescrever resultados anteriores.
- [ ] Garantir ownership no backend, derivado da sessão autenticada.
- [ ] Fornecer exemplos versionados para casos completos, dados ausentes, ambiguidades, conflitos e falhas.

### 3.4 Estados do fluxo

Definir uma máquina de estados coerente com os serviços existentes. Os nomes abaixo são conceituais:

- `created`: execução registrada com entradas identificadas;
- `analyzing`: análise do currículo e da vaga em andamento;
- `strategy_ready`: evidências e estratégia disponíveis;
- `generating`: proposta textual em produção;
- `validation_pending`: proposta aguardando validação;
- `needs_confirmation`: há ambiguidades ou achados que exigem revisão do usuário;
- `validated`: não há bloqueios técnicos pendentes segundo as regras vigentes;
- `approved`: usuário confirmou a versão final;
- `failed`: falha controlada, com código e possibilidade de retry quando aplicável;
- `cancelled`: execução cancelada sem alterar currículo-base ou versão aprovada.

- [ ] Definir transições válidas e idempotência de operações.
- [ ] Não marcar `approved` somente porque a geração ou validação terminou.
- [ ] Distinguir falha técnica, bloqueio factual e confirmação pendente.
- [ ] Permitir retry seguro sem duplicar propostas nem sobrescrever versões anteriores.
- [ ] Preservar o estado e o resultado parcial útil quando uma etapa posterior falhar.

## 4. Motores do Tailoring

### 4.1 TD-01 — Contratos de entrada e resultado

- [ ] Auditar os modelos, serviços, handlers, prompts, armazenamento e respostas do Tailoring atual na F3.
- [ ] Identificar quais contratos já podem ser mantidos, quais precisam de extensão e quais são incompatíveis com o modelo rastreável.
- [ ] Definir contrato de entrada para currículo-base, documento normalizado, vaga, requisitos, idioma e versões dos recursos utilizados.
- [ ] Definir saída versionada para análise, evidências, estratégia, blocos da proposta, validações e estado de aprovação.
- [ ] Registrar ordem, texto, origem, referências de evidência e estado de validação de cada bloco.
- [ ] Preservar currículo original, proposta e versão aprovada em entidades/estados separados.
- [ ] Reutilizar o contrato canônico do CP20 e os recursos genéricos da F5/B11.
- [ ] Definir compatibilidade/migração com os endpoints e dados de Tailoring existentes, evitando quebra desnecessária de clientes.
- [ ] Registrar ADR dos contratos e do ciclo de vida da execução.

**Validação TD-01:** uma execução pode ser auditada e comparada; entradas, versões, seleção de evidências, blocos e validação permanecem identificáveis sem depender de um provedor LLM.

### 4.2 TD-02 — Resume Analyzer

- [ ] Consumir o currículo-base ativo/selecionado e as estruturas canônicas do CP20.
- [ ] Manter suporte ao currículo Markdown atual enquanto houver conteúdo legado que ainda não tenha representação documental completa, usando adaptador delimitado e testado.
- [ ] Identificar seções, experiências, cargos, organizações, períodos, formação, skills, certificações, idiomas, resultados e demais campos realmente presentes.
- [ ] Preservar o trecho original e a localização de origem para cada informação extraída.
- [ ] Normalizar nomes, datas e termos sem apagar nem substituir o valor original.
- [ ] Diferenciar fato declarado, normalização, interpretação de baixa confiança e informação ausente.
- [ ] Não completar datas, cargos, senioridade, certificações, ferramentas, competências ou métricas por inferência não sustentada.
- [ ] Tratar conflitos internos do currículo como inconsistências a revisar, sem escolher silenciosamente um dos valores.
- [ ] Registrar versão do analyzer, NLP Core, taxonomias e configuração aplicadas.
- [ ] Permitir reprocessamento quando o currículo ou as versões dos recursos mudarem, mantendo a análise anterior para comparação.

**Validação TD-02:** informações consumidas pela estratégia são rastreáveis ao currículo-base; ausência e ambiguidade são representadas explicitamente e não preenchidas por suposição.

### 4.3 TD-03 — Job Analyzer

- [ ] Consumir a descrição da vaga e, quando disponível, o documento canônico do CP20.
- [ ] Decompor o conteúdo em requisitos, responsabilidades, qualificações, skills, senioridade, experiência, idiomas, modalidade e localização quando presentes.
- [ ] Preservar o trecho-fonte de cada requisito e sua referência de origem.
- [ ] Classificar requisito como obrigatório ou desejável somente quando houver evidência textual suficiente; casos ambíguos permanecem incertos.
- [ ] Distinguir texto explícito, normalização terminológica e interpretação do sistema.
- [ ] Não inferir senioridade, faixa salarial, modalidade, localização ou requisito não declarado.
- [ ] Identificar duplicidade e possíveis contradições sem apagar o conteúdo original.
- [ ] Não tratar conteúdo promocional, instruções embutidas ou texto malicioso como comandos do sistema.
- [ ] Registrar versão do analyzer, recursos linguísticos e regras de classificação.

**Validação TD-03:** cada requisito estruturado aponta para o trecho que o originou; incertezas não são convertidas em requisitos confirmados.

### 4.4 TD-04 — Evidence Mapping Engine

- [ ] Relacionar requisitos da vaga às evidências identificadas no currículo.
- [ ] Implementar as classes de relação previstas: `evidencia_direta`, `evidencia_relacionada`, `sem_evidencia` e `confirmacao_necessaria` (nomes finais compatíveis com as convenções do projeto).
- [ ] Guardar requisito, trecho do currículo, seção/experiência de origem, tipo de correspondência, justificativa, confiança quando calculada e versão das regras.
- [ ] Distinguir correspondência lexical, sinônimo/taxonomia e relação semântica; não apresentá-las como equivalentes.
- [ ] Não converter proximidade lexical em comprovação de experiência, senioridade ou domínio prático.
- [ ] Exigir confirmação quando a relação depender de interpretação sem suporte documental claro.
- [ ] Não inserir, promover ou reclassificar como comprovadas competências classificadas como ausentes.
- [ ] Permitir múltiplas evidências para um requisito e múltiplos requisitos sustentados por uma evidência, mantendo cada vínculo explícito.
- [ ] Tratar evidências conflitantes como conflito/atenção, não como autorização para escolher o dado mais favorável.
- [ ] Preservar a referência ao CP20 e ao texto original mesmo quando o termo tiver sido normalizado.
- [ ] Criar golden tests para correspondência direta, sinônimos, falsos positivos lexicais, ausência e ambiguidade.

**Validação TD-04:** toda relação de evidência é explicável e rastreável; correspondência lexical isolada não comprova experiência e lacunas permanecem lacunas.

### 4.5 TD-05 — Tailoring Strategy Engine

- [ ] Criar motor de estratégia separado dos analisadores, do gerador textual e dos handlers HTTP.
- [ ] Priorizar requisitos e evidências por regras versionadas, explicáveis e testáveis.
- [ ] Selecionar experiências, skills e resultados relevantes sem alterar fatos, valores ou contexto.
- [ ] Definir quais seções devem ser mantidas, reordenadas, resumidas ou destacadas, respeitando a estrutura e o conteúdo do currículo-base.
- [ ] Evitar repetição, excesso de keywords e inclusão de termos sem contexto probatório.
- [ ] Tratar lacunas como lacunas; não disfarçá-las como experiência ou domínio demonstrado.
- [ ] Produzir razões de inclusão, exclusão, prioridade, resumo e ordenação.
- [ ] Permitir que regras considerem importância do requisito, força da evidência e relevância da experiência, sem expor pesos arbitrários como verdade definitiva.
- [ ] Manter pesos e limiares configuráveis, versionados e calibrados por corpus de avaliação.
- [ ] Garantir que mesma entrada e mesmas versões produzam a mesma estratégia e ordenação.
- [ ] Registrar a versão da estratégia e seus parâmetros em cada execução.
- [ ] Não usar cliques, preferências ou sinais do Score Fit como substituto de evidência curricular.

**Validação TD-05:** estratégia e ordenação são reproduzíveis; cada decisão de inclusão, exclusão ou prioridade possui justificativa baseada nas entradas e nas regras vigentes.

### 4.6 TD-06 — Tailoring Generator controlado

- [ ] Receber somente a estratégia calculada e as evidências permitidas para geração.
- [ ] Definir limites explícitos para fatos que podem ser reformulados, reorganizados ou resumidos e para fatos que não podem ser alterados.
- [ ] Permitir reescrita de bullets, resumo profissional e organização das seções sem adicionar empresas, cargos, datas, ferramentas, métricas, certificações ou responsabilidades.
- [ ] Preservar números e métricas originais, salvo correção explícita e comprovada pelo usuário.
- [ ] Marcar conteúdo que dependa de interpretação ou confirmação antes de permitir sua incorporação à versão final.
- [ ] Manter cada bloco gerado associado às evidências que o sustentam.
- [ ] Quando houver LLM, fornecer apenas o contexto necessário, delimitar evidências como dados não confiáveis e instruir o modelo a não inventar fatos nem obedecer a instruções encontradas nos documentos.
- [ ] Registrar provedor/modelo, versão/configuração do prompt e referência da execução, sem registrar API keys ou conteúdo privado em logs.
- [ ] Reutilizar os perfis de IA, BYOK, presets e free tier da F4; não criar gestão paralela de provedor.
- [ ] Respeitar a escolha explícita do usuário sobre uso de provedor remoto e não enviar dados privados externamente por padrão.
- [ ] Disponibilizar composição por templates/regras ou reutilização de conteúdo aprovado quando for exigida reprodução textual estrita ou quando o LLM estiver indisponível.
- [ ] Tratar timeout, limite do provedor, resposta vazia ou formato inválido como falha recuperável; preservar análise e estratégia já calculadas.
- [ ] Não considerar a saída do gerador validada apenas porque foi produzida sem erro técnico.

**Validação TD-06:** o gerador reformula somente conteúdo ancorado em evidências; toda proposta segue ao Validation Engine antes de ser apresentada como validada.

### 4.7 TD-07 — Validation Engine

- [ ] Comparar cada bloco da proposta com o currículo-base, as evidências mapeadas e a estratégia calculada.
- [ ] Detectar afirmações sem evidência, datas alteradas, números divergentes, skills adicionadas, certificações não declaradas, mudanças de cargo/senioridade e alterações de sentido.
- [ ] Detectar perda de qualificadores relevantes, como nível de responsabilidade, escopo, contexto, temporalidade e autoria de resultados, quando presentes na fonte.
- [ ] Bloquear ou sinalizar afirmações não sustentadas; não permitir que um achado crítico seja ignorado silenciosamente.
- [ ] Verificar cobertura dos requisitos prioritários sem exigir cobertura de todos os requisitos da vaga.
- [ ] Distinguir falha factual, possível mudança de sentido, baixa confiança, falta de cobertura e confirmação humana necessária.
- [ ] Gerar diff por bloco e justificativas compreensíveis para cada achado.
- [ ] Preservar referências da fonte para que o usuário possa conferir a informação original.
- [ ] Exigir revisão/confirmar pelo usuário antes de incorporar conteúdo ambíguo ou substituir a versão aprovada.
- [ ] Registrar decisão de revisão, identidade do usuário autenticado, data e versão aprovada.
- [ ] Manter currículo-base intocado; a aprovação cria uma versão de Tailoring separada.
- [ ] Executar validações determinísticas mesmo quando a geração vier de templates ou de LLM.
- [ ] Não usar a aprovação humana como licença para apagar a proveniência ou alterar a fonte original.

**Validação TD-07:** nenhuma afirmação factual sem suporte passa silenciosamente; o usuário consegue revisar diffs, evidências e pendências antes de aprovar.

### 4.8 TD-08 — API, interface e integração

- [ ] Auditar endpoints, componentes, estados e persistência do Tailoring atual antes de desenhar novos contratos.
- [ ] Integrar motores por uma fachada/serviço de Tailoring; handlers não devem conter regras de análise, seleção de evidências ou validação.
- [ ] Preservar endpoints atuais quando possível; versionar mudanças incompatíveis e documentar migração.
- [ ] Oferecer operações para iniciar execução, consultar estado/resultado, repetir etapa recuperável, revisar achados e aprovar versão final, ajustadas aos padrões reais do projeto.
- [ ] Derivar usuário da sessão CP18; validar ownership em leitura, edição, exportação, exclusão e aprovação.
- [ ] Exibir análise da vaga, evidências, lacunas, estratégia resumida, proposta, diff, estado de validação e confirmações necessárias.
- [ ] Distinguir visualmente proposta gerada, proposta validada e versão aprovada.
- [ ] Permitir ao usuário revisar conteúdo e pendências sem perder o currículo-base nem resultados de execuções anteriores.
- [ ] Reutilizar cópia, exportação, renderização Markdown e catálogo de erros entregues/revisados na F3.
- [ ] Preservar PDF/TXT/MD e os formatos de saída que a F3 efetivamente suportar; não prometer novos formatos neste checkpoint.
- [ ] Manter responsividade, acessibilidade e i18n conforme os padrões existentes.
- [ ] Preservar o fluxo atual durante migração gradual, com rollback/feature flag quando a arquitetura de release exigir.
- [ ] Tratar indisponibilidade de LLM como falha localizada da geração; análise, evidências e estratégia devem continuar consultáveis quando já concluídas.

**Validação TD-08:** usuário consegue acompanhar e revisar o ciclo completo; APIs e interface consomem os mesmos contratos, sem duplicar motores nem alterar silenciosamente o fluxo atual.

### 4.9 TD-09 — Testes e gate

- [ ] Criar golden tests para Resume Analyzer, Job Analyzer, Evidence Mapping, estratégia, ordenação e regras do Validation Engine.
- [ ] Criar corpus versionado com currículos e vagas representativos, incluindo PT-BR, EN-US e ES-419 conforme a cobertura linguística suportada.
- [ ] Criar testes negativos para skill ausente, certificação não declarada, datas conflitantes, métrica inventada, senioridade inferida sem evidência e mudança de sentido.
- [ ] Verificar que cada afirmação factual da proposta aponta para uma fonte ou fica pendente de confirmação.
- [ ] Verificar estabilidade da análise/estratégia para a mesma entrada e as mesmas versões.
- [ ] Testar diferenças de saída quando versões de regras/taxonomias mudarem, sem sobrescrever execuções anteriores.
- [ ] Testar falha, timeout, indisponibilidade e resposta inválida de LLM sem corromper currículo-base, análise ou estratégia.
- [ ] Testar modo por templates/regras e comportamento sem LLM configurado.
- [ ] Testar autorização e isolamento entre usuários em APIs, serviços, documentos, propostas, exports e aprovações (CP18).
- [ ] Testar que conteúdo de currículo/vaga com instruções embutidas não altera regras nem autoriza envio externo.
- [ ] Testar exportação, cópia, Markdown, acessibilidade, responsividade e regressão da F3.
- [ ] Executar `make test`, `make vet`, `make test-ui` e testes de migrations/ownership pertinentes.
- [ ] Registrar corpus, resultados, limitações conhecidas e critérios de qualidade aprovados; não inventar metas numéricas sem baseline medido.

**Gate TD:** análise, mapeamento e estratégia reproduzíveis; evidências rastreáveis; nenhuma afirmação factual sem suporte é aprovada silenciosamente; revisão humana funciona para ambiguidades; integração não regride o fluxo atual e todos os testes obrigatórios passam.

## 5. Persistência, ciclo de vida e contratos de serviço

### 5.1 Persistência

A implementação deve respeitar o banco e os padrões atuais do projeto. A auditoria definirá se cada conceito exige tabela própria, extensão de estruturas existentes ou armazenamento versionado equivalente.

- [ ] Persistir execuções vinculadas ao usuário autenticado e às versões de currículo/vaga utilizadas.
- [ ] Persistir resultados de análise, Evidence Mapping e estratégia com referência às versões dos motores e regras.
- [ ] Persistir propostas em estado separado do currículo-base e da versão aprovada.
- [ ] Persistir achados de validação e decisões de revisão necessárias para explicar o resultado final.
- [ ] Garantir idempotência nas operações de início/retry e evitar duplicação acidental de execuções.
- [ ] Preservar histórico suficiente para comparação e auditoria, com política explícita de retenção/exclusão.
- [ ] Criar índices e constraints conforme consultas e ownership reais; não adicionar `user_id` a tabelas globais sem justificativa de modelagem.
- [ ] Garantir que excluir ou atualizar um currículo-base não reescreva versões históricas de Tailoring já produzidas.
- [ ] Não duplicar o documento original do CP20 sem necessidade; armazenar referências versionadas quando o contrato permitir.

### 5.2 Contratos de serviço

A arquitetura final deve seguir os padrões internos existentes. Os serviços conceituais abaixo servem para delimitar responsabilidades, não para impor nomes de pacotes ou tecnologias:

- `ResumeAnalyzer`: converte currículo-base/documento canônico em fatos estruturados e rastreáveis.
- `JobAnalyzer`: converte descrição da vaga em requisitos estruturados e rastreáveis.
- `EvidenceMapper`: associa requisitos às evidências do currículo e classifica a relação.
- `StrategyEngine`: escolhe, prioriza e ordena conteúdo por regras versionadas.
- `TailoringGenerator`: produz proposta textual a partir da estratégia e das evidências permitidas.
- `ValidationEngine`: valida proposta, produz achados e determina bloqueios/necessidade de revisão.
- `TailoringService` ou fachada equivalente: coordena etapas, persistência, estados, autorização e contratos de API.

- [ ] Definir dependências explícitas entre serviços e impedir ciclos.
- [ ] Manter serviços puros/determinísticos sempre que a tarefa não exigir I/O ou geração externa.
- [ ] Isolar chamadas de provedores de IA no componente de geração, utilizando F4.
- [ ] Não permitir que gerador ou handler altere diretamente o currículo-base.
- [ ] Não permitir que a UI determine o estado factual de validação ou aprovação.

### 5.3 API conceitual

Ajustar aos padrões e rotas existentes; os exemplos abaixo não são uma ordem para criar endpoints duplicados:

- iniciar uma execução de Tailoring para currículo e vaga selecionados;
- consultar estado, análises, evidências, estratégia, proposta e validação;
- solicitar retry de uma etapa recuperável;
- registrar revisão/confirmar achados;
- aprovar uma versão final;
- consultar, copiar e exportar uma versão aprovada.

O backend deve:

- derivar o usuário da sessão CP18;
- validar que currículo, vaga/documento e execução podem ser acessados no contexto daquele usuário;
- validar transições de estado e versões de entrada;
- não aceitar do frontend afirmações de evidência, validação ou aprovação como verdade;
- executar ou consultar o Validation Engine antes de permitir aprovação;
- garantir que a aprovação se refira exatamente à proposta e à validação visualizadas pelo usuário;
- impedir replay de aprovação sobre versão diferente ou desatualizada.

## 6. Segurança, qualidade e regras de produto

### 6.1 Segurança e privacidade

- [ ] Aplicar ownership em todas as operações de Tailoring por meio da sessão autenticada.
- [ ] Verificar acesso a currículos, documentos CP20, vagas privadas, execuções, propostas e exports.
- [ ] Não confiar em IDs fornecidos pelo frontend sem verificar sua relação com o usuário e com a execução.
- [ ] Não registrar currículos, descrições completas, prompts com conteúdo privado, tokens ou API keys em logs.
- [ ] Redigir/mascarar dados em traces e relatórios operacionais.
- [ ] Não enviar dados a provedor remoto sem configuração e autorização explícita aplicáveis.
- [ ] Tratar entradas documentais como conteúdo não confiável e prevenir prompt injection na etapa LLM.
- [ ] Sanitizar conteúdo apresentado e exportado conforme as regras de segurança da F3.
- [ ] Garantir que falha de provider ou parser não exponha dados de outro usuário.

### 6.2 Qualidade e factualidade

- [ ] Toda afirmação factual deve ter evidência rastreável ou estado explícito de confirmação pendente.
- [ ] Termos equivalentes e sinônimos não comprovam, por si só, experiência prática.
- [ ] Métricas, percentuais, datas, nomes, escopo de responsabilidade e resultados não podem ser ampliados ou alterados para melhorar a apresentação.
- [ ] A ausência de evidência não deve ser tratada como prova de que o usuário não possui a competência; deve ser registrada como ausência no material fornecido.
- [ ] A cobertura de requisitos não deve forçar a inclusão de conteúdo irrelevante ou não comprovado.
- [ ] Resultados devem explicar lacunas e incertezas sem declarar que o candidato não possui uma capacidade quando o currículo apenas não a documenta.
- [ ] O usuário deve conseguir comparar a versão proposta com o currículo-base e entender o motivo das mudanças.

### 6.3 Regressão e compatibilidade

- [ ] Preservar a experiência de Tailoring atual enquanto os novos motores são integrados.
- [ ] Reutilizar controles e formatos de saída existentes na F3.
- [ ] Manter compatibilidade com os documentos Markdown existentes durante a transição.
- [ ] Versionar mudanças incompatíveis em contratos e dados persistidos.
- [ ] Prever rollback do caminho de geração/validação sem apagar resultados de execução.

## 7. Critérios de verificação e aceite

- [ ] O fluxo utiliza currículo-base e vaga identificados por versões e acessíveis ao usuário autenticado.
- [ ] Resume Analyzer extrai apenas informações presentes no currículo e preserva a fonte de cada fato relevante.
- [ ] Job Analyzer estrutura requisitos e responsabilidades preservando trechos-fonte e incertezas.
- [ ] Obrigatório/desejável só é definido quando houver evidência suficiente; ambiguidade permanece explícita.
- [ ] Evidence Mapping distingue evidência direta, relacionada, ausente e confirmação necessária.
- [ ] Correspondência lexical isolada não é apresentada como comprovação de experiência.
- [ ] Toda evidência pode ser rastreada até currículo/documento, seção e trecho de origem disponíveis.
- [ ] Strategy Engine produz seleção e ordenação reproduzíveis para as mesmas entradas e versões.
- [ ] Razões de inclusão, exclusão, prioridade e ordenação são consultáveis.
- [ ] Pesos e limiares são versionados e não tratados como definitivos sem avaliação.
- [ ] Generator utiliza somente evidências e decisões permitidas pela estratégia.
- [ ] Uso de LLM é opcional, controlado pela F4 e sujeito à autorização explícita para envio remoto.
- [ ] Modo por templates/regras ou conteúdo aprovado permanece disponível quando reprodução estrita ou ausência de LLM exigir.
- [ ] Generator não adiciona empresas, cargos, datas, ferramentas, métricas, certificações, skills ou responsabilidades sem suporte.
- [ ] Validation Engine detecta e sinaliza afirmações sem evidência, divergências numéricas, datas alteradas e mudanças de sentido cobertas pelos testes.
- [ ] Diffs, evidências, lacunas e confirmações necessárias são apresentados antes da aprovação.
- [ ] Nenhuma versão proposta é considerada aprovada sem ação explícita do usuário.
- [ ] Currículo-base permanece intacto em falha, retry, geração, validação e aprovação.
- [ ] Proposta, validação e versão aprovada permanecem separadas e versionadas.
- [ ] Falha ou indisponibilidade de LLM não corrompe a análise, a estratégia ou o currículo-base.
- [ ] Nenhum usuário acessa análises, propostas, documentos ou exports de outro usuário.
- [ ] A integração preserva cópia, exportação, Markdown, acessibilidade e o fluxo atual da F3.
- [ ] `make test`, `make vet`, `make test-ui` e testes pertinentes de migrations/ownership passam.

## 8. Testes obrigatórios

### 8.1 Backend e contratos

- [ ] Contratos versionados e exemplos de entrada/saída válidos, incompletos e ambíguos.
- [ ] Ownership para leitura, criação, retry, atualização, aprovação, exclusão e exportação.
- [ ] Rejeição de IDs de currículo, documento, vaga ou execução pertencentes a outro usuário.
- [ ] Transições válidas de estado, idempotência e retry sem duplicação.
- [ ] Integridade de referências de evidência para o CP20 e para o texto original.
- [ ] Execução reproduzível dos analisadores, Evidence Mapping e Strategy Engine por versão.
- [ ] Persistência separada de currículo-base, proposta e versão aprovada.
- [ ] Aprovação vinculada à versão exata da proposta e da validação.

### 8.2 Análise e evidências

- [ ] Skill presente com correspondência direta.
- [ ] Sinônimo/taxonomia que exige distinção de correspondência lexical.
- [ ] Termo lexicalmente semelhante sem evidência de experiência.
- [ ] Skill ausente do currículo.
- [ ] Certificação não declarada.
- [ ] Datas explícitas, incompletas e conflitantes.
- [ ] Senioridade não declarada ou inferida sem suporte.
- [ ] Métrica/percentual ausente ou divergente.
- [ ] Requisito obrigatório/desejável ambíguo.
- [ ] Experiências repetidas, requisitos duplicados e evidências conflitantes.
- [ ] Conteúdo em PT-BR, EN-US e ES-419 dentro da cobertura definida.

### 8.3 Geração e validação

- [ ] Reescrita fiel de bullet sem alteração de responsabilidade ou resultado.
- [ ] Tentativa de adicionar ferramenta, cargo, empresa, certificação, data ou métrica inexistente.
- [ ] Alteração de número, percentual, período ou unidade.
- [ ] Mudança de sentido na reformulação.
- [ ] Proposta com evidência insuficiente e encaminhamento para confirmação.
- [ ] Cobertura incompleta de requisitos sem forçar afirmações não sustentadas.
- [ ] Resposta vazia, inválida ou fora do formato esperado do LLM.
- [ ] Timeout, rate limit e indisponibilidade de provedor.
- [ ] Modo sem LLM e composição por templates/regras.
- [ ] Prompt injection/instrução maliciosa contida em currículo ou descrição da vaga.
- [ ] Retry e nova execução sem perda de proposta anterior ou currículo-base.

### 8.4 UI e regressão

- [ ] Apresentação de evidências, estratégia, proposta, diff e achados de validação.
- [ ] Estados de gerando, validando, atenção necessária, falha e aprovado.
- [ ] Confirmação explícita antes de aprovar versão final.
- [ ] Cópia e exportação em formatos efetivamente suportados na F3.
- [ ] Renderização segura de Markdown e conteúdos longos.
- [ ] Fluxo responsivo, teclado, leitor de tela e foco visível.
- [ ] Regressão das telas e fluxos atuais de currículo, Tailoring e feed.

## 9. Entregáveis

- Contratos versionados de entrada, análises, evidências, estratégia, proposta, validação e aprovação.
- Resume Analyzer integrado ao currículo-base/representação documental, com proveniência.
- Job Analyzer integrado à descrição da vaga/documento canônico, com requisitos rastreáveis.
- Evidence Mapping Engine com classificação de relação e justificativas.
- Tailoring Strategy Engine com seleção e ordenação determinísticas e versionadas.
- Tailoring Generator controlado, integrado aos perfis de IA da F4 e com alternativa por templates/regras.
- Validation Engine com detecção de afirmações sem suporte, divergências e mudanças de sentido.
- Fachada/serviço de Tailoring e integração com API, UI, cópia e exportação existentes.
- Persistência separada de currículo-base, proposta, resultados de validação e versão aprovada.
- Golden tests, corpus de referência, testes negativos, testes de ownership e relatório de qualidade.
- Documentação de operação, limitações, versões, decisões e estratégia de rollback.

## 10. Registrar

- ADR de contratos, ciclo de vida e versionamento das execuções de Tailoring.
- ADR dos limites entre NLP Core, CP20, analisadores de domínio, estratégia, geração e validação.
- Matriz de fatos/entidades do currículo e requisitos da vaga, com distinção entre declaração, normalização e incerteza.
- Catálogo de tipos de evidência e regras de Evidence Mapping.
- Registro das regras, pesos, limiares, taxonomias e versões da Strategy Engine.
- Política de uso de LLM, autorização de envio remoto, proteção contra prompt injection e modo sem LLM.
- Contrato de validação, severidades, bloqueios e fluxo de confirmação humana.
- Plano de persistência, retenção, retry, comparação de versões e rollback.
- Corpus de referência, golden tests, resultados, limitações e critérios de qualidade.
- Matriz de ownership e testes de isolamento associados ao CP18.
- Registro de compatibilidade com o Tailoring atual e componentes da F3.

## 11. Dependências, riscos e mitigação

| Risco/dependência | Impacto | Mitigação |
|---|---|---|
| Contrato do CP20 instável ou sem proveniência suficiente | Analisadores não conseguem apontar a origem das informações | Estabilizar referências/offsets do CP20; manter adaptador delimitado para Markdown legado e não duplicar parser. |
| Currículo contém informação incompleta ou conflitante | Análise pode selecionar fatos incorretos | Preservar origem e conflito; encaminhar ambiguidade para revisão em vez de escolher silenciosamente. |
| Descrição da vaga é vaga ou contraditória | Classificação de requisito pode ser incorreta | Preservar trechos, distinguir interpretação e deixar obrigatório/desejável incerto quando necessário. |
| Similaridade lexical gera falso positivo | Experiência não comprovada pode ser apresentada como requisito atendido | Separar lexical, taxonomia e evidência direta; cobrir com golden tests negativos. |
| Estratégia usa pesos arbitrários | Saídas instáveis ou sem justificativa | Versionar parâmetros e calibrar com corpus; não declarar limiares definitivos sem avaliação. |
| LLM acrescenta ou altera fatos | Currículo adaptado perde confiabilidade | Restringir contexto, associar evidências por bloco, validar toda saída e exigir confirmação de ambiguidades. |
| Provedor remoto recebe dados privados sem consentimento | Risco de privacidade | Não enviar por padrão; exigir configuração e autorização explícitas; preferir processamento local quando selecionado. |
| Prompt injection em currículo/vaga | Instruções da fonte tentam alterar comportamento do sistema | Tratar conteúdo como dado não confiável; separar instruções do sistema e validar saídas. |
| Falha do provedor bloqueia todo o fluxo | Usuário perde análise já calculada | Isolar geração; preservar análise/estratégia e oferecer modo por templates/regras quando aplicável. |
| Validation Engine não detecta mudança semântica sutil | Afirmação alterada pode chegar à proposta aprovada | Golden tests, diff por bloco, regras conservadoras e confirmação humana para ambiguidades. |
| Integração paralela à F3 | Duplicidade e regressão de UX | Auditar e estender fluxo existente, com migração gradual e rollback. |
| Ownership incompleto em exports ou retries | Vazamento de conteúdo entre usuários | Derivar identidade da sessão e testar todos os caminhos, inclusive workers e downloads. |
| Corpus pouco representativo | Testes aprovam apenas casos simples | Usar exemplos com layouts, idiomas, conflitos e conteúdo realista, documentando limitações. |

## 12. Fora de escopo

- Implementar autenticação, sessão, criação de usuários ou gestão geral de ownership (CP18).
- Implementar parsers, adaptadores de formato, OCR, ingestão, classificação documental genérica ou persistência documental canônica (CP20).
- Duplicar o NLP Core, tokenizer, normalizador, stopwords, extração de keywords ou BM25 da F5/B11.
- Implementar onboarding gráfico ou etapas de primeira execução (CP19).
- Criar BYOK, catálogo de provedores, presets, armazenamento de API keys ou OAuth de LLM; essas capacidades pertencem à F4.
- Alterar o algoritmo BM25, o candidate generation ou o score-base de aderência da Parte 1.
- Implementar timeline, captura de impressões, dwell time, event log ou feedback comportamental (CP22).
- Implementar perfil individual de preferências, re-ranking comportamental ou Evaluation Engine do Score Fit (CP23).
- Fazer scraping, buscar dados profissionais em serviços externos ou automatizar candidaturas.
- Inventar experiências, competências, certificações, cargos, datas, ferramentas, responsabilidades ou métricas para melhorar o currículo.
- Substituir revisão humana em casos de ambiguidade ou confirmação necessária.
- Substituir ou sobrescrever o currículo-base como consequência automática da geração ou aprovação.
- Introduzir treinamento de modelos, collaborative filtering ou aprendizado de ranking.

## 13. Sequência de implementação e gates internos

O CP21 é um único checkpoint, com execução em etapas internas. Os gates organizam a implementação e não criam checkpoints adicionais nem transferem cobertura para outro documento.

### Etapa A — Auditoria, contratos e analisadores (TD-01/TD-03)

- [ ] Auditar Tailoring atual, F3, contratos CP20, F5/B11, perfis F4 e persistência.
- [ ] Aprovar contratos de entrada/saída, versões e ciclo de vida.
- [ ] Implementar Resume Analyzer e Job Analyzer com proveniência.
- [ ] Validar distinção entre declaração, normalização, interpretação e ausência.

**Gate A:** análises estruturadas, versionadas e rastreáveis para currículo e vaga; nenhuma decisão de evidência ou estratégia é presumida nesta etapa.

### Etapa B — Evidence Mapping e estratégia (TD-04/TD-05)

- [ ] Implementar classificação de relações e justificativas.
- [ ] Implementar seleção, priorização e ordenação por regras determinísticas.
- [ ] Criar corpus e golden tests para falsos positivos, ausências, ambiguidades e conflitos.
- [ ] Calibrar parâmetros sem fixar pesos definitivos antes da avaliação.

**Gate B:** toda correspondência é rastreável e a mesma entrada/versões produz a mesma estratégia.

### Etapa C — Geração controlada e validação factual (TD-06/TD-07)

- [ ] Integrar geração via F4 com consentimento e limites de privacidade.
- [ ] Implementar composição por templates/regras quando aplicável.
- [ ] Implementar Validation Engine, diff, achados e confirmações necessárias.
- [ ] Testar tentativas de inserir ou alterar fatos e falhas/indisponibilidade do LLM.

**Gate C:** proposta gerada permanece separada; nenhuma afirmação factual sem suporte passa silenciosamente para aprovação.

### Etapa D — API, interface, persistência e aceite (TD-08/TD-09)

- [ ] Integrar a fachada de Tailoring aos endpoints e componentes existentes.
- [ ] Implementar persistência versionada e operações seguras de retry/revisão/aprovação.
- [ ] Reutilizar cópia, exportação, Markdown e catálogo de erros da F3.
- [ ] Executar testes E2E, acessibilidade, i18n, ownership e regressão.
- [ ] Documentar limites, versões, decisões e rollback.

**Gate D:** ciclo completo utilizável e auditável, sem regressão do fluxo atual e com todos os critérios finais de aceite atendidos.

## 14. Definições pendentes antes da estimativa final

- [ ] Auditar o código atual do Tailoring e separar componentes reutilizáveis de implementações que precisam ser substituídas.
- [ ] Confirmar quais formatos do CP20 estarão disponíveis no início do CP21 e como o Markdown legado será consumido.
- [ ] Confirmar o modelo de versões do currículo-base e a referência estável à vaga selecionada.
- [ ] Definir corpus de currículos e vagas para PT-BR, EN-US e ES-419 dentro da cobertura pretendida.
- [ ] Definir protocolo de avaliação de Resume Analyzer, Job Analyzer, Evidence Mapping, estratégia e validação.
- [ ] Selecionar taxonomias/dicionários e verificar licença, cobertura linguística, manutenção e integração ao NLP Core.
- [ ] Definir pesos/limiares iniciais somente após baseline e avaliação, mantendo-os configuráveis e versionados.
- [ ] Auditar os provedores/modelos e perfis F4 que podem ser usados para geração controlada, incluindo opção local e política de autorização remota.
- [ ] Definir severidades e condições de bloqueio do Validation Engine com casos de teste, sem limiares arbitrários.
- [ ] Definir retenção de execuções, propostas, achados e versões aprovadas.
- [ ] Confirmar contratos de API/UI existentes, plano de migração e rollback.

**Regra final de escopo:** TD-01 a TD-09 pertencem integralmente ao CP21. Análise, Evidence Mapping, estratégia, ordenação e validação devem ser determinísticos e versionados; a geração natural pode utilizar LLM de forma controlada, mas nenhuma afirmação factual sem evidência pode ser aprovada silenciosamente. O CP21 consome CP20, F4 e F5/B11, preserva o currículo-base do CP18 e integra o fluxo atual da F3 sem criar motores ou experiências paralelas.
