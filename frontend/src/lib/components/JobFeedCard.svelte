<script lang="ts">
	import { t } from '#lib/i18n/t.js';
	import { statusLabel, SOURCE_LABELS, scoreTone, type Job } from '#lib/types/job.js';
	import Badge from '#lib/ui/Badge.svelte';
	import Button from '#lib/ui/Button.svelte';
	import Icon from '#lib/ui/Icon.svelte';
	import MarkdownView from '#lib/components/MarkdownView.svelte';
	import { toast } from '#lib/toast.svelte.ts';

	interface Props {
		job: Job;
		index: number;
		total: number;
		active: boolean;
		onApprove: (job: Job) => Promise<void>;
		onDiscard: (job: Job) => Promise<void>;
		acting?: boolean;
	}

	let {
		job,
		index,
		total,
		active,
		onApprove,
		onDiscard,
		acting = false
	}: Props = $props();

	function getTeaser(text: string, maxLen = 280): string {
		if (!text) return '';
		const trimmed = text.trim();
		if (trimmed.length <= maxLen) return trimmed;
		const cutoff = trimmed.lastIndexOf(' ', maxLen);
		return (cutoff > 0 ? trimmed.slice(0, cutoff) : trimmed.slice(0, maxLen)) + '…';
	}

	async function shareJob(): Promise<void> {
		try {
			const shareUrl = `${window.location.origin}/jobs/${job.id}`;
			if (navigator.clipboard) {
				await navigator.clipboard.writeText(shareUrl);
			}
			toast('success', t('jobs.share_copied'));
		} catch {
			toast('info', t('jobs.share_copied'));
		}
	}
</script>

<article
	class="job-feed-card"
	class:active
	aria-label="{job.title} — {job.company}"
	inert={!active}
>
	<header class="card-header">
		<div class="header-top">
			<div class="company-row">
				<span class="company-name">{job.company}</span>
				{#if job.source}
					<Badge tone="info">{SOURCE_LABELS[job.source] ?? job.source}</Badge>
				{/if}
				{#if job.score_fit !== null}
					<Badge tone={scoreTone(job.score_fit)}>{job.score_fit}% fit</Badge>
				{:else}
					<Badge tone="neutral">{t('jobs.no_score')}</Badge>
				{/if}
				{#if job.status === 'approved'}
					<Badge tone="success">{statusLabel(job.status)}</Badge>
				{:else if job.status === 'discarded'}
					<Badge tone="error">{statusLabel(job.status)}</Badge>
				{/if}
			</div>
			<span class="position-badge" aria-hidden="true">{index + 1} / {total}</span>
		</div>

		<h2 class="job-title">{job.title}</h2>

		{#if job.url}
			<a
				href={job.url}
				target="_blank"
				rel="noopener noreferrer"
				class="source-link"
				title="{SOURCE_LABELS[job.source] ?? job.source} — {job.url}"
			>
				<Icon name="external" size={16} />
				<span class="source-url">{job.url}</span>
			</a>
		{/if}
	</header>

	<div class="card-body">
		<div class="teaser-wrapper">
			<MarkdownView content={getTeaser(job.description)} />
			<div class="teaser-gradient" aria-hidden="true"></div>
		</div>

		<div class="read-more-row">
			<a href="/jobs/{job.id}" class="read-more-link">
				<Icon name="file" size={16} />
				<span>{t('jobs.read_more')}</span>
				<Icon name="chevron-right" size={16} />
			</a>
		</div>
	</div>

	<footer class="card-footer">
		<div class="actions-left">
			<Button
				variant="primary"
				size="sm"
				loading={acting}
				disabled={job.status === 'approved'}
				onclick={() => onApprove(job)}
			>
				<Icon name="check" size={16} />
				<span>{t('jobs.card_apply')}</span>
			</Button>

			<Button
				variant="ghost"
				size="sm"
				disabled={job.status === 'discarded'}
				onclick={() => onDiscard(job)}
			>
				<Icon name="trash" size={16} />
				<span>{t('jobs.card_discard')}</span>
			</Button>
		</div>

		<div class="actions-right">
			<Button
				variant="ghost"
				size="sm"
				onclick={shareJob}
			>
				<Icon name="arrow-up-right" size={16} />
				<span>{t('jobs.card_share')}</span>
			</Button>
		</div>
	</footer>
</article>

<style>
	.job-feed-card {
		display: flex;
		flex-direction: column;
		justify-content: space-between;
		height: 100%;
		min-height: 100%;
		padding: var(--space-6);
		background: var(--color-surface);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-lg);
		box-shadow: var(--shadow-sm);
		box-sizing: border-box;
		gap: var(--space-4);
	}

	.card-header {
		display: flex;
		flex-direction: column;
		gap: var(--space-2);
		flex-shrink: 0;
	}

	.header-top {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: var(--space-2);
	}

	.company-row {
		display: flex;
		align-items: center;
		flex-wrap: wrap;
		gap: var(--space-2);
	}

	.company-name {
		font-weight: var(--weight-bold);
		font-size: var(--text-body);
		color: var(--color-text-secondary);
	}

	.position-badge {
		font-size: var(--text-caption);
		font-weight: var(--weight-medium);
		color: var(--color-text-muted);
		flex-shrink: 0;
	}

	.job-title {
		margin: 0;
		font-size: var(--text-h2);
		line-height: var(--leading-h2);
		font-weight: var(--weight-bold);
		color: var(--color-text-primary);
	}

	.source-link {
		display: inline-flex;
		align-items: center;
		gap: var(--space-1);
		font-size: var(--text-caption);
		color: var(--color-text-muted);
		text-decoration: none;
		max-width: 100%;
		overflow: hidden;
	}

	.source-link:hover {
		color: var(--color-primary);
		text-decoration: underline;
	}

	.source-url {
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.card-body {
		display: flex;
		flex-direction: column;
		flex: 1 1 auto;
		min-height: 0;
		gap: var(--space-3);
		justify-content: center;
	}

	.teaser-wrapper {
		position: relative;
		overflow: hidden;
		max-height: 14rem;
	}

	.teaser-gradient {
		position: absolute;
		left: 0;
		right: 0;
		bottom: 0;
		height: var(--space-8);
		background: linear-gradient(to bottom, transparent, var(--color-surface));
		pointer-events: none;
	}

	.read-more-row {
		display: flex;
		align-items: center;
	}

	.read-more-link {
		display: inline-flex;
		align-items: center;
		gap: var(--space-1);
		font-size: var(--text-body);
		font-weight: var(--weight-semibold);
		color: var(--color-primary);
		text-decoration: none;
		padding: var(--space-1) 0;
	}

	.read-more-link:hover {
		text-decoration: underline;
	}

	.card-footer {
		display: flex;
		align-items: center;
		justify-content: space-between;
		flex-wrap: wrap;
		gap: var(--space-2);
		flex-shrink: 0;
		padding-top: var(--space-3);
		border-top: 1px solid var(--color-border);
	}

	.actions-left,
	.actions-right {
		display: flex;
		align-items: center;
		flex-wrap: wrap;
		gap: var(--space-2);
	}

	@media (max-width: 767px) {
		.job-feed-card {
			padding: var(--space-4);
			gap: var(--space-3);
		}

		.job-title {
			font-size: var(--text-h3);
			line-height: var(--leading-h3);
		}

		.teaser-wrapper {
			max-height: 10rem;
		}
	}
</style>