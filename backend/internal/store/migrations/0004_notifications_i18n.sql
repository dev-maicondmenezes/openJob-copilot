-- CP13 / ADR 0020: notificações passam a gravar a chave de tradução e seus
-- parâmetros. Colunas aditivas e nullable: notificações antigas mantêm
-- title/body resolvidos e seguem funcionando como fallback.
ALTER TABLE notifications ADD COLUMN i18n_key TEXT;
ALTER TABLE notifications ADD COLUMN i18n_params TEXT;
