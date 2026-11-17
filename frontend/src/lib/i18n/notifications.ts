/** Render localizado de notificações (CP13 §4.3 / ADR 0020). */
import { hasMessage, t } from './t.js';

export interface NotificacaoLocalizavel {
	i18n_key?: string | null;
	i18n_params?: Record<string, string | number | boolean | null> | string | null;
	title?: string;
	body?: string;
}

function params(item: NotificacaoLocalizavel): Record<string, string | number | boolean | null> {
	if (!item.i18n_params) return {};
	if (typeof item.i18n_params === 'string') {
		try {
			return JSON.parse(item.i18n_params) as Record<string, string | number | boolean | null>;
		} catch {
			return {};
		}
	}
	return item.i18n_params;
}

/** Título: chave i18n (nova) ou texto resolvido legado. */
export function notificationTitle(item: NotificacaoLocalizavel): string {
	if (!item.i18n_key) return item.title ?? '';
	if (hasMessage(item.i18n_key)) return t(item.i18n_key, params(item));
	const titleKey = `${item.i18n_key}.title`;
	return hasMessage(titleKey) ? t(titleKey, params(item)) : item.title ?? '';
}

/** Corpo: chave i18n (nova) ou texto resolvido legado. */
export function notificationBody(item: NotificacaoLocalizavel): string {
	if (!item.i18n_key) return item.body ?? '';
	const bodyKey = `${item.i18n_key}.body`;
	return hasMessage(bodyKey) ? t(bodyKey, params(item)) : item.body ?? '';
}
