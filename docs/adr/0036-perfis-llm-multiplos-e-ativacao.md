# ADR 0036 — Perfis LLM múltiplos, catálogo de presets e ativação espelhada

**Status:** aceito (Fase 4 MVP)

**Data:** 2026-11-23
**Relaciona:** docs/plan/release-mvp.md, docs/backlog.md (B1–B4), ADR 0005, ADR 0021

## Contexto

A configuração de IA nasceu como três chaves planas (`llm.api_key`,
`llm.base_url`, `llm.model`). Isso atende ao BYOK inicial, mas força o usuário a
reconfigurar tudo para alternar entre provedores e não oferece um caminho guiado
para quem nunca criou uma API key. O backlog B1–B4 pede onboarding, presets
homologados, indicação de free tier e múltiplos perfis com um ativo, mantendo a
operação simples no celular.

## Decisão

1. **Catálogo de presets no backend, versionado.** `internal/llm/presets.go`
   mantém os provedores homologados (OpenAI-compatible) com `base_url`, lista de
   modelos, link de documentação, link oficial de criação de chave e flags de
   free tier/local. A UI nunca hardcodeia provedores: consome
   `GET /api/settings/llm/presets`. O preset é apenas prefill; o modo
   “Personalizado” continua permitindo `base_url` e modelo livres.
2. **Perfis em tabela própria.** A migração `0007_llm_profiles.sql` cria
   `llm_profiles` (`id`, `name`, `config` JSON, `is_default`, timestamps). O JSON
   guarda `base_url`, `model`, `preset_id` (dica de UI) e `sealed_key` — a chave
   continua cifrada em AES-GCM e nunca sai do servidor.
3. **Um único perfil ativo.** Um índice único parcial garante no máximo um
   `is_default = 1`. `POST /api/settings/llm/activate` alterna o ativo em uma
   transação e espelha o perfil nas chaves planas `llm.*`, de modo que tailoring,
   scoring, classificação de e-mail e discovery passam a usá-lo imediatamente,
   sem restart nem reconfiguração.
4. **Migração preguiçosa e não destrutiva.** `MigrateFromFlatConfig` converte a
   configuração plana existente no primeiro perfil ativo apenas quando ainda não
   há perfis; instalação nova permanece com zero perfis e a UI mostra o estado de
   onboarding.
5. **Chave write-only.** Nenhuma resposta ecoa a chave ou o material selado.
   Atualizações com `api_key` vazio preservam a chave existente; runtimes locais
   usam o placeholder `local-no-key` porque o cliente OpenAI-compatible sempre
   envia um bearer token.

## Alternativas rejeitadas

- **Manter tudo em `settings` planas com prefixos por perfil:** dificulta
  atomicidade da ativação e a garantia de um único ativo.
- **Presets fixos no frontend:** duplica catálogo, desatualiza modelos e impede
  evoluir links oficiais sem novo build da UI.
- **Apagar as chaves planas após a migração:** quebraria consumidores legados e
  rollback; o espelhamento mantém compatibilidade.

## Consequências

- Usuário leigo começa de graça com poucos toques; usuário avançado alterna
  provedores e testa cada perfil sem repetir a configuração.
- Consumidores existentes continuam lendo `llm.*` e não precisam de mudança.
- O catálogo e os links oficiais evoluem no backend; a UI apenas renderiza.
- A chave permanece local, cifrada e write-only, preservando o modelo de
  segurança do BYOK (ADR 0005/0021).
