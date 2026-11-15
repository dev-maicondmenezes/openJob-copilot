# ADR 0007 — Modelagem de Fontes com Capacidades Declarativas e APIs Gratuitas

## Contexto

No planejamento original do Checkpoint 6, as fontes de descoberta de vagas seriam estáticas no `config.yaml`/variáveis de ambiente, com estratégia aberta a webscraping.

Entretanto, fontes públicas variam significativamente em suas características técnicas:
1. Algumas oferecem suporte nativo a busca por palavra-chave via query params (`Himalayas`, `Remotive`), enquanto outras entregam feeds consolidados exigindo filtro client-side (`Jobicy`, `Remote OK`).
2. Algumas impõem limites de requisição rígidos (ex.: Remotive ~4 req/dia), enquanto outras exigem atribuição/backlink obrigatório.
3. Webscraping de páginas HTML em lote é frágil, bloqueável e sujeito a frequentes quebras de layout. Em contraste, fontes com APIs públicas JSON gratuitas fornecem dados estruturados, confiáveis e sem custo.

Ademais, ter fontes fixas em código ou em arquivos de configuração estática impede que o usuário habilite/desabilite fontes, altere prioridades ou insira chaves de API pela interface web.

## Decisão

1. **Eliminar webscraping do módulo de descoberta automática**:
   O módulo de descoberta (`internal/discovery`) utiliza **exclusivamente APIs públicas gratuitas** com payloads estruturados em JSON (`Himalayas`, `Remotive`, `Jobicy`, `Remote OK`). O webscraping fica restrito exclusivamente ao fluxo de inserção manual de vaga por URL pontual já concluído no CP05 (`internal/jobfetch`).
2. **Modelagem de fontes no banco com `capabilities` JSON**:
   Criar a tabela `sources` no SQLite, armazenando capacidades declarativas em formato JSON (`supports_keyword_search`, `supports_pagination`, `max_results_per_run`, `rate_limit`, `attribution_required`, etc.).
3. **Registry de Adapters em código**:
   Cada fonte possui um adaptador em Go implementando a interface comum `discovery.Adapter`. O CRUD em `/api/sources` valida que o `adapter` existe no registry antes de persistir, evitando fontes órfãs ou inexistentes.
4. **Orquestrador agnóstico de fonte**:
   O orquestrador não contém `switch/case` ou `if source == "remotive"` espalhados. Ele consulta o objeto de capacidades declarativas da fonte para decidir se envia keywords na chamada ou aplica o filtro client-side, aplica rate limiting e isola falhas (agregador tolerante).

## Consequências

- **Positivas**:
  - Usuário pode gerenciar fontes em `/settings/sources` sem reiniciar container ou editar arquivos.
  - Adição de novos adapters de APIs gratuitas não requer migrações de schema (novas capacidades residem no JSON).
  - Robustez: falha em uma API específica não interrompe a busca nas demais.
  - Segurança: chaves de API opcionais (ex.: Himalayas) são cifradas com a master key local (AES-GCM).
- **Negativas/Trade-offs**:
  - Menor volume de fontes se comparado a scrapers indiscriminados de qualquer site, porém com estabilidade e qualidade de dados ordens de grandeza superior.
