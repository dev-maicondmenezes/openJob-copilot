<script lang="ts">
	import type { Snippet } from 'svelte';

	interface Props {
		tone?: 'neutral' | 'primary' | 'success' | 'warning' | 'error' | 'info';
		title?: string;
		children: Snippet;
	}

	let { tone = 'neutral', title, children }: Props = $props();
</script>

<span class="badge {tone}" {title}>{@render children()}</span>

<style>
	.badge {
		display: inline-flex;
		align-items: center;
		gap: var(--space-1);
		padding: 2px var(--space-2);
		border-radius: var(--radius-pill);
		background: var(--color-surface-elevated);
		color: var(--color-text-secondary);
		font-size: var(--text-caption);
		line-height: var(--leading-caption);
		font-weight: var(--weight-semibold);
		white-space: nowrap;
	}

	.primary {
		background: var(--color-info-soft);
		color: var(--color-primary);
	}
	.success {
		background: var(--color-success-soft);
		color: var(--color-success);
	}
	.warning {
		background: var(--color-warning-soft);
		color: var(--color-warning);
	}
	.error {
		background: var(--color-error-soft);
		color: var(--color-error);
	}
	.info {
		background: var(--color-info-soft);
		color: var(--color-info);
	}
</style>
