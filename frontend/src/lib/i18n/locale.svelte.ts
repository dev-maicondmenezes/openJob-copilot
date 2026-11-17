/**
 * Locale da interface (CP13 / ADR 0018).
 *
 * Ordem de preferência: `localStorage.openjob.locale` → `settings.locale`
 * (buscado do backend pelo layout) → `navigator.languages`. O fallback chain
 * segue o plano: pt-* → pt-BR; es-* → es-419; demais → en-US.
 *
 * `t()` lê `localeState.current` de dentro de um componente/efeito, o que
 * torna toda a UI reativa à troca de idioma sem reload.
 */
import { setLocale as paraglideSetLocale } from '../../paraglide/runtime.js';

export const LOCALES = ['pt-BR', 'en-US', 'es-419'] as const;
export type Locale = (typeof LOCALES)[number];

export const LOCALE_STORAGE_KEY = 'openjob.locale';

/** Nomes exibidos no seletor — as próprias labels não são traduzidas (autônimo). */
export const LOCALE_OPTIONS: { value: Locale; label: string }[] = [
	{ value: 'pt-BR', label: 'Português (Brasil)' },
	{ value: 'en-US', label: 'English (US)' },
	{ value: 'es-419', label: 'Español (Latinoamérica)' }
];

export const localeState = $state<{ current: Locale }>({ current: 'en-US' });

/** Resolve um tag BCP-47 arbitrário para um dos 3 locales suportados. */
export function resolveLocale(tag: string | null | undefined): Locale {
	if (!tag) return 'en-US';
	const normalized = tag.trim().toLowerCase().replace('_', '-');
	if (normalized === 'pt' || normalized.startsWith('pt-')) return 'pt-BR';
	if (normalized === 'es' || normalized.startsWith('es-')) return 'es-419';
	if (normalized === 'en' || normalized.startsWith('en-')) return 'en-US';
	return 'en-US';
}

/** Detecta o locale preferido pelo navegador (`navigator.languages`). */
export function detectBrowserLocale(): Locale {
	if (typeof navigator === 'undefined') return 'en-US';
	const languages = navigator.languages?.length ? navigator.languages : [navigator.language];
	for (const language of languages) {
		const resolved = resolveLocale(language);
		// Só aceita a primeira entrada "útil": um idioma desconhecido não
		// deve ser ignorado em favor de um segundo idioma da lista.
		if (resolved !== 'en-US' || language.toLowerCase().startsWith('en')) return resolved;
	}
	return 'en-US';
}

export function readStoredLocale(): Locale | null {
	if (typeof localStorage === 'undefined') return null;
	try {
		const stored = localStorage.getItem(LOCALE_STORAGE_KEY);
		return stored && (LOCALES as readonly string[]).includes(stored) ? (stored as Locale) : null;
	} catch {
		return null;
	}
}

/**
 * Aplica o locale: atualiza o estado reativo, o paraglide, o `<html lang>`
 * e o cache local. Nunca recarrega a página.
 */
export function applyLocale(locale: Locale, persist = true): void {
	localeState.current = locale;
	// `reload: false` é o escape hatch browser-only do paraglide v3: a
	// reatividade das mensagens é feita por `t()` lendo `localeState`.
	paraglideSetLocale(locale, { reload: false });
	if (typeof document !== 'undefined') document.documentElement.lang = locale;
	if (persist) {
		try {
			localStorage.setItem(LOCALE_STORAGE_KEY, locale);
		} catch {
			// armazenamento bloqueado: locale segue ativo nesta sessão
		}
	}
}

/** Inicialização síncrona feita no layout (antes de qualquer render útil). */
export function initLocale(): Locale {
	const stored = readStoredLocale();
	const initial = stored ?? detectBrowserLocale();
	applyLocale(initial, stored === null);
	return initial;
}
