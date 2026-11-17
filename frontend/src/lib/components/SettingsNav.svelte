<script lang="ts">
	import { t } from '#lib/i18n/t.js';
	type Section = 'llm' | 'sources' | 'discovery' | 'notifications' | 'appearance';

	interface Props {
		active?: Section;
	}

	let { active = 'llm' }: Props = $props();

	const links = $derived<{ key: Section; href: string; label: string; icon: string }[]>([
		{ key: 'llm', href: '/settings', label: t('nav.settings_llm'), icon: 'key' },
		{ key: 'sources', href: '/settings/sources', label: t('nav.settings_sources'), icon: 'rss' },
		{ key: 'discovery', href: '/settings/discovery', label: t('nav.settings_discovery'), icon: 'sliders' },
		{ key: 'notifications', href: '/settings/notifications', label: t('nav.settings_notifications'), icon: 'bell' },
		{ key: 'appearance', href: '/settings/appearance', label: t('nav.settings_appearance'), icon: 'palette' }
	]);
</script>

<nav class="settings-nav" aria-label={t('nav.settings_sections')}>
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
