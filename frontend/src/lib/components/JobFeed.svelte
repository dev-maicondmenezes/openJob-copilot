<script lang="ts">
	import { t } from '#lib/i18n/t.js';
	import { errorText } from '#lib/i18n/errors.js';
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { toast } from '#lib/toast.svelte.ts';
	import { currentLocale } from '#lib/i18n/format.js';
	import { viewport } from '#lib/ui/media.svelte.js';
	import { SOURCE_LABELS, type Job } from '#lib/types/job.js';

	import JobFeedCard from '#lib/components/JobFeedCard.svelte';
	import Button from '#lib/ui/Button.svelte';
	import IconButton from '#lib/ui/IconButton.svelte';
	import LinkButton from '#lib/ui/LinkButton.svelte';
	import Badge from '#lib/ui/Badge.svelte';
	import Dialog from '#lib/ui/Dialog.svelte';
	import Field from '#lib/ui/Field.svelte';
	import Textarea from '#lib/ui/Textarea.svelte';
	import Select from '#lib/ui/Select.svelte';
	import MultiSelect from '#lib/ui/MultiSelect.svelte';
	import EmptyState from '#lib/ui/EmptyState.svelte';
	import Skeleton from '#lib/ui/Skeleton.svelte';
	import Icon from '#lib/ui/Icon.svelte';

	let jobs: Job[] = $state([]);
	let loading = $state(true);
	let discovering = $state(false);
	let actingId = $state<number | null>(null);

	// Navegação no feed
	let activeIndex = $state(0);
	let scrollContainer = $state<HTMLElement | null>(null);
	let slotElements = $state<Record<number, HTMLElement>>({});

	// Filtros
	let statusFilter = $state('all');
	let sourceFilter = $state<string[]>([]);
	let searchTerm = $state(page.url.searchParams.get('q') ?? '');
	let filtersOpen = $state(false);

	// Formulário Nova Vaga
	let newJobOpen = $state(false);
	let url = $state('');
	let title = $state('');
	let company = $state('');
	let description = $state('');
	let fetching = $state(false);
	let saving = $state(false);

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

	const hasActiveFilters = $derived(
		statusFilter !== 'all' || sourceFilter.length > 0 || searchTerm.trim() !== ''
	);

	onMount(async () => {
		await loadJobs();
	});

	async function loadJobs(): Promise<void> {
		try {
			const res = await fetch('/api/jobs');
			if (res.ok) {
				const body = await res.json();
				jobs = body.jobs ?? [];
			}
		} catch {
			toast('error', t('jobs.error.load_network'));
		} finally {
			loading = false;
		}
	}

	async function triggerDiscovery(): Promise<void> {
		if (discovering) return;
		discovering = true;
		try {
			const res = await fetch('/api/jobs/discover', { method: 'POST' });
			if (res.status === 409) {
				toast('error', t('jobs.flash.search_running'));
			} else if (!res.ok) {
				const err = await res.json().catch(() => ({}));
				toast('error', errorText(err, t('jobs.flash.search_start_failed')));
				discovering = false;
				return;
			}
			const previousCount = jobs.length;
			for (let i = 0; i < 40; i++) {
				await new Promise((r) => setTimeout(r, 1200));
				const sRes = await fetch('/api/jobs/discover/status');
				if (sRes.ok) {
					const sData = await sRes.json();
					if (!sData.running) {
						const stats = sData.latest_run?.stats as
							| { fetched: number; new_jobs: number; scored: number }
							| undefined;
						if (stats && stats.new_jobs > 0) {
							toast('success', t('jobs.flash.new_jobs_added', { count: stats.new_jobs }));
						} else {
							toast('info', t('jobs.flash.no_new_jobs'));
						}
						break;
					}
				}
			}
			await loadJobs();
		} catch {
			toast('error', t('jobs.error.search_network'));
		} finally {
			discovering = false;
		}
	}

	async function approve(job: Job): Promise<void> {
		actingId = job.id;
		try {
			const res = await fetch(`/api/jobs/${job.id}/approve`, { method: 'POST' });
			const body = await res.json().catch(() => ({}));
			if (res.ok) {
				job.status = 'approved';
				toast('success', t('jobs.flash.approved', { title: job.title }));
			} else {
				toast('error', errorText(body, t('jobs.error.approve_http', { status: res.status })));
			}
		} catch {
			toast('error', t('jobs.error.approve_network'));
		} finally {
			actingId = null;
		}
	}

	async function discard(job: Job): Promise<void> {
		actingId = job.id;
		try {
			const res = await fetch(`/api/jobs/${job.id}/discard`, { method: 'POST' });
			const body = await res.json().catch(() => ({}));
			if (res.ok) {
				job.status = 'discarded';
				toast('success', t('jobs.flash.discarded'));
			} else {
				toast('error', errorText(body, t('jobs.error.discard_http', { status: res.status })));
			}
		} catch {
			toast('error', t('jobs.error.discard_network'));
		} finally {
			actingId = null;
		}
	}

	async function fetchDraft(link: string): Promise<void> {
		const target = link.trim();
		if (!/^https?:\/\//.test(target) || fetching) return;
		fetching = true;
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
				toast('success', t('jobs.flash.filled_from_url'));
			} else {
				toast('error', errorText(body, t('jobs.error.fetch_http', { status: res.status })));
			}
		} catch {
			toast('error', t('jobs.error.fetch_network'));
		} finally {
			fetching = false;
		}
	}

	function onUrlPaste(e: ClipboardEvent): void {
		const text = e.clipboardData?.getData('text') ?? '';
		if (/^https?:\/\//.test(text.trim())) {
			e.preventDefault();
			url = text.trim();
			void fetchDraft(url);
		}
	}

	async function saveNewJob(): Promise<void> {
		if (!canSave) return;
		saving = true;
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
				toast('success', t('jobs.flash.saved'));
				await loadJobs();
			} else {
				toast('error', errorText(body, t('jobs.error.save_http', { status: res.status })));
			}
		} catch {
			toast('error', t('jobs.error.save_network'));
		} finally {
			saving = false;
		}
	}

	function clearFilters(): void {
		statusFilter = 'all';
		sourceFilter = [];
		searchTerm = '';
		activeIndex = 0;
	}

	function scrollToIndex(targetIndex: number): void {
		if (targetIndex < 0 || targetIndex >= filteredJobs.length) return;
		activeIndex = targetIndex;
		const el = slotElements[targetIndex];
		if (el) {
			const prefersReduced = typeof window !== 'undefined' && window.matchMedia('(prefers-reduced-motion: reduce)').matches;
			el.scrollIntoView({ behavior: prefersReduced ? 'auto' : 'smooth', block: 'start' });
		}
	}

	function goPrev(): void {
		scrollToIndex(activeIndex - 1);
	}

	function goNext(): void {
		scrollToIndex(activeIndex + 1);
	}

	function onFeedKeyDown(event: KeyboardEvent): void {
		if (event.target instanceof HTMLElement && event.target.closest('input, textarea, select')) return;
		if (event.key === 'ArrowDown') {
			event.preventDefault();
			goNext();
		} else if (event.key === 'ArrowUp') {
			event.preventDefault();
			goPrev();
		}
	}

	$effect(() => {
		if (!scrollContainer) return;
		const observer = new IntersectionObserver(
			(entries) => {
				for (const entry of entries) {
					if (entry.isIntersecting) {
						const idx = Number((entry.target as HTMLElement).dataset.index);
						if (!Number.isNaN(idx)) {
							activeIndex = idx;
						}
					}
				}
			},
			{ root: scrollContainer, threshold: 0.6 }
		);

		for (const el of Object.values(slotElements)) {
			if (el) observer.observe(el);
		}

		return () => observer.disconnect();
	});
</script>

<svelte:window onkeydown={onFeedKeyDown} />

<div class="job-feed">
	<header class="feed-header">
		<div class="feed-header-actions">
			<Button
				variant="primary"
				size="sm"
				loading={discovering}
				onclick={triggerDiscovery}
			>
				<Icon name="sparkles" size={16} />
				<span>{discovering ? t('jobs.searching') : t('jobs.search_now')}</span>
			</Button>

			<Button
				variant="secondary"
				size="sm"
				onclick={() => (newJobOpen = true)}
			>
				<Icon name="plus" size={16} />
				<span>{t('jobs.new')}</span>
			</Button>
		</div>

		<div class="feed-header-views">
			<Button
				variant="ghost"
				size="sm"
				onclick={() => (filtersOpen = true)}
			>
				<Icon name="sliders" size={16} />
				<span>{t('common.filters')}</span>
				{#if hasActiveFilters}
					<Badge tone="primary">•</Badge>
				{/if}
			</Button>

			<LinkButton
				href="/jobs/list"
				variant="ghost"
				size="sm"
			>
				<Icon name="clipboard" size={16} />
				<span>{t('jobs.view_list')}</span>
			</LinkButton>
		</div>
	</header>

	<div class="visually-hidden feed-announcement" aria-live="polite">
		{t('jobs.feed_announcement', { current: activeIndex + 1, total: filteredJobs.length })}
	</div>

	<div
		class="feed-scroll-container"
		role={filteredJobs.length > 0 ? 'feed' : undefined}
		aria-label={filteredJobs.length > 0 ? t('jobs.feed_title') : undefined}
		bind:this={scrollContainer}
	>
		{#if loading}
			<div class="feed-slot">
				<div class="skeleton-card">
					<Skeleton />
				</div>
			</div>
		{:else if jobs.length === 0}
			<div class="feed-slot">
				<div class="state-card">
					<EmptyState
						title={t('jobs.empty.title')}
						description={t('jobs.empty.desc')}
					>
						<Button variant="primary" size="sm" onclick={triggerDiscovery}>{t('jobs.search_now')}</Button>
						<Button variant="secondary" size="sm" onclick={() => (newJobOpen = true)}>{t('jobs.new')}</Button>
					</EmptyState>
				</div>
			</div>
		{:else if filteredJobs.length === 0}
			<div class="feed-slot">
				<div class="state-card">
					<EmptyState
						title={t('jobs.empty.filtered_title')}
						description={t('jobs.empty.filtered_desc')}
					>
						<Button variant="secondary" size="sm" onclick={clearFilters}>{t('jobs.filters.clear')}</Button>
					</EmptyState>
				</div>
			</div>
		{:else}
			{#each filteredJobs as job, i (job.id)}
				<div
					class="feed-slot"
					data-index={i}
					role="none"
					bind:this={slotElements[i]}
				>
					{#if Math.abs(i - activeIndex) <= 1}
						<JobFeedCard
							{job}
							index={i}
							total={filteredJobs.length}
							active={i === activeIndex}
							onApprove={approve}
							onDiscard={discard}
							acting={actingId === job.id}
						/>
					{:else}
						<div class="feed-slot-placeholder" aria-hidden="true"></div>
					{/if}
				</div>
			{/each}
		{/if}
	</div>

	{#if viewport.isDesktop && filteredJobs.length > 1}
		<div class="desktop-nav" aria-label="Navegação do feed">
			<IconButton
				icon="arrow-up"
				label={t('jobs.nav_prev')}
				variant="secondary"
				size="sm"
				disabled={activeIndex === 0}
				onclick={goPrev}
			/>
			<IconButton
				icon="arrow-down"
				label={t('jobs.nav_next')}
				variant="secondary"
				size="sm"
				disabled={activeIndex >= filteredJobs.length - 1}
				onclick={goNext}
			/>
		</div>
	{/if}
</div>

<Dialog open={newJobOpen} onclose={() => (newJobOpen = false)} title={t('jobs.new')} description={t('jobs.dialog.new_desc')}>
	<form id="feed-new-job-form" onsubmit={(e) => { e.preventDefault(); void saveNewJob(); }}>
		<Field label={t('jobs.form.url')} for="url">
			<div class="url-input-row">
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
	{#snippet footer()}
		<Button variant="ghost" onclick={() => (newJobOpen = false)}>{t('common.cancel')}</Button>
		<Button type="submit" form="feed-new-job-form" loading={saving} disabled={!canSave}>{t('jobs.dialog.save')}</Button>
	{/snippet}
</Dialog>

<Dialog open={filtersOpen} onclose={() => (filtersOpen = false)} title={t('jobs.dialog.filter_title')}>
	<div class="filters-form">
		<Field label={t('common.status')} for="feed-status">
			<Select
				id="feed-status"
				value={statusFilter}
				onchange={(e) => { statusFilter = e.currentTarget.value; activeIndex = 0; }}
				options={statusOptions}
			/>
		</Field>
		<Field label={t('jobs.filters.source_multi')}>
			<MultiSelect
				label={t('jobs.filters.source_button')}
				options={sourceOptions}
				bind:value={sourceFilter}
				allLabel={t('jobs.filters.all_sources')}
				onchange={() => (activeIndex = 0)}
			/>
		</Field>
		<Field label={t('jobs.filters.search')} for="feed-search">
			<input
				id="feed-search"
				type="search"
				bind:value={searchTerm}
				oninput={() => (activeIndex = 0)}
				placeholder={t('jobs.filters.search_placeholder')}
			/>
		</Field>
	</div>
	{#snippet footer()}
		<Button variant="ghost" onclick={clearFilters}>{t('common.clear')}</Button>
		<Button onclick={() => (filtersOpen = false)}>{t('common.apply')}</Button>
	{/snippet}
</Dialog>

<style>
	.job-feed {
		position: relative;
		display: flex;
		flex-direction: column;
		width: 100%;
		height: 100%;
		min-height: 0;
		overflow: hidden;
	}

	.feed-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		flex-shrink: 0;
		padding: var(--space-2) var(--space-4);
		margin-bottom: var(--space-2);
		background: var(--color-surface);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-md);
		gap: var(--space-3);
	}

	.feed-header-actions,
	.feed-header-views {
		display: flex;
		align-items: center;
		gap: var(--space-2);
	}

	.feed-scroll-container {
		flex: 1 1 auto;
		min-height: 0;
		overflow-y: auto;
		scroll-snap-type: y mandatory;
		scroll-behavior: smooth;
		-webkit-overflow-scrolling: touch;
		display: flex;
		flex-direction: column;
		gap: var(--space-4);
	}

	@media (prefers-reduced-motion: reduce) {
		.feed-scroll-container {
			scroll-behavior: auto;
		}
	}

	.feed-slot {
		flex: 0 0 100%;
		height: 100%;
		min-height: 100%;
		scroll-snap-align: start;
		scroll-snap-stop: always;
		display: flex;
		flex-direction: column;
	}

	.feed-slot-placeholder {
		height: 100%;
		width: 100%;
		background: transparent;
	}

	.skeleton-card,
	.state-card {
		height: 100%;
		padding: var(--space-8);
		background: var(--color-surface);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-lg);
		display: flex;
		align-items: center;
		justify-content: center;
		box-sizing: border-box;
	}

	.desktop-nav {
		position: absolute;
		right: var(--space-4);
		bottom: var(--space-4);
		display: flex;
		flex-direction: column;
		gap: var(--space-2);
		z-index: 10;
	}

	.url-input-row {
		display: flex;
		gap: var(--space-2);
	}

	.url-input-row input {
		flex: 1;
		min-width: 0;
	}

	form {
		display: flex;
		flex-direction: column;
		gap: var(--space-4);
	}

	.filters-form {
		display: flex;
		flex-direction: column;
		gap: var(--space-4);
	}
</style>