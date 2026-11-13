<script lang="ts">
	import { page } from '$app/state';

	const items = [
		{
			href: '/',
			label: 'Início',
			icon: 'home'
		},
		{
			href: '/curriculum',
			label: 'Currículo',
			icon: 'document'
		},
		{
			href: '/jobs',
			label: 'Vagas',
			icon: 'briefcase'
		},
		{
			href: '/applications',
			label: 'Candidaturas',
			icon: 'list'
		},
		{
			href: '/settings',
			label: 'Config',
			icon: 'gear'
		}
	];

	function is_active(href: string, pathname: string): boolean {
		return href === '/' ? pathname === '/' : pathname.startsWith(href);
	}
</script>

<nav class="bottom-nav" aria-label="Navegação principal">
	{#each items as item (item.href)}
		<a
			href={item.href}
			class="item"
			class:active={is_active(item.href, page.url.pathname)}
			aria-current={is_active(item.href, page.url.pathname) ? 'page' : undefined}
		>
			<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
				{#if item.icon === 'home'}
					<path d="M3 10.5 12 3l9 7.5" />
					<path d="M5 9.5V21h14V9.5" />
					<path d="M9.5 21v-6h5v6" />
				{:else if item.icon === 'document'}
					<path d="M7 3h7l4 4v14H7z" />
					<path d="M14 3v4h4" />
					<path d="M10 12h5M10 16h5" />
				{:else if item.icon === 'briefcase'}
					<rect x="3.5" y="7.5" width="17" height="12" rx="2" />
					<path d="M9 7.5V6a2 2 0 0 1 2-2h2a2 2 0 0 1 2 2v1.5" />
					<path d="M3.5 12.5h17" />
				{:else if item.icon === 'list'}
					<path d="M4 5.5h16" />
					<path d="M4 12h16" />
					<path d="M4 18.5h10" />
				{:else if item.icon === 'gear'}
					<circle cx="12" cy="12" r="3.2" />
					<path
						d="M19.4 15a1.6 1.6 0 0 0 .3 1.8l.1.1a2 2 0 1 1-2.8 2.8l-.1-.1a1.6 1.6 0 0 0-1.8-.3 1.6 1.6 0 0 0-1 1.5V21a2 2 0 1 1-4 0v-.1a1.6 1.6 0 0 0-1-1.5 1.6 1.6 0 0 0-1.8.3l-.1.1a2 2 0 1 1-2.8-2.8l.1-.1a1.6 1.6 0 0 0 .3-1.8 1.6 1.6 0 0 0-1.5-1H3a2 2 0 1 1 0-4h.1a1.6 1.6 0 0 0 1.5-1 1.6 1.6 0 0 0-.3-1.8l-.1-.1a2 2 0 1 1 2.8-2.8l.1.1a1.6 1.6 0 0 0 1.8.3h.1a1.6 1.6 0 0 0 1-1.5V3a2 2 0 1 1 4 0v.1a1.6 1.6 0 0 0 1 1.5h.1a1.6 1.6 0 0 0 1.8-.3l.1-.1a2 2 0 1 1 2.8 2.8l-.1.1a1.6 1.6 0 0 0-.3 1.8v.1a1.6 1.6 0 0 0 1.5 1H21a2 2 0 1 1 0 4h-.1a1.6 1.6 0 0 0-1.5 1z"
					/>
				{/if}
			</svg>
			<span class="label">{item.label}</span>
		</a>
	{/each}
</nav>

<style>
	.bottom-nav {
		position: fixed;
		left: 0;
		right: 0;
		bottom: 0;
		z-index: 10;
		display: flex;
		background: var(--surface);
		border-top: 1px solid var(--border);
		/* Respeita a home indicator do iPhone (viewport-fit=cover) */
		padding-bottom: env(safe-area-inset-bottom);
	}

	.item {
		flex: 1;
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 0.125rem;
		padding: 0.5rem 0.25rem 0.375rem;
		color: var(--text-muted);
		text-decoration: none;
		font-size: 0.6875rem;
		/* Alvo de toque confortável (≥ 44px) para o polegar */
		min-height: 3rem;
		-webkit-tap-highlight-color: transparent;
	}

	.item svg {
		width: 1.5rem;
		height: 1.5rem;
	}

	.item.active {
		color: var(--accent);
		font-weight: 600;
	}
</style>
