# Checkpoint 11 — Empacotamento em binário único, wizard de instalação e release (4–5 dias)

> Origem: spec §13 (distribuição) + README (Quick start atual via Docker). Pré-requisito: **todos os checkpoints anteriores (1–10) concluídos**. Este é o checkpoint de *fechamento*: o openJob deixa de exigir Docker/Compose para o usuário final e passa a ser um **único executável Go** que se auto-configura na primeira execução.

## 1. Objetivo

Entregar o openJob como **um único binário Go executável**, sem dependências externas em runtime (sem Docker, sem Node, sem arquivos soltos), contendo:

- O servidor HTTP (API + SPA embutida via `//go:embed`, conforme já decidido no [ADR 0003](0003-adapter-static-spa.md)).
- O driver SQLite puro-Go (`modernc.org/sqlite`, [ADR 0002](0002-sqlite-driver-modernc.md)) — sem CGO, portanto **cross-compilation viável**.
- Um **wizard de instalação interativo** (TUI) que roda na primeira execução (ou via subcomando explícito) e guia o usuário pelas configurações mínimas: porta, chave do provedor de IA (BYOK), palavras-chave e filtro de currículo do CP6, token do Telegram do CP7.
- **Transparência total** ao final do wizard: permissões de rede, portas abertas, limites de uso das APIs de vagas (Remotive 4 req/dia, Remote OK 100 vagas/janela, etc.), custo estimado de LLM (BYOK — o app não cobra nada), e o que **não** faz (sem automação de candidatura no LinkedIn, spec §3.3).
- Ao final, o wizard oferece **“Executar o openJob agora?”** e, se sim, exibe **duas URLs**: a local (`http://localhost:<porta>`) e a **URL de rede local acessível por dispositivos móveis na mesma rede Wi-Fi** (ex.: `http://192.168.15.X:8080` e/ou `http://openjob.copilot:8080`).

> **Princípio:** o usuário final nunca deve precisar editar `.env` nem `docker-compose.yml` para o caminho feliz. O wizard escreve a configuração inicial no diretório de dados (`./data/`), que continua sendo o cofre local (master key AES-GCM do CP3).

## 2. Pré-requisitos

- CP1–CP10 concluídos e estáveis.
- `frontend/` produz `backend/web/dist` via `make build` (já existente).
- `modernc.org/sqlite` confirmado como driver (ADR 0002) — **sem CGO** é o que permite cross-compile.
- Mecanismo de cifragem AES-GCM do CP3 reutilizado para qualquer segredo que o wizard coletar (API key do LLM, token do Telegram).

## 3. Modelo de dados / artefatos (novo)

### 3.1 Subcomandos do binário

O binário passa a ser **subcommand-driven** (como sugerido no padrão de single-binary release):

| Comando | Comportamento |
|---|---|
| `openjob` (sem args) | Se `data/config.json` não existir → executa o **wizard**. Se existir → sobe o servidor. |
| `openjob setup` | Força o wizard (re-configuração). |
| `openjob serve` | Sobe o servidor sem wizard (falha se não configurado). |
| `openjob version` | Imprime versão (injetada via `-ldflags -X …version.Version=vX.Y.Z`). |

### 3.2 `data/config.json` (bootstrap mínimo)

Arquivo **não-secreto** (segredos ficam cifrados no SQLite, como no CP3), com:

- `port` (int), `hostname` (string, default `openjob.copilot`), `lan_ip_hint` (string, opcional).
- `discovery.{keywords: []string (máx. 5), resume_filter:{...}}` — espelha o CP6 para o wizard poder semear.
- `llm.{provider, model}` (a **chave** não fica aqui).
- `telegram.chat_id` (o **token** não fica aqui).
- `wizard_completed_at` (timestamp) — gate do subcomando `serve`.

> **Por que JSON e não só SQLite?** O wizard precisa gravar antes de o SQLite estar inicializado/migrado. `config.json` é o *seed*; na primeira subida do `serve`, ele é lido e os valores sensíveis são migrados para o SQLite cifrado (chaves) e para `settings` (CP6/CP10). Depois disso, a UI de `/settings` passa a ser a fonte da verdade e o `config.json` só é reescrito pelo wizard.

### 3.3 Descoberta de IP local (para exibir a URL mobile)

- Usar `net.Interfaces()` + `net.InterfaceAddrs()` para listar IPs **não-loopback, não-link-local, IPv4, privados** (RFC 1918).
- Se houver mais de um (Wi-Fi + Ethernet + VPN), o wizard pergunta qual interface priorizar (dropdown), com default na que tiver rota default (via `net.Dial("udp", "8.8.8.8:80")` + `LocalAddr`, sem enviar pacote).
- Exibir **duas** URLs móveis: `http://<ip-privado>:<porta>` e, se o mDNS estiver ativo (CP1), `http://<hostname>:<porta>`. Aviso claro: **Android/Chrome não resolve `.local`** (limitação já documentada no README) → para Android, priorizar a URL por IP ou `nip.io`.

## 4. Tarefas

### 4.1 Empacotamento single-binary

- **SPA embutida:** manter `//go:embed all:dist` em `backend/web/` (padrão já usado no projeto; o `all:` inclui dot-files). O binário final **não** lê `backend/web/dist` do disco em produção — só em dev via `make run` ou override explícito (ex.: `OPENJOB_WEB_DIST` para debug).
- **SQLite sem CGO:** confirmar `CGO_ENABLED=0` no build de release. `modernc.org/sqlite` é puro Go; o único ponto de atenção é o tamanho do binário (SQLite puro-Go adiciona alguns MB — aceitável).
- **Assets do wizard:** o wizard é TUI (`charmbracelet/huh` v2 / `charm.land/huh/v2`), sem assets externos.
- **Sem tarball obrigatório:** o release anexa o **binário puro** por plataforma + checksums; arquivo compactado é opcional. Seguir o padrão “true single file via `go:embed` — no tarball, no build-tag split”.
- **Tamanho alvo:** documentar no README o tamanho aproximado por plataforma (referência: binários Go com SPA + SQLite puro costumam ficar na faixa de ~20–35 MB).

### 4.2 Wizard de instalação (TUI)

Biblioteca: **`charmbracelet/huh`** (v2, import `charm.land/huh/v2`), que oferece `Input`, `Select`, `MultiSelect`, `Confirm` com tema e modo acessível. Alternativa mantida: `bubbletea` direto se o wizard precisar de telas muito customizadas (não é o caso).

Fluxo em **5 grupos** (poucos passos, linguagem direta):

1. **Boas-vindas + transparência antecipada** (uma tela de texto):
   - “O openJob roda **só no seu computador**. Nada é enviado para servidores nossos — não temos servidores.”
   - “Suas chaves de API ficam **cifradas no seu disco** com uma master key local. Se você perder `data/.master.key`, perde as chaves.”
   - “O app **não automatiza candidaturas** no LinkedIn. Ele descobre, adapta e pontua — **você aprova e aplica**.”
2. **Rede**:
   - Porta (default `8080`, validação de disponibilidade com `net.Listen`).
   - Hostname mDNS (default `openjob.copilot`, CP1).
   - Seleção da interface de rede para a URL mobile (se múltiplas).
3. **Inteligência Artificial (BYOK)**:
   - Provedor (dropdown: OpenAI, Anthropic, Google Gemini, “outro compatível com OpenAI API”).
   - API key (campo mascarado com `EchoModePassword`), testada com uma chamada mínima (`/models` ou equivalente) antes de aceitar. Se falhar, oferece “salvar mesmo assim” (para ambientes offline/air-gapped).
   - Modelo (default por provedor).
4. **Descoberta de vagas (CP6)**:
   - Até **5 palavras-chave** (mesma validação dura do CP6: rejeita >5, trim, dedup).
   - Filtro por currículo: toggle + `min_matches` (default 3) + `match_mode` (`any`/`all`).
   - Aviso dos limites das fontes: “Remotive limita a ~4 buscas/dia; Remote OK entrega as 100 vagas mais recentes; Jobicy cobre os últimos 7 dias.” — transparência desde o wizard.
5. **Telegram (CP7, opcional)**:
   - Token do bot + `chat_id`, com botão **“Enviar mensagem de teste”** embutido no wizard (reutiliza `POST /api/notify/test` do CP7, mas chamado antes de o servidor subir — implementar um `notify.SendTestDirect` que não depende do bus/HTTP).
   - Se o usuário pular, o CP7 continua funcionando depois via `/settings`.

**Comportamento transversal do wizard:**
- **Modo não-interativo (`--yes` / CI):** aceita defaults e variáveis de ambiente (`OPENJOB_*`), permitindo automação de testes e instalações por script.
- **Acessibilidade:** `.WithAccessible(console.IsAccessibleMode())` no form-level (padrão recomendado pelos mantenedores do `huh`).
- **Cancelamento seguro:** `Ctrl+C` → pergunta “Sair sem configurar? O app não subirá até você rodar `openjob setup`.” Nenhum arquivo parcial é deixado (escrever `config.json` em `tmp` + rename atômico).

### 4.3 Tela final do wizard (transparência total + URLs)

Após salvar a config, o wizard renderiza um **resumo final** (não só uma pergunta “rodar?”):

**Permissões e limites (explícitos):**
- “O openJob abrirá a porta **8080** no seu computador, acessível **apenas na sua rede local** (não exposto à internet por padrão).”
- “Para acessar do celular, o computador e o celular precisam estar **na mesma rede Wi-Fi**.”
- “Limites das fontes de vagas: Remotive ~4 req/dia; Remote OK 100 vagas/janela; Jobicy últimos 7 dias.”
- “Custo de IA: **você paga diretamente ao provedor** (BYOK). O openJob não intermedeia nem cobra.”
- “O que o app **não** faz: automação de candidatura (LinkedIn Easy Apply), scraping de perfis, envio de mensagens em seu nome.”

**URLs disponíveis:**
- **Neste computador:** `http://localhost:<porta>`
- **Celular / outro dispositivo na mesma rede:** `http://<ip-privado>:<porta>` (e `http://<hostname>:<porta>` se mDNS ativo)
- Aviso para Android: “No Chrome do Android, use a URL por IP; `.local` não funciona (limitação do Chromium).” (coerente com o README)

**Ação final:**
- `Confirm`: **“Executar o openJob agora?”** → se sim, chama o mesmo caminho de `serve` (sem re-executar o wizard) e imprime as URLs no terminal + log estruturado.
- Se não, imprime: “Para iniciar depois, rode `openjob serve`.”

### 4.4 Release e distribuição

- **GoReleaser** como ferramenta de release (`goreleaser init` → `.goreleaser.yaml` versionado).
  - Builds: `linux/amd64`, `linux/arm64`, `darwin/amd64`, `darwin/arm64`, `windows/amd64` (**sem CGO** em todos, viabilizado pelo `modernc.org/sqlite`).
  - `ldflags`: injetar `version.Version` a partir da tag Git (`vX.Y.Z`).
  - Checksums SHA-256 anexados ao release; assinatura (cosign/GPG) como **backlog** (não bloqueia esta CP).
  - **Sem** `brews`/tap nesta CP (backlog), **sem** Docker Hub obrigatório (Docker continua como canal secundário, mas o código converge para o mesmo `go:embed` — padrão descrito em).
- **GitHub Actions:** workflow `release.yml` disparado por tag `v*`, roda `goreleaser release --clean` com `GITHUB_TOKEN` (escopo `contents: write`).
- **Versionamento semântico:** `goreleaser check` valida a tag (`vX.Y.Z`).
- **Smoke test pós-build:** em CI, baixar o binário `linux/amd64` do release *snapshot* e rodar `openjob version` + `openjob serve --dry-run` (sobe, responde `/api/health`, derruba) — garante que o `go:embed` não quebrou.

### 4.5 README e documentação de instalação

- Substituir o “Quick start” atual (Docker) por **duas vias**:
  1. **Binário único (recomendado para usuário final):** baixar da página de releases, `chmod +x openjob`, executar `./openjob` → wizard.
  2. **Docker (mantido para dev/self-hosting avançado):** `docker compose up -d` (inalterado).
- Documentar a **master key** (`data/.master.key`) com destaque ainda maior: no fluxo single-binary, o usuário pode não ter o hábito de “fazer backup do volume Docker” — o README deve dizer explicitamente para copiar `data/` (DB + master key).
- Documentar os **subcomandos** (`setup`, `serve`, `version`) e o modo `--yes`.

## 5. Critérios de verificação

- Baixar o binário `linux/amd64` do release e executar `./openjob` em uma máquina **sem Docker e sem Go** → wizard abre, coleta as 5 etapas, salva `data/config.json`, cria `data/.master.key` + DB, e a tela final exibe as duas URLs.
- Responder “Sim” em “Executar agora?” → servidor sobe, `GET http://localhost:<porta>/api/health` retorna `200`, SPA abre no navegador do próprio computador.
- Acessar a **URL por IP** a partir de um celular na mesma Wi-Fi → SPA carrega; `GET /api/health` responde (prova de que a URL mobile exibida é real, não só calculada).
- Rodar o binário novamente **sem** apagar `data/` → **não** reabre o wizard; sobe direto (`serve`).
- `openjob setup` → reabre o wizard com os valores atuais pré-preenchidos; alterar a porta reflete na próxima subida.
- `openjob serve` em diretório sem `config.json` → falha com mensagem clara (“rode `openjob setup` primeiro”), **sem** criar arquivos parciais.
- `--yes` em ambiente limpo → conclui sem interação, usando defaults + variáveis `OPENJOB_*`; smoke test de `version` e `health` passa.
- Cancelar o wizard no meio (`Ctrl+C`) → nenhum `config.json` parcial; próxima execução reabre o wizard.
- **Cross-compile:** `goreleaser release --snapshot --clean` produz os 5 targets sem erro de CGO.
- **Transparência:** a tela final contém, no mínimo, as 5 afirmações de permissões/limites listadas em §4.3 (teste de snapshot de texto no TUI, se viável; caso contrário, revisão manual documentada no PR).

## 6. Registrar

- **ADR `0011-single-binary-wizard.md`** — decisão de distribuir como binário único com wizard TUI (`huh` v2) como caminho primário, mantendo Docker como canal secundário. Justificativa: elimina a barreira de entrada (Docker/Compose) sem abrir mão do `go:embed` já existente; o SQLite puro-Go permite cross-compile sem CGO. Alternativa rejeitada: instalador nativo por SO (MSI/DMG/deb) — multiplica manutenção por plataforma sem ganho proporcional nesta fase.
- **ADR `0012-wizard-transparency.md`** — decisão de que a transparência (permissões, limites de fontes, custo BYOK, “o que não faz”) é **parte do produto**, não um README opcional; toda versão do wizard deve exibir o resumo final. Justificativa: alinhamento com spec §3.3 (proibição de automação) e com o caráter local-first do app.
- **Testes:**
  - `config.json` atômico (tmp + rename) — matar o processo no meio não deixa arquivo corrompido.
  - Validação de porta (ocupada → erro claro no wizard).
  - Descoberta de IP: mock de `net.Interfaces` com múltiplos IPs (Wi-Fi + VPN) → wizard prioriza a interface correta.
  - `serve` sem config → erro sem efeitos colaterais.
  - Smoke test do binário no CI (ver §4.4).
- **Riscos anotados:**
  - **Firewall do SO:** no Windows/macOS, a primeira execução pode disparar diálogo de firewall; o wizard deve **avisar antecipadamente** (“o SO pode pedir permissão para aceitar conexões na rede local — autorize para usar no celular”). No Linux, `ufw`/`firewalld` podem bloquear a porta — o wizard detecta e sugere o comando, mas **não** executa `sudo` automaticamente.
  - **Android + `.local`:** limitação do Chromium já conhecida (README) — o wizard **não** promete `.local` para Android; oferece a URL por IP como caminho garantido.
  - **Tamanho do binário:** SQLite puro-Go + SPA aumenta o artefato; documentar e monitorar no CI (falha se ultrapassar um teto configurado, ex.: 50 MB).
  - **`huh` v2 e caminhos de import vanity (`charm.land/...`):** a migração de v1 → v2 é recente; fixar a versão e cobrir o wizard com um teste de “renderização headless” (modo acessível) para detectar quebras de API.
- **Backlog (não nesta CP):** assinatura de binários (cosign/GPG); auto-update; Homebrew tap e pacotes `deb`/`rpm`; instalador gráfico (fora do escopo TUI).

## 7. Fora de escopo (deliberadamente adiado)

- **Auto-update in-app** — exige endpoint de versionamento e verificação de integridade; fica para pós-validação.
- **Instaladores nativos por SO** (MSI, DMG, `.deb`/`.rpm`) — o binário puro + wizard cobre o caminho feliz nesta fase.
- **Assinatura/notarização de código** (Apple/Windows) — necessário para distribuição em larga escala, mas não para o público inicial (early adopters que baixam de GitHub Releases).
- **Serviço systemd/launchd/Windows Service** — o wizard sobe o processo em foreground; rodar como serviço é documentado como opção avançada, não automatizado.
- **Push/multi-tenant/licenciamento** (spec §13.9) — inalterado.
- **Automação de candidatura** — permanece proibida (spec §3.3); o wizard reafirma isso explicitamente na tela de transparência.