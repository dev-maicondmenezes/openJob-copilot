# Checkpoint 18 — Usuários, autenticação e perfil profissional

> **Estimativa inicial:** 5–8 dias úteis, sujeita à auditoria de ownership e migrations.  
> **Status:** proposto.  
> **Pré-requisitos:** release MVP F0–F8 concluído; CP17 e gerenciamento de perfis de IA da F4 disponíveis; migrations e suítes atuais verdes.  
> **Relação com CP19:** este checkpoint fornece identidade, sessão, perfil, currículo-base e serviços de conexão/importação. O onboarding gráfico e sua apresentação ficam no CP19.  
> **Relação com CP11:** instalação, bootstrap técnico do executável, host/porta e abertura do navegador continuam fora deste checkpoint.

## 1. Objetivo

Permitir que várias pessoas utilizem a mesma instância local do openJob, inclusive pela LAN, com contas autenticadas e isolamento efetivo de seus dados. Cada usuário deverá ter perfil profissional e currículo-base próprios, além de conexões externas explicitamente autorizadas.

O checkpoint entrega a fundação de identidade e propriedade dos dados para o onboarding CP19, para a Inteligência de documentos e para a personalização individual do Score Fit — Parte 2.

**Princípio central:** autenticação, conexão com um provedor e importação de dados são capacidades distintas. Entrar com Google não concede automaticamente acesso à caixa de entrada Gmail; conectar LinkedIn não garante acesso a todo o histórico profissional; nenhuma integração pode prometer dados que os escopos aprovados não forneçam.

## 2. Escopo e decisões arquiteturais

### 2.1 Identidade local-first

- A instância continua local-first e poderá ser acessada por mais de uma pessoa na rede local.
- SQLite, migrations versionadas e AES-GCM/master key existentes serão reutilizados.
- Não implementar multi-tenant hospedado na nuvem.
- Não adicionar um provedor externo de identidade como dependência obrigatória. O login local deve continuar sendo um caminho funcional.
- O primeiro usuário criado no bootstrap será o administrador inicial da instância.
- Depois do bootstrap, não haverá cadastro público irrestrito. Novas contas serão criadas por fluxo autenticado e autorizado do administrador, com ativação de credencial pelo próprio usuário.

### 2.2 Separação entre dados globais e dados por usuário

Antes de adicionar `user_id` indiscriminadamente, auditar o schema e classificar as entidades:

- **Globais/compartilhadas:** catálogo canônico de vagas deduplicadas, fontes e dados técnicos compartilháveis da instalação, conforme arquitetura da Parte 1.
- **Pertencentes ao usuário:** perfil, currículo, candidaturas, preferências, filtros pessoais, feedback, vagas salvas/ocultadas, perfis de IA, contas/conexões de e-mail, mensagens associadas à conta, tags pessoais, tarefas e notificações.
- **Relacionamentos usuário–vaga:** usar associações com ownership quando uma vaga global for salva, pontuada, ocultada, associada a uma candidatura ou apresentada como parte do histórico de um usuário.

A propriedade será aplicada nas queries e nos repositórios do backend, não apenas escondida na interface. O backend sempre derivará a identidade da sessão autenticada.

### 2.3 Autenticação e sessão

- Login local por credencial, com hash de senha usando Argon2id ou algoritmo equivalente mantido e aprovado.
- Sessões opacas com token aleatório de alta entropia; persistir preferencialmente o hash do token, nunca o token em claro.
- Cookies `HttpOnly`, `SameSite` apropriado e `Secure` quando HTTPS estiver habilitado.
- Proteções contra CSRF, session fixation, força bruta e reutilização de sessões revogadas.
- Expiração, renovação e logout com revogação no servidor.
- Validar `Origin`/CSRF nas operações mutáveis.
- Não armazenar senha, token de sessão ou segredo em texto puro.

### 2.4 OAuth: identidade, conexão e importação

Manter três finalidades distintas:

1. **Login/identidade:** identificar a pessoa por um provedor, usando somente os escopos necessários de identidade.
2. **Conexão:** autorizar um serviço específico, como acesso à caixa de entrada, com consentimento e escopos próprios.
3. **Importação:** ler campos realmente disponibilizados pelo provedor e apresentar uma prévia para revisão antes de persistir.

Regras:

- Implementar somente OAuth/OIDC oficial, com `state`, PKCE quando suportado, callback validado, escopos mínimos, desconexão/revogação e tokens cifrados.
- **Google:** login OIDC (`openid`, `email`, `profile`) não equivale a permissão para ler Gmail. A sincronização de e-mail do CP17 continua exigindo consentimento específico para seus escopos; não reutilizar silenciosamente uma autorização de login.
- **LinkedIn:** a integração técnica de conexão/importação pertence a este checkpoint, mas sua entrada de produto será apresentada pelo onboarding CP19. Fazer spike de disponibilidade de API, aprovação/Partner Program e campos retornados. Implementar apenas os campos realmente autorizados. Se o acesso ao histórico profissional não for aprovado, registrar `no-go` e manter importação manual como fallback.
- Não criar uma fase OAuth separada para LinkedIn: o serviço/adaptador é deste CP18 e a experiência guiada de conexão é do CP19.
- Não implementar OAuth para provedores de LLM. A antiga fase de OAuth de provedores foi retirada do plano.
- Não fazer scraping, automação de candidatura, Easy Apply ou contorno de restrições de provedores.

## 3. Modelo de dados

A estrutura final deve seguir os padrões reais do projeto e ser formalizada em migrations versionadas. Entidades mínimas esperadas:

- `users`: ID, nome de exibição, identificador de login/e-mail normalizado quando disponível, hash de senha quando login local estiver habilitado, papel (`admin`/`member`), status e timestamps.
- `user_sessions` (ou mecanismo equivalente): hash do token, usuário, criação, expiração, última atividade e revogação.
- `user_profiles`: headline, localização profissional opcional, experiência, skills e preferências profissionais editáveis.
- `user_resume_versions` (ou equivalente): currículo-base por usuário, conteúdo editável, origem, versão, data de importação e metadados. Preservar versão anterior ao substituir.
- `user_oauth_connections`: provedor, subject/provider-user-id, finalidade da conexão, escopos concedidos, estado/expiração e tokens cifrados AES-GCM.
- `user_invitations` ou mecanismo equivalente: convite/token de ativação com validade, uso único e vínculo ao administrador que criou a conta, caso esse seja o mecanismo escolhido na auditoria.

A auditoria do schema deve gerar uma matriz explícita de ownership para todas as tabelas existentes. Não adicionar `user_id` a tabelas globais quando a modelagem correta for uma associação por usuário.

## 4. Tarefas

### 4.1 Auditoria de dados e plano de migração

- [ ] Inventariar tabelas, stores, repositórios, handlers, workers, jobs e tarefas agendadas.
- [ ] Classificar cada entidade como global, por usuário ou associação usuário–entidade.
- [ ] Identificar todos os pontos de leitura/escrita que precisam receber contexto de usuário.
- [ ] Definir a estratégia de migração do banco single-user atual para o modelo multiusuário.
- [ ] Preparar backup, rollback e validação de integridade antes de executar migrations destrutivas ou de grande alcance.
- [ ] Documentar explicitamente dados globais que continuarão compartilhados entre usuários da mesma instância.

### 4.2 Bootstrap de usuário e gestão mínima de contas

- [ ] Criar o primeiro administrador de forma controlada no bootstrap da aplicação.
- [ ] Garantir que o bootstrap não possa criar administradores repetidamente nem reabrir cadastro público após a inicialização.
- [ ] Implementar criação/convite de usuários adicionais por administrador, com token de ativação de uso único e validade limitada, ou mecanismo equivalente aprovado na auditoria.
- [ ] Permitir ativar conta, definir/alterar senha e desativar usuário.
- [ ] Impedir que uma conta desativada inicie novas sessões e revogar sessões existentes.
- [ ] Evitar introduzir sistema complexo de permissões: neste checkpoint, limitar-se a administrador da instância e usuário comum.

### 4.3 Login, sessão e segurança

- [ ] Implementar login local, logout, consulta da sessão atual e renovação/expiração.
- [ ] Implementar armazenamento seguro de senha com Argon2id ou equivalente aprovado.
- [ ] Implementar cookies e proteção CSRF/Origin conforme a seção 2.3.
- [ ] Rotacionar identificador de sessão após autenticação e mudança de privilégio.
- [ ] Aplicar limite/backoff para tentativas de login e respostas que não revelem se um e-mail existe.
- [ ] Implementar revogação de sessão no logout, desativação de conta e alteração de credencial.
- [ ] Garantir que APIs privadas retornem `401` sem sessão válida e `403` para falta de autorização.

### 4.4 Isolamento e migração de ownership

- [ ] Aplicar ownership nas queries e repositórios de todas as entidades pessoais/operacionais.
- [ ] Revisar handlers REST, rotas de exportação, downloads, WebSocket/eventos se existentes e workers em background.
- [ ] Garantir que tarefas assíncronas preservem o usuário proprietário e não executem com contexto implícito compartilhado.
- [ ] Migrar dados existentes para o administrador inicial sem perda, preservando IDs quando possível.
- [ ] Atribuir relacionamentos pessoais ao administrador inicial; manter o catálogo global de vagas e fontes conforme a decisão de arquitetura.
- [ ] Criar constraints e índices por usuário onde necessários.
- [ ] Garantir que manipular IDs no frontend/API não permita ler, alterar, excluir, exportar ou associar dados de outro usuário.
- [ ] Testar especialmente `curriculum`, `applications`, `settings`, `llm_profiles`, `email_accounts`, `email_messages`, `tags`, `tasks` e `notifications`.

### 4.5 Perfil profissional e currículo-base

- [ ] Implementar CRUD de perfil profissional: nome, headline, localização profissional opcional, experiência, skills e preferências relevantes.
- [ ] Manter currículo-base individual, editável e versionado por usuário.
- [ ] Permitir edição/colagem de texto e Markdown.
- [ ] Reutilizar somente os formatos e parsers que já estejam efetivamente suportados pelo produto; não ampliar a matriz de formatos neste checkpoint.
- [ ] Apresentar prévia do conteúdo importado antes de salvar/substituir.
- [ ] Exigir confirmação explícita para substituir currículo-base.
- [ ] Preservar a versão anterior e registrar origem/data da atualização.
- [ ] Expor uma forma de selecionar o currículo-base ativo, se mais de uma versão for mantida.
- [ ] Não implementar a camada genérica de extração, classificação e inteligência documental: ela será especificada em checkpoint próprio de Inteligência de documentos.

### 4.6 Conexões OAuth e importação de perfil

- [ ] Implementar infraestrutura reutilizável para state, PKCE quando suportado, callback, validação de issuer/audience/redirect, escopos, expiração, desconexão e revogação.
- [ ] Cifrar tokens em repouso com o mecanismo AES-GCM/master key existente.
- [ ] Garantir que tokens e códigos de autorização nunca apareçam em logs, erros ou respostas da API.
- [ ] Implementar login Google OIDC somente com escopos de identidade, caso o spike e as credenciais de aplicação estejam disponíveis.
- [ ] Fazer spike de LinkedIn e documentar campos e escopos realmente acessíveis.
- [ ] Se viável, implementar adaptador backend para importar os campos profissionais aprovados; a interface guiada de conexão será chamada pelo CP19.
- [ ] Mostrar a lista de campos que serão importados e exigir revisão/confirmar antes de atualizar perfil ou currículo.
- [ ] Se LinkedIn não for viável, registrar decisão `no-go`, manter upload/colagem e não bloquear o CP19.
- [ ] Manter o acesso Gmail para sincronização de e-mail do CP17 como autorização separada e explícita; login Google não deve conceder leitura de caixa de entrada.
- [ ] Não implementar conexão OAuth de provedores de LLM.

### 4.7 API e interface mínima

API esperada, ajustada aos padrões existentes:

- `POST /api/auth/login`
- `POST /api/auth/logout`
- `GET /api/auth/me`
- endpoint protegido de bootstrap/convite/ativação de usuário
- endpoints administrativos mínimos para listar, convidar, desativar e reativar usuários
- `GET/PUT /api/profile`
- `GET/PUT /api/curriculum/base` e endpoint de importação nos formatos já suportados
- endpoints de conexão OAuth: iniciar, callback, status, desconexão e revogação

UI mínima deste checkpoint:

- tela de login e ativação de conta;
- estado de sessão e logout;
- formulário de perfil e currículo-base;
- gestão mínima de usuários para administrador;
- estado de conexões e revisão de dados importados.

Não implementar o wizard de primeira execução, a sequência de etapas ou o design completo do onboarding. Isso pertence ao CP19.

## 5. Critérios de verificação e aceite

- [ ] Primeiro administrador é criado uma única vez por mecanismo controlado.
- [ ] Administrador consegue criar/convidar pelo menos um segundo usuário sem cadastro público irrestrito.
- [ ] Dois usuários autenticam e utilizam a mesma instância simultaneamente.
- [ ] Nenhum usuário consegue consultar, alterar, excluir, exportar ou associar dados privados de outro, inclusive manipulando IDs e requests.
- [ ] Dados globais de vagas permanecem íntegros e relacionamentos pessoais ficam isolados.
- [ ] Banco single-user migra para multiusuário sem perda; banco novo sobe corretamente.
- [ ] Login, logout, expiração, renovação, revogação, proteção CSRF e bloqueio de conta desativada funcionam.
- [ ] Senhas e tokens nunca são armazenados ou registrados em texto puro.
- [ ] Perfil e currículo-base são persistidos por usuário; substituir currículo exige confirmação e preserva versão anterior.
- [ ] O login Google, quando habilitado, utiliza apenas escopos de identidade; acesso Gmail exige autorização de mailbox própria.
- [ ] LinkedIn é implementado somente com API/escopos aprovados; no-go documentado não bloqueia o checkpoint nem o onboarding.
- [ ] Nenhuma conexão OAuth de provedor LLM é criada.
- [ ] `make test`, `make vet`, `make test-ui` e testes de migrations/ownership passam.

## 6. Entregáveis

- Modelo de identidade, sessão e ownership implementado e versionado.
- Migração single-user → multi-user com backup/rollback documentados.
- Login local seguro, bootstrap do administrador e criação autorizada de usuários.
- Perfil profissional e currículo-base por usuário.
- Infraestrutura de conexões OAuth separada por finalidade.
- Spike/decisão de viabilidade do LinkedIn e, se viável, adaptador técnico consumível pelo CP19.
- Testes negativos de isolamento, autenticação, sessão e migração.
- Documentação de operação e segurança para uma instância local compartilhada na LAN.

## 7. Registrar

- ADR de identidade, login local, sessão e isolamento multiusuário.
- ADR de classificação de dados globais versus dados por usuário.
- ADR de separação entre identidade, conexão OAuth e importação de dados.
- Registro do spike LinkedIn com escopos/campos acessíveis e decisão go/no-go.
- Matriz de ownership por tabela, com testes associados.
- Plano de backup/rollback da migração single-user → multi-user.

## 8. Dependências e integração com os próximos checkpoints

- **CP19 — Onboarding web:** depende de autenticação, sessão, perfil, currículo-base e serviços de conexão/importação definidos aqui. CP19 será responsável por apresentar o fluxo de conexão/importação ao usuário; não duplicará OAuth no frontend/backend.
- **Checkpoint de Inteligência de documentos:** poderá ampliar parsers e extração estruturada sobre o currículo-base deste checkpoint; CP18 não antecipa essa camada.
- **Checkpoint de Tailoring determinístico:** consumirá perfil/currículo do usuário, com ownership já aplicado.
- **Score Fit — Parte 2:** dependerá da identidade e do isolamento para persistir exposições, eventos e preferências individuais.

## 9. Fora de escopo

- Onboarding gráfico e wizard de primeira execução (CP19).
- Instalação, distribuição, bootstrap técnico do executável e abertura automática do navegador (CP11).
- Multi-tenant hospedado em nuvem.
- Scraping de LinkedIn, Easy Apply, mensagens/candidaturas automatizadas ou importação não autorizada.
- Parser genérico multimodal, OCR e Inteligência de documentos completa.
- Tailoring determinístico, Evidence Mapping e geração/validação de currículo adaptado.
- Personalização comportamental e re-ranking do Score Fit — Parte 2.
- OAuth para provedores de LLM.

**Gate final do CP18:** duas ou mais contas utilizam uma única instância local com isolamento comprovado; migração concluída sem perda; perfil/currículo-base individuais funcionais; sessões seguras; e conexões/importações externas limitadas aos escopos realmente concedidos. CP19 pode então construir o onboarding sobre contratos estáveis, sem reimplementar identidade ou integração.
