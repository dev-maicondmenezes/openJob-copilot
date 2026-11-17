<script lang="ts">
	import type { Snippet } from 'svelte';

	interface Props {
		title: string;
		description?: string;
		actions?: Snippet;
		children?: Snippet;
		label?: string;
	}

	let { title, description, actions, children, label }: Props = $props();
</script>

<section class="section" aria-label={label}>
	<div class="section-head">
		<div>
			<h2>{title}</h2>
			{#if description}<p>{description}</p>{/if}
		</div>
		{#if actions}<div class="actions">{@render actions()}</div>{/if}
	</div>
	{#if children}<div class="section-body">{@render children()}</div>{/if}
</section>

<style>
	.section {
		margin-bottom: var(--space-8);
	}

	.section-head {
		display: flex;
		align-items: flex-start;
		justify-content: space-between;
		gap: var(--space-3);
		margin-bottom: var(--space-3);
	}

	h2 {
		margin: 0;
		font-size: var(--text-h2);
	}

	p {
		margin: var(--space-1) 0 0;
		font-size: var(--text-caption);
		line-height: var(--leading-caption);
		color: var(--color-text-muted);
	}

	.actions {
		display: flex;
		gap: var(--space-2);
		flex-shrink: 0;
	}
</style>
