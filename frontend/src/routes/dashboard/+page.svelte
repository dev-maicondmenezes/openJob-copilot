<script lang="ts">
	import { t } from '#lib/i18n/t.js';
	import PageHeader from '#lib/ui/PageHeader.svelte';
	import Card from '#lib/ui/Card.svelte';
	import Icon from '#lib/ui/Icon.svelte';
	import EmptyState from '#lib/ui/EmptyState.svelte';

	const shortcuts = $derived([
		{ href: '/jobs', title: t('common.jobs'), description: t('home.shortcuts.jobs_desc'), icon: 'briefcase' },
		{ href: '/applications', title: t('common.applications'), description: t('home.shortcuts.applications_desc'), icon: 'clipboard' },
		{ href: '/curriculum', title: t('common.curriculum'), description: t('home.shortcuts.curriculum_desc'), icon: 'file' },
		{ href: '/settings', title: t('common.settings'), description: t('home.shortcuts.settings_desc'), icon: 'settings' }
	]);
</script>

<svelte:head>
	<title>{t('home.dashboard')} · {t('common.app_name')}</title>
</svelte:head>

<PageHeader
	title={t('home.dashboard')}
	subtitle={t('home.dashboard_desc')}
	breadcrumb={[{ label: t('common.home'), href: '/' }, { label: t('home.dashboard') }]}
/>

<div class="grid grid-2 app-links">
	{#each shortcuts as item (item.href)}
		<Card href={item.href} class="shortcut">
			<div class="shortcut-inner">
				<span class="shortcut-icon"><Icon name={item.icon} size={24} /></span>
				<span class="shortcut-text">
					<strong>{item.title}</strong>
					<small>{item.description}</small>
				</span>
				<Icon name="chevron-right" size={20} />
			</div>
		</Card>
	{/each}
</div>

<Card title={t('home.dashboard')} subtitle={t('home.dashboard_desc')} class="dashboard-stub">
	<EmptyState
		icon="sparkles"
		title={t('home.coming_soon')}
		description={t('home.coming_soon_desc')}
	/>
</Card>

<style>
	.app-links {
		margin-bottom: var(--space-6);
	}

	.shortcut-inner {
		display: flex;
		align-items: center;
		gap: var(--space-3);
	}

	.shortcut-icon {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		width: 44px;
		height: 44px;
		flex-shrink: 0;
		border-radius: var(--radius-md);
		background: var(--color-info-soft);
		color: var(--color-primary);
	}

	.shortcut-text {
		display: flex;
		flex-direction: column;
		min-width: 0;
		flex: 1;
		line-height: 1.3;
	}

	.shortcut-text small {
		margin-top: var(--space-1);
		font-size: var(--text-caption);
		line-height: var(--leading-caption);
		color: var(--color-text-muted);
	}
</style>
