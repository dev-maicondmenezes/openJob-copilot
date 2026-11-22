# Checkpoint 17 — Automação de e-mail, pipeline "em processo" e tags de fase (6–8 dias)

> Origem: spec §3.4 (tracking) + CP7 (pipeline e Telegram) + CP10 (fila assíncrona e notificações). Pré-requisito: **CP7, CP10 e CP12 concluídos**. Este checkpoint ataca duas frentes que hoje são manuais: o usuário precisa abrir o e-mail para saber o que aconteceu com cada candidatura, e o pipeline tem estágios fixos que não refletem a diversidade real de processos por empresa/plataforma.

---

## 1. Objetivo

1. **Ler automaticamente o e-mail informado no currículo** e atualizar o estágio das candidaturas com base nos e-mails recebidos (confirmações, convites de entrevista, propostas, rejeições).
2. **Reformular o pipeline de candidaturas** para incluir a coluna generalista **`in_process` ("em processo")** — que agrupa todos os estágios intermediários entre "enviada" e "encerrada" — e um **sistema de tags** que permite representar o estágio específico de cada empresa/plataforma sem engessar o modelo.
3. **Não perder a autoridade do usuário**: toda transição sugerida pela automação é **revisável**; avanços automáticos só ocorrem com alta confiança; **nunca retrocede** um estágio automaticamente.

> **Princípio:** o pipeline deixa de ser uma escada rígida de 5 degraus e passa a ser um **estado + conjunto de tags de fase**. A automação de e-mail alimenta esse estado, mas o usuário continua sendo a fonte da verdade final.

---

## 2. Pré-requisitos e contexto de pesquisa

### 2.1 Pré-requisitos técnicos

- **CP7** — `applications` com `status`, `notify_log`, pipeline visual.
- **CP10** — fila de tarefas assíncronas (`tasks`) e event bus; esta automação vira um **handler de tarefa** (`sync_email`), reutilizando retry/cancel/progresso.
- **CP12** — `lib/ui/` e `DataView` para a nova UI de tags.
- **CP4** — currículo com o campo de e-mail; o endereço é pré-preenchido na configuração da conta de e-mail, mas o usuário confirma.

### 2.2 Descoberta crítica da pesquisa (que muda o design)

A intuição inicial — "casar o domínio do remetente com o domínio da vaga" — **não funciona na prática** para a maioria das candidaturas. Levantamentos reais de inbox de candidaturas mostram que **os e-mails vêm de relays de ATS, não do domínio da empresa**:

- `us.greenhouse-mail.io` (Greenhouse)
- `hire.lever.co` (Lever)
- `*.myworkday.com` (Workday)
- `notifications@ashbyhq.com` (Ashby)
- `smartrecruiters.com`

Em um conjunto real de 237 e-mails, **o matching por domínio do remetente contra o domínio da empresa teve 0 acertos** — porque o domínio é do ATS, não do empregador. Portanto, o checkpoint adota uma **cascata determinística** (thread → nome da empresa no assunto/corpo → domínio do ATS como *allowlist*, não como identidade) e deixa o LLM apenas para a cauda de baixa confiança.

---

## 3. Modelo de dados

### 3.1 `email_accounts` (nova)

| Coluna | Tipo | Descrição |
|---|---|---|
| `id` | TEXT PK | Slug (`gmail`, `imap`) |
| `provider` | TEXT | `gmail` \| `imap` |
| `address` | TEXT | E-mail monitorado (pré-preenchido do currículo) |
| `auth_config` | BLOB | Cifrado AES-GCM (OAuth token / senha de app) |
| `folder` | TEXT | Pasta monitorada (default `INBOX`) |
| `last_uid` / `last_history_id` | TEXT | Cursor incremental (IMAP UID / Gmail `historyId`) |
| `enabled` | INTEGER | Liga/desliga a sincronização |
| `last_sync_at` / `last_error` | DATETIME / TEXT | Observabilidade |

### 3.2 `email_messages` (nova)

| Coluna | Tipo | Descrição |
|---|---|---|
| `id` | TEXT PK | ULID |
| `account_id` | TEXT FK | Conta de origem |
| `message_id` | TEXT UNIQUE | `Message-ID` do e-mail (dedup natural) |
| `thread_id` | TEXT | Para continuidade de thread |
| `from_address` / `from_name` | TEXT | Remetente |
| `subject` | TEXT | Assunto |
| `received_at` | DATETIME | Data do recebimento |
| `body_text` | TEXT | Corpo em texto puro (truncado em ~20 KB) |
| `matched_application_id` | TEXT FK (nullable) | Candidatura vinculada |
| `match_confidence` | REAL | 0–1 |
| `stage_signal` | TEXT (nullable) | Sinal classificado (`ack`, `screening`, `interview`, `assessment`, `offer`, `rejection`, `info_request`, `other`) |
| `classified_at` | DATETIME | Quando foi classificado |
| `link_source` | TEXT | `auto` \| `suggested` \| `manual` |

### 3.3 Reformulação de `applications`

| Coluna | Tipo | Descrição |
|---|---|---|
| `stage_group` | TEXT | **`pending`** \| **`in_process`** \| **`interview`** \| **`offer`** \| **`rejected`** \| **`closed`** |
| `stage_label` | TEXT (nullable) | Rótulo livre do estágio específico (ex.: "Desafio técnico", "Entrevista com gestor", "Painel final") |
| `stage_updated_at` | DATETIME | Quando o estágio mudou |
| `stage_source` | TEXT | `manual` \| `email_auto` \| `email_suggested` |

**Relação com o modelo antigo:** o campo `status` do CP7 é **substituído** por `stage_group` (migração mapeia `sent → in_process` e `pending → pending`). A UI de pipeline passa a exibir **`stage_group` como coluna** e **`stage_label` como tag dentro do card**.

### 3.4 `tags` + `application_tags` (novas)

| `tags` | |
|---|---|
| `id` | INTEGER PK |
| `namespace` | TEXT | `phase`, `company`, `source`, `priority` |
| `value` | TEXT | Ex.: `technical_interview`, `acme_corp`, `gupy`, `high` |
| `label` | TEXT | Exibição (i18n via CP13) |

| `application_tags` | |
|---|---|
| `application_id` | FK |
| `tag_id` | FK |
| PK composta | `(application_id, tag_id)` |

**Regra de namespace:** `phase:` é **reservado** para o estágio específico e é o único que pode ser alterado pela automação de e-mail. `company:`, `source:` e `priority:` são geridos pelo usuário (ou derivados na criação da candidatura).

---

## 4. Pipeline reformulado

### 4.1 Os seis `stage_group`

| `stage_group` | Significado | Inclui |
|---|---|---|
| `pending` | Descoberta, ainda não aplicada | Vaga salva, currículo em tailoring |
| **`in_process`** | **"Em processo" — candidatura enviada, aguardando ou em triagem** | Confirmação de recebimento, triagem de currículo, assessment, desafio técnico |
| `interview` | Entrevistas agendadas ou realizadas | Qualquer rodada de entrevista |
| `offer` | Proposta recebida | Oferta formal |
| `rejected` | Rejeição explícita | Rejeição em qualquer fase |
| `closed` | Encerrada sem desfecho | Vaga cancelada, processo congelado, desistência do usuário |

**Por que `in_process` separado de `interview`?** A pesquisa mostra que a maioria dos e-mails de candidatura é de **confirmação/triagem**, não de entrevista. Agrupar tudo em "em processo" evita inflar a coluna de entrevistas e dá ao usuário uma visão realista de onde estão as candidaturas. O `stage_label` (tag `phase:`) preserva a granularidade.

### 4.2 Transições permitidas

| De → Para | Automático? | Condição |
|---|---|---|
| `pending → in_process` | ✅ Sim | Confirmação de recebimento (`ack`) com confiança ≥ 0.8 |
| `in_process → interview` | ✅ Sim | Convite de entrevista (`interview`) com confiança ≥ 0.8 |
| `in_process → rejected` | ✅ Sim | Rejeição (`rejection`) com confiança ≥ 0.8 |
| `interview → offer` | ✅ Sim | Proposta (`offer`) com confiança ≥ 0.8 |
| `interview → rejected` | ✅ Sim | Rejeição pós-entrevista |
| Qualquer → `closed` | ❌ Não | Apenas manual |
| **Qualquer retrocesso** | ❌ **Nunca** | O sistema **recusa** mover para trás automaticamente; uma rejeição nunca "reabre" um processo |
| Confiança < 0.8 | ❌ Não | Vira **sugestão** na UI (banner "Parece que esta candidatura avançou — confirmar?") |

---

## 5. Tarefas

### 5.1 Conector de e-mail (`internal/email/`)

- **Gmail:** OAuth read-only (`gmail.readonly`); sincronização incremental via `users.history.list` com **polling** (default 5 min), **sem Pub/Sub** — a arquitetura local-first não deve exigir webhook público nem provisionamento de tópico. O `historyId` é o cursor; em `404`, faz re-sync completo.
- **IMAP genérico:** `emersion/go-imap` v2 + `emersion/go-message` (ou `jhillyerd/enmime`) para MIME; cursor por `UID` (`last_uid`), busca incremental por `UID > last_uid`. IDLE é **opcional** (backlog); o default é polling.
- **Parser MIME:** `enmime` para multipart/charset (inclui PT/ES); corpo em texto puro, sem renderizar HTML.
- **Rate limit:** conservador por conta (ex.: 1 sync a cada 5 min; Gmail quota é generosa, mas o polling não deve ser agressivo).

### 5.2 Cascata de matching (`internal/email/match/`)

Ordem determinística (do mais forte para o mais fraco):

1. **Continuidade de thread:** se o `thread_id` já está vinculado a uma candidatura, herda o vínculo.
2. **Nome da empresa no assunto/corpo:** normalização (lowercase, sem acentos) + *fuzzy match* contra as empresas das candidaturas abertas do usuário.
3. **Remetente com domínio de ATS conhecido:** `greenhouse.io`, `lever.co`, `myworkday.com`, `ashbyhq.com`, `smartrecruiters.com`, `icims.com` — usado como **allowlist** (confirma que é candidatura), **não** como identidade da empresa. O vínculo à candidatura vem do passo 2.
4. **Domínio corporativo:** só tenta se o remetente **não** for ATS; cobertura real baixa (0/237 em dados de referência), mas mantido como último recurso determinístico.
5. **LLM (opcional, BYOK):** apenas para a cauda não resolvida pelos passos 1–4; extrai empresa + sinal de estágio. **Nunca** é o primeiro passo.

**Saída:** `(application_id, confidence, stage_signal)`.

### 5.3 Classificação de estágio (`internal/email/classify/`)

- **Vocabulário controlado** (nunca persistir valor fora do vocabulário): `ack`, `screening`, `interview`, `assessment`, `offer`, `rejection`, `info_request`, `other`.
- **Regras determinísticas primeiro:** regex/lexical para "unfortunately", "we regret", "interview scheduled", "offer letter", "next steps", "application received".
- **LLM apenas se as regras não decidirem** e o usuário tiver chave configurada.
- **Sanitização:** saída do LLM é validada contra o vocabulário; fora dele → `other` (nunca persistir string livre).

### 5.4 Worker assíncrono (reutiliza CP10)

- Novo handler `sync_email` na fila de tarefas: busca novos e-mails → parse → matching → classificação → aplica transições.
- **Agendamento:** ticker configurável (default 5 min) **desligado por padrão** em `/settings/email`, como o discovery do CP6.
- **Progresso:** cada execução reporta `emails_fetched`, `matched`, `advanced`, `suggested` — visível em `/tasks`.
- **Falha isolada:** erro de uma conta não derruba as demais; notificação `task_failed` no sino (CP10).

### 5.5 API e UI

**API:**
- `GET/PUT /api/settings/email` — conta, provider, OAuth/IMAP, intervalo de sync.
- `POST /api/settings/email/test` — conexão + fetch de 1 e-mail de amostra.
- `GET /api/email/messages?application_id=…` — e-mails vinculados a uma candidatura.
- `POST /api/email/messages/{id}/link` — vinculação manual (quando o matching falhou).
- `PATCH /api/applications/{id}` — passa a aceitar `stage_group`, `stage_label` e `tags`.

**UI (`/applications`):**
- Pipeline em colunas com os **6 `stage_group`**; a coluna **"Em processo"** é a mais larga (comporta o volume real).
- Card da candidatura exibe **tags de fase** (`phase:`) como chips; clique abre o seletor de tags.
- **Banner de sugestão:** quando há um e-mail com confiança < 0.8, aparece no topo da coluna: *"3 candidaturas podem ter avançado — revisar"*.
- **Drawer de detalhe da candidatura:** lista os e-mails vinculados com badge de `stage_signal`; botão "Vincular e-mail" para casos não resolvidos.
- **`/settings/email`:** conectar Gmail (OAuth) ou IMAP; mostrar status da última sincronização; aviso de que **só e-mails de candidatura são processados** (allowlist de ATS + matching), e que o conteúdo **nunca sai do computador** exceto para o LLM se o usuário tiver configurado.

---

## 6. Critérios de verificação

- **Gmail OAuth:** conectar uma conta → `sync_email` roda; e-mails de confirmação de candidatura aparecem vinculados; `applications.stage_group` muda de `pending` para `in_process` com `stage_source = email_auto`.
- **IMAP:** conectar uma conta IMAP genérica → mesma verificação.
- **Matching por thread:** uma resposta na mesma thread de um convite de entrevista herda o vínculo e classifica como `interview` (não cria duplicata).
- **Matching por nome da empresa:** e-mail com "Acme Corp" no assunto vincula à candidatura da Acme mesmo que o remetente seja `no-reply@greenhouse.io`.
- **Domínio do ATS como allowlist:** e-mail de `hire.lever.co` sem nome de empresa no corpo **não** é vinculado automaticamente (fica como sugestão).
- **Domínio corporativo:** e-mail de `rh@empresa.com.br` vincula à candidatura da empresa (quando o domínio bate).
- **Confiança baixa:** e-mail ambíguo gera **sugestão**, não transição automática; o banner aparece em `/applications`.
- **Monotonicidade:** após uma rejeição (`rejected`), um e-mail posterior de "confirmação" **não** move o estágio para trás; o sistema recusa e registra em log.
- **Pipeline:** os 6 `stage_group` aparecem; a coluna "Em processo" agrupa confirmações/triagem/assessment; `stage_label` é editável e vira tag `phase:`.
- **Tags:** criar tag `phase:desafio_tecnico` e aplicá-la a uma candidatura; filtrar por ela no pipeline.
- **Privacidade:** o corpo do e-mail é truncado e armazenado localmente; nenhum e-mail é enviado ao provedor LLM sem que o usuário tenha configurado a chave (BYOK); o log não expõe corpo.
- **Sem regressão:** o Telegram do CP7 continua recebendo `application.status_changed` (agora derivado de `stage_group`); notificações do CP10 continuam funcionando.

---

## 7. Registrar

- **ADR `0032-email-matching-cascata-ats.md`** — cascata determinística thread → nome da empresa → allowlist de ATS → domínio corporativo → LLM na cauda. **Justificativa:** o matching por domínio do remetente falha porque os e-mails vêm de relays de ATS (0/237 em dados de referência); a cascata preserva precisão sem depender de LLM no caminho principal.
- **ADR `0033-pipeline-stage-group-tags.md`** — substituir o `status` fixo por `stage_group` (6 valores, incluindo `in_process`) + `stage_label` + tags com namespace `phase:`. **Justificativa:** cada empresa/plataforma tem estágios próprios; tags preservam a granularidade sem inflar o enum. Alternativa rejeitada: enum com 20+ valores (rígido e impossível de manter).
- **ADR `0034-email-sync-polling-local-first.md`** — polling incremental (Gmail `history.list` / IMAP `UID > last_uid`), sem Pub/Sub nem webhook público. **Justificativa:** o app roda no notebook do usuário; provisionar tópico Pub/Sub e re-armar `watch` a cada 7 dias é complexidade desnecessária para single-user. IDLE fica como backlog.
- **ADR `0035-email-advance-monotonic.md`** — avanço automático **apenas para frente** e **apenas com confiança ≥ 0.8**; rejeição nunca reabre. **Justificativa:** um falso positivo não pode "fechar" ou "reabrir" um processo silenciosamente; o usuário precisa confiar no pipeline.
- **Testes:**
  - Cascata de matching com fixtures de Greenhouse, Lever, Workday, Ashby e domínio corporativo.
  - Classificação com vocabulário controlado (nunca persiste fora do vocabulário).
  - Monotonicidade (rejeição → confirmação posterior não retrocede).
  - Dedup por `Message-ID`.
  - Sync incremental (não re-processa e-mails já vistos).
  - UI: sugestão de baixa confiança aparece e pode ser confirmada/rejeitada.
- **Riscos anotados:**
  - **Matching por nome da empresa é frágil** para nomes genéricos ("Global", "Digital"); mitigação: exigir confiança mínima e usar thread como âncora.
  - **LLM na cauda** pode alucinar empresa/estágio; mitigação: vocabulário fechado + confiança + revisão humana.
  - **Gmail OAuth** exige app no Google Cloud; para uso local, o usuário cria o próprio client_id (documentar em `/settings/email`), ou usa IMAP com senha de app (mais simples, porém menos "plug and play").
  - **Volume de e-mail:** inbox pessoal tem ruído; a allowlist de ATS + matching por empresa reduz falsos positivos, mas o usuário deve poder desvincular um e-mail errado com um clique.

---

## 8. Fora de escopo (deliberadamente adiado)

- **IMAP IDLE** (push em tempo real) — polling de 5 min é suficiente para um copiloto; IDLE fica como backlog.
- **Webhook público / Pub/Sub** — incompatível com local-first sem servidor central (CP11).
- **Extração de datas de entrevista e criação de eventos de calendário** — o CP17 só classifica o estágio; agendamento fica para um checkpoint futuro.
- **Múltiplas contas de e-mail por usuário** — uma conta nesta CP; multi-conta fica para backlog.
- **Envio de e-mail pelo openJob** — o app continua **somente leitura**; não responde a recrutadores (spec §3.3).
- **Automação de candidatura** — permanece proibida.
