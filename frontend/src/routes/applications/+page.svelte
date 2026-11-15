<script lang="ts">
	import { onMount } from 'svelte';

	// Checkpoint 5: lista mínima do pipeline. Colunas/estágios e mudança de
	// status (PATCH /api/applications/:id) chegam no Checkpoint 7.
	interface AppItem {
		id: number;
		job_id: number;
		status: string;
		sent_at: string | null;
		updated_at: string;
		has_tailored: boolean;
		job?: {
			title: string;
			company: string;
			url: string;
			score_fit: number | null;
		};
	}

	let apps: AppItem[] = $state([]);
	let loading = $state(true);
	let error = $state('');

	onMount(async () => {
		try {
			const res = await fetch('/api/applications');
			if (res.ok) {
				const body = await res.json();
				apps = body.applications ?? [];
			} else {
				const err = await res.json().catch(() => ({}));
				error = err.error ?? `Erro ${res.status} ao carregar candidaturas.`;
			}
		} catch {
			error = 'Falha de rede ao carregar candidaturas.';
		} finally {
			loading = false;
		}
	});

	function dateFmt(iso: string): string {
		return new Date(iso).toLocaleDateString('pt-BR', {
			day: '2-digit',
			month: 'short'
		});
	}
</script>

<svelte:head>
	<title>Candidaturas · openJob</title>
</svelte:head>

<h1 class="page-title">Candidaturas</h1>
<p class="page-subtitle">Pipeline das vagas que você aprovou.</p>

{#if error}
	<div class="status error" role="status">{error}</div>
{/if}

{#if loading}
	<p class="empty">Carregando candidaturas…</p>
{:else if apps.length === 0}
	<p class="empty">
		Nenhuma candidatura ainda — aprove uma vaga em <a href="/jobs">Vagas</a>.
	</p>
{:else}
	<div class="list">
		{#each apps as app (app.id)}
			<article class="card">
				<header class="card-head">
					<div>
						<h2>{app.job?.title ?? `Vaga #${app.job_id}`}</h2>
						<p class="muted">{app.job?.company ?? ''}</p>
					</div>
					<span class="pill st-{app.status}">{app.status}</span>
				</header>

				{#if app.job?.url}
					<a class="link" href={app.job.url} target="_blank" rel="noopener noreferrer">
						{app.job.url}
					</a>
				{/if}

				{#if app.job?.score_fit !== null && app.job?.score_fit !== undefined}
					<p class="muted small">Score de fit: {app.job.score_fit}/100</p>
				{/if}

				<p class="muted small">Atualizada em {dateFmt(app.updated_at)}</p>
			</article>
		{/each}
	</div>
{/if}

<style>
	.card {
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: var(--radius);
		padding: 1rem;
		margin-bottom: 1rem;
	}

	.card-head {
		display: flex;
		align-items: flex-start;
		justify-content: space-between;
		gap: 0.75rem;
	}

	.card-head h2 {
		font-size: 1rem;
		margin: 0;
	}

	.pill {
		font-size: 0.75rem;
		font-weight: 600;
		border-radius: 999px;
		padding: 0.15rem 0.6rem;
		background: var(--accent-soft);
		color: var(--accent);
		flex-shrink: 0;
	}

	.pill.st-rejected {
		background: #fef2f2;
		color: #b91c1c;
	}

	.pill.st-offer {
		background: #ecfdf5;
		color: #047857;
	}

	.muted {
		color: var(--text-muted);
		font-size: 0.85rem;
		margin: 0.1rem 0 0;
	}

	.small {
		font-size: 0.8rem;
		margin: 0.35rem 0 0;
	}

	.link {
		display: block;
		color: var(--accent);
		font-size: 0.82rem;
		margin-top: 0.4rem;
		overflow-wrap: anywhere;
	}

	.empty {
		color: var(--text-muted);
		text-align: center;
		padding: 1.5rem 0;
	}

	.empty a {
		color: var(--accent);
		font-weight: 600;
	}

	.status {
		border-radius: var(--radius);
		padding: 0.6rem 0.8rem;
		font-size: 0.9rem;
		margin-bottom: 0.75rem;
	}

	.status.error {
		background: #fef2f2;
		color: #b91c1c;
	}
</style>
