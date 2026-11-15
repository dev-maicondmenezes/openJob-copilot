<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';

	interface Job {
		id: number;
		title: string;
		company: string;
		url: string;
		description: string;
		source: string;
		score_fit: number | null;
		score_reason: string;
		status: string;
		created_at: string;
		external_id?: string;
		matched_keywords?: string[];
		matched_resume_terms?: string[];
	}

	let jobs: Job[] = $state([]);
	let loading = $state(true);
	let llmConfigured = $state(true);

	// Discovery state
	let discovering = $state(false);
	let discoveryKeywords = $state<string[]>([]);
	let resumeFilterActive = $state(false);
	let lastRunStats = $state<{
		fetched: number;
		filtered_keywords: number;
		filtered_resume: number;
		new_jobs: number;
		duplicates: number;
		scored: number;
	} | null>(null);

	// Form state — a URL é a fonte primária: colar o link preenche o resto.
	let title = $state('');
	let company = $state('');
	let url = $state('');
	let description = $state('');
	let saving = $state(false);
	let fetching = $state(false);

	// Per-action busy flags + feedback
	let scoringId = $state<number | null>(null);
	let actingId = $state<number | null>(null);
	let status: { kind: 'success' | 'error'; text: string } | null = $state(null);

	const canSave = $derived(
		title.trim() !== '' && company.trim() !== '' && description.trim() !== ''
	);

	// Preenche o formulário a partir da URL da vaga (POST /api/jobs/fetch).
	// Nada é salvo automaticamente — os campos vêm preenchidos para revisão.
	async function fetchDraft(link: string) {
		const target = link.trim();
		if (!/^https?:\/\//.test(target) || fetching) return;
		fetching = true;
		status = null;
		try {
			const res = await fetch('/api/jobs/fetch', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ url: target })
			});
			const body = await res.json().catch(() => ({}));
			if (res.ok) {
				url = body.url ?? target;
				if (body.title) title = body.title;
				if (body.company) company = body.company;
				if (body.description) description = body.description;
				if (body.exists) {
					status = { kind: 'error', text: 'Atenção: já existe uma vaga cadastrada com essa URL.' };
				} else if (!body.description) {
					status = {
						kind: 'error',
						text: 'Página lida, mas sem descrição extraível — cole a JD manualmente.'
					};
				} else {
					status = { kind: 'success', text: 'Dados preenchidos a partir da URL — revise antes de salvar.' };
				}
			} else {
				status = { kind: 'error', text: body.error ?? `Erro ${res.status} ao buscar a vaga.` };
			}
		} catch {
			status = { kind: 'error', text: 'Falha de rede ao buscar a vaga.' };
		} finally {
			fetching = false;
		}
	}

	function onUrlPaste(e: ClipboardEvent) {
		const text = e.clipboardData?.getData('text') ?? '';
		if (/^https?:\/\//.test(text.trim())) {
			e.preventDefault();
			url = text.trim();
			void fetchDraft(url);
		}
	}

	onMount(async () => {
		await Promise.all([loadJobs(), checkLLM(), loadDiscoveryConfig()]);
	});

	async function loadDiscoveryConfig() {
		try {
			const res = await fetch('/api/settings/discovery');
			if (res.ok) {
				const data = await res.json();
				discoveryKeywords = data.keywords ?? [];
				resumeFilterActive = data.resume_filter?.enabled ?? false;
			}
		} catch {
			// silencioso
		}
	}

	async function triggerDiscovery() {
		if (discovering) return;
		discovering = true;
		status = null;
		try {
			const res = await fetch('/api/jobs/discover', { method: 'POST' });
			if (res.status === 409) {
				status = { kind: 'error', text: 'Já existe uma busca em andamento. Aguarde a conclusão.' };
			} else if (!res.ok) {
				const err = await res.json().catch(() => ({}));
				status = { kind: 'error', text: err.error ?? 'Falha ao iniciar busca automática.' };
				discovering = false;
				return;
			}

			// Polling status
			for (let i = 0; i < 40; i++) {
				await new Promise((r) => setTimeout(r, 1200));
				const sRes = await fetch('/api/jobs/discover/status');
				if (sRes.ok) {
					const sData = await sRes.json();
					if (!sData.running) {
						if (sData.latest_run?.stats) {
							const stats = sData.latest_run.stats;
							lastRunStats = stats;
							status = {
								kind: 'success',
								text: `Busca finalizada: ${stats.fetched} analisadas, ${stats.new_jobs} novas cadastradas (${stats.scored} pontuadas).`
							};
						}
						break;
					}
				}
			}
			await loadJobs();
		} catch {
			status = { kind: 'error', text: 'Falha de rede durante a busca de vagas.' };
		} finally {
			discovering = false;
		}
	}

	async function loadJobs() {
		try {
			const res = await fetch('/api/jobs');
			if (res.ok) {
				const body = await res.json();
				jobs = body.jobs ?? [];
			} else {
				const err = await res.json().catch(() => ({}));
				status = { kind: 'error', text: err.error ?? `Erro ${res.status} ao carregar vagas.` };
			}
		} catch {
			status = { kind: 'error', text: 'Falha de rede ao carregar vagas.' };
		} finally {
			loading = false;
		}
	}

	async function checkLLM() {
		try {
			const res = await fetch('/api/settings/llm-key');
			if (res.ok) llmConfigured = (await res.json()).configured;
		} catch {
			// silencioso — o erro real aparece ao pedir score
		}
	}

	async function save() {
		if (!canSave) return;
		saving = true;
		status = null;
		try {
			const res = await fetch('/api/jobs/manual', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ title, company, url, description })
			});
			const body = await res.json().catch(() => ({}));
			if (res.ok) {
				title = company = url = description = '';
				status = { kind: 'success', text: 'Vaga salva! Cole a próxima ou calcule o score.' };
				await loadJobs();
			} else {
				status = { kind: 'error', text: body.error ?? `Erro ${res.status} ao salvar.` };
			}
		} catch {
			status = { kind: 'error', text: 'Falha de rede ao salvar.' };
		} finally {
			saving = false;
		}
	}

	async function score(job: Job) {
		scoringId = job.id;
		status = null;
		try {
			const res = await fetch(`/api/jobs/${job.id}/score`, { method: 'POST' });
			const body = await res.json().catch(() => ({}));
			if (res.ok) {
				status = { kind: 'success', text: `Score calculado: ${body.score_fit}/100` };
			} else {
				status = { kind: 'error', text: body.error ?? `Erro ${res.status} no score.` };
			}
		} catch {
			status = { kind: 'error', text: 'Falha de rede ao calcular score.' };
		} finally {
			scoringId = null;
			await loadJobs();
		}
	}

	async function approve(job: Job) {
		actingId = job.id;
		status = null;
		try {
			const res = await fetch(`/api/jobs/${job.id}/approve`, { method: 'POST' });
			const body = await res.json().catch(() => ({}));
			if (res.ok) {
				await goto('/applications');
				return;
			}
			status = { kind: 'error', text: body.error ?? `Erro ${res.status} ao aprovar.` };
		} catch {
			status = { kind: 'error', text: 'Falha de rede ao aprovar.' };
		} finally {
			actingId = null;
		}
	}

	async function discard(job: Job) {
		actingId = job.id;
		status = null;
		try {
			const res = await fetch(`/api/jobs/${job.id}/discard`, { method: 'POST' });
			const body = await res.json().catch(() => ({}));
			if (res.ok) {
				status = { kind: 'success', text: 'Vaga descartada.' };
				await loadJobs();
			} else {
				status = { kind: 'error', text: body.error ?? `Erro ${res.status} ao descartar.` };
			}
		} catch {
			status = { kind: 'error', text: 'Falha de rede ao descartar.' };
		} finally {
			actingId = null;
		}
	}

	function scoreClass(score: number): string {
		if (score >= 75) return 'high';
		if (score >= 50) return 'mid';
		return 'low';
	}

	const statusLabels: Record<string, string> = {
		new: 'nova',
		approved: 'aprovada',
		discarded: 'descartada'
	};

	const sourceLabels: Record<string, string> = {
		himalayas: 'Himalayas',
		remotive: 'Remotive',
		jobicy: 'Jobicy',
		remoteok: 'Remote OK',
		manual: 'Manual'
	};
</script>

<svelte:head>
	<title>Vagas · openJob</title>
</svelte:head>

<h1 class="page-title">Vagas</h1>
<p class="page-subtitle">Cole a vaga que você está mirando, calcule o fit e aprove para o pipeline.</p>

{#if status}
	<div class="status {status.kind}" role="status">{status.text}</div>
{/if}

{#if !llmConfigured}
	<div class="status warn" role="status">
		Provedor de IA não configurado — o botão Score precisa da sua chave.
		<a href="/settings">Configurar</a>
	</div>
{/if}

<section class="card discovery-card" aria-label="Descoberta Automática de Vagas">
	<div class="discovery-content">
		<div class="discovery-text">
			<div class="discovery-title-row">
				<h2>Descoberta Automática de Vagas</h2>
				<span class="badge-free">APIs Gratuitas</span>
			</div>
			<p class="hint">
				Busca remota em fontes oficiais com APIs públicas gratuitas (Himalayas, Remotive, Jobicy, Remote OK) com filtros prévios.
			</p>
			<div class="discovery-meta">
				<span class="meta-item">
					<strong>Palavras-chave:</strong>
					{#if discoveryKeywords.length > 0}
						{discoveryKeywords.join(', ')}
					{:else}
						<em>nenhuma configurada</em>
					{/if}
				</span>
				<span class="meta-item">
					<strong>Filtro por currículo:</strong>
					{resumeFilterActive ? 'Ativo' : 'Desativado'}
				</span>
				<a href="/settings/discovery" class="settings-link">⚙ Ajustar busca & filtros</a>
			</div>
		</div>

		<div class="discovery-actions">
			<button
				type="button"
				class="primary btn-discover"
				disabled={discovering}
				onclick={triggerDiscovery}
			>
				{discovering ? 'Buscando vagas…' : '⚡ Buscar Vagas Agora'}
			</button>
		</div>
	</div>

	{#if lastRunStats}
		<div class="discovery-stats">
			<span>Analisadas: <strong>{lastRunStats.fetched}</strong></span>
			<span>Após palavras-chave: <strong>{lastRunStats.fetched - lastRunStats.filtered_keywords}</strong></span>
			<span>Após currículo: <strong>{lastRunStats.fetched - lastRunStats.filtered_keywords - lastRunStats.filtered_resume}</strong></span>
			<span>Novas adicionadas: <strong class="highlight-green">{lastRunStats.new_jobs}</strong></span>
			<span>Com score IA: <strong>{lastRunStats.scored}</strong></span>
		</div>
	{/if}
</section>

<section class="card" aria-label="Nova vaga">
	<header class="card-head">
		<h2>Nova vaga</h2>
	</header>
	<form onsubmit={(e) => { e.preventDefault(); save(); }}>
		<label for="url">URL da vaga</label>
		<div class="url-row">
			<input
				id="url"
				type="url"
				placeholder="https://jobs.example.com/vaga"
				bind:value={url}
				onpaste={onUrlPaste}
				spellcheck="false"
			/>
			<button
				type="button"
				class="secondary fill"
				disabled={fetching || !/^https?:\/\//.test(url.trim())}
				onclick={() => fetchDraft(url)}
			>
				{fetching ? 'Buscando…' : 'Preencher'}
			</button>
		</div>
		<p class="hint">Cole o link da vaga — título, empresa e descrição são preenchidos automaticamente.</p>

		<label for="title">Título</label>
		<input id="title" type="text" placeholder="Senior Go Developer" bind:value={title} />

		<label for="company">Empresa</label>
		<input id="company" type="text" placeholder="Acme Corp" bind:value={company} />

		<label for="description">Descrição da vaga (JD)</label>
		<textarea
			id="description"
			rows="8"
			placeholder="Cole aqui a descrição completa da vaga — HTML é limpo automaticamente."
			bind:value={description}
		></textarea>

		<div class="actions">
			<button type="submit" class="primary" disabled={saving || !canSave}>
				{saving ? 'Salvando…' : 'Salvar vaga'}
			</button>
		</div>
	</form>
</section>

{#if loading}
	<p class="empty">Carregando vagas…</p>
{:else if jobs.length === 0}
	<p class="empty">Nenhuma vaga ainda — cole a primeira acima.</p>
{:else}
	<div class="list">
		{#each jobs as job (job.id)}
			<article class="card job" class:discarded={job.status === 'discarded'}>
				<header class="card-head">
					<div class="job-title">
						<h2>{job.title}</h2>
						<div class="company-row">
							<span class="muted">{job.company}</span>
							{#if job.source}
								<span class="source-tag">{sourceLabels[job.source] ?? job.source}</span>
							{/if}
						</div>
					</div>
					<div class="badges">
						{#if job.score_fit !== null}
							<span class="score {scoreClass(job.score_fit)}">{job.score_fit}</span>
						{:else}
							<span class="score none">sem score</span>
						{/if}
						<span class="pill st-{job.status}">{statusLabels[job.status] ?? job.status}</span>
					</div>
				</header>

				{#if job.url}
					<div class="link-row">
						<a class="link" href={job.url} target="_blank" rel="noopener noreferrer">{job.url}</a>
						{#if job.source === 'remotive' || job.source === 'remoteok' || job.source === 'jobicy'}
							<span class="attrib-badge">via {sourceLabels[job.source] ?? job.source}</span>
						{/if}
					</div>
				{/if}

				{#if (job.matched_keywords && job.matched_keywords.length > 0) || (job.matched_resume_terms && job.matched_resume_terms.length > 0)}
					<div class="matched-box">
						{#if job.matched_keywords && job.matched_keywords.length > 0}
							<div class="matched-group">
								<span class="matched-label">Palavras-chave:</span>
								{#each job.matched_keywords as kw}
									<span class="chip-kw">{kw}</span>
								{/each}
							</div>
						{/if}
						{#if job.matched_resume_terms && job.matched_resume_terms.length > 0}
							<div class="matched-group">
								<span class="matched-label">Aderência ao currículo:</span>
								{#each job.matched_resume_terms as rt}
									<span class="chip-resume">{rt}</span>
								{/each}
							</div>
						{/if}
					</div>
				{/if}

				{#if job.score_reason}
					<details class="reason">
						<summary>Justificativa do score</summary>
						<p>{job.score_reason}</p>
					</details>
				{/if}

				<div class="actions">
					<button
						class="secondary"
						disabled={scoringId === job.id || actingId === job.id || job.status === 'discarded'}
						onclick={() => score(job)}
					>
						{scoringId === job.id ? 'Calculando…' : 'Score'}
					</button>
					<button
						class="primary"
						disabled={actingId === job.id || scoringId === job.id || job.status === 'approved' || job.status === 'discarded'}
						onclick={() => approve(job)}
					>
						{actingId === job.id ? 'Aprovando…' : 'Aprovar'}
					</button>
					<button
						class="danger"
						disabled={actingId === job.id || scoringId === job.id || job.status === 'discarded'}
						onclick={() => discard(job)}
					>
						{actingId === job.id ? 'Descartando…' : 'Descartar'}
					</button>
				</div>
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
		margin-bottom: 0.75rem;
	}

	.card-head h2 {
		font-size: 1rem;
		margin: 0;
	}

	label {
		display: block;
		font-size: 0.85rem;
		font-weight: 600;
		margin: 0.75rem 0 0.25rem;
	}

	input,
	textarea {
		width: 100%;
		border: 1px solid var(--border);
		border-radius: var(--radius);
		padding: 0.6rem 0.75rem;
		font-size: 1rem;
		background: var(--bg);
		color: var(--text);
		font-family: inherit;
	}

	textarea {
		resize: vertical;
		line-height: 1.4;
	}

	input:focus,
	textarea:focus {
		outline: 2px solid var(--accent);
		outline-offset: 1px;
		border-color: var(--accent);
	}

	.url-row {
		display: flex;
		gap: 0.5rem;
	}

	.url-row input {
		flex: 1;
		min-width: 0;
	}

	.url-row .fill {
		flex-shrink: 0;
		padding: 0 0.9rem;
	}

	.hint {
		color: var(--text-muted);
		font-size: 0.8rem;
		margin: 0.3rem 0 0;
	}

	.actions {
		display: flex;
		gap: 0.5rem;
		margin-top: 1rem;
	}

	.primary,
	.secondary,
	.danger {
		flex: 1;
		border: 0;
		border-radius: var(--radius);
		padding: 0.6rem 0.5rem;
		font-size: 0.9rem;
		font-weight: 600;
		cursor: pointer;
		white-space: nowrap;
	}

	.primary {
		background: var(--accent);
		color: #fff;
	}

	.secondary {
		background: var(--accent-soft);
		color: var(--accent);
	}

	.danger {
		background: #fef2f2;
		color: #b91c1c;
	}

	.primary:disabled,
	.secondary:disabled,
	.danger:disabled {
		opacity: 0.5;
		cursor: default;
	}

	.status {
		border-radius: var(--radius);
		padding: 0.6rem 0.8rem;
		font-size: 0.9rem;
		margin-bottom: 0.75rem;
	}

	.status.success {
		background: #ecfdf5;
		color: #047857;
	}

	.status.error {
		background: #fef2f2;
		color: #b91c1c;
	}

	.status.warn {
		background: #fffbeb;
		color: #b45309;
	}

	.status a {
		color: inherit;
		font-weight: 600;
	}

	.badges {
		display: flex;
		align-items: center;
		gap: 0.4rem;
		flex-shrink: 0;
	}

	.score {
		font-size: 0.8rem;
		font-weight: 700;
		border-radius: 999px;
		padding: 0.2rem 0.6rem;
	}

	.score.high {
		background: #ecfdf5;
		color: #047857;
	}

	.score.mid {
		background: #fffbeb;
		color: #b45309;
	}

	.score.low {
		background: #fef2f2;
		color: #b91c1c;
	}

	.score.none {
		background: var(--bg);
		color: var(--text-muted);
		font-weight: 600;
	}

	.pill {
		font-size: 0.72rem;
		font-weight: 600;
		border-radius: 999px;
		padding: 0.15rem 0.55rem;
		background: var(--bg);
		color: var(--text-muted);
	}

	.pill.st-approved {
		background: var(--accent-soft);
		color: var(--accent);
	}

	.job.discarded {
		opacity: 0.55;
	}

	.job-title {
		min-width: 0;
	}

	.muted {
		color: var(--text-muted);
		font-size: 0.85rem;
		margin: 0.1rem 0 0;
	}

	.link {
		display: block;
		color: var(--accent);
		font-size: 0.82rem;
		margin-bottom: 0.4rem;
		overflow-wrap: anywhere;
	}

	.reason {
		margin-bottom: 0.4rem;
		font-size: 0.88rem;
	}

	.reason summary {
		cursor: pointer;
		color: var(--accent);
		font-weight: 600;
	}

	.reason p {
		margin: 0.5rem 0 0;
		color: var(--text);
	}

	.empty {
		color: var(--text-muted);
		text-align: center;
		padding: 1.5rem 0;
	}

	/* Discovery Banner Styles */
	.discovery-card {
		background: linear-gradient(to right bottom, var(--surface), var(--surface-soft, #f8fafc));
		border-left: 4px solid var(--primary, #2563eb);
	}

	.discovery-content {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 1rem;
		flex-wrap: wrap;
	}

	.discovery-text {
		flex: 1;
		min-width: 280px;
	}

	.discovery-title-row {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		margin-bottom: 0.25rem;
	}

	.discovery-title-row h2 {
		margin: 0;
		font-size: 1.05rem;
	}

	.badge-free {
		font-size: 0.65rem;
		font-weight: 700;
		text-transform: uppercase;
		letter-spacing: 0.05em;
		background: #ecfdf5;
		color: #047857;
		padding: 0.15rem 0.45rem;
		border-radius: 999px;
	}

	.discovery-meta {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: 0.75rem;
		font-size: 0.8rem;
		color: var(--text-muted);
		margin-top: 0.5rem;
	}

	.settings-link {
		color: var(--primary, #2563eb);
		text-decoration: underline;
		font-weight: 500;
	}

	.btn-discover {
		font-weight: 600;
		white-space: nowrap;
		padding: 0.6rem 1rem;
	}

	.discovery-stats {
		display: flex;
		flex-wrap: wrap;
		gap: 1rem;
		margin-top: 1rem;
		padding-top: 0.75rem;
		border-top: 1px dashed var(--border);
		font-size: 0.8rem;
		color: var(--text-muted);
	}

	.highlight-green {
		color: #047857;
		font-size: 0.95rem;
	}

	.company-row {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		margin-top: 0.1rem;
	}

	.source-tag {
		font-size: 0.7rem;
		font-weight: 600;
		background: var(--surface-soft, #f1f5f9);
		color: var(--text-muted);
		padding: 0.1rem 0.4rem;
		border-radius: 4px;
	}

	.link-row {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		margin-bottom: 0.4rem;
		flex-wrap: wrap;
	}

	.attrib-badge {
		font-size: 0.65rem;
		background: #eff6ff;
		color: #1d4ed8;
		padding: 0.1rem 0.35rem;
		border-radius: 4px;
	}

	.matched-box {
		background: var(--surface-soft, #f8fafc);
		border-radius: 6px;
		padding: 0.5rem;
		margin-bottom: 0.6rem;
		display: flex;
		flex-direction: column;
		gap: 0.35rem;
	}

	.matched-group {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: 0.35rem;
	}

	.matched-label {
		font-size: 0.72rem;
		font-weight: 600;
		color: var(--text-muted);
	}

	.chip-kw {
		font-size: 0.7rem;
		background: #dbeafe;
		color: #1e40af;
		padding: 0.05rem 0.35rem;
		border-radius: 3px;
	}

	.chip-resume {
		font-size: 0.7rem;
		background: #dcfce7;
		color: #166534;
		padding: 0.05rem 0.35rem;
		border-radius: 3px;
	}
</style>
