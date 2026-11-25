# Distribuição e operação do openJob

Três caminhos oficiais, todos gerados da mesma tag `vX.Y.Z`
([contrato de release](release/ci-cd.md)). Em todos eles, **o primeiro uso
funcional (IA, currículo, fontes, Telegram) acontece na webapp** — o
bootstrap/instalador não configura provedores nem integrações.

| Caminho | Quando usar | Dados ficam em |
|---|---|---|
| [Binário](#1-binário) | qualquer máquina, sem Docker/Go | diretório do usuário |
| [`.deb`](#2-pacote-deb) | Debian/Ubuntu | diretório do usuário |
| [Docker](#3-docker) | servidor/headless, isolamento | volume `/data` |

## 1. Binário

Baixe o arquivo da sua plataforma na página de Releases (`openjob_X.Y.Z_<os>_<arch>`,
`.exe` no Windows) e o `checksums.txt`.

```bash
sha256sum -c --ignore-missing checksums.txt   # confira a integridade
chmod +x openjob_X.Y.Z_linux_amd64 && mv openjob_X.Y.Z_linux_amd64 ~/.local/bin/openjob
openjob            # prepara, sobe o servidor e abre o navegador
openjob serve      # só sobe o servidor (SSH/headless)
openjob version
```

macOS: o binário não é assinado nem notarizado (fora do escopo); na primeira
execução pode ser necessário liberá-lo em *Ajustes → Privacidade e Segurança*.

**Atualizar:** substitua o arquivo pelo da nova versão. Dados e master key são
preservados (ficam fora do binário). **Remover:** apague o binário; para apagar
também os dados, veja [Dados](#dados-backup-e-master-key).

## 2. Pacote `.deb`

```bash
sudo apt install ./openjob_X.Y.Z_amd64.deb     # ou _arm64.deb
openjob                                         # ou procure "openJob" no menu
```

Instala apenas `/usr/bin/openjob` e um atalho de desktop. **Não cria serviço,
não inicia servidor e não toca em dados.** Os dados são por usuário.

- **Atualizar:** `sudo apt install ./openjob_NOVA.deb` — banco e master key preservados.
- **Remover:** `sudo apt remove openjob` (e `purge` também) **preserva seus
  dados**. Para apagá-los: `rm -r ~/.local/share/openjob`.

## 3. Docker

Imagem oficial: **`maicondmenezes/openjob`** no Docker Hub *(confirme o nome
final antes do primeiro release — ver checklist em `release/ci-cd.md`)*.

- Tags: `X.Y.Z` (imutável) e `latest` (só versões estáveis). Para produção,
  fixe `X.Y.Z`.
- Arquiteturas: `linux/amd64`, `linux/arm64`.
- Política: só tags `vX.Y.Z` publicam; PRs e branches nunca.

```bash
docker run -d --name openjob -p 8080:8080 -v openjob-data:/data \
  maicondmenezes/openjob:X.Y.Z
```

ou Docker Compose (imagem publicada + volume nomeado `openjob-data`):

```bash
docker compose up -d
```

Para construir do código-fonte (contribuidores), com dados em `./data`:

```bash
mkdir -p data && sudo chown 65532:65532 data    # a imagem roda como não-root
docker compose -f docker-compose.yml -f docker-compose.build.yml up -d --build
```

- **Atualizar:** `docker pull …:NOVA` e recrie o container com o **mesmo
  volume**; dados e master key permanecem.
- **Remover:** `docker rm -f openjob`; o volume continua. Para apagar os dados:
  `docker volume rm openjob-data`.
- O container roda **sem interação**, como usuário não-root, e tem
  `HEALTHCHECK` (`/api/health`).

### Variáveis de ambiente

Precedência: **flag > variável de ambiente > default**. Valor inválido encerra
o processo com código 2 e uma mensagem explicando o que corrigir.

| Variável | Flag | Default (imagem) | Default (binário) | Validação |
|---|---|---|---|---|
| `OPENJOB_PORT` | `--port` | `8080` | `8080`, ou a próxima livre até 8099 | 1–65535 |
| `OPENJOB_DB_PATH` | `--db-path` | `/data/openjob.db` | `<diretório do usuário>/openjob.db` | não vazio |
| `OPENJOB_HOSTNAME` | `--hostname` | `openjob.copilot` | `openjob.copilot` | nome DNS simples |
| `OPENJOB_FEATURE_UI_FEED` | — | `true` | `true` | `false`/`0` desliga; outro valor mantém o default |
| `OPENJOB_FEATURE_EMAIL_SYNC` | — | `true` | `true` | idem |
| `OPENJOB_FEATURE_CURRICULUM_EXPORT` | — | `true` | `true` | idem |
| `OPENJOB_FEATURE_SCORE_DETERMINISTIC` | — | `true` | `true` | idem (master switch do B11) |
| `OPENJOB_FEATURE_KEYWORDS_DETERMINISTIC` | — | `true` | `true` | idem; exige o master switch |
| `OPENJOB_FEATURE_FILTER_DETERMINISTIC` | — | `true` | `true` | idem; exige o master switch |

`OPENJOB_CONTAINER=true` é um marcador interno da imagem (desliga os defaults
por usuário, o navegador e a descoberta de LAN); não é configuração de produto.
Porta escolhida explicitamente e ocupada **falha** em vez de trocar sozinha.

## Dados, backup e master key

| Instalação | Diretório |
|---|---|
| Linux (binário/.deb) | `$XDG_DATA_HOME/openjob` ou `~/.local/share/openjob` |
| macOS | `~/Library/Application Support/openjob` |
| Windows | `%AppData%\openjob` |
| Docker | volume montado em `/data` |

Contém `openjob.db` e **`.master.key`** (AES-GCM, cifra suas chaves de API).
Diretório `700`, key `600` (POSIX).

- **Backup:** pare o app e copie **o banco e a key juntos**. Nunca versione ou
  compartilhe a key.
- **Restore:** devolva os dois ao diretório de dados. Uma key restaurada com
  permissão frouxa é restringida a `600` automaticamente (com aviso).
- **Key perdida:** as chaves de API cifradas ficam irrecuperáveis (erro
  explícito); basta cadastrá-las de novo em `/settings`. O app **nunca**
  gera uma key nova por cima de uma existente — nem se o arquivo estiver
  truncado (nesse caso ele para e explica).
- **Migrar de `./data` (Compose antigo) para volume:** copie `openjob.db` e
  `.master.key` para o volume; não há migração automática.

## Acesso local, LAN, firewall e headless

Ao subir, o app imprime as URLs **reais**: `localhost` e o IP de LAN de cada
interface utilizável (sem loopback, VPN ou bridges de Docker). O nome
`openjob.copilot` só é mostrado se resolver para esta máquina.

- **Celular:** precisa estar na mesma rede e o firewall deve permitir a porta
  (o app não altera o firewall). Android/Chrome não resolve `.local`: use a URL
  por IP.
- **SSH/headless:** `openjob serve`; nenhuma tentativa de abrir navegador —
  copie a URL impressa. Em SSH, o `openjob` sem subcomando também não abre
  navegador.
- **Docker:** o container não sabe o IP do host; acesse pela porta publicada
  (`http://<ip-do-host>:8080`).
- Nada é exposto à internet automaticamente.
