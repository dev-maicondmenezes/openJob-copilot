# ADR 0034 — Sincronização de e-mails via polling incremental local-first sem webhooks

**Status:** aceito (Checkpoint 17 / Fase 2 MVP)  
**Data:** 2026-11-22  
**Relaciona:** docs/plan/17-email-tracking-pipeline.md, docs/plan/release-mvp.md, ADR 0010, ADR 0011  

## Contexto

A automação de leitura de e-mails precisa funcionar no ambiente local do usuário (notebook, desktop ou container pessoal), respeitando a premissa de arquitetura local-first e sem servidor intermediário gerido pelo projeto openJob. Protocolos baseados em push público (como webhooks do Gmail via Google Cloud Pub/Sub ou IMAP Webhooks de terceiros) exigem endereços IP públicos expostos, certificados SSL públicos e re-autorização periódica de inscrições com alta complexidade operacional.

## Decisão

Adotar a estratégia de **polling incremental com cursores locais**, utilizando conectores integrados para os dois protocolos suportados:

1. **Gmail:** uso da API REST oficial (escopo read-only `gmail.readonly`) com sincronização incremental via `users.history.list`. O cursor é o `historyId` armazenado em `email_accounts.last_history_id`. Se a API retornar HTTP 404 (histórico expirado), realiza sincronização das mensagens recentes.
2. **IMAP Genérico:** uso de conexão padrão TLS via `emersion/go-imap/v2` com busca incremental por `UID > last_uid`.
3. **Segurança de credenciais:** tokens OAuth e credenciais IMAP são cifrados localmente no banco SQLite com chave mestra AES-256-GCM (mesmo padrão BYOK).
4. **Agendamento assíncrono:** a sincronização roda como tarefa na fila assíncrona CP10 (`sync_email`), com ticker periódico opcional configurável (padrão desligado para dar controle total ao usuário).

## Alternativas rejeitadas

- **Google Cloud Pub/Sub com webhooks:** inviável em aplicações locais sem servidor público; adiciona custos de infraestrutura e barreiras técnicas intransponíveis para usuários finais.
- **IMAP IDLE (conexão TCP permanente aberta):** alta instabilidade em conexões domésticas/móveis e suspensão em laptops; polling intervalado de 5–15 minutos atende perfeitamente ao caso de acompanhamento de candidaturas.

## Consequências

- Funcionamento confiável e autônomo diretamente na máquina do usuário.
- Nenhuma dependência de serviços externos mantidos pelo openJob.
- Privacidade máxima: os e-mails e tokens jamais saem do computador do candidato.
