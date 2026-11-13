<script lang="ts">
	import '../app.css';
	import BottomNav from '#lib/components/BottomNav.svelte';
	import { registerSW } from 'virtual:pwa-register';
	import type { LayoutProps } from './$types';

	// Registra o service worker (autoUpdate): se um novo deploy assumir,
	// o SW ativa e as páginas abertas são recarregadas — evita assets stale.
	registerSW({ immediate: true });

	let { children }: LayoutProps = $props();
</script>

<svelte:head>
	<link rel="icon" href="%sveltekit.assets%/favicon.svg" />
</svelte:head>

<div class="app">
	<main class="content">
		{@render children()}
	</main>
	<BottomNav />
</div>
