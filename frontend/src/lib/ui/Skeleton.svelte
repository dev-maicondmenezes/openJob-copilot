<script lang="ts">
	interface Props {
		width?: string;
		height?: string;
		circle?: boolean;
	}

	let { width = '100%', height = '16px', circle = false }: Props = $props();
</script>

<span
	class="skeleton"
	class:circle
	style:width
	style:height
	aria-hidden="true"
></span>

<style>
	.skeleton {
		display: block;
		border-radius: var(--radius-sm);
		background: linear-gradient(
			90deg,
			var(--color-surface-elevated) 25%,
			var(--color-surface-soft) 50%,
			var(--color-surface-elevated) 75%
		);
		background-size: 200% 100%;
		animation: shimmer 1.4s infinite linear;
	}

	.circle {
		border-radius: var(--radius-pill);
	}

	@keyframes shimmer {
		to {
			background-position: -200% 0;
		}
	}
</style>
