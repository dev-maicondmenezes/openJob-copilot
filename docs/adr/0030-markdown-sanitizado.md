# ADR 0030 — Renderização de Markdown seguro e sanitização contra XSS

**Status:** aceito (Checkpoint 16 / Fase 1 MVP)  
**Data:** 2026-11-22  
**Relaciona:** docs/plan/16-infinite-timeline-design.md, docs/plan/release-mvp.md, ADR 0028, ADR 0029  

## Contexto

Descrições de vagas no openJob provêm de múltiplas origens externas (APIs de terceiros, raspagem HTML convertida e entradas manuais de usuários). Conteúdos externos não confiáveis podem conter payloads de Cross-Site Scripting (XSS), como `<script>`, manipuladores de eventos em tags HTML (`<img onerror="...">`) ou esquemas de URI maliciosos (`javascript:`, `data:`).

## Decisão

Adotar sanitização estrita e renderização de Markdown segura no cliente:

1. **Sanitização de entidades antes do parsing:** em `MarkdownView.svelte`, todo o texto de entrada passa por escape rigoroso de caracteres HTML especiais (`&`, `<`, `>`, `"`, `'`). Tags `<script>` ou qualquer código HTML embutido são transformados em texto literal inerte, impossibilitando sua execução pelo motor do navegador.
2. **Allowlist estrita de protocolos em links:** links Markdown `[texto](url)` só são transformados em links ancorados `<a>` se o esquema da URL iniciar estritamente com `http://` ou `https://`. Protocolos como `javascript:` ou `data:` não recebem tag `<a>`.
3. **Proteção de navegação externa:** todos os links gerados recebem obrigatoriamente `target="_blank"` e `rel="noopener noreferrer"`.
4. **Sem uso de `{@html}` inseguro:** a renderização é controlada exclusivamente pelo componente de sanitização.

## Alternativas rejeitadas

- **Renderizar HTML sem sanitização com `{@html}`:** vulnerabilidade crítica de segurança (XSS).
- **Dependência de biblioteca externa pesada (como DOMPurify) no cliente:** overhead desnecessário de bundle para um conjunto fechado de tags de formatação de texto (headings, listas, ênfase, links e código inline).

## Consequências

- Proteção robusta e auditável contra XSS comprovada por testes automatizados.
- Renderização visual rica e tipografia harmoniosa com os tokens de design do openJob.
