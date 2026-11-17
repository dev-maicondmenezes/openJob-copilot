import { onMount } from 'svelte';
import { BREAKPOINTS, MEDIA, classifyWidth, type DeviceClass } from '../styles/tokens.js';

/**
 * Reativo a `matchMedia` com valor inicial seguro (phone) para SSR/hidratação.
 * Um único observador por consulta evita duplicação de listeners entre telas.
 */
const observers = new Map<string, { mql: MediaQueryList; subscribers: Set<(v: boolean) => void> }>();

function observe(query: string, onChange: (value: boolean) => void): () => void {
	let entry = observers.get(query);
	if (!entry) {
		const mql = window.matchMedia(query);
		entry = { mql, subscribers: new Set() };
		mql.addEventListener('change', (e) => {
			for (const fn of entry!.subscribers) fn(e.matches);
		});
		observers.set(query, entry);
	}
	entry.subscribers.add(onChange);
	onChange(entry.mql.matches);
	return () => {
		entry!.subscribers.delete(onChange);
		if (entry!.subscribers.size === 0) observers.delete(query);
	};
}

/** Estado global compartilhado de responsividade do app. */
class ViewportState {
	// SPA sem SSR: inicializar pela matchMedia evita um frame com a nav errada.
	isDesktop = $state(typeof window !== 'undefined' && window.matchMedia(MEDIA.desktop).matches);
	isTablet = $state(typeof window !== 'undefined' && window.matchMedia(MEDIA.tablet).matches);
	width = $state(typeof window !== 'undefined' ? window.innerWidth : 375);
	readonly isPhone = $derived(!this.isDesktop && !this.isTablet);
	readonly device = $derived<DeviceClass>(classifyWidth(this.width));
	private started = false;

	start(): void {
		if (this.started || typeof window === 'undefined') return;
		this.started = true;
		observe(MEDIA.desktop, (v) => (this.isDesktop = v));
		observe(MEDIA.tablet, (v) => (this.isTablet = v));
		const onResize = () => (this.width = window.innerWidth);
		onResize();
		window.addEventListener('resize', onResize, { passive: true });
	}
}

export const viewport = new ViewportState();

/** Inicia o observador global (idempotente) no montar do componente. */
export function useViewport(): void {
	onMount(() => viewport.start());
}

/** Helper para consultas pontuais fora de componentes. */
export function matches(query: string): boolean {
	return typeof window !== 'undefined' && window.matchMedia(query).matches;
}

export const breakpoints = BREAKPOINTS;
