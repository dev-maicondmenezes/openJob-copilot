<script lang="ts">
	interface Props {
		active?: 'llm' | 'sources' | 'discovery' | 'notifications';
	}
	let { active = 'llm' }: Props = $props();
</script>

<nav class="settings-nav" aria-label="Abas de configuração">
	<a href="/settings" class="nav-item {active === 'llm' ? 'active' : ''}">
		Provedor IA (BYOK)
	</a>
	<a href="/settings/sources" class="nav-item {active === 'sources' ? 'active' : ''}">
		Fontes de Vagas
	</a>
	<a href="/settings/discovery" class="nav-item {active === 'discovery' ? 'active' : ''}">
		Busca & Filtros
	</a>
	<a href="/settings/notifications" class="nav-item {active === 'notifications' ? 'active' : ''}">
		Notificações & Tarefas
	</a>
</nav>

<style>
	.settings-nav {
		display: flex;
		gap: 0.5rem;
		margin-bottom: 1.5rem;
		border-bottom: 1px solid var(--border);
		padding-bottom: 0.5rem;
		overflow-x: auto;
	}

	.nav-item {
		padding: 0.5rem 0.875rem;
		border-radius: var(--radius);
		font-size: 0.875rem;
		font-weight: 500;
		text-decoration: none;
		color: var(--text-muted);
		transition: all 0.15s ease;
		white-space: nowrap;
	}

	.nav-item:hover {
		color: var(--text);
		background: var(--surface);
	}

	.nav-item.active {
		color: var(--primary, #2563eb);
		background: var(--accent-soft, #eff6ff);
		font-weight: 600;
	}
</style>
