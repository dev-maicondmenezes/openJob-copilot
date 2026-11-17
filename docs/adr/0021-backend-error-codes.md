# ADR 0021 — Erros de API com `code` + `params` + `message`

**Status:** aceito (Checkpoint 13)
**Data:** 2026-11-17
**Relaciona:** docs/plan/13-dark-mode-i18n.md, internal/api/server.go

## Contexto

O backend devolve erros em português no campo `error` e o frontend exibia esse
texto cru. Para internacionalizar sem traduzir centenas de `fmt.Errorf` em Go,
era necessário um contrato aditivo que preservasse clients antigos.

## Decisão

`writeAPIError(w, status, code, message, params)` emite:

```json
{
  "code": "job.not_found",
  "params": { "id": 999 },
  "message": "job not found",
  "error": "job not found"
}
```

- `code`: identificador estável do domínio (`job.not_found`,
  `application.not_found`, `errors de validação`, etc.), mapeado no frontend
  para `errors.<code>` do catálogo paraglide.
- `params`: valores para ICU (ex.: `id`), usados na mensagem traduzida.
- `message`: **sempre inglês legível**, fallback seguro para códigos
  desconhecidos pelo frontend.
- `error`: mesmo valor de `message`, mantido por compatibilidade retroativa.

Somente erros **de domínio voltados ao usuário** recebem código: jobs,
applications, discovery, settings, sources, tasks, notifications e curriculum.
Erros internos/debug continuam com `error` em inglês e `500`.

Critical acceptance: `GET /api/jobs/999` → 404 com `code=job.not_found` e
`params.id=999`; o frontend exibe “Vaga não encontrada” / “Job not found” /
“Empleo no encontrado” conforme o locale.

## Alternativas rejeitadas

- **Backend devolver texto já traduzido por `Accept-Language`**: locale é uma
  preferência do cliente; duplicaria catálogos em Go e JS.
- **Traduzir `error` no frontend por substring**: frágil e não versionável.
- **Trocar `error` por `code` (quebrando clients)**: contrato aditivo é
  requisito do CP13 (§2).

## Consequências

- Novos erros voltados ao usuário devem usar `writeAPIError` e ganhar uma
  chave `errors.<code>` nos 3 catálogos.
- Código desconhecido cai em `message` (EN), nunca em uma chave crua.
- Logs do backend permanecem em inglês (developer-facing, fora de escopo).
