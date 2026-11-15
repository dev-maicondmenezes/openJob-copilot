# Checkpoint 4 — Currículo + Adaptação (3–4 dias)

> Origem: spec §8, Checkpoint 4, §3.1. Pré-requisito: [03-sqlite-byok.md](03-sqlite-byok.md).

## Objetivo

Upload de currículo (PDF/MD/TXT) convertido para Markdown estruturado, e geração de **currículo adaptado** via IA a partir de uma JD — **primeiro momento de valor real**.

## Pré-requisitos

- Checkpoint 3 concluído (chave BYOK configurada e testada; tabela `curriculum` migrada).

## Tarefas

1. [x] **Upload** (`internal/api/` + handler de curriculum):
   - `POST /api/curriculum/upload` — multipart: PDF, MD ou TXT.
   - PDF → texto via `github.com/ledongthuc/pdf`; MD/TXT direto.
   - Normalizar para **Markdown estruturado** (seções: experiência, skills, educação...); armazenar em `curriculum.content_md`.
   - Regras: **uma versão ativa** — upload novo substitui o anterior (linha mais recente é a ativa; manter histórico é opcional, decidir e documentar). Decisão: histórico preservado no SQLite (`ORDER BY id DESC LIMIT 1`).
2. [x] **Tailoring** (`internal/tailor/`):
   - `POST /api/curriculum/tailor` — recebe `{job_id}` **ou** `{jd_text}` (texto colado); retorna currículo adaptado em Markdown.
   - **Prompt versionado** em `internal/tailor/prompts.go` (`PromptVersion = "v1"`): instruções de alinhamento de palavras-chave e realizações, sem inventar experiências.
   - Regras de prompt: manter fatos, reordenar/enfatizar, incluir keywords da JD quando verdadeiras, saída **só em Markdown**.
3. [x] **UI** (`/curriculum`):
   - Upload com preview do Markdown (render simples e raw).
   - Form de JD colada + botão "Gerar versão adaptada".
   - **Diff visual base vs. adaptado** (lado a lado e diff unificado com adição/remoção).

## Critério de verificação

- Subir **seu currículo real** (PDF), colar uma **JD real que você está mirando**, gerar a versão adaptada e **ler criticamente**.
- Reiniciar container: currículo persiste; upload novo substitui o ativo.
- Sem chave configurada → erro claro (não 500 genérico).

## Registrar

- **Versão do prompt `v1`**: anotada em `internal/tailor/prompts.go` (`PromptVersion = "v1"`). Foco estrito em veracidade zero alucinação, alinhamento ATS e saída exclusivamente em Markdown limpo.
- **Teste de parsing de PDF**: implementado em `internal/curriculum/parser_test.go` (`TestParsePDF` e `TestParseEmptyPDF`), validando extração de texto real e rejeição de PDFs vazios/escaneados com mensagem orientativa.
- **Limitação de tamanho**: `MaxCurriculumChars = 24000` e `MaxJDChars = 24000` (~8k tokens cada) definidos em `internal/tailor/prompts.go` com truncamento seguro para proteger custos e janelas de contexto.
- **Histórico vs. Ativo**: cada novo upload cria uma linha no SQLite; o endpoint ativo seleciona `ORDER BY id DESC LIMIT 1`. Isso mantém rastreabilidade sem complicar o modelo de dados.

## Fora de escopo

- Salvar o tailored em `applications` (isso acontece no CP5 via aprovação).
- Importação LinkedIn via API (fase de productização; MVP permite colar perfil manualmente — spec §3.1).

## Backlog de Melhorias (detalhadas em `docs/backlog.md`)

- **B7 — Extração de currículo via LinkedIn OAuth**: conectar perfil via autorização para extrair dados direto para o Markdown do currículo-base.
- **B8 — Tratamento de erros amigável**: mascarar detalhes de logs/stacks internas e exibir orientações amigáveis e acionáveis ao usuário.
- **B9 — Exportação da versão adaptada para PDF, TXT e MD**: estender o download para todos os formatos aceitos na importação.
- **B10 — Botão discreto de cópia**: botão minimalista de cópia direta no canto superior direito da área de exibição da versão adaptada.
