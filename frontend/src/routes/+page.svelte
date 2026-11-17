<script lang="ts">
	import Card from '#lib/ui/Card.svelte';
	import Icon from '#lib/ui/Icon.svelte';
	import EmptyState from '#lib/ui/EmptyState.svelte';

	const shortcuts = [
		{ href: '/curriculum', title: 'Currículo', description: 'Base e versões adaptadas', icon: 'file' },
		{ href: '/jobs', title: 'Vagas', description: 'Adicionar e acompanhar vagas', icon: 'briefcase' },
		{ href: '/applications', title: 'Candidaturas', description: 'Status de cada aplicação', icon: 'clipboard' },
		{ href: '/settings', title: 'Configurações', description: 'Chaves e preferências', icon: 'settings' }
	];
</script>

<svelte:head>
	<title>openJob</title>
	<meta name="description" content="Assistente local-first para a busca de emprego" />
</svelte:head>

<!-- Título e tagline agora vivem no header global do shell. -->
<h1 class="visually-hidden">Início</h1>

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

<Card title="Dashboard" subtitle="Resumo do progresso das candidaturas" class="dashboard-stub">
	<EmptyState
		icon="sparkles"
		title="Seu resumo chega em breve"
		description="Por enquanto, use os atalhos acima para navegar pelas seções do openJob."
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
