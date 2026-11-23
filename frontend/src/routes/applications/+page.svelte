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
	import MultiSelect from '#lib/ui/MultiSelect.svelte';
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

	interface TagItem {
		id: number;
		namespace: string;
		value: string;
		label: string;
	}

	interface AppItem {
		id: number;
		job_id: number;
		status: string;
		stage_group?: string;
		stage_label?: string | null;
		stage_source?: string;
		stage_updated_at?: string | null;
		tags?: TagItem[];
		sent_at: string | null;
		updated_at: string;
		has_tailored: boolean;
		job?: { title: string; company: string; url: string; score_fit: number | null };
	}

	interface EmailMessageItem {
		id: string;
		subject: string;
		from_name: string;
		from_address: string;
		received_at: string;
		matched_application_id: number | null;
		match_confidence: number;
		stage_signal: string;
		link_source: string;
	}

	const STAGES = ['pending', 'in_process', 'interview', 'offer', 'rejected', 'closed'] as const;
	type Stage = (typeof STAGES)[number];

	function getStageGroup(app: AppItem): Stage {
		if (app.stage_group && STAGES.includes(app.stage_group as Stage)) {
			return app.stage_group as Stage;
		}
		if (app.status === 'sent') return 'in_process';
		if (app.status && STAGES.includes(app.status as Stage)) {
			return app.status as Stage;
		}
		return 'pending';
	}

	function stageLabel(stage: string): string {
		switch (stage) {
			case 'pending': return t('applications.status.pending');
			case 'in_process': return t('applications.stage.in_process');
			case 'sent': return t('applications.stage.in_process');
			case 'interview': return t('applications.status.interview');
			case 'rejected': return t('applications.status.rejected');
			case 'offer': return t('applications.status.offer');
			case 'closed': return t('applications.stage.closed');
			default: return stage;
		}
	}

	const STAGE_TONE: Record<string, 'warning' | 'info' | 'primary' | 'error' | 'success' | 'neutral'> = {
		pending: 'warning',
		in_process: 'info',
		sent: 'info',
		interview: 'primary',
		offer: 'success',
		rejected: 'error',
		closed: 'neutral'
	};

	const stageOptions = $derived(STAGES.map((value) => ({ value, label: stageLabel(value) })));

	let apps: AppItem[] = $state([]);
	let loading = $state(true);
	let error = $state('');
	let changingId = $state<number | null>(null);
	let detailApp: AppItem | null = $state(null);
	let draggingId = $state<number | null>(null);

	// Sugestões de avanço por e-mail (baixa confiança)
	let suggestions: EmailMessageItem[] = $state([]);
	let reviewSuggestionsOpen = $state(false);

	// E-mails vinculados à candidatura aberta
	let linkedEmails: EmailMessageItem[] = $state([]);
	let loadingLinked = $state(false);

	// Filtros client-side
	let statusFilter = $state<string[]>([]);
	let tagFilter = $state<string[]>([]);
	let searchTerm = $state('');
	let onlyTailored = $state(false);
	let filtersOpen = $state(false);

	// Tags de fase (CP17)
	let allTags: TagItem[] = $state([]);
	let selectedTagValues: string[] = $state([]);
	let newTagLabel = $state('');
	let savingTags = $state(false);

	const tagOptions = $derived(allTags.map((tag) => ({ value: String(tag.id), label: tag.label })));

	const PAGE_SIZE = 8;
	let currentPage = $state(1);
	let visibleCount = $state(8);
	let loadingMore = $state(false);

	const filteredApps = $derived(
		apps.filter((app) => {
			const sg = getStageGroup(app);
			if (statusFilter.length > 0 && !statusFilter.includes(sg) && !statusFilter.includes(app.status)) return false;
			if (tagFilter.length > 0) {
				const ids = tagFilter.map(Number);
				if (!app.tags?.some((tag) => ids.includes(tag.id))) return false;
			}
			if (onlyTailored && !app.has_tailored) return false;
			const q = searchTerm.trim().toLocaleLowerCase(currentLocale());
			if (q && !`${app.job?.title ?? ''} ${app.job?.company ?? ''}`.toLocaleLowerCase(currentLocale()).includes(q)) return false;
			return true;
		})
	);
	const hasActiveFilters = $derived(statusFilter.length > 0 || tagFilter.length > 0 || searchTerm.trim() !== '' || onlyTailored);

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

	async function loadApplications() {
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
	}

	async function loadSuggestions() {
		try {
			const res = await fetch('/api/email/messages?suggested=true');
			if (res.ok) {
				const body = await res.json();
				suggestions = body.messages ?? [];
			}
		} catch {
			// e-mail não configurado ou sem rota
		}
	}

	async function loadTags() {
		try {
			const res = await fetch('/api/tags?namespace=phase');
			if (res.ok) {
				const body = await res.json();
				allTags = body.tags ?? [];
			}
		} catch {
			// tags opcionais
		}
	}

	onMount(async () => {
		await loadApplications();
		await Promise.all([loadSuggestions(), loadTags()]);
	});

	async function changeStage(app: AppItem, targetStage: string): Promise<void> {
		if (targetStage === getStageGroup(app)) return;
		changingId = app.id;
		try {
			const res = await fetch(`/api/applications/${app.id}`, {
				method: 'PATCH',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({
					stage_group: targetStage,
					status: targetStage
				})
			});
			if (res.ok) {
				const body = await res.json().catch(() => ({}));
				app.status = targetStage;
				app.stage_group = targetStage;
				if (body.stage_label !== undefined) app.stage_label = body.stage_label;
				if (body.tags) app.tags = body.tags;
				toast('success', t('applications.toast.changed', { title: app.job?.title ?? t('applications.toast.label'), label: stageLabel(targetStage) }));
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

	async function confirmAdvanceSuggestion(msg: EmailMessageItem) {
		if (!msg.matched_application_id) return;
		try {
			const res = await fetch(`/api/email/messages/${msg.id}/link`, {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({
					application_id: msg.matched_application_id,
					advance_stage: true
				})
			});
			if (res.ok) {
				suggestions = suggestions.filter((s) => s.id !== msg.id);
				toast('success', t('applications.confirm_advance'));
				await loadApplications();
			}
		} catch {
			toast('error', 'Falha ao confirmar avanço');
		}
	}

	async function dismissSuggestion(msg: EmailMessageItem) {
		try {
			const res = await fetch(`/api/email/messages/${msg.id}/unlink`, { method: 'POST' });
			if (res.ok) {
				suggestions = suggestions.filter((s) => s.id !== msg.id);
			}
		} catch {
			// ignore
		}
	}

	async function loadEmailsForApp(appId: number) {
		loadingLinked = true;
		try {
			const res = await fetch(`/api/email/messages?application_id=${appId}`);
			if (res.ok) {
				const body = await res.json();
				linkedEmails = body.messages ?? [];
			}
		} catch {
			linkedEmails = [];
		} finally {
			loadingLinked = false;
		}
	}

	async function unlinkMessageFromApp(msgId: string) {
		try {
			const res = await fetch(`/api/email/messages/${msgId}/unlink`, { method: 'POST' });
			if (res.ok) {
				linkedEmails = linkedEmails.filter((m) => m.id !== msgId);
				toast('success', 'E-mail desvinculado');
			}
		} catch {
			toast('error', 'Falha ao desvincular');
		}
	}

	function slugifyTag(label: string): string {
		return label
			.toLocaleLowerCase(currentLocale())
			.normalize('NFD')
			.replace(/[\u0300-\u036f]/g, '')
			.replace(/[^a-z0-9]+/g, '_')
			.replace(/^_+|_+$/g, '');
	}

	async function createPhaseTag(): Promise<void> {
		const label = newTagLabel.trim();
		if (!label) return;
		try {
			const res = await fetch('/api/tags', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ namespace: 'phase', value: slugifyTag(label), label })
			});
			if (!res.ok) {
				toast('error', t('applications.tags_create_error'));
				return;
			}
			const tag: TagItem | null = await res.json().catch(() => null);
			if (tag?.id) {
				if (!allTags.some((item) => item.id === tag.id)) allTags = [...allTags, tag];
				if (!selectedTagValues.includes(String(tag.id))) {
					selectedTagValues = [...selectedTagValues, String(tag.id)];
				}
			} else {
				await loadTags();
			}
			newTagLabel = '';
		} catch {
			toast('error', t('applications.tags_create_error'));
		}
	}

	function toggleTag(tag: TagItem): void {
		const value = String(tag.id);
		selectedTagValues = selectedTagValues.includes(value)
			? selectedTagValues.filter((item) => item !== value)
			: [...selectedTagValues, value];
	}

	async function saveTags(): Promise<void> {
		if (!detailApp) return;
		savingTags = true;
		try {
			const res = await fetch(`/api/applications/${detailApp.id}`, {
				method: 'PATCH',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({
					stage_group: getStageGroup(detailApp),
					stage_label: detailApp.stage_label ?? '',
					tag_ids: selectedTagValues.map(Number)
				})
			});
			if (!res.ok) {
				const err = await res.json().catch(() => ({}));
				toast('error', errorText(err, t('applications.tags_save_error')));
				return;
			}
			const body = await res.json().catch(() => ({}));
			const tags: TagItem[] = body.tags ?? allTags.filter((tag) => selectedTagValues.includes(String(tag.id)));
			detailApp.tags = tags;
			const index = apps.findIndex((app) => app.id === detailApp?.id);
			if (index >= 0) apps[index].tags = tags;
			toast('success', t('applications.tags_saved'));
		} catch {
			toast('error', t('applications.tags_save_error'));
		} finally {
			savingTags = false;
		}
	}

	$effect(() => {
		if (detailApp) {
			void loadEmailsForApp(detailApp.id);
			selectedTagValues = (detailApp.tags ?? []).map((tag) => String(tag.id));
			newTagLabel = '';
		} else {
			linkedEmails = [];
			selectedTagValues = [];
			newTagLabel = '';
		}
	});

	function dateFmt(iso: string): string {
		return formatDate(iso, { day: '2-digit', month: 'short', year: undefined });
	}

	function signalLabel(signal: string): string {
		switch (signal) {
			case 'ack': return t('applications.signal.ack');
			case 'screening': return t('applications.signal.screening');
			case 'interview': return t('applications.signal.interview');
			case 'assessment': return t('applications.signal.assessment');
			case 'offer': return t('applications.signal.offer');
			case 'rejection': return t('applications.signal.rejection');
			case 'info_request': return t('applications.signal.info_request');
			default: return t('applications.signal.other');
		}
	}

	function itemsFor(app: AppItem) {
		const current = getStageGroup(app);
		return STAGES.filter((s) => s !== current).map((s) => ({
			label: t('applications.move_to', { label: stageLabel(s) }),
			icon: 'chevron-right',
			onSelect: () => changeStage(app, s)
		}));
	}

	function onDragStart(event: DragEvent, app: AppItem): void {
		draggingId = app.id;
		event.dataTransfer?.setData('text/plain', String(app.id));
		if (event.dataTransfer) event.dataTransfer.effectAllowed = 'move';
	}

	function onDrop(event: DragEvent, stage: string): void {
		event.preventDefault();
		const id = Number(event.dataTransfer?.getData('text/plain'));
		const app = apps.find((item) => item.id === id);
		if (app) void changeStage(app, stage);
		draggingId = null;
	}

	function clearFilters(): void {
		statusFilter = [];
		tagFilter = [];
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
	{@const currentStage = getStageGroup(app)}
	<Card
		class="app-card {draggingId === app.id ? 'dragging' : ''}"
		compact
		onclick={() => (detailApp = app)}
		draggable={viewport.isDesktop}
		ondragstart={(event) => onDragStart(event, app)}
		ondragend={() => (draggingId = null)}
	>
		{#snippet actions()}
			<Badge tone={STAGE_TONE[currentStage] ?? 'neutral'}>
				{app.stage_label || stageLabel(currentStage)}
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

			{#if app.tags && app.tags.length > 0}
				<div class="tags-row">
					{#each app.tags as tag (tag.id)}
						<span class="tag-chip tag-{tag.namespace}">{tag.label}</span>
					{/each}
				</div>
			{/if}

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
		<MultiSelect label={t('applications.filters.status_multi')} options={stageOptions} bind:value={statusFilter} allLabel={t('jobs.status_all')} onchange={() => (currentPage = 1)} />
	</Field>
	<Field label={t('applications.filters.search')} for="{prefix}-search">
		<input id="{prefix}-search" type="search" bind:value={searchTerm} oninput={() => (currentPage = 1)} placeholder={t('applications.filters.search_placeholder')} />
	</Field>
	{#if tagOptions.length > 0}
		<Field label={t('applications.filters.tags')}>
			<MultiSelect label={t('applications.filters.tags_multi')} options={tagOptions} bind:value={tagFilter} allLabel={t('applications.filters.all_tags')} onchange={() => (currentPage = 1)} />
		</Field>
	{/if}
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

{#if suggestions.length > 0}
	<aside class="suggestions-banner" role="status">
		<div class="banner-content">
			<Icon name="bell" size={20} />
			<span>{t('applications.suggestions_banner', { count: suggestions.length })}</span>
		</div>
		<Button size="sm" variant="secondary" onclick={() => (reviewSuggestionsOpen = true)}>
			{t('applications.review_suggestions')}
		</Button>
	</aside>
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
		{#each STAGES as stage (stage)}
			<section
				class="column tone-{stage}"
				class:drop-target={draggingId !== null}
				aria-label={stageLabel(stage)}
				ondragover={(event) => event.preventDefault()}
				ondrop={(event) => onDrop(event, stage)}
			>
				<header class="column-head">
					<span class="column-title"><span class="dot"></span>{stageLabel(stage)}</span>
					<Badge>{pagedApps.filter((app) => getStageGroup(app) === stage).length}</Badge>
				</header>
				<div class="column-body">
					{#each pagedApps.filter((app) => getStageGroup(app) === stage) as app (app.id)}
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

<!-- Dialog para revisar sugestões de e-mail -->
<Dialog open={reviewSuggestionsOpen} onclose={() => (reviewSuggestionsOpen = false)} title={t('applications.review_suggestions')}>
	<div class="suggestions-list">
		{#each suggestions as msg (msg.id)}
			{@const matchedApp = apps.find((a) => a.id === msg.matched_application_id)}
			<div class="suggestion-item">
				<div class="suggestion-info">
					<strong>{msg.subject || '(sem assunto)'}</strong>
					<span class="muted">{msg.from_name || msg.from_address} • {dateFmt(msg.received_at)}</span>
					{#if matchedApp}
						<span class="job-ref">Vaga: {matchedApp.job?.title} @ {matchedApp.job?.company}</span>
					{/if}
					<div class="signal-tag">
						<Badge tone="info">{signalLabel(msg.stage_signal)}</Badge>
						<span class="conf">Confiança: {Math.round(msg.match_confidence * 100)}%</span>
					</div>
				</div>
				<div class="suggestion-actions">
					<Button size="sm" variant="primary" onclick={() => confirmAdvanceSuggestion(msg)}>
						{t('applications.confirm_advance')}
					</Button>
					<Button size="sm" variant="ghost" onclick={() => dismissSuggestion(msg)}>
						{t('applications.dismiss_suggestion')}
					</Button>
				</div>
			</div>
		{:else}
			<p class="muted">Nenhuma sugestão pendente.</p>
		{/each}
	</div>
</Dialog>

{#if detailApp}
	<Dialog open onclose={() => (detailApp = null)} title={detailApp.job?.title ?? t('applications.job_label', { id: detailApp.job_id })} description={detailApp.job?.company}>
		<div class="detail">
			<Badge tone={STAGE_TONE[getStageGroup(detailApp)] ?? 'neutral'}>
				{detailApp.stage_label || stageLabel(getStageGroup(detailApp))}
			</Badge>
			{#if detailApp.job?.score_fit !== null && detailApp.job?.score_fit !== undefined}
				<p class="muted">{t('applications.score_fit', { score: detailApp.job.score_fit })}</p>
			{/if}
			<div class="tags-section">
				<strong class="section-title">{t('applications.tags')}</strong>
				{#if allTags.length > 0}
					<div class="tag-options">
						{#each allTags as tag (tag.id)}
							<label class="tag-option">
								<input
									type="checkbox"
									checked={selectedTagValues.includes(String(tag.id))}
									onchange={() => toggleTag(tag)}
								/>
								<span class="tag-chip tag-{tag.namespace}">{tag.label}</span>
							</label>
						{/each}
					</div>
				{:else}
					<p class="muted">{t('applications.tags_empty')}</p>
				{/if}
				<div class="tag-create">
					<input
						type="text"
						bind:value={newTagLabel}
						placeholder={t('applications.tags_new_placeholder')}
						aria-label={t('applications.tags_new_placeholder')}
					/>
					<Button size="sm" variant="secondary" onclick={createPhaseTag}>
						{t('applications.tags_create')}
					</Button>
					<Button size="sm" variant="primary" onclick={saveTags} disabled={savingTags}>
						{t('common.save')}
					</Button>
				</div>
			</div>

			<!-- E-mails vinculados -->
			<div class="linked-section">
				<strong class="section-title">{t('applications.linked_emails')}</strong>
				{#if loadingLinked}
					<p class="hint">Carregando e-mails...</p>
				{:else if linkedEmails.length > 0}
					<div class="emails-list">
						{#each linkedEmails as email (email.id)}
							<div class="email-item">
								<div class="email-details">
									<strong class="email-subject">{email.subject}</strong>
									<span class="email-meta">{email.from_name || email.from_address} • {dateFmt(email.received_at)}</span>
								</div>
								<div class="email-right">
									<Badge tone="info">{signalLabel(email.stage_signal)}</Badge>
									<IconButton
										icon="x"
										label={t('applications.unlink_email')}
										variant="ghost"
										size="sm"
										onclick={() => unlinkMessageFromApp(email.id)}
									/>
								</div>
							</div>
						{/each}
					</div>
				{:else}
					<p class="muted">{t('applications.no_linked_emails')}</p>
				{/if}
			</div>

			<Field label={t('applications.change_status_dialog')}>
				<select
					value={getStageGroup(detailApp)}
					onchange={(e) => {
						const val = (e.currentTarget as HTMLSelectElement).value;
						if (detailApp) void changeStage(detailApp, val);
					}}
				>
					{#each STAGES as s}
						<option value={s}>{stageLabel(s)}</option>
					{/each}
				</select>
			</Field>
		</div>
	</Dialog>
{/if}

<style>
	.suggestions-banner {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: var(--space-3);
		padding: var(--space-3) var(--space-4);
		background: var(--color-surface-elevated);
		border: 1px solid var(--color-primary);
		border-radius: var(--radius-md);
		margin-bottom: var(--space-4);
		color: var(--color-text-primary);
		font-size: var(--text-caption);
	}

	.banner-content {
		display: flex;
		align-items: center;
		gap: var(--space-2);
		color: var(--color-primary);
		font-weight: var(--weight-medium);
	}

	.suggestions-list {
		display: flex;
		flex-direction: column;
		gap: var(--space-3);
		max-height: 60dvh;
		overflow-y: auto;
	}

	.suggestion-item {
		display: flex;
		justify-content: space-between;
		align-items: center;
		gap: var(--space-3);
		padding: var(--space-3);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-md);
		background: var(--color-surface-soft);
	}

	.suggestion-info {
		display: flex;
		flex-direction: column;
		gap: var(--space-1);
		font-size: var(--text-caption);
	}

	.job-ref {
		font-size: var(--text-2xs, 0.75rem);
		color: var(--color-text-secondary);
	}

	.signal-tag {
		display: flex;
		align-items: center;
		gap: var(--space-2);
		margin-top: var(--space-1);
	}

	.conf {
		font-size: var(--text-2xs, 0.75rem);
		color: var(--color-text-muted);
	}

	.suggestion-actions {
		display: flex;
		gap: var(--space-2);
		flex-shrink: 0;
	}

	.tags-row {
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-1);
		margin-top: var(--space-1);
	}

	.tag-options {
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-2);
	}

	.tag-option {
		display: inline-flex;
		align-items: center;
		gap: var(--space-1);
		cursor: pointer;
	}

	.tag-create {
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-2);
		align-items: center;
	}

	.tag-create input {
		flex: 1;
		min-width: 10rem;
	}

	.tag-chip {
		display: inline-flex;
		align-items: center;
		padding: var(--space-05) var(--space-1-5);
		font-size: var(--text-2xs, 0.7rem);
		font-weight: var(--weight-medium);
		border-radius: var(--radius-pill);
		background: var(--color-surface-elevated);
		border: 1px solid var(--color-border);
		color: var(--color-text-secondary);
	}

	.tag-chip.tag-phase {
		border-color: var(--color-primary-soft, var(--color-border));
		color: var(--color-primary);
		background: var(--color-surface-soft);
	}

	.linked-section, .tags-section {
		display: flex;
		flex-direction: column;
		gap: var(--space-2);
		width: 100%;
		border-top: 1px solid var(--color-border);
		padding-top: var(--space-3);
	}

	.section-title {
		font-size: var(--text-caption);
		font-weight: var(--weight-semibold);
		color: var(--color-text-secondary);
	}

	.emails-list {
		display: flex;
		flex-direction: column;
		gap: var(--space-2);
	}

	.email-item {
		display: flex;
		justify-content: space-between;
		align-items: center;
		gap: var(--space-2);
		padding: var(--space-2) var(--space-3);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-sm);
		background: var(--color-surface-soft);
	}

	.email-details {
		display: flex;
		flex-direction: column;
		gap: var(--space-05, 2px);
		font-size: var(--text-caption);
	}

	.email-subject {
		font-size: var(--text-caption);
	}

	.email-meta {
		font-size: var(--text-2xs, 0.75rem);
		color: var(--color-text-muted);
	}

	.email-right {
		display: flex;
		align-items: center;
		gap: var(--space-2);
		flex-shrink: 0;
	}

	.filters-mobile {
		display: flex;
		align-items: center;
		gap: var(--space-2);
		margin-bottom: var(--space-4);
	}

	.filters-inline {
		display: none;
		align-items: flex-end;
		gap: var(--space-3);
		margin-bottom: var(--space-6);
		padding: var(--space-3);
		background: var(--color-surface-soft);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-lg);
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
		grid-template-columns: minmax(170px, 1fr) minmax(200px, 1.25fr) minmax(170px, 1fr) minmax(170px, 1fr) minmax(170px, 1fr) minmax(170px, 1fr);
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
	.tone-in_process .dot { background: var(--color-info); }
	.tone-sent .dot { background: var(--color-info); }
	.tone-interview .dot { background: var(--color-primary); }
	.tone-rejected .dot { background: var(--color-error); }
	.tone-offer .dot { background: var(--color-success); }
	.tone-closed .dot { background: var(--color-text-muted); }

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
