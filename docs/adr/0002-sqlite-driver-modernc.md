# ADR-0002 — SQLite sem CGO (`modernc.org/sqlite`) e binário estático

- **Status:** aceito (decisão antecipada no Checkpoint 1, conforme planejado)
- **Data:** 2026-11-13
- **Contexto:** spec §5 oferece `modernc.org/sqlite` ou `mattn/go-sqlite3`. O Dockerfile visa binário **estático** em runtime `distroless/static` (imagem mínima, sem libc/host deps).
- **Decisão:** adotar **`modernc.org/sqlite`** (pure Go, sem CGO) a partir do Checkpoint 3, e construir com `CGO_ENABLED=0` desde o Checkpoint 1 (`Makefile` e Dockerfile). `AUTOINCREMENT` e SQL específico ficam encapsulados no `internal/store` (portabilidade futura para Postgres, spec §4/§10).
- **Alternativas descartadas:**
  - `mattn/go-sqlite3`: exige CGO → build cruzado e imagem de runtime mais complexos, sem benefício real para o caso de uso pessoal.
- **Riscos conhecidos:** driver pure-Go pode ser mais lento em escritas intensas — irrelevante para single-user local-first. Atualização (CP3): o DSN do store ativa `journal_mode=WAL`, `busy_timeout=5000`, `foreign_keys=ON` e limita o pool a 1 conexão — leituras não bloqueiam a escrita e `SQLITE_BUSY` deixa de ser preocupação no workload single-user.
- **Consequência:** runtime `distroless/static-debian12` funciona; build Docker simples e reprodutível.
