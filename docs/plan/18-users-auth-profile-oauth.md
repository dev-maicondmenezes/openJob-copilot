# Checkpoint 18 — Usuários, autenticação e perfil profissional (5–8 dias)

> Origem: uso compartilhado na LAN + preparação da primeira execução. Pré-requisito: CP17 concluído e migrations/testes atuais verdes. Este checkpoint estabelece identidade e propriedade dos dados; não implementa o onboarding guiado nem o instalador final.

## 1. Objetivo

- Permitir que mais de uma pessoa use a mesma instância openJob na rede local, cada uma com conta e dados isolados.
- Criar autenticação e sessão seguras para a SPA e API.
- Manter por usuário o perfil profissional e o currículo-base, com atualização explícita.
- Preparar OAuth para autenticação/conexão e importação quando os provedores realmente disponibilizarem os dados.

**Princípio:** login OAuth e importação de currículo são capacidades distintas. Autorizar identidade (por exemplo, e-mail) não significa que o provedor disponibilize histórico profissional ou currículo completo.

## 2. Pré-requisitos e decisões

- Reutilizar SQLite, migrations versionadas e AES-GCM/master key existentes.
- Definir sessão segura para aplicação local acessada pelo navegador; proteger contra CSRF, fixation, expiração e logout.
- OAuth somente com APIs oficiais, escopos mínimos e consentimento explícito. Não implementar scraping de LinkedIn, automação de candidatura ou contorno de restrições.
- **Gate de viabilidade:** avaliar Gmail e LinkedIn separadamente para login, e-mail básico e importação de perfil/currículo. Não prometer dados que API/escopos aprovados não retornem.

## 3. Modelo de dados

- users: ID, nome de exibição, e-mail normalizado quando disponível, status e timestamps.
- user_sessions ou mecanismo equivalente: token de sessão armazenado com segurança (preferir hash no banco), expiração, revogação e timestamps.
- user_profiles: dados profissionais editáveis, preferências e metadados do currículo-base.
- user_oauth_connections: provider, subject/provider-user-id, escopos concedidos, estado/expiração e tokens cifrados AES-GCM.
- Currículo-base por usuário: conteúdo normalizado editável (Markdown), origem, data de importação e metadados; preservar versão anterior ao substituir.

Adicionar ownership user_id a todas as entidades pessoais/operacionais: candidaturas, currículo, configurações, perfis de IA, contas de e-mail, mensagens, tags pessoais, tarefas e notificações. Confirmar entidades restantes por auditoria do schema.

## 4. Tarefas

### 4.1 Autenticação e sessão
- Criar fluxo seguro para primeiro usuário; não manter endpoint público de criação administrativa após bootstrap.
- Login/logout, sessão persistente com expiração, renovação e revogação.
- Hash de senha com algoritmo seguro e mantido (Argon2id ou equivalente); nunca armazenar senha em texto puro.
- Cookies HttpOnly, SameSite apropriado e Secure quando HTTPS estiver habilitado; validar Origin/CSRF em mutações.
- Middleware de autenticação/autorização; API não confia em user_id enviado pelo cliente.

### 4.2 Isolamento e migração
- Aplicar ownership no repositório/queries, não apenas na UI.
- Migrations para banco novo e upgrade do banco single-user atual.
- Migrar dados existentes para o usuário inicial sem perda; documentar backup/rollback.
- Testes negativos de acesso cruzado para jobs, curriculum, settings, email, tasks e notifications.

### 4.3 Perfil e currículo-base
- CRUD de perfil: nome, headline, localização profissional opcional, experiência, skills e preferências relevantes.
- Currículo-base importável por PDF, DOCX, Markdown/TXT e colagem de texto (confirmar formatos suportados pelo stack).
- Extrair para Markdown/estrutura editável; apresentar prévia e pedir confirmação antes de substituir.
- Persistir currículo-base por usuário; permitir atualizar via novo arquivo ou texto sem apagar silenciosamente a versão anterior.

### 4.4 OAuth e importação
- Framework OAuth com state/PKCE quando suportado, callback local seguro, escopos mínimos, revogação/desconexão e tokens cifrados.
- Gmail: avaliar login/e-mail básico e acesso a anexos de currículo separadamente. Qualquer leitura de e-mail deve ser opt-in, escopada e explicada; não reutilizar silenciosamente a conexão de tracking do CP17.
- LinkedIn: spike de API/Partner Program e campos retornados. Se perfil/currículo completo não estiver disponível para app pessoal, registrar no-go e manter upload/colar texto.
- Mostrar quais dados serão importados e permitir revisão antes de salvar.

## 5. API e UI

API esperada (ajustar aos padrões existentes):
- POST /api/auth/register (somente bootstrap/fluxo autorizado)
- POST /api/auth/login, POST /api/auth/logout, GET /api/auth/me
- GET/PUT /api/profile
- GET/PUT /api/curriculum/base e endpoint de upload/importação
- endpoints de conexão OAuth, callback, status e desconexão

UI: login, estado de sessão, perfil/currículo-base e gestão de conexões; erros amigáveis e confirmação para substituir currículo.

## 6. Critérios de verificação

- Dois usuários na mesma instância autenticam e usam o app simultaneamente.
- Um usuário não consegue consultar, alterar, exportar ou associar dados do outro, mesmo manipulando IDs/requests.
- Banco existente migra para o usuário inicial sem perda; banco novo inicia corretamente.
- Login/logout, expiração e revogação funcionam; tokens não aparecem em logs nem respostas.
- Upload/colagem converte para conteúdo editável, apresenta prévia e salva no perfil correto.
- OAuth funciona apenas para escopos realmente aprovados; no-go do LinkedIn mantém fallback manual.
- make test, make vet, make test-ui e testes de migration/ownership passam.

## 7. Registrar

- ADR de identidade, sessão e isolamento local multiusuário.
- ADR de OAuth: separação entre autenticação, conexão e importação de dados.
- Testes de autorização por usuário e migração single-user → multi-user.
- Riscos: exposição na LAN, OAuth sem acesso a dados completos, parser inconsistente e ownership em tabelas legadas.

## 8. Fora de escopo

- Onboarding completo (CP19).
- Empacotamento, bootstrap técnico e abertura do navegador (CP11).
- Scraping de LinkedIn, Easy Apply, envio de mensagens/candidaturas ou importação não autorizada.
- Multi-tenant hospedado na nuvem; a instância continua local-first.
