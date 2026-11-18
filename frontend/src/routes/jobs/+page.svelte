<script lang="ts">
	import { t } from '#lib/i18n/t.js';
	import { errorText } from '#lib/i18n/errors.js';
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { toast } from '#lib/toast.svelte.ts';
	import { currentLocale, formatDate } from '#lib/i18n/format.js';

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
	import { SOURCE_LABELS, statusLabel, scoreTone, type Job } from '#lib/types/job.js';

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

	const statusOptions = $derived([
		{ value: 'all', label: t('jobs.status_all') },
		{ value: 'new', label: t('jobs.status_new') },
		{ value: 'approved', label: t('jobs.status_approved') },
		{ value: 'discarded', label: t('jobs.status_discarded') }
	]);

	const filteredJobs = $derived(
		jobs.filter((job) => {
			if (statusFilter !== 'all' && job.status !== statusFilter) return false;
			if (sourceFilter.length > 0 && !sourceFilter.includes(job.source)) return false;
			const q = searchTerm.trim().toLocaleLowerCase(currentLocale());
			if (q && !`${job.title} ${job.company} ${job.description}`.toLocaleLowerCase(currentLocale()).includes(q)) return false;
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
			return String(av).localeCompare(String(bv), currentLocale()) * factor;
		});
	});

	const totalPages = $derived(Math.max(1, Math.ceil(sortedJobs.length / PAGE_SIZE)));
	const pagedJobs = $derived(
		sortedJobs.slice((currentPage - 1) * PAGE_SIZE, currentPage * PAGE_SIZE)
	);
	const rangeLabel = $derived(
		t('jobs.range', {
			start: sortedJobs.length === 0 ? 0 : (currentPage - 1) * PAGE_SIZE + 1,
			end: Math.min(currentPage * PAGE_SIZE, sortedJobs.length),
			total: sortedJobs.length
		})
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

	const columns = $derived<DataColumn<Job>[]>([
		{ key: 'title', header: t('jobs.col.job'), cardLabel: t('jobs.col.job'), value: (job) => job.title },
		{ key: 'company', header: t('jobs.col.company'), value: (job) => job.company },
		{ key: 'source', header: t('common.source'), value: (job) => SOURCE_LABELS[job.source] ?? job.source },
		{ key: 'score', header: t('common.score'), value: (job) => (job.score_fit === null ? t('jobs.no_score') : String(job.score_fit)) },
		{ key: 'status', header: t('common.status'), value: (job) => statusLabel(job.status) },
		{
			key: 'created_at',
			header: t('common.date'),
			value: (job) => formatDate(job.created_at)
		},
	]);

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
					status = { kind: 'error', text: t('jobs.flash.duplicate_url') };
				} else if (!body.description) {
					status = { kind: 'error', text: t('jobs.flash.no_description') };
				} else {
					status = { kind: 'success', text: t('jobs.flash.filled_from_url') };
				}
			} else {
				status = { kind: 'error', text: errorText(body, t('jobs.error.fetch_http', { status: res.status })) };
			}
		} catch {
			status = { kind: 'error', text: t('jobs.error.fetch_network') };
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
				status = { kind: 'error', text: t('jobs.flash.search_running') };
			} else if (!res.ok) {
				const err = await res.json().catch(() => ({}));
				status = { kind: 'error', text: errorText(err, t('jobs.flash.search_start_failed')) };
				discovering = false;
				return;
			} else {
				const queued = await res.json().catch(() => ({}));
				if (queued.status === 'queued' && queued.queue_position > 1) {
					status = {
						kind: 'success',
						text: t('jobs.flash.search_queued', { position: queued.queue_position })
					};
				}
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
								text: t('jobs.flash.search_done', { fetched: stats.fetched, created: stats.new_jobs, scored: stats.scored })
							};
						}
						break;
					}
				}
			}
			await loadJobs();
		} catch {
			status = { kind: 'error', text: t('jobs.error.search_network') };
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
				status = { kind: 'error', text: errorText(err, t('jobs.error.load_http', { status: res.status })) };
			}
		} catch {
			status = { kind: 'error', text: t('jobs.error.load_network') };
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
				status = { kind: 'success', text: t('jobs.flash.saved') };
				await loadJobs();
			} else {
				status = { kind: 'error', text: errorText(body, t('jobs.error.save_http', { status: res.status })) };
			}
		} catch {
			status = { kind: 'error', text: t('jobs.error.save_network') };
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
				status = { kind: 'success', text: t('jobs.flash.scored', { score: body.score_fit }) };
			} else {
				status = { kind: 'error', text: errorText(body, t('jobs.error.score_http', { status: res.status })) };
			}
		} catch {
			status = { kind: 'error', text: t('jobs.error.score_network') };
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
				? t('jobs.flash.approved', { title: job.title })
				: errorText(body, t('jobs.error.approve_http', { status: res.status })));
		} catch {
			toast('error', t('jobs.error.approve_network'));
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
				status = { kind: 'success', text: t('jobs.flash.discarded') };
				detailJob = null;
				await loadJobs();
			} else {
				status = { kind: 'error', text: errorText(body, t('jobs.error.discard_http', { status: res.status })) };
			}
		} catch {
			status = { kind: 'error', text: t('jobs.error.discard_network') };
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
		const verb = t(`jobs.flash.batch_verb_${action}`);
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
			t('jobs.flash.batch', { verb, ok, total: targets.length, failures }));
		if (action !== 'approve') {
			if (action === 'discard') clearSelection();
			await loadJobs();
		}
	}
</script>

<svelte:head>
	<title>{t('jobs.meta_title')}</title>
</svelte:head>


{#snippet renderCell(job: Job, column: DataColumn<Job>)}
	{#if column.key === 'title'}
		<strong>{job.title}</strong>
	{:else if column.key === 'score'}
		<Badge tone={job.score_fit === null ? 'neutral' : scoreTone(job.score_fit)}>
			{job.score_fit === null ? t('jobs.no_score') : `${job.score_fit}/100`}
		</Badge>
	{:else if column.key === 'status'}
		<Badge tone={job.status === 'approved' ? 'success' : job.status === 'discarded' ? 'error' : 'primary'}>
			{statusLabel(job.status)}
		</Badge>
	{:else}
		{column.value?.(job) ?? ''}
	{/if}
{/snippet}

{#snippet newJobForm()}
	<form id="new-job-form" onsubmit={(e) => { e.preventDefault(); save(); }}>
		<Field label={t('jobs.form.url')} for="url">
			<div class="url-row">
				<input
					id="url"
					type="url"
					placeholder={t('jobs.form.url_placeholder')}
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
					{t('jobs.form.fill')}
				</Button>
			</div>
		</Field>
		<Field label={t('jobs.form.title')} for="title">
			<input id="title" bind:value={title} placeholder={t('jobs.form.title_placeholder')} />
		</Field>
		<Field label={t('jobs.col.company')} for="company">
			<input id="company" bind:value={company} placeholder={t('jobs.form.company_placeholder')} />
		</Field>
		<Field label={t('jobs.form.jd')} for="description">
			<Textarea id="description" rows={6} bind:value={description} placeholder={t('jobs.form.jd_placeholder')} />
		</Field>
	</form>
{/snippet}

{#snippet filterIcons()}
	{#if hasActiveFilters}
		<IconButton icon="x" label={t('jobs.filters.clear')} variant="ghost" onclick={clearFilters} />
	{/if}
	<IconButton icon="check-double" label={t('jobs.filters.select_all')} variant="ghost" onclick={selectAllFiltered} />
	<IconLinkButton href="/settings/sources" icon="settings" label={t('jobs.filters.configure_sources')} />
{/snippet}

{#snippet filterFields(prefix: string)}
	<Field label={t('common.status')} for="{prefix}-status">
		<Select id="{prefix}-status" value={statusFilter} onchange={(e) => { statusFilter = e.currentTarget.value; resetPage(); }} options={statusOptions} />
	</Field>
	<Field label={t('jobs.filters.source_multi')}>
		<MultiSelect label={t('jobs.filters.source_button')} options={sourceOptions} bind:value={sourceFilter} allLabel={t('jobs.filters.all_sources')} onchange={resetPage} />
	</Field>
	<Field label={t('jobs.filters.search')} for="{prefix}-search">
		<input id="{prefix}-search" type="search" bind:value={searchTerm} oninput={resetPage} placeholder={t('jobs.filters.search_placeholder')} />
	</Field>
{/snippet}

<PageHeader
	title={t('common.jobs')}
	subtitle={t('jobs.subtitle')}
	breadcrumb={[{ label: t('common.home'), href: '/' }, { label: t('common.jobs') }]}
>
	{#snippet actions()}
		<Button onclick={triggerDiscovery} loading={discovering}>
			<Icon name="sparkles" size={20} />
			{discovering ? t('jobs.searching') : t('jobs.search_now')}
		</Button>
		<Button variant="secondary" onclick={() => (newJobOpen = true)}>
			<Icon name="plus" size={20} />
			{t('jobs.new')}
		</Button>
	{/snippet}
</PageHeader>

{#if status}
	<Alert severity={status.kind} title={status.kind === 'error' ? t('common.attention') : t('common.success')}>{status.text}</Alert>
{/if}

{#if !llmConfigured}
	<Alert severity="warning" title={t('jobs.provider_warning_title')}>
		O botão {t('common.score')} precisa da sua chave. <a href="/settings">Configurar</a>
	</Alert>
{/if}

<Card class="keywords-strip">
	<div class="keywords-strip-inner">
		<span class="keywords-icon"><Icon name="sliders" size={20} /></span>
		<div class="keywords-copy">
			<strong>{t('jobs.keywords_label')}</strong>
			<span class="hint">
				{discoveryKeywords.length > 0
					? discoveryKeywords.join(' · ')
					: t('jobs.keywords_empty')}
			</span>
		</div>
		<IconLinkButton href="/settings/discovery" icon="external" label={t('jobs.configure_search')} />
	</div>
</Card>


<section aria-label={t('jobs.list.title')} class="jobs-section">
	<div class="list-head">
		<p class="hint count-mobile">{t('jobs.list.count', { count: sortedJobs.length })}</p>
		<div class="list-actions">
			<Button variant="ghost" onclick={() => (filtersOpen = true)} class="filters-trigger">
				<Icon name="sliders" size={20} /> {t('common.filters')}
			</Button>
			<div class="mobile-filter-actions">{@render filterIcons()}</div>
		</div>
	</div>

	<div class="filters-inline">
		{@render filterFields('inline')}
		<div class="inline-filter-actions">{@render filterIcons()}</div>
	</div>

	{#if loading}
		<Card><p class="hint">{t('jobs.list.loading')}</p></Card>
	{:else if jobs.length === 0}
		<Card>
			<EmptyState
				title={t('jobs.empty.title')}
				description={t('jobs.empty.desc')}
			>
				<Button variant="secondary" size="sm" onclick={() => (newJobOpen = true)}>{t('jobs.new')}</Button>
			</EmptyState>
		</Card>
	{:else if filteredJobs.length === 0}
		<Card>
			<EmptyState title={t('jobs.empty.filtered_title')} description={t('jobs.empty.filtered_desc')}>
				<Button variant="secondary" size="sm" onclick={clearFilters}>{t('jobs.filters.clear')}</Button>
			</EmptyState>
		</Card>
	{:else}
		<DataView
			columns={columns}
			rows={visibleJobs}
			getKey={(job) => job.id}
			caption={t('jobs.list.title')}
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
		label={t('jobs.batch.approve')}
		variant="primary"
		size={size}
		disabled={selectedIds.length === 0}
		onclick={() => runBatch('approve')}
	/>
	<IconButton
		icon="sparkles"
		label={t('jobs.batch.score')}
		variant="secondary"
		size={size}
		disabled={selectedIds.length === 0}
		onclick={() => runBatch('score')}
	/>
{/snippet}

{#snippet barActions(size: 'sm' | 'md')}
	<IconButton
		icon="trash"
		label={t('jobs.batch.discard')}
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
			label={t('jobs.batch.pagination')}
			leading={barLeading}
			actions={barActions}
			count={barCount}
		/>
	</PageFooter>
{:else if selectedIds.length > 0}
	<!-- No mobile as ações em lote flutuam sobre a lista quando há seleção. -->
	<div class="mobile-actions" role="toolbar" aria-label={t('jobs.batch.actions')}>
		<span class="selection-count">{t('jobs.batch.selected', { count: selectedIds.length })}</span>
		<IconButton icon="check" label={t('jobs.batch.approve')} variant="primary" onclick={() => runBatch('approve')} />
		<IconButton icon="sparkles" label={t('jobs.batch.score')} variant="secondary" onclick={() => runBatch('score')} />
		<IconButton icon="trash" label={t('jobs.batch.discard')} variant="danger" onclick={() => runBatch('discard')} />
	</div>
{/if}

<Dialog open={newJobOpen} onclose={() => (newJobOpen = false)} title={t('jobs.new')} description={t('jobs.dialog.new_desc')}>
	{@render newJobForm()}
	{#snippet footer()}
		<Button variant="ghost" onclick={() => (newJobOpen = false)}>{t('common.cancel')}</Button>
		<Button type="submit" form="new-job-form" loading={saving} disabled={!canSave}>{t('jobs.dialog.save')}</Button>
	{/snippet}
</Dialog>

<Dialog open={filtersOpen} onclose={() => (filtersOpen = false)} title={t('jobs.dialog.filter_title')}>
	<div class="filter-sheet">
		{@render filterFields('sheet')}
	</div>
	{#snippet footer()}
		<Button variant="ghost" onclick={clearFilters}>{t('common.clear')}</Button>
		<Button onclick={() => (filtersOpen = false)}>{t('common.apply')}</Button>
	{/snippet}
</Dialog>

{#if detailJob}
	<Drawer
		title={t('jobs.dialog.detail_title')}
		description={detailJob.company}
		onclose={() => (detailJob = null)}
	>
		<JobDetail job={detailJob} />
		{#snippet footer()}
			<a class="full-link" href={`/jobs/${detailJob!.id}`}>{t('jobs.dialog.open_full')}</a>
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
