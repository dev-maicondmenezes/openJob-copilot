# Checkpoint 8 — Validação pessoal (2–4 semanas, sem código novo) 🆕

> Origem: spec §8, Checkpoint 8 e §9. Pré-requisito: Checkpoints 1–7 concluídos e o app em uso real.

## Objetivo

**Usar o app de verdade**: aplicar para 5–15 vagas com currículos adaptados, acompanhar o pipeline, receber notificações — e **coletar dados** para decidir se vale productizar. **Não avançar sem isso** (spec §8).

## Ação (sem código novo — exceções permitidas abaixo)

- Aplicar para 5–15 vagas usando o fluxo completo: descoberta/manual → score → aprovação → tailoring → candidatura no ATS → tracking de status.
- Correções triviais (typos, copy, pequenos ajustes de prompt) são permitidas; **nenhuma feature nova** — qualquer ideia nova vai para uma lista "deferred post-validation".

## O que registrar (log semanal, ex. `docs/plan/validation-log.md`)

1. Quantas vagas **aprovou vs. descartou**?
2. Quantas **geraram resposta**?
3. O **tailoring** está bom o suficiente ou você edita manualmente toda vez?
4. O **score de fit** está calibrado ou está tudo 70–80?
5. O que você **sente falta** toda vez que abre o app?

## Métricas da fase de validação (spec §9)

| Pergunta | O que indica |
|---|---|
| O tailoring gerou entrevistas? | IA funcionando ou prompt precisa de iteração |
| Usou o app mais de 10x em 2 semanas? | Se não, o problema é seu, não do mercado |
| Alguma feature foi ignorada? | Cortar antes de productizar |
| Telegram foi útil ou irritante? | Ajustar frequência |
| Maior atrito no uso diário? | Esse é o próximo feature |

**Critério de decisão:** em 4 semanas, 10+ candidaturas e 1+ entrevista → o núcleo tem valor. Parou de usar → problema é mais profundo que "faltou productização".

## Critério de verificação

- **Dados reais documentados** no log de validação respondendo às 5 perguntas do §9.
- Decisão explícita registrada: productizar, iterar, ou arquivar.

## Saídas possíveis

- **Productizar** → ler [09-ats-assistido.md](09-ats-assistido.md) apenas se houver dor real com formulários ATS; caso contrário, priorizar o maior atrito medido.
- **Iterar** → ajustar prompts (tailoring/scoring), frequência do Telegram, fluxo de UI — sem escopo de produto.
- **Arquivar** → documentar aprendizados; o custo foi baixo (~R$ 5–20/mês, spec §11).
