# Checkpoint 19 — Onboarding web de primeira execução

> **Estimativa inicial:** 5–8 dias úteis, sujeita à auditoria dos fluxos existentes e à disponibilidade do adaptador de importação do LinkedIn definido no CP18.  
> **Status:** proposto.  
> **Pré-requisitos:** CP18 concluído; F4 (UX de IA: BYOK, presets, free tier e perfis) disponível; F1/CP16, fontes e configurações de discovery funcionais; componentes e validações existentes acessíveis por serviços/API.  
> **Responsabilidade:** experiência gráfica guiada dentro da webapp. Autenticação, ownership, perfil, currículo-base e serviços OAuth pertencem ao CP18; bootstrap técnico pertence ao CP11.

## 1. Objetivo

Entregar uma experiência de primeira execução que conduza cada usuário autenticado desde a entrada na webapp até a home/feed de vagas, apresentando e validando as configurações essenciais do openJob.

O onboarding deve ser progressivo, persistente e não bloqueante. O usuário precisa entender o que está configurado, o que ainda falta e quais funcionalidades dependem de cada configuração. Etapas opcionais podem ser ignoradas e concluídas posteriormente em Settings.

O fluxo deve reutilizar os serviços, contratos, formulários, componentes e validações existentes. Não deve criar uma segunda implementação de autenticação, importação de currículo, configuração de fontes ou configuração de IA.

## 2. Princípios e limites arquiteturais

### 2.1 Responsabilidades

- **CP18:** login, sessão, identidade, ownership, perfil profissional, currículo-base, conexões externas, callbacks OAuth, tokens e adaptadores de importação.
- **CP19:** sequência do onboarding, apresentação visual, navegação entre etapas, progresso, retomada, validação de conclusão e resumo final.
- **CP11:** instalação, bootstrap técnico do executável, host/porta, abertura do navegador e configuração técnica da instância.
- **F4:** BYOK, presets de provedores, free tier, múltiplos perfis de IA e teste de provedor. O onboarding apenas reutiliza esses recursos.
- **F2/CP17:** configuração e sincronização de e-mail. O onboarding não cria outra implementação de conexão de mailbox.

### 2.2 Regras de integração

- O onboarding não implementa autenticação, cadastro administrativo ou gestão de sessão; consome os estados e APIs do CP18.
- Não armazenar senhas, tokens OAuth, API keys ou conteúdo de currículo em `localStorage`.
- O progresso do onboarding é persistido no backend e vinculado ao usuário autenticado; o frontend não é fonte da verdade.
- Configurações de fontes, discovery, filtros e IA continuam tendo seus próprios serviços e persistência. O estado do onboarding não deve duplicar seus valores.
- Não implementar OAuth para provedores de IA.
- LinkedIn será apresentado como opção de conexão/importação de dados profissionais dentro da etapa de currículo. Não é um método de login do onboarding. O fluxo consome o adaptador e o resultado do gate técnico do CP18; não implementa OAuth no CP19.
- Login Google e autorização Gmail são permissões diferentes. O onboarding registra o provedor de identidade e os escopos efetivamente concedidos, mas não presume que login Google autorize a leitura de e-mails.
- Não enviar telemetria para serviços externos. O estado de progresso é funcional e local à instância.

## 3. Política de primeira execução e conclusão

### 3.1 Quem deve iniciar o onboarding

- Uma conta recém-criada ou um usuário que ainda não concluiu a versão atual do onboarding deve ser direcionado ao fluxo após autenticação.
- Usuário que já concluiu a versão atual deve ir diretamente para a home/feed.
- Usuários existentes durante a migração para CP19 não devem ser forçados a repetir todo o wizard. A regra de migração deve preservar o estado existente e oferecer acesso manual a “Revisar configuração” em Settings.
- O administrador inicial também percorre o onboarding por usuário, depois de concluir o bootstrap/autenticação do CP18.
- Um usuário convidado posteriormente recebe seu próprio estado e percurso, sem herdar o progresso ou os dados de outro usuário.

### 3.2 Obrigatório versus opcional

- **Obrigatório:** sessão autenticada válida e confirmação de que o usuário está configurando o próprio perfil.
- **Recomendado:** revisar perfil profissional e configurar/confirmar currículo-base.
- **Opcional:** conectar LinkedIn, configurar fontes adicionais, ajustar filtros/preferências e configurar IA.
- O usuário poderá chegar à página de vagas sem IA configurada.
- O usuário poderá chegar à página de vagas sem currículo-base, caso essa política seja confirmada na implementação. Nesse estado, a interface deve explicar que score/filtragem por currículo e tailoring dependem de um currículo-base.
- Falha de uma integração opcional nunca deve bloquear o uso básico do aplicativo.

**Decisão de produto a registrar antes da implementação:** confirmar que currículo-base vazio é permitido. A recomendação deste checkpoint é permitir a conclusão sem currículo, apresentando claramente as funcionalidades indisponíveis até o cadastro do documento.

### 3.3 Estados do fluxo

O fluxo deve distinguir pelo menos:

- `not_started`: onboarding ainda não iniciado;
- `in_progress`: etapas em andamento;
- `completed`: versão atual concluída;
- `dismissed`: usuário escolheu sair/dispensar sem concluir;
- `reopen_requested` ou estado equivalente: usuário iniciou revisão manual em Settings.

Etapas devem ter estados próprios, como `pending`, `completed`, `skipped` e `needs_attention`. A conclusão do onboarding não deve significar que toda integração opcional foi configurada.

## 4. Fluxo de experiência

### Etapa 1 — Boas-vindas e conta

- [ ] Apresentar o que é o openJob, o funcionamento local-first e uma explicação clara sobre privacidade e dados armazenados na instância.
- [ ] Confirmar o usuário autenticado pelo CP18; não criar uma segunda tela de registro/login dentro do wizard.
- [ ] Oferecer saída para login quando a sessão estiver ausente ou expirada.
- [ ] Registrar no estado da conta/provedor qual método de identidade foi utilizado (por exemplo, login local ou Google OIDC), consumindo a informação fornecida pelo CP18.
- [ ] Explicar que entrar com Google não concede automaticamente acesso à caixa de entrada Gmail.

**Validação:** sessão e identidade são obtidas do backend; o onboarding não confia em identificadores enviados pelo frontend.

### Etapa 2 — Perfil profissional

- [ ] Carregar o perfil atual do CP18 e preencher os campos já existentes.
- [ ] Permitir revisar e editar nome de exibição, headline, localização profissional opcional, experiência e skills, conforme o contrato de perfil do CP18.
- [ ] Mostrar quais campos são opcionais e não inventar valores ausentes.
- [ ] Salvar por meio da API/serviço do perfil do CP18.
- [ ] Confirmar persistência real antes de marcar a etapa como concluída.
- [ ] Permitir continuar sem preencher campos opcionais.

**Validação:** ao recarregar o wizard, os valores persistidos são recuperados do backend.

### Etapa 3 — Currículo-base e importação do LinkedIn

- [ ] Apresentar as opções de entrada que o CP18 efetivamente suporta: upload nos formatos já homologados, colagem de texto/Markdown e conexão/importação do LinkedIn quando viável.
- [ ] Não ampliar a matriz de formatos nem implementar parsers genéricos neste checkpoint; isso pertence ao futuro checkpoint de Inteligência de documentos.
- [ ] Para upload/colagem, reutilizar a importação do CP18, mostrar prévia e exigir confirmação antes de criar/substituir o currículo-base.
- [ ] Exibir o estado do currículo-base atual e informar quando não houver currículo cadastrado.
- [ ] Para LinkedIn, apresentar a opção somente se o CP18 tiver concluído o spike e o adaptador estiver habilitado para os escopos aprovados.
- [ ] Iniciar a conexão pelo endpoint/serviço do CP18; o frontend não deve implementar callback, trocar código OAuth por token ou persistir tokens.
- [ ] Mostrar ao usuário quais campos o provedor autorizou retornar antes de importar.
- [ ] Exigir revisão e confirmação explícita para gravar informações no perfil/currículo-base.
- [ ] Se o resultado do CP18 for `no-go`, ocultar ou explicar a indisponibilidade da opção LinkedIn e manter upload/colagem como alternativas funcionais.
- [ ] Se a conexão falhar, expirar ou retornar campos insuficientes, preservar o currículo-base anterior e oferecer upload/colagem.
- [ ] Não declarar a etapa concluída apenas porque o usuário abriu o diálogo OAuth; exigir confirmação do resultado persistido.
- [ ] Permitir pular a etapa de currículo conforme a política definida na seção 3.2, com aviso das funcionalidades dependentes.

**Validação:** o usuário vê a prévia, pode editar e confirma a importação; o currículo anterior nunca é substituído silenciosamente.

### Etapa 4 — Fontes de vagas

- [ ] Carregar a lista de fontes já implementadas e suas configurações atuais.
- [ ] Permitir habilitar/desabilitar fontes existentes usando os mesmos serviços e validações de Settings.
- [ ] Explicar limites, credenciais ou requisitos de cada fonte quando já houver informação disponível no produto.
- [ ] Executar teste/validação real apenas quando a fonte oferecer esse recurso.
- [ ] Mostrar estados distintos de configurada, validada, indisponível e ignorada.
- [ ] Não criar novas fontes nem alterar o motor de discovery neste checkpoint.
- [ ] Permitir continuar sem configurar fontes opcionais, mantendo as fontes padrão disponíveis conforme a configuração atual.

**Validação:** a etapa reflete o estado persistido das fontes; não usa apenas o avanço do wizard como sinal de sucesso.

### Etapa 5 — Filtros e preferências de discovery

- [ ] Carregar filtros e preferências já suportados pelo discovery atual.
- [ ] Permitir revisar os valores existentes por meio dos mesmos controles/serviços de Settings.
- [ ] Apresentar keywords do currículo somente se já houver currículo e extração disponível.
- [ ] Reutilizar a funcionalidade de edição de keywords quando a revisão F5-R1 do release estiver concluída; não reimplementar extração ou gerenciamento de keywords dentro do CP19.
- [ ] Explicar de forma simples o efeito dos filtros sem prometer resultados que o motor atual não oferece.
- [ ] Salvar e validar a persistência antes de concluir a etapa.
- [ ] Permitir pular a etapa e configurar posteriormente em Settings.

**Validação:** os filtros usados pelo discovery após o onboarding são os mesmos valores persistidos e apresentados em Settings.

### Etapa 6 — Inteligência artificial

- [ ] Reutilizar o catálogo de presets, os perfis de IA, a indicação de free tier e a configuração BYOK da F4.
- [ ] Permitir selecionar um perfil existente ou criar/configurar um perfil pelos componentes já existentes.
- [ ] Permitir escolher opção gratuita/local homologada ou configurar chave própria.
- [ ] Executar teste real de conectividade/completion quando o usuário solicitar e houver configuração suficiente.
- [ ] Exibir sucesso, erro e ação sugerida usando o catálogo amigável de erros da F2/B6.
- [ ] Não armazenar ou devolver API keys no estado do onboarding.
- [ ] Se IA não estiver configurada ou o teste falhar, permitir continuar e explicar quais recursos dependem de IA.
- [ ] Não criar login OAuth para provedores de IA.

**Validação:** o perfil selecionado permanece ativo após refresh/relogin e o teste usa o mesmo serviço disponível em Settings.

### Etapa 7 — Resumo e conclusão

- [ ] Mostrar resumo de perfil, currículo, fontes, filtros e IA.
- [ ] Separar visualmente itens concluídos, ignorados, opcionais e que precisam de atenção.
- [ ] Exibir links para corrigir etapas incompletas sem reiniciar o fluxo.
- [ ] Permitir concluir mesmo com etapas opcionais ignoradas ou com falha, respeitando a política da seção 3.2.
- [ ] Registrar `completed_at` e `onboarding_version` no backend.
- [ ] Direcionar o usuário para a home/feed de vagas após conclusão.
- [ ] Permitir abrir novamente o onboarding em Settings em modo de revisão, sem apagar configurações existentes.

**Validação:** a conclusão é persistida no servidor e não é inferida apenas pelo estado visual do frontend.

## 5. Persistência e contratos de API

### 5.1 Estado do onboarding

Persistir por usuário e versão do fluxo, no mínimo:

- `user_id`;
- `onboarding_version`;
- `status`;
- `current_step`;
- estado de cada etapa (`pending`, `completed`, `skipped`, `needs_attention`);
- `started_at`, `updated_at` e `completed_at`;
- metadados não sensíveis necessários para retomada e apresentação do resumo.

Não duplicar no registro de onboarding dados de perfil, currículo, fontes, filtros, credenciais ou perfis de IA. Guardar referências/estados de etapa e consultar os serviços donos desses dados.

### 5.2 API esperada

Ajustar aos padrões existentes do projeto; nomes abaixo são contratos conceituais:

- `GET /api/onboarding/status` — estado e versão aplicável ao usuário autenticado;
- `PUT /api/onboarding/progress` — atualizar etapa atual e estados de etapa;
- `POST /api/onboarding/complete` — concluir a versão atual após validar regras;
- `POST /api/onboarding/reopen` — iniciar revisão manual sem apagar os dados existentes.

O backend deve:

- derivar usuário da sessão CP18;
- validar transições de estado e versão do wizard;
- impedir atualização cruzada de progresso entre usuários;
- rejeitar conclusão inválida quando uma etapa realmente obrigatória estiver ausente;
- não aceitar do frontend como verdade afirmações de que currículo, fonte ou perfil de IA foram persistidos/testados;
- consultar os serviços correspondentes ou validar seus estados persistidos.

### 5.3 OAuth e escopos Gmail

- O CP19 deve preservar a origem do login de identidade (`local`, `google_oidc` ou outro valor suportado pelo CP18) e os escopos concedidos, sem copiar tokens para o frontend.
- Login Google durante o onboarding não deve solicitar automaticamente permissão de leitura da caixa postal.
- A tela de configuração de e-mail da F2 deve consumir o estado de identidade/conexões do CP18 para aplicar a regra de exibir o botão de login Gmail somente quando o usuário não utilizou Gmail para entrar no onboarding.
- Se a identidade Google já foi usada, mas não existe escopo de leitura de mailbox, a configuração de e-mail deverá oferecer o fluxo de consentimento incremental/autorização de mailbox ou alternativa IMAP, sem apresentar uma segunda conexão de identidade como se fosse necessária.
- A regra de conta única de e-mail pertence à F2/CP17 e deve continuar implementada no serviço/store de e-mail, não no estado do wizard.

## 6. Interface e acessibilidade

- [ ] Criar rota/layout de onboarding separado da navegação principal, protegido pela sessão CP18.
- [ ] Aplicar o design system openJob: tipografia, espaçamentos, cores, superfícies, controles e feedbacks existentes.
- [ ] Usar progressive disclosure: uma tarefa principal por etapa, indicador de progresso e resumo compreensível.
- [ ] Exibir ações claras de voltar, continuar, salvar/sair, pular etapa e concluir.
- [ ] Distinguir etapa obrigatória, recomendada e opcional sem depender somente de cor.
- [ ] Manter responsividade para smartphone, tablet e desktop.
- [ ] Garantir navegação por teclado, foco visível, labels, mensagens associadas a campos e anúncios de mudança de etapa para leitor de tela.
- [ ] Respeitar `prefers-reduced-motion` e não depender de animações para comunicar estado.
- [ ] Reutilizar componentes e feedbacks de Settings; evitar um formulário paralelo que produza configurações divergentes.
- [ ] Localizar todos os textos em `pt-BR`, `en-US` e `es-419`.

## 7. Critérios de verificação e aceite

- [ ] Usuário novo autenticado e sem conclusão da versão atual é encaminhado ao onboarding.
- [ ] Usuário que já concluiu a versão atual vai diretamente à home/feed.
- [ ] Usuário existente migrado não é forçado a repetir o wizard; revisão manual está disponível em Settings.
- [ ] O progresso pertence ao usuário autenticado e permanece após refresh, logout/login e retomada em outro dispositivo/navegador da mesma instância.
- [ ] Dois usuários simultâneos não veem nem alteram o progresso um do outro.
- [ ] Perfil é salvo e relido pelo serviço do CP18.
- [ ] Currículo importado é apresentado em prévia e só substitui a versão anterior após confirmação explícita.
- [ ] LinkedIn aparece somente se o adaptador do CP18 estiver habilitado; no-go ou indisponibilidade não bloqueia o onboarding.
- [ ] Dados retornados pelo LinkedIn são exibidos para revisão e somente campos autorizados são importados.
- [ ] Login Google não é confundido com autorização Gmail; origem do login e escopos são preservados para a regra de configuração de e-mail.
- [ ] Fontes, filtros e IA são persistidos/validados pelos serviços existentes, sem duplicar suas regras.
- [ ] IA não configurada, falha de provedor, fonte indisponível ou falha de importação opcional não impedem a conclusão.
- [ ] Resumo diferencia etapas concluídas, ignoradas e pendentes de atenção.
- [ ] Conclusão direciona à home/feed; revisão posterior em Settings funciona sem apagar dados.
- [ ] `make test`, `make vet` e `make test-ui` passam; regressão de Settings, autenticação, discovery, currículo e feed verde.

## 8. Testes obrigatórios

### Backend

- [ ] Criação/leitura/atualização de progresso por usuário.
- [ ] Rejeição de progresso enviado para outro usuário ou sem sessão.
- [ ] Validação de transições e versão do onboarding.
- [ ] Conclusão bloqueada quando regra obrigatória não estiver satisfeita.
- [ ] Estado de conclusão idempotente e reabertura sem apagar configurações.
- [ ] Migração de contas existentes sem forçar onboarding concluído ou perder estado anterior.
- [ ] Consumo correto de estado OAuth/identidade e escopos sem retornar tokens.

### E2E e UI

- [ ] Fluxo feliz completo com conta local.
- [ ] Fluxo com login Google e verificação de que login não concede `gmail.readonly` implicitamente.
- [ ] Fluxo LinkedIn disponível e com confirmação de importação, quando o adaptador de CP18 estiver habilitado.
- [ ] Fluxo LinkedIn `no-go`/indisponível com fallback para upload/colagem.
- [ ] Upload, colagem, prévia, edição e confirmação do currículo.
- [ ] Falha e retry em fonte, importação e teste de IA.
- [ ] Pular etapas opcionais e concluir sem IA; testar também a política aprovada para currículo vazio.
- [ ] Refresh, logout/login e retomada exatamente na etapa persistida.
- [ ] Usuário existente com onboarding concluído vai direto ao feed.
- [ ] Reabrir em Settings não apaga dados nem reinicia configurações.
- [ ] Testes visuais e acessibilidade nas resoluções usadas pelo projeto.

## 9. Entregáveis

- Rota e layout dedicados para onboarding web.
- Estado persistido por usuário e versão, com retomada e reabertura.
- Integração visual com perfil e currículo-base do CP18.
- Etapa de LinkedIn integrada ao serviço de importação do CP18 quando viável, com fallback manual.
- Etapas de fontes, discovery e IA reutilizando os serviços existentes.
- Resumo de conclusão com estados reais das configurações.
- Testes backend, migrations necessárias, E2E, acessibilidade, i18n e documentação.

## 10. Registrar

- ADR de experiência de primeira execução, estados do wizard e política de obrigatoriedade/opcionalidade.
- Decisão documentada sobre permitir conclusão com currículo-base vazio.
- Contrato/versionamento do estado de onboarding e estratégia de migração de usuários existentes.
- Decisão de UX para LinkedIn conforme o resultado do spike/adaptador do CP18.
- Registro de como a origem do login Google e os escopos Gmail são apresentados aos serviços de configuração de e-mail.
- Testes E2E para fluxo feliz, skip/retry, retomada, usuário existente, OAuth disponível e fallback `no-go`.

## 11. Dependências, riscos e mitigação

| Risco/dependência | Impacto | Mitigação |
|---|---|---|
| CP18 incompleto ou contratos instáveis | Onboarding duplicaria autenticação/importação | Não iniciar integração final antes do gate CP18; consumir contratos versionados. |
| LinkedIn sem acesso aprovado a dados profissionais | Etapa de currículo não pode importar dados | Apresentar a opção somente quando o adaptador estiver habilitado; manter upload/colagem. |
| Login Google confundido com acesso Gmail | Expectativa incorreta ou autorização insuficiente | Registrar provedor e escopos; consentimento de mailbox continua explícito e separado. |
| Wizard longo ou cansativo | Abandono antes de chegar às vagas | Progressive disclosure, pular opcionais, salvar progresso e permitir retomada. |
| Divergência entre onboarding e Settings | Configurações inconsistentes | Reutilizar APIs, serviços, componentes e validações de Settings. |
| Rate limit/indisponibilidade de provedor | Falha em teste de IA ou importação | Tratar como etapa opcional; preservar dados e permitir retry/skip. |
| Usuário existente recebe wizard indevidamente | Regressão e fricção | Migração explícita de estado; não inferir primeira execução somente por ausência de registro. |

## 12. Fora de escopo

- Implementar autenticação, gestão de sessão, criação de usuário ou isolamento de dados (CP18).
- Implementar callbacks, troca de tokens ou adaptadores OAuth no frontend/backend do onboarding (CP18).
- Scraping de LinkedIn ou importação de campos não disponibilizados oficialmente.
- OAuth para provedores de IA.
- Configuração completa de sincronização de e-mail, leitura de mailbox ou gestão de conta única (F2/CP17); o CP19 apenas preserva a origem do login e os escopos necessários para a integração.
- Criar fontes de vagas ou alterar algoritmos de discovery, filtros ou Score Fit.
- Implementar extração documental genérica, OCR ou ampliar formatos suportados (futuro checkpoint de Inteligência de documentos).
- Implementar o instalador, bootstrap técnico, host/porta ou abertura automática do navegador (CP11).

**Gate final do CP19:** usuário autenticado novo conclui ou dispensa de forma consciente as etapas opcionais; perfil e currículo são persistidos pelo CP18; LinkedIn é apresentado somente quando tecnicamente viável e com revisão explícita; fontes, filtros e IA usam os serviços existentes; progresso é recuperável e isolado por usuário; e a conclusão leva à home/feed sem bloquear o produto por integrações opcionais.
