# Backlog — openJob

> **Versão consolidada após a revisão do `release-mvp.md`.**
>
> Este documento separa: (1) correções e revisões ainda necessárias para fechar as entregas já previstas no release; e (2) evoluções posteriores ao MVP: CP18, Inteligência de documentos, Tailoring determinístico, Score Fit — Parte 2 e CP19.
>
> O plano de release continua sendo a fonte da verdade para o MVP. Este backlog não reabre funcionalidades já aceitas, não recria as antigas fases OAuth e não duplica a implementação da Fase 5/B11.

---

## 0. Última atualização — F7 hardening (2026-11-25)

**F7 (hardening e release): concluída, com publicação/tag e aceite manual pendentes.**

- CP11 integrado ao release (`release/f7-hardening`): binário único, bootstrap,
  `.deb`, imagem Docker distroless não-root, Compose e release por tag `vX.Y.Z`.
- ADR da imagem Docker renumerado de `0036` para `0043`; o `0036` permanece com
  os perfis LLM. Nenhuma colisão restante.
- Flags de release ligadas por padrão (`ui.feed`, `email.sync`,
  `curriculum.export`), com rollback por `false`/`0`; determinísticas seguem on.
- Gates verdes: `make vet`, `make test` e `make test-ui` (43 e2e, 57 a11y, 63
  visuais na imagem Playwright pinada). Migrações cobertas em banco novo e
  upgrade `0007 → 0008`; smokes de binário, `.deb` e Docker verdes.
- Pipeline de UI corrigido: `test:e2e` roda apenas o projeto `e2e` e os
  snapshots visuais usam `scripts/visual-test.sh` (imagem Playwright fixa), em
  vez de comparar baselines gerados em Debian contra o runner Ubuntu.
- `release-mvp.md`, `README.md`, `docs/plan/README.md`, `docs/backlog.md`,
  `docs/distribution.md` e `docs/release/v0.1.0.md` atualizados.
- **B5 e B7 registrados como no-go no MVP** (fallback manual mantido). Não são
  tarefas ativas aqui; a experiência de LinkedIn continua no onboarding/CP19 e
  pode ser reavaliada no CP18.
- **Ainda pendente no release:** revisões da F3 (seção 3), publicação da imagem
  Docker/GitHub Release com secrets, tag `v0.1.0` e aceite manual em celular e
  desktop.

---

## 1. Regras de organização e escopo

1. **Release primeiro:** requisitos necessários para concluir ou corrigir F1–F5 permanecem relacionados ao release, mesmo quando o respectivo gate já foi marcado como concluído. A F7 (hardening) foi concluída em 2026-11-25; restam a publicação/tag e o aceite manual.
2. **F3 parcialmente concluída:** a implementação mínima de B8/B9/B10 foi iniciada/concluída parcialmente, conforme informado, mas ainda precisa dos ajustes de UX e Markdown descritos neste backlog. Não considerar a fase integralmente aceita até validar esses pontos.
3. **F5/B11 é a base determinística:** sua entrega contempla NLP Core, extração de keywords, ranking/BM25 e filtro. Ela não contempla, por si só, Tailoring determinístico nem Inteligência de documentos.
4. **Sem fases OAuth autônomas:** a antiga fase de LinkedIn OAuth não permanece como item separado; a experiência de LinkedIn será tratada no onboarding. A antiga fase de OAuth para provedores de IA foi removida por não fazer mais sentido para o produto. Não recriar B5 ou B7 neste backlog.
5. **Reutilização:** não criar um segundo NLP Core, um segundo motor de ranking, um segundo pipeline de candidaturas ou um segundo catálogo de erros. Estender os componentes existentes por contratos bem definidos.
6. **Determinismo com limites claros:** análise, seleção de evidências, estratégia, ordenação e validação do Tailoring devem ser determinísticas. A redação em linguagem natural poderá continuar usando LLM de forma controlada, mas não pode introduzir fatos sem evidência.
7. **Privacidade:** documentos, currículos, eventos e preferências devem permanecer isolados por usuário e não devem ser enviados a serviços externos sem decisão e autorização explícitas.
8. **Sem estimativas artificiais:** esforço e calendário serão definidos após auditoria dos pacotes existentes e spikes dos pontos de maior risco.

### Status usados

- `pendente`: requisito ainda não validado.
- `em_andamento`: implementação iniciada.
- `parcial`: existe implementação mínima, mas o aceite não foi concluído.
- `bloqueado`: depende de decisão ou pré-requisito.
- `concluído`: validado por testes e aceite.
- `adiado`: mantido fora da sequência atual por decisão explícita.

---

## 2. Corte de escopo: o que não deve voltar como tarefa

| Capacidade | Destino no plano de release atual | Tratamento neste backlog |
|---|---|---|
| F0 — preparação, baseline, ADRs e flags | F0 | Não reabrir; apenas corrigir documentação caso haja inconsistência. |
| CP16 — feed, lista, detalhe, acessibilidade, sanitização e performance | F1 | Funcionalidade principal concluída. Correções de regressão e UX listadas na seção 3. |
| UI Revision do CP6 — descoberta/fontes | F1 | Concluída; não duplicar. |
| CP17/B12 — e-mail, matching, pipeline, tags e transições monotônicas | F2 | Base funcional concluída. Ajustes de política e UX do e-mail listados na seção 3. |
| B6 — catálogo de erros amigáveis de provedor | F2 | Concluído; reutilizar. |
| B8 — erros amigáveis em upload, conversão e Tailoring | F3 | Implementação mínima considerada parte da F3; validar regressões e cobertura durante o fechamento da fase. |
| B9 — exportação PDF/TXT/MD | F3 | Implementação mínima considerada parte da F3; revisar a interação e o seletor de formato na seção 3. |
| B10 — cópia do currículo adaptado | F3 | Implementação mínima considerada parte da F3; padronizar ícone e posicionamento na seção 3. |
| B1 — onboarding BYOK | F4 | Concluído; não duplicar. |
| B2 — múltiplos perfis de IA | F4 | Concluído; não duplicar. |
| B3 — catálogo de presets | F4 | Concluído; não duplicar. |
| B4 — presets free tier | F4 | Concluído; não duplicar. |
| B11 — infraestrutura determinística inicial | F5 | Entrega do release; as frentes futuras devem consumir seus contratos. |
| F6/B7 — LinkedIn OAuth | **No-go no MVP** (decisão registrada) | Não criar fluxo OAuth separado neste backlog. A experiência de LinkedIn fica no onboarding/CP19 e pode ser reavaliada no CP18. |
| B5 — OAuth de provedores de IA | **No-go no MVP** (decisão registrada) | Não manter como tarefa futura. BYOK e perfis existentes continuam sendo o caminho suportado. |
| F7 — hardening, instalação, backup/restore, segurança e publicação | F7 | **Concluída em 2026-11-25**; restam publicação/tag `v0.1.0`, secrets do Docker Hub e aceite manual. |

---

# PARTE A — Pendências para fechar e estabilizar o release

> Os itens desta parte não são novas features pós-MVP. São revisões, correções ou regras complementares em capacidades já existentes. Devem ser priorizados antes do fechamento definitivo do release.

## 3. F3 — Revisão final de currículo, cópia, exportação e Markdown

**Status:** `parcial` — implementação mínima realizada; revisão pendente.

**Objetivo:** concluir a experiência de copiar, exportar e visualizar currículo adaptado, com controles discretos e renderização consistente de Markdown.

### F3-R1 — Unificar exportação em um único botão com menu de formatos

- [ ] Substituir os botões separados de PDF, TXT e MD por **um único botão de Exportar**.
- [ ] Ao acionar o botão, abrir um menu suspenso com os formatos disponíveis: PDF, TXT e Markdown.
- [ ] Manter a geração/download específica de cada formato já implementada; esta tarefa altera principalmente a interação da interface.
- [ ] Fechar o menu após a seleção e fornecer feedback de erro/sucesso sem bloquear a página.
- [ ] Garantir navegação por teclado, foco visível, rótulos acessíveis e funcionamento responsivo.
- [ ] Preservar o conteúdo e a estrutura adequados a cada formato: PDF formatado, TXT sem marcação e MD com Markdown.

**Aceite:** existe apenas um controle visível de exportação; o usuário escolhe o formato no menu suspenso e o arquivo correto é baixado.

### F3-R2 — Padronizar os botões de copiar

- [ ] No resultado/resposta do Tailoring, usar o ícone clássico de copiar, sem texto permanente.
- [ ] No editor/visualizador do currículo, usar o mesmo ícone clássico de copiar.
- [ ] Posicionar ambos no canto superior direito da respectiva área de conteúdo.
- [ ] Manter tooltip com a ação (“Copiar”) e feedback breve após sucesso (“Copiado!”), sem deixar o texto permanentemente ocupando espaço.
- [ ] Garantir que a cópia use o conteúdo correto: Markdown original/adaptado, conforme a área em que o botão aparece.
- [ ] Implementar rótulo acessível e estado de foco; o ícone não pode ser o único nome acessível do controle.

**Aceite:** os dois lugares usam o mesmo padrão visual e comportamental, com ícone de copiar no canto superior direito.

### F3-R3 — Corrigir e padronizar a renderização Markdown

- [ ] Auditar o parser/renderizador atual e sua configuração antes de adicionar correções pontuais.
- [ ] Reconhecer `---` como divisor horizontal (`hr`) quando estiver em uma linha própria.
- [ ] Reconhecer tabelas Markdown, inclusive alinhamentos como `|:---|`, `|:---:|` e `|---:|`.
- [ ] Validar cabeçalho, separador, alinhamento de colunas e linhas de dados.
- [ ] Garantir suporte consistente aos elementos usados nos currículos: títulos, parágrafos, listas, listas numeradas, negrito, itálico, links e blocos de código quando aplicável.
- [ ] Manter o conteúdo-fonte Markdown preservado; a renderização não deve reescrever o documento salvo.
- [ ] Aplicar sanitização/allowlist para impedir execução de HTML ou links perigosos.
- [ ] Evitar CSS que corte ou esconda tabelas e conteúdo em telas menores; oferecer rolagem horizontal controlada quando necessário.

**Aceite:** divisores horizontais e tabelas com alinhamento são renderizados corretamente; o conteúdo continua seguro e legível em desktop e mobile.

### F3-R4 — Revalidar o gate da F3

- [ ] Revalidar exportação de PDF/TXT/MD depois da mudança para menu único.
- [ ] Revalidar cópia do resultado do Tailoring e do editor/visualizador.
- [ ] Revalidar mensagens amigáveis nos erros de upload, conversão de PDF e provedor de IA, reutilizando B6.
- [ ] Confirmar que os detalhes técnicos ficam nos logs do servidor e não são exibidos na interface.
- [ ] Atualizar testes E2E, acessibilidade, i18n e snapshots visuais.

**Gate F3:** exportação por menu único, cópia padronizada nos dois locais, Markdown com divisores e tabelas corretos, erros amigáveis e testes verdes.

---

## 4. Correções transversais de layout e experiência

**Status:** `pendente`.

**Objetivo:** corrigir inconsistências introduzidas por regressões e alinhar feed, lista, páginas, dashboard e componentes globais.

### UI-01 — Padronizar o início vertical das páginas

- [ ] Auditar o layout compartilhado, headers, barras de filtro e containers de conteúdo.
- [ ] Reduzir o espaço entre o fim do header e o início do conteúdo nas telas que possuem barra de filtro.
- [ ] Fazer o conteúdo dessas telas começar na mesma referência vertical usada pelas páginas sem barra de filtro.
- [ ] Corrigir o caso em que uma página sem barra de filtro ocupa mais espaço útil do que uma página que possui a barra.
- [ ] Centralizar a regra em layout/componente compartilhado quando possível, evitando margens específicas e divergentes por rota.
- [ ] Validar desktop, tablet e mobile, com e sem barra de filtro.

**Aceite:** a primeira linha útil do conteúdo começa na mesma posição visual nas páginas equivalentes, independentemente da presença de uma barra de filtro.

### UI-02 — Refatorar o modo lista para acompanhar o modo feed

- [ ] Revisar a composição dos cards/linhas do modo lista.
- [ ] Alinhar tipografia, hierarquia visual, espaçamentos, proporções, densidade e tratamento de metadados ao padrão do feed.
- [ ] Preservar as vantagens próprias da lista: leitura rápida, comparação e navegação por vários resultados.
- [ ] Evitar simplesmente transformar cada card do feed em uma linha; estabelecer uma composição própria, mas visualmente consistente.
- [ ] Validar títulos longos, empresas, localização, modalidade, score e descrições em diferentes larguras.
- [ ] Atualizar testes visuais e responsivos.

**Aceite:** lista e feed parecem modos diferentes do mesmo produto, com tipografia e proporções coerentes.

### UI-03 — Reposicionar e explicar os controles de navegação do feed

- [ ] Mover os controles de próxima/anterior para o canto inferior direito, em posição fixa em relação à viewport.
- [ ] Garantir que os controles não cubram título, descrição, score ou ações principais da vaga.
- [ ] Atribuir tooltip ao controle de próxima vaga com a tecla de atalho `ArrowDown` (seta para baixo).
- [ ] Atribuir tooltip ao controle de vaga anterior com a tecla de atalho `ArrowUp` (seta para cima).
- [ ] Garantir que os atalhos funcionem quando o foco não estiver em campo de texto, editor ou controle que dependa das setas.
- [ ] Manter foco acessível, `aria-label` e estado visível de foco.
- [ ] Testar em diferentes alturas de viewport para não sobrepor elementos fixos do browser/app.

**Aceite:** controles inferiores à direita, tooltips com as teclas corretas e nenhuma perda de conteúdo visível.

### UI-04 — Reduzir poluição visual dos botões do feed

- [ ] Converter os botões de ação do feed para apresentação somente com ícones.
- [ ] Mover o texto atual de cada ação para tooltip.
- [ ] Reduzir ligeiramente dimensões, padding e distância entre os botões, mantendo área de toque adequada no mobile.
- [ ] Manter rótulos acessíveis independentes do texto visual.
- [ ] Padronizar tamanho e estilo dos ícones.
- [ ] Garantir que tooltips não sejam a única forma de descobrir a ação em dispositivos sem hover; considerar rótulo acessível e affordance visual.

**Aceite:** ações do feed são compactas, somente ícones, reconhecíveis e utilizáveis com mouse, teclado e toque.

### UI-05 — Recolher sidebar ao entrar no feed

- [ ] Ao entrar/navegar para a tela do feed, recolher a sidebar automaticamente.
- [ ] Aplicar o comportamento em toda entrada no feed, mesmo quando a sidebar estava expandida na rota anterior.
- [ ] Preservar a possibilidade de reabrir manualmente a sidebar durante a sessão, sem impedir o comportamento de recolhimento na próxima entrada no feed.
- [ ] Verificar que o comportamento não cria loop de navegação, flicker ou perda de foco.
- [ ] Validar desktop e comportamento responsivo; no mobile, preservar a navegação própria do dispositivo.

**Aceite:** toda entrada no feed inicia com a sidebar recolhida, privilegiando a visualização de uma vaga por vez.

### UI-06 — Melhorar renderização das descrições das vagas

- [ ] Auditar formatos de descrição gerados pelas fontes de vagas e pelos extratores já existentes.
- [ ] Definir uma etapa de normalização/renderização que trate de forma consistente texto simples, Markdown e HTML permitido.
- [ ] Preservar elementos estruturais comuns: H1–H5, parágrafos, listas com marcadores, listas numeradas, negrito, itálico, links e tabelas quando existirem no conteúdo original.
- [ ] Preservar a ordem e o máximo possível do conteúdo original; não resumir nem eliminar blocos relevantes para economizar espaço.
- [ ] Padronizar espaçamentos e estilos para que descrições de fontes distintas tenham leitura semelhante.
- [ ] Aproveitar melhor a área disponível na tela, reduzindo espaços vazios e limites de altura desnecessários.
- [ ] Evitar que títulos e listas sejam achatados em texto corrido durante a extração.
- [ ] Sanitizar HTML e URLs; nunca executar scripts, handlers ou links perigosos.
- [ ] Criar fixtures de descrições com Markdown, HTML, texto puro e conteúdo parcialmente malformado.

**Aceite:** a descrição mostra o máximo de conteúdo original possível, com hierarquia e formatação consistentes e sem risco de XSS.

### UI-07 — Restaurar o dashboard do CP15

- [ ] Localizar a implementação anterior do dashboard no histórico do repositório/MRs.
- [ ] Identificar em qual alteração a rota ou os componentes foram removidos/substituídos.
- [ ] Restaurar a implementação funcional anterior, adaptando-a ao layout atual sem recriar funcionalidades já existentes.
- [ ] Preservar a rota `/dashboard` indicada no plano de release.
- [ ] Verificar navegação, dados, estados vazios/erro/loading, permissões e responsividade.
- [ ] Criar teste de rota e testes visuais para impedir nova remoção acidental.

**Aceite:** dashboard do CP15 volta a estar acessível e funcional na rota prevista, sem regressão do feed.

### UI-08 — Padronizar toasts

- [ ] Exibir um ícone correspondente ao tipo de toast antes do texto.
- [ ] Centralizar o texto no espaço útil do toast.
- [ ] Padronizar alinhamento vertical entre ícone e texto.
- [ ] Cobrir os estados usados pelo produto: sucesso, informação, aviso e erro.
- [ ] Manter contraste, tempo de leitura, responsividade e acessibilidade para leitores de tela.
- [ ] Garantir que mensagens longas quebrem linha sem deslocar o ícone ou perder centralização.

**Aceite:** todos os toasts exibem ícone à esquerda do texto e texto centralizado, mantendo legibilidade e acessibilidade.

---

## 5. Revisões funcionais complementares de F5 e F2

### F5-R1 — Permitir edição das keywords extraídas do currículo

**Status:** `pendente`.

**Objetivo:** permitir que o usuário remova termos extraídos automaticamente que não representam suas competências ou que não se aplicam ao currículo.

- [ ] Exibir as keywords atualmente extraídas em uma interface editável.
- [ ] Permitir remover individualmente termos incorretos ou irrelevantes.
- [ ] Persistir as remoções como ajustes do usuário, sem alterar o texto original do currículo.
- [ ] Distinguir a lista bruta gerada pelo extrator da lista efetiva usada no ranking.
- [ ] Ao reprocessar o currículo, não reintroduzir silenciosamente termos que o usuário removeu; definir regra clara para reaplicar/limpar ajustes quando o documento ou versão do extrator mudar.
- [ ] Permitir restaurar um termo removido, caso o usuário mude de ideia.
- [ ] Manter origem e versão do extrator para permitir diagnosticar divergências.
- [ ] Garantir que o cálculo continue determinístico para a mesma versão do currículo, do extrator e das edições do usuário.
- [ ] Integrar a edição ao fluxo de score/filtro sem introduzir chamadas LLM.
- [ ] Validar acessibilidade, i18n e persistência após recarregar a página.

**Aceite:** o usuário consegue remover e restaurar keywords; os ajustes persistem e influenciam o score/filtro de forma previsível, sem modificar o currículo-base.

### F2-R1 — Limitar configuração de e-mail a uma conta por vez

**Status:** `pendente` — regra complementar à F2/CP17.

- [ ] Permitir somente uma conta de e-mail conectada por usuário/instalação, conforme o modelo de identidade atualmente suportado.
- [ ] Aplicar a regra no backend/store, e não apenas ocultar controles na interface.
- [ ] Definir a restrição de unicidade adequada ao escopo atual e documentar como ela evolui com CP18.
- [ ] Se já existir uma conta, a interface deve apresentar seu estado e ações de testar, sincronizar, desconectar ou substituir.
- [ ] Para substituir a conta, exigir desconexão/remoção controlada da anterior antes de ativar a nova.
- [ ] Impedir que duas contas concorram por sync, cursores ou credenciais no mesmo escopo.
- [ ] Definir o que acontece com mensagens e vínculos existentes quando a conta é desconectada; não apagar histórico de candidaturas sem decisão explícita.
- [ ] Testar upgrade de banco que já contenha mais de uma conta: identificar duplicidades e exigir resolução segura, sem excluir dados silenciosamente.

**Aceite:** uma única conta pode estar conectada por escopo de usuário; API e UI impedem segunda conexão simultânea e a substituição não perde histórico de candidaturas.

### F2-R2 — Condicionar o botão de autorização Gmail ao login usado no onboarding

**Status:** `pendente`.

- [ ] Na configuração de e-mail, exibir o botão de autorização/login Gmail somente quando o usuário **não tiver usado Gmail para entrar no onboarding**.
- [ ] Quando o onboarding já tiver utilizado Gmail, não apresentar um segundo botão de login/conexão Gmail como se fosse uma conta independente.
- [ ] Verificar explicitamente quais escopos OAuth foram concedidos no onboarding. Login/identidade Gmail não implica, por si só, autorização para ler mensagens.
- [ ] Quando o onboarding já tiver concedido o escopo necessário de leitura (`gmail.readonly`), reutilizar a autorização válida de forma segura, sem solicitar login duplicado.
- [ ] Quando o onboarding utilizou Gmail, mas não concedeu o escopo de leitura de mensagens, definir um fluxo de consentimento incremental ligado à identidade já autenticada ou orientar o uso de IMAP; não presumir que o token de login permite ler a caixa postal.
- [ ] Quando o onboarding não utilizou Gmail, permitir o botão de autorização Gmail conforme o fluxo de OAuth read-only já previsto na F2.
- [ ] Não armazenar nem registrar tokens em logs; reutilizar o armazenamento cifrado já existente.
- [ ] Testar os cenários: login Gmail com escopo de leitura, login Gmail sem escopo de leitura, onboarding com outro provedor e configuração por IMAP.

**Aceite:** o botão Gmail aparece apenas quando o onboarding não usou Gmail; quando a identidade Gmail já foi usada, o app reutiliza uma autorização com escopos suficientes ou apresenta um caminho de permissão/alternativa sem criar uma segunda conexão independente.

> **Nota de implementação:** autenticação da identidade e autorização para ler e-mails são permissões diferentes. A interface deve respeitar a regra de visibilidade solicitada sem assumir que o simples login Google concede `gmail.readonly`.

---

## 6. Limpeza de consistência do plano de release

**Status:** `concluído` em 2026-11-25 (exceto a matriz das revisões da F3, que
depende da própria seção 3).

A F7 alinhou o plano de release, o backlog principal e este documento:

- [x] A definição de pronto do MVP aceita **decisão registrada** para B5/B7
  (no-go) em vez de exigir implementação; os itens mantidos são B1–B4, B6 e
  B8–B12. As revisões de UX/Markdown da F3 continuam na seção 3.
- [x] A seção de riscos não trata mais OAuth como entrega pendente: registra a
  decisão no-go e o fallback manual.
- [x] A ordem de merge não lista F6/F7 como fases de OAuth; B5/B7 não bloqueiam
  o release.
- [x] LinkedIn OAuth não é fase independente; a experiência fica no
  onboarding/CP19 e pode ser reavaliada no CP18.
- [x] Numeração harmonizada: F0–F5 + **F7 (hardening)**, sem F6 operacional; a
  visão geral, os títulos, a rastreabilidade e a ordem de merge são coerentes.
- [ ] Incluir na matriz de rastreabilidade as revisões da F3 e as correções
  complementares — permanece aberto até a seção 3 ser concluída.
- [x] B5/B7 fora do backlog ativo em `docs/backlog.md`, registrados como no-go.

**Aceite:** `release-mvp.md`, `docs/backlog.md`, matriz de fases e documentação
de planejamento descrevem o mesmo escopo e não contêm referências operacionais
às fases OAuth removidas.

---

# PARTE B — Evoluções pós-MVP

## 7. CP18 — Identidade, autenticação e isolamento multiusuário

**Status:** `proposto` — evolução pós-MVP já indicada no plano de release.

**Objetivo:** permitir usuários distintos com perfil profissional, currículo-base e dados privados isolados. CP18 é pré-requisito importante para manter eventos e personalização do Score Fit — Parte 2 independentes entre pessoas.

### Requisitos

- [ ] Definir o modelo de identidade/autenticação adequado à instalação local e aos usuários previstos.
- [ ] Implementar autenticação, ciclo de sessão e autorização no backend.
- [ ] Nunca confiar em `user_id` enviado livremente pelo frontend.
- [ ] Isolar currículo-base, perfil profissional, candidaturas, feedback, documentos e dados comportamentais por usuário.
- [ ] Definir propriedade e acesso aos perfis de IA e credenciais cifradas.
- [ ] Migrar dados existentes de instalação única para um usuário inicial sem perda de dados.
- [ ] Garantir que consultas, endpoints, workers e tarefas agendadas respeitem o contexto do usuário.
- [ ] Testar tentativas de leitura/alteração cruzada entre usuários.
- [ ] Documentar recuperação de acesso, backup e restauração com múltiplos usuários.

**Aceite:** nenhum usuário consulta ou altera dados privados de outro; migração preserva dados; API e workers aplicam a mesma regra de isolamento.

---

## 8. DI — Inteligência de documentos

**Status:** `proposto`.

**Objetivo:** transformar documentos de entrada em representações estruturadas, normalizadas e rastreáveis. Essa camada deve servir ao currículo, ao Tailoring e a outras funcionalidades sem conhecer as regras específicas desses domínios.

Inteligência de documentos não é apenas converter PDF em texto. Inclui reconhecer formato e tipo documental, reconstruir estrutura, extrair entidades e preservar a origem de cada informação.

### DI-01 — Contrato canônico de documento

- [ ] Definir modelo independente do formato de origem: `Document`, `Page`, `Section`, `Block`, `TextSpan`, `Entity` e `SourceReference`.
- [ ] Preservar texto original e texto normalizado como valores distintos.
- [ ] Preservar página, seção, bloco, offsets e, quando disponíveis, coordenadas no documento.
- [ ] Registrar idioma, MIME real, nome original, tamanho, hash/fingerprint e versão do extrator.
- [ ] Registrar confiança e avisos de extração sem transformar baixa confiança em fato.
- [ ] Versionar schema e extratores para permitir reprocessamento e comparação.
- [ ] Reutilizar tipos genéricos do NLP Core da F5/B11; não criar tokenizer, stopwords ou normalizador paralelo.

**Aceite:** consumidores distintos utilizam o mesmo contrato canônico sem depender da implementação de PDF, DOCX ou das regras do currículo.

### DI-02 — Ingestão segura e identificação de formato

- [ ] Auditar upload e conversão já entregues na F3 antes de adicionar novos caminhos.
- [ ] Detectar formato pelo conteúdo/magic bytes, não apenas pela extensão.
- [ ] Validar MIME, tamanho, páginas e limites de processamento.
- [ ] Tratar arquivo vazio, corrompido, protegido por senha ou incompatível com mensagem acionável.
- [ ] Definir limites de tempo, memória e tamanho descompactado.
- [ ] Isolar arquivos temporários e limpá-los após sucesso, erro ou cancelamento.
- [ ] Não registrar conteúdo integral dos documentos em logs.
- [ ] Manter comportamento local-first; não enviar documentos a serviços externos sem decisão e consentimento explícitos.

**Aceite:** arquivos inválidos/excessivos falham de forma controlada, sem travar workers, expor dados ou comprometer a aplicação.

### DI-03 — Adaptadores de extração por formato

- [ ] Definir matriz inicial de formatos, partindo dos já presentes: PDF, TXT e Markdown.
- [ ] Fazer spike antes de homologar DOCX ou outros formatos.
- [ ] Implementar adaptadores isolados por formato com saída no contrato canônico.
- [ ] Preservar ordem de leitura, títulos, parágrafos, listas, tabelas e separação por página quando o formato permitir.
- [ ] Identificar documentos sem camada textual e encaminhá-los à avaliação de OCR, em vez de retornar texto vazio silenciosamente.
- [ ] Documentar limitações conhecidas de cada adaptador.

**Aceite:** fixtures de formatos homologados chegam à mesma representação canônica, com estrutura e origem preservadas dentro das capacidades do formato.

### DI-04 — Reconstrução estrutural e classificação documental

- [ ] Identificar blocos, cabeçalhos, títulos, seções, listas, tabelas, colunas e ordem de leitura.
- [ ] Classificar documentos quando houver evidência suficiente: currículo, descrição de vaga, carta, certificado ou outro tipo homologado.
- [ ] Separar classificação de extração factual; identificar um arquivo como currículo não prova que os campos extraídos estejam corretos.
- [ ] Produzir confiança e encaminhar ambiguidades para confirmação.
- [ ] Permitir múltiplas seções/anexos sem perder referências de origem.

**Aceite:** a estrutura provável é identificada e ambiguidades são sinalizadas sem inventar seções ou conteúdo.

### DI-05 — Extração de entidades e proveniência

- [ ] Definir entidades úteis: organizações, cargos, períodos, formação, competências, certificações, idiomas, localização e outros campos necessários aos casos de uso.
- [ ] Para cada entidade, preservar trecho-fonte, página/seção, offsets ou coordenadas disponíveis e confiança.
- [ ] Diferenciar texto explicitamente presente, normalização e inferência.
- [ ] Não inferir datas, senioridade, certificações ou competências sem suporte documental.
- [ ] Preservar valores originais ao normalizar nomes, datas e unidades.
- [ ] Permitir correção humana quando a extração for ambígua ou de baixa confiança.
- [ ] Não extrair atributos pessoais sensíveis desnecessários ao caso de uso.

**Aceite:** todo fato relevante pode ser rastreado à fonte e correções humanas não apagam o conteúdo original.

### DI-06 — OCR condicional

- [ ] Detectar ausência/insuficiência de camada textual antes de iniciar OCR.
- [ ] Executar OCR somente nas páginas necessárias.
- [ ] Avaliar engines, idiomas, qualidade, recursos, licenças e operação local antes de escolher a implementação.
- [ ] Preferir execução local e recursos versionados; não exigir download silencioso na inicialização.
- [ ] Registrar confiança por página/bloco e sinalizar baixa qualidade.
- [ ] Permitir revisão do resultado pelo usuário.
- [ ] Identificar texto OCR como extraído, não como texto original.

**Aceite:** PDFs digitais não passam por OCR desnecessário; documentos escaneados produzem texto com indicação de confiança e origem.

### DI-07 — Persistência, reprocessamento e integração

- [ ] Definir ciclo de vida: original, extraído, normalizado e eventuais correções do usuário.
- [ ] Associar documentos/resultados ao usuário autenticado (CP18).
- [ ] Garantir idempotência por fingerprint, versão de extrator e opções de processamento.
- [ ] Permitir reprocessamento após atualização de parser sem sobrescrever correções humanas silenciosamente.
- [ ] Expor status, progresso, avisos e erros amigáveis, reutilizando B6/B8.
- [ ] Integrar resultados ao currículo e ao Tailoring sem fazer o adaptador conhecer suas regras de negócio.

**Aceite:** reprocessamento é rastreável, não duplica documentos e não destrói correções ou conteúdo anterior.

### DI-08 — Testes e gate

- [ ] Criar fixtures de PDF digital, PDF escaneado, TXT, Markdown, arquivo corrompido e documentos com tabelas/colunas.
- [ ] Testar PT-BR, EN-US e ES-419 nos idiomas previstos.
- [ ] Testar offsets, páginas, ordem de leitura e vínculo entre entidade e trecho-fonte.
- [ ] Testar arquivos grandes, limites, timeouts, falhas de parser e limpeza de temporários.
- [ ] Testar conteúdo malformado e formatos não permitidos.
- [ ] Criar golden tests versionados.
- [ ] Definir metas de performance a partir de um corpus de referência medido.

**Gate DI:** formatos homologados, contrato canônico versionado, proveniência preservada, erros controlados, privacidade validada e golden tests aprovados.

---

## 9. TD — Tailoring determinístico

**Status:** `proposto`.

**Objetivo:** adaptar o currículo a uma vaga por meio de análise, correspondência de evidências, estratégia reproduzível e validação factual. O sistema deve destacar o que realmente existe no currículo, identificar lacunas e impedir que fatos sejam inventados.

**Definição:** análise, seleção de evidências, estratégia, ordenação e validação serão determinísticas para as mesmas entradas e versões. A redação natural poderá usar o LLM já existente de forma controlada, mas nenhuma frase poderá introduzir fatos sem evidência. Para reprodução textual estrita, oferecer composição por templates/regras ou reutilização de conteúdo aprovado.

### TD-01 — Contratos de entrada e resultado

- [ ] Definir entrada para currículo-base, documento normalizado, vaga e requisitos.
- [ ] Definir saída versionada para adaptação, com blocos, ordem, texto, origem e estado de validação.
- [ ] Registrar versões do NLP Analyzer, regras, taxonomias/dicionários e estratégia.
- [ ] Preservar vínculo de cada bloco adaptado com uma ou mais fontes do currículo.
- [ ] Manter currículo original, proposta e versão aprovada separados.
- [ ] Reutilizar o contrato canônico DI e as abstrações genéricas da F5/B11.

**Aceite:** a adaptação pode ser auditada, comparada e reproduzida no nível de estratégia e seleção de evidências.

### TD-02 — Resume Analyzer

- [ ] Consumir currículo estruturado, incluindo Markdown atual e documentos processados pelo DI.
- [ ] Identificar seções, experiências, cargos, organizações, períodos, formação, skills, certificações e resultados explicitamente descritos.
- [ ] Preservar trecho original e localização para cada informação.
- [ ] Normalizar nomes e datas sem apagar valores originais.
- [ ] Diferenciar fato declarado, normalização, inferência de baixa confiança e informação ausente.
- [ ] Não preencher lacunas com suposições.

**Aceite:** informações usadas pelo Tailoring são rastreáveis ao currículo-base.

### TD-03 — Job Analyzer

- [ ] Decompor a descrição da vaga em requisitos, responsabilidades, qualificações, skills, senioridade, experiência, idiomas, modalidade e localização quando presentes.
- [ ] Preservar texto-fonte de cada requisito.
- [ ] Classificar obrigatório/desejável apenas quando houver evidência suficiente; casos incertos permanecem incertos.
- [ ] Normalizar termos usando NLP Core e recursos/taxonomias versionados.
- [ ] Distinguir requisito explícito de interpretação do sistema.

**Aceite:** cada requisito estruturado aponta para o trecho que o originou.

### TD-04 — Evidence Mapping Engine

- [ ] Relacionar requisitos da vaga às evidências do currículo.
- [ ] Classificar relações como `evidência_direta`, `evidência_relacionada`, `sem_evidência` e `confirmação_necessária`.
- [ ] Guardar trecho, seção/experiência de origem, requisito, tipo de correspondência, confiança e versão das regras.
- [ ] Distinguir correspondência lexical de sinônimo/taxonomia.
- [ ] Não converter proximidade lexical em comprovação de experiência.
- [ ] Exigir confirmação quando a relação depender de interpretação sem suporte textual.
- [ ] Não inserir competências classificadas como ausentes.

**Aceite:** toda afirmação adaptada tem evidência rastreável ou fica pendente de confirmação.

### TD-05 — Tailoring Strategy Engine

- [ ] Priorizar requisitos e evidências por regras versionadas e explicáveis.
- [ ] Selecionar experiências, skills e resultados relevantes sem alterar fatos.
- [ ] Definir seções a manter, reordenar, resumir ou destacar.
- [ ] Evitar repetição e excesso de keywords sem contexto.
- [ ] Tratar lacunas como lacunas, sem disfarçá-las como experiência.
- [ ] Produzir razões de inclusão, exclusão, prioridade e ordenação.
- [ ] Manter pesos/limiares configuráveis e calibrados por avaliação.

**Aceite:** mesma entrada e mesmas versões de regras geram a mesma estratégia e ordenação.

### TD-06 — Tailoring Generator controlado

- [ ] Receber somente evidências aprovadas e estratégia calculada.
- [ ] Delimitar fatos que podem ser reformulados e fatos que não podem ser alterados.
- [ ] Permitir reescrita de bullets, resumo e organização sem adicionar empresas, cargos, datas, ferramentas, métricas ou responsabilidades.
- [ ] Preservar números e métricas originais, salvo correção confirmada.
- [ ] Marcar conteúdo que exija confirmação antes de incorporá-lo.
- [ ] Registrar provedor/modelo e versão de prompt quando houver LLM.
- [ ] Oferecer composição por templates/regras quando for exigida reprodução textual estrita.
- [ ] Reutilizar perfis de IA da F4; não reimplementar BYOK ou gestão de provedores.

**Aceite:** o gerador não cria fatos novos e toda proposta passa pela validação seguinte.

### TD-07 — Validation Engine

- [ ] Comparar adaptação com currículo-base e estratégia.
- [ ] Detectar afirmações sem evidência, datas alteradas, números divergentes, skills adicionadas e mudanças de sentido.
- [ ] Bloquear ou sinalizar afirmações não sustentadas.
- [ ] Verificar cobertura dos requisitos prioritários sem exigir cobertura de todos.
- [ ] Gerar diff e justificativas por bloco.
- [ ] Exigir revisão humana antes de substituir/exportar a versão aprovada quando houver ambiguidade.
- [ ] Registrar aprovação e versão final sem sobrescrever currículo-base.

**Aceite:** nenhuma afirmação factual sem suporte passa silenciosamente para a versão aprovada.

### TD-08 — API, interface e integração

- [ ] Integrar motores por uma fachada/serviço de Tailoring, sem regra de negócio nos handlers.
- [ ] Preservar endpoints atuais quando possível; versionar mudanças de contrato.
- [ ] Exibir evidências, diff, lacunas e confirmações necessárias.
- [ ] Distinguir proposta gerada, proposta validada e versão aprovada.
- [ ] Reutilizar cópia, exportação, Markdown e catálogo de erros entregues/revisados na F3.
- [ ] Preservar o fluxo atual durante migração gradual.

### TD-09 — Testes e gate

- [ ] Golden tests para seleção, classificação de evidências, estratégia e ordenação.
- [ ] Testes PT-BR, EN-US e ES-419.
- [ ] Testes negativos: skill ausente, certificação não declarada, datas conflitantes, métrica inventada e senioridade inferida sem evidência.
- [ ] Verificar que cada afirmação aponta para uma fonte ou fica pendente de confirmação.
- [ ] Testar estabilidade por versão e regressão do currículo-base.
- [ ] Testar falha/indisponibilidade do LLM sem corromper o currículo-base.
- [ ] Testar autorização e isolamento por usuário (CP18).

**Gate TD:** estratégia reproduzível, evidências rastreáveis, ausência de fatos fabricados, revisão humana para ambiguidades e integração sem regressão do fluxo atual.

---

## 10. SF2 — Score Fit, Parte 2: timeline comportamental, personalização e avaliação

**Status:** `proposto` — executar após a conclusão e estabilização da Parte 1/B11.

**Objetivo:** evoluir o ranking textual determinístico para recomendação individual, considerando sinais de interesse e resultados do próprio usuário sem confundir aderência ao currículo com preferência comportamental.

A arquitetura permanece local-first e baseada em SQLite. Para o volume previsto, não introduzir Kafka, Redis, MongoDB ou outro banco exclusivamente para registrar interações.

### Pré-requisitos da Parte 1

- [ ] Coleção global de vagas deduplicadas.
- [ ] SQLite WAL e FTS5 operacionais.
- [ ] Normalização linguística determinística.
- [ ] Keywords do currículo extraídas sem LLM.
- [ ] BM25 ponderado integrado à API e aos workers.
- [ ] Versões identificáveis do corpus, analisador e ranking.
- [ ] Contratos do feed e do pipeline estabilizados.
- [ ] Identidade/autorização e isolamento por usuário (CP18) antes de habilitar múltiplas identidades comportamentais.

**Regra:** a Parte 2 não altera o score-base BM25 da Parte 1; a personalização atua depois dele, como reranking.

### SF2.1 — Timeline e captura de sinais comportamentais

#### Instrumentação

- [ ] Usar `IntersectionObserver` para identificar cards efetivamente visíveis.
- [ ] Registrar impressão qualificada quando pelo menos 50% do card estiver visível por pelo menos 1 segundo, com aba visível e janela ativa; manter os valores configuráveis.
- [ ] Usar `document.visibilityState`, `focus` e `blur` para evitar contabilizar tempo inativo.
- [ ] Usar relógio monotônico para medir duração.
- [ ] Separar tempo no card, tempo nos detalhes e tempo com conteúdo visível.
- [ ] Acumular duração localmente e enviar resumo por impressão; não emitir evento a cada segundo.
- [ ] Registrar abertura de detalhes como sinal independente.
- [ ] Não coletar movimentos de mouse, teclas digitadas ou navegação fora do OpenJob.

#### Event log

- [ ] Criar histórico imutável com `event_uuid`, usuário, vaga, sessão/impressão, tipo, horários de ocorrência/recebimento, versão do schema e payload validado.
- [ ] Eventos de exposição: `feed.impression`, `feed.card_dwell`, `job.details_opened`, `job.details_dwell`.
- [ ] Eventos de preferência: `job.liked`, `job.unliked`, `job.disliked`, `job.undisliked`, `job.saved`, `job.unsaved`, `job.hidden`.
- [ ] Eventos de pipeline: `pipeline.job_added`, `pipeline.application_submitted`, `pipeline.stage_changed`, `pipeline.application_withdrawn`, `pipeline.employer_rejected`, `pipeline.offer_received` e `pipeline.hired`.
- [ ] Distinguir exposição, interesse implícito, preferência explícita, comprometimento e resultado.
- [ ] Não interpretar ausência de clique como dislike.
- [ ] Registrar contexto de apresentação: posição, conjunto de candidatos, `search_run_id`, score-base, score exibido e versões de analyzer, corpus e ranking.

#### Persistência SQLite e projeções

- [ ] Criar `feed_sessions` para sessão, usuário, início/fim e versões de geração/ranking.
- [ ] Criar `feed_impressions` para vaga, posição, sessão, score, duração visível, abertura/duração dos detalhes e versões do mecanismo.
- [ ] Criar `interaction_events` como event log imutável com unicidade de `event_uuid`.
- [ ] Criar `user_job_feedback` como projeção do estado atual, com `UNIQUE(user_id, job_id)`.
- [ ] Manter like/dislike mutuamente exclusivos; save independente; hide distinto de dislike.
- [ ] Atualizar projeções na mesma transação da ingestão, com idempotência.
- [ ] Manter pipeline como fonte da verdade; registrar transições no event log em vez de duplicar estados.
- [ ] Criar índices por usuário/data, vaga, posição e estado de feedback; ajustar conforme consultas reais.
- [ ] Auditar `internal/events` antes de criar pacotes/tabelas com responsabilidades duplicadas.

#### Ingestão em lote e frontend

- [ ] Criar endpoint `POST /api/feed/events` para receber lotes.
- [ ] Obter identidade da sessão autenticada; nunca confiar em `user_id` livre no payload.
- [ ] Validar vínculo da impressão com usuário/vaga, tipos de eventos, timestamps e duração.
- [ ] Aplicar idempotência e limites de payload/eventos por lote.
- [ ] Atualizar event log e projeções transacionalmente.
- [ ] Fazer flush ao atingir limite, em intervalos controlados, ao ocultar a aba e ao encerrar sessão.
- [ ] Usar `sendBeacon` como mecanismo complementar, com fallback; não depender apenas de `beforeunload`.
- [ ] Garantir que falha da telemetria não impeça a exibição da timeline.

#### Controles da timeline

- [ ] Disponibilizar Gostei, Não gostei, Salvar e Ocultar como ações distintas.
- [ ] Permitir desfazer/remover feedback, preservando o histórico de eventos.
- [ ] Oferecer motivos opcionais de dislike: área, remuneração, localização, modalidade, senioridade, requisitos incompatíveis, falta de interesse ou outro.
- [ ] Não tratar Salvar como sinônimo de Gostei nem Ocultar como sinônimo de Não gostei.

**Gate SF2.1**

- [ ] Impressões representam exposição real, não apenas renderização no DOM.
- [ ] Dwell considera somente tempo ativo e visível.
- [ ] Eventos são enviados em lote e idempotentes.
- [ ] Like, dislike, save e hide são estados distintos e consistentes.
- [ ] Histórico permite reconstruir projeções.
- [ ] Posição e versão do ranking são registradas.
- [ ] Eventos do pipeline refletem transições reais.
- [ ] Falhas de telemetria não degradam a timeline.
- [ ] Dados isolados por usuário e sem coleta de navegação externa.

### SF2.2 — Personalização individual e avaliação contínua

#### Perfil individual

- [ ] Criar `user_preference_profiles` com usuário, versão do perfil/analisador e timestamps.
- [ ] Criar `user_preference_features` com tipo/chave, pesos positivos/negativos, observações, última observação e versão.
- [ ] Derivar características do conteúdo das vagas; não depender de listas manuais de profissões/tecnologias.
- [ ] Construir preferências a partir de like, dislike, save, abertura, dwell, inclusão no pipeline, candidatura e resultados observados.
- [ ] Manter exposição, interesse, comprometimento e resultado como dimensões distintas.
- [ ] Limitar influência de ações repetidas na mesma vaga.
- [ ] Aplicar deduplicação lógica e decaimento temporal; calibrar meia-vida durante avaliação.
- [ ] Isolar perfil e sinais de cada usuário.

#### Pipeline e resultados

- [ ] Tratar inclusão no pipeline e candidatura como intenção/ação, não como preferência explícita.
- [ ] Tratar entrevista, proposta e contratação como resultados contextuais.
- [ ] Tratar desistência como sinal contextual.
- [ ] Não transformar reprovação da empresa em dislike automático.
- [ ] Evitar que candidaturas em andamento ganhem prioridade indefinidamente; avaliar área própria na interface.

#### Re-ranking determinístico

- [ ] Preservar candidate generation e score BM25 como etapas independentes.
- [ ] Aplicar personalização somente depois do score-base.
- [ ] Considerar preferências positivas/negativas, similaridade com vagas avaliadas, localização/modalidade quando disponíveis, atualidade, ocultações e pipeline.
- [ ] Aplicar confiança baseada na quantidade/qualidade das observações; no cold start, manter ranking próximo do BM25.
- [ ] Manter fatores separados, configuráveis e explicáveis.
- [ ] Não fixar coeficientes antes de avaliação offline.
- [ ] Registrar score-base, score personalizado, versão do perfil e razões de reranking.
- [ ] Aplicar diversidade de empregadores/oportunidades e cobertura de preferências.
- [ ] Não alterar estatísticas globais BM25 com interações individuais.

#### Evaluation Engine

- [ ] Registrar versão do ranking, perfil, corpus, analyzer, posição, score-base, score personalizado e motivos de reranking.
- [ ] Medir `Qualified Impression Rate`, `Details Open Rate`, `Like Rate`, `Dislike Rate`, `Save Rate`, `Pipeline Conversion`, `Application Rate`, `Interview Progression`, `NDCG@K` e `Coverage`.
- [ ] Implementar métricas offline `Precision@K`, `Recall@K`, `NDCG@K` e `MRR`.
- [ ] Separar qualidade textual, preferência do usuário e resultados observados.
- [ ] Controlar viés de posição registrando exposição e posição; não avaliar apenas cliques produzidos pelo próprio ranking.
- [ ] Separar temporalmente o histórico usado para construir perfil dos eventos de avaliação.
- [ ] Não otimizar exclusivamente cliques/dwell: engajamento não equivale automaticamente à qualidade do matching.
- [ ] Gerar relatórios comparáveis entre versões.

#### Privacidade e operação

- [ ] Manter eventos e perfis no SQLite local.
- [ ] Permitir exportar e limpar histórico comportamental de um usuário.
- [ ] Permitir desabilitar personalização comportamental.
- [ ] Impedir acesso cruzado entre membros da família.
- [ ] Não usar sinais individuais para construir perfil de outro usuário.
- [ ] Não introduzir fila ou banco externo para o volume previsto.
- [ ] Manter transações curtas, escrita idempotente, índices úteis e atualização incremental/loteada.
- [ ] Não reconstruir perfil inteiro a cada abertura da timeline.
- [ ] Garantir degradação graciosa quando telemetria/personalização estiver indisponível.

**Gate SF2.2**

- [ ] Perfil individual reproduzível e isolado.
- [ ] Cold start preserva BM25 como principal sinal.
- [ ] Re-ranking determinístico, explicável e com score-base preservado.
- [ ] Diversidade e ocultações aplicadas sem contaminar BM25 global.
- [ ] Avaliação offline com separação temporal e controle de viés de exposição.
- [ ] Privacidade, exportação, limpeza e desativação validadas.
- [ ] Nenhum modelo de ML é necessário para aceitar esta entrega.

### Fora do escopo da Parte 2

- Collaborative filtering.
- Learning to Rank com LambdaMART.
- BPR e embeddings treinados.
- Exploração probabilística avançada.
- Modelos de previsão de contratação.

Essas evoluções exigem volume, representatividade e protocolos de avaliação que não fazem parte desta etapa.

---

## 11. CP19 — Onboarding gráfico de primeira execução

**Status:** `proposto` — evolução pós-MVP indicada no plano de release.

**Objetivo:** oferecer uma primeira execução guiada na webapp, com validação real das etapas e sem bloquear o uso por configurações opcionais.

- [ ] Criar onboarding gráfico de primeira execução.
- [ ] Validar cada etapa com o estado real do sistema, em vez de marcar conclusão apenas por navegação.
- [ ] Manter etapas opcionais não bloqueantes.
- [ ] Definir retomada, dispensa e reabertura do onboarding.
- [ ] Integrar configuração de identidade, perfil profissional e currículo-base conforme o escopo do CP18.
- [ ] Incorporar a experiência de LinkedIn ao fluxo de onboarding, conforme decisão do produto, sem recriar uma fase OAuth independente.
- [ ] Verificar permissões e escopos realmente concedidos; não assumir que login de identidade concede acesso a dados adicionais.
- [ ] Reutilizar onboarding BYOK e perfis de IA já entregues em F4, sem duplicar telas ou fluxos.
- [ ] Testar primeira execução, configuração parcial, retomada, dispensar etapa, acessibilidade e i18n.

**Aceite:** novo usuário entende o estado inicial, conclui a configuração essencial e pode adiar etapas opcionais sem perder acesso ao produto.

---

## 12. Arquitetura compartilhada e limites entre frentes

| Capacidade | Responsabilidade | Não deve fazer |
|---|---|---|
| NLP Core — F5/B11 | Normalização, idioma, tokenização, seções, offsets e versionamento | Conhecer regras de currículo, vaga, evidência ou personalização |
| Inteligência de documentos | Converter formatos em documentos estruturados com proveniência | Decidir como adaptar currículo a uma vaga |
| Tailoring Analyzer/Strategy | Analisar requisitos, mapear evidências e selecionar conteúdo | Reimplementar parser de PDF ou NLP básico |
| Tailoring Generator | Produzir texto sob restrições e evidências aprovadas | Criar experiências, competências ou métricas sem fonte |
| Validation Engine | Verificar factualidade, cobertura, diferenças e pendências | Substituir revisão humana em casos ambíguos |
| BM25 — Parte 1 | Medir aderência lexical currículo/vaga | Aprender preferências pessoais a partir de cliques |
| SF2 Engagement | Capturar exposições e interações reais | Alterar score-base BM25 ou inventar estado do pipeline |
| SF2 Personalization | Reordenar candidatos por preferência individual | Contaminar corpus ou estatísticas globais |
| SF2 Evaluation | Medir qualidade e regressões por versão | Otimizar somente clique/dwell |

---

## 13. Sequência recomendada

1. Fechar as correções do release descritas nas seções 3–6, respeitando os gates de cada fase.
2. Concluir CP18 — identidade e isolamento.
3. Iniciar DI-01/DI-02 — contrato canônico e ingestão segura.
4. Evoluir DI-03 a DI-08 — extração, estrutura, OCR condicional e qualidade.
5. Executar TD-01 a TD-05 — contratos, análise e estratégia determinística. Pode começar com o currículo Markdown atual, mas a cobertura de documentos complexos depende de DI.
6. Executar TD-06 a TD-09 — geração controlada, validação, interface e gate.
7. Executar SF2.1 — captura comportamental, após a Parte 1 e identidade estarem estáveis.
8. Executar SF2.2 — personalização e avaliação, somente após dados confiáveis de SF2.1.
9. Desenvolver CP19 — onboarding gráfico, coordenado com CP18 e com a decisão de integrar LinkedIn ao fluxo.

A sequência representa dependências, não estimativa de calendário. DI pode evoluir em paralelo com a fundação de CP18, desde que o isolamento por usuário seja aplicado antes de persistir dados pessoais em produção.

---

## 14. Critérios transversais de qualidade

- [ ] `make test`, `make vet` e `make test-ui` verdes nas entregas que alterarem backend/frontend.
- [ ] Migrations testadas em banco novo e upgrade, sem perda de dados.
- [ ] Testes de autorização e isolamento por usuário para dados privados.
- [ ] Golden tests para saídas determinísticas e versões de algoritmos.
- [ ] Nenhum documento, segredo ou conteúdo privado em logs.
- [ ] Erros de UI reutilizam catálogo amigável e não revelam stack/payload interno.
- [ ] Feature flags e rollback definidos para mudanças de ranking, extração e Tailoring.
- [ ] Acessibilidade, i18n PT-BR/EN-US/ES-419 e regressão visual verificadas quando aplicável.
- [ ] Atualizar ADRs, `docs/plan/README.md`, este backlog e notas de release ao concluir cada frente.

## 15. Decisões pendentes antes de estimar

- [ ] Auditar a implementação efetiva da F3 e separar o que já funciona do que precisa de correção.
- [ ] Auditar pacotes/contratos da F5/B11 para confirmar integração sem duplicação.
- [ ] Localizar a implementação perdida do dashboard no histórico do repositório.
- [ ] Auditar a configuração de email e o estado atual dos tokens/escopos Gmail antes de implementar a regra do onboarding.
- [ ] Definir a matriz de formatos da Inteligência de documentos e fazer spikes para formatos além de PDF/TXT/MD.
- [ ] Selecionar parser/OCR após comparar manutenção, licença, idiomas, qualidade e operação offline.
- [ ] Confirmar o modelo de identidade/autorização do CP18.
- [ ] Definir corpus de referência e protocolo de avaliação para Tailoring e SF2.
- [ ] Calibrar limiares, pesos e decaimento com dados de teste, sem fixar parâmetros arbitrários como definitivos.
- [x] Harmonizar referências residuais a B5/B7 e a numeração de fases no plano
  de release (concluído na F7, 2026-11-25; resta apenas a matriz das revisões
  da F3, dependente da seção 3).

> **Regra final de escopo:** requisitos novos devem entrar como proposta separada. Não ampliar uma frente em execução sem atualizar dependências, critérios de aceite e prioridade. As antigas fases OAuth não devem ser reintroduzidas como tarefas independentes.
