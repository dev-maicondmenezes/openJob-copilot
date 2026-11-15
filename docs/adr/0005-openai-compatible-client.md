# ADR-0005 — Cliente LLM OpenAI-compatible fino (`net/http`) em vez de SDK multi-provedor

- **Status:** aceito
- **Data:** 2026-11-13
- **Contexto:** spec §6/§10.2 exige abstração `LLMClient` para troca de provedor sem reescrita, e o usuário deve poder apontar qualquer provedor BYOK (OpenAI, Groq, Ollama, Together, Fireworks — todos expõem API OpenAI-compatible).
- **Decisão:** implementar um **cliente fino via `net/http`** em `internal/llm/openai.go` falando o dialeto OpenAI (`GET /models` para ping, `POST /chat/completions` para completions), atrás da interface `llm.Client`. `Config` (base_url + api_key + model) viaja **com cada chamada**, porque os valores vêm das settings cifradas e podem mudar em runtime sem restart.
- **Alternativas descartadas:**
  - `sashabaranov/go-openai`: madura, mas adiciona dependência e modelagem própria de erros/structs para ~2 endpoints que usamos; o cliente fino é < 150 linhas e mantém as quirks de provedor visíveis em um só arquivo.
  - SDK por provedor (Anthropic, Google…): acoplamento múltiplo e versões dessincronizadas — viola spec §10.2.
- **Riscos conhecidos:** nem todo endpoint compatível implementa `max_tokens`/parâmetros idênticos; aceitável — o pacote só usa o denominador comum e os erros de provedor são surfaced com a mensagem original do servidor.
- **Lição (CP4):** o contrato do fio é parte da interface — `llm.Message` sem tags JSON serializava `Role`/`Content` (nomes Go) em vez de `role`/`content`; o Gemini respondia 400 `INVALID_ARGUMENT` genérico. Corrigido com tags explícitas + teste de contrato que valida o JSON saído no fio. Descoberta por proxy de interceptação (mitm) entre o app e o provedor real — ferramenta recomendada para diagnosticar integrações de provedor.
- **Consequência:** zero dependências extras no `go.mod`; trocar de provedor é trocar `base_url`/`model` na UI; um futuro SDK (se necessário) entra atrás da mesma interface. Gemini usa `base_url = https://generativelanguage.googleapis.com/v1beta/openai` (o endpoint nativo `/v1beta` não aceita `Authorization: Bearer`).
