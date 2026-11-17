<script lang="ts">
	interface Option {
		value: string;
		label: string;
	}

	interface Props {
		options: readonly Option[];
		value?: string;
		label: string;
		onchange?: (value: string) => void;
	}

	let { options, value = $bindable(options[0]?.value ?? ''), label, onchange }: Props = $props();

	function select(next: string): void {
		value = next;
		onchange?.(next);
	}
</script>

<div class="segmented" role="group" aria-label={label}>
	{#each options as option (option.value)}
		<button
			type="button"
			class="segment"
			class:active={value === option.value}
			aria-pressed={value === option.value}
			onclick={() => select(option.value)}
		>
			{option.label}
		</button>
	{/each}
</div>

<style>
	.segmented {
		display: inline-flex;
		max-width: 100%;
		overflow-x: auto;
		gap: var(--space-05);
		padding: var(--space-1);
		background: var(--color-surface-elevated);
		border-radius: var(--radius-md);
	}

	.segment {
		min-height: 36px;
		padding: 0 var(--space-3);
		border: 0;
		border-radius: var(--radius-sm);
		background: transparent;
		color: var(--color-text-secondary);
		font-size: var(--text-caption);
		font-weight: var(--weight-medium);
		white-space: nowrap;
		cursor: pointer;
	}

	.segment.active {
		background: var(--color-surface);
		color: var(--color-text-primary);
		box-shadow: var(--shadow-sm);
		font-weight: var(--weight-semibold);
	}
</style>
