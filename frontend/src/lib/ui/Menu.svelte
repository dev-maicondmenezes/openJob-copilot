<script lang="ts">
	import { t } from '#lib/i18n/t.js';
	import Icon from './Icon.svelte';

	interface MenuItem {
		label: string;
		icon?: string;
		tone?: 'default' | 'danger';
		disabled?: boolean;
		onSelect: () => void;
	}

	interface Props {
		items: readonly MenuItem[];
		label?: string;
	}

	let { items, label }: Props = $props();

	let open = $state(false);
	let root: HTMLDivElement;

	function onWindowClick(event: MouseEvent): void {
		if (open && root && !root.contains(event.target as Node)) open = false;
	}
</script>

<svelte:window onclick={onWindowClick} />

<div class="menu" bind:this={root}>
	<button
		type="button"
		class="trigger"
		aria-label={label ?? t('ui.more_actions')}
		aria-haspopup="menu"
		aria-expanded={open}
		onclick={() => (open = !open)}
		onkeydown={(e) => e.key === 'Escape' && (open = false)}
	>
		<Icon name="ellipsis" size={20} />
	</button>
	{#if open}
		<div class="list" role="menu" aria-label={label ?? t('ui.more_actions')}>
			{#each items as item (item.label)}
				<button
					type="button"
					role="menuitem"
					class="item"
					class:danger={item.tone === 'danger'}
					disabled={item.disabled}
					onclick={() => {
						open = false;
						item.onSelect();
					}}
				>
					{#if item.icon}<Icon name={item.icon} size={16} />{/if}
					{item.label}
				</button>
			{/each}
		</div>
	{/if}
</div>

<style>
	.menu {
		position: relative;
		display: inline-flex;
	}

	.trigger {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		width: var(--touch-target);
		height: var(--touch-target);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-md);
		background: var(--color-surface);
		color: var(--color-text-secondary);
		cursor: pointer;
	}

	.list {
		position: absolute;
		top: calc(100% + var(--space-1));
		right: 0;
		z-index: 30;
		min-width: 180px;
		padding: var(--space-1);
		list-style: none;
		background: var(--color-surface);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-md);
		box-shadow: var(--shadow-md);
	}

	.item {
		display: flex;
		align-items: center;
		gap: var(--space-2);
		width: 100%;
		min-height: var(--touch-target);
		padding: 0 var(--space-3);
		border: 0;
		border-radius: var(--radius-sm);
		background: none;
		color: var(--color-text-primary);
		font-size: var(--text-caption);
		text-align: left;
		cursor: pointer;
	}

	.item:hover:not(:disabled) {
		background: var(--color-surface-elevated);
	}

	.item.danger {
		color: var(--color-error);
	}

	.item:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}
</style>
