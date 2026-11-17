<script lang="ts">
	import { t } from '#lib/i18n/t.js';
	import { formatDate } from '#lib/i18n/format.js';
	import type { Snippet } from 'svelte';
	import Badge from '#lib/ui/Badge.svelte';
	import Panel from '#lib/ui/Panel.svelte';
	import Icon from '#lib/ui/Icon.svelte';
	import { SOURCE_LABELS, statusLabel, scoreTone, type Job } from '#lib/types/job.js';

	interface Props {
		job: Job;
		actions?: Snippet;
	}

	let { job, actions }: Props = $props();
</script>

<div class="detail">
	<div class="title-row">
		<h2>{job.title}</h2>
		<Badge tone={job.status === 'approved' ? 'success' : job.status === 'discarded' ? 'error' : 'neutral'}>
			{statusLabel(job.status)}
		</Badge>
	</div>

	<p class="company">{job.company}</p>

	<div class="meta">
		{#if job.source}
			<Badge tone="info">{SOURCE_LABELS[job.source] ?? job.source}</Badge>
		{/if}
		{#if job.score_fit !== null}
			<Badge tone={scoreTone(job.score_fit)}>{t('common.score')} {job.score_fit}/100</Badge>
		{:else}
			<Badge>{t('jobs.no_score')}</Badge>
		{/if}
		<Badge>{formatDate(job.created_at)}</Badge>
	</div>

	{#if job.url}
		<a class="link" href={job.url} target="_blank" rel="noopener noreferrer">
			<Icon name="external" size={16} />
			<span>{job.url}</span>
		</a>
	{/if}

	{#if (job.matched_keywords?.length ?? 0) > 0 || (job.matched_resume_terms?.length ?? 0) > 0}
		<Panel title={t('jobdetail.fit')}>
			{#if job.matched_keywords?.length}
				<div class="chips">
					<span class="chip-label">{t('jobdetail.keywords')}</span>
					{#each job.matched_keywords as kw}
						<Badge tone="primary">{kw}</Badge>
					{/each}
				</div>
			{/if}
			{#if job.matched_resume_terms?.length}
				<div class="chips">
					<span class="chip-label">{t('jobdetail.resume')}</span>
					{#each job.matched_resume_terms as term}
						<Badge tone="success">{term}</Badge>
					{/each}
				</div>
			{/if}
		</Panel>
	{/if}

	{#if job.score_reason}
		<Panel title={t('jobdetail.score_reason')}>
			<p>{job.score_reason}</p>
		</Panel>
	{/if}

	{#if job.description}
		<Panel title={t('jobdetail.description')}>
			<p class="description">{job.description}</p>
		</Panel>
	{/if}

	{#if actions}
		<div class="actions">{@render actions()}</div>
	{/if}
</div>

<style>
	.detail {
		display: flex;
		flex-direction: column;
		gap: var(--space-4);
	}

	.title-row {
		display: flex;
		align-items: flex-start;
		justify-content: space-between;
		gap: var(--space-3);
	}

	h2 {
		margin: 0;
		font-size: var(--text-h2);
	}

	.company {
		margin: 0;
		color: var(--color-text-secondary);
	}

	.meta,
	.chips {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: var(--space-2);
	}

	.chip-label {
		font-size: var(--text-caption);
		font-weight: var(--weight-semibold);
		color: var(--color-text-muted);
	}

	.link {
		display: inline-flex;
		align-items: center;
		gap: var(--space-2);
		overflow-wrap: anywhere;
		font-size: var(--text-caption);
	}

	p {
		margin: 0;
		font-size: var(--text-caption);
		line-height: var(--leading-caption);
	}

	.description {
		max-height: 16rem;
		overflow-y: auto;
		white-space: pre-wrap;
	}

	.actions {
		display: flex;
		gap: var(--space-2);
		flex-wrap: wrap;
		margin-top: var(--space-2);
	}
</style>
