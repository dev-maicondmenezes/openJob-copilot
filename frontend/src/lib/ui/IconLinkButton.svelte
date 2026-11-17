<script lang="ts">
	import type { HTMLAnchorAttributes } from 'svelte/elements';
	import Icon from './Icon.svelte';

	interface Props extends HTMLAnchorAttributes {
		href: string;
		icon: string;
		/** Obrigatório: texto do tooltip e nome acessível. */
		label: string;
		variant?: 'neutral' | 'primary' | 'danger';
	}

	let { href, icon, label, variant = 'neutral', class: className = '', ...rest }: Props = $props();
</script>

<a {...rest} {href} class="icon-link {variant} {className}" aria-label={label} title={label}>
	<Icon name={icon} size={20} />
</a>

<style>
	.icon-link {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		width: var(--touch-target);
		height: var(--touch-target);
		flex-shrink: 0;
		border: 1px solid var(--color-border);
		border-radius: var(--radius-md);
		background: var(--color-surface);
		color: var(--color-text-secondary);
		text-decoration: none;
		transition:
			background var(--motion-micro) var(--ease-standard),
			border-color var(--motion-micro) var(--ease-standard);
	}

	.icon-link:hover {
		background: var(--color-surface-elevated);
		border-color: var(--color-primary);
		color: var(--color-primary);
	}

	.icon-link.primary {
		background: var(--color-primary);
		border-color: var(--color-primary);
		color: var(--color-on-primary);
	}

	.icon-link.danger {
		color: var(--color-error);
	}

	.icon-link.danger:hover {
		border-color: var(--color-error);
		color: var(--color-error);
	}
</style>
