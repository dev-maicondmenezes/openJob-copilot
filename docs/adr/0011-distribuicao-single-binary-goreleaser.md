# ADR 0011 — Distribuição: binário único, GoReleaser e canais oficiais

**Status:** aceito (Checkpoint 11)
**Data:** 2026-11-23
**Relaciona:** docs/plan/11-package-release.md, ADR 0002 (SQLite puro-Go), ADR 0003 (SPA embutida), ADR 0012, ADR 0036

## Contexto

Até o CP10 o único caminho de instalação era `docker compose up` a partir do
código-fonte. O produto é local-first e precisa rodar também em máquinas sem
Docker nem toolchain, e a imagem Docker precisa ser um artefato publicado, não
algo que cada pessoa constrói.

## Decisão

Três canais oficiais, todos produzidos **a partir da mesma tag Git `vX.Y.Z`**:

1. **Binário único por plataforma** — linux/amd64, linux/arm64, darwin/amd64,
   darwin/arm64, windows/amd64. SPA embutida (`go:embed`), SQLite puro-Go,
   `CGO_ENABLED=0`. Versão/commit/data injetados por `-ldflags` em
   `internal/version`. O binário individual é o artefato (sem tarball), com
   `checksums.txt` SHA-256.
2. **Pacote `.deb`** (amd64 e arm64), gerado pelo **mesmo** GoReleaser via nfpm.
   Instala `/usr/bin/openjob` e um atalho desktop. **Sem scripts de
   mantenedor, sem serviço systemd, sem iniciar servidor no post-install.**
3. **Imagem Docker** multi-arquitetura no Docker Hub (ADR 0036).

GoReleaser é a fonte única de binários, checksums, `.deb` e GitHub Release. Não
há pipeline separado para o `.deb`.

### Decisões de detalhe

- **Windows arm64 fica de fora**: não há runner para testá-lo; o plano exige
  smoke test em todo alvo publicado.
- **Smoke por alvo no CI**: linux/amd64, darwin/arm64, windows/amd64 nativos;
  darwin/amd64 via Rosetta no runner arm64; linux/arm64 pela imagem Docker e
  pelo `.deb` em um Debian arm64 (QEMU).
- **Checksums sim, assinatura não** (fora do escopo do CP11).
- **Dados por usuário, não do sistema**: o `.deb` não cria diretórios de dados;
  o primeiro uso cria `~/.local/share/openjob` (ADR 0012). Upgrade e remoção
  (inclusive `purge`) nunca tocam nesses dados.
- **Sem auto-update, RPM, MSI, DMG, serviços** (fora do escopo).

## Consequências

- Um release = uma tag; versão idêntica em binários, `.deb`, GitHub Release e
  imagem (contrato em `docs/release/ci-cd.md`).
- Quem instala por `.deb` ou binário não precisa de Docker nem de Go.
- Os alvos darwin/amd64 só rodam em CI sob Rosetta; é um risco aceito e
  documentado.
- Sem LICENSE no repositório, o campo `license` do pacote e o label OCI estão
  como `NOASSERTION` até o dono definir.

## Riscos

Firewall local e múltiplas interfaces/VPN afetam o acesso por celular (tratado
em ADR 0012); diferença de diretório de dados entre `docker` (`/data`) e
binário (por usuário) — não há migração silenciosa entre eles.
