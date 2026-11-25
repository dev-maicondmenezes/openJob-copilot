# Plano de Release — MVP openJob

> Objetivo: fechar o MVP executando **CP16 + CP17 + backlog restante**, em fases
> auto-contidas, testáveis e mergeáveis de forma independente. Cada fase tem
> escopo fechado, critérios de aceite próprios e um gate de verificação antes de
> avançar.

---

## 1. Definição de pronto do MVP

O MVP é considerado pronto quando **todos** os itens abaixo forem verdadeiros:

1. **CP16 concluído** — feed vertical é a home, lista tradicional preservada,
   Markdown sanitizado, acessibilidade e performance validadas.
2. **CP17 concluído** — leitura de e-mail, matching por cascata, pipeline
   `in_process`, tags `phase:` e transições monotônicas funcionando.
3. **Backlog concluído ou explicitamente decidido** — todos os itens B1–B12 e a
   revisão de UI do CP6 entregues, ou com decisão *go/no-go* registrada (B5/B7).
4. **Suíte verde** — `make test`, `make vet` e `make test-ui` passando no CI.
5. **Migrations validadas** — banco novo e upgrade de um banco anterior sobem sem
   perda de dados.
6. **Aceite manual** — fluxo completo no celular e no desktop: descobrir vaga →
   pontuar → adaptar currículo → candidatar → receber e-mail → estágio muda →
   notificação.
7. **Documentação** — ADRs numerados sem colisão, `docs/plan/README.md`,
   `docs/backlog.md` e notas de release atualizados.

---

## 2. Mapa de absorção — backlog dentro dos checkpoints

Só entram no checkpoint os itens que **compartilham código/rotas** com ele. O
restante vira fase própria.

| Item de backlog | Absorvido em | Por quê |
|---|---|---|
| **UI Revision (CP6)** — alinhar descoberta/fontes/settings ao design system | **F1 / CP16** | CP16 muda navegação, home e componentes de `jobs`; a revisão de UI de descoberta/fontes usa as mesmas rotas, `lib/ui` e testes visuais. Fazer junto evita retrabalho de layout. |
| **B6** — erros amigáveis de provedor de IA | **F2 / CP17** | CP17 introduz novas superfícies de erro (sync de e-mail + LLM na cauda). B6 cria o catálogo de erros amigáveis que o CP17 consome. |
| **B12** — automação de e-mail + pipeline/tags | **F2 / CP17** | É o próprio checkpoint. |

> **B8** (erros amigáveis no tailoring/upload) **não** entra no CP17: ele reusa o
> catálogo criado em F2, mas atua no currículo. Fica na F3.

---

## 3. Visão geral das fases

| Fase | Conteúdo | Depende de | Duração | Gate |
|---|---|---|---|---|
| **F0** | Preparação e normalização | — | 0,5–1 dia | Baseline + ADRs + flags |
| **F1** | CP16 + UI Revision (CP6) | F0 | 7–9 dias | Feed + a11y + performance |
| **F2** | CP17 + B6 | F0, F1 | 7–9 dias | E-mail + pipeline + privacidade |
| **F3** | B10 + B8 + B9 (currículo) | F2 (catálogo de erros) | 2–4 dias | Exportação + erros + cópia |
| **F4** | B1 + B3 + B4 + B2 (UX de IA) | F2 | 5–7 dias | Onboarding + presets + perfis |
| **F5** | B11 (backend determinístico) | F1 | 4–6 dias | Determinismo + performance |
| **F6** | B7 (LinkedIn OAuth) | F4 | 5–8 dias | Spike go/no-go + importação |
| **F7** | B5 (OAuth de provedores) | F4 | 5–8 dias | Spike go/no-go + conexão |
| **F8** | Hardening e release | F1–F7 | 2–3 dias | Regressão completa |

**Total estimado:** 33–55 dias úteis (7–11 semanas), dependendo dos gates de F6/F7.

> As fases F3, F4 e F5 são independentes entre si e podem ser paralelizadas por
> pessoas diferentes. F6 e F7 só começam depois de F4 (dependem da UX de perfis).

---

## 4. Fase 0 — Preparação e normalização (0,5–1 dia)

**Objetivo:** deixar o terreno pronto para as fases seguintes sem misturar
refatoração de infraestrutura com feature.

**Tarefas:**
- Confirmar baseline: `make test`, `make vet` e `make test-ui` verdes antes de
  qualquer mudança.
- **Normalizar ADRs:** havia colisão de numeração entre CP15, CP16 e CP17.
  - CP15 usa `0026` e `0027`.
  - **Decisão já aplicada nos planos:** CP16 usa `0028`–`0031`; CP17 usa
    `0032`–`0035`. F0 apenas confirma que nenhum ADR novo colide.
- Definir **feature flags** (settings ou env) para: feed (`ui.feed`), sync de
  e-mail (`email.sync`), exportação (`curriculum.export`) e refactor de scoring
  (`score.deterministic`). Flags permitem merge sem expor comportamento instável.
- Definir estratégia de branch/merge: uma branch por fase, merge somente com o
  gate da fase atendido.
- Congelar o escopo: qualquer item novo vai para `docs/backlog.md`, não para o
  release.

**Gate:**
- Baseline verde.
- ADRs renumerados e planos atualizados.
- Flags documentadas em `/settings` ou no `README`.

**Status (2026-11-22): concluída.**
- `make vet`, `make test` e `make test-ui` verdes (snapshots de
  `/settings/sources` atualizados para a UI do CP14; fixture de fontes passou a
  incluir `health`/consumo).
- ADRs confirmados: CP15 `0026–0027`, CP16 `0028–0031`, CP17 `0032–0035`.
- Flags implementadas como env vars (`OPENJOB_FEATURE_*`), expostas em
  `GET /api/features` e documentadas no `README.md`/`.env.example`.
- Estratégia: uma branch por fase (`release/f1-cp16`, `release/f2-cp17`, …),
  merge em `main` somente com o gate da fase atendido. Escopo congelado: itens
  novos vão para `docs/backlog.md`.

---

## 5. Fase 1 — CP16 + UI Revision (CP6) (7–9 dias)

**Objetivo:** transformar o feed vertical na home e alinhar a UI de descoberta e
fontes ao design system, sem quebrar as rotas existentes.

**Escopo:**
- Feed vertical de tela cheia em `/jobs` (scroll-snap, janela deslizante,
  carregamento infinito, estados de loading/vazio/erro).
- Lista tradicional preservada em `/jobs/list` (ou toggle Feed/Lista).
- Detalhe em `/jobs/[id]` com Markdown sanitizado e botão “Voltar ao feed”.
- Botão “Buscar agora” integrado ao feed (dispara discovery do CP6/CP14).
- Navegação adaptada por dispositivo (bottom nav no mobile, sidebar no desktop).
- **UI Revision (CP6):** alinhar `settings/sources`, `settings/discovery` e
  controles de descoberta aos componentes/tokens do CP12; badges, cards, modais,
  foco acessível e responsividade.
- Atualizar `frontend/src/routes/+layout.svelte` e a hierarquia de navegação
  (feed como item principal; dashboard do CP15 em `/dashboard`).

**Fora de escopo:** algoritmo de recomendação, mídia/vídeo, edição inline.

**Arquivos principais:**
- `frontend/src/routes/jobs/+page.svelte` (feed)
- `frontend/src/routes/jobs/list/+page.svelte` (lista)
- `frontend/src/routes/jobs/[id]/+page.svelte` (detalhe)
- `frontend/src/lib/components/` (`JobFeed`, `JobFeedCard`)
- `frontend/src/routes/settings/sources/+page.svelte`
- `frontend/src/routes/settings/discovery/+page.svelte`
- `frontend/src/routes/+layout.svelte`
- Testes em `frontend/tests/{interaction,a11y,visual}/`

**Testes e verificação:**
- `make test-ui` completo (check, i18n, lint de UI, e2e, a11y, visual).
- Visual: 375×667, 768×1024, 1440×900 (feed, lista, detalhe, settings de fontes).
- Acessibilidade: `role="feed"`, `inert` nos cards fora da tela, teclado ↑/↓,
  anúncio de posição, `prefers-reduced-motion`.
- Segurança: descrição com `<script>`/`javascript:` não executa (teste XSS).
- Performance: DOM com no máximo ~3–5 cards; primeira página < 1 s em rede local.
- Manual no celular: abrir o app cai no feed, rolar, “Saiba mais”, voltar, buscar
  agora.

**Gate:**
- Home é o feed; lista continua acessível.
- Nenhuma regressão em candidaturas, notificações, tarefas e settings.
- UI de descoberta/fontes consistente com o design system.

**Riscos:** snap + leitor de tela (mitigar com modo lista), perda de foco na
reciclagem de nós (manter item ativo montado), Markdown de fontes não confiáveis
(sanitização no servidor + cliente).

**Status (2026-11-22): concluída.**
- Feed vertical de tela cheia implementado em `/jobs` e definido como home (`/` e `/jobs`), com scroll-snap nativo (`scroll-snap-type: y mandatory`), janela deslizante (≤ 3 cards no DOM com reciclagem de slots) e botão “Buscar agora” integrado ao topo com feedback não bloqueante via toast.
- Lista tradicional preservada em `/jobs/list` com alternância fluida entre Feed e Lista no header.
- Rota dedicada de detalhe em `/jobs/[id]` com renderização segura em Markdown contra XSS (escape rigoroso e allowlist de links) e botão “Voltar ao feed”.
- UI Revision (CP6) concluída: `/settings/sources` e `/settings/discovery` alinhadas aos tokens e componentes do design system CP12 (cards, modais, badges, foco acessível e responsividade).
- Dashboard do CP15 reorganizado em `/dashboard`.
- Navegação adaptada: sidebar no desktop e bottom nav com 4 itens no mobile.
- ADRs 0028, 0029, 0030 e 0031 registrados sem colisões.
- Gate e suíte de testes 100% verdes: `make vet`, `make test`, `make build` e `make test-ui` (check, i18n, lint de UI, e2e/interação, a11y axe-core e testes visuais com snapshots para smartphone 375×667, tablet 768×1024 e desktop 1440×900).

---

## 6. Fase 2 — CP17 + B6 (7–9 dias)

**Objetivo:** automatizar a atualização de estágio por e-mail e reformular o
pipeline com `in_process` e tags de fase, com erros amigáveis desde o início.

**Escopo:**
- Tabelas `email_accounts`, `email_messages`, `tags`, `application_tags`.
- `applications`: `stage_group`, `stage_label`, `stage_updated_at`,
  `stage_source` (migração de `status`).
- Conector Gmail (OAuth read-only, `history.list`) e IMAP genérico
  (`emersion/go-imap` v2 + parser MIME), sync incremental por cursor.
- Cascata de matching: thread → nome da empresa → allowlist de ATS → domínio
  corporativo → LLM na cauda.
- Classificador com vocabulário fechado (`ack`, `screening`, `interview`,
  `assessment`, `offer`, `rejection`, `info_request`, `other`).
- Handler `sync_email` na fila do CP10, ticker desligado por padrão.
- API: `GET/PUT /api/settings/email`, `POST /api/settings/email/test`,
  `GET /api/email/messages`, `POST /api/email/messages/{id}/link`,
  `PATCH /api/applications/{id}` com `stage_group`/`stage_label`/`tags`.
- UI: pipeline com 6 colunas, coluna “Em processo”, chips de tags, banner de
  sugestões de baixa confiança, drawer de e-mails vinculados, `/settings/email`.
- **B6:** catálogo de erros amigáveis de provedor (401/403, 404, 429, 5xx,
  timeout, decrypt), sem expor payload/stack; reutilizar `internal/llm/redact.go`
  e os códigos de erro do ADR 0021.

**Fora de escopo:** IMAP IDLE, Pub/Sub/webhook, envio de e-mail, múltiplas contas,
automação de candidatura, criação de eventos de calendário.

**Arquivos principais:**
- `backend/internal/store/migrations/0006_email_tracking.sql`
- `backend/internal/email/` (conector, match, classify)
- `backend/internal/tasks/` (handler `sync_email`)
- `backend/internal/api/` (settings de e-mail, mensagens, applications)
- `backend/internal/llm/` (erros amigáveis — B6)
- `frontend/src/routes/applications/+page.svelte`
- `frontend/src/routes/settings/email/+page.svelte`
- `frontend/src/lib/components/` (pipeline, tags, sugestões)

**Testes e verificação:**
- Go: `go test ./internal/email/... ./internal/tasks/... ./internal/api/...`
  com fixtures de Greenhouse, Lever, Workday, Ashby e domínio corporativo.
- Migração: banco novo e upgrade de banco com `applications.status` antigo.
- Monotonicidade: rejeição seguida de confirmação não retrocede.
- Dedup por `Message-ID`; sync incremental não reprocessa.
- Privacidade: corpo truncado, log sem corpo, LLM só com BYOK configurado.
- UI: sugestão de baixa confiança aparece, confirma e rejeita; filtro por tag.
- Regressão: Telegram do CP7 e notificações do CP10 continuam funcionando.
- `make test` + `make test-ui`.

**Gate:**
- E-mail de confirmação move `pending → in_process` com `email_auto`.
- Caso ambíguo vira sugestão, não transição automática.
- Pipeline com 6 colunas e tags funcionando.
- Nenhum segredo ou corpo de e-mail em log.

**Riscos:** matching por nome genérico (exigir confiança + thread), alucinação do
LLM na cauda (vocabulário fechado + revisão), OAuth do Gmail exigir app no Google
Cloud (documentar criação do client_id; IMAP como alternativa).

**Status (2026-11-23): concluída.**
- Migração `0006_email_tracking.sql` cria `email_accounts`, `email_messages`,
  `tags` e `application_tags`, e adiciona `stage_group`, `stage_label`,
  `stage_updated_at` e `stage_source` a `applications` (migrando
  `sent → in_process`).
- Conectores Gmail (OAuth read-only via `users.history.list`, com fallback para
  listagem completa) e IMAP (`emersion/go-imap` v2, cursor `UID > last_uid` e
  parser MIME); credenciais cifradas em AES-256-GCM.
- Cascata de matching thread → nome da empresa → allowlist de ATS → domínio
  corporativo → LLM na cauda; classificador com vocabulário fechado (8 sinais)
  e regras PT/EN/ES; dedup por `Message-ID`; avanço monotônico apenas com
  confiança ≥ 0.8 (ADRs 0032–0035).
- Handler `sync_email` no pool do CP10 com progresso (`emails_fetched`,
  `matched`, `advanced`, `suggested`); ticker opt-in (`email.sync_interval`,
  padrão 5 min, inativo até uma conta ser habilitada) e sincronização manual em
  `/settings/email`.
- API: `GET/PUT /api/settings/email`, `POST /api/settings/email/test`,
  `POST /api/settings/email/sync`, `GET /api/email/messages`,
  `POST /api/email/messages/{id}/link|unlink`, `GET/POST /api/tags` e
  `PATCH /api/applications/{id}` com `stage_group`, `stage_label` e `tag_ids`.
- UI: pipeline com as 6 colunas (coluna “Em processo” mais larga), chips de tags,
  filtro por tag, banner de sugestões com confirmar/descartar, drawer com
  e-mails vinculados e `/settings/email` (IMAP/Gmail, status e aviso de
  privacidade).
- B6: catálogo `internal/llm/errors.go` com os códigos do ADR 0021 e mensagens
  amigáveis + ação sugerida para 401/403, 404, 429, 5xx, timeout, rede, decrypt
  e não configurado, reutilizando `internal/llm/redact.go`.
- Testes: fixtures de matching/classificação, monotonicidade, dedup, sync
  incremental, API de e-mail/tags, scheduler e migração; e2e CP17 (sugestão,
  tags e settings) e a11y/visual de `/settings/email`.
- Gate e suíte verdes: `make vet`, `make test` e `make test-ui`.

---

## 7. Fase 3 — Currículo: B10 + B8 + B9 (2–4 dias)

**Objetivo:** fechar o ciclo do currículo adaptado (copiar, exportar) e padronizar
erros amigáveis no upload/tailoring.

**Escopo:**
- **B10:** botão discreto de cópia na área de visualização/diff do currículo
  adaptado, com tooltip e feedback “Copiado!”.
- **B8:** aplicar o catálogo de erros da F2 ao upload, conversão de PDF e
  tailoring; backend loga detalhe, UI mostra causa provável + ação.
- **B9:** exportação do currículo adaptado em **PDF**, **TXT** e **MD**:
  - TXT limpo, sem marcação Markdown.
  - PDF formatado para envio (backend ou CSS de impressão).
  - MD mantém o download atual.

**Arquivos principais:**
- `frontend/src/routes/curriculum/+page.svelte`
- `backend/internal/curriculum/` (exportação)
- `backend/internal/api/curriculum.go`
- `frontend/src/lib/i18n` / `frontend/messages` (textos de erro/exportação)

**Testes e verificação:**
- E2E: copiar adaptado → clipboard contém o Markdown; feedback visível.
- Exportar PDF/TXT/MD → arquivo baixa, abre e mantém conteúdo/estrutura.
- Erros: simular PDF inválido e provedor 401/429 → mensagem amigável, sem stack.
- `make test` + `make test-ui` (a11y e i18n incluídos).

**Gate:** usuário consegue copiar e exportar o currículo adaptado nos 3 formatos,
e nenhum erro de upload/tailoring expõe detalhe interno.

---

## 8. Fase 4 — UX de IA: B1 + B3 + B4 + B2 (5–7 dias)

**Objetivo:** reduzir a fricção de configuração de IA e permitir múltiplos perfis
com seleção do ativo.

**Sequência interna (cada passo testável):**
1. **B1 — Onboarding de chave:** explicação curta de BYOK, links oficiais por
   provedor, nota de segurança (AES-GCM, chave nunca volta ao navegador).
2. **B3 — Presets homologados:** catálogo versionado no backend
   (`internal/llm/presets.go`) exposto por `GET /api/settings/llm/presets`;
   prefill de `base_url` e sugestão de modelo, sem travar edição.
3. **B4 — Free tier:** marcar presets gratuitos, sugerir modelo default gratuito
   e exibir “gratuito — sujeito a rate limit”.
4. **B2 — Múltiplos perfis:** tabela/migration de perfis cifrados, CRUD,
   `POST /api/settings/llm/activate`, migração da config plana atual para o
   primeiro perfil, badge “ativo” e teste por perfil.

**Arquivos principais:**
- `backend/internal/llm/presets.go`
- `backend/internal/api/settings.go`
- `backend/internal/store/` (migration de perfis)
- `frontend/src/routes/settings/+page.svelte` e sub-rotas de IA
- `frontend/messages` (i18n)

**Testes e verificação:**
- API: criar/editar/ativar/testar perfis; chave nunca ecoada.
- Migration: config antiga vira primeiro perfil sem perda.
- E2E: “começar de graça” em ≤ 3 toques; trocar perfil ativo reflete em
  tailor/scoring sem restart.
- `make test` + `make test-ui`.

**Gate:** usuário leigo configura uma chave e começa a usar; usuário avançado
alterna perfis sem reconfigurar.

**Status (2026-11-23): concluída.**
- **B1:** `/settings` abre em estado de onboarding quando não há provedor, com
  explicação de BYOK e atalho “Começar de graça”; a chave continua write-only e
  cifrada em AES-GCM.
- **B3:** catálogo versionado em `backend/internal/llm/presets.go` exposto por
  `GET /api/settings/llm/presets`, com `base_url`, modelos disponíveis, link de
  documentação e link oficial de criação de chave por provedor. A UI consome o
  catálogo e nunca hardcodeia provedores.
- **B4:** presets free tier marcados (`free_rate_limited`/`free_local`) com nota
  honesta de limite e sugestão de modelo gratuito; Ollama/LM Studio não exigem
  chave.
- **B2:** tabela `llm_profiles` (migração `0007`) com CRUD, ativação atômica e
  espelhamento em `llm.*`; migração da config plana para o primeiro perfil é
  idempotente e a chave nunca é ecoada.
- UI: lista de perfis com badge “Ativo”, testar/editar/remover por perfil e
  troca de ativo em um toque. O formulário usa seletor de preset, seletor de
  modelos do preset (texto simples apenas em “Personalizado”) e card de ajuda
  do provedor com documentação + criação de chave, oculto no modo custom.
- i18n: “Personalizado” e todos os textos de perfis/ajuda traduzidos em
  `pt-BR`, `en-US` e `es-419`.
- ADR 0036 registra a decisão de perfis cifrados, catálogo versionado e
  ativação espelhada em `llm.*`.
- Testes: API de presets/perfis (chave nunca ecoada, validação, ativação e
  migração), catálogo de presets (modelos/docs), e2e de seletor de modelos,
  card de ajuda, cadastro e troca de perfil ativo; a11y e snapshots visuais de
  `/settings` atualizados.
- Gate e suíte verdes: `make vet`, `make test` e `make test-ui`.

---

## 9. Fase 5 — Backend determinístico: B11 (4–6 dias)

**Objetivo:** remover LLM de features determinísticas (score, keywords, filtro),
mantendo contratos de API e explicabilidade.

**Escopo:**
- `score_fit`: BM25 (Okapi, `k1=1.2`, `b=0.75`) entre currículo e descrição.
- Keywords do currículo: YAKE (fallback RAKE), por seção, top-N configurável.
- Filtro por currículo: TF-IDF + cosine ou BM25 com limiar configurável
  (equivalência com `min_matches` documentada).
- Remover chamadas LLM dos handlers `score_job` e `extract_resume_keywords`;
  manter endpoints e formato de resposta.
- Feature flag `score.deterministic` para rollback.

**Fora de escopo:** tailoring, carta e análise de currículo continuam com LLM.

**Testes e verificação:**
- Golden tests: mesma entrada → mesma saída.
- Nenhuma chamada de rede no scoring (mock/contador zerado).
- Performance: < 50 ms por vaga.
- Multilíngue: PT-BR, EN-US, ES-419.
- Regressão: ranking do feed e filtros continuam coerentes.
- `make test` + `make vet`.

**Gate:** scoring determinístico, auditável e sem LLM; funcionalidade preservada.

---

## 10. Fase 6 — LinkedIn OAuth: B7 (5–8 dias, com gate)

**Objetivo:** importar o perfil do LinkedIn para o currículo-base, se viável.

**Etapa 1 — Spike (1–2 dias):**
- Verificar acesso à Member Data Portability API / LinkedIn Partner Program.
- Confirmar se o escopo disponível retorna histórico profissional completo ou
  apenas `openid profile email`.
- **Gate go/no-go:** se não houver acesso viável, registrar decisão, manter o
  fallback (upload de PDF/MD/TXT) e encerrar o item como “não viável no MVP”.

**Etapa 2 — Implementação (4–6 dias, se go):**
- Botão “Importar do LinkedIn” em `/curriculum`.
- Fluxo OAuth 2.0, callback local, extração para Markdown estruturado.
- Fallback explícito para upload manual.

**Testes e verificação:**
- Fluxo OAuth com conta de teste; erro/expiração cai no fallback.
- Currículo importado editável e persistido em `curriculum.content_md`.
- Nenhum token em log; revogação suportada.

**Gate:** importação funciona ou decisão no-go documentada; em ambos os casos o
upload manual continua funcional.

---

## 11. Fase 7 — Hardening e release (2–3 dias)

**Objetivo:** validar o conjunto e publicar o MVP.

**Tarefas:**
- Regressão completa: `make test`, `make vet`, `make test-ui`.
- Migração: banco novo + upgrade de banco da versão anterior.
- Instalação limpa: binário único, bootstrap mínimo, pacote .deb, imagem Docker publicada e Docker Compose; acesso LAN pelo celular.
- Backup/restore do SQLite e da master key documentados.
- Revisão de segurança: segredos, logs, redaction, permissões de arquivo.
- Revisão de acessibilidade final (axe sem violações sérias).
- Atualizar `README.md`, `docs/plan/README.md`, `docs/backlog.md` e notas de
  release; confirmar ADRs sem colisão.
- Tag de release e checklist de publicação.

**Gate de release:**
- Todos os critérios da seção 1 atendidos.
- Nenhum item de backlog aberto sem decisão registrada.
- Fluxo ponta a ponta validado em celular e desktop.

---

## 12. Matriz de rastreabilidade

| Item | Fase | Teste principal |
|---|---|---|
| CP16 — feed vertical | F1 | Visual + a11y + performance + XSS |
| UI Revision (CP6) | F1 | Visual + a11y de sources/discovery |
| CP17 — e-mail/pipeline/tags | F2 | Fixtures de ATS + migração + monotonicidade |
| B6 — erros de provedor | F2 | API 401/404/429/5xx/timeout |
| B12 — automação de e-mail | F2 | (é o CP17) |
| B10 — botão de cópia | F3 | E2E clipboard |
| B8 — erros de currículo | F3 | E2E PDF inválido + provedor |
| B9 — exportação PDF/TXT/MD | F3 | Download + conteúdo |
| B1 — onboarding de IA | F4 | E2E settings |
| B3 — presets | F4 | API presets + prefill |
| B4 — free tier | F4 | E2E “começar de graça” |
| B2 — múltiplos perfis | F4 | API CRUD + ativação + migração |
| B11 — backend determinístico | F5 | Golden + determinismo + <50 ms |
| B7 — LinkedIn OAuth | F6 | Spike go/no-go + OAuth |
| B5 — OAuth de provedores | F7 | Spike go/no-go + OAuth |
| Release | F8 | Regressão + instalação limpa |

---

## 13. Riscos transversais e mitigação

| Risco | Impacto | Mitigação |
|---|---|---|
| Colisão de ADRs (CP15/16/17) | Documentação inconsistente | F0 renumera antes de codar |
| Feed quebrar rotas existentes | Regressão de navegação | Lista preservada + testes de regressão |
| Sync de e-mail gerar falsos positivos | Pipeline perde confiança | Confiança ≥ 0.8, sugestão, monotonicidade |
| OAuth externo inviável | Bloqueio de B5/B7 | Spike com gate go/no-go; fallback manual |
| Refactor B11 alterar ranking | Perda de qualidade percebida | Feature flag + golden tests + comparação |
| Escopo crescer durante o release | Atraso | Itens novos vão para backlog, não para fase |

---

## 14. Ordem de merge e liberação

1. F0 (infra/documentação) — merge direto.
2. F1 (CP16) — merge com flag `ui.feed`; liberar para uso interno.
3. F2 (CP17) — merge com flag `email.sync` desligada por padrão.
4. F3, F4 e F5 — podem ser mescladas em qualquer ordem após F2/F1.
5. F6 e F7 — somente após os spikes; se no-go, registrar e não bloquear.
6. F8 — habilitar flags por padrão, validar e taguear o MVP.


---

## 15. Evolução pós-MVP — CP18 e CP19

As features seguintes são uma evolução posterior ao escopo congelado deste plano de release; não alteram os gates F0–F8 do MVP.

- **CP18:** identidade, autenticação, isolamento multiusuário, perfil profissional e currículo-base; OAuth somente conforme viabilidade e escopos oficiais.
- **CP19:** onboarding gráfico de primeira execução na webapp, com validação real e etapas opcionais não bloqueantes.

A distribuição e o bootstrap mínimo permanecem no CP11; a sequência, dependências e critérios de CP18/CP19 ficam em docs/plan/README.md e nos planos individuais.
