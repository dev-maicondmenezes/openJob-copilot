# Checkpoint 5 — Entrada manual de vagas + scoring (1–2 dias) 🆕

> Origem: spec §8, Checkpoint 5, §3.3 Fase 1. Pré-requisitos: [03-sqlite-byok.md](03-sqlite-byok.md), [04-curriculo-tailoring.md](04-curriculo-tailoring.md).

## Objetivo

Usuário cola título, empresa, URL e JD; sistema calcula `score_fit` (0–100) + justificativa via IA; aprovação cria candidatura no pipeline. **A partir daqui o produto é usável de verdade** — não esperar o CP6.

## Pré-requisitos

- Checkpoints 3 e 4 concluídos (LLM client funcionando, tabela `jobs` migrada).

## Tarefas

1. **Entrada manual**:
   - `POST /api/jobs/manual` — `{title, company, url, description}` → persiste com `source='manual'`, `status='new'`.
   - Validar URL (se presente; campo opcional para vagas sem link).
2. **Scoring**:
   - `POST /api/jobs/:id/score` — envia JD + currículo base ao LLM; retorna e persiste `score_fit` (0–100) e `score_reason`.
   - **Parsing de JD robusto** (spec §10.5): limpar HTML/whitespace antes de mandar ao LLM.
   - Parser da resposta: exigir JSON estruturado no prompt (`{"score": int, "reason": str}`) com fallback de extração; clamp 0–100.
   - Sem chave configurada ou erro de API → erro claro, `score_fit` permanece NULL.
3. **Aprovação → pipeline**:
   - `POST /api/jobs/:id/approve` — cria `application` com `status='pending'`, `job_id` vinculado.
   - Impedir aprovação duplicada (uma `application` ativa por job).
4. **UI** (`/jobs`):
   - Form: colar título, empresa, URL, JD → salvar.
   - Lista/cards ordenados por `score_fit` (NULLs por último), badge com score e status (new/approved/discarded).
   - Botões: **"Score"**, **"Aprovar"** (cria candidatura → redireciona para `/applications`), **"Descartar"**.
   - Mostrar `score_reason` expandível no card.

## Critério de verificação

- Colar a **JD real da vaga que você está mirando** → clicar Score → receber score + justificativa coerente.
- Aprovar → candidatura aparece em `/applications` como `pending`.
- Colar a mesma JD com mesma URL duas vezes → erro de unicidade (ou no-op) — dedup por `url` já exercitada.
- **Testar no celular real** (fluxo completo colar → score → aprovar).

## Registrar

- Testes críticos (spec §10.5): **parsing de JD** e validação/clamp do score.
- Questionamento para calibrar (leva ao CP8): "o score de fit está calibrado ou está tudo 70–80?" — anotar v1 do prompt de scoring.
- Se surgir tentação de pular direto para o CP6: **não** — valor manual primeiro (spec §13.8).

## Fora de escopo

- Scrapers e ticker (CP6); Telegram (CP7); candidatura automática em ATS (CP9, opcional).
