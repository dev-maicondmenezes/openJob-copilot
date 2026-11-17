<script lang="ts">
	import type { Snippet } from 'svelte';

	interface Props {
		title?: string;
		subtitle?: string;
		label?: string;
		href?: string;
		actions?: Snippet;
		children?: Snippet;
		class?: string;
		/** Remove o padding interno (útil para tabelas). */
		flush?: boolean;
		/** Cartão mais baixo, para listas densas (ex.: kanban de candidaturas). */
		compact?: boolean;
		/** Torna o card clicável. Cliques em controles internos são ignorados. */
		onclick?: (event: MouseEvent) => void;
		/** Suporte a drag and drop (ex.: kanban de candidaturas). */
		draggable?: boolean;
		ondragstart?: (event: DragEvent) => void;
		ondragend?: (event: DragEvent) => void;
	}

	let {
		title,
		subtitle,
		label,
		href,
		actions,
		children,
		class: className = '',
		flush = false,
		compact = false,
		onclick,
		draggable,
		ondragstart,
		ondragend
	}: Props = $props();

	const hasHeader = $derived(Boolean(title || actions));

	const classes = $derived(
		`card ${className} ${flush ? 'flush' : ''} ${compact ? 'compact' : ''} ${href || onclick ? 'interactive' : ''}`
	);
	const attrs = $derived({ class: classes, 'aria-label': label });

	function handleClick(event: MouseEvent): void {
		// Não dispara a ação do card ao clicar em botões/links/controles internos.
		const target = event.target as HTMLElement | null;
		if (target?.closest('button, a, input, select, textarea, [role="menuitem"], [role="radio"], [role="switch"]')) return;
		onclick?.(event);
	}

</script>

{#snippet content()}
	{#if hasHeader}
		<header class="card-head">
			<div class="heading">
				{#if title}<h2>{title}</h2>{/if}
				{#if subtitle}<p class="subtitle">{subtitle}</p>{/if}
			</div>
			{#if actions}<div class="actions">{@render actions()}</div>{/if}
		</header>
	{/if}
	{#if children}<div class="card-body">{@render children()}</div>{/if}
{/snippet}

{#if href}
	<a {...attrs} {href}>{@render content()}</a>
{:else if onclick}
	<!--
		O card é clicável com mouse/toque; o acesso por teclado é feito por um
		botão explícito no conteúdo (evita `nested-interactive` quando o card
		contém outros controles, como no kanban de candidaturas).
	-->
	<!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
	<div
		{...attrs}
		{draggable}
		{ondragstart}
		{ondragend}
		onclick={handleClick}
	>
		{@render content()}
	</div>
{:else}
	<section {...attrs} {draggable} {ondragstart} {ondragend}>{@render content()}</section>
{/if}

<style>
	.card {
		display: block;
		background: var(--color-surface);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-lg);
		padding: var(--space-4);
		box-shadow: var(--shadow-sm);
		color: var(--color-text-primary);
	}

	.card.flush {
		padding: 0;
		overflow: hidden;
	}

	.card.compact {
		padding: var(--space-3);
	}

	.card.compact .card-head {
		margin-bottom: var(--space-2);
	}

	.card.interactive {
		text-decoration: none;
		transition:
			border-color var(--motion-micro) var(--ease-standard),
			transform var(--motion-micro) var(--ease-out);
	}

	.card.interactive:hover {
		border-color: var(--color-primary);
	}

	.card.interactive:active {
		transform: translateY(1px);
	}

	.card-head {
		display: flex;
		align-items: flex-start;
		justify-content: space-between;
		gap: var(--space-3);
		margin-bottom: var(--space-4);
	}

	.heading {
		min-width: 0;
	}

	h2 {
		margin: 0;
		font-size: var(--text-h3);
		line-height: var(--leading-h3);
	}

	.subtitle {
		margin: var(--space-1) 0 0;
		font-size: var(--text-caption);
		line-height: var(--leading-caption);
		color: var(--color-text-muted);
	}

	.actions {
		display: flex;
		align-items: center;
		gap: var(--space-2);
		flex-shrink: 0;
	}

	@media (min-width: 1024px) {
		.card {
			box-shadow: none;
		}
	}
</style>
