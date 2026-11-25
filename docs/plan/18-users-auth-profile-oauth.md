# Checkpoint 18 — Usuários, autenticação e perfil profissional

> **Versão:** 2 — arquitetura de autenticação baseada em biblioteca especializada, condicionada à POC.  
> **Estimativa inicial:** 5–8 dias úteis após aprovação da POC, sujeita à auditoria de ownership, migrations e integração. A POC deve ser estimada e executada antes de confirmar o esforço de implementação.  
> **Status:** proposto — seleção tecnológica pendente de POC.  
> **Pré-requisitos:** gates do release necessários ao CP18 aceitos; CP17/B12 e gerenciamento de perfis de IA da F4 disponíveis; migrations e suítes atuais verdes; toolchain Go e driver SQLite do projeto identificados.  
> **Relação com CP19:** este checkpoint fornece identidade, autenticação, sessão, perfil, currículo-base e serviços de conexão/importação aprovados. O onboarding gráfico e sua apresentação ficam no CP19.  
> **Relação com CP11:** instalação, distribuição, bootstrap técnico do executável, host/porta e abertura do navegador continuam fora deste checkpoint.

## 1. Objetivo

Permitir que várias pessoas utilizem a mesma instância local do openJob, inclusive pela LAN, com contas autenticadas e isolamento efetivo de seus dados. Cada usuário deverá ter perfil profissional e currículo-base próprios. O checkpoint também contempla login Google OIDC exclusivamente para identidade e, se tecnicamente viável e oficialmente autorizado, importação de dados profissionais do LinkedIn.

A implementação deverá priorizar o reaproveitamento de uma solução de autenticação Go pronta, reduzindo código sensível desenvolvido e mantido pelo openJob e o custo de implementação por agentes de IA. A primeira opção a avaliar será **ezauth integrado como biblioteca, acompanhado do ezauth-admin**, mas nenhuma dependência será adotada em produção antes de uma POC aprovada.

**Princípio central:** identidade, autenticação, autorização, ownership e importação de dados são responsabilidades relacionadas, mas distintas. A biblioteca poderá administrar credenciais, sessões e papéis; o openJob continuará responsável pelas regras de autorização do produto, pela propriedade dos dados profissionais e pela aplicação dos direitos de privacidade. Google OIDC será usado somente para identidade. Nenhuma funcionalidade deste checkpoint autoriza leitura ou sincronização de mailbox.

## 2. Decisões arquiteturais e limites

### 2.1 Estratégia tecnológica condicionada à POC

A ordem de decisão será:

1. **Preferencial:** incorporar `github.com/josuebrunel/ezauth` como biblioteca no backend Go existente e, se todos os critérios específicos forem aprovados, incorporar `github.com/josuebrunel/ezauth-admin` como painel administrativo embutido.
2. **Fallback:** implementar a camada de identidade com componentes especializados já compatíveis com o projeto, priorizando `scs` com store persistente, `coreos/go-oidc` para Google OIDC e `golang.org/x/crypto/argon2` para hash de senha. O openJob implementará somente os serviços administrativos, convites, RBAC e auditoria que não puderem ser reaproveitados.
3. **Não permitido:** adotar ezauth parcialmente, manter dois cadastros de usuário independentes ou aceitar sessões em memória como solução final apenas para preservar o painel administrativo.

A POC é uma etapa eliminatória. Se ezauth não atender aos requisitos de compatibilidade, persistência, segurança, integração e operação, a decisão será usar o fallback. A implementação do CP18 não deverá ficar bloqueada indefinidamente tentando adaptar uma biblioteca incompatível.

### 2.2 POC-01 — Compatibilidade, manutenção e licença

- [ ] Auditar a versão efetiva de Go utilizada pelo projeto, `go.mod`, toolchain de CI, imagens de build/runtime, Makefile, GoReleaser e pipelines de distribuição.
- [ ] Verificar a compatibilidade da versão candidata de ezauth e ezauth-admin com a toolchain e com os binários/artefatos atualmente publicados.
- [ ] Considerar o requisito documentado pelo ezauth-admin de EzAuth v0.32.0 ou superior e Go 1.25.4 ou superior; confirmar novamente os requisitos nas versões exatas fixadas para a POC.
- [ ] Fixar versões explícitas de ezauth, ezauth-admin e dependências transitivas. Não usar dependências flutuantes nem atualizar automaticamente para versões futuras.
- [ ] Conferir licença, avisos de copyright, dependências transitivas, histórico de releases, issues abertas relevantes e atividade de manutenção.
- [ ] Executar `govulncheck` e os scanners já utilizados pelo projeto sobre a árvore de dependências da POC.
- [ ] Verificar impacto no tamanho do binário, tempo de build, CGO, imagem distroless/non-root e pipeline de release.
- [ ] Confirmar que a integração pode ocorrer no mesmo processo HTTP, sem novo serviço, container, banco ou porta de rede.
- [ ] Registrar limitações conhecidas e custo estimado de manutenção da biblioteca e do painel.

**Aceite POC-01:** versões e licença identificadas, toolchain compatível, build/distribuição preservados ou alteração explicitamente aprovada, dependências verificadas e ausência de bloqueador técnico não mitigável.

### 2.3 POC-02 — SQLite, migrations e identidade canônica

- [ ] Demonstrar que ezauth consegue utilizar o mesmo arquivo físico SQLite da instalação atual, respeitando o driver e a configuração de conexão adotados pelo openJob.
- [ ] Confirmar a possibilidade de manter dados de autenticação e dados de negócio no mesmo banco, sem criar um segundo banco de usuários.
- [ ] Auditar nomes de tabelas, índices, constraints, migrations, transações, pragmas, WAL, `busy_timeout` e política de `foreign_keys` antes de executar migrations da biblioteca.
- [ ] Verificar colisões de nomes e compatibilidade entre migrations do openJob e migrations próprias do ezauth. As migrations da biblioteca não podem sobrescrever, renumerar ou reverter migrations do produto.
- [ ] Avaliar o efeito de `PRAGMA foreign_keys=ON` em todos os relacionamentos existentes. A documentação do ezauth alerta que habilitar foreign keys no SQLite pode fazer cascades e `SET NULL` anteriormente inativos passarem a ser executados; auditar previamente o impacto em dados existentes.
- [ ] Confirmar o comportamento de múltiplas conexões SQLite abertas para o mesmo arquivo e de inicialização concorrente das migrations.
- [ ] Definir estratégia de backup, rollback e recuperação para migrations de autenticação.
- [ ] Definir o identificador do usuário do ezauth como identidade canônica do openJob, caso a POC confirme compatibilidade.
- [ ] Migrar os usuários existentes sem criar um cadastro paralelo. Preservar IDs quando possível; se houver incompatibilidade, criar mapeamento explícito e auditável de migração, usado apenas durante a transição.
- [ ] Garantir que `user_profiles`, currículo, candidaturas e demais entidades pessoais referenciem a identidade canônica definida.
- [ ] Não criar uma segunda tabela `users` do openJob com ciclo de vida independente da identidade gerenciada pela biblioteca.

**Aceite POC-02:** banco físico SQLite compartilhado, migrations isoladas e reversíveis, identidade canônica definida, dados preservados e ausência de conflito ou cascata destrutiva não controlada.

### 2.4 POC-03 — Sessões persistentes e segurança de navegador

A documentação do ezauth informa que o fluxo de formulário/cookie utiliza cookies `HttpOnly` via SCS e que o store SCS padrão é em memória. Sessões nesse store não sobrevivem a reinicialização do processo. Portanto, a POC deverá substituir explicitamente o store em memória por persistência adequada.

- [ ] Demonstrar store de sessão SCS persistente baseado no SQLite já utilizado pelo openJob, por adaptador compatível ou implementação pequena e testada do contrato `scs.Store`.
- [ ] Não introduzir Redis, PostgreSQL, MySQL ou outro serviço apenas para sessões.
- [ ] Verificar sobrevivência de sessões após reinicialização do processo e após atualização/restart do executável.
- [ ] Testar expiração absoluta, expiração por inatividade, logout, revogação individual, revogação de todas as sessões e desativação de conta.
- [ ] Testar rotação de refresh token e detecção de reutilização quando o fluxo ezauth utilizado depender desses tokens.
- [ ] Confirmar que cookies são `HttpOnly`, têm `Secure` habilitado em HTTPS e utilizam `SameSite` compatível com a topologia real do frontend. Não presumir configuração não documentada: inspecionar `Set-Cookie` em testes.
- [ ] Garantir que tokens de sessão, refresh tokens e segredos nunca sejam expostos em HTML, JavaScript, `localStorage`, logs ou respostas indevidas.
- [ ] Confirmar armazenamento de tokens por hash, conforme mecanismo documentado pelo ezauth, e revisar backups e permissões do arquivo SQLite.
- [ ] Revisar o comportamento CSRF do ezauth para o modelo same-origin do openJob. A documentação descreve validação com Fetch Metadata e comparação de `Origin`, mas permite fallback quando ambos estão ausentes; adicionar proteção própria nas rotas mutáveis do browser caso a política do produto exija rejeitar requisições sem origem verificável.
- [ ] Testar requisições cross-site, same-site, same-origin, origem ausente, cabeçalhos Fetch Metadata ausentes e clientes não-browser.
- [ ] Garantir `401` para ausência de autenticação e `403` para falta de autorização nas APIs privadas.

**Aceite POC-03:** sessão persistente no SQLite, revogação eficaz, cookies com atributos corretos, CSRF validado para a topologia real, nenhum segredo exposto e nenhuma dependência de store em memória.

### 2.5 POC-04 — Google OIDC exclusivamente para identidade

- [ ] Configurar somente o provedor Google OIDC; não habilitar outros provedores oferecidos pela biblioteca.
- [ ] Verificar os escopos efetivamente enviados e concedidos. A finalidade permitida limita-se a identidade (`openid`, `email`, `profile`, conforme suporte efetivo da biblioteca e do provedor).
- [ ] Não solicitar escopos Gmail, Drive ou outros escopos não necessários à identidade.
- [ ] Validar issuer, audience, assinatura, expiração, state, nonce quando aplicável, redirect URI e PKCE quando suportado pelo fluxo implementado.
- [ ] Testar associação de uma identidade Google a uma conta local existente, evitando takeover por coincidência de endereço de e-mail sem verificação e confirmação adequadas.
- [ ] Definir comportamento para conta Google já associada a outro usuário, endereço de e-mail alterado, e-mail não verificado e conta local preexistente.
- [ ] Não criar automaticamente uma segunda conta quando a pessoa já possuir uma conta local, salvo regra explícita e segura de vinculação.
- [ ] Não armazenar tokens Google se a finalidade de login puder ser concluída sem mantê-los. Se algum token for tecnicamente indispensável, justificar, limitar sua finalidade e cifrá-lo.
- [ ] Garantir que o fluxo de identidade não invoque Gmail API, não solicite `gmail.readonly` e não ofereça consentimento incremental de mailbox.

**Aceite POC-04:** Google funciona somente como identidade, com escopos mínimos verificados, vinculação segura e nenhuma chamada ou permissão de mailbox.

### 2.6 POC-05 — Painel e autorização administrativa

- [ ] Integrar ezauth-admin como `http.Handler` no roteador existente, sem serviço ou deployment separado.
- [ ] Confirmar que o painel é servido no mesmo processo e não faz requisições a serviços de terceiros para carregar seus assets.
- [ ] Garantir que todas as rotas administrativas exijam sessão autenticada e papel `admin`; usuários comuns devem receber `403`.
- [ ] Configurar o RBAC do ezauth como fonte de autorização para os papéis deste checkpoint. Usar somente `admin` e `member`/usuário comum; não criar organizações ou multi-tenancy da biblioteca.
- [ ] Verificar que handlers e métodos de serviço administrativos chamados diretamente pelo código do openJob sejam sempre protegidos por autorização explícita. Não presumir que métodos de serviço façam a checagem por conta própria.
- [ ] Testar proteção CSRF/same-origin nas operações do painel e nas rotas administrativas próprias do openJob.
- [ ] Avaliar e desabilitar ou remover a funcionalidade de impersonation do painel. Impersonation não faz parte do CP18 e não poderá ficar acessível por padrão. Se não for possível desabilitá-la com segurança, não expor o painel completo; utilizar ezauth como biblioteca e construir apenas a interface administrativa mínima necessária.
- [ ] Limitar o painel às ações aprovadas: listar/buscar usuários, convidar, ativar, suspender/reativar e consultar auditoria. Exclusão de conta deverá acionar o fluxo de privacidade e remoção de dados do openJob, não apenas apagar a identidade.
- [ ] Garantir que endpoints de registro público da biblioteca estejam desabilitados ou inacessíveis. Novas contas devem nascer pelo bootstrap ou por convite autorizado do administrador.
- [ ] Verificar o fluxo de convite sem SMTP: a aplicação deve conseguir apresentar/compartilhar link de ativação de uso único, com validade, sem exigir envio de e-mail. Não introduzir dependência de mailbox nem serviço SMTP neste checkpoint.
- [ ] Confirmar que o administrador inicial é criado por procedimento controlado, uma única vez, sem credencial padrão hardcoded ou bootstrap repetível.

**Aceite POC-05:** painel integrado e protegido, registro público bloqueado, ações administrativas mínimas disponíveis, impersonation indisponível e convites utilizáveis sem serviço de e-mail.

### 2.7 Decisão de adoção da POC

A POC somente será aprovada se todos os critérios eliminatórios abaixo forem satisfeitos:

- [ ] Compatibilidade com toolchain, build, binário e distribuição do openJob.
- [ ] SQLite único, migrations compatíveis e plano seguro de foreign keys/cascades.
- [ ] Identidade canônica sem duplicação de usuários.
- [ ] Sessões persistentes após restart, revogáveis e com cookies/CSRF adequados.
- [ ] Google OIDC restrito à identidade.
- [ ] RBAC e painel administrativo protegidos, com impersonation desabilitado.
- [ ] Bootstrap único e criação de usuários somente por convite/admin.
- [ ] Dependências, licença e vulnerabilidades avaliadas.
- [ ] Plano de retenção, exclusão e auditoria compatível com as necessidades do produto.

Se qualquer critério eliminatório falhar sem mitigação simples, documentar `no-go` para ezauth/ezauth-admin e utilizar o fallback conservador. O resultado deverá ser registrado em ADR antes da implementação do restante do CP18.

### 2.8 Identidade local-first e operação

- A instância continua local-first e poderá ser acessada por mais de uma pessoa na rede local.
- SQLite, migrations versionadas e AES-GCM/master key existentes serão reutilizados.
- Não implementar multi-tenant hospedado na nuvem.
- Login local continua sendo um caminho funcional, independentemente do Google OIDC.
- Não haverá cadastro público irrestrito após o bootstrap.
- Não introduzir segundo serviço, segundo banco ou infraestrutura externa para autenticação.
- A biblioteca selecionada deve permanecer encapsulada por uma interface/adaptador do backend do openJob, para evitar que contratos internos de ezauth contaminem handlers e regras de negócio.

### 2.9 Separação entre identidade, autorização e ownership

- **Identidade/autenticação:** a solução aprovada valida credenciais, identifica o usuário e fornece o contexto autenticado.
- **Autorização do produto:** o backend do openJob decide quais operações cada usuário pode executar, inclusive regras de administrador e usuário comum.
- **Ownership:** repositórios, queries, handlers, downloads, exports, eventos e workers aplicam o identificador canônico da sessão; nunca confiam em `user_id` enviado livremente pelo frontend.
- **Dados profissionais:** perfil, currículo-base, candidaturas, feedback, preferências, filtros, perfis de IA, tags, tarefas e notificações permanecem sob contratos e repositories do openJob.
- **Dados globais:** catálogo canônico de vagas deduplicadas, fontes e dados técnicos compartilháveis permanecem no escopo global da instalação, conforme arquitetura da Parte 1.
- **Relacionamentos usuário–vaga:** utilizar associações com ownership quando uma vaga global for salva, pontuada, ocultada, associada a candidatura ou apresentada no histórico de um usuário.
- **Fora deste checkpoint:** `email_accounts` e `email_messages`. Não adicionar `user_id`, não migrar essas tabelas para o modelo multiusuário, não incluí-las nos testes de ownership e não criar novos fluxos de leitura/sincronização de mailbox no CP18.

A auditoria deverá gerar matriz explícita que classifique cada tabela e relacionamento como global, por usuário, associação usuário–entidade ou fora do escopo deste checkpoint.

## 3. Modelo de dados e contratos

A estrutura final depende do resultado da POC. Se ezauth for aprovado, suas tabelas de identidade, tokens, sessões/refresh tokens, papéis e auditoria serão a fonte canônica para autenticação. O openJob não deverá duplicar essas entidades.

Entidades do domínio openJob esperadas:

- `user_profiles`: `user_id` referenciando a identidade canônica, headline, localização profissional opcional, experiência, skills e preferências profissionais editáveis.
- `user_resume_versions` (ou equivalente): currículo-base por usuário, conteúdo editável, origem, versão, data de importação e metadados; preservar versão anterior ao substituir.
- `user_invitations`: somente se o mecanismo aprovado não for integralmente fornecido pelo ezauth. Não duplicar convites se ezauth for a fonte oficial.
- Vínculo de identidade Google: utilizar a associação mantida pela solução de autenticação, evitando duplicar subject e metadados em outra tabela sem necessidade.
- LinkedIn: se a decisão for `go`, persistir apenas o estado e os dados estritamente necessários ao adaptador específico de importação; não criar tabela genérica de conexões OAuth.

Se a POC reprovar ezauth, o fallback deverá implementar entidades equivalentes para:

- usuários e credenciais;
- sessões persistidas;
- convites de uso único e validade limitada;
- associação Google OIDC;
- papéis mínimos e trilha de auditoria.

Independentemente da solução escolhida, o contrato do restante do openJob deverá abstrair o provedor de autenticação e expor pelo menos:

- identificador canônico do usuário autenticado;
- e-mail/identificador de login, quando disponível e necessário;
- estado da conta (`active`, `disabled` ou equivalente);
- papel mínimo (`admin`/`member`);
- origem da autenticação (`local`, `google_oidc` ou equivalente);
- operações de consultar sessão, revogar sessão, encerrar sessão e exigir autorização.

Nunca expor senha, hash, access token, refresh token, segredo OAuth ou conteúdo de convite nos contratos de perfil, onboarding ou APIs públicas.

## 4. Tarefas de implementação

### 4.1 Auditoria de dados e plano de migração

- [ ] Inventariar tabelas, stores, repositórios, handlers, workers, jobs e tarefas agendadas.
- [ ] Classificar cada entidade como global, por usuário, associação usuário–entidade ou fora do escopo.
- [ ] Identificar pontos de leitura/escrita que precisam receber contexto de usuário.
- [ ] Definir estratégia de migração single-user → multiusuário, considerando o ID canônico da solução de autenticação aprovada.
- [ ] Preparar backup, rollback e validação de integridade antes de executar migrations destrutivas ou de grande alcance.
- [ ] Documentar dados globais que continuarão compartilhados entre usuários da mesma instância.
- [ ] Excluir `email_accounts` e `email_messages` da migração de ownership do CP18; não adicionar `user_id` nem alterar essas tabelas.
- [ ] Auditar explicitamente constraints `ON DELETE CASCADE`/`SET NULL` e comportamento do SQLite antes de habilitar foreign keys globalmente.

### 4.2 Bootstrap e gestão mínima de contas

- [ ] Criar o primeiro administrador de forma controlada, uma única vez.
- [ ] Garantir que o bootstrap não possa criar administradores repetidamente nem reabrir cadastro público após inicialização.
- [ ] Implementar criação/convite de usuários adicionais por administrador, usando o fluxo nativo da biblioteca se aprovada ou mecanismo equivalente no fallback.
- [ ] Garantir token de convite de uso único, validade limitada, não exposição em logs e aceitação vinculada à conta correta.
- [ ] Permitir ativar conta, definir/alterar senha e desativar usuário.
- [ ] Impedir que conta desativada inicie novas sessões e revogar sessões existentes.
- [ ] Não exigir serviço SMTP ou autorização de mailbox. Quando não houver entrega transacional aprovada, permitir compartilhamento seguro do link de ativação pelo próprio administrador.
- [ ] Limitar permissões a administrador da instância e usuário comum; evitar sistema complexo de autorização.

### 4.3 Login local, sessão e segurança

- [ ] Implementar login local, logout, consulta da sessão atual, expiração e revogação por meio do adaptador da solução aprovada.
- [ ] Se ezauth for aprovado, configurar Argon2id, rate limiting e account lockout conforme os parâmetros validados na POC.
- [ ] Se o fallback for utilizado, implementar hash com Argon2id e parâmetros documentados, calibrados no ambiente de distribuição.
- [ ] Usar store persistente de sessão compatível com SQLite; nunca aceitar store em memória em produção.
- [ ] Configurar cookies `HttpOnly`, `Secure` quando HTTPS estiver habilitado e `SameSite` apropriado, validando os cabeçalhos reais.
- [ ] Proteger operações mutáveis contra CSRF, validar origem e tratar explicitamente requests sem `Origin`/Fetch Metadata.
- [ ] Rotacionar identificadores/tokens após autenticação e mudança de privilégio, de acordo com o mecanismo adotado.
- [ ] Aplicar limite/backoff para tentativas de login e respostas genéricas que não revelem se uma conta existe.
- [ ] Revogar sessões no logout, desativação de conta, alteração de credencial e eventos de segurança que exijam invalidação.
- [ ] Garantir `401` sem sessão válida e `403` para falta de autorização.
- [ ] Não armazenar senha, token de sessão, refresh token ou segredo em texto puro.

### 4.4 Autorização e painel administrativo

- [ ] Implementar os papéis `admin` e `member` como conjunto mínimo.
- [ ] Se ezauth for aprovado, usar RBAC persistido para autorização administrativa; não depender apenas de campo legado de roles sem enforcement.
- [ ] Montar ezauth-admin somente se POC-05 aprovar a configuração e a possibilidade de remover/desabilitar impersonation.
- [ ] Proteger todas as operações administrativas, inclusive quando serviços forem chamados diretamente por handlers próprios.
- [ ] Disponibilizar listagem, busca, convite, ativação, suspensão/reativação e consulta de auditoria.
- [ ] Bloquear cadastro público e operações administrativas para usuários comuns.
- [ ] Não implementar impersonation, organizações, multi-tenancy, API keys administrativas ou permissões granulares neste checkpoint.
- [ ] Se ezauth-admin não permitir o recorte seguro, manter ezauth como biblioteca e criar apenas telas administrativas próprias estritamente necessárias.

### 4.5 Isolamento e migração de ownership

- [ ] Aplicar ownership nas queries e repositórios de todas as entidades pessoais/operacionais.
- [ ] Revisar handlers REST, rotas de exportação, downloads, WebSocket/eventos se existentes e workers em background.
- [ ] Garantir que tarefas assíncronas preservem o usuário proprietário e não executem com contexto implícito compartilhado.
- [ ] Migrar dados existentes para o administrador inicial sem perda, preservando IDs quando possível ou utilizando mapeamento de migração auditável.
- [ ] Atribuir relacionamentos pessoais ao administrador inicial; manter o catálogo global de vagas e fontes conforme a arquitetura existente.
- [ ] Criar constraints e índices por usuário onde necessários.
- [ ] Garantir que manipular IDs no frontend/API não permita ler, alterar, excluir, exportar ou associar dados de outro usuário.
- [ ] Testar especialmente `curriculum`, `applications`, `settings`, `llm_profiles`, `tags`, `tasks` e `notifications`.
- [ ] Não incluir `email_accounts` ou `email_messages` na matriz de ownership, migrations ou testes de isolamento deste checkpoint.

### 4.6 Perfil profissional e currículo-base

- [ ] Implementar CRUD de perfil profissional: nome, headline, localização profissional opcional, experiência, skills e preferências relevantes.
- [ ] Manter currículo-base individual, editável e versionado por usuário.
- [ ] Permitir edição/colagem de texto e Markdown.
- [ ] Reutilizar somente os formatos e parsers efetivamente suportados pelo produto; não ampliar a matriz de formatos neste checkpoint.
- [ ] Apresentar prévia do conteúdo importado antes de salvar/substituir.
- [ ] Exigir confirmação explícita para substituir currículo-base.
- [ ] Preservar versão anterior e registrar origem/data da atualização.
- [ ] Expor seleção do currículo-base ativo se mais de uma versão for mantida.
- [ ] Não implementar extração, classificação e inteligência documental genéricas; essa camada pertence ao CP20.

### 4.7 Google OIDC e importação profissional do LinkedIn

- [ ] Se ezauth for aprovado, habilitar exclusivamente Google OIDC e confirmar os escopos efetivos; não habilitar outros provedores disponíveis na biblioteca.
- [ ] Se o fallback for utilizado, implementar Google OIDC com `coreos/go-oidc` e as proteções correspondentes.
- [ ] Limitar o consentimento Google a identidade (`openid`, `email`, `profile`, conforme fluxo validado); não solicitar Gmail, Drive ou outros escopos.
- [ ] Associar a identidade Google à conta local com validação de issuer, audience, assinatura, expiração, state, nonce quando aplicável, redirect URI e PKCE quando suportado.
- [ ] Evitar takeover por coincidência de e-mail: vinculação de conta deve exigir identidade verificada e regra explícita, sem duplicar usuário existente.
- [ ] Não manter tokens Google quando não forem necessários após o login; quando a persistência for indispensável, justificar e cifrar.
- [ ] Fazer spike de LinkedIn e documentar APIs oficiais, aprovação/Partner Program, escopos e campos realmente acessíveis.
- [ ] Se houver decisão `go`, implementar adaptador backend específico para importar somente campos profissionais aprovados; a experiência guiada será chamada pelo CP19.
- [ ] Cifrar tokens LinkedIn que precisem ser mantidos com AES-GCM/master key existente; não persistir tokens se o fluxo não exigir.
- [ ] Mostrar os campos que serão importados e exigir revisão/confirmação antes de atualizar perfil ou currículo.
- [ ] Se LinkedIn não for viável, registrar `no-go`, não criar conexão e manter upload/colagem como alternativas; isso não bloqueia o CP19.
- [ ] Não implementar infraestrutura OAuth genérica, conexão de mailbox ou OAuth de provedores LLM.

### 4.8 APIs e interface mínima

As rotas exatas dependerão do resultado da POC e deverão seguir os padrões atuais do projeto. O backend deverá oferecer contratos estáveis para:

- autenticação local, logout e consulta da identidade atual;
- ativação/aceitação de convite e gestão administrativa mínima;
- listagem/revogação de sessões quando suportado pela solução;
- `GET/PUT /api/profile`;
- `GET/PUT /api/curriculum/base` e importação nos formatos já suportados;
- início/callback/status de Google OIDC, se habilitado;
- início/callback/status/desconexão do LinkedIn somente se o spike resultar em `go`.

UI mínima deste checkpoint:

- tela de login e ativação de conta;
- estado de sessão e logout;
- formulário de perfil e currículo-base;
- gestão mínima de usuários para administrador;
- consulta/revogação de sessões conforme suporte;
- estado do login Google e, somente se aprovado, conexão/importação LinkedIn e revisão dos dados profissionais.

Se ezauth-admin for aprovado, reaproveitar seu painel para as operações administrativas cobertas pela POC, sem duplicar telas. Não implementar wizard, sequência de etapas ou design completo do onboarding; isso pertence ao CP19.

## 5. Segurança, privacidade e LGPD

A adoção de ezauth ou de qualquer biblioteca não representa, por si só, conformidade com a LGPD. O openJob continua responsável por definir finalidades, bases legais, transparência, retenção, atendimento aos direitos dos titulares e medidas técnicas/administrativas adequadas.

### 5.1 Minimização e finalidade

- [ ] Armazenar somente atributos necessários à autenticação e ao funcionamento da conta.
- [ ] Não preencher telefone, avatar, localização, fuso horário ou metadados de perfil se o produto não precisar deles.
- [ ] Não habilitar magic links, SMS OTP, passkeys, MFA, API keys, organizações, impersonation ou outros módulos só porque a biblioteca os oferece; cada recurso adicional exige decisão própria.
- [ ] Separar claramente dados de autenticação, dados profissionais e dados de comportamento.
- [ ] Não enviar documentos, currículo, eventos pessoais ou preferências a serviços externos sem decisão e autorização explícitas.
- [ ] Não registrar senhas, tokens, códigos de autorização, convites, currículos ou conteúdo de documentos em logs.

### 5.2 Direitos, retenção e exclusão

- [ ] Definir como o usuário consulta e exporta seus dados pessoais e profissionais em formato utilizável.
- [ ] Definir processo de correção dos dados de conta e perfil.
- [ ] Definir exclusão de conta e tratamento de dados associados: currículo e versões, candidaturas, preferências, feedback, tarefas e notificações.
- [ ] Definir quando a exclusão deverá ser efetiva, anonimizada ou limitada por obrigação legal ou outra hipótese aplicável, com decisão documentada.
- [ ] Definir retenção de sessões, convites expirados, tokens revogados e registros de auditoria.
- [ ] Auditar se logs de auditoria sobrevivem à exclusão da conta no mecanismo escolhido. Se forem preservados, minimizar os dados, remover vínculo identificável quando aplicável e documentar finalidade e prazo de retenção.
- [ ] Garantir que exclusão de usuário não execute cascades inesperadas em dados globais ou em registros que precisem ser preservados conforme regra aprovada.
- [ ] Documentar política de backup e prazo de expiração/eliminação de cópias, respeitando recuperação operacional e direitos aplicáveis.

### 5.3 Segurança operacional e incidentes

- [ ] Manter arquivo SQLite e backups com permissões restritas ao processo/operador autorizado.
- [ ] Proteger master key e segredos OIDC fora do repositório e dos logs.
- [ ] Documentar atualização de dependências, análise de vulnerabilidades e rollback.
- [ ] Definir procedimento para identificar, conter, registrar e tratar incidentes de segurança envolvendo dados pessoais.
- [ ] Avaliar necessidade de comunicação à ANPD e aos titulares conforme legislação e regulamentação vigentes; não hardcode prazos sem conferir o enquadramento aplicável.
- [ ] Registrar as medidas de segurança e decisões de privacidade em documentação técnica/operacional.

## 6. Critérios de verificação e aceite

### 6.1 Gate tecnológico

- [ ] POC executada e decisão `go`/`no-go` registrada em ADR.
- [ ] Se `go`, versões exatas de ezauth/ezauth-admin fixadas e compatíveis com toolchain/build/distribuição.
- [ ] Se `no-go`, fallback conservador definido e testado sem duplicar responsabilidades.
- [ ] Nenhum serviço externo, banco ou store em memória foi introduzido para autenticação.

### 6.2 Gate de identidade, contas e sessões

- [ ] Primeiro administrador é criado uma única vez por mecanismo controlado.
- [ ] Cadastro público irrestrito permanece desabilitado.
- [ ] Administrador consegue convidar/ativar pelo menos um segundo usuário sem depender de SMTP.
- [ ] Dois usuários autenticam e utilizam a mesma instância simultaneamente.
- [ ] Login local, logout, expiração, renovação, revogação e bloqueio de conta desativada funcionam.
- [ ] Sessões persistem após restart e podem ser revogadas no servidor.
- [ ] Senhas são armazenadas com Argon2id ou mecanismo aprovado e tokens não são persistidos em claro.
- [ ] CSRF, Origin, cookies e respostas `401`/`403` passam nos testes negativos.

### 6.3 Gate de autorização e ownership

- [ ] Painel/rotas administrativas são acessíveis somente por `admin`; usuário comum recebe `403`.
- [ ] Impersonation e funcionalidades administrativas fora do escopo estão indisponíveis.
- [ ] Nenhum usuário consegue consultar, alterar, excluir, exportar ou associar dados privados de outro, inclusive manipulando IDs e requests.
- [ ] Dados globais de vagas permanecem íntegros e relacionamentos pessoais ficam isolados.
- [ ] Workers, jobs, downloads e exports respeitam o contexto do usuário proprietário.
- [ ] Banco single-user migra para multiusuário sem perda; banco novo sobe corretamente.
- [ ] `email_accounts` e `email_messages` não recebem ownership, não são migradas pelo CP18 e não entram nos testes de isolamento deste checkpoint.

### 6.4 Gate de perfil e integrações

- [ ] Perfil e currículo-base são persistidos por usuário.
- [ ] Substituir currículo exige confirmação e preserva versão anterior.
- [ ] Google OIDC, quando habilitado, utiliza exclusivamente escopos de identidade; nenhum escopo Gmail é solicitado e nenhuma API de mailbox é acessada.
- [ ] E-mail poderá ser identificador normalizado da conta, quando aplicável, sem conceder acesso à caixa postal.
- [ ] LinkedIn é implementado somente com API/escopos oficialmente aprovados; `no-go` documentado não bloqueia CP18 nem CP19.
- [ ] Nenhuma infraestrutura OAuth genérica ou conexão OAuth de provedor LLM é criada.

### 6.5 Gate de privacidade e qualidade

- [ ] Exportação, correção e exclusão/anonimização seguem política documentada.
- [ ] Retenção de sessões, convites, tokens e auditoria está definida.
- [ ] Segredos e dados pessoais desnecessários não aparecem em logs.
- [ ] `make test`, `make vet`, `make test-ui` e testes de migrations/ownership passam.
- [ ] Testes de migrations cobrem banco novo, upgrade, rollback e comportamento de foreign keys.
- [ ] Testes negativos cobrem autenticação, autorização, sessão, OIDC, convites e isolamento multiusuário.

## 7. Entregáveis

- POC comparativa/decisória e ADR `go`/`no-go` para ezauth/ezauth-admin.
- Biblioteca de autenticação integrada ou fallback conservador implementado, encapsulado por adaptador do openJob.
- Identidade canônica e modelo de ownership documentados e versionados.
- Login local seguro, bootstrap único do administrador e criação autorizada de usuários.
- Sessões persistentes em SQLite, com renovação/expiração/revogação testadas.
- Painel administrativo mínimo integrado, se ezauth-admin passar na POC; caso contrário, interface própria mínima.
- Migração single-user → multi-user com backup/rollback documentados.
- Perfil profissional e currículo-base por usuário.
- Google OIDC, se habilitado, exclusivamente para identidade.
- Spike/decisão de viabilidade do LinkedIn e, somente se viável, adaptador específico consumível pelo CP19.
- Política técnica de privacidade, retenção, exportação e exclusão para dados sob responsabilidade do CP18.
- Testes negativos de isolamento, autenticação, autorização, sessão e migração.
- Documentação de operação e segurança para uma instância local compartilhada na LAN.

## 8. Registrar

- [ ] ADR de seleção tecnológica: ezauth/ezauth-admin ou fallback conservador, incluindo resultado da POC, versões, licença, dependências e limitações.
- [ ] ADR de identidade canônica, login local, sessões persistentes e autorização.
- [ ] ADR de classificação de dados globais versus dados por usuário.
- [ ] ADR de integração SQLite, migrations e política de foreign keys/cascades.
- [ ] ADR de uso do Google OIDC exclusivamente para identidade e limites de escopo.
- [ ] Registro do spike LinkedIn com APIs, escopos/campos acessíveis e decisão `go`/`no-go`.
- [ ] Matriz de ownership por tabela, com testes associados.
- [ ] Plano de backup/rollback da migração single-user → multi-user.
- [ ] Política de retenção, exportação, correção, exclusão/anonimização e auditoria.
- [ ] Procedimento de resposta a incidentes e atualização de dependências.

## 9. Dependências e integração com os próximos checkpoints

- **CP19 — Onboarding web:** depende da identidade, autenticação, sessão, perfil e currículo-base definidos aqui. Poderá apresentar login Google como método de identidade e importação LinkedIn somente se o adaptador específico estiver aprovado. Não deverá implementar nem depender de autorização de mailbox.
- **CP20 — Inteligência de documentos:** utilizará o currículo-base e os limites de ownership estabelecidos aqui. Documentos e resultados de extração deverão permanecer isolados pelo usuário autenticado.
- **CP21 — Tailoring determinístico:** consumirá perfil/currículo do usuário, com ownership aplicado e proveniência preservada.
- **CP22 — Score Fit: timeline e captura comportamental:** dependerá de identidade canônica e isolamento para persistir sessões, impressões e eventos individuais.
- **CP23 — Score Fit: personalização e avaliação:** dependerá do ownership e da separação dos sinais comportamentais por usuário.
- **CP11 — distribuição:** deve continuar fornecendo o bootstrap técnico e os artefatos de instalação; a implementação escolhida no CP18 não poderá quebrar o binário único e os ambientes de distribuição aprovados.

## 10. Fora de escopo

- Onboarding gráfico e wizard de primeira execução (CP19).
- Instalação, distribuição, bootstrap técnico do executável, host/porta e abertura automática do navegador (CP11).
- Multi-tenant hospedado em nuvem e organizações da biblioteca de autenticação.
- Construir uma infraestrutura OAuth genérica para múltiplos provedores/finalidades.
- Scraping de LinkedIn, Easy Apply, mensagens/candidaturas automatizadas ou importação não autorizada.
- Parser genérico multimodal, OCR e Inteligência de documentos completa (CP20).
- Tailoring determinístico, Evidence Mapping e geração/validação de currículo adaptado (CP21).
- Personalização comportamental e re-ranking do Score Fit — Parte 2 (CP22/CP23).
- OAuth para provedores de LLM.
- Autorização de leitura de mailbox, Gmail API, escopos de leitura de mensagens, sincronização de e-mail e consentimento incremental para acesso à caixa postal.
- Ownership, migração ou testes de isolamento de `email_accounts` e `email_messages`.
- Impersonation administrativa, passkeys, MFA, SMS OTP, magic links, API keys de usuário, organizações e permissões complexas, salvo decisão posterior formal e fora deste escopo.
- Automação de classificação de e-mails ou atualização de candidaturas/pipeline a partir da leitura de mensagens; permanece adiada, aguardando POC verificável, e não é dependência do CP18 ou CP19.

**Gate final do CP18:** uma decisão tecnológica `go`/`no-go` documentada; duas ou mais contas utilizam uma única instância local com isolamento comprovado; a migração é concluída sem perda; perfil/currículo-base individuais funcionam; sessões persistem e podem ser revogadas; painel e operações administrativas são protegidos; Google OIDC, quando habilitado, limita-se à identidade; e LinkedIn é integrado somente se oficialmente viável e aprovado. O checkpoint não solicita nem utiliza autorização de mailbox e não migra `email_accounts`/`email_messages`. CP19 poderá construir o onboarding sobre contratos estáveis, sem reimplementar identidade ou importação profissional.

---

## Referências técnicas consultadas para a POC

As referências abaixo descrevem capacidades e requisitos documentados dos projetos na data desta revisão. Devem ser novamente conferidas quando as versões candidatas forem fixadas; documentação do fornecedor não substitui os testes de aceitação deste checkpoint.

- EzAuth — repositório e documentação principal: https://github.com/josuebrunel/ezauth
- EzAuth — sessões, middleware, CSRF e store SCS: https://josuebrunel.github.io/ezauth/guides/sessions-middleware-helpers/
- EzAuth — segurança de contas, tokens, sessões e revogação: https://josuebrunel.github.io/ezauth/guides/account-security/
- EzAuth — painel administrativo: https://josuebrunel.github.io/ezauth/guides/admin-panel/
- EzAuth Admin — documentação do projeto: https://github.com/josuebrunel/ezauth-admin
- SCS — sessões HTTP e stores: https://github.com/alexedwards/scs
- CoreOS go-oidc: https://github.com/coreos/go-oidc
- Go x/crypto Argon2: https://pkg.go.dev/golang.org/x/crypto/argon2
- LGPD — Lei nº 13.709/2018: https://www.planalto.gov.br/ccivil_03/_ato2015-2018/2018/lei/l13709.htm
- ANPD — Guia orientativo de segurança da informação para agentes de tratamento de pequeno porte: https://www.gov.br/anpd/pt-br/documentos-e-publicacoes/guia-vf.pdf