# ADR 0012 — Bootstrap mínimo (binário/.deb), execução headless e onboarding web

**Status:** aceito (Checkpoint 11)
**Data:** 2026-11-23
**Relaciona:** docs/plan/11-package-release.md, ADR 0004 (master key), ADR 0011, ADR 0043

## Contexto

Uma versão anterior do plano previa um wizard TUI que coletava chaves de API,
filtros e Telegram. O produto passou a ter onboarding na webapp (CP18/CP19),
então o terminal não deve duplicar nem ser a fonte de configuração funcional.

## Decisão

O executável faz **só preparação técnica**; tudo que é do produto acontece na
webapp.

| Comando | Comportamento |
|---|---|
| `openjob` | bootstrap → servidor → abre o navegador (só em sessão gráfica, só após o health check) |
| `openjob serve` | bootstrap → servidor, nunca abre navegador (SSH/headless) |
| `openjob version` | imprime a versão injetada |
| `openjob healthcheck` | sonda `/api/health` local; existe para o `HEALTHCHECK` do Docker |

`healthcheck` não estava na tabela do plano: a imagem final é distroless, sem
`curl`/`wget`, e o binário precisa se auto-sondar.

**Não existe `setup`.** Não há subcomando, prompt nem arquivo `.env` para
configurar IA, descoberta, currículo ou notificações.

### Bootstrap (`internal/bootstrap`)

- Cria o diretório de dados (0700 quando é criado agora; diretório existente,
  como um volume, mantém seu modo) e a master key (0600), de forma **atômica**
  (arquivo temporário + `link`): nunca sobrescreve e nunca deixa uma key
  truncada que bloqueie a próxima tentativa.
- Verifica escrita no diretório e devolve erro com a ação sugerida
  (`--db-path` / `OPENJOB_DB_PATH`).
- Restringe uma key com modo frouxo (ex.: restaurada de backup) e avisa.
- Migrações SQLite continuam no `store.Open` existente.
- **Sem segundo arquivo de configuração**: o diretório de dados só contém o
  banco (e arquivos WAL/SHM) e a key. Verificado pelo smoke test.

### Precedência e defaults

`flag > variável de ambiente > default`. Defaults diferem por contexto:

- **Fora de container** (binário/.deb): o diretório de dados é por usuário
  (`~/.local/share/openjob`, `~/Library/Application Support/openjob`,
  `%AppData%\openjob`) e, se a porta não foi escolhida e 8080 está ocupada,
  procura a próxima livre até 8099. Porta escolhida explicitamente e ocupada
  **falha** — nunca troca em silêncio.
- **Em container** (`OPENJOB_CONTAINER=true`, definido pela imagem — opt-in explícito, sem
  heurística como `/.dockerenv`, para o binário/.deb continuar com os defaults
  por usuário em devcontainers e distrobox): nada é adivinhado; porta e caminhos vêm só do ambiente, sem
  navegador e sem descoberta de LAN.

Um `OPENJOB_DB_PATH` já existente no ambiente nunca é substituído: não há
migração silenciosa de dados.

### Rede e navegador

- O listener é aberto uma vez e mantido (sem corrida entre checar e abrir).
- "Pronto" só é anunciado depois de `/api/health` responder 200.
- URLs LAN: IPv4 privado, interface ativa, sem loopback, link-local, túneis nem
  bridges de Docker/VM; mostra a interface de cada IP.
- O hostname amigável (`openjob.copilot`) só é impresso se resolver, aqui, para
  um IP desta máquina. Android/Chrome não resolve `.local`: o aviso orienta
  usar o IP.
- Não abre porta de firewall, não usa `sudo`, não expõe à internet. Avisa que o
  acesso por celular depende da mesma rede e do firewall.
- `IsHeadless`: sessão SSH, ou Linux sem `DISPLAY`/`WAYLAND_DISPLAY`.

## Consequências

- O mesmo `serve` serve a uma sessão SSH num notebook e ao container; a
  diferença é só o contexto detectado.
- Quem já usava `make run` com `./data/` continua funcionando: o `OPENJOB_DB_PATH`
  explícito prevalece.
- Windows: permissões POSIX não se aplicam; o ACL do diretório de dados do
  usuário é o protetor (limitação documentada).
