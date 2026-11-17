<script lang="ts">
	import { toasts } from '#lib/toast.svelte.ts';
	import Toast from '#lib/ui/Toast.svelte';
</script>

<div class="toaster" aria-live="polite" aria-atomic="false">
	{#each $toasts as t (t.id)}
		<Toast kind={t.kind} text={t.text} />
	{/each}
</div>

<style>
	.toaster {
		position: fixed;
		top: var(--space-3);
		left: 50%;
		transform: translateX(-50%);
		display: flex;
		flex-direction: column;
		gap: var(--space-2);
		z-index: 120;
		width: min(26rem, calc(100vw - var(--space-8)));
		pointer-events: none;
	}
</style>
