# Backlog — openJob

> Melhorias coletadas fora do escopo dos checkpoints originais (spec §8).
> Cada item entra como checkpoint de refinamento após a fase de validação
> (CP8), a menos que um checkpoint ativo dependa dele. Ordem sugerida por
> valor × esforço.
>
> O fechamento do MVP está organizado em fases no
> [plano de release MVP](plan/release-mvp.md) (CP16 + CP17 + backlog restante).

---

## IA / Configuração de provedores

### B1 — Tela de configuração de IA amigável (onboarding de chave)
- **Status:** proposto
- **Esforço:** pequeno (frontend + copy)
- **Descrição:** a página `/settings` deve guiar o usuário sem assumir conhecimento prévio:
  - Explicação curta (2–3 frases) do que é uma API key e por que o openJob pede (BYOK — cada um usa a própria conta, custo fica sob controle do usuário).
  - Passo a passo resumido por provedor ("onde crio minha chave") com **link para a documentação oficial**:
    - Google AI Studio (Gemini) — https://aistudio.google.com/apikey
    - Groq — https://console.groq.com/keys
    - OpenRouter (modelos gratuitos) — https://openrouter.ai/keys
    - OpenAI — https://platform.openai.com/api-keys
    - Ollama (local, sem chave) — https://ollama.com
    - Together / Fireworks — links das consoles respectivas
  - Nota de segurança visível: a chave é cifrada (AES-GCM) e nunca volta para o navegador.
- **Aceite:** usuário leigo consegue obter e colar uma chave sem sair da página (guia + links).
- **Origem:** dor real na configuração inicial (CP3/CP4 — usuário precisou descobrir a URL do endpoint OpenAI-compatível do Gemini fora do app).

### B2 — Múltiplos modelos com seleção do ativo
- **Status:** proposto
- **Esforço:** médio (store + API + UI)
- **Descrição:** permitir cadastrar vários perfis de provedor (base_url + model + api_key, cada um cifrado) e alternar qual está **ativo** com um clique.
  - Store: settings `llm.profiles.<id>.{base_url,model,api_key}` + `llm.active_profile` (ou nova tabela `llm_profiles` via migration — preferir migration versionada).
  - API: CRUD de perfis + `POST /api/settings/llm/activate` (nunca ecoar chaves).
  - UI: lista de perfis com badge "ativo", botão testar por perfil, chave mascarada.
  - Compatibilidade: migrar a config atual (`llm.*` plana) para o primeiro perfil automaticamente.
- **Aceite:** trocar o provedor ativo sem reconfigurar; tailor/scoring usam o ativo imediatamente (sem restart).
- **Depende de:** nada; base pronta (CP3).

### B3 — Presets de plataformas homologadas (prefill de URL/modelos)
- **Status:** proposto
- **Esforço:** pequeno–médio
- **Descrição:** seletor fixo ("catálogo") de plataformas homologadas que preenche automaticamente `base_url` e sugere `model`:
  - OpenAI, Gemini (`/v1beta/openai`), Groq, OpenRouter, Together, Fireworks, Ollama (`http://localhost:11434/v1`), LM Studio.
  - O preset é só prefill — campos permanecem editáveis (não travar o genérico).
  - Manter a lista versionada no backend (`internal/llm/presets.go`) e expor via `GET /api/settings/llm/presets` — UI nunca hardcodeia.
  - Homologação = testado no fluxo completo (ping + completion) e documentado no ADR-0005.
- **Aceite:** escolher "Gemini" no catálogo preenche `https://generativelanguage.googleapis.com/v1beta/openai` + modelos do free tier, sem digitar nada.
- **Depende de:** idealmente B2 (presets por perfil); pode existir sozinho.

### B4 — Modelos gratuitos pré-configurados
- **Status:** proposto
- **Esforço:** pequeno (por cima de B3)
- **Descrição:** dentro do catálogo, marcar opções **free tier** e sugerir modelos gratuitos estáveis:
  - Gemini free tier (AI Studio), Groq free tier, OpenRouter `:free`, Ollama/LM Studio (100% local).
  - UI mostra custo estimado/cotas ("gratuito — sujeito a rate limit") e o preset já sugere um modelo default gratuito.
- **Aceite:** fluxo "quero começar de graça" em ≤ 3 toques: escolher preset free → criar chave no link (B1) → colar → testar.
- **Depende de:** B1 + B3.

### B5 — Configuração automática via login/autorização (OAuth)
- **Status:** proposto · ⚠️ maior risco/esforço · avaliar viabilidade antes de agendar
- **Esforço:** grande
- **Descrição:** "conectar conta" nas plataformas homologadas: app abre a página de autorização do provedor, recebe o token via callback local (`http://<host>:8080/api/settings/llm/oauth/callback`) e configura o perfil automaticamente — sem o usuário criar/copiar chave.
  - Viabilidade por plataforma (verificar antes de implementar): a maioria das APIs OpenAI-compatible **não emite tokens via OAuth público para apps pessoais**; Gemini/OpenAI exigem OAuth client de projeto GCP/dev account. Ollama local não precisa. Provável viável primeiro para Gemini (OAuth do Google Cloud) — mas envolve cadastro de app, client_id embutido no binário local e consent screen em modo "testing".
  - Requisitos de segurança: token obtido via OAuth entra no mesmo pipeline cifrado do BYOK (AES-GCM + master key); nunca logado (redação já cobre — `internal/llm/redact.go`); revogação suportada pelo provedor.
  - Fallback obrigatório: fluxo manual por chave (B1) continua disponível se OAuth falhar/expirar.
- **Aceite:** usuário conecta uma plataforma homologada com login e o perfil fica pronto para ping/tailoring sem colar chave nenhuma.
- **Depende de:** B2 (perfis) + spike de viabilidade por provedor.

### B6 — Erros amigáveis sem detalhes internos
- **Status:** proposto
- **Esforço:** pequeno–médio
- **Descrição:** mapear falhas de provedor para mensagens de usuário com **causa provável + ação sugerida**, sem expor payloads/stacks internos:
  - 401/403 → "Chave inválida, expirada ou sem permissão para este modelo — gere uma nova em <link do provedor>".
  - 404 no model → "Modelo não disponível para sua conta/chave — confira no catálogo (B3)".
  - 429 → "Cota/rate limit atingido — aguarde ou troque de modelo".
  - 5xx/503 "high demand" → "Provedor sobrecarregado agora — tente de novo em instantes ou troque de modelo".
  - timeout/rede → "Não consegui alcançar o provedor — confira internet/base_url".
  - decrypt (`ErrKeyMismatch`) → "Master key mudou — reconfigure a chave" (já implementado no ping; padronizar no tailor/scoring).
  - Regras: nunca mostrar JSON bruto do provedor, stack do Go, nem URL completa; **manter a redação do secret** (`internal/llm/redact.go`); log do servidor mantém o detalhe técnico (nível debug).
- **Aceite:** nenhuma mensagem de erro na UI expõe payload interno; todo erro sugere o próximo passo concreto.
- **Depende de:** nada; independente dos demais.

---

## Currículo & Tailoring (Checkpoint 4)

### B7 — Extração de currículo direto do LinkedIn via OAuth / autorização
- **Status:** proposto · ⚠️ maior esforço/restrições da API do LinkedIn
- **Esforço:** grande
- **Descrição:** permitir conectar conta do LinkedIn via login e autorização OAuth 2.0 para extrair os dados do perfil profissional diretamente para o currículo-base:
  - Fluxo: botão "Importar do LinkedIn" na página de currículo, autorização do usuário na tela do LinkedIn, e backend recebe os dados de histórico profissional, formação, habilidades e resumo para estruturar em Markdown (`curriculum.content_md`).
  - Restrições a avaliar: as APIs do LinkedIn para extração de perfil completo e histórico profissional costumam exigir aprovação em programas de parceria (Member Data Portability API ou LinkedIn Partner Program); o escopo básico OpenID (`openid profile email`) retorna apenas nome e foto. Necessário spike de viabilidade.
  - Fallback: continuar suportando upload de PDF (inclusive o exportado pelo próprio LinkedIn) e MD/TXT (spec §3.1).
- **Aceite:** usuário autoriza o aplicativo via LinkedIn e tem o currículo preenchido automaticamente em Markdown no banco local.

### B8 — Tratamento de erros amigável no tailoring e upload (sem expor detalhes de log)
- **Status:** proposto
- **Esforço:** pequeno
- **Descrição:** assegurar que erros no upload, conversão de PDF e tailoring com IA apresentem mensagens amigáveis e orientativas para o usuário, sem expor mensagens de erro internas, payloads brutos, stacks ou detalhes de log do servidor:
  - Backend: registrar detalhes técnicos completos via `slog.Error` no terminal/log do servidor, mas retornar JSON com descrições amigáveis e orientadas à ação (ex.: substituir mensagens contendo JSON do provedor de IA ou erros de rede por "O provedor de IA não conseguiu gerar a adaptação no momento. Verifique sua chave em Configurações ou tente novamente em instantes.").
  - Frontend: exibir banners de erro com texto claro, sem códigos de erro técnicos ou stacks.
- **Aceite:** nenhuma falha de upload, parse ou IA vaza stack trace ou detalhes de log internos na tela do usuário.

### B9 — Exportação da versão adaptada para os formatos aceitos para importação (PDF, TXT e MD)
- **Status:** proposto
- **Esforço:** médio
- **Descrição:** permitir exportar o currículo adaptado para todos os formatos aceitos na importação (além do `.md` já existente):
  - **TXT**: download do texto puro limpo, com marcadores simplificados e sem tags markdown.
  - **PDF**: geração de PDF formatado e pronto para envio, gerado no backend via biblioteca Go ou no frontend via CSS de impressão (`@media print`) / `html2pdf`.
  - **MD**: manter opção existente de baixar `.md`.
- **Aceite:** usuário pode escolher baixar o currículo adaptado em PDF, TXT ou MD com um clique.

### B10 — Botão discreto de cópia na área de exibição da versão adaptada
- **Status:** proposto
- **Esforço:** muito pequeno (UI)
- **Descrição:** adicionar um botão discreto de cópia diretamente na área de exibição da versão adaptada (estilo botão de cópia de blocos de código/previews), tanto na visualização renderizada quanto nas abas de diff:
  - Posicionamento discreto no canto superior direito do container de visualização.
  - Ícone minimalista com tooltip e feedback rápido "Copiado!" para que o usuário não precise rolar até o topo da página para copiar o conteúdo adaptado.
- **Aceite:** botão discreto presente no canto da área de exibição, copiando o Markdown adaptado e fornecendo feedback visual imediato.

---

## Discovery & Sources (Checkpoint 6)

### UI Revision for Checkpoint 6

- **Status:** proposto
- **Esforço:** médio
- **Descrição:** Alinhar a UI da descoberta automática, gerenciamento de fontes e páginas de settings com o design system do projeto (navegação em abas, cards, badges, modais, foco acessível). Incluir polimento visual, layout responsivo e revisão de design.

---

## Tracking & Pipeline de candidaturas (Checkpoint 17)

### B12 — Atualização automática do status das vagas via e-mail
- **Status:** ✅ promovido a checkpoint — ver [17-email-tracking-pipeline.md](plan/17-email-tracking-pipeline.md)
- **Esforço:** grande (6–8 dias)
- **Descrição:** ler automaticamente o e-mail informado no currículo (Gmail OAuth read-only ou IMAP) e atualizar o estágio das candidaturas a partir das respostas dos recrutadores/ATSs. Inclui:
  - **Conector de e-mail** com sync incremental (Gmail `history.list` / IMAP `UID > last_uid`) e polling local-first (sem Pub/Sub nem webhook público).
  - **Cascata de matching** thread → nome da empresa → allowlist de ATS (Greenhouse, Lever, Workday, Ashby, SmartRecruiters, iCIMS) → domínio corporativo → LLM apenas na cauda de baixa confiança. O matching por domínio do remetente **não funciona** na prática (0/237 em dados de referência) porque os e-mails vêm de relays de ATS.
  - **Pipeline com `in_process` ("em processo")** como coluna generalista entre "enviada" e "encerrada", substituindo o `status` fixo por `stage_group` (6 valores) + `stage_label`.
  - **Tags com namespace `phase:`** para representar o estágio específico de cada empresa/plataforma sem inflar o enum.
  - **Transições monotônicas:** avanço automático só para frente e com confiança ≥ 0.8; confiança baixa vira sugestão revisável; rejeição nunca reabre o processo.
- **Aceite:** e-mail de confirmação/entrevista/proposta/rejeição vincula-se à candidatura correta e move o estágio; casos ambíguos aparecem como sugestão; nenhuma transição automática retrocede.
- **Depende de:** CP7, CP10 e CP12 concluídos.
- **Origem:** sessão de chat "APIs de vagas remotas" (`docs/context/6c7eea4c-71d0-49e7-9f5b-5c374f9b73c6.json`, msgs. 34–35). Detalhes completos no plano do CP17.

---

## Refatoração de Backend

### B11 — Refatoração do backend: eliminar IA de features determinísticas

- **Status:** proposto
- **Esforço:** médio–grande (backend + testes + remoção de código LLM)
- **Descrição:** restringir o uso de LLM no openJob às três features que exigem **geração de linguagem natural**:
  1. Adaptação de currículo (tailoring)
  2. Criação de carta de apresentação
  3. Análise do currículo (feedback qualitativo)

  As demais features que hoje dependem de LLM — **score de fit**, **extração de keywords do currículo** e **filtro por currículo** — passam a usar **heurísticas determinísticas** baseadas em algoritmos consolidados de recuperação de informação e processamento de texto.

  **Justificativa:**
  - **Determinismo e reprodutibilidade:** a mesma vaga e o mesmo currículo produzem sempre o mesmo score, o que é essencial para auditoria, testes e confiança do usuário.
  - **Custo zero de IA:** elimina chamadas ao provedor LLM nessas etapas, reduzindo custo e latência.
  - **Explicabilidade:** um score baseado em BM25 ou TF-IDF pode ser decomposto termo a termo (“esta vaga pontuou alto porque contém ‘Kubernetes’, ‘Go’ e ‘AWS’, que estão no seu currículo”).
  - **Independência de provedor:** o app deixa de depender de uma chave de IA válida para funcionar no fluxo principal de descoberta e triagem.

  **Escopo da refatoração:**

  | Feature atual (LLM) | Substituta determinística | Estado da arte |
  |---|---|---|
  | `score_fit` (CP5/CP6/CP10) | **BM25 (Okapi)** entre currículo e descrição da vaga | BM25 é o algoritmo de referência em *resume–job matching*; supera TF-IDF por lidar com saturação de frequência e normalização por comprimento de documento. Há implementações em Go maduras: `crawlab-team/bm25`, `JLugagne/bm25`, `richardwooding/bm25`. |
  | Extração de keywords do currículo (CP6) | **YAKE** (ou RAKE como fallback) | YAKE é um extrator estatístico não supervisionado, sem treinamento, adequado a textos curtos e multilíngue (inclui PT e ES). RAKE é mais simples e bom para frases multi-palavra. Ambos têm implementações em Go (`LIAAD/yake`, `afjoseph/RAKE.Go`). |
  | Filtro por currículo (CP6) | **Similaridade de cosseno com TF-IDF** ou **BM25** com limiar configurável | TF-IDF + cosine similarity é a abordagem clássica e transparente para medir alinhamento entre currículo e vaga. O limiar (`min_matches`) vira um corte objetivo no score. |

  **Detalhes de implementação sugeridos:**
  - **BM25:** usar `Okapi BM25` com parâmetros padrão (`k1=1.2`, `b=0.75`). O “documento” é a descrição da vaga; a “query” é o texto do currículo (ou as keywords extraídas dele). O score normalizado (0–100) alimenta o `score_fit`.
  - **YAKE:** extrair top-N termos do currículo (N configurável, default 20). Esses termos substituem as `resume_keywords` que hoje podem vir do LLM. YAKE não depende de idioma — funciona para currículos em PT, EN e ES.
  - **Filtro por currículo:** usar o **mesmo** score BM25 como base. O filtro binário (`passa/não passa`) aplica um limiar (`min_matches` convertido para um score mínimo). Isso unifica score e filtro em um único pipeline determinístico.
  - **Remoção de código LLM:** eliminar as chamadas ao provedor LLM nos handlers `score_job` (CP10) e `extract_resume_keywords` (CP6). O handler `score_job` passa a ser uma função pura (sem I/O de rede), o que **elimina a necessidade de rate limiting por provedor** nessa tarefa.
  - **Compatibilidade:** manter os mesmos endpoints e contratos de API; apenas o *backend* de cálculo muda. O `score_fit` continua sendo um número 0–100; o filtro por currículo continua respeitando `min_matches` e `match_mode`.

  **O que NÃO muda (IA preservada):**
  - **Adaptação de currículo (CP4):** continua usando LLM. É geração de linguagem: reescrever bullets, adaptar tom, destacar experiências relevantes para uma vaga específica.
  - **Carta de apresentação (CP4):** continua usando LLM. É geração de texto original.
  - **Análise do currículo (CP4):** continua usando LLM. É feedback qualitativo (“seu resumo está genérico”, “faltam métricas de impacto”) que exige compreensão semântica profunda.

  **Vantagens esperadas:**
  - Score de fit determinístico e auditável.
  - Redução de custo e latência no discovery e no scoring em batch.
  - O app funciona para triagem mesmo sem chave de IA configurada (a chave passa a ser necessária apenas para tailoring/carta/análise).
  - Eliminação de uma classe de falhas (rate limit do provedor, timeout, resposta malformada do LLM) no caminho crítico de descoberta.

  **Riscos e mitigações:**
  - **Perda de nuance semântica:** BM25/TF-IDF são lexicais; não capturam sinônimos (“Kubernetes” vs “K8s”). **Mitigação:** enriquecer o pipeline com uma **taxonomia de skills** (ESCO/O*NET) ou um dicionário de sinônimos configurável, como fase 2. Nesta fase, o determinismo e a explicabilidade são priorizados sobre a cobertura semântica.
  - **Qualidade do YAKE em currículos longos:** YAKE é mais eficaz em textos curtos. **Mitigação:** extrair keywords por seção do currículo (experiência, habilidades) em vez do texto inteiro, e usar RAKE como fallback para frases multi-palavra.
  - **Calibração do limiar do filtro:** o `min_matches` atual (contagem de termos) precisa ser convertido para um limiar de score BM25. **Mitigação:** expor o limiar como um valor calibrável na UI (`/settings/discovery`) e documentar a equivalência.

- **Aceite:**
  - `score_fit` é calculado sem nenhuma chamada de rede a provedor LLM; mesma entrada → mesma saída.
  - Extração de `resume_keywords` funciona offline e em PT-BR, EN-US e ES-419.
  - Filtro por currículo usa score determinístico com limiar configurável.
  - Nenhuma chamada LLM é feita nos handlers `score_job` e `extract_resume_keywords` (verificável por log/mock).
  - Tempo de resposta do scoring < 50 ms por vaga (vs. segundos com LLM).
  - Funcionalidade preservada: tailoring, carta e análise continuam usando LLM normalmente.

- **Depende de:** nada; a base de `jobs` e `applications` já existe. Pode ser feito em paralelo ao CP16 (feed) se a prioridade for reduzir custo antes da reformulação visual.

- **Origem:** análise de arquitetura + diretriz de produto (“IA só onde agrega valor de geração”).

---

## Priorização sugerida (valor × esforço)

1. **B10** (botão discreto de cópia) — esforço mínimo de UI, ganho imediato de ergonomia no CP4.
2. **B8 / B6** (erros amigáveis sem expor logs) — dor imediata, esforço pequeno, melhora a experiência e a segurança.
3. **B1** (onboarding + links) — reduz a maior fricção de primeiro uso de IA.
4. **B9** (exportação para PDF e TXT) — fecha o ciclo do currículo adaptado para envio direto aos ATSs.
5. **B3 → B4** (presets + free tier) — "começar de graça em 3 toques".
6. **B2** (multi-perfis) — base para flexibilidade real de custo/fallback.
7. **B7** (LinkedIn OAuth) e **B5** (OAuth de provedores) — dependem de spikes de viabilidade com provedores externos e parceiros.
8. **B12 → CP17** (automação de e-mail + pipeline "em processo" + tags de fase) — maior ganho operacional no tracking; depende de CP7, CP10 e CP12 e já está detalhado em [`plan/17-email-tracking-pipeline.md`](plan/17-email-tracking-pipeline.md).