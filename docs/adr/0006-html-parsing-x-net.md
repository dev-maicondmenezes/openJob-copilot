# ADR 0006 — Parsing HTML com golang.org/x/net

## Contexto

O Checkpoint 5 tornou a **URL a fonte primária** do cadastro manual de vagas
(o usuário cola o link e o sistema preenche título/empresa/descrição).
Extrair esses dados exige interpretar HTML real (meta tags, JSON-LD,
maior bloco de texto), e o CP6 vai repetir a necessidade nos scrapers.

O backend tinha apenas `chi`, `modernc.org/sqlite` e `ledongthuc/pdf`;
fazer parsing de HTML com regex é frágil (atributos, entidades, DOM aninhado).

## Decisão

Adotar **`golang.org/x/net/html`** como parser HTML único do projeto
(usado em `internal/jobfetch`).

## Consequências

- Puro Go, sem CGO — consistente com [ADR 0002](0002-sqlite-driver-modernc.md).
- Dependência pequena e mantida pela equipe do Go; o mesmo pacote serve ao
  CP6 (scrapers), evitando segunda biblioteca (goquery/colly) no futuro próximo.
- Extração de vagas em camadas de confiança: JSON-LD `JobPosting` →
  meta tags (og:*) → maior bloco de texto (readability-lite), com `<script>`/
  `<style>` ignorados e páginas 100% client-rendered degradando para
  "descrição vazia → colagem manual".
