<script lang="ts">
	interface Props {
		kind?: 'success' | 'error' | 'info';
		text: string;
	}

	let { kind = 'info', text }: Props = $props();
</script>

<div class="toast {kind}" role="status">{text}</div>

<style>
	.toast {
		width: 100%;
		padding: var(--space-3) var(--space-4);
		border-radius: var(--radius-md);
		border: 1px solid var(--color-border);
		background: var(--color-surface);
		box-shadow: var(--shadow-md);
		font-size: var(--text-caption);
		line-height: var(--leading-caption);
		color: var(--color-text-primary);
	}

	.success {
		background: var(--color-success-soft);
		border-color: var(--color-success-border);
		color: var(--color-success);
	}
	.error {
		background: var(--color-error-soft);
		border-color: var(--color-error-border);
		color: var(--color-error);
	}
	.info {
		background: var(--color-info-soft);
		border-color: var(--color-info-border);
		color: var(--color-info);
	}
</style>
