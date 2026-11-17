<script lang="ts">
	import type { Snippet } from 'svelte';

	interface Props {
		title?: string;
		label?: string;
		actions?: Snippet;
		children?: Snippet;
		class?: string;
	}

	let { title, label, actions, children, class: className = '' }: Props = $props();
</script>

<section class="panel {className}" aria-label={label}>
	{#if title || actions}
		<div class="panel-head">
			{#if title}<h3 class="panel-title">{title}</h3>{/if}
			{#if actions}<div class="actions">{@render actions()}</div>{/if}
		</div>
	{/if}
	{#if children}{@render children()}{/if}
</section>

<style>
	.panel {
		background: var(--color-surface-soft);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-md);
		padding: var(--space-4);
		min-width: 0;
	}

	.panel-head {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: var(--space-3);
		margin-bottom: var(--space-3);
	}

	.actions {
		display: flex;
		gap: var(--space-2);
	}

	.panel-title {
		margin: 0;
		font-size: var(--text-caption);
		line-height: var(--leading-caption);
		font-weight: var(--weight-semibold);
		text-transform: uppercase;
		letter-spacing: 0.04em;
		color: var(--color-text-muted);
	}
</style>
