# ADR 0032 — Cascata determinística de matching de e-mails de candidaturas com ATS como allowlist

**Status:** aceito (Checkpoint 17 / Fase 2 MVP)  
**Data:** 2026-11-22  
**Relaciona:** docs/plan/17-email-tracking-pipeline.md, docs/plan/release-mvp.md, ADR 0021, ADR 0035  

## Contexto

A intuição inicial de associar o e-mail recebido à candidatura através da comparação do domínio do remetente com o domínio da empresa da vaga falha completamente na prática. Em amostragens reais de caixas de entrada de candidaturas de emprego (237 e-mails avaliados), a taxa de acerto por domínio corporativo foi de 0%, uma vez que a esmagadora maioria das comunicações é enviada por relays de ATS (Applicant Tracking Systems), como `us.greenhouse-mail.io`, `hire.lever.co`, `myworkday.com`, `ashbyhq.com` e `smartrecruiters.com`.

Ademais, delegar o matching integralmente a modelos de linguagem (LLM) traria latência desnecessária, custos contínuos ao usuário e risco de alucinações.

## Decisão

Adotar uma **cascata determinística de matching em 5 etapas**, priorizando certeza estrutural antes de qualquer recurso a heurísticas ou IA:

1. **Continuidade de thread:** se o cabeçalho `In-Reply-To` ou `References` (`ThreadID`) coincidir com uma mensagem já associada a uma candidatura no banco local, herda a vinculação com confiança 1.0.
2. **Nome da empresa no assunto ou remetente/corpo:** normalização fonética/lexical (sem acentos, remoção de sufixos empresariais como Ltda, S/A, Inc, LLC) e verificação de limites de palavra (`\b`). Caso ocorra em assunto/remetente e o remetente pertença a um domínio de ATS conhecido, a confiança é atribuída como 0.95 (avanço automático). No corpo, a confiança é 0.85–0.90.
3. **Domínio corporativo direto:** testado exclusivamente quando o remetente **não** pertencer à lista de ATS conhecidos e não for webmail público (`gmail.com`, `outlook.com`, etc.).
4. **Relay de ATS sem nome explícito:** gera sugestão de baixa confiança (0.50–0.75) para revisão pelo usuário em banner na interface, evitando decisões automáticas prematuras.
5. **Cauda com LLM (opcional):** utilizado apenas quando há chave de IA configurada pelo usuário (BYOK) e os passos determinísticos anteriores não resolveram. O modelo opera sob vocabulário fechado e com teto de confiança em 0.85.

## Alternativas rejeitadas

- **Matching direto por domínio do remetente:** inviável devido à intermediação universal por ATS relays.
- **LLM no caminho principal de todas as mensagens:** lento, custoso e propenso a inconsistências.
- **Associação manual exclusiva:** transfere trabalho maçante e repetitivo para o usuário.

## Consequências

- Alta precisão na vinculação automática (> 90% para ATS populares com nome da empresa).
- Falsos positivos evitados: mensagens ambíguas são apresentadas como sugestão visual para confirmação do usuário.
- Custo computacional e financeiro mínimo: matching determinístico executado localmente em microssegundos.
