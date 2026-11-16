<script lang="ts">
	import { toasts } from '#lib/toast.svelte.ts';
</script>

<div class="toaster" aria-live="polite">
	{#each $toasts as t (t.id)}
		<div class="toast {t.kind}" role="status">{t.text}</div>
	{/each}
</div>

<style>
	.toaster {
		position: fixed;
		top: 0.75rem;
		left: 50%;
		transform: translateX(-50%);
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
		z-index: 100;
		width: min(26rem, calc(100vw - 2rem));
		pointer-events: none;
	}

	.toast {
		border-radius: var(--radius);
		padding: 0.6rem 0.9rem;
		font-size: 0.85rem;
		background: var(--surface);
		border: 1px solid var(--border);
		box-shadow: 0 6px 18px rgba(0, 0, 0, 0.12);
	}

	.toast.success {
		background: #ecfdf5;
		border-color: #a7f3d0;
		color: #047857;
	}

	.toast.error {
		background: #fef2f2;
		border-color: #fecaca;
		color: #b91c1c;
	}

	.toast.info {
		background: var(--accent-soft);
		border-color: var(--border);
		color: var(--accent);
	}
</style>
