<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { toast } from '#lib/toast.svelte.ts';

	import PageHeader from '#lib/ui/PageHeader.svelte';
	import Card from '#lib/ui/Card.svelte';
	import Field from '#lib/ui/Field.svelte';
	import Input from '#lib/ui/Input.svelte';
	import Textarea from '#lib/ui/Textarea.svelte';
	import Button from '#lib/ui/Button.svelte';
	import Badge from '#lib/ui/Badge.svelte';
	import Alert from '#lib/ui/Alert.svelte';
	import Select from '#lib/ui/Select.svelte';
	import MultiSelect from '#lib/ui/MultiSelect.svelte';
	import Dialog from '#lib/ui/Dialog.svelte';
	import Drawer from '#lib/ui/Drawer.svelte';
	import DataView from '#lib/ui/DataView.svelte';
	import Icon from '#lib/ui/Icon.svelte';
	import IconButton from '#lib/ui/IconButton.svelte';
	import IconLinkButton from '#lib/ui/IconLinkButton.svelte';
	import Pagination from '#lib/ui/Pagination.svelte';
	import PageFooter from '#lib/ui/PageFooter.svelte';
	import EmptyState from '#lib/ui/EmptyState.svelte';
	import JobDetail from '#lib/components/JobDetail.svelte';
	import type { DataColumn } from '#lib/ui/data.js';
	import { viewport } from '#lib/ui/media.svelte.js';
	import { SOURCE_LABELS, STATUS_LABELS, scoreTone, type Job } from '#lib/types/job.js';

	let jobs: Job[] = $state([]);
	let loading = $state(true);
	let llmConfigured = $state(true);

	// Discovery (o botão do header dispara a busca; o resumo de execução saiu da tela)
	let discovering = $state(false);
	let discoveryKeywords = $state<string[]>([]);

	// Formulário: a URL é a fonte primária.
	let title = $state('');
	let company = $state('');
	let url = $state('');
	let description = $state('');
	let saving = $state(false);
	let fetching = $state(false);

	let scoringId = $state<number | null>(null);
	let actingId = $state<number | null>(null);
	let status: { kind: 'success' | 'error'; text: string } | null = $state(null);

	// Filtros — apenas UI sobre dados já carregados (nenhuma mudança funcional).
	let statusFilter = $state('all');
	let sourceFilter = $state<string[]>([]);
	let searchTerm = $state(page.url.searchParams.get('q') ?? '');
	let filtersOpen = $state(false);
	let newJobOpen = $state(false);
	let detailJob: Job | null = $state(null);
	let selectedJob: Job | null = $state(null);
	let selectedIds = $state<number[]>([]);
	let sortKey = $state('score');
	let sortDir = $state<'asc' | 'desc'>('desc');

	const PAGE_SIZE = 10;
	let currentPage = $state(1);
	let visibleCount = $state(8);
	let loadingMore = $state(false);


	const canSave = $derived(
		title.trim() !== '' && company.trim() !== '' && description.trim() !== ''
	);

	const sourceOptions = $derived(
		Array.from(new Set(jobs.map((job) => job.source).filter(Boolean)))
			.sort()
			.map((source) => ({ value: source, label: SOURCE_LABELS[source] ?? source }))
	);

	const statusOptions = [
		{ value: 'all', label: 'Todos os status' },
		{ value: 'new', label: 'Novas' },
		{ value: 'approved', label: 'Aprovadas' },
		{ value: 'discarded', label: 'Descartadas' }
	];

	const filteredJobs = $derived(
		jobs.filter((job) => {
			if (statusFilter !== 'all' && job.status !== statusFilter) return false;
			if (sourceFilter.length > 0 && !sourceFilter.includes(job.source)) return false;
			const q = searchTerm.trim().toLocaleLowerCase('pt-BR');
			if (q && !`${job.title} ${job.company} ${job.description}`.toLocaleLowerCase('pt-BR').includes(q)) return false;
			return true;
		})
	);

	const sortedJobs = $derived.by(() => {
		const column = columns.find((item) => item.key === sortKey);
		const factor = sortDir === 'asc' ? 1 : -1;
		return [...filteredJobs].sort((a, b) => {
			const av = column?.value?.(a) ?? '';
			const bv = column?.value?.(b) ?? '';
			const an = Number(av);
			const bn = Number(bv);
			if (!Number.isNaN(an) && !Number.isNaN(bn) && av !== '' && bv !== '') return (an - bn) * factor;
			return String(av).localeCompare(String(bv), 'pt-BR') * factor;
		});
	});

	const totalPages = $derived(Math.max(1, Math.ceil(sortedJobs.length / PAGE_SIZE)));
	const pagedJobs = $derived(
		sortedJobs.slice((currentPage - 1) * PAGE_SIZE, currentPage * PAGE_SIZE)
	);
	const rangeLabel = $derived(
		`${sortedJobs.length === 0 ? 0 : (currentPage - 1) * PAGE_SIZE + 1}..${Math.min(currentPage * PAGE_SIZE, sortedJobs.length)} de ${sortedJobs.length} vagas`
	);
	// Desktop pagina; phone/tablet carrega progressivamente (sem paginação).
	const visibleJobs = $derived(
		viewport.isDesktop ? pagedJobs : sortedJobs.slice(0, visibleCount)
	);

	function loadMoreJobs(): void {
		if (viewport.isDesktop || loadingMore || visibleCount >= sortedJobs.length) return;
		loadingMore = true;
		setTimeout(() => {
			visibleCount += 8;
			loadingMore = false;
		}, 200);
	}

	const hasActiveFilters = $derived(
		statusFilter !== 'all' || sourceFilter.length > 0 || searchTerm.trim() !== ''
	);

	function resetPage(): void {
		currentPage = 1;
		visibleCount = 8;
	}

	function goToPage(next: number): void {
		currentPage = Math.min(totalPages, Math.max(1, next));
	}

	// Se a lista encolher (filtro/exclusão), mantém a página dentro do intervalo.
	$effect(() => {
		if (currentPage > totalPages) currentPage = totalPages;
	});

	const columns: DataColumn<Job>[] = [
		{ key: 'title', header: 'Vaga', cardLabel: 'Vaga', value: (job) => job.title },
		{ key: 'company', header: 'Empresa', value: (job) => job.company },
		{ key: 'source', header: 'Fonte', value: (job) => SOURCE_LABELS[job.source] ?? job.source },
		{ key: 'score', header: 'Score', value: (job) => (job.score_fit === null ? 'sem score' : String(job.score_fit)) },
		{ key: 'status', header: 'Status', value: (job) => STATUS_LABELS[job.status] ?? job.status },
		{
			key: 'created_at',
			header: 'Data',
			value: (job) => new Date(job.created_at).toLocaleDateString('pt-BR')
		},
	];

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
					status = { kind: 'error', text: 'Página lida, mas sem descrição extraível — cole a JD manualmente.' };
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
			for (let i = 0; i < 40; i++) {
				await new Promise((r) => setTimeout(r, 1200));
				const sRes = await fetch('/api/jobs/discover/status');
				if (sRes.ok) {
					const sData = await sRes.json();
					if (!sData.running) {
						const stats = sData.latest_run?.stats as
							| { fetched: number; new_jobs: number; scored: number }
							| undefined;
						if (stats) {
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
				if (selectedJob) selectedJob = jobs.find((job) => job.id === selectedJob!.id) ?? null;
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
			// silencioso
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
				newJobOpen = false;
				status = { kind: 'success', text: 'Vaga salva! Você pode calcular o score ou aprovar.' };
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
		try {
			const res = await fetch(`/api/jobs/${job.id}/approve`, { method: 'POST' });
			const body = await res.json().catch(() => ({}));
			// Apenas feedback: sem redirect e sem recarregar a lista (UX contínua).
			toast(res.ok ? 'success' : 'error', res.ok
				? `${job.title} aprovada e enviada para candidaturas.`
				: body.error ?? `Erro ${res.status} ao aprovar.`);
		} catch {
			toast('error', 'Falha de rede ao aprovar.');
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
				detailJob = null;
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

	function openDetail(job: Job): void {
		selectedJob = job;
		if (viewport.isDesktop) detailJob = job;
		else void goto(`/jobs/${job.id}`);
	}

	function clearFilters(): void {
		statusFilter = 'all';
		sourceFilter = [];
		searchTerm = '';
		resetPage();
	}

	function onSort(key: string): void {
		if (sortKey === key) sortDir = sortDir === 'asc' ? 'desc' : 'asc';
		else {
			sortKey = key;
			sortDir = key === 'score' || key === 'created_at' ? 'desc' : 'asc';
		}
	}

	function toggleSelect(id: number, checked: boolean): void {
		selectedIds = checked ? [...selectedIds, id] : selectedIds.filter((item) => item !== id);
	}

	function selectAllVisible(checked: boolean): void {
		const ids = pagedJobs.map((job) => job.id);
		selectedIds = checked
			? Array.from(new Set([...selectedIds, ...ids]))
			: selectedIds.filter((id) => !ids.includes(id));
	}

	function selectAllFiltered(): void {
		selectedIds = sortedJobs.map((job) => job.id);
	}

	function clearSelection(): void {
		selectedIds = [];
	}

	async function runBatch(action: 'score' | 'approve' | 'discard'): Promise<void> {
		const targets = jobs.filter((job) => selectedIds.includes(job.id));
		if (targets.length === 0) return;
		const verb = { score: 'Score calculado', approve: 'Aprovada', discard: 'Descartada' }[action];
		let ok = 0;
		for (const job of targets) {
			try {
				const res = await fetch(`/api/jobs/${job.id}/${action}`, { method: 'POST' });
				if (res.ok) ok++;
			} catch {
				// segue com os demais
			}
		}
		const failures = targets.length - ok;
		toast(failures > 0 ? 'error' : 'success',
			`${verb}: ${ok} de ${targets.length} vaga(s).${failures > 0 ? ` ${failures} falha(s).` : ''}`);
		if (action !== 'approve') {
			if (action === 'discard') clearSelection();
			await loadJobs();
		}
	}
</script>

<svelte:head>
	<title>Vagas · openJob</title>
</svelte:head>


{#snippet renderCell(job: Job, column: DataColumn<Job>)}
	{#if column.key === 'title'}
		<strong>{job.title}</strong>
	{:else if column.key === 'score'}
		<Badge tone={job.score_fit === null ? 'neutral' : scoreTone(job.score_fit)}>
			{job.score_fit === null ? 'sem score' : `${job.score_fit}/100`}
		</Badge>
	{:else if column.key === 'status'}
		<Badge tone={job.status === 'approved' ? 'success' : job.status === 'discarded' ? 'error' : 'primary'}>
			{STATUS_LABELS[job.status] ?? job.status}
		</Badge>
	{:else}
		{column.value?.(job) ?? ''}
	{/if}
{/snippet}

{#snippet newJobForm()}
	<form id="new-job-form" onsubmit={(e) => { e.preventDefault(); save(); }}>
		<Field label="URL da vaga" for="url">
			<div class="url-row">
				<Input
					id="url"
					type="url"
					placeholder="https://jobs.example.com/vaga"
					bind:value={url}
					onpaste={onUrlPaste}
					spellcheck="false"
				/>
				<Button
					type="button"
					variant="secondary"
					loading={fetching}
					disabled={!/^https?:\/\//.test(url.trim())}
					onclick={() => fetchDraft(url)}
				>
					Preencher
				</Button>
			</div>
		</Field>
		<Field label="Título" for="title">
			<Input id="title" bind:value={title} placeholder="Senior Go Developer" />
		</Field>
		<Field label="Empresa" for="company">
			<Input id="company" bind:value={company} placeholder="Acme Corp" />
		</Field>
		<Field label="Descrição da vaga (JD)" for="description">
			<Textarea id="description" rows={6} bind:value={description} placeholder="Cole aqui a descrição completa da vaga — HTML é limpo automaticamente." />
		</Field>
	</form>
{/snippet}

{#snippet filterIcons()}
	{#if hasActiveFilters}
		<IconButton icon="x" label="Limpar filtros" variant="ghost" onclick={clearFilters} />
	{/if}
	<IconButton icon="check-double" label="Marcar todas as vagas" variant="ghost" onclick={selectAllFiltered} />
	<IconLinkButton href="/settings/sources" icon="settings" label="Configurar fontes de vagas" />
{/snippet}

{#snippet filterFields(prefix: string)}
	<Field label="Status" for="{prefix}-status">
		<Select id="{prefix}-status" value={statusFilter} onchange={(e) => { statusFilter = e.currentTarget.value; resetPage(); }} options={statusOptions} />
	</Field>
	<Field label="Fonte (múltipla)">
		<MultiSelect label="Filtrar por fonte" options={sourceOptions} bind:value={sourceFilter} allLabel="Todas as fontes" onchange={resetPage} />
	</Field>
	<Field label="Buscar nas vagas" for="{prefix}-search">
		<Input id="{prefix}-search" type="search" bind:value={searchTerm} oninput={resetPage} placeholder="Título, empresa ou descrição" />
	</Field>
{/snippet}

<PageHeader
	title="Vagas"
	subtitle="O openJob consulta APIs públicas de vagas e aplica suas palavras-chave como filtro inicial."
	breadcrumb={[{ label: 'Início', href: '/' }, { label: 'Vagas' }]}
>
	{#snippet actions()}
		<Button onclick={triggerDiscovery} loading={discovering}>
			<Icon name="sparkles" size={20} />
			{discovering ? 'Buscando…' : 'Buscar agora'}
		</Button>
		<Button variant="secondary" onclick={() => (newJobOpen = true)}>
			<Icon name="plus" size={20} />
			Nova Vaga
		</Button>
	{/snippet}
</PageHeader>

{#if status}
	<Alert severity={status.kind} title={status.kind === 'error' ? 'Atenção' : 'Sucesso'}>{status.text}</Alert>
{/if}

{#if !llmConfigured}
	<Alert severity="warning" title="Provedor de IA não configurado">
		O botão Score precisa da sua chave. <a href="/settings">Configurar</a>
	</Alert>
{/if}

<Card class="keywords-strip">
	<div class="keywords-strip-inner">
		<span class="keywords-icon"><Icon name="sliders" size={20} /></span>
		<div class="keywords-copy">
			<strong>Palavras-chave configuradas</strong>
			<span class="hint">
				{discoveryKeywords.length > 0
					? discoveryKeywords.join(' · ')
					: 'nenhuma — todas as vagas retornadas serão avaliadas'}
			</span>
		</div>
		<IconLinkButton href="/settings/discovery" icon="external" label="Configurar busca e filtros" />
	</div>
</Card>


<section aria-label="Vagas cadastradas" class="jobs-section">
	<div class="list-head">
		<p class="hint count-mobile">{sortedJobs.length} vagas</p>
		<div class="list-actions">
			<Button variant="ghost" onclick={() => (filtersOpen = true)} class="filters-trigger">
				<Icon name="sliders" size={20} /> Filtros
			</Button>
			<div class="mobile-filter-actions">{@render filterIcons()}</div>
		</div>
	</div>

	<div class="filters-inline">
		{@render filterFields('inline')}
		<div class="inline-filter-actions">{@render filterIcons()}</div>
	</div>

	{#if loading}
		<Card><p class="hint">Carregando vagas…</p></Card>
	{:else if jobs.length === 0}
		<Card>
			<EmptyState
				title="Nenhuma vaga ainda"
				description="Use o botão “Nova Vaga” para cadastrar a primeira ou a descoberta automática."
			>
				<Button variant="secondary" size="sm" onclick={() => (newJobOpen = true)}>Nova Vaga</Button>
			</EmptyState>
		</Card>
	{:else if filteredJobs.length === 0}
		<Card>
			<EmptyState title="Nenhuma vaga com esses filtros" description="Ajuste os filtros para ver mais resultados.">
				<Button variant="secondary" size="sm" onclick={clearFilters}>Limpar filtros</Button>
			</EmptyState>
		</Card>
	{:else}
		<DataView
			columns={columns}
			rows={visibleJobs}
			getKey={(job) => job.id}
			caption="Vagas cadastradas"
			onrowclick={openDetail}
			render={renderCell}
			selectable
			{selectedIds}
			onselect={(id, checked) => toggleSelect(Number(id), checked)}
			onselectall={selectAllVisible}
			sort={{ key: sortKey, dir: sortDir }}
			onsort={onSort}
			scrollable
			onreachEnd={loadMoreJobs}
			{loadingMore}
		/>
	{/if}
</section>

{#snippet barLeading(size: 'sm' | 'md')}
	<IconButton
		icon="check"
		label="Aprovar vagas selecionadas"
		variant="primary"
		size={size}
		disabled={selectedIds.length === 0}
		onclick={() => runBatch('approve')}
	/>
	<IconButton
		icon="sparkles"
		label="Calcular score das vagas selecionadas"
		variant="secondary"
		size={size}
		disabled={selectedIds.length === 0}
		onclick={() => runBatch('score')}
	/>
{/snippet}

{#snippet barActions(size: 'sm' | 'md')}
	<IconButton
		icon="trash"
		label="Descartar vagas selecionadas"
		variant="danger"
		size={size}
		disabled={selectedIds.length === 0}
		onclick={() => runBatch('discard')}
	/>
{/snippet}

{#snippet barCount()}
	<span class="hint">{rangeLabel}</span>
{/snippet}

{#if viewport.isDesktop}
	<PageFooter>
		<Pagination
			page={currentPage}
			totalPages={totalPages}
			onchange={goToPage}
			label="Paginação e ações de vagas"
			leading={barLeading}
			actions={barActions}
			count={barCount}
		/>
	</PageFooter>
{:else if selectedIds.length > 0}
	<!-- No mobile as ações em lote flutuam sobre a lista quando há seleção. -->
	<div class="mobile-actions" role="toolbar" aria-label="Ações em lote">
		<span class="selection-count">{selectedIds.length} selecionada(s)</span>
		<IconButton icon="check" label="Aprovar vagas selecionadas" variant="primary" onclick={() => runBatch('approve')} />
		<IconButton icon="sparkles" label="Calcular score das vagas selecionadas" variant="secondary" onclick={() => runBatch('score')} />
		<IconButton icon="trash" label="Descartar vagas selecionadas" variant="danger" onclick={() => runBatch('discard')} />
	</div>
{/if}

<Dialog open={newJobOpen} onclose={() => (newJobOpen = false)} title="Nova Vaga" description="Cole o link — os dados são preenchidos para revisão antes de salvar.">
	{@render newJobForm()}
	{#snippet footer()}
		<Button variant="ghost" onclick={() => (newJobOpen = false)}>Cancelar</Button>
		<Button type="submit" form="new-job-form" loading={saving} disabled={!canSave}>Salvar vaga</Button>
	{/snippet}
</Dialog>

<Dialog open={filtersOpen} onclose={() => (filtersOpen = false)} title="Filtrar vagas">
	<div class="filter-sheet">
		{@render filterFields('sheet')}
	</div>
	{#snippet footer()}
		<Button variant="ghost" onclick={clearFilters}>Limpar</Button>
		<Button onclick={() => (filtersOpen = false)}>Aplicar</Button>
	{/snippet}
</Dialog>

{#if detailJob}
	<Drawer
		title="Detalhe da vaga"
		description={detailJob.company}
		onclose={() => (detailJob = null)}
	>
		<JobDetail job={detailJob} />
		{#snippet footer()}
			<a class="full-link" href={`/jobs/${detailJob!.id}`}>Abrir página completa</a>
		{/snippet}
	</Drawer>
{/if}

<style>
	/* Cartão bem curto com as palavras-chave da busca. */
	.keywords-strip-inner {
		display: flex;
		align-items: center;
		gap: var(--space-3);
	}

	.keywords-icon {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		width: 36px;
		height: 36px;
		flex-shrink: 0;
		border-radius: var(--radius-md);
		background: var(--color-info-soft);
		color: var(--color-primary);
	}

	.keywords-copy {
		display: flex;
		flex-direction: column;
		gap: var(--space-1);
		flex: 1;
		min-width: 0;
	}

	.keywords-copy strong {
		font-size: var(--text-body);
	}

	form {
		display: flex;
		flex-direction: column;
		gap: var(--space-4);
	}

	.url-row {
		display: flex;
		gap: var(--space-3);
	}

	.url-row :global(.input) {
		flex: 1;
		min-width: 0;
	}

	.jobs-section {
		display: flex;
		flex: 1 1 auto;
		flex-direction: column;
		min-height: 0;
		margin-top: var(--space-6);
	}

	@media (max-width: 767px) {
		.jobs-section {
			margin-top: var(--space-4);
		}

		/* Resumo de palavras-chave em uma linha só no phone. */
		:global(.card.keywords-strip) {
			padding: var(--space-3);
		}

		.keywords-strip-inner {
			gap: var(--space-2);
		}

		.keywords-copy strong {
			display: none;
		}

		.keywords-icon {
			width: 32px;
			height: 32px;
		}
	}

	.jobs-section :global(.table-wrap.scrollable),
	.jobs-section :global(.card-list.scrollable) {
		flex: 1 1 auto;
		/* Garante área útil mesmo em telas baixas; o overflow é da lista. */
		min-height: 160px;
	}

	.count-mobile {
		font-weight: var(--weight-medium);
	}

	.mobile-actions {
		position: fixed;
		left: 50%;
		bottom: calc(var(--bottom-nav-height) + var(--space-4));
		z-index: 45;
		transform: translateX(-50%);
		display: flex;
		align-items: center;
		gap: var(--space-1);
		padding: var(--space-2);
		background: var(--color-surface);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-md);
		box-shadow: var(--shadow-md);
	}

	.selection-count {
		padding: 0 var(--space-2);
		font-size: var(--text-caption);
		color: var(--color-text-muted);
		white-space: nowrap;
	}

	.list-head {
		flex-shrink: 0;
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: var(--space-3);
		margin-bottom: var(--space-4);
	}

	.list-actions {
		display: flex;
		align-items: center;
		gap: var(--space-2);
		flex-shrink: 0;
	}

	.mobile-filter-actions,
	.inline-filter-actions {
		display: flex;
		align-items: center;
		gap: var(--space-2);
		flex-shrink: 0;
	}

	.inline-filter-actions {
		margin-left: auto;
		align-self: end;
	}

	.filters-inline {
		flex-shrink: 0;
		display: none;
		gap: var(--space-3);
		align-items: end;
		margin-bottom: var(--space-4);
		flex-wrap: wrap;
	}

	.filters-inline > :global(.field) {
		min-width: 10rem;
		flex: 1;
	}

	.filter-sheet {
		display: flex;
		flex-direction: column;
		gap: var(--space-3);
	}

	/* Ações ocupam a largura toda no card, mantendo os botões em uma linha. */
	:global(.record .field-actions) {
		grid-column: 1 / -1;
		grid-template-columns: 1fr;
		gap: var(--space-2);
	}

	.full-link {
		display: inline-flex;
		align-items: center;
		min-height: var(--touch-target);
		color: var(--color-primary);
		font-weight: var(--weight-semibold);
		font-size: var(--text-caption);
	}

	@media (min-width: 1024px) {
		.count-mobile {
			display: none;
		}

		.filters-inline {
			display: flex;
		}

		:global(.btn.filters-trigger) {
			display: none;
		}

		.mobile-filter-actions {
			display: none;
		}
	}
</style>
