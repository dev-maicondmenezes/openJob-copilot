# Checkpoint 19 — Onboarding web de primeira execução (4–6 dias)

> Pré-requisito: CP18 concluído. O onboarding usa autenticação, perfil e currículo-base fornecidos pelo CP18. A preparação guiada acontece dentro da webapp; o bootstrap técnico do executável é tratado no CP11.

## 1. Objetivo

Entregar uma experiência gráfica moderna, elegante e consistente com o design system openJob, que conduza uma pessoa nova desde a primeira abertura até a página de vagas com a configuração essencial pronta.

Wizard e onboarding são a mesma experiência de preparação, iniciada no navegador. O fluxo deve ser guiado e validável, sem se tornar bloqueio rígido: etapas opcionais podem ser puladas e concluídas depois em Settings.

## 2. Princípios de UX

- Progressive disclosure: uma tarefa clara por etapa, indicador de progresso e resumo.
- Reutilizar serviços, formulários, componentes e validações existentes em Settings; não criar caminho de configuração divergente.
- Validar de verdade quando possível (persistência, conexão, teste de IA), com feedback de sucesso/erro e ação sugerida.
- Distinguir requisito essencial, recomendação e integração opcional.
- Permitir voltar, salvar progresso, sair e continuar depois; não perder dados ao atualizar.
- Responsivo para desktop e celular, acessível por teclado/leitor de tela e localizado conforme i18n.

## 3. Fluxo proposto

1. **Boas-vindas e conta** — explicar local-first, privacidade e o que o app faz/não faz; criar conta inicial ou entrar (CP18).
2. **Perfil profissional** — nome/headline, localização profissional opcional, experiência e skills; importar ou editar.
3. **Currículo-base** — conectar provedor disponível (OAuth e dados explicitamente acessíveis), upload de arquivo suportado ou colar texto; mostrar conteúdo extraído, permitir editar e confirmar. Não assumir que OAuth Gmail/LinkedIn retorna CV completo.
4. **Fontes de vagas** — habilitar/desabilitar fontes existentes, explicar limites e validar configuração.
5. **Filtros e preferências** — palavras-chave, filtros de currículo/mercado e preferências já suportadas; reutilizar regras e limites atuais.
6. **IA** — selecionar preset/provedor, configurar BYOK ou opção gratuita homologada, escolher perfil/modelo e executar teste real quando configurado. Reutilizar B1–B4 e catálogo de erros amigáveis.
7. **Resumo e conclusão** — mostrar o que está pronto, o que foi pulado e links para ajustar; “Ir para vagas” leva à home/feed.

## 4. Regras de conclusão

- Não exigir OAuth, Telegram, sync de e-mail ou integração opcional para entrar no app.
- Se IA não estiver configurada, permitir seguir para vagas e mostrar ação clara para configurar depois; explicar recursos que dependem de IA.
- Requisitos essenciais limitados à criação/seleção de usuário e confirmação de perfil/currículo-base conforme decisão de produto; registrar se currículo vazio é permitido.
- Persistir estado por usuário e versão do fluxo; não reapresentar completo após conclusão, mas permitir reabrir em Settings.
- Se etapa falhar, manter dados já salvos e permitir retry/skip quando opcional.

## 5. Implementação

- Criar rota/layout de onboarding separado da navegação principal, com guarda de sessão e redirecionamento por estado.
- Persistir estado no backend por usuário (onboarding_version, completed_at, current_step ou equivalente); frontend não é fonte de verdade.
- Implementar endpoints de status/progresso e conclusão usando contratos existentes.
- Integrar importação do CP18, fontes/discovery, presets/perfis de IA, teste de provedor e validações existentes.
- Não guardar tokens OAuth, API keys ou conteúdo sensível em localStorage; usar mecanismos existentes do backend.
- Não enviar telemetria externa; progresso é estado funcional local.

## 6. Critérios de verificação

- Usuário novo é encaminhado ao onboarding; usuário existente concluído vai direto para vagas.
- Fluxo desktop/mobile permite criar perfil, confirmar currículo, configurar fontes/filtros e testar IA quando disponível.
- Upload, OAuth e colagem são alternativas; falha de uma não impede as outras.
- Configuração inválida apresenta erro e ação; etapa opcional pode ser pulada sem bloquear.
- Progresso persiste por usuário e retoma após refresh ou relogin.
- Conclusão direciona à página de vagas; reabrir onboarding em Settings funciona.
- Testes garantem isolamento entre usuários, i18n, acessibilidade e snapshots visuais nas resoluções do projeto.
- make test, make vet e make test-ui passam; regressão de Settings verde.

## 7. Registrar

- ADR de experiência de primeira execução e política de etapas obrigatórias/opcionais.
- Testes e2e do fluxo feliz, skip/retry, retomada e usuário existente.
- Riscos: wizard longo, OAuth indisponível, validação sujeita a rate limit e inconsistência com Settings.

## 8. Fora de escopo

- Bootstrap/instalação do executável, configuração técnica de host/porta e abertura automática do navegador (CP11).
- Implementar APIs OAuth não aprovadas; CP18 registra gates e fallback.
- Alterar regras de discovery/scoring ou criar novas fontes de vagas.
