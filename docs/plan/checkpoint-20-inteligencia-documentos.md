# Checkpoint 20 — Inteligência de documentos

> **Estimativa inicial:** a definir após auditoria dos parsers, do fluxo de upload/conversão da F3, dos contratos de storage e dos recursos locais disponíveis para OCR. Não fixar prazo antes dessa auditoria e dos spikes técnicos.  
> **Status:** proposto.  
> **Pré-requisitos:** CP18 concluído nos pontos de autenticação, autorização e ownership; fluxo de upload/conversão da F3 disponível para auditoria; NLP Core da F5/B11 acessível por contratos reutilizáveis; baseline de testes e migrations verde.  
> **Responsabilidade:** processamento documental genérico, local-first e rastreável, desde a ingestão segura até a representação estruturada persistida e revisável.  
> **Relação com CP21:** o Tailoring determinístico consumirá os documentos canônicos e suas evidências, mas não implementará parsers, OCR ou infraestrutura documental própria.  
> **Relação com CP22/CP23:** Score Fit — Parte 2 não depende diretamente do pipeline documental; não deve ser acoplado à extração de documentos.  
> **Relação com CP18:** identidade, sessão e autorização são fornecidas pelo CP18. O CP20 deriva o proprietário da sessão autenticada e não aceita `user_id` livre do frontend.  
> **Relação com F3/F5:** reutilizar o upload, conversão, exportação e catálogo de erros existentes da F3, além dos recursos genéricos do NLP Core da F5/B11. Não criar implementações paralelas.

## 1. Objetivo

Entregar uma camada genérica de Inteligência de documentos capaz de receber arquivos com segurança, reconhecer seus formatos, extrair conteúdo, reconstruir sua estrutura, identificar entidades úteis, preservar a proveniência de cada informação e armazenar resultados versionados para consulta, revisão e reprocessamento.

A camada deve atender a diferentes consumidores do openJob — inicialmente currículo-base e descrição de vaga — sem incorporar regras específicas desses domínios. Seu resultado será uma representação canônica, estruturada e rastreável, capaz de preservar o documento original e distinguir conteúdo textual, normalização, extração, classificação e inferência.

O processamento deve permanecer local-first. Documentos e resultados privados ficam associados ao usuário autenticado e não são enviados a serviços externos por padrão. Recursos de OCR devem ser executados localmente sempre que tecnicamente viável, sem downloads silenciosos ou dependências externas obrigatórias na inicialização.

**Princípio central:** toda informação extraída deve manter sua relação com a fonte. O sistema não pode transformar ausência de informação em fato, confundir normalização com conteúdo original nem apresentar inferências como declarações explícitas do documento.

## 2. Escopo e decisões arquiteturais

### 2.1 Responsabilidades

- **CP18:** identidade, sessão, autorização e ownership. O CP20 consome o contexto autenticado e aplica isolamento aos documentos e resultados; não reimplementa autenticação ou gestão de usuários.
- **CP20:** contrato canônico, ingestão segura, identificação de formatos, adaptadores de extração, reconstrução estrutural, classificação documental, extração de entidades, proveniência, OCR condicional, persistência, revisão e reprocessamento.
- **CP21 — Tailoring determinístico:** consome os documentos canônicos para analisar currículo e vaga, mapear evidências e construir estratégia. Não cria parser documental, OCR ou NLP Core paralelo.
- **CP22 — Score Fit: captura comportamental:** registra exposições e interações da experiência de vagas. Não depende do processamento documental para sua instrumentação.
- **CP23 — Score Fit: personalização e avaliação:** consome os eventos confiáveis do CP22 e mantém o score-base da Parte 1 separado da personalização.
- **F3:** upload, conversão, exportação e catálogo de erros já existentes. O CP20 deve auditar e estender o caminho adequado, não criar um segundo fluxo de upload nem substituir a exportação.
- **F5/B11:** NLP Core, normalização linguística, tokenização, recursos de idioma, offsets e ranking existentes. O CP20 reutiliza os recursos genéricos aplicáveis, sem criar um segundo NLP Core.
- **CP11:** instalação, bootstrap técnico, diretórios da aplicação e operação do executável. O CP20 respeita os diretórios e mecanismos existentes, sem criar bootstrap ou instalação próprios.
- **CP19:** onboarding apresenta e consome os serviços de currículo/importação que forem disponibilizados; não implementa um pipeline documental paralelo.

### 2.2 Limites de domínio

- O pipeline documental deve ser genérico e não conhecer as regras de negócio de currículo, vaga, candidatura ou Tailoring.
- A classificação documental pode indicar categorias suportadas, como currículo, descrição de vaga, carta, certificado ou outro tipo reconhecido, mas não substitui a análise semântica de cada domínio.
- A extração de entidades identifica informações presentes no documento e preserva sua fonte; não decide se uma competência atende a um requisito de vaga.
- A camada documental não calcula Score Fit, não seleciona evidências para Tailoring e não gera currículo adaptado.
- O contrato canônico deve servir a mais de um consumidor sem exigir que cada consumidor conheça o formato original ou a biblioteca usada para processá-lo.

### 2.3 Formatos e estratégia de homologação

- Auditar primeiro os formatos efetivamente aceitos pelo upload/conversão da F3.
- Usar PDF, TXT e Markdown como formatos iniciais a avaliar, conforme a matriz prevista no backlog; cada formato somente será considerado homologado quando houver adaptador, fixtures e critérios de qualidade aprovados.
- Fazer spike antes de assumir suporte a DOCX ou qualquer formato adicional.
- Não anunciar suporte a um formato apenas porque uma biblioteca consegue abrir alguns arquivos de exemplo.
- Documentar limitações conhecidas por formato, como perda de layout, ordem de leitura, tabelas, colunas, imagens e metadados.
- Manter a matriz de formatos versionada para permitir evolução sem alterar silenciosamente o comportamento de documentos já processados.

### 2.4 Reutilização e limites tecnológicos

- Reutilizar tipos, tokenizer, normalizador, recursos linguísticos e convenções de offsets da F5/B11 quando compatíveis.
- Auditar upload, conversão, storage temporário, workers e catálogo de erros antes de adicionar infraestrutura.
- Escolher parsers e OCR após comparar qualidade, manutenção, licença, idiomas, consumo de recursos e operação offline.
- Não introduzir banco, fila ou serviço externo exclusivamente para documentos sem evidência de necessidade e decisão arquitetural.
- Não exigir download silencioso de binários, modelos ou pacotes de idioma na inicialização.
- Não enviar documentos, texto extraído ou metadados privados a APIs externas. Qualquer exceção exige decisão explícita de produto, avaliação de privacidade e consentimento apropriado.

### 2.5 Separação entre fonte, processamento e correção humana

O sistema deve manter separadas pelo menos estas camadas lógicas:

1. **Original:** arquivo recebido, imutável após aceitação, com hash/fingerprint e metadados da origem.
2. **Extração:** resultado produzido pelo adaptador de formato, incluindo texto e estrutura reconhecida.
3. **Normalização:** representação normalizada e enriquecida, sem sobrescrever os valores originais.
4. **Interpretação:** classificação documental e entidades reconhecidas, cada qual com confiança e proveniência quando aplicável.
5. **Revisão humana:** correções ou confirmações do usuário registradas como alterações próprias, sem apagar o resultado bruto do extrator.

Um novo processamento não pode substituir silenciosamente a fonte original, um resultado anterior ou uma correção humana.

## 3. Modelo canônico e contratos

A estrutura final deve seguir os padrões reais do projeto e ser formalizada em tipos/contratos versionados. Os nomes abaixo são conceituais e podem ser ajustados após a auditoria, preservando semântica e rastreabilidade.

### 3.1 Entidades mínimas

- `Document`: identidade do documento, schema, ownership, metadados de origem, formato reconhecido e estado geral do processamento.
- `Page`: página ou unidade equivalente, com índice, dimensões e unidade quando disponíveis.
- `Section`: seção ou agrupamento estrutural, com hierarquia, ordem e localização de origem quando reconhecidas.
- `Block`: unidade estrutural, como título, parágrafo, lista, item de lista, tabela, linha/célula ou outro bloco homologado.
- `TextSpan`: trecho textual com conteúdo original, conteúdo normalizado opcional e localização precisa no documento.
- `Entity`: informação tipada extraída, com valor original, valor normalizado opcional, confiança quando calculada e referências à fonte.
- `SourceReference`: localização que permite rastrear um trecho ou entidade até página, seção, bloco, offsets e coordenadas, quando disponíveis.
- `ProcessingRun` ou equivalente: execução de processamento com versão de schema, adaptador, configuração, timestamps, estado, avisos e erros.

### 3.2 Regras obrigatórias do contrato

- [ ] Versionar explicitamente o schema e documentar compatibilidade entre versões.
- [ ] Preservar texto original e texto normalizado em campos distintos; normalização não pode sobrescrever a fonte.
- [ ] Preservar a ordem de leitura e a localização por página, seção, bloco e offsets quando fornecidos pelo adaptador.
- [ ] Representar coordenadas, idioma, confiança e demais campos como ausentes quando não houver evidência; não inventar valores para preencher o schema.
- [ ] Registrar nome original, extensão, MIME declarado, MIME detectado, tamanho, hash/fingerprint e data de ingestão.
- [ ] Registrar versão do schema, adaptador, parser, OCR e recursos linguísticos usados em cada execução, quando aplicável.
- [ ] Definir identificadores estáveis dentro de cada documento/versão de processamento para páginas, blocos, trechos e entidades.
- [ ] Preservar a relação entre entidades extraídas e os trechos que as sustentam.
- [ ] Permitir extensões versionadas sem vincular o contrato a um parser específico ou a um domínio de negócio.
- [ ] Reutilizar tipos e convenções genéricas da F5/B11 quando compatíveis.
- [ ] Fornecer exemplos versionados de documentos antes e depois do processamento, inclusive com campos ausentes e avisos.

### 3.3 Localização e proveniência

- [ ] Definir convenção única para offsets de texto, inclusive unidade e intervalo inclusivo/exclusivo.
- [ ] Registrar offsets em relação ao texto de origem claramente identificado, evitando offsets ambíguos após normalização.
- [ ] Preservar página e bloco de origem sempre que o formato permitir.
- [ ] Preservar coordenadas de página, unidade e sistema de referência quando fornecidos pelo adaptador; não exigir coordenadas para TXT/Markdown.
- [ ] Permitir que uma entidade aponte para mais de um trecho-fonte.
- [ ] Distinguir localização exata, localização aproximada e ausência de localização.
- [ ] Não considerar uma entidade rastreável quando sua referência aponta apenas para o documento inteiro, se o trecho específico estiver disponível.

### 3.4 Estados de processamento

Definir uma máquina de estados coerente com os serviços existentes. Os nomes abaixo são conceituais e podem ser ajustados:

- `received`: arquivo recebido, ainda não validado;
- `rejected`: arquivo não aceito pela política de formato/segurança/limites;
- `accepted`: arquivo validado e pronto para processamento;
- `processing`: extração ou análise em andamento;
- `needs_review`: resultado disponível com ambiguidades ou confiança insuficiente;
- `completed`: processamento concluído dentro dos critérios definidos;
- `failed`: falha controlada, com código e possibilidade de retry quando aplicável.

- [ ] Definir transições válidas e impedir que a UI marque sucesso apenas porque o upload terminou.
- [ ] Separar status da ingestão de status da extração, OCR, classificação e revisão quando necessário.
- [ ] Expor progresso, avisos e erros sem revelar conteúdo privado ou detalhes internos sensíveis.
- [ ] Garantir que retry não duplique documentos nem corrompa execuções anteriores.

## 4. Pipeline de processamento

### 4.1 DI-01 — Contrato canônico de documento

- [ ] Implementar os tipos `Document`, `Page`, `Section`, `Block`, `TextSpan`, `Entity`, `SourceReference` e `ProcessingRun` ou equivalentes compatíveis com os padrões do projeto.
- [ ] Definir IDs, ownership, versão de schema, campos obrigatórios/opcionais e semântica de ausência.
- [ ] Separar metadados do arquivo original, conteúdo extraído, texto normalizado, interpretação e correções humanas.
- [ ] Definir proveniência compatível com texto simples e documentos paginados; offsets e coordenadas devem ser opcionais quando não existirem.
- [ ] Registrar versão de parser/extrator, recursos e configurações que influenciem o resultado.
- [ ] Reutilizar tipos genéricos da F5/B11 e documentar limites entre NLP Core e Inteligência de documentos.
- [ ] Criar exemplos versionados de documentos válidos, parcialmente processados, com baixa confiança e com falhas.
- [ ] Registrar ADR do contrato, versionamento e compatibilidade.

**Validação DI-01:** consumidores conseguem utilizar o contrato sem depender do formato de origem; valores originais, normalizados e referências de origem permanecem distinguíveis.

### 4.2 DI-02 — Ingestão segura e identificação de formato

- [ ] Inventariar o fluxo de upload/conversão da F3, stores, handlers, diretórios temporários e limites já implementados.
- [ ] Estender o serviço existente ou criar uma abstração interna comum, evitando um segundo caminho de upload concorrente.
- [ ] Detectar formato pelo conteúdo real do arquivo (assinatura/magic bytes e inspeção compatível), não apenas pela extensão ou MIME informado pelo navegador.
- [ ] Comparar extensão, MIME declarado e MIME detectado; documentar a política para divergências.
- [ ] Aplicar allowlist baseada na matriz efetivamente homologada; não ampliar formatos sem spike, fixtures e aceite.
- [ ] Validar arquivo vazio, assinatura inválida, estrutura básica corrompida, incompatibilidade e proteção por senha quando detectável.
- [ ] Definir limites de bytes, páginas, memória, tempo de processamento e tamanho expandido/descompactado a partir de auditoria e medição.
- [ ] Rejeitar entradas acima dos limites antes de consumir recursos desproporcionais.
- [ ] Garantir que a inspeção de metadados não execute macros, scripts, conteúdo ativo ou recursos externos embutidos.
- [ ] Mapear falhas para códigos estáveis e mensagens acionáveis, reutilizando B6/B8.
- [ ] Não registrar conteúdo integral, trechos privados, tokens ou caminhos internos sensíveis.

#### Staging temporário

- [ ] Colocar arquivos recebidos em área temporária privada, fora de diretórios públicos e sem nomes controlados pelo usuário.
- [ ] Usar IDs/nomes internos aleatórios e permissões restritivas.
- [ ] Impedir path traversal, colisões, substituição indevida e leitura cruzada entre usuários.
- [ ] Derivar ownership do contexto autenticado fornecido pelo CP18; nunca confiar em `user_id` do payload.
- [ ] Impedir acesso direto via rota estática ou exposição de caminhos do filesystem.
- [ ] Limpar arquivos temporários em sucesso, falha, cancelamento e timeout.
- [ ] Definir TTL e rotina para artefatos órfãos, sem remover arquivos em uso por processamento ativo.
- [ ] Documentar o comportamento após encerramento/reinício e a recuperação/limpeza de temporários.
- [ ] Tornar falhas de limpeza observáveis sem expor conteúdo privado nem interromper a aplicação.

**Validação DI-02:** arquivos inválidos ou excessivos falham de forma controlada; staging é privado, temporário, isolado por usuário e limpo conforme política documentada.

### 4.3 DI-03 — Adaptadores de extração por formato

- [ ] Definir a matriz inicial de formatos a partir do que a F3 realmente recebe e converte; avaliar PDF, TXT e Markdown como candidatos iniciais conforme o backlog.
- [ ] Fazer spike antes de homologar DOCX ou qualquer formato adicional.
- [ ] Implementar adaptadores isolados por formato, todos produzindo o contrato canônico.
- [ ] Separar o contrato de saída da implementação/biblioteca específica do parser.
- [ ] Preservar ordem de leitura, títulos, parágrafos, listas, tabelas e separação por página quando o formato/adaptador permitir.
- [ ] Preservar texto sem estrutura quando não houver evidência suficiente para reconstruí-la.
- [ ] Detectar documentos sem camada textual e encaminhá-los à avaliação de OCR, em vez de retornar texto vazio como sucesso.
- [ ] Documentar limitações conhecidas por formato e tipo de documento.
- [ ] Registrar versão do adaptador e parâmetros relevantes em cada execução.
- [ ] Garantir que falha de um adaptador não comprometa o arquivo original nem outros documentos em processamento.

**Validação DI-03:** fixtures dos formatos homologados chegam ao mesmo contrato canônico, com ordem e estrutura preservadas dentro das capacidades conhecidas de cada formato.

### 4.4 DI-04 — Reconstrução estrutural e classificação documental

- [ ] Identificar blocos e elementos estruturais suportados: cabeçalhos, títulos, seções, parágrafos, listas, tabelas, colunas e ordem de leitura.
- [ ] Preservar a hierarquia de seções e a sequência do conteúdo, incluindo documentos com múltiplas colunas quando a qualidade do adaptador permitir.
- [ ] Não inventar títulos, separações ou relações estruturais quando a evidência for insuficiente.
- [ ] Classificar documentos somente entre categorias previstas e com evidência suficiente, incluindo inicialmente currículo, descrição de vaga, carta, certificado ou `other/unknown`.
- [ ] Permitir classificação incerta ou múltiplas possibilidades quando não houver confiança suficiente.
- [ ] Separar classificação documental de extração factual: classificar como currículo não comprova que cargos, períodos ou competências tenham sido extraídos corretamente.
- [ ] Registrar método/versão e confiança da classificação quando calculada.
- [ ] Encaminhar ambiguidades relevantes para revisão, sem bloquear o uso de conteúdo que possa ser apresentado com ressalvas.
- [ ] Preservar múltiplas seções e anexos sem perder a referência ao documento de origem.

**Validação DI-04:** estrutura provável é representada com ordem e hierarquia; classificações incertas são sinalizadas e nenhum conteúdo estrutural é inventado silenciosamente.

### 4.5 DI-05 — Extração de entidades e proveniência

- [ ] Definir entidades genéricas úteis aos consumidores: organizações, cargos, períodos, formação, competências, certificações, idiomas, localização e outras categorias justificadas por casos de uso reais.
- [ ] Para cada entidade, preservar trecho-fonte, documento, página/seção, bloco, offsets ou coordenadas disponíveis e confiança quando calculada.
- [ ] Diferenciar conteúdo explicitamente presente, normalização determinística e inferência/interpretação.
- [ ] Preservar valor original ao normalizar nomes, datas, unidades ou termos.
- [ ] Não inferir datas, senioridade, certificações, métricas ou competências sem suporte documental.
- [ ] Não converter semelhança lexical em comprovação de experiência ou domínio.
- [ ] Permitir múltiplas ocorrências e referências para uma entidade quando o conteúdo aparecer em mais de um trecho.
- [ ] Permitir correção/confirmação humana de entidades ambíguas ou com baixa confiança.
- [ ] Registrar correções como camada separada, preservando resultado bruto e histórico de processamento.
- [ ] Não extrair atributos pessoais sensíveis desnecessários ao caso de uso.
- [ ] Definir como consumidores consultam entidades e referências sem depender da biblioteca do extrator.

**Validação DI-05:** fatos usados pelos consumidores podem ser rastreados à fonte; normalizações e inferências são identificadas; correções humanas não apagam a extração original.

### 4.6 DI-06 — OCR condicional

- [ ] Detectar ausência ou insuficiência de camada textual antes de iniciar OCR.
- [ ] Executar OCR somente nas páginas necessárias, preservando páginas digitais que já tenham texto utilizável.
- [ ] Fazer spike comparativo de engines considerando qualidade, PT-BR/EN-US/ES-419, manutenção, licença, consumo de CPU/memória, tamanho de recursos e operação local.
- [ ] Registrar a decisão de engine e versões de modelos/idiomas utilizados.
- [ ] Preferir execução local; não enviar páginas a serviços externos por padrão.
- [ ] Não exigir download silencioso de executáveis ou modelos na inicialização. Definir instalação/obtenção explícita dos recursos quando aplicável.
- [ ] Registrar confiança por página/bloco e sinalizar baixa qualidade ou páginas sem resultado.
- [ ] Identificar explicitamente o texto OCR como extraído por OCR, distinguindo-o de camada textual original.
- [ ] Preservar referências de página e coordenadas quando fornecidas pelo engine.
- [ ] Permitir revisão do resultado pelo usuário e preservar a versão anterior após correções ou novo OCR.
- [ ] Tratar engine/idioma indisponível como estado controlado, sem derrubar o processamento dos demais documentos.

**Validação DI-06:** PDFs digitais não passam por OCR desnecessário; documentos escaneados podem produzir texto com origem e confiança identificadas; ausência de engine/recurso não provoca falha silenciosa.

### 4.7 DI-07 — Persistência, reprocessamento e integração

- [ ] Definir ciclo de vida para original, resultados de extração, normalização, classificação, entidades e correções do usuário.
- [ ] Associar documentos e resultados ao usuário autenticado pelo CP18; validar ownership em queries, APIs, downloads, workers e reprocessamentos.
- [ ] Definir política de retenção e exclusão do original e dos resultados, respeitando as regras de dados pessoais do projeto.
- [ ] Implementar idempotência por fingerprint, versão do schema, versão do adaptador/extrator, versão de OCR e opções de processamento relevantes.
- [ ] Diferenciar reprocessamento idêntico, reprocessamento com nova versão e nova ingestão de arquivo diferente.
- [ ] Preservar execuções anteriores ou histórico suficiente para comparação e auditoria; não sobrescrever correções humanas silenciosamente.
- [ ] Permitir reprocessamento após atualização de parser/recursos com indicação clara de diferenças e versões.
- [ ] Expor status, progresso, avisos e erros amigáveis, reutilizando B6/B8.
- [ ] Definir retry seguro, cancelamento e tratamento de execução interrompida.
- [ ] Integrar resultados ao serviço de currículo e aos consumidores posteriores por contratos, sem fazer adaptadores conhecerem as regras desses domínios.
- [ ] Disponibilizar ao CP21 acesso às entidades e evidências documentais com origem e versão.
- [ ] Manter o original separado do currículo-base ativo; a substituição do currículo continua sujeita aos fluxos de confirmação e versionamento do CP18.
- [ ] Evitar duplicar o estado de onboarding do CP19 ou as configurações que pertencem a Settings.

**Validação DI-07:** reprocessamento é rastreável e idempotente; versões anteriores e correções não são perdidas; documentos e resultados permanecem isolados por usuário.

### 4.8 DI-08 — Testes, corpus de referência e gate

- [ ] Criar corpus versionado de fixtures representativas: PDF digital, PDF escaneado, TXT, Markdown, arquivo corrompido e documentos com tabelas/colunas.
- [ ] Incluir variações de qualidade, fontes, páginas, layout e conteúdo para evitar validar somente exemplos triviais.
- [ ] Testar PT-BR, EN-US e ES-419 nos idiomas homologados e registrar limitações por engine/adaptador.
- [ ] Testar offsets, referências de página, ordem de leitura, hierarquia de seções e vínculo entre entidade e trecho-fonte.
- [ ] Testar classificação documental correta, incerta e `unknown`.
- [ ] Testar documentos sem texto, OCR parcial, baixa confiança e ausência de pacote de idioma.
- [ ] Testar arquivos grandes, limites, timeouts, falhas de parser/OCR, cancelamento e limpeza de temporários.
- [ ] Testar conteúdo malformado, formatos não permitidos, extensões/MIME divergentes e entradas potencialmente maliciosas.
- [ ] Testar idempotência, reprocessamento por versão, preservação de correções e exclusão conforme política.
- [ ] Testar isolamento entre usuários em leitura, alteração, reprocessamento, exportação e exclusão.
- [ ] Criar golden tests versionados para texto, estrutura, classificação, entidades, proveniência e OCR.
- [ ] Definir métricas de qualidade e metas de performance após medir o corpus de referência; não fixar limiares arbitrários sem baseline.
- [ ] Registrar falsos positivos, falsos negativos e limitações conhecidas por versão.
- [ ] Executar `make test`, `make vet`, testes de migrations e `make test-ui` quando houver alteração visual ou integração afetada.

**Gate DI-08:** formatos homologados e limites documentados; contrato versionado; proveniência preservada; OCR condicional; reprocessamento seguro; isolamento validado; golden tests e metas de qualidade aprovados.

## 5. Persistência e contratos de serviço

### 5.1 Responsabilidade da persistência

A auditoria deve identificar os mecanismos existentes de storage e migrations. A modelagem final deve seguir os padrões do projeto, sem impor nomes de tabelas antes da inspeção do schema real.

A solução deve representar, no mínimo:

- documento original e seus metadados;
- proprietário derivado da sessão;
- fingerprint/hash para deduplicação e idempotência;
- versões do schema e dos adaptadores;
- execuções de processamento e respectivos estados;
- resultados extraídos/normalizados/classificados;
- entidades e referências de proveniência;
- correções e confirmações humanas;
- timestamps, avisos, erros e histórico necessário para auditoria.

### 5.2 Contratos de serviço

Ajustar aos padrões do projeto; nomes abaixo são conceituais:

- `ingest`: recebe arquivo validado pelo serviço de upload, identifica formato, aplica política de segurança e cria referência privada;
- `process`: executa adaptadores e etapas configuradas para o formato e versão;
- `get status`: retorna estado, progresso, avisos e erros seguros do processamento;
- `get canonical document`: retorna representação canônica autorizada ao proprietário;
- `review/update corrections`: registra correções humanas sem alterar silenciosamente a extração original;
- `reprocess`: inicia execução idempotente com versão/opções identificadas;
- `delete`: aplica política de exclusão ao original, resultados e artefatos derivados.

O desenho real pode usar serviços internos, interfaces Go ou endpoints já existentes. Não criar endpoint público separado para cada adaptador sem necessidade demonstrada.

### 5.3 Regras de backend e workers

- Derivar usuário da sessão CP18 em toda operação iniciada pelo usuário.
- Transportar contexto de ownership explicitamente para workers/jobs; não executar processamento de documentos privados sob contexto global implícito.
- Validar ownership novamente ao consultar status, resultado, cancelar, reprocessar, baixar ou excluir.
- Não permitir que identificadores previsíveis ou fingerprint sejam usados como autorização.
- Isolar falhas de um documento para que não interrompam lotes ou outros usuários.
- Manter transações curtas; processamento pesado não deve ocorrer dentro de uma transação de banco aberta por toda a execução.
- Garantir consistência entre status, resultado persistido e evento/aviso de conclusão.
- Não retornar caminho local, stack trace, conteúdo integral ou detalhes internos de parser nas respostas da API.

## 6. Critérios de verificação e aceite

- [ ] O contrato canônico está versionado, documentado e neutro de formato e domínio.
- [ ] `Document`, `Page`, `Section`, `Block`, `TextSpan`, `Entity` e `SourceReference` possuem semântica e exemplos versionados.
- [ ] Texto original, normalizado, interpretação e correção humana são representados separadamente.
- [ ] Página, seção, bloco, ordem, offsets e coordenadas são preservados quando disponíveis; ausência não é preenchida por suposição.
- [ ] Cada entidade relevante possui referência rastreável à fonte ou é marcada como sem localização suficiente.
- [ ] Formatos são reconhecidos pelo conteúdo; extensão ou MIME declarados não são aceitos isoladamente.
- [ ] A matriz de formatos homologados é baseada na auditoria da F3; DOCX e outros formatos só entram após spike e aceite.
- [ ] Arquivos inválidos, incompatíveis, protegidos ou excessivos falham com mensagens acionáveis e sem travar workers.
- [ ] Limites de tamanho, páginas, tempo, memória e expansão são documentados e aplicados antes de processamento custoso.
- [ ] Adaptadores homologados preservam texto e estrutura dentro de suas limitações documentadas.
- [ ] Reconstrução estrutural mantém ordem/hierarquia e não inventa seções ou conteúdo.
- [ ] Classificação documental distingue resultado certo, incerto e desconhecido.
- [ ] Entidades distinguem conteúdo explícito, normalização e inferência; nenhuma competência, data, senioridade ou métrica é fabricada.
- [ ] OCR só é executado quando a camada textual está ausente/insuficiente e somente nas páginas necessárias.
- [ ] OCR preserva origem, página, confiança e versão do engine/recursos; PDFs digitais não sofrem OCR redundante.
- [ ] Persistência é idempotente e reprocessamento identifica versões de schema, parser, OCR e opções.
- [ ] Reprocessamento não destrói original, execução anterior ou correção humana silenciosamente.
- [ ] Documentos, resultados, revisão, reprocessamento e exclusão são isolados por usuário em backend, APIs e workers.
- [ ] Temporários são privados, têm TTL/limpeza definidos e são removidos nos caminhos de sucesso, erro, cancelamento e timeout.
- [ ] Nenhum documento, texto ou metadado privado é enviado a serviço externo sem decisão e consentimento explícitos.
- [ ] Logs e respostas não expõem conteúdo integral, caminhos internos, tokens ou detalhes sensíveis.
- [ ] O upload/conversão da F3 continua funcionando sem regressão.
- [ ] NLP Core da F5/B11 é reutilizado; não há tokenizer, normalizador ou catálogo NLP paralelo.
- [ ] Corpus, golden tests, idiomas, qualidade e performance têm critérios reproduzíveis e limitações documentadas.
- [ ] `make test`, `make vet`, migrations e testes UI/E2E afetados passam.

**Gate final do CP20:** a aplicação recebe documentos de forma segura, processa formatos homologados por adaptadores isolados, produz representação canônica estruturada e versionada, preserva proveniência, classifica e extrai entidades com limites explícitos, utiliza OCR condicional quando necessário, persiste e reprocessa sem perda de histórico/correções e mantém isolamento por usuário. Os consumidores posteriores utilizam os contratos publicados sem implementar pipelines documentais próprios.

## 7. Entregáveis

- ADR do contrato canônico, versionamento e separação entre arquivo original, extração, normalização, interpretação e correção humana.
- Matriz de formatos homologados, MIME/assinaturas, limitações e política para formatos ainda não suportados.
- Contratos/tipos versionados de `Document`, `Page`, `Section`, `Block`, `TextSpan`, `Entity`, `SourceReference` e execução de processamento.
- Serviço de ingestão segura integrado ao upload existente, com ownership, detecção de formato, limites e staging temporário.
- Adaptadores homologados de extração para os formatos aprovados.
- Reconstrução estrutural e classificação documental com estados de confiança/incerteza.
- Extração de entidades e proveniência, incluindo referências ao trecho-fonte.
- Decisão documentada de OCR e, se aprovado, engine/recursos locais com execução condicional.
- Persistência versionada, idempotência, revisão humana, limpeza e reprocessamento.
- APIs/serviços de consulta e operação consumíveis pelo CP21 e pelos fluxos de currículo.
- Corpus de referência, fixtures, golden tests, testes de segurança/ownership e relatório de qualidade/performance.
- Documentação de operação, limites, erros conhecidos e evolução dos contratos.

## 8. Registrar

- ADR do contrato canônico e da estratégia de versionamento/compatibilidade.
- ADR de ingestão segura, identificação de formatos, limites operacionais e staging temporário.
- Matriz de formatos, extensões, MIME declarado, MIME detectado e assinatura real, com política para divergências.
- Decisão de parsers/adaptadores por formato, incluindo licenças, manutenção e limitações conhecidas.
- ADR/registro de decisão de OCR com engines avaliadas, idiomas, qualidade, licença, recursos e operação offline.
- Matriz de entidades, distinção entre declaração/normalização/inferência e regras de proveniência.
- Plano de persistência, retenção, exclusão, idempotência e reprocessamento.
- Política de revisão humana e preservação de correções.
- Corpus de referência, métricas, golden tests, resultados e limitações conhecidas.
- Matriz de ownership e testes de isolamento derivados do CP18.
- Registro de integração com upload/conversão da F3 e reutilização do NLP Core da F5/B11.

## 9. Dependências, riscos e mitigação

| Risco/dependência | Impacto | Mitigação |
|---|---|---|
| Upload/conversão da F3 não cobre todos os formatos esperados | A matriz de suporte seria superestimada | Auditar o comportamento real; homologar somente formatos com fixtures e aceite. |
| Parser perde ordem, tabelas ou colunas | Extração incompleta ou incorreta | Corpus representativo, adaptadores isolados, limitações documentadas e revisão quando necessário. |
| OCR consome muitos recursos ou tem baixa qualidade | Lentidão e resultados pouco confiáveis | Executar condicionalmente, avaliar por idioma/página e definir limites após benchmark. |
| Engine OCR exige recursos não instalados | Falha de operação offline ou downloads inesperados | Registrar dependências explícitas e não baixar recursos silenciosamente na inicialização. |
| Entidades são interpretadas como fatos | Tailoring pode introduzir afirmações sem suporte | Proveniência obrigatória, distinção declaração/normalização/inferência e revisão humana. |
| Reprocessamento sobrescreve correções | Perda de trabalho do usuário | Versionar execuções e manter correções como camada separada. |
| Arquivos malformados ou excessivos | Consumo desproporcional de CPU/memória ou indisponibilidade | Validação anterior ao processamento, limites, timeout, isolamento e testes negativos. |
| Vazamento entre usuários | Exposição de currículos/documentos privados | Ownership derivado da sessão e testes em APIs, workers, downloads e exclusão. |
| Bibliotecas/licenças inadequadas | Risco de manutenção ou distribuição | Comparar licença, manutenção, dependências e distribuição antes da escolha. |
| NLP Core duplicado | Divergência de tokens, offsets e normalização | Auditar e reutilizar F5/B11 por contratos explícitos. |
| Processamento documental acoplado ao Tailoring | Regras específicas contaminam a camada genérica | Manter contratos neutros e testar consumidores independentes. |
| Corpus de teste pouco representativo | Golden tests aprovam casos triviais | Construir corpus com formatos, idiomas, layouts e níveis de qualidade reais. |

## 10. Fora de escopo

- Implementar autenticação, sessão, cadastro de usuários ou gestão geral de ownership (CP18).
- Criar um segundo upload de currículo ou substituir a UX de upload/exportação/cópia já entregue na F3.
- Implementar o onboarding gráfico ou duplicar etapas do CP19.
- Alterar o NLP Core, o algoritmo BM25, o ranking de vagas ou os filtros da F5/B11.
- Fazer scraping ou buscar documentos em serviços externos.
- Enviar documentos ou conteúdo privado a serviços externos como comportamento padrão.
- Implementar Resume Analyzer, Job Analyzer, Evidence Mapping ou Tailoring Strategy (CP21).
- Gerar currículo adaptado, validar afirmações do Tailoring ou substituir revisão humana de conteúdo ambíguo (CP21).
- Implementar captura comportamental, event log de feed ou timeline do Score Fit (CP22).
- Implementar personalização, re-ranking comportamental ou Evaluation Engine do Score Fit (CP23).
- Introduzir OCR obrigatório para todos os PDFs ou exigir recursos não disponíveis localmente.
- Implementar extração de atributos pessoais sensíveis sem necessidade justificada pelo caso de uso.
- Transformar a classificação documental em julgamento de qualidade profissional ou adequação a uma vaga.

## 11. Dependências e integração com os próximos checkpoints

- **CP18 — Usuários, autenticação e perfil profissional:** obrigatório para ownership de documentos, resultados e correções. O CP20 usa a sessão existente e não cria outro modelo de identidade.
- **F3 — Upload, conversão e UX de currículo:** infraestrutura de entrada a ser auditada e reutilizada. O CP20 preserva o comportamento existente e evolui os contratos necessários.
- **F5/B11 — NLP Core:** fornece recursos genéricos de idioma, normalização e tokenização. O CP20 os consome quando aplicáveis e não os duplica.
- **CP19 — Onboarding web:** pode chamar os serviços de ingestão/processamento para currículo-base, mas não deve implementar parser ou persistência paralela.
- **CP21 — Tailoring determinístico:** consome entidades, trechos e documentos canônicos para Resume Analyzer, Job Analyzer e Evidence Mapping. Deve preservar as referências de origem fornecidas pelo CP20.
- **CP22/CP23 — Score Fit, Parte 2:** não são dependências do CP20; a Inteligência de documentos não registra eventos comportamentais nem altera ranking.

## 12. Sequência de implementação e gates internos

O CP20 é um único checkpoint, mas sua execução deve ocorrer em etapas internas para reduzir risco e validar contratos antes de ampliar o processamento. Os gates abaixo não criam novos checkpoints nem alteram a cobertura de DI-01 a DI-08.

### Etapa A — Auditoria, contrato e ingestão (DI-01/DI-02)

- [ ] Auditar F3, storage, workers, limites, erros e F5/B11.
- [ ] Aprovar ADR e exemplos do contrato canônico.
- [ ] Implementar identificação, validação, staging e ownership.
- [ ] Executar testes de segurança, limites e regressão de upload.

**Gate A:** contrato versionado e intake seguro; nenhuma extração semântica é presumida.

### Etapa B — Extração, estrutura e entidades (DI-03/DI-05)

- [ ] Homologar formatos com fixtures reais.
- [ ] Implementar adaptadores e reconstrução estrutural.
- [ ] Implementar classificação documental e extração de entidades com proveniência.
- [ ] Avaliar golden tests e registrar limitações por formato/idioma.

**Gate B:** formatos homologados geram saída canônica rastreável e sem fatos fabricados.

### Etapa C — OCR, persistência e reprocessamento (DI-06/DI-07)

- [ ] Comparar engines OCR e registrar decisão.
- [ ] Implementar OCR somente para páginas sem texto suficiente, se aprovado no spike.
- [ ] Implementar persistência versionada, revisão humana, idempotência e reprocessamento.
- [ ] Validar ownership e recuperação de falhas.

**Gate C:** OCR e persistência operam de forma controlada; versões e correções são preservadas.

### Etapa D — Qualidade e aceite final (DI-08)

- [ ] Executar corpus, golden tests, testes de segurança, idiomas e performance.
- [ ] Resolver regressões e documentar limitações remanescentes.
- [ ] Atualizar ADRs, contratos, documentação e matriz de formatos.
- [ ] Confirmar gates internos A, B e C.

**Gate D:** critérios finais de aceite do CP20 aprovados e contratos liberados para consumo pelo CP21.

## 13. Definições pendentes antes da estimativa final

- [ ] Confirmar após auditoria quais formatos a F3 efetivamente recebe, converte e preserva com qualidade suficiente.
- [ ] Definir limites máximos de bytes, páginas, tempo, memória e tamanho expandido com base em benchmark e operação local.
- [ ] Selecionar bibliotecas/adaptadores por formato após comparar qualidade, licença, manutenção e distribuição.
- [ ] Definir corpus de referência e critérios de qualidade por formato e idioma.
- [ ] Selecionar ou rejeitar OCR após spike com documentos escaneados representativos.
- [ ] Definir política de retenção, exclusão e armazenamento dos originais após auditoria de storage e privacidade.
- [ ] Confirmar convenção de offsets, coordenadas, unidades e estabilidade dos identificadores.
- [ ] Definir quais entidades são realmente necessárias aos consumidores iniciais, evitando extração sem finalidade.
- [ ] Definir critérios de confiança e de encaminhamento para revisão humana com dados de teste, sem fixar limiares arbitrários.
- [ ] Confirmar estratégia de execução assíncrona usando workers existentes, sem introduzir infraestrutura desnecessária.

**Regra final de escopo:** DI-01 a DI-08 pertencem integralmente ao CP20. A divisão em etapas internas serve apenas para organizar execução e gates de qualidade; não transfere extração, OCR, persistência ou testes para outro checkpoint. Qualquer ampliação de formatos, novas classes de entidades ou dependência externa deve ser justificada por evidência, registrada em ADR e incorporada ao escopo antes da implementação.
