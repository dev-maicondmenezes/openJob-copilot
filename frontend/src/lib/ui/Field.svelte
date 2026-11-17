<script lang="ts">
	import type { Snippet } from 'svelte';

	interface Props {
		/** Texto do rótulo; ausente em grupos já rotulados (use aria-label no controle). */
		label?: string;
		for?: string;
		hint?: string;
		error?: string;
		required?: boolean;
		children: Snippet;
		class?: string;
	}

	let {
		label,
		for: htmlFor,
		hint,
		error,
		required = false,
		children,
		class: className = ''
	}: Props = $props();
</script>

<div class="field {className}">
	{#if label}
		<label for={htmlFor} class="label">
			{label}{#if required}<span class="req" aria-hidden="true"> *</span>{/if}
		</label>
	{/if}
	{@render children()}
	{#if error}
		<p class="error-text" role="alert">{error}</p>
	{:else if hint}
		<p class="hint">{hint}</p>
	{/if}
</div>

<style>
	.field {
		display: flex;
		flex-direction: column;
		gap: var(--space-2);
		min-width: 0;
	}

	.label {
		font-size: var(--text-caption);
		font-weight: var(--weight-semibold);
		color: var(--color-text-primary);
		line-height: var(--leading-caption);
	}

	.req {
		color: var(--color-error);
	}

	.hint {
		margin: 0;
		font-size: var(--text-caption);
		line-height: var(--leading-caption);
		color: var(--color-text-muted);
	}

	.error-text {
		margin: 0;
		font-size: var(--text-caption);
		line-height: var(--leading-caption);
		color: var(--color-error);
	}
</style>
