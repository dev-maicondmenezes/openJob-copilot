<script lang="ts">
	import { t } from '#lib/i18n/t.js';
	import { errorText } from '#lib/i18n/errors.js';
	import { formatDateTime } from '#lib/i18n/format.js';
	import { onMount } from 'svelte';
	import MarkdownView from '#lib/components/MarkdownView.svelte';
	import { computeUnifiedDiff, computeSideBySideDiff, type DiffLine, type SideBySideRow } from '../../lib/diff';
	import PageHeader from '#lib/ui/PageHeader.svelte';
	import Card from '#lib/ui/Card.svelte';
	import Panel from '#lib/ui/Panel.svelte';
	import Field from '#lib/ui/Field.svelte';
	import Textarea from '#lib/ui/Textarea.svelte';
	import Button from '#lib/ui/Button.svelte';
	import Badge from '#lib/ui/Badge.svelte';
	import Alert from '#lib/ui/Alert.svelte';
	import SegmentedControl from '#lib/ui/SegmentedControl.svelte';
	import Select from '#lib/ui/Select.svelte';
	import Icon from '#lib/ui/Icon.svelte';
	import type { Job } from '#lib/types/job.js';

	interface ActiveCurriculum {
		id: number;
		content_md: string;
		source_filename: string;
		uploaded_at: string;
	}

	let activeCV: ActiveCurriculum | null = $state(null);
	let llmConfigured = $state(false);
	let loadingCV = $state(true);
	let pendingJobs: Job[] = $state([]);
	let selectedJobId = $state('');
	let uploading = $state(false);
	let fileInput: HTMLInputElement | null = $state(null);
	let showBasePreview = $state(false);
	let rawBaseView = $state(false);
	let jdText = $state('');
	let tailoring = $state(false);
	let tailoredMD = $state('');
	let promptVersion = $state('');
	let status: { kind: 'success' | 'error'; text: string } | null = $state(null);
	let diffMode: 'diff' | 'side-by-side' | 'preview' = $state('diff');
	let copyFeedback = $state(false);

	onMount(async () => {
		await Promise.all([loadCurriculum(), checkLLM(), loadPendingJobs()]);
		loadingCV = false;
	});

	async function loadPendingJobs(): Promise<void> {
		try {
			const res = await fetch('/api/jobs');
			if (!res.ok) return;
			const body = await res.json();
			pendingJobs = ((body.jobs as Job[] | undefined) ?? []).filter(
				(job) => job.status === 'new' || job.status === 'pending'
			);
		} catch {
			// silencioso — a seleção simplesmente fica vazia
		}
	}

	const selectedJob = $derived(pendingJobs.find((job) => String(job.id) === selectedJobId) ?? null);
	const jobOptions = $derived([
		{ value: '', label: t('curriculum.tailoring.select_job') },
		...pendingJobs.map((job) => ({ value: String(job.id), label: `${job.title} · ${job.company}` }))
	]);

	function selectJob(id: string): void {
		selectedJobId = id;
		const job = pendingJobs.find((item) => String(item.id) === id);
		// Pré-preenche a JD com a descrição da vaga escolhida (ainda editável).
		jdText = job?.description ?? '';
	}

	async function checkLLM() {
		try {
			const res = await fetch('/api/settings/llm-key');
			if (res.ok) llmConfigured = !!(await res.json()).configured;
		} catch {
			// ignore
		}
	}

	async function loadCurriculum() {
		try {
			const res = await fetch('/api/curriculum');
			if (res.ok) activeCV = (await res.json()).curriculum;
		} catch {
			status = { kind: 'error', text: t('curriculum.error.load') };
		}
	}

	async function handleFileUpload(e: Event) {
		const target = e.target as HTMLInputElement;
		const file = target.files?.[0];
		if (!file) return;
		uploading = true;
		status = null;
		const form = new FormData();
		form.append('file', file);
		try {
			const res = await fetch('/api/curriculum/upload', { method: 'POST', body: form });
			const body = await res.json().catch(() => ({}));
			if (res.ok) {
				activeCV = {
					id: body.id,
					content_md: body.content_md,
					source_filename: body.source_filename,
					uploaded_at: body.uploaded_at ?? new Date().toISOString()
				};
				showBasePreview = true;
				status = { kind: 'success', text: t('curriculum.error.uploaded', { file: file.name }) };
			} else {
				status = { kind: 'error', text: errorText(body, t('curriculum.error.upload_http', { status: res.status })) };
			}
		} catch {
			status = { kind: 'error', text: t('curriculum.error.upload_network') };
		} finally {
			uploading = false;
			if (fileInput) fileInput.value = '';
		}
	}

	async function generateTailored() {
		if (!activeCV) {
			status = { kind: 'error', text: t('curriculum.error.no_base') };
			return;
		}
		if (!jdText.trim()) {
			status = { kind: 'error', text: t('curriculum.error.no_jd') };
			return;
		}
		if (!llmConfigured) {
			status = { kind: 'error', text: t('curriculum.error.no_key') };
			return;
		}
		tailoring = true;
		status = null;
		try {
			const res = await fetch('/api/curriculum/tailor', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ jd_text: jdText })
			});
			const body = await res.json().catch(() => ({}));
			if (res.ok) {
				tailoredMD = body.tailored_md;
				promptVersion = body.prompt_version ?? 'v1';
				status = { kind: 'success', text: t('curriculum.error.tailored') };
			} else {
				status = { kind: 'error', text: errorText(body, t('curriculum.error.tailor_http', { status: res.status })) };
			}
		} catch {
			status = { kind: 'error', text: t('curriculum.error.tailor_network') };
		} finally {
			tailoring = false;
		}
	}

	const unifiedDiff = $derived.by<DiffLine[]>(() =>
		activeCV?.content_md && tailoredMD ? computeUnifiedDiff(activeCV.content_md, tailoredMD) : []
	);
	const sideBySideDiff = $derived.by<SideBySideRow[]>(() =>
		activeCV?.content_md && tailoredMD ? computeSideBySideDiff(activeCV.content_md, tailoredMD) : []
	);

	function copyTailoredMarkdown() {
		if (!tailoredMD) return;
		void navigator.clipboard.writeText(tailoredMD);
		copyFeedback = true;
		setTimeout(() => (copyFeedback = false), 2000);
	}

	function downloadTailoredMarkdown() {
		if (!tailoredMD) return;
		const blob = new Blob([tailoredMD], { type: 'text/markdown;charset=utf-8' });
		const objectUrl = URL.createObjectURL(blob);
		const anchor = document.createElement('a');
		anchor.href = objectUrl;
		anchor.download = `curriculo_adaptado_${promptVersion || 'v1'}.md`;
		anchor.click();
		URL.revokeObjectURL(objectUrl);
	}
</script>

<svelte:head>
	<title>{t('curriculum.meta_title')}</title>
</svelte:head>

<PageHeader
	title={t('curriculum.title')}
	subtitle={t('curriculum.subtitle_full')}
	breadcrumb={[{ label: t('common.home'), href: '/' }, { label: t('common.curriculum') }]}
/>

{#if status}
	<Alert severity={status.kind}>{status.text}</Alert>
{/if}

{#if !llmConfigured && !loadingCV}
	<Alert severity="warning" title={t('jobs.provider_warning_title')}>
		{t('curriculum.warning.body')}
	</Alert>
{/if}

<div class="grid grid-2 curriculum-grid">
	<Card title={t('curriculum.base.title')} subtitle={t('curriculum.base.one_active')}>
		{#snippet actions()}
			<Badge tone={activeCV ? 'success' : 'warning'}>{activeCV ? t('curriculum.base.active') : t('curriculum.base.none')}</Badge>
		{/snippet}
		<input
			type="file"
			id="cv-file"
			aria-label={t('curriculum.base.select_file')}
			accept=".pdf,.md,.markdown,.txt"
			bind:this={fileInput}
			onchange={handleFileUpload}
			class="visually-hidden"
		/>
		{#if activeCV}
			<div class="file-info">
				<span class="file-icon"><Icon name="file" size={24} /></span>
				<div class="file-text">
					<strong>{activeCV.source_filename || t('curriculum.base.filename')}</strong>
					<span class="hint">
						{formatDateTime(activeCV.uploaded_at)} · {activeCV.content_md.length} {t('curriculum.base.chars')}
					</span>
				</div>
			</div>
			<div class="actions">
				<Button variant="secondary" loading={uploading} onclick={() => fileInput?.click()}>{t('curriculum.base.replace')}</Button>
				<Button variant="ghost" onclick={() => (showBasePreview = !showBasePreview)}>
					{showBasePreview ? t('curriculum.base.hide_preview') : t('curriculum.base.show_preview')}
				</Button>
			</div>
			{#if showBasePreview}
				<div class="preview-box">
					<div class="preview-head">
						<span>{t('curriculum.base.preview_title')}</span>
						<Button type="button" variant="link" size="sm" onclick={() => (rawBaseView = !rawBaseView)}>
							{rawBaseView ? t('curriculum.base.rendered') : t('curriculum.base.raw')}
						</Button>
					</div>
					<div class="preview-content">
						{#if rawBaseView}
							<pre class="raw-markdown">{activeCV.content_md}</pre>
						{:else}
							<MarkdownView content={activeCV.content_md} />
						{/if}
					</div>
				</div>
			{/if}
		{:else}
			<div class="empty-upload">
				<p>{t('curriculum.base.hint')}</p>
				<Button loading={uploading} onclick={() => fileInput?.click()}>{t('curriculum.base.select')}</Button>
			</div>
		{/if}
	</Card>

	<Card title={t('curriculum.tailoring.title')} subtitle={t('curriculum.tailoring.subtitle')}>
		{#snippet actions()}
			{#if promptVersion}<Badge tone="info">{t('curriculum.tailoring.prompt', { version: promptVersion })}</Badge>{/if}
		{/snippet}
		<form onsubmit={(e) => { e.preventDefault(); generateTailored(); }}>
			<Field label="{t('jobs.col.job')} pendente" for="tailoring-job" hint={pendingJobs.length === 0 ? t('curriculum.tailoring.no_job') : undefined}>
				<Select id="tailoring-job" options={jobOptions} value={selectedJobId} onchange={(e) => selectJob(e.currentTarget.value)} />
			</Field>
			{#if selectedJob}
				<Panel title={t('curriculum.tailoring.selected')}>
					<div class="selected-job">
						<strong>{selectedJob.title}</strong>
						<span class="hint">{selectedJob.company}</span>
						{#if selectedJob.score_fit !== null}
							<Badge tone="info">{t('common.score')} {t('jobdetail.score_label', { score: selectedJob.score_fit })}</Badge>
						{/if}
					</div>
				</Panel>
			{/if}
			<Field label={t('curriculum.tailoring.jd')} for="jd_text" hint={t('curriculum.tailoring.jd_hint')}>
				<Textarea
					id="jd_text"
					rows={8}
					bind:value={jdText}
					disabled={tailoring}
					placeholder={t('curriculum.tailoring.jd_placeholder')}
				/>
			</Field>
			<div class="actions">
				<Button type="submit" loading={tailoring} disabled={!activeCV || !jdText.trim() || !llmConfigured}>
					{t('curriculum.tailoring.generate')}
				</Button>
			</div>
		</form>
	</Card>
</div>

{#if tailoredMD}
	<Card title={t('curriculum.result.title')} subtitle={t('curriculum.result.desc')}>
		{#snippet actions()}
			<SegmentedControl
				options={[
					{ value: 'diff', label: t('curriculum.result.diff') },
					{ value: 'side-by-side', label: t('curriculum.result.side_by_side') },
					{ value: 'preview', label: t('curriculum.result.rendered') }
				]}
				value={diffMode}
				label={t('curriculum.result.mode')}
				onchange={(value) => (diffMode = value as typeof diffMode)}
			/>
		{/snippet}
		<div class="result-toolbar">
			<Button variant="secondary" size="sm" onclick={copyTailoredMarkdown}>
				{copyFeedback ? t('curriculum.result.copied') : t('curriculum.result.copy')}
			</Button>
			<Button variant="ghost" size="sm" onclick={downloadTailoredMarkdown}>{t('curriculum.result.download')}</Button>
		</div>
		<div class="diff-container">
			{#if diffMode === 'diff'}
				<div class="unified-diff">
					{#each unifiedDiff as line}
						<div class="diff-row {line.type}">
							<span class="sign">{line.type === 'added' ? '+' : line.type === 'removed' ? '-' : ' '}</span>
							<span class="line-content">{line.text || ' '}</span>
						</div>
					{/each}
				</div>
			{:else if diffMode === 'side-by-side'}
				<div class="side-by-side-wrapper">
					<div class="sbs-header"><span>{t('curriculum.base.title')}</span><span>{t('curriculum.result.tailored')}</span></div>
					{#each sideBySideDiff as row}
						<div class="sbs-row">
							<div class="sbs-col left {row.left?.type ?? ''}">
								<span class="lineno">{row.left?.lineNo ?? ''}</span>
								<span class="line-text">{row.left?.text ?? ''}</span>
							</div>
							<div class="sbs-col right {row.right?.type ?? ''}">
								<span class="lineno">{row.right?.lineNo ?? ''}</span>
								<span class="line-text">{row.right?.text ?? ''}</span>
							</div>
						</div>
					{/each}
				</div>
			{:else}
				<div class="adapted-preview"><MarkdownView content={tailoredMD} /></div>
			{/if}
		</div>
	</Card>
{/if}

<style>
	.curriculum-grid {
		margin-bottom: var(--space-6);
	}

	.selected-job {
		display: flex;
		flex-direction: column;
		align-items: flex-start;
		gap: var(--space-2);
	}

	.file-info {
		display: flex;
		align-items: center;
		gap: var(--space-3);
		padding: var(--space-3);
		background: var(--color-surface-soft);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-md);
	}

	.file-icon {
		color: var(--color-primary);
	}

	.file-text {
		display: flex;
		flex-direction: column;
		gap: var(--space-1);
		min-width: 0;
	}

	.actions {
		display: flex;
		gap: var(--space-2);
		flex-wrap: wrap;
		margin-top: var(--space-4);
	}

	form {
		display: flex;
		flex-direction: column;
	}

	.empty-upload {
		border: 2px dashed var(--color-border);
		border-radius: var(--radius-md);
		padding: var(--space-8) var(--space-4);
		text-align: center;
		background: var(--color-surface-soft);
	}

	.empty-upload p {
		margin-bottom: var(--space-4);
		font-size: var(--text-body);
		color: var(--color-text-muted);
	}

	.preview-box {
		margin-top: var(--space-4);
		border-top: 1px solid var(--color-border);
		padding-top: var(--space-3);
	}

	.preview-head {
		display: flex;
		justify-content: space-between;
		gap: var(--space-3);
		font-size: var(--text-caption);
		font-weight: var(--weight-semibold);
		color: var(--color-text-muted);
		margin-bottom: var(--space-2);
	}

	.preview-content {
		max-height: 20rem;
		overflow-y: auto;
		background: var(--color-surface-soft);
		padding: var(--space-3);
		border-radius: var(--radius-md);
		border: 1px solid var(--color-border);
	}

	.raw-markdown {
		margin: 0;
		font-family: var(--font-mono);
		font-size: var(--text-mono);
		white-space: pre-wrap;
		word-break: break-word;
	}

	.result-toolbar {
		display: flex;
		gap: var(--space-2);
		margin-bottom: var(--space-3);
	}

	.diff-container {
		background: var(--color-surface-soft);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-md);
		overflow: auto;
		max-height: 30rem;
	}

	.unified-diff {
		font-family: var(--font-mono);
		font-size: var(--text-mono);
		line-height: 1.45;
	}

	.diff-row {
		display: flex;
		padding: var(--space-1) var(--space-2);
		white-space: pre-wrap;
		word-break: break-word;
	}

	.diff-row.added {
		background: var(--color-success-soft);
		color: var(--color-success);
	}

	.diff-row.removed {
		background: var(--color-error-soft);
		color: var(--color-error);
	}

	.sign {
		width: 1.25rem;
		user-select: none;
		font-weight: var(--weight-bold);
	}

	.line-content {
		flex: 1;
	}

	.side-by-side-wrapper {
		min-width: 35rem;
		font-family: var(--font-mono);
		font-size: var(--text-mono);
	}

	.sbs-header {
		display: grid;
		grid-template-columns: 1fr 1fr;
		background: var(--color-surface-elevated);
		font-weight: var(--weight-semibold);
		padding: var(--space-2) var(--space-3);
		border-bottom: 1px solid var(--color-border);
	}

	.sbs-row {
		display: grid;
		grid-template-columns: 1fr 1fr;
		border-bottom: 1px solid var(--color-border);
	}

	.sbs-col {
		display: flex;
		padding: var(--space-1) var(--space-2);
		overflow-x: hidden;
		white-space: pre-wrap;
		word-break: break-word;
	}

	.sbs-col.left {
		border-right: 1px solid var(--color-border);
	}

	.sbs-col.added {
		background: var(--color-success-soft);
		color: var(--color-success);
	}

	.sbs-col.removed {
		background: var(--color-error-soft);
		color: var(--color-error);
	}

	.lineno {
		width: 2rem;
		color: var(--color-text-muted);
		user-select: none;
	}

	.line-text {
		flex: 1;
	}

	.adapted-preview {
		padding: var(--space-4);
		background: var(--color-surface);
	}
</style>
