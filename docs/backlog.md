# Backlog — openJob

> Melhorias coletadas fora do escopo dos checkpoints originais (spec §8).
> Cada item entra como checkpoint de refinamento após a fase de validação
> (CP8), a menos que um checkpoint ativo dependa dele. Ordem sugerida por
> valor × esforço.

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

## Priorização sugerida (valor × esforço)

1. **B10** (botão discreto de cópia) — esforço mínimo de UI, ganho imediato de ergonomia no CP4.
2. **B8 / B6** (erros amigáveis sem expor logs) — dor imediata, esforço pequeno, melhora a experiência e a segurança.
3. **B1** (onboarding + links) — reduz a maior fricção de primeiro uso de IA.
4. **B9** (exportação para PDF e TXT) — fecha o ciclo do currículo adaptado para envio direto aos ATSs.
5. **B3 → B4** (presets + free tier) — "começar de graça em 3 toques".
6. **B2** (multi-perfis) — base para flexibilidade real de custo/fallback.
7. **B7** (LinkedIn OAuth) e **B5** (OAuth de provedores) — dependem de spikes de viabilidade com provedores externos e parceiros.
