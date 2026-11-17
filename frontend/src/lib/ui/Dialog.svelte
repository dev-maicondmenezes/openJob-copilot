<script lang="ts">
	import type { Snippet } from 'svelte';
	import Modal from './Modal.svelte';
	import Sheet from './Sheet.svelte';
	import { useViewport, viewport } from './media.svelte.js';

	/**
	 * API única de diálogo: Sheet (slide-up) em phone/tablet, Modal centralizado
	 * em desktop. Use `Drawer` quando o contexto da lista deve permanecer visível.
	 */
	interface Props {
		open: boolean;
		title: string;
		description?: string;
		onclose: () => void;
		dismissible?: boolean;
		children: Snippet;
		footer?: Snippet;
	}

	let { open, title, description, onclose, dismissible = true, children, footer }: Props = $props();

	useViewport();
	const Surface = $derived(viewport.isDesktop ? Modal : Sheet);
</script>

{#snippet footerSnippet()}
	{#if footer}{@render footer()}{/if}
{/snippet}

{#if open}
	<Surface {title} {description} {onclose} {dismissible} footer={footerSnippet}>
		{@render children()}
	</Surface>
{/if}
