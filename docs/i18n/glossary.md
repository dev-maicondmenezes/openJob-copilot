# Glossário canônico de i18n (CP13)

> Requisito de PR: toda PR que toca strings de UI deve conferir/atualizar este
> glossário e rodar `npm run i18n:check`. A revisão de **ES-419 por falante
> nativo é gate manual obrigatório** antes do release (não automatizável).

## Termos

| PT-BR | EN-US | ES-419 | Observação |
|---|---|---|---|
| Vagas | Jobs | Empleos | “Vacantes” é sinônimo aceito no glossário; a UI usa “Empleos”. |
| Candidaturas | Applications | Postulaciones | “Aplicaciones” é rejeitado (ambíguo: apps). |
| Currículo | Resume | Currículum | “CV” aceito como forma curta. |
| Configurações | Settings | Configuración | |
| Notificações | Notifications | Notificaciones | |
| Tarefas | Tasks | Tareas | |
| Salvar | Save | Guardar | “Salvar” é rejeitado em ES-419. |
| Arquivo | File | Archivo | |
| Celular | Phone | Celular | “Móvil” é rejeitado em ES-419. |
| Score | Score | Puntaje | Nome do campo/produto; em texto corrido use “puntaje”. |
| Fonte | Source | Fuente | Refere-se a fonte de vagas, não a tipografia. |
| Status | Status | Estado | No backend o campo continua `status`. |
| Tema | Theme | Tema | Tri-state: Claro / Escuro / Sistema. |
| Idioma | Language | Idioma | Preferência de locale da interface. |

## Regras de redação ES-419

- `ustedes` (nunca `vosotros`).
- `tú` (nunca `vos` — evita voseo regional).
- Prefira “computadora”, “celular”, “aplicación”, “enlace”, “archivo”,
  “guardar”, “postulación”.
- Evite regionalismos: “ordenador”, “móvil”, “fichero”, “app” como substantivo
  principal, “coche”, “piso”.
- Formatos: datas `dd/mm/aaaa`, números `1.234,56`, moeda `US$ 1.234,56`.

## Não traduzir

- Nomes de provedores: OpenAI, Anthropic, Google, Groq, Ollama, Together,
  Fireworks.
- Nomes de fontes: Himalayas, Remotive, Jobicy, Remote OK, WWR.
- Slugs de adapter, URLs, headers HTTP, códigos de erro (`job.not_found`),
  chaves de log, chaves de mensagem e nomes de arquivo.
- Autônimos no seletor de idioma: “Português (Brasil)”, “English (US)”,
  “Español (Latinoamérica)”.

## Estrutura dos catálogos

`frontend/messages/{pt-BR,en-US,es-419}.json` com chaves hierárquicas planas:

- `jobs.*`, `applications.*`, `curriculum.*`, `notifications.*`, `tasks.*`,
  `settings.*`, `discovery.*`, `sources.*`, `appearance.*`, `shortcuts.*`,
  `nav.*`, `common.*`, `errors.*`, `devui.*`, `ui.*`.
- `en-US` é o canônico estrutural: nenhuma chave extra ou faltante nos outros.
- Parâmetros ICU usam `{name}`; pluralização usa
  `{count, plural, one {...} other {...}}`.
