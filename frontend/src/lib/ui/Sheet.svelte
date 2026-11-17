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

	const titleId = `sheet-${Math.random().toString(36).slice(2)}`;
</script>

<div
	class="backdrop"
	role="presentation"
	onclick={(e) => dismissible && e.target === e.currentTarget && onclose()}
>
	<div
		class="sheet"
		use:trapFocus={onclose}
		role="dialog"
		aria-modal="true"
		aria-labelledby={titleId}
		tabindex="-1"
	>
		<div class="grabber" aria-hidden="true"></div>
		<header class="sheet-head">
			<div>
				<h2 id={titleId}>{title}</h2>
				{#if description}<p>{description}</p>{/if}
			</div>
			{#if dismissible}
				<IconButton icon="x" label="Fechar" size="sm" onclick={onclose} />
			{/if}
		</header>
		<div class="sheet-body">{@render children()}</div>
		{#if footer}<footer class="sheet-footer">{@render footer()}</footer>{/if}
	</div>
</div>

<style>
	.backdrop {
		position: fixed;
		inset: 0;
		z-index: 100;
		display: flex;
		align-items: flex-end;
		justify-content: center;
		background: var(--overlay-backdrop);
	}

	.sheet {
		width: min(40rem, 100%);
		max-height: 90dvh;
		overflow-y: auto;
		padding: var(--space-2) var(--space-4)
			calc(var(--space-6) + env(safe-area-inset-bottom));
		background: var(--color-surface);
		border: 1px solid var(--color-border);
		border-bottom: 0;
		border-radius: var(--radius-lg) var(--radius-lg) 0 0;
		box-shadow: var(--shadow-lg);
		animation: slide-up var(--motion-route) var(--ease-out);
	}

	.grabber {
		width: 36px;
		height: 4px;
		margin: 0 auto var(--space-3);
		border-radius: var(--radius-pill);
		background: var(--color-border);
	}

	.sheet-head {
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

	.sheet-body {
		display: flex;
		flex-direction: column;
		gap: var(--space-4);
	}

	.sheet-footer {
		display: flex;
		gap: var(--space-2);
		margin-top: var(--space-6);
	}

	.sheet-footer :global(.btn) {
		flex: 1;
	}

	@keyframes slide-up {
		from {
			transform: translateY(100%);
		}
		to {
			transform: translateY(0);
		}
	}
</style>
