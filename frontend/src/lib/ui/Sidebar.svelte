<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import Icon from './Icon.svelte';
	import { onShortcut } from '../shortcuts.js';
	import { FOOTER_NAV, PRIMARY_NAV, SECONDARY_NAV, isActive } from './nav.js';
	import { sidebar } from './sidebar.svelte.js';

	const collapsed = $derived(sidebar.collapsed);

	onMount(() => onShortcut('rail', () => sidebar.toggle()));
</script>

<aside class="sidebar" class:collapsed aria-label="Navegação lateral">
	<div class="brand-row">
		<a href="/" class="brand">
			<span class="mark" aria-hidden="true">oj</span>
			<span class="word">openJob</span>
			<span class="visually-hidden">— início</span>
		</a>
		<button
			type="button"
			class="collapse-btn"
			aria-label={collapsed ? 'Expandir menu lateral' : 'Recolher menu lateral'}
			title={collapsed ? 'Expandir menu ([)' : 'Recolher menu ([)'}
			onclick={() => sidebar.toggle()}
		>
			<Icon name={collapsed ? 'panel-open' : 'panel-close'} size={20} />
		</button>
	</div>

	<nav class="nav" aria-label="Navegação principal">
		{#each PRIMARY_NAV as item (item.href)}
			{@const active = isActive(item.href, page.url.pathname)}
			<a
				href={item.href}
				class="nav-link"
				class:active
				aria-current={active ? 'page' : undefined}
				title={collapsed ? item.label : undefined}
			>
				<Icon name={item.icon} size={20} />
				<span class="text">{item.label}</span>
			</a>
		{/each}

		<div class="divider" role="separator"></div>

		{#each SECONDARY_NAV as item (item.href)}
			{@const active = isActive(item.href, page.url.pathname)}
			<a
				href={item.href}
				class="nav-link"
				class:active
				aria-current={active ? 'page' : undefined}
				title={collapsed ? item.label : undefined}
			>
				<Icon name={item.icon} size={20} />
				<span class="text">{item.label}</span>
			</a>
		{/each}
	</nav>

	<div class="footer">
		<div class="divider" role="separator"></div>
		{#each FOOTER_NAV as item (item.href)}
			{@const active = isActive(item.href, page.url.pathname)}
			<a
				href={item.href}
				class="nav-link"
				class:active
				aria-current={active ? 'page' : undefined}
				title={collapsed ? item.label : undefined}
			>
				<Icon name={item.icon} size={20} />
				<span class="text">{item.label}</span>
			</a>
		{/each}
	</div>
</aside>

<style>
	.sidebar {
		position: fixed;
		top: 0;
		bottom: 0;
		left: 0;
		z-index: 30;
		display: none;
		flex-direction: column;
		width: var(--sidebar-width);
		background: var(--color-surface);
		border-right: 1px solid var(--color-border);
		padding: var(--space-4) var(--space-3);
		transition: width var(--motion-transition) var(--ease-standard);
	}

	.sidebar.collapsed {
		width: var(--sidebar-rail-width);
	}

	.brand-row {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: var(--space-2);
		padding: 0 var(--space-2) var(--space-6);
	}

	.brand {
		display: inline-flex;
		align-items: center;
		gap: var(--space-2);
		color: var(--color-text-primary);
		text-decoration: none;
		font-weight: var(--weight-bold);
		overflow: hidden;
	}

	.mark {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		width: 32px;
		height: 32px;
		flex-shrink: 0;
		border-radius: var(--radius-md);
		background: var(--color-primary);
		color: var(--color-on-primary);
		font-size: var(--text-caption);
	}

	.word {
		white-space: nowrap;
	}

	.collapse-btn {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		width: 36px;
		height: 36px;
		flex-shrink: 0;
		border: 1px solid var(--color-border);
		border-radius: var(--radius-md);
		background: var(--color-surface);
		color: var(--color-text-muted);
		cursor: pointer;
	}

	.nav {
		display: flex;
		flex-direction: column;
		gap: var(--space-1);
		flex: 1;
	}

	.nav-link {
		display: flex;
		align-items: center;
		gap: var(--space-3);
		min-height: var(--touch-target);
		padding: 0 var(--space-3);
		border-radius: var(--radius-md);
		color: var(--color-text-secondary);
		text-decoration: none;
		font-size: var(--text-body);
		font-weight: var(--weight-medium);
		white-space: nowrap;
		overflow: hidden;
		cursor: pointer;
	}

	.nav-link:hover {
		background: var(--color-surface-elevated);
		color: var(--color-text-primary);
	}

	.nav-link.active {
		background: var(--color-info-soft);
		color: var(--color-primary);
		font-weight: var(--weight-semibold);
	}

	.divider {
		height: 1px;
		margin: var(--space-3) var(--space-2);
		background: var(--color-border);
	}

	.footer {
		display: flex;
		flex-direction: column;
	}

	.collapsed .word,
	.collapsed .text {
		display: none;
	}

	.collapsed .brand-row {
		flex-direction: column;
		gap: var(--space-2);
	}

	.collapsed .nav-link {
		justify-content: center;
		padding: 0;
	}

	@media (min-width: 1024px) {
		.sidebar {
			display: flex;
		}
	}
</style>
