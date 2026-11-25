<script lang="ts">
	import { t } from '#lib/i18n/t.js';
	import { errorText } from '#lib/i18n/errors.js';
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import PageHeader from '#lib/ui/PageHeader.svelte';
	import Card from '#lib/ui/Card.svelte';
	import Button from '#lib/ui/Button.svelte';
	import LinkButton from '#lib/ui/LinkButton.svelte';
	import Alert from '#lib/ui/Alert.svelte';
	import EmptyState from '#lib/ui/EmptyState.svelte';
	import Icon from '#lib/ui/Icon.svelte';
	import JobDetail from '#lib/components/JobDetail.svelte';
	import type { Job } from '#lib/types/job.js';

	const jobId = $derived(Number(page.params.id));
	let job = $state<Job | null>(null);
	let loading = $state(true);
	let error = $state('');
	let busy = $state(false);

	onMount(() => void load());

	async function load(): Promise<void> {
		loading = true;
		error = '';
		try {
			const res = await fetch('/api/jobs');
			if (!res.ok) throw new Error(t('jobdetail.error_http', { status: res.status }));
			const body = await res.json();
			job = (body.jobs as Job[] | undefined)?.find((item) => item.id === jobId) ?? null;
			if (!job) error = t('jobdetail.error_not_found');
		} catch {
			error = t('jobdetail.error_network');
		} finally {
			loading = false;
		}
	}

	async function act(action: 'approve' | 'discard'): Promise<void> {
		if (!job) return;
		busy = true;
		try {
			const res = await fetch(`/api/jobs/${job.id}/${action}`, { method: 'POST' });
			if (!res.ok) {
				const body = await res.json().catch(() => ({}));
				error = errorText(body, t('jobdetail.error_action', { action }));
				return;
			}
			if (action === 'approve') {
				await goto('/applications');
				return;
			}
			await load();
		} catch {
			error = t('jobdetail.error_action_network', { action });
		} finally {
			busy = false;
		}
	}
</script>

<svelte:head>
	<title>{t('jobdetail.meta_title')}</title>
</svelte:head>

<PageHeader
	title={t('jobs.dialog.detail_title')}
	subtitle={t('jobdetail.subtitle')}
	breadcrumb={[{ label: t('common.home'), href: '/' }, { label: t('common.jobs'), href: '/jobs' }, { label: `#${jobId}` }]}
>
	{#snippet actions()}
		<LinkButton href="/jobs" variant="ghost">
			<Icon name="arrow-left" size={20} /> {t('jobdetail.back')}
		</LinkButton>
	{/snippet}
</PageHeader>

{#if error}
	<Alert severity="error" title={t('jobdetail.cannot_open')}>{error}</Alert>
{/if}

{#if loading}
	<Card><p class="hint">{t('jobdetail.loading')}</p></Card>
{:else if job}
	<Card>
		<JobDetail {job}>
			{#snippet actions()}
				<Button loading={busy} disabled={job!.status !== 'new'} onclick={() => act('approve')}>{t('jobdetail.approve')}</Button>
				<Button variant="danger" loading={busy} disabled={job!.status === 'discarded'} onclick={() => act('discard')}>{t('jobdetail.discard')}</Button>
			{/snippet}
		</JobDetail>
	</Card>
{:else}
	<Card>
		<EmptyState title={t('jobdetail.not_found')} description={t('jobdetail.not_found_desc')}>
			<Button onclick={() => goto('/jobs')}>{t('jobdetail.see_jobs')}</Button>
		</EmptyState>
	</Card>
{/if}

<style>
</style>
