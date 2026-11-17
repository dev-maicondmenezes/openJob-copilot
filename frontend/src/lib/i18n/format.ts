/** Formatadores localizados (CP13 §4.7) — wrappers sobre `Intl.*`. */
import { localeState, type Locale } from './locale.svelte.js';

function locale(): Locale {
	// Leitura reativa (a função é chamada durante o render/efeito do componente).
	return localeState.current;
}

/** Locale atual para uso em `toLocale*` e `Intl.*` diretos. */
export function currentLocale(): Locale {
	return locale();
}

/** 15/11/2026 (pt-BR/es-419) · 11/15/2026 (en-US). */
export function formatDate(value: string | Date | number, options?: Intl.DateTimeFormatOptions): string {
	const date = value instanceof Date ? value : new Date(value);
	if (Number.isNaN(date.getTime())) return '—';
	return new Intl.DateTimeFormat(locale(), { day: '2-digit', month: '2-digit', year: 'numeric', ...options }).format(date);
}

/** Data + hora curtas, no locale atual. */
export function formatDateTime(value: string | Date | number): string {
	const date = value instanceof Date ? value : new Date(value);
	if (Number.isNaN(date.getTime())) return '—';
	return new Intl.DateTimeFormat(locale(), {
		day: '2-digit',
		month: 'short',
		hour: '2-digit',
		minute: '2-digit'
	}).format(date);
}

/** "há 2 minutos" / "2 minutes ago" / "hace 2 minutos". */
export function formatRelative(value: string | Date | number, now: Date = new Date()): string {
	const date = value instanceof Date ? value : new Date(value);
	if (Number.isNaN(date.getTime())) return '—';
	const diff = date.getTime() - now.getTime();
	const abs = Math.abs(diff);
	const rtf = new Intl.RelativeTimeFormat(locale(), { numeric: 'auto', style: 'long' });
	const thresholds: [Intl.RelativeTimeFormatUnit, number][] = [
		['year', 365 * 24 * 3600],
		['month', 30 * 24 * 3600],
		['week', 7 * 24 * 3600],
		['day', 24 * 3600],
		['hour', 3600],
		['minute', 60],
		['second', 1]
	];
	for (const [unit, seconds] of thresholds) {
		if (abs >= seconds || unit === 'second') {
			return rtf.format(Math.round(diff / seconds), unit);
		}
	}
	return '';
}

/** 1.234,56 (pt-BR/es-419) · 1,234.56 (en-US). */
export function formatNumber(value: number, options?: Intl.NumberFormatOptions): string {
	return new Intl.NumberFormat(locale(), options).format(value);
}

/** US$ 1.234,56 (pt-BR/es-419) · $1,234.56 (en-US). */
export function formatCurrency(value: number, currency = 'USD'): string {
	return new Intl.NumberFormat(locale(), {
		style: 'currency',
		currency,
		currencyDisplay: locale() === 'en-US' ? 'narrowSymbol' : 'symbol'
	}).format(value);
}

/** Tamanhos de arquivo com unidade localizada. */
export function formatBytes(bytes: number): string {
	if (!Number.isFinite(bytes) || bytes <= 0) return '0 B';
	const units = ['B', 'KB', 'MB', 'GB'];
	const index = Math.min(Math.floor(Math.log(bytes) / Math.log(1024)), units.length - 1);
	return `${formatNumber(bytes / 1024 ** index, { maximumFractionDigits: index === 0 ? 0 : 1 })} ${units[index]}`;
}
