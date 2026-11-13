# ADR-0005 — Cliente LLM OpenAI-compatible fino (`net/http`) em vez de SDK multi-provedor

- **Status:** aceito
- **Data:** 2026-11-13
- **Contexto:** spec §6/§10.2 exige abstração `LLMClient` para troca de provedor sem reescrita, e o usuário deve poder apontar qualquer provedor BYOK (OpenAI, Groq, Ollama, Together, Fireworks — todos expõem API OpenAI-compatible).
- **Decisão:** implementar um **cliente fino via `net/http`** em `internal/llm/openai.go` falando o dialeto OpenAI (`GET /models` para ping, `POST /chat/completions` para completions), atrás da interface `llm.Client`. `Config` (base_url + api_key + model) viaja **com cada chamada**, porque os valores vêm das settings cifradas e podem mudar em runtime sem restart.
- **Alternativas descartadas:**
  - `sashabaranov/go-openai`: madura, mas adiciona dependência e modelagem própria de erros/structs para ~2 endpoints que usamos; o cliente fino é < 150 linhas e mantém as quirks de provedor visíveis em um só arquivo.
  - SDK por provedor (Anthropic, Google…): acoplamento múltiplo e versões dessincronizadas — viola spec §10.2.
- **Riscos conhecidos:** nem todo endpoint compatível implementa `max_tokens`/parâmetros idênticos; aceitável — o pacote só usa o denominador comum e os erros de provedor são surfaced com a mensagem original do servidor.
- **Consequência:** zero dependências extras no `go.mod`; trocar de provedor é trocar `base_url`/`model` na UI; um futuro SDK (se necessário) entra atrás da mesma interface.
