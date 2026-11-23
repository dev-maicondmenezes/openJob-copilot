# ADR 0033 — Reformulação do pipeline de candidaturas: stage_group in_process e sistema de tags

**Status:** aceito (Checkpoint 17 / Fase 2 MVP)  
**Data:** 2026-11-22  
**Relaciona:** docs/plan/17-email-tracking-pipeline.md, docs/plan/release-mvp.md, ADR 0009, ADR 0035  

## Contexto

O modelo de status fixo herdado do Checkpoint 5 e 7 (`pending`, `sent`, `interview`, `rejected`, `offer`) revelou-se restritivo diante da realidade dos processos seletivos. Primeiro, a vasta maioria dos e-mails de recrutamento são confirmações de recebimento, triagens curriculares ou desafios técnicos, que ficavam dispersos ou forçavam transições arbitrárias para `interview`. Segundo, cada empresa utiliza etapas próprias ("Fit cultural", "Live coding", "Entrevista com gestor", "Painel com diretoria"), tornando um enum fixo impossível de manter sem inflar o esquema.

## Decisão

Reformular o modelo do pipeline decompondo o estágio em duas dimensões complementares:

1. **`stage_group` (macro-estágio em 6 colunas):**
   - `pending`: vaga aprovada/salva, currículo em tailoring.
   - `in_process` ("Em processo"): candidatura submetida, aguardando resposta, em triagem ou teste técnico.
   - `interview`: qualquer etapa síncrona de entrevista.
   - `offer`: proposta formal de emprego recebida.
   - `rejected`: encerramento explícito do processo.
   - `closed`: encerramento administrativo pelo usuário (desistência, vaga congelada).
2. **`stage_label` e tags com namespace `phase:` (micro-estágio granular):**
   - Rótulo livre e tags associadas (`phase:desafio_tecnico`, `phase:entrevista_rh`, etc.) armazenadas na tabela `tags` e associadas via `application_tags`.
   - Namespaces suportados: `phase:` (estágio específico atualizável por automação/usuário), `company:`, `source:`, `priority:`.
3. **Retrocompatibilidade de dados:**
   - A migração `0006_email_tracking.sql` adiciona as colunas preservando a coluna legacy `status`, mapeando `sent → in_process` e preenchendo `stage_updated_at` e `stage_source`.

## Alternativas rejeitadas

- **Enum com dezenas de status (`screening`, `take_home`, `tech_interview`, `manager_interview`, etc.):** rígido, complexo de manter e confuso visualmente na interface kanban.
- **Campos de texto totalmente livres sem grupos:** perde a agregação por coluna e a clareza analítica do funil de candidaturas.

## Consequências

- Interface do pipeline reflete fielmente o funil real, com a coluna "Em processo" absorvendo o volume de triagens.
- Usuário mantém total flexibilidade para etiquetar etapas sem quebrar os filtros e agregações gerais.
- Facilidade de extensibilidade e relatórios futuros.
