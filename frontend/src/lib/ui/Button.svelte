<script lang="ts">
	import type { Snippet } from 'svelte';
	import type { HTMLButtonAttributes } from 'svelte/elements';
	import Spinner from './Spinner.svelte';

	interface BaseProps extends HTMLButtonAttributes {
		variant?: 'primary' | 'secondary' | 'ghost' | 'danger' | 'link';
		size?: 'sm' | 'md' | 'lg';
		loading?: boolean;
		/** Densidade extra para barras de ação com vários botões na mesma linha. */
		compact?: boolean;
		full?: boolean;
		children?: Snippet;
	}

	/** Botão só-ícone exige aria-label por contrato de tipo. */
	type Props =
		| (BaseProps & { iconOnly: true; 'aria-label': string })
		| (BaseProps & { iconOnly?: false });

	let {
		variant = 'primary',
		size = 'md',
		loading = false,
		iconOnly = false,
		compact = false,
		full = false,
		disabled = false,
		type = 'button',
		class: className = '',
		children,
		...rest
	}: Props = $props();

	const spinnerSize = $derived(size === 'lg' ? 20 : 16);
</script>

<button
	{...rest}
	{type}
	class="btn {variant} {size} {className}"
	class:icon-only={iconOnly}
	class:compact
	class:full
	disabled={disabled || loading}
	aria-busy={loading || undefined}
>
	{#if loading}
		<Spinner size={spinnerSize} />
	{/if}
	{#if children}
		<span class="label">{@render children()}</span>
	{/if}
</button>

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
		cursor: pointer;
		text-decoration: none;
		white-space: nowrap;
		-webkit-tap-highlight-color: transparent;
		transition:
			background var(--motion-micro) var(--ease-standard),
			border-color var(--motion-micro) var(--ease-standard),
			transform var(--motion-micro) var(--ease-out);
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

	.btn.icon-only {
		width: var(--touch-target);
		padding: 0;
	}

	.btn.compact {
		min-height: 36px;
		padding: 0 var(--space-2);
		font-size: var(--text-caption);
	}

	.btn:active:not(:disabled) {
		transform: translateY(1px);
	}

	.btn:disabled {
		opacity: 0.55;
		cursor: not-allowed;
	}

	.primary {
		background: var(--color-primary);
		color: var(--color-on-primary);
	}
	.primary:hover:not(:disabled) {
		background: var(--color-primary-hover);
	}

	.secondary {
		background: var(--color-info-soft);
		color: var(--color-primary);
	}
	.secondary:hover:not(:disabled) {
		background: var(--color-info-border);
	}

	.ghost {
		background: var(--color-surface);
		border-color: var(--color-border);
		color: var(--color-text-primary);
	}
	.ghost:hover:not(:disabled) {
		background: var(--color-surface-elevated);
	}

	.danger {
		background: var(--color-error-soft);
		color: var(--color-error);
	}
	.danger:hover:not(:disabled) {
		background: var(--color-error-border);
	}

	.link {
		min-height: 0;
		padding: 0;
		border: 0;
		background: none;
		color: var(--color-primary);
		font-weight: var(--weight-medium);
		text-decoration: underline;
	}

	.label {
		display: inline-flex;
		align-items: center;
		gap: var(--space-2);
	}

	@media (min-width: 768px) {
		.btn:not(.sm) {
			min-height: var(--touch-target);
		}
	}
</style>
