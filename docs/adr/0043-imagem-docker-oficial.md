# ADR 0043 — Imagem Docker oficial: runtime, env vars, persistência e publicação

**Status:** aceito (Checkpoint 11)
**Data:** 2026-11-23
**Relaciona:** docs/plan/11-package-release.md, ADR 0011, ADR 0012, ADR 0004

> Número 0043: o plano reserva 0011 e 0012 e não fixa um para este ADR; 0043 é
> o próximo livre após 0042.

## Decisão

### Runtime

- Dockerfile multi-stage. Os estágios de build (Node para a SPA, Go para o
  binário) rodam na arquitetura **nativa do builder** (`--platform=$BUILDPLATFORM`)
  e fazem cross-compile para `$TARGETARCH`: o arm64 não precisa de emulação
  para compilar, só para ser testado.
- Estágio final `gcr.io/distroless/static-debian12`: sem shell, sem toolchain,
  sem Node, sem arquivos de desenvolvimento.
- Usuário **numérico não-root** `65532:65532` (satisfaz `runAsNonRoot`).
- `WORKDIR /data`, `EXPOSE 8080`, `ENTRYPOINT ["/openjob"]`, `CMD ["serve"]`.
- **Nenhum** wizard, bootstrap interativo, navegador, descoberta de LAN ou
  comando privilegiado (ADR 0012).

### Configuração: somente variáveis de ambiente

| Variável | Default na imagem | Validação |
|---|---|---|
| `OPENJOB_PORT` | `8080` | inteiro 1–65535; inválido → exit 2 com mensagem |
| `OPENJOB_DB_PATH` | `/data/openjob.db` | não vazio |
| `OPENJOB_HOSTNAME` | `openjob.copilot` | nome DNS simples (sem esquema, `/`, `:`, `@`, espaço) |
| `OPENJOB_FEATURE_*` | desligadas | `true`/`1` ligam; qualquer outro valor desliga |
| `OPENJOB_CONTAINER` | `true` | marcador interno da imagem; não é configuração de produto |

Precedência: flag do comando > variável > default. No container, uma porta
ocupada falha; nada é adivinhado.

### Persistência

Banco e master key ficam no volume `/data`. O diretório é criado na imagem com
dono `65532`, de modo que um **volume nomeado novo herda a posse** e o usuário
não-root consegue escrever. Recriar o container com o mesmo volume mantém dados
e **a mesma key** (verificado por `scripts/smoke-docker.sh`).

Bind mounts (`./data`) precisam ser graváveis pelo uid 65532 — por isso o
`docker-compose.build.yml` documenta o `chown`.

### Health check

`HEALTHCHECK --interval=30s --timeout=5s --start-period=20s --retries=3` com
`/openjob healthcheck`, que sonda `/api/health` e lê o mesmo `OPENJOB_PORT` do
servidor. (Sem `curl` na imagem; ver ADR 0012.)

### Publicação (Docker Hub)

- Tags: `X.Y.Z` (imutável) e `latest` (só releases estáveis). **Não** há tag
  `X.Y`: ela mudaria a cada patch e contradiz imutabilidade.
- Arquiteturas: `linux/amd64` e `linux/arm64`, ambas **testadas** no CI antes
  da publicação (arm64 sob QEMU).
- Labels OCI: `source`, `revision`, `version`, `created`, `licenses`,
  `title`, `description`. Provenance e SBOM do buildx habilitados.
- Imutabilidade: antes de publicar, o job consulta `X.Y.Z`; se existe com o
  mesmo commit, pula (rerun idempotente); se existe com outro commit, falha.
- Credenciais só por GitHub Actions Secrets (`DOCKERHUB_USERNAME`,
  `DOCKERHUB_TOKEN`); o nome da imagem vem da variável `DOCKERHUB_IMAGE`.
  PRs e branches **nunca** publicam (o CI usa `load`, sem login).
- Sem segredos, `.env` ou dados pessoais em camadas (`.dockerignore` exclui
  `.env`, `data/`, `docs/`).

## Consequências

- O `docker-compose.yml` passa a usar a imagem publicada com volume nomeado
  `openjob-data`. Quem usava `./data` (bind) mantém esse comportamento com
  `docker-compose.build.yml`; não há migração automática entre os dois.
- O nome real da imagem no Docker Hub precisa ser confirmado pelo dono antes do
  primeiro release (o default documentado é `maicondmenezes/openjob`).

## Riscos

Segurança das credenciais do Docker Hub (tokens com escopo mínimo, rotação);
falha parcial na publicação coordenada (mitigada por release em *draft* e
reexecução idempotente); builds arm64 só validados sob emulação.
