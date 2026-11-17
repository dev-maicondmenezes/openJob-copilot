<script lang="ts">
	import type { Snippet } from 'svelte';
	import IconButton from './IconButton.svelte';
	import { useViewport, viewport } from './media.svelte.js';

	type ControlSize = 'sm' | 'md';

	interface Props {
		page: number;
		totalPages: number;
		onchange: (page: number) => void;
		label?: string;
		/** Ações contextuais no início da barra (ex.: Score da vaga selecionada). */
		leading?: Snippet<[ControlSize]>;
		/** Ações contextuais à direita, antes das setas (ex.: Aprovar/Descartar). */
		actions?: Snippet<[ControlSize]>;
		/** Contador exibido depois das setas (ex.: "1..10 de 37 vagas"). */
		count?: Snippet;
	}

	let { page, totalPages, onchange, label = 'Paginação', leading, actions, count }: Props = $props();

	useViewport();
	// Em phone/tablet os controles usam alvo de toque de 48px.
	const controlSize = $derived<ControlSize>(viewport.isDesktop ? 'sm' : 'md');
</script>

<nav class="pagination" aria-label={label}>
	{#if leading}
		<div class="slot leading">{@render leading(controlSize)}</div>
	{/if}
	{#if actions}
		<div class="slot actions">{@render actions(controlSize)}</div>
	{/if}
	<div class="arrows">
		<IconButton
			icon="arrow-left"
			label="Página anterior"
			variant="ghost"
			size={controlSize}
			disabled={page <= 1}
			onclick={() => onchange(page - 1)}
		/>
		<span class="indicator" aria-live="polite"><strong>{page}</strong> / {totalPages}</span>
		<IconButton
			icon="chevron-right"
			label="Próxima página"
			variant="ghost"
			size={controlSize}
			disabled={page >= totalPages}
			onclick={() => onchange(page + 1)}
		/>
	</div>
	{#if count}
		<div class="slot count">{@render count()}</div>
	{/if}
</nav>

<style>
	/* Rodapé da página: mesma largura do cartão, após a lista, sem sobreposição. */
	.pagination {
		display: flex;
		align-items: center;
		gap: var(--space-4);
		width: 100%;
		margin: 0;
		padding: var(--space-2) var(--space-3);
		background: var(--color-surface);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-md);
		box-shadow: var(--shadow-md);
	}

	.slot {
		display: flex;
		align-items: center;
		gap: var(--space-2);
		flex-shrink: 0;
	}

	.leading {
		margin-right: auto;
	}

	.actions {
		margin-left: auto;
	}

	.arrows {
		display: flex;
		align-items: center;
		gap: var(--space-1);
		flex-shrink: 0;
	}

	.indicator {
		min-width: 5ch;
		text-align: center;
		font-size: var(--text-caption);
		line-height: var(--leading-caption);
		color: var(--color-text-muted);
	}

	.indicator strong {
		color: var(--color-text-primary);
	}

	.count {
		font-size: var(--text-caption);
		line-height: var(--leading-caption);
		color: var(--color-text-muted);
		white-space: nowrap;
	}

	@media (max-width: 767px) {
		.pagination {
			gap: var(--space-05);
			padding: var(--space-1) var(--space-1);
		}

		.leading,
		.actions {
			margin: 0;
		}

		.indicator {
			min-width: 3ch;
		}

		.pagination {
			flex-wrap: wrap;
			justify-content: center;
		}

		.count {
			width: 100%;
			margin-top: var(--space-1);
			justify-content: center;
			text-align: center;
			font-size: var(--text-caption);
		}
	}
</style>
