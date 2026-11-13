<script lang="ts">
	import PageStub from '#lib/components/PageStub.svelte';
</script>

<svelte:head>
	<title>openJob</title>
	<meta name="description" content="Assistente local-first para a busca de emprego" />
</svelte:head>

<h1 class="page-title">openJob</h1>
<p class="page-subtitle">Seu assistente de busca de emprego, rodando local.</p>

<div class="cards">
	<a class="card" href="/curriculum">
		<span class="card-icon">📄</span>
		<span class="card-text">
			<strong>Currículo</strong>
			<small>Base e versões adaptadas</small>
		</span>
	</a>
	<a class="card" href="/jobs">
		<span class="card-icon">💼</span>
		<span class="card-text">
			<strong>Vagas</strong>
			<small>Adicionar e acompanhar vagas</small>
		</span>
	</a>
	<a class="card" href="/applications">
		<span class="card-icon">📮</span>
		<span class="card-text">
			<strong>Candidaturas</strong>
			<small>Status de cada aplicação</small>
		</span>
	</a>
	<a class="card" href="/settings">
		<span class="card-icon">⚙️</span>
		<span class="card-text">
			<strong>Configurações</strong>
			<small>Chaves e preferências</small>
		</span>
	</a>
</div>

<PageStub
	title="Dashboard"
	subtitle="Resumo do progresso das candidaturas"
	emoji="📊"
/>

<style>
	.cards {
		display: grid;
		grid-template-columns: repeat(2, 1fr);
		gap: 0.625rem;
		margin-bottom: 1.5rem;
	}

	.card {
		display: flex;
		align-items: center;
		gap: 0.625rem;
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: var(--radius);
		padding: 0.75rem;
		text-decoration: none;
		color: inherit;
		-webkit-tap-highlight-color: transparent;
	}

	.card:active {
		background: var(--accent-soft);
	}

	.card-icon {
		font-size: 1.375rem;
	}

	.card-text {
		display: flex;
		flex-direction: column;
		line-height: 1.25;
	}

	.card-text small {
		color: var(--text-muted);
		font-size: 0.75rem;
	}
</style>
