# ADR 0008 — Filtro Prévio de Aderência por Currículo e Palavras-chave

## Contexto

A descoberta automática de vagas consulta APIs públicas que retornam dezenas a centenas de vagas remotas.
Se todas as vagas retornadas fossem enviadas imediatamente para pontuação de fit via LLM (`score_fit` 0–100), o consumo de quota e custos de API do modelo BYOK seriam proibitivos, além de sobrecarregar limites de taxa de provedores de IA (ex.: 20 req/dia no tier gratuito do Gemini ou rate limits da OpenAI).

Muitas dessas vagas são de áreas totalmente desalinhadas do perfil do candidato (ex.: vagas de vendas, marketing, design quando o usuário é desenvolvedor Go).

## Decisão

Adotar um pipeline de filtragem em estágios progressivos antes de acionar a IA:

1. **Filtro de Palavras-chave (Search Keywords)**:
   - Configuração de até 5 termos pelo usuário em `/settings/discovery`.
   - Enviado na query das APIs com `supports_keyword_search: true`.
   - Safety net no cliente para fontes sem busca nativa ou para garantir correspondência exata com fronteiras de palavra (`\bgo\b` vs `google`).
2. **Extração Heurística Local de Keywords do Currículo (Zero Custo)**:
   - Tokenização, remoção de stopwords bilíngues (Português e Inglês) e ruídos de templates de currículo.
   - Análise de frequência de termos técnicos/competências (`top N`, default 20).
   - Cacheado em `settings.resume_keywords`, reextraído automaticamente quando solicitado ou na primeira busca.
3. **Filtro por Currículo Pré-LLM**:
   - Compara o texto da vaga (título + descrição) com as palavras-chave do currículo.
   - Requer `min_matches` (default 3) e modo `any`/`all`.
   - Vagas abaixo do limiar são descartadas da persistência e não consomem scoring.
4. **Scoring em Batch Priorizado e Rate-Limited**:
   - Vagas aprovadas nos filtros prévios são ordenadas pela quantidade de termos coincidentes com o currículo.
   - Batch limitado (`max 15` por execução) com espaçamento de requisições.

## Consequências

- **Positivas**:
  - Redução drástica (70%–90%) do volume de chamadas a provedores de LLM.
  - Vagas salvas no banco possuem alta aderência prévia ao perfil do usuário.
  - Auditoria transparente: cada vaga salva armazena `matched_keywords` e `matched_resume_terms` que justificaram sua aprovação no filtro.
- **Negativas/Trade-offs**:
  - Currículos muito resumidos ou com jargões incomuns podem exigir ajuste do limiar `min_matches` na interface (`/settings/discovery`).
