<script lang="ts">
	import { onMount } from 'svelte';
	import {
		tasks,
		taskCounts,
		tasksStreamConnected,
		startTasksStream,
		stopTasksStream,
		cancelTask,
		retryTask,
		type TaskStatus,
		type TaskItem
	} from '#lib/tasks.svelte.ts';
	import PageHeader from '#lib/ui/PageHeader.svelte';
	import Card from '#lib/ui/Card.svelte';
	import Badge from '#lib/ui/Badge.svelte';
	import Tabs from '#lib/ui/Tabs.svelte';
	import Button from '#lib/ui/Button.svelte';
	import ProgressBar from '#lib/ui/ProgressBar.svelte';
	import Drawer from '#lib/ui/Drawer.svelte';
	import EmptyState from '#lib/ui/EmptyState.svelte';
	import DataView from '#lib/ui/DataView.svelte';
	import type { DataColumn } from '#lib/ui/data.js';

	type Tab = 'all' | TaskStatus;
	const tabs: { key: Tab; label: string }[] = [
		{ key: 'all', label: 'Todas' },
		{ key: 'queued', label: 'Na fila' },
		{ key: 'running', label: 'Rodando' },
		{ key: 'done', label: 'Concluídas' },
		{ key: 'failed', label: 'Falharam' },
		{ key: 'cancelled', label: 'Canceladas' }
	];

	let tab = $state<Tab>('all');
	let detail: TaskItem | null = $state(null);

	const typeLabel: Record<string, string> = {
		score_job: 'Score de vaga',
		extract_resume_keywords: 'Palavras-chave do currículo',
		tailor_resume: 'Adaptação de currículo'
	};

	const statusFilter = $derived(tab === 'all' ? '' : tab);
	const visible = $derived(tab === 'all' ? $tasks : $tasks.filter((task) => task.status === tab));
	const tabItems = $derived(
		tabs.map((item) => ({
			value: item.key,
			label: item.label,
			count: item.key === 'all' ? $tasks.length : $taskCounts[item.key as TaskStatus] ?? 0
		}))
	);

	const statusPill: Record<string, string> = {
		queued: 'Na fila',
		running: 'Rodando',
		done: 'Concluída',
		failed: 'Falhou',
		cancelled: 'Cancelada'
	};

	const columns: DataColumn<TaskItem>[] = [
		{ key: 'label', header: 'Tarefa', cardLabel: 'Tarefa', value: (task) => label(task) },
		{ key: 'status', header: 'Status', value: (task) => statusPill[task.status] ?? task.status },
		{ key: 'progress', header: 'Progresso', value: (task) => `${task.progress}%` },
		{ key: 'attempts', header: 'Tentativas', value: (task) => `${task.attempts}/${task.max_attempts}` },
		{ key: 'elapsed', header: 'Tempo', value: (task) => elapsed(task) },
		{ key: 'actions', header: 'Ações', value: () => '' }
	];

	onMount(() => {
		startTasksStream({ status: statusFilter });
		return () => stopTasksStream();
	});

	function switchTab(next: string): void {
		tab = next as Tab;
		stopTasksStream();
		startTasksStream({ status: statusFilter });
	}

	async function cancel(item: TaskItem): Promise<void> {
		if (await cancelTask(item.id)) refreshQuiet();
	}
	async function retry(item: TaskItem): Promise<void> {
		if (await retryTask(item.id)) refreshQuiet();
	}
	function refreshQuiet(): void {
		stopTasksStream();
		startTasksStream({ status: statusFilter });
	}

	function label(task: TaskItem): string {
		const p = task.payload as { job_label?: string; job_id?: number } | null;
		if (task.type === 'score_job' && p?.job_label) return `Score da vaga: ${p.job_label}`;
		if (task.type === 'score_job' && p?.job_id) return `Score da vaga #${p.job_id}`;
		return typeLabel[task.type] ?? task.type;
	}

	function elapsed(task: TaskItem): string {
		const start = task.started_at ?? task.created_at;
		const end = task.finished_at ?? new Date().toISOString();
		const secs = Math.max(0, Math.round((new Date(end).getTime() - new Date(start).getTime()) / 1000));
		if (secs < 60) return `${secs}s`;
		return `${Math.floor(secs / 60)}m${secs % 60}s`;
	}

	function fmtResult(result: unknown): string {
		if (result === null || result === undefined) return '';
		return typeof result === 'string' ? result : JSON.stringify(result, null, 2);
	}
</script>

<svelte:head>
	<title>Tarefas de IA · openJob</title>
</svelte:head>

{#snippet renderCell(task: TaskItem, column: DataColumn<TaskItem>)}
	{#if column.key === 'label'}
		<strong>{label(task)}</strong>
	{:else if column.key === 'status'}
		<Badge tone={task.status === 'done' ? 'success' : task.status === 'failed' ? 'error' : task.status === 'running' ? 'info' : 'neutral'}>
			{statusPill[task.status] ?? task.status}
		</Badge>
	{:else if column.key === 'progress'}
		{#if task.status === 'running'}
			<ProgressBar value={task.progress} showValue />
		{:else}
			<span class="muted">—</span>
		{/if}
	{:else if column.key === 'actions'}
		<div class="row-actions">
			{#if task.status === 'queued' || task.status === 'running'}
				<Button variant="ghost" size="sm" onclick={() => cancel(task)}>Cancelar</Button>
			{/if}
			{#if task.status === 'failed' || task.status === 'cancelled'}
				<Button variant="ghost" size="sm" onclick={() => retry(task)}>Retry</Button>
			{/if}
			<Button variant="secondary" size="sm" onclick={() => (detail = task)}>Ver detalhe</Button>
		</div>
	{:else}
		{column.value?.(task) ?? ''}
	{/if}
{/snippet}

<PageHeader
	title="Tarefas de IA"
	subtitle="Fila assíncrona: scoring de vagas, extração de palavras-chave e futuras tarefas."
	breadcrumb={[{ label: 'Início', href: '/' }, { label: 'Tarefas' }]}
>
	{#snippet actions()}
		<Badge tone={$tasksStreamConnected ? 'success' : 'neutral'}>
			{$tasksStreamConnected ? '● em tempo real' : '○ via polling'}
		</Badge>
	{/snippet}
</PageHeader>

<Tabs items={tabItems} value={tab} label="Filtrar tarefas por status" onchange={switchTab} />

{#if visible.length === 0}
	<Card>
		<EmptyState
			title={`Nenhuma tarefa ${tab === 'all' ? '' : 'neste estado'}`}
			description="Rodar “Buscar agora” em Vagas enfileira o scoring das vagas novas."
		/>
	</Card>
{:else}
	<DataView
		columns={columns}
		rows={visible}
		getKey={(task) => task.id}
		caption="Tarefas de IA"
		render={renderCell}
	/>
{/if}

{#if detail}
	<Drawer
		title={label(detail)}
		description={`${statusPill[detail.status] ?? detail.status} · ${detail.attempts}/${detail.max_attempts} tentativas`}
		onclose={() => (detail = null)}
	>
		{#if detail.status === 'running'}
			<ProgressBar value={detail.progress} label="Progresso" showValue />
		{/if}
		{#if detail.error}
			<div class="block error">{detail.error}</div>
		{/if}
		<div class="detail-group">
			<h3>Payload</h3>
			<pre class="block">{JSON.stringify(detail.payload, null, 2)}</pre>
		</div>
		{#if detail.result !== null && detail.result !== undefined}
			<div class="detail-group">
				<h3>Resultado</h3>
				<pre class="block">{fmtResult(detail.result)}</pre>
			</div>
		{/if}
		<p class="muted">Criada em {new Date(detail.created_at).toLocaleString('pt-BR')}</p>
		{#snippet footer()}
			{#if detail?.status === 'failed' || detail?.status === 'cancelled'}
				<Button variant="secondary" onclick={() => retry(detail!)}>Retry</Button>
			{/if}
			<Button variant="ghost" onclick={() => (detail = null)}>Fechar</Button>
		{/snippet}
	</Drawer>
{/if}

<style>
	.row-actions {
		display: flex;
		gap: var(--space-2);
		flex-wrap: wrap;
	}

	.muted {
		color: var(--color-text-muted);
		font-size: var(--text-caption);
	}

	.detail-group {
		display: flex;
		flex-direction: column;
		gap: var(--space-2);
	}

	h3 {
		margin: 0;
		font-size: var(--text-caption);
		text-transform: uppercase;
		letter-spacing: 0.04em;
		color: var(--color-text-muted);
	}

	.block {
		margin: 0;
		padding: var(--space-3);
		background: var(--color-surface-soft);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-md);
		font-family: var(--font-mono);
		font-size: var(--text-mono);
		white-space: pre-wrap;
		overflow-wrap: anywhere;
	}

	.block.error {
		color: var(--color-error);
		background: var(--color-error-soft);
		border-color: var(--color-error-border);
	}
</style>
