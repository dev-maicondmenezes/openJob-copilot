# Checkpoint 3 — SQLite + BYOK (3–4 dias)

> Origem: spec §8, Checkpoint 3, §3.2, §7. Pré-requisitos: [01-fundacao.md](01-fundacao.md), [02-spa-pwa.md](02-spa-pwa.md).

## Objetivo

Persistência local em SQLite com migrations embutidas, criptografia AES-GCM para a chave BYOK com master key local em `data/.master.key`, e UI de settings funcional com ping ao provedor de IA.

## Pré-requisitos

- Checkpoints 1 e 2 concluídos (SPA servida, container rodando).

## Tarefas

1. **Store SQLite** (`internal/store/`):
   - Driver: `modernc.org/sqlite` (sem CGO, binário estático — decisão do CP1).
   - **Interface `Repository`** definida já com os métodos previstos para `settings`, `curriculum`, `jobs`, `applications`, `notify_log` (spec §10.1).
   - Migrations versionadas embutidas (`golang-migrate` **ou** SQL puro com `//go:embed` + tabela `schema_migrations`).
   - Criar tabelas conforme spec §7: `settings`, `curriculum`, `jobs`, `applications`, `notify_log`.
   - `AUTOINCREMENT` encapsulado **dentro do store** (nada de SQL vazando para handlers — portabilidade Postgres).
2. **Criptografia BYOK** (`internal/store/crypto.go` ou pacote próprio):
   - Na primeira execução: gerar `data/.master.key` com 32 bytes aleatórios, `chmod 0600`.
   - AES-GCM para cifrar/decifrar valores em `settings`.
   - Testes: roundtrip cifra/decifra, arquivo criado com permissão correta, chave ausente → regeneração não quebra dados existentes? (decidir: erro explícito é aceitável — documentar).
3. **LLM client** (`internal/llm/`):
   - **Abstração `LLMClient`** + cliente fino OpenAI-compatible (`sashabaranov/go-openai` ou `net/http` puro) com `base_url` + `api_key` + `model` configuráveis (cobre OpenAI, Groq, Ollama, Together, Fireworks, Anthropic-compatível).
4. **Endpoints**:
   - `POST /api/settings/llm-key` — recebe `api_key` (e `base_url`, `model`), **cifra e persiste**; nunca ecoa a chave de volta.
   - `GET /api/settings/llm-key` — retorna apenas `{configured: bool}`.
   - `POST /api/settings/llm-ping` — chamada de teste ao provedor (ex.: list models ou completion de 1 token); retorna sucesso/falha legível.
5. **UI de settings** (`/settings`):
   - Campos `base_url`, `model`, `api_key` (input mascarado/password).
   - Botão "Testar conexão" consumindo `llm-ping`; feedback visual de sucesso/erro.

## Critério de verificação

- Salvar chave + base_url + model → **reiniciar container** → `GET llm-key` retorna `{configured:true}` → `llm-ping` retorna sucesso.
- Chave nunca aparece em resposta HTTP nem no frontend; no DB está cifrada (inspecionar `openjob.db`).
- `data/.master.key` existe com permissão `0600`.

## Registrar

- ADR: **criptografia BYOK** — chave local em arquivo `0600` em vez de senha mestra + PBKDF2 (suficiente para uso pessoal; spec §6).
- ADR: **cliente LLM OpenAI-compatible** em vez de SDK multi-provedor (spec §6).
- Testes críticos (spec §10.5): criptografia BYOK (roundtrip + permissões).
- README: seção sobre `data/.master.key` — "é o segredo do seu cofre local; faça backup junto com o DB; nunca o commite" (spec §3.2).

## Fora de escopo

- Tailoring e scoring (usos da chave — CP4/CP5).
