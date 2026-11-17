<script lang="ts">
	/**
	 * Menu de configurações do cabeçalho (substitui o link "Config" da sidebar
	 * e o botão de atalhos avulso). Reúne as abas internas com ícones nomeados
	 * e o acesso aos atalhos de teclado em um único ponto de entrada.
	 */
	import { t } from '#lib/i18n/t.js';
	import { tick } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import IconButton from '#lib/ui/IconButton.svelte';
	import Icon from '#lib/ui/Icon.svelte';

	interface Props {
		onShortcuts: () => void;
	}

	let { onShortcuts }: Props = $props();

	let open = $state(false);
	let root: HTMLDivElement;

	const tabs = $derived([
		{ href: '/settings', label: t('nav.settings_llm'), icon: 'key' },
		{ href: '/settings/sources', label: t('nav.settings_sources'), icon: 'rss' },
		{ href: '/settings/discovery', label: t('nav.settings_discovery'), icon: 'sliders' },
		{ href: '/settings/notifications', label: t('nav.settings_notifications'), icon: 'bell' },
		{ href: '/settings/appearance', label: t('nav.settings_appearance'), icon: 'palette' }
	]);

	function isCurrent(href: string): boolean {
		return href === '/settings'
			? page.url.pathname === '/settings'
			: page.url.pathname.startsWith(href);
	}

	const hasActiveTab = $derived(tabs.some((tab) => isCurrent(tab.href)));

	function menuItems(): HTMLButtonElement[] {
		return root ? Array.from(root.querySelectorAll<HTMLButtonElement>('[role="menuitem"]')) : [];
	}

	async function toggle(): Promise<void> {
		open = !open;
		if (open) {
			await tick();
			menuItems()[0]?.focus();
		}
	}

	function close(restoreFocus = false): void {
		open = false;
		if (restoreFocus) root?.querySelector<HTMLButtonElement>('.trigger')?.focus();
	}

	async function navigate(href: string): Promise<void> {
		close();
		await goto(href);
	}

	function onWindowClick(event: MouseEvent): void {
		if (open && root && !root.contains(event.target as Node)) close();
	}

	function onPanelKeydown(event: KeyboardEvent): void {
		const items = menuItems();
		const index = items.indexOf(document.activeElement as HTMLButtonElement);
		switch (event.key) {
			case 'Escape':
				event.preventDefault();
				close(true);
				break;
			case 'ArrowDown':
				event.preventDefault();
				items[(index + 1) % items.length]?.focus();
				break;
			case 'ArrowUp':
				event.preventDefault();
				items[(index - 1 + items.length) % items.length]?.focus();
				break;
			case 'Home':
				event.preventDefault();
				items[0]?.focus();
				break;
			case 'End':
				event.preventDefault();
				items[items.length - 1]?.focus();
				break;
		}
	}
</script>

<svelte:window onclick={onWindowClick} />

<div class="settings-menu" bind:this={root}>
	<div class="trigger">
		<IconButton
			icon="settings"
			label={t('common.settings')}
			variant="ghost"
			iconSize={24}
			aria-haspopup="menu"
			aria-expanded={open}
			class={hasActiveTab ? 'active' : ''}
			onclick={() => void toggle()}
		/>
	</div>

	{#if open}
		<div
			class="panel"
			role="menu"
			tabindex="-1"
			aria-label={t('common.settings')}
			onkeydown={onPanelKeydown}
		>
			<p class="group-label">{t('common.settings')}</p>
			{#each tabs as tab (tab.href)}
				<button
					type="button"
					role="menuitem"
					class="item"
					class:current={isCurrent(tab.href)}
					aria-current={isCurrent(tab.href) ? 'page' : undefined}
					onclick={() => void navigate(tab.href)}
				>
					<Icon name={tab.icon} size={16} />
					<span>{tab.label}</span>
				</button>
			{/each}

			<div class="separator" role="separator"></div>

			<button
				type="button"
				role="menuitem"
				class="item"
				onclick={() => {
					close();
					onShortcuts();
				}}
			>
				<Icon name="keyboard" size={16} />
				<span>{t('nav.keyboard_shortcuts')}</span>
				<kbd>?</kbd>
			</button>
		</div>
	{/if}
</div>

<style>
	.settings-menu {
		position: relative;
		display: inline-flex;
	}

	.trigger :global(button.active) {
		color: var(--color-primary);
	}

	.panel {
		position: absolute;
		top: calc(100% + var(--space-2));
		right: 0;
		z-index: 40;
		width: min(18rem, calc(100vw - var(--space-8)));
		padding: var(--space-2);
		background: var(--color-surface);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-lg);
		box-shadow: var(--shadow-md);
	}

	.group-label {
		margin: 0 0 var(--space-1);
		padding: 0 var(--space-3);
		font-size: 11px;
		font-weight: var(--weight-semibold);
		letter-spacing: 0.04em;
		text-transform: uppercase;
		color: var(--color-text-muted);
	}

	.item {
		display: flex;
		align-items: center;
		gap: var(--space-3);
		width: 100%;
		min-height: var(--touch-target);
		padding: 0 var(--space-3);
		border: 0;
		border-radius: var(--radius-md);
		background: none;
		color: var(--color-text-primary);
		font-size: var(--text-caption);
		text-align: left;
		cursor: pointer;
	}

	.item:hover,
	.item:focus-visible {
		background: var(--color-surface-elevated);
	}

	.item.current {
		background: var(--color-info-soft);
		color: var(--color-primary);
		font-weight: var(--weight-semibold);
	}

	.item kbd {
		margin-left: auto;
		padding: var(--space-05) var(--space-2);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-sm);
		background: var(--color-surface-soft);
		color: var(--color-text-muted);
		font-family: var(--font-mono);
		font-size: var(--text-caption);
	}

	.separator {
		height: 1px;
		margin: var(--space-1) var(--space-2);
		background: var(--color-border);
	}
</style>
