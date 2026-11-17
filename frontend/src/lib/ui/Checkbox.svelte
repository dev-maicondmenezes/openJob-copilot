<script lang="ts">
	interface Props {
		checked?: boolean;
		label?: string;
		description?: string;
		id?: string;
		disabled?: boolean;
		onchange?: (event: Event) => void;
	}

	let {
		checked = $bindable(false),
		label,
		description,
		id,
		disabled = false,
		onchange
	}: Props = $props();
</script>

<label class="checkbox" class:disabled>
	<input type="checkbox" {id} bind:checked {disabled} {onchange} />
	{#if label || description}
		<span class="text">
			{#if label}<span class="label">{label}</span>{/if}
			{#if description}<span class="description">{description}</span>{/if}
		</span>
	{/if}
</label>

<style>
	.checkbox {
		display: inline-flex;
		align-items: flex-start;
		gap: var(--space-2);
		min-height: var(--touch-target);
		cursor: pointer;
		font-size: var(--text-body);
	}

	.checkbox.disabled {
		opacity: 0.6;
		cursor: not-allowed;
	}

	input {
		width: 20px;
		height: 20px;
		margin: calc(var(--space-4) - 10px) 0 0;
		accent-color: var(--color-primary);
		flex-shrink: 0;
		cursor: inherit;
	}

	.text {
		display: flex;
		flex-direction: column;
		gap: var(--space-05);
	}

	.label {
		font-weight: var(--weight-medium);
	}

	.description {
		font-size: var(--text-caption);
		line-height: var(--leading-caption);
		color: var(--color-text-muted);
	}
</style>
