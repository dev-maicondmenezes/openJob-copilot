# spec.md (atualizado)

> **Nota de revisão:** este documento foi revisado a partir de `plan_update.md` para refletir a decisão estratégica de **priorizar uso pessoal antes de productização**. As seções marcadas com 🆕 ou ♻️ foram adicionadas/reescritas; o restante é consistente com a spec original, apenas ajustado para o escopo reduzido.

---

# Prompt: Planejamento e Desenvolvimento do **openJob** — Copiloto de Candidaturas (Local-first, Go + SvelteKit)

## 1. Contexto e Objetivo ♻️

Você é uma IA engenheira de software sênior. Projete e desenvolva o **openJob**, um aplicativo **local-first** para automatizar o fluxo de candidaturas a vagas internacionais remotas (pagamento em USD). O app roda **encapsulado em Docker** no notebook do usuário, mas deve ser acessível pela **rede local via URL amigável** (não IP) a partir do navegador do celular.

**Estratégia revisada:** o openJob começa como **ferramenta pessoal** ("resolver meu próprio problema primeiro"), com arquitetura pensada para **productização futura** como SaaS multi-tenant, sem reescrita estrutural. O objetivo do MVP é **chegar ao primeiro uso real em ~3 semanas** e validar a tese com dados antes de investir em infraestrutura de produto (licenciamento, multi-tenancy, ATS assistido em escala).

O openJob é um **copiloto**, não um "spray and pray": ele busca vagas, adapta o currículo com IA (BYOK), sugere candidaturas e notifica o usuário, que mantém a aprovação final. Automação direta no LinkedIn (Easy Apply) é **proibida**; a integração com o LinkedIn é somente leitura (importação de perfil) e descoberta pública de links de vagas, redirecionando a candidatura para o ATS da empresa.

---

## 2. Convenções de Nomenclatura do Projeto

| Item | Valor |
|---|---|
| Nome do projeto | **openJob** |
| Módulo Go | `github.com/maicondmenezes/openjob` |
| Binário | `openjob` |
| Container / serviço Compose | `openjob` |
| Hostname local (DNS) | **`openjob.copilot`** (configurável via `OPENJOB_HOSTNAME`) |
| Banco SQLite | `/data/openjob.db` |
| Arquivo de chave local | `/data/.master.key` (chmod `0600`) |
| Variáveis de ambiente | prefixo `OPENJOB_` (ex.: `OPENJOB_PORT`, `OPENJOB_DB_PATH`, `OPENJOB_HOSTNAME`) |
| Bot Telegram | `@OpenJobBot` (sugestão) |

---

## 3. Requisitos Funcionais ♻️

### 3.1. Ingestão de Currículo e Perfil
- O usuário pode fazer upload de um currículo base (PDF, Markdown ou texto).
- O sistema converte o conteúdo para Markdown estruturado e o armazena no SQLite local.
- Importação estruturada do LinkedIn via **Member Data Portability API** fica **para a fase de productização**; no MVP, o usuário pode colar o texto do perfil manualmente se quiser enriquecer o currículo.
- Apenas **uma versão ativa** do currículo base por vez (upload novo substitui o anterior).

### 3.2. Motor de IA BYOK (Bring Your Own Key)
- O usuário fornece sua própria chave de API para um provedor de IA.
- O sistema usa um **cliente fino sobre a API OpenAI-compatible** (`github.com/sashabaranov/go-openai` ou `net/http` puro), com `base_url` + `api_key` + `model` configuráveis. Isso cobre **OpenAI, Groq, Ollama, Together, Fireworks** e **Anthropic** (via endpoint compatível) — ~95% dos casos de uso pessoal.
- A chave é cifrada em repouso no SQLite com **AES-GCM**, usando uma chave local gerada na primeira execução em `data/.master.key` (32 bytes aleatórios, `chmod 0600`). **Não há senha mestra derivada (PBKDF2)** no MVP — decisão documentada em ADR.
- O README deve instruir: *"este arquivo é o segredo do seu cofre local; faça backup dele junto com o app.db, nunca o commite"*.
- O motor recebe a descrição da vaga (JD) e o currículo base, e gera uma versão adaptada do currículo, destacando palavras-chave e realizações alinhadas.

### 3.3. Descoberta e Orquestração de Vagas ♻️
- **Fase 1 (MVP, Checkpoint 5):** entrada manual de vagas — usuário cola título, empresa, URL e JD; o sistema calcula `score_fit` (0–100) e justificativa via IA. **Este é o primeiro momento de valor real.**
- **Fase 2 (Checkpoint 6):** descoberta automática em fontes públicas (Remotive, We Work Remotely, Remote OK) via `gocolly/colly` ou `net/http + goquery`. Roda **sob demanda** (botão "Buscar agora") ou via `time.Ticker` a cada 6h (desabilitado por padrão). Sem cron do sistema operacional.
- Deduplicação por `url` (UNIQUE constraint).
- Scoring automático em batch após discovery, com **rate limiting** para não estourar a API do provedor.
- A candidatura é feita **preferencialmente no site do ATS da empresa** (Greenhouse, Lever, Workday), não no LinkedIn.
- **Automação de preenchimento de formulários ATS é opcional e vem no fim** (Checkpoint 9, somente se houver dor real na validação). Copiar/colar resolve no MVP.
- LinkedIn **não** é automatizado (nem login, nem Easy Apply, nem scraping agressivo).

### 3.4. Notificações e Relatórios
- O openJob envia **um relatório diário** para o usuário via **Telegram Bot API** (às 8h, configurável).
- Notificações imediatas quando o status de uma `application` muda para `interview` ou `rejected`.
- O relatório diário inclui: novas vagas encontradas, candidaturas pendentes de aprovação, enviadas, em andamento.
- Configuração de `token` e `chat_id` via UI, persistidos em `settings`.

### 3.5. UI de Gestão (SPA Mobile-first) ♻️
- Interface para gerenciar: currículo base, chaves BYOK, pipeline de candidaturas, vagas descobertas/coladas e logs.
- **Sem UI de "habilitar/desabilitar plataformas"** no MVP — fontes de scraping são configuradas via `config.yaml` ou variáveis de ambiente. Essa UI fica para a fase de productização.
- Design **mobile-first**, responsivo, com suporte a toque e safe areas (iOS).
- Funciona como **PWA** (ícone na tela inicial, cache offline básico).
- Consome a API do backend Go na **mesma origem**, eliminando CORS.

---

## 4. Requisitos Não Funcionais e Restrições ♻️

- **Local-first:** o app roda inteiramente na máquina do usuário; a nuvem não é necessária no MVP. SQLite é o banco primário.
- **Docker:** um único container (ou Docker Compose) expõe a porta `8080` na rede local.
- **Acesso via URL:** o openJob deve ser acessível por hostname local **`openjob.copilot`**, não apenas por IP.
- **Segurança (modo pessoal):** chaves BYOK nunca são expostas no frontend; cifradas em repouso no SQLite com AES-GCM e chave local em arquivo `0600`. **Não há senha mestra** no MVP. Documentar claramente no README que o arquivo `data/.master.key` é o segredo do cofre.
- **Multi-tenancy:** **sem `tenant_id` no MVP**. Adicionar depois via migrations é trivial; a interface `Repository` e a estrutura de pastas já preparam o terreno.
- **Portabilidade para SaaS:** a lógica de negócio fica no backend Go; a UI é stateless; o schema do SQLite deve ser compatível com migração para PostgreSQL (usar `TEXT`/`INTEGER`/`DATETIME` no SQLite, mapear para `VARCHAR`/`BIGSERIAL`/`TIMESTAMPTZ` no Postgres via interface; encapsular `AUTOINCREMENT` no store).
- **Notificações:** Telegram Bot API via HTTP POST; sem dependência de webhooks complexos.
- **LinkedIn:** não é alvo de scraping agressivo nem de automação de login/Easy Apply.

---

## 5. Stack Tecnológica Recomendada ♻️

| Camada | Escolha | Observações |
|---|---|---|
| **Backend** | Go 1.22+ com `chi` | `chi` é idiomático e compatível com `net/http`; use `//go:embed` para servir a SPA. |
| **Frontend** | SvelteKit com `adapter-static` (modo SPA) + `@vite-pwa/sveltekit` | Bundle leve; fallback para `index.html`. |
| **Banco local** | SQLite (`modernc.org/sqlite` ou `mattn/go-sqlite3`) | Persistência em volume Docker `./data` → `openjob.db`. Migrations versionadas embutidas (`golang-migrate` ou SQL puro no embed). |
| **IA BYOK** | Cliente fino sobre API OpenAI-compatible | `github.com/sashabaranov/go-openai` ou `net/http` puro. Configurável via `base_url` + `api_key` + `model`. **Substitui `llmux`/`any-llm-go`** — reduz dependências e cobre OpenAI, Groq, Ollama, Together, Fireworks e Anthropic via endpoint compatível. |
| **Scraper** | `github.com/gocolly/colly` ou `net/http` + `goquery` | Roda sob demanda ou via `time.Ticker` a cada 6h (desabilitado por padrão). |
| **DNS local** | mDNS (`.local`) via container `skhmt/mdns` **ou** dnsmasq no host | `skhmt/mdns` com `network_mode: host` e `HOST=openjob.copilot`. No Linux funciona nativo; no Mac/Windows, o Bonjour do host resolve `openjob.copilot` automaticamente. Documentar os dois caminhos. |
| **Criptografia BYOK** | AES-GCM + chave local em arquivo `0600` | Gerar `data/.master.key` na primeira execução. Sem PBKDF2 no MVP. |
| **Notificações** | Telegram Bot API | Criar bot com `@BotFather`; enviar via `POST https://api.telegram.org/bot<TOKEN>/sendMessage`. |
| **Container** | Docker multi-stage | Estágio Node para build da SPA; estágio Go para binário estático com assets embutidos. |

---

## 6. Arquitetura de Referência ♻️

```text
openjob/
├── backend/
│   ├── cmd/openjob/main.go
│   ├── internal/
│   │   ├── api/          # handlers HTTP (chi)
│   │   ├── llm/          # cliente OpenAI-compatible + abstração
│   │   ├── store/        # SQLite + interface Repository + migrations
│   │   ├── tailor/       # adaptação de currículo + prompts versionados
│   │   ├── scraper/      # Remotive, WWR, RemoteOK (sob demanda)
│   │   ├── notify/       # Telegram Bot
│   │   └── tenant/       # vazio hoje; preparado para multi-tenancy futura
│   ├── web/              # SPA compilada (embed)
│   └── go.mod            # module github.com/maicondmenezes/openjob
├── frontend/             # projeto SvelteKit
│   ├── src/routes/
│   ├── svelte.config.js  # adapter-static, fallback: 'index.html'
│   └── vite.config.ts
├── Dockerfile
├── docker-compose.yml
└── data/
    ├── openjob.db        # SQLite
    └── .master.key       # chave AES-GCM (chmod 0600)
```

**Fluxo de dados:**
1. SPA (celular/navegador) → `http://openjob.copilot:8080` → backend Go.
2. Backend lê/escreve no SQLite, chama o provedor de IA com a chave BYOK e dispara notificações via Telegram.
3. Nenhum dado sai da rede local, exceto as chamadas explícitas à API de IA e ao Telegram.

**Decisões de simplificação (ADRs curtos a documentar):**
- **LLM client**: cliente fino sobre OpenAI-compatible em vez de SDK multi-provedor — reduz dependências, cobre 95% dos casos pessoais.
- **Scraper local**: roda sob demanda via botão na UI ou `time.Ticker` gerenciado pelo servidor Go; sem cron do SO.
- **Criptografia BYOK**: chave local em arquivo `0600` em vez de senha mestra + PBKDF2 — mais simples, suficiente para uso pessoal.
- **Sem `tenant_id` agora**: adicionar via migrations quando productizar é trivial; a interface `Repository` isola a mudança.
- **mDNS mantido**: mesmo sendo uso pessoal, revisar vagas no sofá pelo celular é caso de uso central.
- **Telegram mantido**: forma mais prática de receber alertas com o laptop fechado.
- **Sem UI de plataformas no MVP**: fontes configuradas por `config.yaml`/env; UI de habilitar/desabilitar fica para productização.

---

## 7. Modelo de Dados Inicial (SQLite — `openjob.db`) ♻️

Sem `tenant_id`, sem `platforms`, com `score_reason` em `jobs` e `notify_log` preservado:

```sql
-- settings: chave-valor (inclui chaves BYOK cifradas)
CREATE TABLE settings (
  key TEXT PRIMARY KEY,
  value TEXT NOT NULL,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- curriculum: currículo base (uma linha ativa por vez; a mais recente é a ativa)
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
  source TEXT NOT NULL,              -- 'remotive', 'wwr', 'remoteok', 'manual'
  score_fit INTEGER,                 -- 0-100, NULL até ser calculado
  score_reason TEXT,                 -- justificativa da IA
  status TEXT NOT NULL DEFAULT 'new',-- new | approved | discarded
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- applications: candidaturas aprovadas/enviadas
CREATE TABLE applications (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  job_id INTEGER NOT NULL REFERENCES jobs(id),
  tailored_curriculum_md TEXT,
  status TEXT NOT NULL DEFAULT 'pending', -- pending | sent | interview | rejected | offer
  sent_at DATETIME,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- notify_log: histórico de notificações enviadas
CREATE TABLE notify_log (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  type TEXT NOT NULL,                -- 'daily_report', 'status_change', 'test'
  payload TEXT,
  sent_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

**Compatibilidade futura com PostgreSQL:** usar `TEXT`/`INTEGER`/`DATETIME` no SQLite e mapear para `VARCHAR`/`BIGSERIAL`/`TIMESTAMPTZ` via interface `Repository`. Nada de `AUTOINCREMENT` hardcoded fora do store.

---

## 8. Roadmap com Checkpoints Verificáveis ♻️

Cada checkpoint deve ser **demonstrável e testável** antes de avançar. Desenvolvimento incremental, validado no celular real sempre que envolver UI ou rede.

### ✅ Checkpoint 1 — Fundação (2–3 dias)

**Entregáveis:**
- Backend Go + `chi` servindo "Hello World" via `//go:embed`.
- `Dockerfile` multi-stage e `docker-compose.yml` com porta `8080`.
- Sidecar mDNS para `openjob.copilot`.
- Acesso pelo celular confirmado.

**Critério de verificação:** abrir `http://openjob.copilot:8080` no celular na mesma rede Wi-Fi.

---

### ✅ Checkpoint 2 — SPA + PWA (2–3 dias)

**Entregáveis:**
- SvelteKit com `adapter-static` e `fallback: 'index.html'`.
- Build copiado para `backend/web/dist`, embutido no binário.
- Rotas `/`, `/settings`, `/curriculum`, `/jobs`, `/applications` funcionando com reload.
- PWA com manifest + service worker (`@vite-pwa/sveltekit`).
- Safe areas iOS (`env(safe-area-inset-*)`).

**Critério de verificação:** navegar, recarregar, instalar PWA na tela inicial do celular.

---

### ✅ Checkpoint 3 — SQLite + BYOK (3–4 dias)

**Entregáveis:**
- SQLite com migrations embutidas (`modernc.org/sqlite` + `golang-migrate` ou SQL puro no embed).
- Geração automática de `data/.master.key` na primeira execução (`chmod 0600`).
- `POST /api/settings/llm-key` (cifra e persiste), `GET /api/settings/llm-key` (retorna `{configured: bool}`).
- `POST /api/settings/llm-ping` — chamada de teste ao provedor configurado.
- UI de settings com campos `base_url`, `model`, `api_key` (mascarado).

**Critério de verificação:** salvar chave, reiniciar container, ping retornar sucesso.

---

### ✅ Checkpoint 4 — Currículo + Adaptação (3–4 dias)

**Entregáveis:**
- `POST /api/curriculum/upload` — aceita PDF/MD/TXT, converte para Markdown estruturado (`github.com/ledongthuc/pdf` para PDF; MD/TXT direto).
- `POST /api/curriculum/tailor` — recebe `job_id` (ou texto de JD colado) e retorna currículo adaptado em Markdown.
- Prompt de tailoring versionado em `internal/tailor/prompts.go` com instruções de alinhamento de palavras-chave e conquistas.
- UI para upload, visualização e diff (base vs. adaptado).

**Critério de verificação:** subir seu currículo real, colar uma JD que você está mirando, gerar versão adaptada e ler criticamente. **Primeiro momento de valor real.**

---

### ✅ Checkpoint 5 — Entrada manual de vagas + scoring (1–2 dias) 🆕

**Entregáveis:**
- `POST /api/jobs/manual` — recebe `{title, company, url, description}` e persiste.
- `POST /api/jobs/:id/score` — calcula `score_fit` via IA, com `score_reason`.
- UI: cole uma JD, clique "score", veja ranking.
- `POST /api/jobs/:id/approve` — cria `application` com status `pending`.

**Critério de verificação:** você cola a JD da vaga que está mirando, recebe score e justificativa, aprova, e a candidatura entra no pipeline. **A partir daqui você já pode usar o produto de verdade** — não espere o Checkpoint 6 para isso.

---

### ✅ Checkpoint 6 — Descoberta automática (3–4 dias)

**Entregáveis:**
- Scrapers para Remotive, WWR e RemoteOK (públicas, sem login).
- `POST /api/jobs/discover` — dispara scraping sob demanda.
- `time.Ticker` opcional a cada 6h (configurável, desabilitado por padrão).
- Deduplicação por `url` (UNIQUE constraint).
- Scoring automático em batch (rate-limited para não estourar API).

**Critério de verificação:** clicar "Buscar agora", ver 30–80 vagas novas, com score calculado, ordenadas por `score_fit`.

---

### ✅ Checkpoint 7 — Telegram + tracking (2–3 dias)

**Entregáveis:**
- `POST /api/notify/test` — mensagem de teste.
- Cron diário (via `robfig/cron` ou `time.Ticker`) às 8h enviando relatório: novas vagas, pendentes de aprovação, enviadas, em andamento.
- `PATCH /api/applications/:id` — atualiza status; dispara notificação imediata se virar `interview` ou `rejected`.
- UI para configurar `token` + `chat_id`.

**Critério de verificação:** receber mensagem de teste; no dia seguinte, receber o relatório.

---

### ✅ Checkpoint 8 — Validação pessoal (2–4 semanas, sem código novo) 🆕

**Ação:** você usa o app de verdade. Aplica para 5–15 vagas com currículos adaptados, acompanha o pipeline, recebe notificações.

**Registre:**
- Quantas vagas você aprovou vs. descartou?
- Quantas geraram resposta?
- O tailoring está bom o suficiente ou você edita manualmente?
- O score de fit está calibrado ou está tudo 70–80?
- O que você sente falta toda vez que abre o app?

**Critério de verificação:** você tem dados reais para decidir se vale productizar. **Não avance sem isso.**

---

### ⚠️ Checkpoint 9 (opcional) — ATS assistido 🆕

**Só faça se, na validação, você sentir dor real de preencher formulários Greenhouse/Lever manualmente.**

**Entregáveis:**
- Integração com Greenhouse e Lever via Playwright (headless) para preencher formulários.
- O preenchimento para antes do botão "Enviar"; o usuário revisa na UI e confirma o envio manualmente.
- Limite diário de candidaturas assistidas (5–10) para evitar bloqueios.
- LinkedIn **não** é automatizado.

**Critério de verificação:** para uma vaga Greenhouse de teste, o formulário é preenchido automaticamente e o usuário consegue revisar e enviar manualmente.

---

## 9. Fase de Validação — o que medir 🆕

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

## 10. O que Preservar para o Futuro (não jogar fora) 🆕

Mesmo sendo uso pessoal, **não corte estas coisas** — o custo de adicionar depois é muito maior do que manter agora:

1. **Interface `Repository` no store** — permite trocar SQLite por PostgreSQL sem tocar em handlers.
2. **Abstração `LLMClient`** — permite trocar OpenAI-compatible por outro SDK se necessário.
3. **Migrations versionadas** — mesmo com um único usuário, evitam "drop table" no futuro.
4. **ADRs curtos** — documente decisões (por que chave local e não senha mestra; por que sem `tenant_id` agora; por que mDNS e não IP). Esses docs valem ouro na productização.
5. **Testes nos pontos críticos** — criptografia de BYOK, parsing de JD, deduplicação de vagas. Sem cobertura alta, mas esses três quebram silenciosamente.
6. **Estrutura de pastas preparada para `internal/tenant/`** — mesmo vazia hoje, evita refatoração estrutural depois.

---

## 11. Custos Operacionais no Modo Pessoal 🆕

| Item | Custo |
|---|---|
| Infraestrutura | **R$ 0** (seu laptop) |
| Banco | **R$ 0** (SQLite local) |
| IA | **BYOK** — você paga direto ao provedor (ex.: ~US$ 0,05–0,20 por tailoring com GPT-4o-mini ou Claude Haiku; ~US$ 0,01 por scoring) |
| Telegram | **R$ 0** (Bot API gratuita) |
| Domínio | **R$ 0** (`.local` via mDNS) |
| **Total mensal** | **~R$ 5–20** (só custo de API, dependendo do volume) |

**Insight econômico:** rodando 30 tailorings + 100 scorings por mês com GPT-4o-mini, o custo fica abaixo de **R$ 10**. Isso dá um "orçamento" realista para calibrar o quanto o produto economiza e qual seria o preço justo quando productizar.

---

## 12. Entregáveis Finais ♻️

- Código-fonte completo do **openJob** em monorepo (`backend/`, `frontend/`, `Dockerfile`, `docker-compose.yml`).
- Documentação de instalação e uso (`README.md` em inglês e português) com instruções para `openjob.copilot` e para o arquivo `data/.master.key`.
- Script de setup do DNS local (mDNS ou dnsmasq) com instruções para o usuário.
- Suíte de testes automatizados para os checkpoints 1–7 (foco em criptografia BYOK, parsing de JD, deduplicação de vagas).
- ADRs curtos documentando: criptografia BYOK (chave local vs. senha mestra), ausência de `tenant_id`, escolha de mDNS, cliente LLM OpenAI-compatible.
- Relatório de riscos e limitações (LinkedIn, mudanças de API, segurança de chaves, mDNS em Mac/Windows).
- (Opcional, pós-validação) Integração com Greenhouse/Lever via Playwright.

---

## 13. Instruções para a IA Desenvolvedora ♻️

1. **Planeje antes de codar:** produza um plano de implementação checkpoint por checkpoint, com estimativas e dependências.
2. **Entregue código funcional e testável** em cada checkpoint; não avance sem o critério de verificação atendido.
3. **Priorize a segurança:** nunca exponha chaves no frontend; use AES-GCM com chave local `0600`; não automatize o LinkedIn; documente claramente o papel de `data/.master.key`.
4. **Mantenha a portabilidade:** evite acoplamento a um provedor de IA ou a um banco específico; use interfaces (`LLMClient`, `Repository`).
5. **Documente decisões de arquitetura** (ADRs curtos) especialmente para BYOK, DNS local, ausência de multi-tenancy e escolha do cliente LLM.
6. **Valide no celular real** a cada checkpoint que envolva UI ou acesso via rede.
7. **Mantenha a identidade do projeto:** todos os artefatos (binário, container, hostname `openjob.copilot`, banco `openjob.db`, variáveis `OPENJOB_*`) devem usar o nome **openJob**.
8. **Respeite a sequência de valor:** entregue o Checkpoint 5 (entrada manual + scoring) **antes** do Checkpoint 6 (descoberta automática). O usuário deve poder usar o app de verdade o mais cedo possível.
9. **Não productize antes do Checkpoint 8:** qualquer feature de multi-tenancy, licenciamento, UI de plataformas ou ATS assistido em escala **só** entra após a validação pessoal. Se a tentação aparecer, registre como ADR "deferred post-validation".

---

**Resumo da revisão:** o escopo foi cortado em ~40% (sem licenciamento, sem multi-tenancy, sem PBKDF2, sem UI de plataformas, sem ATS assistido no MVP), o caminho até o primeiro uso real encurtou de ~6–8 semanas para **~3 semanas**, e o roadmap agora termina com uma **fase explícita de validação com dados** antes de qualquer decisão de productização. A arquitetura preserva as interfaces (`Repository`, `LLMClient`), migrations versionadas e estrutura de pastas (`internal/tenant/`) que tornam a productização uma evolução, não uma reescrita.