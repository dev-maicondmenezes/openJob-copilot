<script lang="ts">
	interface Props {
		value: number;
		max?: number;
		label?: string;
		showValue?: boolean;
	}

	let { value, max = 100, label, showValue = false }: Props = $props();

	const pct = $derived(max > 0 ? Math.min(100, Math.max(0, (value / max) * 100)) : 0);
</script>

<div class="progress-wrap">
	{#if label || showValue}
		<div class="meta">
			{#if label}<span>{label}</span>{/if}
			{#if showValue}<span class="value">{Math.round(pct)}%</span>{/if}
		</div>
	{/if}
	<div
		class="progress"
		role="progressbar"
		aria-valuemin="0"
		aria-valuemax={max}
		aria-valuenow={value}
		aria-label={label ?? 'Progresso'}
	>
		<div class="bar" style:width="{pct}%"></div>
	</div>
</div>

<style>
	.progress-wrap {
		display: flex;
		flex-direction: column;
		gap: var(--space-1);
	}

	.meta {
		display: flex;
		justify-content: space-between;
		gap: var(--space-2);
		font-size: var(--text-caption);
		line-height: var(--leading-caption);
		color: var(--color-text-muted);
	}

	.value {
		font-weight: var(--weight-semibold);
	}

	.progress {
		width: 100%;
		height: 8px;
		border-radius: var(--radius-pill);
		background: var(--color-surface-elevated);
		overflow: hidden;
	}

	.bar {
		height: 100%;
		border-radius: var(--radius-pill);
		background: var(--color-primary);
		transition: width var(--motion-transition) var(--ease-standard);
	}
</style>
