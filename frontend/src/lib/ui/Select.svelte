<script lang="ts">
	import type { Snippet } from 'svelte';
	import type { HTMLSelectAttributes } from 'svelte/elements';

	interface Option {
		value: string;
		label: string;
		disabled?: boolean;
	}

	interface Props extends HTMLSelectAttributes {
		value?: string;
		options?: readonly Option[];
		placeholder?: string;
		children?: Snippet;
	}

	let {
		value = $bindable(''),
		options = [],
		placeholder,
		class: className = '',
		children,
		...rest
	}: Props = $props();
</script>

<select {...rest} class="select {className}" bind:value>
	{#if placeholder}<option value="" disabled>{placeholder}</option>{/if}
	{#each options as option (option.value)}
		<option value={option.value} disabled={option.disabled}>{option.label}</option>
	{/each}
	{@render children?.()}
</select>

<style>
	.select {
		width: 100%;
		min-height: var(--touch-target);
		padding: var(--space-2) var(--space-3);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-md);
		background: var(--color-surface);
		color: var(--color-text-primary);
		font-size: var(--text-body);
		cursor: pointer;
	}

	.select:focus {
		outline: 2px solid var(--color-focus);
		outline-offset: 1px;
		border-color: var(--color-focus);
	}

	.select:disabled {
		background: var(--color-surface-elevated);
		color: var(--color-text-muted);
	}
</style>
