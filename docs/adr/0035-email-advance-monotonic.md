# ADR 0035 — Princípio de monotonicidade e autoridade do usuário nas transições por e-mail

**Status:** aceito (Checkpoint 17 / Fase 2 MVP)  
**Data:** 2026-11-22  
**Relaciona:** docs/plan/17-email-tracking-pipeline.md, docs/plan/release-mvp.md, ADR 0032, ADR 0033  

## Contexto

A automação de atualização de status pode gerar falsos positivos em caixas de entrada concorridas. Se um sistema de rastreamento de vagas retroceder um estágio automaticamente (por exemplo, após uma entrevista receber um e-mail com confirmação de recebimento genérica) ou reabrir uma candidatura já arquivada/rejeitada, a confiança do usuário no copiloto é imediatamente destruída.

## Decisão

Instituir o **princípio da monotonicidade estrita** e da **autoridade final do usuário** nas transições automáticas:

1. **Avanço estritamente progressivo:** a automação só avança o estágio se a classificação apontar para um estágio de classificação superior na hierarquia (`pending` < `in_process` < `interview` < `offer`).
2. **Rejeição como estado terminal:** uma vez que a candidatura se encontra em `rejected` ou `closed`, nenhuma mensagem subsequente pode reabri-la ou alterar seu estágio automaticamente.
3. **Exigência de alta confiança:** transições automáticas (`stage_source = 'email_auto'`) exigem pontuação de matching e classificação com confiança mínima de 0.80.
4. **Sugestões para casos de baixa confiança:** qualquer correlação com confiança menor que 0.80 ou ambígua é gravada com `link_source = 'suggested'`, gerando um banner informativo na interface para que o usuário decida confirmar ou descartar.
5. **Transições manuais ilimitadas:** o usuário pode a qualquer momento mover a candidatura para qualquer coluna, sobrescrevendo decisões da automação.

## Alternativas rejeitadas

- **Permitir retrocessos automáticos:** causava oscilações erráticas quando múltiplos e-mails de uma mesma thread eram processados fora de ordem cronológica.
- **Decisão sem limiar de confiança:** gerava falsos avanços indesejados a partir de newsletters de recrutamento ou e-mails automáticos de marketing de plataformas de vagas.

## Consequências

- Garantia absoluta de estabilidade do pipeline: estágios consolidados nunca são desfeitos por automação.
- O usuário permanece como a autoridade central do processo, com a automação atuando como assistente diligente.
