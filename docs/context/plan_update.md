## 🎯 O que muda e o que permanece

| Camada | Antes (produto) | Agora (uso pessoal) |
|---|---|---|
| **Descoberta de vagas** | Servidor externo + feed curado | Scraping local, rodando no seu laptop quando ele está ligado |
| **Licenciamento/pagamento** | Merchant of Record, PPP, pricing | Removido. Sem verificação, sem trial, sem checkout |
| **Criptografia BYOK** | Senha mestra derivada (PBKDF2) | Chave local em arquivo `0600` (mais simples, suficiente para uso pessoal) |
| **Multi-tenancy** | `tenant_id` em todas as tabelas | Sem `tenant_id` agora. Adicionar depois é trivial com migrations |
| **ATS assistido** | Checkpoint 7 obrigatório | **Opcional e no fim**. Para uso pessoal, o "copiar/colar" do formulário resolve |
| **DNS local (mDNS)** | Necessário para celular | **Mantido** — você vai querer revisar vagas no celular no sofá |
| **Telegram** | Canal principal de notificação | **Mantido** — é a forma mais prática de receber alertas quando o laptop está fechado |
| **Backend Go + chi + embed** | Mantido | Mantido (arquitetura limpa facilita productização) |
| **SQLite + Repository** | Mantido | Mantido (interface pronta para PostgreSQL futuro) |
| **SvelteKit + PWA** | Mantido | Mantido (mobile-first continua sendo o modo de uso principal) |
| **BYOK multi-provider** | llmux/any-llm-go | **Simplificado**: cliente OpenAI-compatible com `base_url` configurável (cobre OpenAI, Groq, Ollama, Together e Anthropic via endpoint compatível) |

**Resumo:** você corta ~40% do escopo e ganha ~3 semanas de velocidade até o primeiro uso real.

---

## 🏗️ Arquitetura simplificada

```text
.
├── backend/
│   ├── cmd/server/main.go
│   ├── internal/
│   │   ├── api/          # handlers HTTP (chi)
│   │   ├── llm/          # cliente OpenAI-compatible + abstração
│   │   ├── store/        # SQLite + interface Repository
│   │   ├── tailor/       # adaptação de currículo
│   │   ├── scraper/      # Remotive, WWR, RemoteOK (roda sob demanda)
│   │   └── notify/       # Telegram Bot
│   ├── web/              # SPA compilada (embed)
│   └── go.mod
├── frontend/             # SvelteKit + PWA
├── Dockerfile
├── docker-compose.yml
└── data/
    ├── app.db            # SQLite
    └── .master.key       # chave AES-GCM (chmod 0600)
```

**Decisões de simplificação:**

- **LLM client**: em vez de `llmux` ou `any-llm-go`, use um cliente fino sobre a API OpenAI-compatible (`github.com/sashabaranov/go-openai` ou `net/http` puro). Você configura `base_url` + `api_key` + `model`. Funciona com OpenAI, Groq, Ollama, Together, Fireworks e até Anthropic (que oferece endpoint compatível). Isso cobre 95% dos casos de uso pessoal e reduz dependências.
- **Scraper local**: use `github.com/gocolly/colly` ou `net/http` + `goquery`. Roda sob demanda via botão na UI ("Buscar agora") ou via `time.Ticker` a cada 6h enquanto o laptop estiver ligado. Não precisa de cron do sistema; o servidor Go gerencia.
- **Criptografia BYOK**: na primeira execução, gere um arquivo `data/.master.key` com 32 bytes aleatórios e `chmod 0600`. Use AES-GCM para cifrar as chaves de API no SQLite. Documente no README: *"este arquivo é o segredo do seu cofre local; faça backup dele junto com o app.db, nunca o commite"*.
- **mDNS**: mantenha o sidecar `skhmt/mdns` com `network_mode: host`. No Linux funciona nativo; no Mac/Windows, o Bonjour do host resolve `maicon-cv.local` automaticamente. Documente os dois caminhos.
- **Sem plataformas gerenciáveis na UI**: no MVP pessoal, as fontes de scraping são configuradas por arquivo `config.yaml` ou variáveis de ambiente. A UI de "habilitar/desabilitar plataformas" fica para a fase de productização.

---

## 🗄️ Schema SQLite revisado

Sem `tenant_id`, sem `platforms`, sem `notifications_log` separado (pode usar `jobs` + `applications` com timestamps). Mais enxuto:

```sql
-- settings: chave-valor (inclui chaves BYOK cifradas)
CREATE TABLE settings (
  key TEXT PRIMARY KEY,
  value TEXT NOT NULL,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- curriculum: currículo base (uma linha ativa por vez)
CREATE TABLE curriculum (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  content_md TEXT NOT NULL,
  source_filename TEXT,
  uploaded_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- jobs: vagas descobertas ou coladas manualmente
CREATE TABLE jobs (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  title TEXT NOT NULL,
  company TEXT NOT NULL,
  url TEXT UNIQUE,
  description TEXT NOT NULL,
  source TEXT NOT NULL,          -- 'remotive', 'wwr', 'remoteok', 'manual'
  score_fit INTEGER,             -- 0-100, NULL até ser calculado
  score_reason TEXT,             -- justificativa da IA
  status TEXT NOT NULL DEFAULT 'new',  -- new | approved | discarded
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- applications: candidaturas aprovadas/enviadas
CREATE TABLE applications (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  job_id INTEGER NOT NULL REFERENCES jobs(id),
  tailored_curriculum_md TEXT,
  status TEXT NOT NULL DEFAULT 'pending',  -- pending | sent | interview | rejected | offer
  sent_at DATETIME,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- notify_log: histórico de notificações enviadas
CREATE TABLE notify_log (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  type TEXT NOT NULL,            -- 'daily_report', 'status_change', 'test'
  payload TEXT,
  sent_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

**Compatibilidade futura com PostgreSQL:** use `TEXT`/`INTEGER`/`DATETIME` no SQLite e mapeie para `VARCHAR`/`BIGSERIAL`/`TIMESTAMPTZ` no Postgres via interface Repository. Nada de `AUTOINCREMENT` hardcoded — encapsule no store.

---

## 🗺️ Roadmap revisado (checkpoints)

### ✅ Checkpoint 1 — Fundação (2–3 dias)

**Entregáveis:**
- Backend Go + chi servindo "Hello World" via `//go:embed`.
- `Dockerfile` multi-stage e `docker-compose.yml` com porta 8080.
- Sidecar mDNS para `maicon-cv.local`.
- Acesso pelo celular confirmado.

**Verificação:** abrir `http://maicon-cv.local:8080` no celular na mesma rede.

---

### ✅ Checkpoint 2 — SPA + PWA (2–3 dias)

**Entregáveis:**
- SvelteKit com `adapter-static` e `fallback: 'index.html'`.
- Build copiado para `backend/web/dist`, embutido no binário.
- Rotas `/`, `/settings`, `/curriculum`, `/jobs`, `/applications` funcionando com reload.
- PWA com manifest + service worker (`@vite-pwa/sveltekit`).
- Safe areas iOS (`env(safe-area-inset-*)`).

**Verificação:** navegar, recarregar, instalar PWA na tela inicial do celular.

---

### ✅ Checkpoint 3 — SQLite + BYOK (3–4 dias)

**Entregáveis:**
- SQLite com migrations embutidas (`modernc.org/sqlite` + `golang-migrate` ou SQL puro no embed).
- Geração automática de `data/.master.key` na primeira execução.
- `POST /api/settings/llm-key` (cifra e persiste), `GET /api/settings/llm-key` (retorna `{configured: bool}`).
- `POST /api/settings/llm-ping` — chamada de teste ao provedor configurado.
- UI de settings com campos `base_url`, `model`, `api_key` (mascarado).

**Verificação:** salvar chave, reiniciar container, ping retornar sucesso.

---

### ✅ Checkpoint 4 — Currículo + Adaptação (3–4 dias)

**Entregáveis:**
- `POST /api/curriculum/upload` — aceita PDF/MD/TXT, converte para Markdown estruturado (use `github.com/ledongthuc/pdf` para PDF; MD/TXT direto).
- `POST /api/curriculum/tailor` — recebe `job_id` (ou texto de JD colado) e retorna currículo adaptado em Markdown.
- Prompt de tailoring versionado em `internal/tailor/prompts.go` com instruções de alinhamento de palavras-chave e conquistas.
- UI para upload, visualização e diff (base vs. adaptado).

**Verificação:** subir seu currículo real, colar uma JD que você está mirando, gerar versão adaptada e ler criticamente. Este é o **primeiro momento de valor real**.

---

### ✅ Checkpoint 5 — Entrada manual de vagas + scoring (1–2 dias)

**Entregáveis:**
- `POST /api/jobs/manual` — recebe `{title, company, url, description}` e persiste.
- `POST /api/jobs/:id/score` — calcula `score_fit` via IA, com `score_reason`.
- UI: cole uma JD, clique "score", veja ranking.
- `POST /api/jobs/:id/approve` — cria `application` com status `pending`.

**Verificação:** você cola a JD da vaga que está mirando, recebe score e justificativa, aprova, e a candidatura entra no pipeline. **A partir daqui você já pode usar o produto de verdade.**

---

### ✅ Checkpoint 6 — Descoberta automática (3–4 dias)

**Entregáveis:**
- Scrapers para Remotive, WWR e RemoteOK (públicas, sem login).
- `POST /api/jobs/discover` — dispara scraping sob demanda.
- `time.Ticker` opcional a cada 6h (configurável, desabilitado por padrão).
- Deduplicação por `url` (UNIQUE constraint).
- Scoring automático em batch (rate-limited para não estourar API).

**Verificação:** clicar "Buscar agora", ver 30–80 vagas novas, com score calculado, ordenadas por `score_fit`.

---

### ✅ Checkpoint 7 — Telegram + tracking (2–3 dias)

**Entregáveis:**
- `POST /api/notify/test` — mensagem de teste.
- Cron diário (via `robfig/cron` ou `time.Ticker`) às 8h enviando relatório: novas vagas, pendentes de aprovação, enviadas, em andamento.
- `PATCH /api/applications/:id` — atualiza status; dispara notificação imediata se virar `interview` ou `rejected`.
- UI para configurar token + chat_id.

**Verificação:** receber mensagem de teste; no dia seguinte, receber o relatório.

---

### ✅ Checkpoint 8 — Validação pessoal (2–4 semanas, sem código novo)

**Ação:** você usa o app de verdade. Aplica para 5–15 vagas com currículos adaptados, acompanha o pipeline, recebe notificações. **Registre:**
- Quantas vagas você aprovou vs. descartou?
- Quantas geraram resposta?
- O tailoring está bom o suficiente ou você edita manualmente?
- O score de fit está calibrado ou está tudo 70–80?
- O que você sente falta toda vez que abre o app?

**Critério de verificação:** você tem dados reais para decidir se vale productizar. Não avance sem isso.

---

### ⚠️ Checkpoint 9 (opcional) — ATS assistido

**Só faça se, na validação, você sentir dor real de preencher formulários Greenhouse/Lever manualmente.** Playwright headless, preenche até antes do submit, você revisa e envia. LinkedIn nunca é automatizado.

---

## 🧪 Fase de validação: o que medir

Antes de productizar, responda com dados:

| Pergunta | O que indica |
|---|---|
| O tailoring gerou entrevistas? | Se sim, a IA está funcionando; se não, prompt precisa de iteração |
| Você usou o app mais de 10 vezes em 2 semanas? | Se não, o problema é seu, não do mercado |
| Alguma feature foi ignorada? | Corte antes de productizar |
| O Telegram foi útil ou irritante? | Ajuste a frequência |
| Qual foi o maior atrito no uso diário? | Esse é o próximo feature, não o que você planejou |

**Decisão de productização:** se em 4 semanas você aplicou para 10+ vagas e teve 1+ entrevista, o núcleo tem valor. Se você parou de usar, o problema é mais profundo do que "faltou productização".

---

## 💾 O que preservar para o futuro (não jogar fora)

Mesmo sendo uso pessoal, **não corte estas coisas** — o custo de adicionar depois é muito maior do que manter agora:

1. **Interface `Repository` no store** — permite trocar SQLite por PostgreSQL sem tocar em handlers.
2. **Abstração `LLMClient`** — permite trocar OpenAI-compatible por outro SDK se necessário.
3. **Migrations versionadas** — mesmo que você seja o único usuário, elas evitam "drop table" no futuro.
4. **ADRs curtos** — documente as decisões (por que chave local e não senha mestra; por que sem `tenant_id` agora; por que mDNS e não IP). Esses docs valem ouro quando você productizar.
5. **Testes nos pontos críticos** — criptografia de BYOK, parsing de JD, deduplicação de vagas. Não precisa de cobertura alta, mas esses três quebram silenciosamente.
6. **Estrutura de pastas já preparada para `internal/tenant/`** — mesmo que vazia hoje, evita refatoração estrutural depois.

---

## 💰 Custos operacionais no modo pessoal

| Item | Custo |
|---|---|
| Infraestrutura | **R$ 0** (seu laptop) |
| Banco | **R$ 0** (SQLite local) |
| IA | **BYOK** — você paga direto ao provedor (ex.: ~US$ 0,05–0,20 por tailoring com GPT-4o-mini ou Claude Haiku; ~US$ 0,01 por scoring) |
| Telegram | **R$ 0** (Bot API gratuita) |
| Domínio | **R$ 0** (`.local` via mDNS) |
| **Total mensal** | **~R$ 5–20** (só custo de API, dependendo do volume) |

**Insight econômico:** rodando 30 tailorings + 100 scorings por mês com GPT-4o-mini, seu custo fica abaixo de **R$ 10**. Isso te dá um "orçamento" realista para calibrar o quanto o produto economiza e qual seria o preço justo quando productizar.

---

## 🚀 Próximos passos concretos

1. **Hoje:** crie o repo, o `go.mod`, o `docker-compose.yml` com mDNS e valide o Checkpoint 1 no celular.
2. **Esta semana:** Checkpoints 1–3 (fundação, SPA, BYOK). Ao final, você tem um app que roda no celular e guarda sua chave.
3. **Próxima semana:** Checkpoints 4–5 (currículo + tailoring + entrada manual). **Aqui você começa a usar de verdade** — não espere o Checkpoint 6 para isso.
4. **Terceira semana:** Checkpoints 6–7 (descoberta automática + Telegram). Complete o MVP.
5. **Semanas 4–7:** Checkpoint 8 (validação). Use, meça, anote. Não escreva código novo.
6. **Semana 8:** Decisão — productizar (voltar ao plano original com feed + licença) ou iterar mais no pessoal.

Esse replanejamento transforma o projeto de "construir um SaaS" em **"resolver seu próprio problema primeiro, com a opção de productizar depois"**. É a única forma sustentável de validar a tese sem queimar meses em infraestrutura que talvez nunca seja usada. Boa sorte — e volte com os dados da fase de validação, porque eles vão mudar todo o planejamento de productização que fizemos antes.