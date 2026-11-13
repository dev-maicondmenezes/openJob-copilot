<script lang="ts">
	import type { Snippet } from 'svelte';

	interface Props {
		title: string;
		subtitle: string;
		emoji?: string;
		children?: Snippet;
	}

	let { title, subtitle, emoji = '🚧', children }: Props = $props();
</script>

<h1 class="page-title">{title}</h1>
<p class="page-subtitle">{subtitle}</p>

<section class="stub" aria-label="Stub desta tela">
	<span class="badge">stub</span>
	<p class="emoji">{emoji}</p>
	<p class="hint">Conteúdo chega em um checkpoint posterior — a API ainda não é consumida.</p>
	{#if children}
		{@render children()}
	{/if}
</section>

<style>
	.stub {
		background: var(--surface);
		border: 1px dashed var(--border);
		border-radius: var(--radius);
		padding: 2rem 1rem;
		text-align: center;
	}

	.badge {
		display: inline-block;
		background: var(--accent-soft);
		color: var(--accent);
		font-size: 0.75rem;
		font-weight: 600;
		letter-spacing: 0.05em;
		text-transform: uppercase;
		border-radius: 999px;
		padding: 0.125rem 0.625rem;
	}

	.emoji {
		font-size: 2.25rem;
		margin: 0.75rem 0 0.25rem;
	}

	.hint {
		color: var(--text-muted);
		font-size: 0.875rem;
		margin: 0;
	}
</style>
