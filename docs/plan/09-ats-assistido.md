# Checkpoint 9 (⚠️ opcional) — ATS assistido

> Origem: spec §8, Checkpoint 9 e §3.3. **Só iniciar se, na validação do [Checkpoint 8](08-validacao-pessoal.md), houver dor real** de preencher formulários Greenhouse/Lever manualmente. Não iniciar antes do fim da validação pessoal (spec §13.9).

## Objetivo

Assistir o preenchimento de formulários de ATS (Greenhouse, Lever) via Playwright headless, **parando antes do botão "Enviar"** — o usuário revisa na UI e confirma o envio manualmente.

## Pré-requisitos (rígidos)

- Checkpoint 8 concluído com decisão de productizar/continuar.
- Evidência registrada de dor real (ex.: "perdi 20 min por candidatura preenchendo Greenhouse").

## Tarefas

1. **Runtime Playwright**:
   - Playwright headless dentro do container (imagem base precisa incluir navegador — reavaliar tamanho da imagem; alternativa: usar apenas Greenhouse/Lever por API pública quando disponível antes de recorrer a browser automation).
   - Segregação de responsabilidade: módulo próprio (`internal/ats/`), nunca misturado ao tailoring/scoring.
2. **Fluxo assistido**:
   - O preenchimento **sempre para antes do envio**; estado `draft` visível na UI com os campos preenchidos.
   - Usuário revisa, edita o que quiser e **confirma o envio manualmente** (regra inegociável — copiloto, não "spray and pray", spec §1).
3. **Guardrails**:
   - **Limite diário de candidaturas assistidas: 5–10** (configurável) para evitar bloqueios.
   - Timeout e retry limitado por formulário; falha → degrada para "preencha manualmente" com link do formulário.
4. **LinkedIn**:
   - **Não automatizado** — nem login, nem Easy Apply, nem scraping agressivo (spec §1, §4, §8).

## Critério de verificação

- Para uma vaga Greenhouse de teste: formulário é preenchido automaticamente, usuário **revisa na UI e envia manualmente** com sucesso.
- Limite diário respeitado (10ª tentativa no dia é bloqueada com mensagem clara).

## Registrar

- ADR: "deferred post-validation" para ATS assistido (spec §13.9) — com a evidência de dor que justificou a feature.
- Riscos: seletores de formulários mudam com frequência; bloqueios anti-bot; custo de manutenção alto — reavaliar continuamente se vale manter.
