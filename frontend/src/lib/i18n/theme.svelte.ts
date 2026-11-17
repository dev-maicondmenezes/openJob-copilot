/**
 * Tema da interface (CP13 / ADR 0019).
 *
 * Tri-state `light | dark | system`, persistido por dispositivo em
 * `localStorage.openjob.theme`. O modo `system` acompanha
 * `prefers-color-scheme` em tempo real (sem reload).
 *
 * O valor resolvido é sempre `light` ou `dark` e é aplicado como
 * `data-theme` no <html>. O script inline em `app.html` aplica o mesmo
 * valor antes do CSS carregar (anti-FOUC) — este módulo apenas mantém
 * o estado reativo do Svelte em sincronia.
 */

export type ThemeMode = 'light' | 'dark' | 'system';
export type ResolvedTheme = 'light' | 'dark';

export const THEME_STORAGE_KEY = 'openjob.theme';

const DARK_QUERY = '(prefers-color-scheme: dark)';

export const themeState = $state<{ mode: ThemeMode; resolved: ResolvedTheme }>({
	mode: 'system',
	resolved: 'light'
});

let media: MediaQueryList | null = null;
let listening = false;

function systemPrefersDark(): boolean {
	return typeof window !== 'undefined' && window.matchMedia(DARK_QUERY).matches;
}

function resolve(mode: ThemeMode): ResolvedTheme {
	return mode === 'system' ? (systemPrefersDark() ? 'dark' : 'light') : mode;
}

/** Aplica o tema ao documento. Seguro para chamar antes/after do mount. */
export function applyResolvedTheme(resolved: ResolvedTheme): void {
	if (typeof document === 'undefined') return;
	document.documentElement.dataset.theme = resolved;
	document.documentElement.style.colorScheme = resolved;
}

export function refreshTheme(): void {
	themeState.resolved = resolve(themeState.mode);
	applyResolvedTheme(themeState.resolved);
}

function onSystemChange(event: MediaQueryListEvent): void {
	if (themeState.mode === 'system') {
		themeState.resolved = event.matches ? 'dark' : 'light';
		applyResolvedTheme(themeState.resolved);
	}
}

/** Inicializa o tema a partir do localStorage e registra o listener do SO. */
export function initTheme(): void {
	if (typeof window === 'undefined') return;
	let stored: string | null = null;
	try {
		stored = localStorage.getItem(THEME_STORAGE_KEY);
	} catch {
		// armazenamento bloqueado: usa o padrão `system`
	}
	themeState.mode = stored === 'light' || stored === 'dark' || stored === 'system' ? stored : 'system';

	if (!listening && window.matchMedia) {
		media = window.matchMedia(DARK_QUERY);
		media.addEventListener('change', onSystemChange);
		listening = true;
	}
	refreshTheme();
}

/** Troca o tema, persiste e atualiza o DOM (sem reload). */
export function setTheme(mode: ThemeMode): void {
	themeState.mode = mode;
	try {
		localStorage.setItem(THEME_STORAGE_KEY, mode);
	} catch {
		// armazenamento bloqueado: tema ainda é aplicado nesta sessão
	}
	refreshTheme();
}

/** Para uso em testes / HMR. */
export function destroyTheme(): void {
	if (media && listening) media.removeEventListener('change', onSystemChange);
	listening = false;
	media = null;
}
