<script lang="ts">
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
			if (!res.ok) throw new Error(`Erro ${res.status}`);
			const body = await res.json();
			job = (body.jobs as Job[] | undefined)?.find((item) => item.id === jobId) ?? null;
			if (!job) error = 'Vaga não encontrada. Ela pode ter sido removida.';
		} catch {
			error = 'Falha de rede ao carregar a vaga.';
		} finally {
			loading = false;
		}
	}

	async function act(action: 'score' | 'approve' | 'discard'): Promise<void> {
		if (!job) return;
		busy = true;
		try {
			const res = await fetch(`/api/jobs/${job.id}/${action}`, { method: 'POST' });
			if (!res.ok) {
				const body = await res.json().catch(() => ({}));
				error = body.error ?? `Falha ao executar ${action}.`;
				return;
			}
			if (action === 'approve') {
				await goto('/applications');
				return;
			}
			await load();
		} catch {
			error = `Falha de rede ao executar ${action}.`;
		} finally {
			busy = false;
		}
	}
</script>

<svelte:head>
	<title>Detalhe da vaga · openJob</title>
</svelte:head>

<PageHeader
	title="Detalhe da vaga"
	subtitle="A URL é a fonte da verdade — este conteúdo abre direto pelo link."
	breadcrumb={[{ label: 'Início', href: '/' }, { label: 'Vagas', href: '/jobs' }, { label: `#${jobId}` }]}
>
	{#snippet actions()}
		<LinkButton href="/jobs" variant="ghost">
			<Icon name="arrow-left" size={20} /> Voltar para vagas
		</LinkButton>
	{/snippet}
</PageHeader>

{#if error}
	<Alert severity="error" title="Não foi possível abrir a vaga">{error}</Alert>
{/if}

{#if loading}
	<Card><p class="hint">Carregando vaga…</p></Card>
{:else if job}
	<Card>
		<JobDetail {job}>
			{#snippet actions()}
				<Button variant="secondary" loading={busy} onclick={() => act('score')}>Calcular score</Button>
				<Button loading={busy} disabled={job!.status !== 'new'} onclick={() => act('approve')}>Aprovar</Button>
				<Button variant="danger" loading={busy} disabled={job!.status === 'discarded'} onclick={() => act('discard')}>Descartar</Button>
			{/snippet}
		</JobDetail>
	</Card>
{:else}
	<Card>
		<EmptyState title="Vaga não encontrada" description="Volte para a lista de vagas e tente novamente.">
			<Button onclick={() => goto('/jobs')}>Ver vagas</Button>
		</EmptyState>
	</Card>
{/if}

<style>
</style>
