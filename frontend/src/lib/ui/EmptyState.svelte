<script lang="ts">
	import type { Snippet } from 'svelte';
	import Icon from './Icon.svelte';

	interface Props {
		title: string;
		description?: string;
		icon?: string;
		children?: Snippet;
	}

	let { title, description, icon = 'inbox', children }: Props = $props();
</script>

<div class="empty-state">
	<span class="icon"><Icon name={icon} size={24} /></span>
	<h3>{title}</h3>
	{#if description}<p>{description}</p>{/if}
	{#if children}<div class="actions">{@render children()}</div>{/if}
</div>

<style>
	.empty-state {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: var(--space-2);
		padding: var(--space-12) var(--space-4);
		text-align: center;
		color: var(--color-text-muted);
	}

	.icon {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		width: 48px;
		height: 48px;
		border-radius: var(--radius-pill);
		background: var(--color-surface-elevated);
		color: var(--color-text-muted);
	}

	h3 {
		margin: 0;
		font-size: var(--text-h3);
		color: var(--color-text-primary);
	}

	p {
		margin: 0;
		max-width: 40ch;
		font-size: var(--text-caption);
		line-height: var(--leading-caption);
	}

	.actions {
		display: flex;
		gap: var(--space-2);
		flex-wrap: wrap;
		justify-content: center;
		margin-top: var(--space-2);
	}
</style>
