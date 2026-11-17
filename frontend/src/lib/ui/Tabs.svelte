<script lang="ts">
	interface Item {
		value: string;
		label: string;
		count?: number;
	}

	interface Props {
		items: readonly Item[];
		value?: string;
		label: string;
		onchange?: (value: string) => void;
	}

	let { items, value = $bindable(items[0]?.value ?? ''), label, onchange }: Props = $props();

	function select(next: string): void {
		value = next;
		onchange?.(next);
	}

	function onKeydown(event: KeyboardEvent, index: number): void {
		let next = index;
		if (event.key === 'ArrowRight') next = (index + 1) % items.length;
		else if (event.key === 'ArrowLeft') next = (index - 1 + items.length) % items.length;
		else if (event.key === 'Home') next = 0;
		else if (event.key === 'End') next = items.length - 1;
		else return;
		event.preventDefault();
		select(items[next].value);
		const buttons = (event.currentTarget as HTMLElement).parentElement?.querySelectorAll<HTMLButtonElement>('[role="tab"]');
		buttons?.[next]?.focus();
	}
</script>

<div class="tabs" role="tablist" aria-label={label}>
	{#each items as item, i (item.value)}
		<button
			type="button"
			role="tab"
			id="tab-{item.value}"
			aria-selected={value === item.value}
			tabindex={value === item.value ? 0 : -1}
			class="tab"
			class:active={value === item.value}
			onclick={() => select(item.value)}
			onkeydown={(e) => onKeydown(e, i)}
		>
			<span>{item.label}</span>
			{#if item.count !== undefined}<span class="count">{item.count}</span>{/if}
		</button>
	{/each}
</div>

<style>
	.tabs {
		display: flex;
		gap: var(--space-2);
		flex-wrap: wrap;
		margin-bottom: var(--space-4);
	}

	.tab {
		display: inline-flex;
		align-items: center;
		gap: var(--space-2);
		min-height: var(--touch-target);
		padding: 0 var(--space-3);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-pill);
		background: var(--color-surface);
		color: var(--color-text-secondary);
		font-size: var(--text-caption);
		font-weight: var(--weight-medium);
		cursor: pointer;
	}

	.tab.active {
		background: var(--color-info-soft);
		color: var(--color-primary);
		border-color: var(--color-primary);
		font-weight: var(--weight-semibold);
	}

	.count {
		background: var(--color-surface-elevated);
		border-radius: var(--radius-pill);
		padding: 0 var(--space-2);
		font-size: 11px;
		line-height: 18px;
	}
</style>
