<script lang="ts">
	import { t } from '#lib/i18n/t.js';
	import { errorText } from '#lib/i18n/errors.js';
	import { onMount } from 'svelte';
	import { toast } from '#lib/toast.svelte.ts';
	import { currentLocale, formatDate } from '#lib/i18n/format.js';
	import PageHeader from '#lib/ui/PageHeader.svelte';
	import Card from '#lib/ui/Card.svelte';
	import Badge from '#lib/ui/Badge.svelte';
	import Alert from '#lib/ui/Alert.svelte';
	import EmptyState from '#lib/ui/EmptyState.svelte';
	import SegmentedControl from '#lib/ui/SegmentedControl.svelte';
	import MultiSelect from '#lib/ui/MultiSelect.svelte';
	import Input from '#lib/ui/Input.svelte';
	import Checkbox from '#lib/ui/Checkbox.svelte';
	import Button from '#lib/ui/Button.svelte';
	import IconButton from '#lib/ui/IconButton.svelte';
	import Icon from '#lib/ui/Icon.svelte';
	import Menu from '#lib/ui/Menu.svelte';
	import Dialog from '#lib/ui/Dialog.svelte';
	import Field from '#lib/ui/Field.svelte';
	import Pagination from '#lib/ui/Pagination.svelte';
	import PageFooter from '#lib/ui/PageFooter.svelte';
	import { viewport } from '#lib/ui/media.svelte.js';

	interface AppItem {
		id: number;
		job_id: number;
		status: string;
		sent_at: string | null;
		updated_at: string;
		has_tailored: boolean;
		job?: { title: string; company: string; url: string; score_fit: number | null };
	}

	const STATUSES = ['pending', 'sent', 'interview', 'rejected', 'offer'] as const;
	type Status = (typeof STATUSES)[number];
	function statusLabel(status: string): string {
		switch (status) {
			case 'pending': return t('applications.status.pending');
			case 'sent': return t('applications.status.sent');
			case 'interview': return t('applications.status.interview');
			case 'rejected': return t('applications.status.rejected');
			case 'offer': return t('applications.status.offer');
			default: return status;
		}
	}
	const STATUS_TONE: Record<Status, 'warning' | 'info' | 'primary' | 'error' | 'success'> = {
		pending: 'warning',
		sent: 'info',
		interview: 'primary',
		rejected: 'error',
		offer: 'success'
	};
	const statusOptions = $derived(STATUSES.map((value) => ({ value, label: statusLabel(value) })));

	let apps: AppItem[] = $state([]);
	let loading = $state(true);
	let error = $state('');
	let changingId = $state<number | null>(null);
	let detailApp: AppItem | null = $state(null);
	let draggingId = $state<number | null>(null);

	// Filtros client-side (mesmo padrão visual de /jobs).
	let statusFilter = $state<string[]>([]);
	let searchTerm = $state('');
	let onlyTailored = $state(false);
	let filtersOpen = $state(false);

	const PAGE_SIZE = 8;
	let currentPage = $state(1);
	let visibleCount = $state(8);
	let loadingMore = $state(false);


	const filteredApps = $derived(
		apps.filter((app) => {
			if (statusFilter.length > 0 && !statusFilter.includes(app.status)) return false;
			if (onlyTailored && !app.has_tailored) return false;
			const q = searchTerm.trim().toLocaleLowerCase(currentLocale());
			if (q && !`${app.job?.title ?? ''} ${app.job?.company ?? ''}`.toLocaleLowerCase(currentLocale()).includes(q)) return false;
			return true;
		})
	);
	const hasActiveFilters = $derived(statusFilter.length > 0 || searchTerm.trim() !== '' || onlyTailored);

	const totalPages = $derived(Math.max(1, Math.ceil(filteredApps.length / PAGE_SIZE)));
	const pagedApps = $derived(
		filteredApps.slice((currentPage - 1) * PAGE_SIZE, currentPage * PAGE_SIZE)
	);
	const rangeLabel = $derived(
		t('applications.range_full', {
			start: filteredApps.length === 0 ? 0 : (currentPage - 1) * PAGE_SIZE + 1,
			end: Math.min(currentPage * PAGE_SIZE, filteredApps.length),
			count: filteredApps.length
		})
	);

	// Desktop pagina; phone/tablet usa carregamento progressivo na rolagem.
	const visibleApps = $derived(
		viewport.isDesktop ? pagedApps : filteredApps.slice(0, visibleCount)
	);

	function loadMoreApps(): void {
		if (viewport.isDesktop || loadingMore || visibleCount >= filteredApps.length) return;
		loadingMore = true;
		setTimeout(() => {
			visibleCount += 8;
			loadingMore = false;
		}, 200);
	}

	function onListScroll(event: Event): void {
		const element = event.currentTarget as HTMLElement;
		if (element.scrollHeight - element.scrollTop - element.clientHeight < 80) loadMoreApps();
	}


	$effect(() => {
		if (currentPage > totalPages) currentPage = totalPages;
	});

	function goToPage(next: number): void {
		currentPage = Math.min(totalPages, Math.max(1, next));
	}

	onMount(async () => {
		try {
			const res = await fetch('/api/applications');
			if (res.ok) {
				const body = await res.json();
				apps = body.applications ?? [];
			} else {
				const err = await res.json().catch(() => ({}));
				error = errorText(err, t('applications.error.load_http', { status: res.status }));
			}
		} catch {
			error = t('applications.error.load_network');
		} finally {
			loading = false;
		}
	});

	async function changeStatus(app: AppItem, status: string): Promise<void> {
		if (status === app.status) return;
		changingId = app.id;
		try {
			const res = await fetch(`/api/applications/${app.id}`, {
				method: 'PATCH',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ status })
			});
			if (res.ok) {
				app.status = status;
				toast('success', t('applications.toast.changed', { title: app.job?.title ?? t('applications.toast.label'), label: statusLabel(status) }));
			} else {
				const err = await res.json().catch(() => ({}));
				toast('error', errorText(err, t('applications.error.status')));
			}
		} catch {
			toast('error', t('applications.error.status_network'));
		} finally {
			changingId = null;
		}
	}

	function dateFmt(iso: string): string {
		return formatDate(iso, { day: '2-digit', month: 'short', year: undefined });
	}

	function itemsFor(app: AppItem) {
		return STATUSES.filter((status) => status !== app.status).map((status) => ({
			label: t('applications.move_to', { label: statusLabel(status) }),
			icon: 'chevron-right',
			onSelect: () => changeStatus(app, status)
		}));
	}

	function onDragStart(event: DragEvent, app: AppItem): void {
		draggingId = app.id;
		event.dataTransfer?.setData('text/plain', String(app.id));
		if (event.dataTransfer) event.dataTransfer.effectAllowed = 'move';
	}

	function onDrop(event: DragEvent, status: string): void {
		event.preventDefault();
		const id = Number(event.dataTransfer?.getData('text/plain'));
		const app = apps.find((item) => item.id === id);
		if (app) void changeStatus(app, status);
		draggingId = null;
	}

	function clearFilters(): void {
		statusFilter = [];
		searchTerm = '';
		onlyTailored = false;
		currentPage = 1;
		visibleCount = 8;
	}
</script>

<svelte:head>
	<title>{t('common.applications')} · {t('common.app_name')}</title>
</svelte:head>

{#snippet statusMenu(app: AppItem)}
	<Menu items={itemsFor(app)} label={t('applications.change_status', { title: app.job?.title ?? app.id })} />
{/snippet}

{#snippet appCard(app: AppItem)}
	<Card
		class="app-card {draggingId === app.id ? 'dragging' : ''}"
		compact
		onclick={() => (detailApp = app)}
		draggable={viewport.isDesktop}
		ondragstart={(event) => onDragStart(event, app)}
		ondragend={() => (draggingId = null)}
	>
		{#snippet actions()}
			<Badge tone={STATUS_TONE[app.status as Status] ?? 'neutral'}>
				{statusLabel(app.status)}
			</Badge>
			{@render statusMenu(app)}
			<IconButton
				icon="external"
				label={t('applications.view_detail')}
				variant="ghost"
				size="sm"
				onclick={() => (detailApp = app)}
			/>
		{/snippet}
		<div class="card-content">
			<div class="headline">
				<strong class="job-title" title={app.job?.title ?? ''}>{app.job?.title ?? t('applications.job_label', { id: app.job_id })}</strong>
				<span class="company" title={app.job?.company ?? ''}>{app.job?.company ?? ''}</span>
			</div>
			<div class="meta-row">
				{#if app.job?.score_fit !== null && app.job?.score_fit !== undefined}
					<Badge tone="info">{t('common.score')} {app.job.score_fit}/100</Badge>
				{/if}
				{#if app.has_tailored}<Badge tone="success">{t('applications.resume_adapted')}</Badge>{/if}
				<span class="updated">{t('sources.col.updated')} {dateFmt(app.updated_at)}</span>
			</div>
			{#if app.job?.url}
				<a class="link" href={app.job.url} target="_blank" rel="noopener noreferrer">
					<Icon name="external" size={16} /> {t('applications.view_job')}
				</a>
			{/if}
		</div>
	</Card>
{/snippet}

{#snippet filterFields(prefix: string)}
	<Field label={t('common.status')}>
		<MultiSelect label={t('applications.filters.status_multi')} options={statusOptions} bind:value={statusFilter} allLabel={t('jobs.status_all')} onchange={() => (currentPage = 1)} />
	</Field>
	<Field label={t('applications.filters.search')} for="{prefix}-search">
		<input id="{prefix}-search" type="search" bind:value={searchTerm} oninput={() => (currentPage = 1)} placeholder={t('applications.filters.search_placeholder')} />
	</Field>
	<Field label={t('common.curriculum')}>
		<Checkbox label={t('applications.filters.only_adapted')} bind:checked={onlyTailored} />
	</Field>
{/snippet}

{#snippet filterIcons()}
	{#if hasActiveFilters}
		<IconButton icon="x" label={t('jobs.filters.clear')} variant="ghost" onclick={clearFilters} />
	{/if}
{/snippet}

<PageHeader
	title={t('common.applications')}
	subtitle={viewport.isDesktop
		? t('applications.subtitle_desktop')
		: t('applications.subtitle_mobile')}
	breadcrumb={[{ label: t('common.home'), href: '/' }, { label: t('common.applications') }]}
/>

{#if error}
	<Alert severity="error" title={t('applications.error.load')}>{error}</Alert>
{/if}

<div class="filters-mobile">
	<Button variant="ghost" onclick={() => (filtersOpen = true)} class="filters-trigger">
		<Icon name="sliders" size={20} /> {t('common.filters')}
	</Button>
	{@render filterIcons()}
</div>

<div class="filters-inline">
	{@render filterFields('inline')}
	<div class="inline-filter-actions">{@render filterIcons()}</div>
</div>

{#if loading}
	<Card><p class="hint">{t('applications.loading')}</p></Card>
{:else if apps.length === 0}
	<Card>
		<EmptyState title={t('applications.empty.title')} description={t('applications.empty.desc')}>
			<a class="cta" href="/jobs">{t('shortcut.jobs_desc')}</a>
		</EmptyState>
	</Card>
{:else if filteredApps.length === 0}
	<Card>
		<EmptyState title={t('applications.empty.filtered_title')} description={t('jobs.empty.filtered_desc')}>
			<Button variant="secondary" size="sm" onclick={clearFilters}>{t('jobs.filters.clear')}</Button>
		</EmptyState>
	</Card>
{:else if viewport.isDesktop}
	<div class="kanban" role="region" aria-label={t('applications.pipeline')}>
		{#each STATUSES as status (status)}
			<section
				class="column tone-{status}"
				class:drop-target={draggingId !== null}
				aria-label={statusLabel(status)}
				ondragover={(event) => event.preventDefault()}
				ondrop={(event) => onDrop(event, status)}
			>
				<header class="column-head">
					<span class="column-title"><span class="dot"></span>{statusLabel(status)}</span>
					<Badge>{pagedApps.filter((app) => app.status === status).length}</Badge>
				</header>
				<div class="column-body">
					{#each pagedApps.filter((app) => app.status === status) as app (app.id)}
						{@render appCard(app)}
					{:else}
						<p class="column-empty">{t('applications.drop_here')}</p>
					{/each}
				</div>
			</section>
		{/each}
	</div>
{:else}
	<p class="hint count-mobile">{t('applications.count', { count: filteredApps.length })}</p>
	<div class="apps-scroll" onscroll={onListScroll}>
		<div class="grid grid-2 app-grid">
			{#each visibleApps as app (app.id)}
				{@render appCard(app)}
			{/each}
		</div>
		{#if loadingMore}
			<div class="list-loading" role="status">{t('ui.loading_more')}</div>
		{/if}
	</div>
{/if}

{#snippet appCount()}
	<span class="hint">{rangeLabel}</span>
{/snippet}

{#if viewport.isDesktop}
	<PageFooter>
		<Pagination
			page={currentPage}
			totalPages={totalPages}
			onchange={goToPage}
			label={t('applications.pagination')}
			count={appCount}
		/>
	</PageFooter>
{/if}

<Dialog open={filtersOpen} onclose={() => (filtersOpen = false)} title={t('applications.filter_dialog')}>
	<div class="filter-sheet">
		{@render filterFields('sheet')}
	</div>
	{#snippet footer()}
		<Button variant="ghost" onclick={clearFilters}>{t('common.clear')}</Button>
		<Button onclick={() => (filtersOpen = false)}>{t('common.apply')}</Button>
	{/snippet}
</Dialog>

{#if detailApp}
	<Dialog open onclose={() => (detailApp = null)} title={detailApp.job?.title ?? t('applications.job_label', { id: detailApp.job_id })} description={detailApp.job?.company}>
		<div class="detail">
			<Badge tone={STATUS_TONE[detailApp.status as Status] ?? 'neutral'}>
				{statusLabel(detailApp.status)}
			</Badge>
			{#if detailApp.job?.score_fit !== null && detailApp.job?.score_fit !== undefined}
				<p class="muted">{t('applications.score_fit', { score: detailApp.job.score_fit })}</p>
			{/if}
			{#if detailApp.job?.url}
				<a class="link" href={detailApp.job.url} target="_blank" rel="noopener noreferrer">
					<Icon name="external" size={16} /> {t('applications.view_job')}
				</a>
			{/if}
			<SegmentedControl
				options={statusOptions}
				value={detailApp.status}
				label={t('applications.change_status_dialog')}
				onchange={(status) => changeStatus(detailApp!, status)}
			/>
			<p class="muted">{t('applications.updated_at', { date: dateFmt(detailApp.updated_at) })}</p>
		</div>
	</Dialog>
{/if}

<style>
	:global(.card.app-card) {
		min-width: 0;
		overflow: hidden;
		height: auto;
	}

	:global(.app-card .menu .trigger) {
		width: 36px;
		height: 36px;
		min-height: 36px;
	}

	:global(.app-card.dragging) {
		opacity: 0.5;
	}

	.card-content {
		display: flex;
		flex-direction: column;
		gap: var(--space-2);
		min-width: 0;
	}

	.headline {
		display: flex;
		flex-direction: column;
		gap: var(--space-1);
		min-width: 0;
	}

	.job-title,
	.company {
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.job-title {
		font-size: var(--text-body);
	}

	.company {
		font-size: var(--text-caption);
		color: var(--color-text-muted);
	}

	.meta-row {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: var(--space-2);
		min-width: 0;
	}

	.updated {
		font-size: var(--text-caption);
		color: var(--color-text-muted);
	}

	.muted {
		margin: 0;
		font-size: var(--text-caption);
		color: var(--color-text-muted);
	}

	.cta {
		display: inline-flex;
		align-items: center;
		min-height: var(--touch-target);
		font-weight: var(--weight-semibold);
	}

	.count-mobile {
		font-weight: var(--weight-medium);
	}

	.apps-scroll {
		flex: 1 1 auto;
		min-height: 0;
		overflow-y: auto;
		-webkit-overflow-scrolling: touch;
	}

	.list-loading {
		padding: var(--space-4);
		text-align: center;
		font-size: var(--text-caption);
		color: var(--color-text-muted);
	}

	.filters-mobile,
	.filters-inline {
		display: flex;
		gap: var(--space-2);
		align-items: center;
		margin-bottom: var(--space-4);
	}

	.filters-inline {
		display: none;
		align-items: end;
		gap: var(--space-3);
	}

	.filters-inline > :global(.field) {
		flex: 1;
		min-width: 12rem;
	}

	.inline-filter-actions {
		display: flex;
		gap: var(--space-2);
		margin-left: auto;
	}

	.filter-sheet {
		display: flex;
		flex-direction: column;
		gap: var(--space-4);
	}

	.kanban {
		display: grid;
		grid-template-columns: repeat(5, minmax(200px, 1fr));
		gap: var(--space-3);
		align-items: start;
		overflow-x: auto;
		padding-bottom: var(--space-2);
	}

	.column {
		background: var(--color-surface-soft);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-lg);
		padding: var(--space-3);
		min-width: 0;
	}

	.column.drop-target {
		border-color: var(--color-primary);
	}

	.column-head {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: var(--space-2);
		margin-bottom: var(--space-3);
	}

	.column-title {
		display: inline-flex;
		align-items: center;
		gap: var(--space-2);
		font-size: var(--text-caption);
		font-weight: var(--weight-semibold);
		text-transform: uppercase;
		letter-spacing: 0.04em;
		color: var(--color-text-secondary);
	}

	.dot {
		width: 8px;
		height: 8px;
		border-radius: var(--radius-pill);
		background: var(--color-text-muted);
	}

	.tone-pending .dot { background: var(--color-warning); }
	.tone-sent .dot { background: var(--color-info); }
	.tone-interview .dot { background: var(--color-primary); }
	.tone-rejected .dot { background: var(--color-error); }
	.tone-offer .dot { background: var(--color-success); }

	.column-body {
		display: flex;
		flex-direction: column;
		gap: var(--space-3);
		max-height: 60dvh;
		overflow-y: auto;
	}

	.column-empty {
		margin: var(--space-4) 0;
		text-align: center;
		font-size: var(--text-caption);
		color: var(--color-text-muted);
	}

	.detail {
		display: flex;
		flex-direction: column;
		align-items: flex-start;
		gap: var(--space-4);
	}

	@media (max-width: 767px) {
		/* Cartões curtos: o link e a data ficam no detalhe. */
		.updated,
		.link {
			display: none;
		}
	}

	@media (min-width: 1024px) {
		.count-mobile {
			display: none;
		}

		.filters-mobile {
			display: none;
		}

		.filters-inline {
			display: flex;
		}
	}
</style>
