<script lang="ts">
	interface Props {
		checked?: boolean;
		label: string;
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

<label class="switch-row" class:disabled>
	<span class="text">
		<span class="label">{label}</span>
		{#if description}<span class="description">{description}</span>{/if}
	</span>
	<input
		type="checkbox"
		role="switch"
		{id}
		bind:checked
		{disabled}
		{onchange}
		aria-label={label}
	/>
	<span class="track" aria-hidden="true"><span class="thumb"></span></span>
</label>

<style>
	.switch-row {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: var(--space-4);
		min-height: var(--touch-target);
		padding: var(--space-2) 0;
		cursor: pointer;
	}

	.switch-row.disabled {
		opacity: 0.6;
		cursor: not-allowed;
	}

	.text {
		display: flex;
		flex-direction: column;
		gap: var(--space-05);
		min-width: 0;
	}

	.label {
		font-weight: var(--weight-medium);
	}

	.description {
		font-size: var(--text-caption);
		line-height: var(--leading-caption);
		color: var(--color-text-muted);
	}

	input {
		position: absolute;
		opacity: 0;
		width: 1px;
		height: 1px;
	}

	.track {
		position: relative;
		flex-shrink: 0;
		width: 44px;
		height: 26px;
		border-radius: var(--radius-pill);
		background: var(--color-border);
		transition: background var(--motion-transition) var(--ease-standard);
	}

	.thumb {
		position: absolute;
		top: 3px;
		left: 3px;
		width: 20px;
		height: 20px;
		border-radius: var(--radius-pill);
		background: var(--color-surface);
		box-shadow: var(--shadow-sm);
		transition: transform var(--motion-transition) var(--ease-standard);
	}

	input:checked + .track {
		background: var(--color-primary);
	}

	input:checked + .track .thumb {
		transform: translateX(18px);
	}

	input:focus-visible + .track {
		outline: 2px solid var(--color-focus);
		outline-offset: 2px;
	}
</style>
