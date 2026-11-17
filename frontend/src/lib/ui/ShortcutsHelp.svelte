<script lang="ts">
	import Dialog from './Dialog.svelte';
	import { SHORTCUTS } from '../shortcuts.js';

	interface Props {
		open: boolean;
		onclose: () => void;
	}

	let { open, onclose }: Props = $props();
</script>

<Dialog {open} {onclose} title="Atalhos de teclado" description="Disponíveis em desktop, fora de campos de texto.">
	<dl class="shortcuts">
		{#each SHORTCUTS as shortcut (shortcut.id)}
			<div class="row">
				<dt>{shortcut.description}</dt>
				<dd><kbd>{shortcut.keys}</kbd></dd>
			</div>
		{/each}
	</dl>
</Dialog>

<style>
	.shortcuts {
		display: flex;
		flex-direction: column;
		gap: var(--space-1);
		margin: 0;
	}

	.row {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: var(--space-4);
		padding: var(--space-2) 0;
		border-bottom: 1px solid var(--color-border);
	}

	.row:last-child {
		border-bottom: 0;
	}

	dt {
		font-size: var(--text-body);
	}

	dd {
		margin: 0;
	}

	kbd {
		display: inline-block;
		min-width: 32px;
		padding: var(--space-1) var(--space-2);
		border: 1px solid var(--color-border);
		border-bottom-width: 2px;
		border-radius: var(--radius-sm);
		background: var(--color-surface-soft);
		font-family: var(--font-mono);
		font-size: var(--text-caption);
		text-align: center;
	}
</style>
