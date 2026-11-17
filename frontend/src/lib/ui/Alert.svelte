<script lang="ts">
	import type { Snippet } from 'svelte';
	import Icon from './Icon.svelte';

	interface Props {
		severity?: 'info' | 'success' | 'warning' | 'error';
		title?: string;
		children?: Snippet;
		role?: 'status' | 'alert';
	}

	let {
		severity = 'info',
		title,
		children,
		role = severity === 'error' ? 'alert' : 'status'
	}: Props = $props();

	const icon = {
		info: 'info',
		success: 'circle-check',
		warning: 'warning',
		error: 'circle-alert'
	} as const;
</script>

<div class="alert {severity}" {role}>
	<Icon name={icon[severity]} size={20} />
	<div class="body">
		{#if title}<strong>{title}</strong>{/if}
		{#if children}<div>{@render children()}</div>{/if}
	</div>
</div>

<style>
	.alert {
		display: flex;
		gap: var(--space-3);
		align-items: flex-start;
		padding: var(--space-3) var(--space-4);
		border: 1px solid var(--color-border);
		border-left-width: 3px;
		border-radius: var(--radius-md);
		font-size: var(--text-body);
	}

	.body {
		display: flex;
		flex-direction: column;
		gap: var(--space-1);
		min-width: 0;
	}

	.info {
		background: var(--color-info-soft);
		border-color: var(--color-info-border);
		color: var(--color-info);
	}
	.success {
		background: var(--color-success-soft);
		border-color: var(--color-success-border);
		color: var(--color-success);
	}
	.warning {
		background: var(--color-warning-soft);
		border-color: var(--color-warning-border);
		color: var(--color-warning);
	}
	.error {
		background: var(--color-error-soft);
		border-color: var(--color-error-border);
		color: var(--color-error);
	}
</style>
