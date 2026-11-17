<script lang="ts">
	interface Option {
		value: string;
		label: string;
		description?: string;
	}

	interface Props {
		name: string;
		label: string;
		value?: string;
		options: readonly Option[];
		onchange?: (value: string) => void;
	}

	let { name, label, value = $bindable(''), options, onchange }: Props = $props();
</script>

<fieldset class="radio-group">
	<legend>{label}</legend>
	{#each options as option (option.value)}
		<label class="option" class:selected={value === option.value}>
			<input
				type="radio"
				{name}
				value={option.value}
				bind:group={value}
				onchange={() => onchange?.(option.value)}
			/>
			<span>
				<span class="label">{option.label}</span>
				{#if option.description}<span class="description">{option.description}</span>{/if}
			</span>
		</label>
	{/each}
</fieldset>

<style>
	.radio-group {
		display: flex;
		flex-direction: column;
		gap: var(--space-2);
		border: 0;
		padding: 0;
		margin: 0;
	}

	legend {
		font-size: var(--text-caption);
		font-weight: var(--weight-semibold);
		margin-bottom: var(--space-2);
	}

	.option {
		display: flex;
		align-items: flex-start;
		gap: var(--space-2);
		min-height: var(--touch-target);
		padding: var(--space-2) var(--space-3);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-md);
		cursor: pointer;
	}

	.option.selected {
		border-color: var(--color-primary);
		background: var(--color-info-soft);
	}

	input {
		margin: var(--space-1) 0 0;
		accent-color: var(--color-primary);
	}

	.label {
		font-weight: var(--weight-medium);
	}

	.description {
		display: block;
		font-size: var(--text-caption);
		line-height: var(--leading-caption);
		color: var(--color-text-muted);
	}
</style>
