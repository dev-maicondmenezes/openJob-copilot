<script lang="ts">
	import type { HTMLTextareaAttributes } from 'svelte/elements';

	interface Props extends HTMLTextareaAttributes {
		value?: string;
		invalid?: boolean;
	}

	let {
		value = $bindable(''),
		invalid = false,
		rows = 6,
		class: className = '',
		...rest
	}: Props = $props();
</script>

<textarea {...rest} {rows} class="textarea {className}" class:invalid bind:value></textarea>

<style>
	.textarea {
		width: 100%;
		min-height: calc(var(--space-16) + var(--space-8));
		padding: var(--space-3);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-md);
		background: var(--color-surface);
		color: var(--color-text-primary);
		font-size: var(--text-body);
		line-height: var(--leading-body);
		resize: vertical;
		transition: border-color var(--motion-micro) var(--ease-standard);
	}

	.textarea::placeholder {
		color: var(--color-text-muted);
	}

	.textarea:focus {
		outline: 2px solid var(--color-focus);
		outline-offset: 1px;
		border-color: var(--color-focus);
	}

	.textarea.invalid {
		border-color: var(--color-error);
	}

	.textarea:disabled {
		background: var(--color-surface-elevated);
		color: var(--color-text-muted);
	}
</style>
