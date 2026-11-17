<script lang="ts">
	import type { Snippet } from 'svelte';
	import IconButton from './IconButton.svelte';
	import { trapFocus } from './focus.js';

	interface Props {
		title: string;
		description?: string;
		onclose: () => void;
		dismissible?: boolean;
		children: Snippet;
		footer?: Snippet;
	}

	let {
		title,
		description,
		onclose,
		dismissible = true,
		children,
		footer
	}: Props = $props();

	const titleId = `modal-${Math.random().toString(36).slice(2)}`;
</script>

<div
	class="backdrop"
	role="presentation"
	onclick={(e) => dismissible && e.target === e.currentTarget && onclose()}
>
	<div
		class="modal"
		use:trapFocus={onclose}
		role="dialog"
		aria-modal="true"
		aria-labelledby={titleId}
		tabindex="-1"
	>
		<header class="modal-head">
			<div>
				<h2 id={titleId}>{title}</h2>
				{#if description}<p>{description}</p>{/if}
			</div>
			{#if dismissible}
				<IconButton icon="x" label="Fechar" size="sm" onclick={onclose} />
			{/if}
		</header>
		<div class="modal-body">{@render children()}</div>
		{#if footer}<footer class="modal-footer">{@render footer()}</footer>{/if}
	</div>
</div>

<style>
	.backdrop {
		position: fixed;
		inset: 0;
		z-index: 100;
		display: flex;
		align-items: center;
		justify-content: center;
		padding: var(--space-4);
		background: var(--overlay-backdrop);
	}

	.modal {
		width: min(480px, 100%);
		max-height: 90dvh;
		overflow-y: auto;
		background: var(--color-surface);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-lg);
		box-shadow: var(--shadow-lg);
		padding: var(--space-6);
	}

	.modal-head {
		display: flex;
		align-items: flex-start;
		justify-content: space-between;
		gap: var(--space-3);
		margin-bottom: var(--space-4);
	}

	h2 {
		margin: 0;
		font-size: var(--text-h2);
	}

	p {
		margin: var(--space-1) 0 0;
		font-size: var(--text-caption);
		color: var(--color-text-muted);
	}

	.modal-body {
		display: flex;
		flex-direction: column;
		gap: var(--space-4);
	}

	.modal-footer {
		display: flex;
		justify-content: flex-end;
		gap: var(--space-2);
		margin-top: var(--space-6);
	}
</style>
