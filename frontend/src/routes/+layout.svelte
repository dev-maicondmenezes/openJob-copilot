<script lang="ts">
	import { t } from '#lib/i18n/t.js';
	import '../app.css';
	import { onMount, tick } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { registerSW } from 'virtual:pwa-register';
	import type { LayoutProps } from './$types';

	import BottomNav from '#lib/ui/BottomNav.svelte';
	import Sidebar from '#lib/ui/Sidebar.svelte';
	import ShortcutsHelp from '#lib/ui/ShortcutsHelp.svelte';
	import Icon from '#lib/ui/Icon.svelte';
	import Input from '#lib/ui/Input.svelte';
	import NotificationBell from '#lib/components/NotificationBell.svelte';
	import SettingsMenu from '#lib/components/SettingsMenu.svelte';
	import Toaster from '#lib/components/Toaster.svelte';
	import { useViewport, viewport } from '#lib/ui/media.svelte.js';
	import { sidebar } from '#lib/ui/sidebar.svelte.js';
	import { pageFooter } from '#lib/ui/footer.svelte.js';
	import { installShortcuts, onShortcut } from '#lib/shortcuts.js';
	import { initTheme } from '#lib/i18n/theme.svelte.js';
	import { initLocale, applyLocale, readStoredLocale, type Locale } from '#lib/i18n/locale.svelte.js';

	// Registra o service worker (autoUpdate): novo deploy recarrega abas abertas.
	registerSW({ immediate: true });

	let { children }: LayoutProps = $props();
	let helpOpen = $state(false);
	let search = $state('');

	useViewport();

	onMount(() => {
		// Marker usado pelos testes para esperar a hidratação/atalhos instalados.
		document.documentElement.setAttribute('data-app-ready', '1');
		initTheme();
		initLocale();
		// Preferência do servidor só quando não há cache local (CP13 §3.1).
		if (!readStoredLocale()) {
			void fetch('/api/settings')
				.then((res) => (res.ok ? res.json() : null))
				.then((body) => {
					const loc = body?.settings?.locale as Locale | undefined;
					if (loc && ['pt-BR', 'en-US', 'es-419'].includes(loc)) applyLocale(loc, false);
				})
				.catch(() => {});
		}
		sidebar.load();
		const stopShortcuts = installShortcuts();
		const offHelp = onShortcut('help', () => (helpOpen = true));
		const offSearch = onShortcut('search', () => void focusSearch());
		return () => {
			stopShortcuts();
			offHelp();
			offSearch();
		};
	});

	async function focusSearch(): Promise<void> {
		let input = document.querySelector<HTMLInputElement>('input[data-global-search]');
		if (!input && page.url.pathname !== '/jobs') {
			await goto('/jobs');
			await tick();
			input = document.querySelector<HTMLInputElement>('input[data-global-search]');
		}
		input?.focus();
		input?.select();
	}

	function submitSearch(event: SubmitEvent): void {
		event.preventDefault();
		const query = search.trim();
		void goto(query ? `/jobs?q=${encodeURIComponent(query)}` : '/jobs');
	}
</script>

<svelte:head>
	<link rel="icon" href="%sveltekit.assets%/favicon.svg" />
</svelte:head>

<a href="#main" class="skip-link">{t('nav.skip_to_content')}</a>

{#if viewport.isDesktop}
	<Sidebar />
{/if}

<div
	class="app"
	class:with-sidebar={viewport.isDesktop}
	class:rail={viewport.isDesktop && sidebar.collapsed}
	class:with-footer={Boolean(pageFooter.content)}
>
	<header class="topbar">
		<a class="brand" href="/">
			<span class="mark" aria-hidden="true">oj</span>
			<span class="brand-copy">
				<strong>{t('common.app_name')}</strong>
				<small aria-hidden="true">{t('nav.tagline')}</small>
			</span>
			<span class="visually-hidden">{t('nav.home_suffix')}</span>
		</a>

		<form class="search" role="search" onsubmit={submitSearch}>
			<Icon name="search" size={24} />
			<input
				data-global-search
				type="search"
				placeholder={t('nav.search_placeholder')}
				aria-label={t('nav.search_button')}
				class="search-input"
				bind:value={search}
			/>
		</form>

		<div class="head-actions">
			<NotificationBell />
			<SettingsMenu onShortcuts={() => (helpOpen = true)} />
		</div>
	</header>

	<main id="main" class="content" tabindex="-1">
		{@render children()}
	</main>

	{#if pageFooter.content}
		<footer class="app-footer">
			{@render pageFooter.content()}
		</footer>
	{/if}

	{#if !viewport.isDesktop}
		<BottomNav />
	{/if}
</div>

<Toaster />
<ShortcutsHelp open={helpOpen} onclose={() => (helpOpen = false)} />

<style>
	.app.with-sidebar {
		margin-left: var(--sidebar-width);
	}

	.app.with-sidebar.rail {
		margin-left: var(--sidebar-rail-width);
	}

	/* 4ª área do shell: rodapé opcional, após a página de conteúdo. */
	.app.with-footer :global(.content) {
		margin-bottom: var(--space-4);
	}

	.app-footer {
		flex-shrink: 0;
		width: calc(100% - var(--space-6));
		max-width: var(--content-max-desktop);
		margin: 0 auto calc(var(--bottom-nav-height) + var(--space-6));
	}

	@media (min-width: 768px) {
		.app-footer {
			max-width: var(--content-max-tablet);
		}
	}

	@media (min-width: 1024px) {
		.app-footer {
			/* Alinhado ao cartão da página, com o mesmo respiro de 48px. */
			width: calc(100% - (var(--space-12) * 2));
			max-width: calc(var(--content-max-desktop) - (var(--space-12) * 2));
			margin: 0 auto var(--space-12);
		}
	}

	/* Header fixo no topo (sticky) — busca e notificações sempre acessíveis. */
	.topbar {
		position: sticky;
		top: 0;
		z-index: 25;
		flex-shrink: 0;
		display: flex;
		align-items: center;
		gap: var(--space-3);
		max-width: var(--content-max-tablet);
		margin: 0 auto;
		padding: var(--space-3) var(--space-4);
		background: var(--color-surface);
		border-bottom: 1px solid var(--color-border);
	}

	.brand {
		display: inline-flex;
		align-items: center;
		gap: var(--space-3);
		color: var(--color-text-primary);
		text-decoration: none;
		min-width: 0;
	}

	.mark {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		width: 40px;
		height: 40px;
		flex-shrink: 0;
		border-radius: var(--radius-md);
		background: var(--color-primary);
		color: var(--color-on-primary);
		font-size: var(--text-caption);
		font-weight: var(--weight-bold);
	}

	.brand-copy {
		display: flex;
		flex-direction: column;
		gap: var(--space-1);
		min-width: 0;
	}

	.brand-copy strong {
		font-size: var(--text-h3);
		line-height: var(--leading-h3);
		font-weight: var(--weight-bold);
		white-space: nowrap;
	}

	.brand-copy small {
		font-size: var(--text-caption);
		line-height: var(--leading-caption);
		color: var(--color-text-muted);
	}

	.search {
		display: flex;
		align-items: center;
		gap: var(--space-2);
		flex: 1;
		min-width: 0;
		max-width: 22rem;
		margin-left: auto;
		padding: 0;
		background: transparent;
		border: 0;
		color: var(--color-text-muted);
	}

	.search:focus-within {
		color: var(--color-primary);
	}

	.search :global(input.search-input) {
		width: 100%;
		min-height: 44px;
		padding: 0;
		border: 0;
		background: transparent;
		font-size: var(--text-caption);
	}

	/* Sem moldura no campo: o foco do teclado continua visível no próprio input. */
	.search :global(input.search-input:focus) {
		outline: 2px solid var(--color-focus);
		outline-offset: var(--space-1);
		border-radius: var(--radius-sm);
	}

	.head-actions {
		display: flex;
		align-items: center;
		gap: var(--space-2);
		flex-shrink: 0;
	}

	@media (min-width: 1024px) {
		/* No desktop o header ocupa toda a largura disponível ao lado da sidebar. */
		.topbar {
			width: 100%;
			max-width: none;
			margin: 0;
			padding: var(--space-3) var(--space-8);
		}

		.search {
			margin-left: auto;
		}
	}

	/* No phone o cabeçalho prioriza busca/ações; a tagline aparece a partir do tablet. */
	@media (max-width: 767px) {
		.mark,
		.brand-copy small {
			display: none;
		}

		:global(.page-header) {
			margin-bottom: var(--space-4);
		}
	}
</style>
