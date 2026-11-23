# Checkpoint 20 — Instalador, bootstrap e release final (3–5 dias)

> Origem: evolução da experiência de distribuição do CP11 + onboarding CP19. Pré-requisitos: CP18 e CP19 concluídos, CP11 (binário único/wizard TUI/GoReleaser) preservado como base, e fases de release do release-mvp.md concluídas ou com decisão go/no-go registrada.

## 1. Objetivo

Fechar a distribuição para usuário final com caminho simples de instalação e primeira abertura, incluindo pacote .deb quando suportado pelo alvo de release, inicialização da webapp e instruções claras de acesso local e pela rede.

O CP20 não reimplementa o binário único nem o onboarding web: integra e finaliza a jornada:
**instalar → executar bootstrap mínimo → abrir navegador → concluir CP19 → chegar à página de vagas**.

## 2. Escopo do bootstrap

- Manter o CP11 single-binary como artefato/base de runtime; Docker permanece opção avançada/dev.
- Bootstrap do executável limitado ao necessário para iniciar a webapp:
  - escolher/confirmar diretório de dados e porta somente se não houver configuração válida;
  - criar/validar configuração e master key com permissões restritas;
  - iniciar servidor e aguardar health check;
  - exibir URL local e URL LAN real (IP/interface detectados), além de fallback quando hostname local não resolver.
- Tela final informa como acessar pelo computador/celular na mesma rede e apresenta ação **“Abrir no navegador”** que lança o navegador padrão.
- Não configurar perfil, currículo, fontes, filtros ou IA no TUI: esses passos pertencem ao CP19.
- Em execução headless/SSH, não tentar abrir GUI; imprimir URLs e instruções no terminal.

## 3. Empacotamento e distribuição

- Definir alvo inicial suportado para pacote .deb (arquitetura conforme CI/ambiente do projeto); validar dependências e instalação limpa.
- Integrar geração do .deb ao pipeline GoReleaser/CI existente ou ferramenta compatível já adotada; evitar dois pipelines de release.
- Instalar executável, entrada de desktop (quando aplicável), ícone e documentação mínima; não instalar como serviço privilegiado por padrão.
- Definir diretórios de configuração/dados/logs conforme padrões Linux, permissões e ownership; preservar dados em upgrade/uninstall.
- Documentar instalação, atualização, remoção, backup/restore e recuperação da master key.
- Gerar checksums e manter release reproduzível; assinatura/notarização somente se estiver no escopo aprovado.

## 4. Segurança e rede

- Servidor não deve expor-se à internet por padrão; documentar que acesso LAN depende de firewall e rede confiável.
- Nunca executar sudo automaticamente nem abrir portas/firewall sem consentimento explícito.
- Não colocar segredos em argumentos de processo, logs, arquivos world-readable ou URLs.
- Validar abertura de browser sem interpolação insegura de comandos; usar API de processo apropriada.
- Exibir aviso de firewall e requisitos de mesma rede; Android deve receber URL por IP/nip.io quando hostname local não resolver.

## 5. Critérios de verificação

- Instalação limpa do .deb em distro/versão suportada cria arquivos nos locais corretos e inicia openJob.
- Bootstrap em máquina com desktop inicia servidor, aguarda health e abre navegador padrão na URL correta.
- Bootstrap em ambiente sem GUI/SSH não falha: imprime URLs e instruções.
- Celular na mesma LAN acessa URL anunciada; URL não é inventada nem baseada em interface VPN/link-local.
- Reinstalação/upgrade preserva banco, usuários, currículos e master key; remoção padrão preserva dados e explica como apagá-los manualmente.
- Instalação/desinstalação não exige Docker nem Go para usuário final.
- Testar banco novo e upgrade, make test, make vet, make test-ui e smoke test do pacote.
- README e notas de release descrevem caminho recomendado e caminho Docker avançado.

## 6. Registrar

- ADR de estratégia do pacote .deb, diretórios e ciclo de vida de dados.
- Registrar decisão sobre bootstrap mínimo versus wizard TUI do CP11: CP11 continua válido como setup avançado/reconfiguração se necessário, mas a primeira instalação delega preparação do produto ao CP19.
- Checklist de release e teste de instalação limpa.

## 7. Fora de escopo

- Reescrever o CP11 single-binary/GoReleaser.
- Reimplementar onboarding web (CP19) ou autenticação/perfil (CP18).
- Auto-update, pacote RPM/MSI/DMG ou serviços systemd/launchd, salvo decisão explícita posterior.
- Automação de candidatura, scraping privado ou exposição pública do serviço.
