<script lang="ts">
	type Section = 'llm' | 'sources' | 'discovery' | 'notifications';

	interface Props {
		active?: Section;
	}

	let { active = 'llm' }: Props = $props();

	const links: { key: Section; href: string; label: string; icon: string }[] = [
		{ key: 'llm', href: '/settings', label: 'Provedor IA (BYOK)', icon: 'key' },
		{ key: 'sources', href: '/settings/sources', label: 'Fontes de Vagas', icon: 'rss' },
		{ key: 'discovery', href: '/settings/discovery', label: 'Busca & Filtros', icon: 'sliders' },
		{ key: 'notifications', href: '/settings/notifications', label: 'Notificações & Tarefas', icon: 'bell' }
	];
</script>

<nav class="settings-nav" aria-label="Seções de configuração">
	{#each links as link (link.key)}
		<a
			href={link.href}
			class="nav-item"
			class:active={active === link.key}
			aria-current={active === link.key ? 'page' : undefined}
		>
			{link.label}
		</a>
	{/each}
</nav>

<style>
	.settings-nav {
		display: flex;
		gap: var(--space-1);
		margin-bottom: var(--space-6);
		border-bottom: 1px solid var(--color-border);
		overflow-x: auto;
		scrollbar-width: thin;
	}

	.nav-item {
		display: inline-flex;
		align-items: center;
		min-height: var(--touch-target);
		padding: 0 var(--space-3);
		border-bottom: 2px solid transparent;
		border-radius: var(--radius-sm) var(--radius-sm) 0 0;
		font-size: var(--text-caption);
		font-weight: var(--weight-medium);
		text-decoration: none;
		color: var(--color-text-muted);
		white-space: nowrap;
	}

	.nav-item:hover {
		color: var(--color-text-primary);
		background: var(--color-surface-elevated);
	}

	.nav-item.active {
		color: var(--color-primary);
		border-bottom-color: var(--color-primary);
		font-weight: var(--weight-semibold);
	}
</style>
