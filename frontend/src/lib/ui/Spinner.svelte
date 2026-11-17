<script lang="ts">
	interface Props {
		size?: number;
		label?: string;
		class?: string;
	}

	let { size = 16, label, class: className = '' }: Props = $props();
</script>

<span
	class="spinner {className}"
	style:width="{size}px"
	style:height="{size}px"
	role={label ? 'status' : undefined}
	aria-label={label}
	aria-hidden={label ? undefined : 'true'}
></span>

<style>
	.spinner {
		display: inline-block;
		border: 2px solid currentColor;
		border-right-color: transparent;
		border-radius: var(--radius-pill);
		animation: spin 0.8s linear infinite;
		vertical-align: middle;
	}

	@keyframes spin {
		to {
			transform: rotate(360deg);
		}
	}
</style>
