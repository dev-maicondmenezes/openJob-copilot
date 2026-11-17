/**
 * Helper único de tradução (CP13 §4.2).
 *
 * Proibido importar `@inlang/paraglide-js` ou `src/paraglide/*` fora de
 * `src/lib/i18n/` — use `t()` (ou os helpers deste módulo). As mensagens
 * são funções tipadas geradas em compile-time pelo paraglide; `t()` apenas
 * resolve a chave e injeta o locale reativo.
 */
import { m } from '../../paraglide/messages.js';
import { localeState, type Locale } from './locale.svelte.js';

export type MessageKey = keyof typeof m;

/** Valores aceitos como parâmetro de mensagem (ICU). */
export type MsgParams = Record<string, string | number | boolean | null | undefined>;

/** Existe uma mensagem compile-time para esta chave? */
export function hasMessage(key: string): key is MessageKey {
	return typeof (m as Record<string, unknown>)[key] === 'function';
}

/**
 * Traduz `key` no locale atual. A leitura de `localeState.current` acontece
 * dentro do chamador (componente/efeito), tornando o resultado reativo.
 *
 * Chaves desconhecidas retornam a própria chave — nunca textos em português
 * hardcoded, para que falhas apareçam no i18n:check/desenvolvimento.
 */
export function t(key: MessageKey | (string & {}), params?: MsgParams): string {
	// Leitura reativa: não mover para um `const` de módulo (ver risco no CP13).
	const locale: Locale = localeState.current;
	const fn = (m as unknown as Record<string, ((p: object, o: { locale: Locale }) => string) | undefined>)[key];
	if (!fn) return key;
	try {
		return fn(params ?? {}, { locale });
	} catch {
		return key;
	}
}

/** Traduz para um locale explícito (útil em logs/testes, não reativo). */
export function tFor(locale: Locale, key: MessageKey | (string & {}), params?: MsgParams): string {
	const fn = (m as unknown as Record<string, ((p: object, o: { locale: Locale }) => string) | undefined>)[key];
	if (!fn) return key;
	try {
		return fn(params ?? {}, { locale });
	} catch {
		return key;
	}
}
