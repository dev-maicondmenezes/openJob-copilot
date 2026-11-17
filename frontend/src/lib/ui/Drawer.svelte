<script lang="ts">
	import { t } from '#lib/i18n/t.js';
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

	const titleId = `drawer-${Math.random().toString(36).slice(2)}`;
</script>

<div
	class="backdrop"
	role="presentation"
	onclick={(e) => dismissible && e.target === e.currentTarget && onclose()}
>
	<div
		class="drawer"
		use:trapFocus={onclose}
		role="dialog"
		aria-modal="true"
		aria-labelledby={titleId}
		tabindex="-1"
	>
		<header class="drawer-head">
			<div>
				<h2 id={titleId}>{title}</h2>
				{#if description}<p>{description}</p>{/if}
			</div>
			{#if dismissible}
				<IconButton icon="x" label={t('common.close')} size="sm" onclick={onclose} />
			{/if}
		</header>
		<div class="drawer-body">{@render children()}</div>
		{#if footer}<footer class="drawer-footer">{@render footer()}</footer>{/if}
	</div>
</div>

<style>
	.backdrop {
		position: fixed;
		inset: 0;
		z-index: 100;
		display: flex;
		justify-content: flex-end;
		background: var(--overlay-backdrop);
	}

	.drawer {
		display: flex;
		flex-direction: column;
		width: min(560px, 100%);
		height: 100dvh;
		overflow-y: auto;
		padding: var(--space-6);
		background: var(--color-surface);
		border-left: 1px solid var(--color-border);
		box-shadow: var(--shadow-lg);
		animation: slide-in var(--motion-route) var(--ease-out);
	}

	.drawer-head {
		display: flex;
		align-items: flex-start;
		justify-content: space-between;
		gap: var(--space-3);
		margin-bottom: var(--space-6);
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

	.drawer-body {
		display: flex;
		flex-direction: column;
		gap: var(--space-4);
		flex: 1;
	}

	.drawer-footer {
		display: flex;
		gap: var(--space-2);
		justify-content: flex-end;
		margin-top: var(--space-6);
	}

	@keyframes slide-in {
		from {
			transform: translateX(100%);
		}
		to {
			transform: translateX(0);
		}
	}

	/* Em telas menores o drawer se comporta como sheet (bottom-up). */
	@media (max-width: 1023px) {
		.backdrop {
			align-items: flex-end;
		}

		.drawer {
			width: 100%;
			height: auto;
			max-height: 90dvh;
			border-left: 0;
			border-top: 1px solid var(--color-border);
			border-radius: var(--radius-lg) var(--radius-lg) 0 0;
			padding-bottom: calc(var(--space-6) + env(safe-area-inset-bottom));
			animation-name: slide-up;
		}

		@keyframes slide-up {
			from {
				transform: translateY(100%);
			}
			to {
				transform: translateY(0);
			}
		}
	}
</style>
