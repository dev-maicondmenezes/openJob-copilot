<script lang="ts">
	import type { Snippet } from 'svelte';
	import { pageFooter } from './footer.svelte.js';

	interface Props {
		children: Snippet;
	}

	let { children }: Props = $props();

	$effect(() => {
		pageFooter.set(children);
		return () => pageFooter.set(null);
	});
</script>

<!-- O conteúdo é renderizado pelo <footer> do layout, após o <main>. -->
