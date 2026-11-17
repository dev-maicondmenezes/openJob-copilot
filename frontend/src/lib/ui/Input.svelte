<script lang="ts">
	import type { HTMLInputAttributes } from 'svelte/elements';

	interface Props extends HTMLInputAttributes {
		value?: string | number;
		invalid?: boolean;
	}

	let {
		value = $bindable(''),
		invalid = false,
		type = 'text',
		class: className = '',
		...rest
	}: Props = $props();
</script>

<input {...rest} {type} class="input {className}" class:invalid bind:value />

<style>
	.input {
		width: 100%;
		min-height: var(--touch-target);
		padding: var(--space-2) var(--space-3);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-md);
		background: var(--color-surface);
		color: var(--color-text-primary);
		font-size: var(--text-body);
		transition: border-color var(--motion-micro) var(--ease-standard);
	}

	.input::placeholder {
		color: var(--color-text-muted);
	}

	.input:hover:not(:disabled) {
		border-color: var(--color-text-muted);
	}

	.input:focus {
		outline: 2px solid var(--color-focus);
		outline-offset: 1px;
		border-color: var(--color-focus);
	}

	.input.invalid {
		border-color: var(--color-error);
	}

	.input:disabled {
		background: var(--color-surface-elevated);
		color: var(--color-text-muted);
		cursor: not-allowed;
	}
</style>
