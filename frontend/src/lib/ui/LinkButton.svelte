<script lang="ts">
	import type { Snippet } from 'svelte';
	import type { HTMLAnchorAttributes } from 'svelte/elements';

	interface Props extends HTMLAnchorAttributes {
		href: string;
		variant?: 'primary' | 'secondary' | 'ghost' | 'danger';
		size?: 'sm' | 'md' | 'lg';
		full?: boolean;
		children?: Snippet;
	}

	let {
		href,
		variant = 'primary',
		size = 'md',
		full = false,
		class: className = '',
		children,
		...rest
	}: Props = $props();
</script>

<a {...rest} {href} class="btn {variant} {size} {className}" class:full>
	{#if children}{@render children()}{/if}
</a>

<style>
	.btn {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		gap: var(--space-2);
		min-height: var(--touch-target);
		padding: 0 var(--space-4);
		border: 1px solid transparent;
		border-radius: var(--radius-md);
		font-weight: var(--weight-semibold);
		font-size: var(--text-body);
		line-height: 1;
		text-decoration: none;
		white-space: nowrap;
		transition: background var(--motion-micro) var(--ease-standard);
	}

	.btn.sm {
		min-height: 32px;
		padding: 0 var(--space-3);
		font-size: var(--text-caption);
	}
	.btn.lg {
		min-height: 52px;
		padding: 0 var(--space-6);
	}
	.btn.full {
		width: 100%;
	}

	.primary {
		background: var(--color-primary);
		color: var(--color-on-primary);
	}
	.primary:hover {
		background: var(--color-primary-hover);
	}
	.secondary {
		background: var(--color-info-soft);
		color: var(--color-primary);
	}
	.ghost {
		background: var(--color-surface);
		border-color: var(--color-border);
		color: var(--color-text-primary);
	}
	.danger {
		background: var(--color-error-soft);
		color: var(--color-error);
	}
</style>
