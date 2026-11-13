# Checkpoint 4 — Currículo + Adaptação (3–4 dias)

> Origem: spec §8, Checkpoint 4, §3.1. Pré-requisito: [03-sqlite-byok.md](03-sqlite-byok.md).

## Objetivo

Upload de currículo (PDF/MD/TXT) convertido para Markdown estruturado, e geração de **currículo adaptado** via IA a partir de uma JD — **primeiro momento de valor real**.

## Pré-requisitos

- Checkpoint 3 concluído (chave BYOK configurada e testada; tabela `curriculum` migrada).

## Tarefas

1. **Upload** (`internal/api/` + handler de curriculum):
   - `POST /api/curriculum/upload` — multipart: PDF, MD ou TXT.
   - PDF → texto via `github.com/ledongthuc/pdf`; MD/TXT direto.
   - Normalizar para **Markdown estruturado** (seções: experiência, skills, educação...); armazenar em `curriculum.content_md`.
   - Regras: **uma versão ativa** — upload novo substitui o anterior (linha mais recente é a ativa; manter histórico é opcional, decidir e documentar).
2. **Tailoring** (`internal/tailor/`):
   - `POST /api/curriculum/tailor` — recebe `{job_id}` **ou** `{jd_text}` (texto colado); retorna currículo adaptado em Markdown.
   - **Prompt versionado** em `internal/tailor/prompts.go` (const com versão, ex. `promptVersion = "v1"`): instruções de alinhamento de palavras-chave e realizações, sem inventar experiências.
   - Regras de prompt: manter fatos, reordenar/enfatizar, incluir keywords da JD quando verdadeiras, saída **só em Markdown**.
3. **UI** (`/curriculum`):
   - Upload com preview do Markdown (render simples).
   - Form de JD colada + botão "Gerar versão adaptada".
   - **Diff visual base vs. adaptado** (lado a lado ou diff simplificado).

## Critério de verificação

- Subir **seu currículo real** (PDF), colar uma **JD real que você está mirando**, gerar a versão adaptada e **ler criticamente**.
- Reiniciar container: currículo persiste; upload novo substitui o ativo.
- Sem chave configurada → erro claro (não 500 genérico).

## Registrar

- Versão do prompt `v1` anotada (iterar depois com base no CP8 — "o tailoring está bom o suficiente?").
- Teste de parsing de PDF (currículo real não pode sair vazio/garbled).
- Nota: limitar tamanho de JD/currículo (ex. truncar em ~8k tokens) para custo e contexto.

## Fora de escopo

- Salvar o tailored em `applications` (isso acontece no CP5 via aprovação).
- Importação LinkedIn via API (fase de productização; MVP permite colar perfil manualmente — spec §3.1).
