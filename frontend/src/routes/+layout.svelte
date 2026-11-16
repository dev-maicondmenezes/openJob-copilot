<script lang="ts">
	import '../app.css';
	import BottomNav from '#lib/components/BottomNav.svelte';
	import NotificationBell from '#lib/components/NotificationBell.svelte';
	import Toaster from '#lib/components/Toaster.svelte';
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
	<header class="topbar">
		<a class="brand" href="/">openJob</a>
		<NotificationBell />
	</header>
	<main class="content">
		{@render children()}
	</main>
	<BottomNav />
</div>

<Toaster />

<style>
	.topbar {
		display: flex;
		align-items: center;
		justify-content: space-between;
		max-width: 40rem;
		margin: 0 auto;
		padding: 0.6rem 1rem 0;
	}

	.brand {
		font-weight: 700;
		color: var(--text);
		text-decoration: none;
		font-size: 0.95rem;
	}
</style>
