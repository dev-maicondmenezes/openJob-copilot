<script lang="ts">
	import Icon from './Icon.svelte';

	interface Option {
		value: string;
		label: string;
	}

	interface Props {
		label: string;
		options: readonly Option[];
		/** Array vazio = "todos". */
		value?: string[];
		allLabel?: string;
		onchange?: () => void;
	}

	let { label, options, value = $bindable([]), allLabel = 'Todas', onchange }: Props = $props();

	let open = $state(false);
	let root: HTMLDivElement;

	const summary = $derived(
		value.length === 0
			? allLabel
			: value.map((item) => options.find((option) => option.value === item)?.label ?? item).join(', ')
	);

	function toggle(option: string): void {
		value = value.includes(option) ? value.filter((item) => item !== option) : [...value, option];
		onchange?.();
	}

	function selectAll(): void {
		value = [];
		onchange?.();
	}

	function onWindowClick(event: MouseEvent): void {
		if (open && root && !root.contains(event.target as Node)) open = false;
	}
</script>

<svelte:window onclick={onWindowClick} />

<div class="multi" bind:this={root}>
	<button
		type="button"
		class="trigger"
		aria-haspopup="listbox"
		aria-expanded={open}
		title={summary}
		onclick={() => (open = !open)}
		onkeydown={(event) => event.key === 'Escape' && (open = false)}
	>
		<span class="visually-hidden">{label}:</span>
		<span class="summary">{summary}</span>
		<Icon name="arrow-down" size={16} />
	</button>

	{#if open}
		<div class="popup" role="listbox" aria-multiselectable="true" aria-label={label}>
			<label class="option">
				<input type="checkbox" checked={value.length === 0} onchange={selectAll} />
				<span>{allLabel}</span>
			</label>
			{#each options as option (option.value)}
				<label class="option">
					<input
						type="checkbox"
						checked={value.includes(option.value)}
						onchange={() => toggle(option.value)}
					/>
					<span>{option.label}</span>
				</label>
			{/each}
		</div>
	{/if}
</div>

<style>
	.multi {
		position: relative;
		min-width: 0;
	}

	.trigger {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: var(--space-2);
		width: 100%;
		min-height: var(--touch-target);
		padding: var(--space-2) var(--space-3);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-md);
		background: var(--color-surface);
		color: var(--color-text-primary);
		font-size: var(--text-body);
		cursor: pointer;
		text-align: left;
	}

	.summary {
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.popup {
		position: absolute;
		top: calc(100% + var(--space-1));
		left: 0;
		right: 0;
		z-index: 30;
		max-height: 240px;
		overflow-y: auto;
		padding: var(--space-1);
		background: var(--color-surface);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-md);
		box-shadow: var(--shadow-md);
	}

	.option {
		display: flex;
		align-items: center;
		gap: var(--space-2);
		min-height: var(--touch-target);
		padding: 0 var(--space-2);
		border-radius: var(--radius-sm);
		font-size: var(--text-caption);
		cursor: pointer;
	}

	.option:hover {
		background: var(--color-surface-elevated);
	}

	.option input {
		accent-color: var(--color-primary);
	}
</style>
