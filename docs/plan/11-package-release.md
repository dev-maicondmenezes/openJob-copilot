# Checkpoint 11 — Binário único, bootstrap, distribuição e CI/CD (7–9 dias)

> Origem: spec §13 (distribuição) + README (Quick start atual via Docker). Pré-requisito: CP1–CP10 concluídos. Este checkpoint entrega os canais oficiais de distribuição do openJob: binário Go, pacote Debian e imagem Docker publicada no Docker Hub.

## 1. Objetivo

Entregar o openJob como aplicação distribuível por:
- Binário Go único para execução local.
- Pacote .deb para Linux Debian/Ubuntu.
- Imagem Docker oficial publicada no catálogo público do Docker Hub.

Os modos de execução têm responsabilidades diferentes:

- **Binário/.deb:** bootstrap técnico mínimo, inicialização do servidor e abertura do navegador em ambiente gráfico.
- **Docker:** processo headless/servidor, sem wizard ou bootstrap interativo; toda configuração operacional é fornecida por variáveis de ambiente e volumes persistentes.
- **Ambos:** o onboarding e as configurações funcionais do produto acontecem na webapp (CP18/CP19), não no terminal nem no entrypoint do container.

Entregas:
- Servidor HTTP e SPA embutida via go:embed, conforme ADR 0003.
- Driver SQLite puro-Go (modernc.org/sqlite, ADR 0002), sem CGO.
- Binário único por plataforma, com versão e checksums.
- Pacote .deb para os alvos Linux Debian/Ubuntu homologados, integrado ao pipeline GoReleaser.
- Imagem Docker multi-arquitetura (linux/amd64 e linux/arm64, se homologadas), publicada no Docker Hub.
- Bootstrap mínimo apenas para execução direta do binário/.deb.
- Configuração Docker por env vars, com volume persistente para banco e master key.
- Health check, documentação operacional e processo de release reproduzível.

## 2. Limites de responsabilidade

### Pertence ao CP11

- Empacotamento do runtime e assets.
- Inicialização do servidor e preparação técnica mínima para binário/.deb.
- Escolha de porta/diretório apenas quando não houver configuração válida, com defaults seguros.
- Descoberta de IP LAN e abertura do navegador somente no modo desktop do binário.
- Criar ou validar a master key e o banco no modo binário/.deb, respeitando permissões restritas.
- Imagem Docker de produção, Dockerfile multi-stage, usuário não-root, health check e volume persistente.
- Configuração do container por variáveis de ambiente, sem setup interativo.
- Publicação da imagem no Docker Hub, com tags versionadas e metadados OCI.
- Distribuição por binário, .deb e imagem Docker; upgrade sem perda de dados e documentação.

### Não pertence ao CP11

- Cadastro/login, usuários, sessões, isolamento de dados ou perfis profissionais: CP18.
- Importação de currículo e conexões OAuth de perfil: CP18.
- Onboarding guiado de produto, currículo, fontes, filtros e IA: CP19.
- Coleta de API keys de LLM, configuração de palavras-chave/filtros, token/chat_id do Telegram ou testes dessas integrações no terminal.
- Wizard TUI para configurar funcionalidades do produto.
- Bootstrap interativo, abertura de navegador ou descoberta de IP LAN dentro do container.
- Auto-update, assinatura/notarização, RPM, MSI, DMG e execução como serviço systemd/launchd/Windows Service.

## 3. Pré-requisitos

- CP1–CP10 concluídos e estáveis.
- frontend produz backend/web/dist via make build.
- modernc.org/sqlite confirmado como driver; builds de release com CGO_ENABLED=0.
- Mecanismo AES-GCM do CP3 disponível para a master key e segredos persistidos pela aplicação.
- Contrato de diretório de dados e configuração compatível com o runtime atual, sem migração silenciosa de dados existentes.
- Nome definitivo da organização/repositório Docker Hub e credenciais de publicação configuradas como secrets no CI; nunca gravadas no repositório.

## 4. Comportamento do executável

### 4.1 Comandos

| Comando | Comportamento |
|---|---|
| openjob | Prepara bootstrap mínimo se necessário, inicia o servidor e tenta abrir a URL local no navegador. |
| openjob serve | Inicia o servidor sem tentar abrir navegador; adequado para SSH/headless e execução controlada. |
| openjob version | Imprime versão injetada no build via ldflags. |

Não haverá subcomando setup para reconfigurar IA, descoberta, currículo ou notificações. Essas configurações serão feitas na webapp.

### 4.2 Bootstrap mínimo — somente binário/.deb

- Usar defaults documentados para porta e diretório de dados; solicitar interação somente quando uma escolha técnica for indispensável e não houver default seguro.
- Não exigir arquivo .env para o caminho feliz.
- Criar diretório de dados e master key com permissões restritas; nunca sobrescrever chave ou banco existentes.
- Inicializar/migrar SQLite usando o mecanismo já existente.
- Iniciar o servidor e aguardar resposta positiva do health check antes de anunciar que está pronto.
- Não criar um segundo arquivo de configuração que replique settings persistidos no SQLite. Parâmetros de processo devem vir de flags/env vars, com precedência documentada.
- Se o bootstrap falhar, apresentar erro acionável e não deixar configuração parcial que impeça nova tentativa.

### 4.3 Rede e navegador — somente binário desktop

- Descobrir interfaces e endereços IPv4 privados, excluindo loopback, link-local e interfaces inadequadas, como VPN quando não forem rota de acesso LAN.
- Exibir URL local (localhost) e URL LAN real, identificando a interface correspondente quando houver mais de uma.
- Não prometer hostname mDNS quando não houver resolução confirmada; para Android, priorizar URL por IP quando .local não resolver.
- Abrir URL local no navegador padrão somente após health check positivo.
- Em SSH/headless, não tentar iniciar navegador; imprimir URLs e instruções.
- Não abrir portas no firewall, executar sudo ou expor o serviço à internet automaticamente.
- Avisar que o acesso por celular depende da mesma rede local e das regras de firewall.

## 5. Empacotamento e release

### 5.1 Binário único

- Manter SPA embutida via go:embed; em produção, o binário não depende de backend/web/dist no disco.
- Confirmar SQLite puro-Go e CGO_ENABLED=0.
- Gerar binários para linux/amd64, linux/arm64, darwin/amd64, darwin/arm64 e windows/amd64, desde que todos os alvos passem nos smoke tests.
- Injetar versão via ldflags a partir da tag Git.
- Anexar checksums SHA-256; assinatura fica fora deste checkpoint.
- Não exigir tarball; binário individual é o artefato principal.

### 5.2 Pacote Debian

- Gerar .deb para linux/amd64 e linux/arm64, se suportados pelo build e pelo ambiente de teste; caso contrário, declarar explicitamente o alvo inicial homologado.
- Integrar o pacote ao GoReleaser usando nfpm ou mecanismo equivalente já compatível com o pipeline.
- Instalar executável, metadados e atalho de desktop quando aplicável; não instalar serviço privilegiado nem iniciar servidor automaticamente durante post-install.
- Definir diretórios, ownership e permissões de configuração/dados/logs sem quebrar o contrato de dados já usado pelo binário.
- Upgrade deve preservar banco, usuários, currículos, configurações e master key.
- Remoção padrão deve preservar dados e explicar como removê-los manualmente.
- Não manter pipeline de release separado para o .deb.

### 5.3 Imagem Docker e publicação no Docker Hub

- Criar Dockerfile multi-stage: build da SPA e backend em estágio de compilação; runtime final mínimo, sem toolchain, Node ou arquivos de desenvolvimento.
- Executar como usuário não-root; definir WORKDIR, ENTRYPOINT/CMD, porta padrão 8080 e EXPOSE 8080.
- Não executar wizard, bootstrap interativo, abertura de navegador, detecção de IP LAN ou comandos privilegiados no container.
- Toda configuração operacional suportada no container deve ser injetável por env vars. Documentar nome, default, validação e precedência. No mínimo, cobrir porta (OPENJOB_PORT), caminho do banco (OPENJOB_DB_PATH) e hostname (OPENJOB_HOSTNAME), além das flags OPENJOB_FEATURE_* existentes.
- Persistir banco e master key no volume de dados (padrão /data); reiniciar/substituir container não pode perder dados ou gerar nova chave quando já existe uma.
- Adicionar HEALTHCHECK usando o endpoint /api/health, com intervalo, timeout, retries e start period adequados.
- Suportar linux/amd64 e linux/arm64 via buildx/manifests multi-platform quando ambos passarem nos testes; não publicar arquitetura sem validação.
- Publicar no Docker Hub no namespace/repositório oficial definido pelo projeto. O nome final da imagem deve ser único e documentado no README.
- Publicar tags imutáveis por versão (X.Y.Z e, se adotado, X.Y) e a tag latest somente para releases estáveis; não usar latest como única referência.
- Incluir labels OCI (org.opencontainers.image.source, revision, version, created, licenses, description) e manter provenance/SBOM se suportados pelo pipeline sem custo operacional desproporcional.
- Automatizar login/publicação no Docker Hub usando secrets; PRs e builds de branches não publicam no repositório público.
- Não embutir segredos, credenciais, arquivos .env ou dados pessoais na imagem/camadas.
- Manter Docker Compose como exemplo de execução local, usando a imagem publicada e volume nomeado para persistência.

### 5.4 CI/CD e publicação por tag Git

- Evoluir as GitHub Actions existentes para incorporar Continuous Delivery ao fluxo de CI, preservando a CI atual como gate obrigatório e evitando pipelines concorrentes ou publicação duplicada.
- Pull requests e pushes em branches continuam executando CI e nunca publicam artefatos. O workflow de release deve depender dos checks aplicáveis do projeto (frontend, backend, testes, vet e build), ampliando a cobertura da CI onde algum desses checks ainda não estiver automatizado.
- A publicação de uma nova versão será iniciada exclusivamente pelo push de uma nova tag estável no formato `vX.Y.Z` (por exemplo, `v1.2.3`). A tag é a única fonte da versão publicada; não haverá publicação manual ou disparada por push comum na branch principal.
- O workflow de release deve validar o formato da tag, obter o commit correspondente e executar os gates de CI antes de publicar qualquer artefato. Falha em validação/testes interrompe a distribuição.
- A partir da mesma tag, o pipeline deve executar o GoReleaser para compilar os binários, gerar checksums e empacotar os arquivos `.deb`, criando/publicando uma GitHub Release no próprio repositório e anexando todos os artefatos gerados.
- No mesmo fluxo de release, construir e publicar a imagem Docker multi-arquitetura no Docker Hub, com tag correspondente à versão Git (`X.Y.Z`) e `latest` somente para versões estáveis. O commit, a versão e os metadados OCI devem corresponder à GitHub Release.
- A publicação do `.deb`, da GitHub Release e da imagem Docker deve ser coordenada pelo mesmo evento/tag e usar a mesma versão, sem workflows independentes com versionamento próprio.
- Atualizar/configurar as actions necessárias para checkout com histórico/tags, setup de Go, GoReleaser, Docker Buildx e autenticação no Docker Hub. Fixar versões das actions utilizadas e revisar permissões mínimas por job.
- Permissões do job de release: `contents: write` para criar a GitHub Release. Credenciais do Docker Hub devem vir exclusivamente de GitHub Actions Secrets (por exemplo, `DOCKERHUB_USERNAME` e `DOCKERHUB_TOKEN`); nunca usar credenciais em texto claro.
- Configurar concorrência para impedir duas publicações simultâneas da mesma versão. O workflow deve falhar claramente se a tag já tiver sido publicada com artefatos divergentes; não sobrescrever releases/imagens de versão imutável.
- Tags de pré-release, se futuramente adotadas, não devem atualizar `latest`; por enquanto, somente tags estáveis `vX.Y.Z` fazem parte do fluxo oficial.
- Se uma etapa de publicação falhar depois de outra já ter publicado, o workflow deve expor qual artefato foi publicado e qual falhou, permitindo reexecução idempotente sem gerar versões inconsistentes.
- Não exigir PAT pessoal quando o `GITHUB_TOKEN` com `contents: write` for suficiente para a GitHub Release.

### 5.5 Documentação

- Atualizar Quick start com três caminhos: binário/.deb, Docker Hub via docker run e Docker Compose.
- Documentar instalação, execução, atualização, remoção, diretórios de dados, backup/restore e recuperação da master key.
- Documentar todas as env vars aceitas pela imagem, defaults, validação e exemplo de volume persistente.
- Documentar acesso local/LAN, firewall e execução headless.
- Deixar explícito que o primeiro uso funcional continua na webapp; o bootstrap não configura provedores nem integrações.
- Incluir nome oficial da imagem, tags, arquiteturas suportadas e política de publicação.

## 6. Critérios de verificação

- Executar binário em máquina sem Docker e sem Go; servidor sobe e health check responde.
- Em desktop, execução padrão abre navegador na URL local somente depois do health check.
- Em SSH/headless, openjob serve funciona e imprime URLs sem tentar abrir GUI.
- URL LAN exibida corresponde a interface/IP realmente utilizável; celular na mesma rede acessa a SPA quando firewall permite.
- Primeira execução do binário/.deb cria diretório, banco e master key com permissões restritas; reexecução preserva dados e não recria segredos.
- Instalação limpa do .deb em cada alvo homologado funciona sem Docker/Go; upgrade preserva banco e master key; remoção padrão preserva dados.
- Container inicia sem interação e sem depender de Docker Compose; health check passa.
- Container roda como não-root; banco e master key permanecem após remoção/recriação do container quando o volume é mantido.
- Env vars alteram apenas as configurações documentadas, com validação e precedência testadas; nenhuma configuração de produto é exigida no entrypoint.
- Imagem multi-arquitetura é testada em cada arquitetura publicada; endpoint /api/health funciona.
- Push de uma tag estável `vX.Y.Z` executa os gates de CI e inicia uma única publicação versionada.
- A tag gera binários, checksums e `.deb` via GoReleaser e publica uma GitHub Release no próprio repositório com os artefatos anexados.
- A mesma tag publica a imagem Docker no Docker Hub com versão idêntica e `latest` atualizado somente para release estável.
- Versão, commit e metadados dos artefatos da GitHub Release e da imagem Docker correspondem entre si.
- PRs e pushes comuns em branches executam CI, mas não publicam releases, pacotes ou imagens; credenciais não aparecem em logs nem artefatos.
- Reexecução após falha parcial não sobrescreve silenciosamente artefatos imutáveis nem cria versões divergentes.
- make test, make vet, make build e smoke tests dos artefatos passam antes da publicação.
- O fluxo de CD reutiliza as validações de CI e coordena binários, .deb, GitHub Release e imagem Docker sem versionamento ou publicação duplicados.

## 7. Registrar

- ADR 0011 — distribuição single-binary, GoReleaser e canais oficiais de distribuição.
- ADR 0012 — bootstrap mínimo para binário/.deb, execução headless e separação entre inicialização técnica e onboarding web.
- ADR para imagem Docker oficial: runtime, env vars, persistência, usuário não-root, health check, tags e publicação no Docker Hub.
- Documentar o contrato CI/CD: tag `vX.Y.Z` como gatilho único, jobs/actions utilizados, permissões, secrets necessários, artefatos da GitHub Release e tags Docker publicadas.
- Testes para bootstrap idempotente, permissões, health check, detecção de IP, execução headless, abertura de navegador, ciclo de vida do .deb e ciclo de vida do container.
- Riscos: firewall local, múltiplas interfaces/VPN, diferenças de diretórios entre instalação direta e .deb, persistência de master key, compatibilidade cross-platform, build multi-arquitetura, segurança das credenciais do Docker Hub e falha parcial durante publicação coordenada.

## 8. Fora de escopo

- Onboarding de produto e configurações de usuário (CP18/CP19).
- Auto-update, assinatura/notarização, RPM, MSI e DMG.
- Serviços systemd/launchd/Windows Service.
- Orquestração Kubernetes/Helm, Docker Swarm e publicação em registries adicionais.
- Automação de candidatura ou exposição pública do serviço.
