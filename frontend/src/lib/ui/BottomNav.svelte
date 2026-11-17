<script lang="ts">
	import { t } from '#lib/i18n/t.js';
	import { page } from '$app/state';
	import Icon from './Icon.svelte';
	import { bottomNav, isActive } from './nav.js';
	const items = $derived(bottomNav());
</script>

<nav class="bottom-nav" aria-label={t('nav.main')}>
	{#each items as item (item.href)}
		{@const active = isActive(item.href, page.url.pathname)}
		<a
			href={item.href}
			class="item"
			class:active
			aria-current={active ? 'page' : undefined}
		>
			<Icon name={item.icon} size={24} />
			<span class="label">{item.label}</span>
		</a>
	{/each}
</nav>

<style>
	.bottom-nav {
		position: fixed;
		left: 0;
		right: 0;
		bottom: 0;
		z-index: 20;
		display: flex;
		background: var(--color-surface);
		border-top: 1px solid var(--color-border);
		padding-bottom: env(safe-area-inset-bottom);
	}

	.item {
		flex: 1;
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: var(--space-1);
		min-height: 64px;
		padding: var(--space-2) var(--space-1);
		color: var(--color-text-muted);
		text-decoration: none;
		font-size: var(--text-caption);
		line-height: 1.2;
		-webkit-tap-highlight-color: transparent;
	}

	.item.active {
		color: var(--color-primary);
		font-weight: var(--weight-semibold);
	}

	/* Tablet: mantém a bottom nav, com alvos de toque e espaçamento maiores. */
	@media (min-width: 768px) and (max-width: 1023px) {
		.item {
			min-height: 68px;
			gap: var(--space-2);
		}
	}

	/* Desktop: a sidebar substitui a bottom nav — sem layout fantasma. */
	@media (min-width: 1024px) {
		.bottom-nav {
			display: none;
		}
	}
</style>
