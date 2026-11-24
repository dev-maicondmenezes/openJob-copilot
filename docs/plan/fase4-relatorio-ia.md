# Relatório — Fase 4: UX de IA e Configuração de Provedores

**Data:** 2026-11-23
**Objetivo:** Documentar alterações já realizadas da Fase 4 e planejar correções para garantir que atualizações de configuração de IA apareçam independentemente de existir um provedor configurado ou não.

---

## 1. Contexto da Fase 4

A **Fase 4** (F4) foca em "UX de IA: B1 + B3 + B4 + B2" com duração de 5–7 dias. O objetivo principal é reduzir a fricção de configuração de IA e permitir múltiplos perfis com seleção do ativo.

### Secuência interna da Fase 4:
1. **B1** — Onboarding de chave (BYOK explanation, links oficiais por provedor)
2. **B3** — Presets homologados (catálogo versionado no backend, exposto via API)
3. **B4** — Free tier (marcar presets gratuitos, sugerir modelo default gratuito)
4. **B2** — Múltiplos perfis (tabela/migration de perfis cifrados, CRUD, ativação)

---

## 2. Já Concluído (Status Atual)

### Itens já implementados ou concluídos antes da Fase 4:

| Item | Status | Localização |
|------|--------|-------------|
| **Flags de feature** | ✅ Concluído (F0) | `OPENJOB_FEATURE_*` env vars, `GET /api/features` |
| **ADRs renumerados** | ✅ Concluído (F0) | CP16: `0028–0031`, CP17: `0032–0035` |
| **B6 — Erros amigáveis de provedor** | ✅ Concluído (F2/CP17) | `backend/internal/llm/errors.go` |
| **Baseline de teste** | ✅ Concluído (F0) | `make test`, `make vet`, `make test-ui` verdes |
| **Estratégia branch/merge** | ✅ Concluído (F0) | Uma branch por fase, merge com gate |

### Itens da Fase 4 ainda pendentes/propostos (docs/backlog.md):
- **B1** — Onboarding de chave (status: proposto)
- **B2** — Múltiplos perfis (status: proposto)
- **B3** — Presets homologados (status: proposto)
- **B4** — Modelos free tier (status: proposto)
- **B5** — OAuth automático (status: proposto, alto risco)
- **B7** — LinkedIn OAuth (status: proposto, depende de spike)

---

## 3. Planejamento de Correção: IA Visível sem Provedor Configurado

### Problema Identificado
Atualizações de configuração de IA deveriam aparecer na UI **independentemente de já existir um provedor configurado ou não**. Atualmente, a UI pode falhar ou exibir estado indefinido quando não há chave/provedor configurado.

### Solução Proposta

#### 3.1 Backend: Rotas sem Dependência de Provedor

**Arquivos para modificar:**

1. **`backend/internal/llm/presets.go`**
   - Garantir retorno mesmo sem configuração ativa
   - Retornar catálogo vazio ou presets padrões em vez de erro 500
   - Expor `GET /api/settings/llm/presets` funcional mesmo com configuração ausente

2. **`backend/internal/api/settings.go`**
   - `GET /api/settings/llm` — retornar config atual (mesmo que vazia) sem crash
   - `POST /api/settings/llm/test` — retornar erro amigável "nenhuma chave configurada" ao invés de falhar crítico
   - `GET /api/settings/llm/presets` — sempre responder com estrutura válida

3. **Feature flags** — Adicionar flag `llm.enabled` ou similar para condicionar comportamentos, mas UI deve funcionar com flag desligada também

#### 3.2 Frontend: Estado Default/Placeholder

**Componentes para atualizar:**

1. **`frontend/src/routes/settings/+page.svelte`**
   - Exibir estado "nenhuma chave configurada" quando não houver provedor
   - Evitar crashes ao montar componente de configuração
   - Mostrar guião/onboarding mesmo sem chave

2. **Componentes de preset selection**
   - Mostrar catálogo de presets homologados (B3) mesmo sem provedor ativo
   - Presets devem aparecer como exemplos/demo, não como opções bloqueadas
   - Estados disabled/grayed-out quando provedor não configurado, mas visíveis

3. **Feedback de erro amigável**
   - Em vez de "internal server error", mostrar "Configure sua chave para começar"
   - Botão/CTA para iniciar onboarding (B1) sempre visível
   - Links para documentação de cada provedor sempre disponíveis

#### 3.3 Migração de Configuração Plana → Perfis

- **`POST /api/settings/llm/activate`** deve funcionar com perfil default sem chave
- Migration `config antiga vira primeiro perfil sem perda` deve funcionar mesmo quando config original está vazia
- Tabela `llm_profiles` deve permitir registro do perfil inicial sem api_key obrigatório (para effects demo)

---

## 4. Lista de Alterações Necessárias

### Backend (prioridade alta):

| # | Arquivo | Alteração |
|---|---------|-----------|
| 1 | `backend/internal/llm/presets.go` | Retornar presets padrão/vazios quando sem configuração ativa |
| 2 | `backend/internal/api/settings.go` | Handler `GET /api/settings/llm` — retornar config vazio sem erro |
| 3 | `backend/internal/api/settings.go` | Handler `POST /api/settings/llm/test` — erro amigável "sem chave" |
| 4 | `backend/internal/api/settings.go` | Handler `GET /api/settings/llm/presets` — sempre resposta válida |
| 5 | `backend/internal/store/` | Migration de perfis que permite perfil inicial sem api_key |

### Frontend (prioridade alta):

| # | Arquivo | Alteração |
|---|---------|-----------|
| 1 | `frontend/src/routes/settings/+page.svelte` | Estado loading/error quando sem chave → UI guiada |
| 2 | `frontend/src/routes/settings/*` | Componentes de preset mostrando catálogo ainda sem provedor |
| 3 | `frontend/src/lib/components/` | Mensagens "configure sua chave" em vez de crashes |
| 4 | `frontend/messages` | Textos i18n para estados "sem chave" e onboarding |

### Segurança/Privacy:

- Chave nunca deve ser requisitada/mostrada se não existir
- Estados disabled/grayed-out quando provedor não configurado, mas nunca "quebrados"
- Fallback para fluxo manual (B1) sempre disponível e visível
- Nenhum token/key exposto em mensagens de erro ou logs de usuário

---

## 5.Critérios de Aceite da Correção

- [ ] Acessar `/settings/llm` sem chave configurada → página carrega sem erros
- [ ] Catálogo de presets (B3) é exibido como exemplos, não como opções bloqueadas
- [ ] Botão "Começar de graça"/onboarding (B1) sempre visível
- [ ] `POST /api/settings/llm/test` retorna 200 com mensagem "nenhuma chave configurada" ao invés de 500
- [ ] Migration de config plana → primeiro perfil funciona mesmo com config vazia
- [ ] Nenhum stack trace ou detalhe interno vaza na UI quando provedor está ausente
- [ ] UI adapta-se corretamente quando usuário adiciona chave posteriormente

---

## 6. Próximos Passos para Execução

1. **Verificar implementação atual** das rotas API de settings/llm
2. **Adicionar handlers default** que retornam estado vazio em vez de erro
3. **Atualizar componentes frontend** para tratar estado "sem configuração"
4. **Testar fluxo**: acessar página settings sem chave configurada → deve mostrar UI guiada
5. **Validar migration** de config plana para primeiro perfil quando está vazio
6. **Rodar suítes de teste**: `make test`, `make test-ui` para garantir não-regressão

---

## 7. Relação com Outras Fases

- **Depende de:** Conclusão de F0 (flags, ADRs, baseline) ✅
- **Habilita:** F5 (Backend determinístico - B11), F6 (LinkedIn OAuth - B7), F7 (OAuth provedores - B5)
- **Independente de:** F3, F4, F5 podem ser paralelizadas
- **Gate:** Usuário leigo configura uma chave e começa a usar; usuário avançado alterna perfis sem reconfigurar

---

## 8. Execução final (2026-11-23)

A Fase 4 foi concluída com os itens B1–B4 e os seguintes refinamentos de UX:

- **Seletor de modelos por preset:** `Preset.Models` traz todos os modelos
  disponíveis no catálogo do backend; a UI renderiza um `<select>` para presets
  homologados e um campo de texto apenas em “Personalizado” (`custom`).
- **Card de ajuda do provedor:** abaixo do campo de API key, um card com link
  para a documentação oficial e outro para a tela de criação de chave do
  provedor selecionado. O card é omitido em “Personalizado”; em runtimes locais
  aparece apenas a documentação (não há chave).
- **Custom internacionalizado:** “Personalizado”/“Custom”/“Personalizado” nos
  três locales (`pt-BR`, `en-US`, `es-419`), sem textos hardcoded.
- **Múltiplos provedores:** cadastro de vários perfis com suas respectivas API
  keys, lista com badge do ativo, ações de ativar/testar/editar/remover e troca
  do ativo em um toque. Layout empilhado e botões em largura total no mobile.
- **Testes:** novos testes de API/catálogo, e2e de seletor de modelos, card de
  ajuda e troca de perfil ativo; snapshots visuais de `/settings` atualizados e
  axe sem violações sérias.
- **Gate verde:** `make vet`, `make test` e `make test-ui`.

**Status: concluída.**
