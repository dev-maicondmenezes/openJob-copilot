# Contrato CI/CD de release (CP11)

Contrato entre o repositório e quem publica uma versão. Implementação:
`.github/workflows/{ci,frontend,release}.yml` e `.goreleaser.yaml`.

## Gatilho

**Somente** o push de uma tag estável `vX.Y.Z` (ex.: `v1.2.3`) em um commit que
esteja na `main`. A tag é a única fonte da versão. Não existe publicação manual
nem por push comum. Pré-releases (`-rc`) não disparam nada e nunca atualizariam
`latest`.

```bash
git switch main && git pull
git tag -a v1.2.3 -m "openJob 1.2.3"
git push origin v1.2.3
```

## Fluxo

```
verify ─► ci + frontend (gates reutilizados) ─┬─► binaries (GoReleaser → release em DRAFT)
                                              └─► docker (Docker Hub)
                                                        └─► publish-release (draft → público)
summary (sempre) — o que saiu e o que falhou
```

| Job | O que faz |
|---|---|
| `verify` | valida o formato da tag, que o commit está na `main` e que a release ainda não é pública |
| `ci` | `ci.yml` por `workflow_call`: `go vet`, `go test`, cross-compile dos 5 alvos, smoke do binário (Linux/macOS/Windows), dry-run do GoReleaser + smoke de binário e `.deb`, build e smoke da imagem em amd64/arm64 |
| `frontend` | `frontend.yml` por `workflow_call`: check, i18n, lint de UI, build, e2e, a11y, visual |
| `binaries` | GoReleaser: binários, `checksums.txt` (SHA-256), `.deb` amd64/arm64; cria a GitHub Release como **draft** |
| `docker` | imagem multi-arch no Docker Hub, tags `X.Y.Z` e `latest` |
| `publish-release` | confere os artefatos e torna a release pública — só se `binaries` **e** `docker` passaram |

A CI **não é duplicada**: PRs e pushes na `main` rodam `ci.yml`/`frontend.yml`;
a release os chama. Push de tag não dispara os dois por conta própria.

## Artefatos

- GitHub Release: `openjob_X.Y.Z_{linux,darwin}_{amd64,arm64}`,
  `openjob_X.Y.Z_windows_amd64.exe`, `openjob_X.Y.Z_{amd64,arm64}.deb`,
  `checksums.txt`.
- Docker Hub: `<DOCKERHUB_IMAGE>:X.Y.Z` e `:latest`, `linux/amd64` +
  `linux/arm64`, com provenance e SBOM.
- Versão, commit e data são os mesmos em todos (`openjob version`, labels OCI).

## Permissões e segredos

- Workflow: `contents: read`. `binaries` e `publish-release`: `contents: write`
  (o `GITHUB_TOKEN` basta; **sem PAT**).
- Secrets: `DOCKERHUB_USERNAME`, `DOCKERHUB_TOKEN` (token com escopo mínimo de
  push no repositório da imagem).
- Variável de repositório: `DOCKERHUB_IMAGE` (ex.: `<namespace>/openjob`).
- Todas as actions fixadas por **SHA de commit** (versão no comentário).
- Concorrência: `release-<tag>`, sem cancelamento em andamento.

## Falha parcial e reexecução

- A release só fica pública quando binários e imagem saíram. Se `binaries`
  passou e `docker` falhou, existe um *draft* invisível; corrigir e usar
  **Re-run failed jobs**.
- Se a imagem foi publicada e `binaries` falhou, a reexecução do `docker`
  detecta mesma versão + mesmo commit e **pula**.
- Versão publicada é imutável: tag já pública, ou imagem `X.Y.Z` vinda de outro
  commit, **falha** o workflow. Corrija publicando uma nova tag.
- O `summary` indica o que foi publicado e o que falhou.

## Antes do primeiro release (checklist do dono)

1. Criar o repositório no Docker Hub e definir `DOCKERHUB_IMAGE`.
2. Cadastrar `DOCKERHUB_USERNAME` e `DOCKERHUB_TOKEN`.
3. Definir a licença (`license` no `.goreleaser.yaml`, `ARG LICENSE` no
   Dockerfile e arquivo `LICENSE`).
4. Proteger tags `v*` (rulesets) para que só mantenedores as criem.
5. Substituir o e-mail do mantenedor em `nfpms.maintainer`.
