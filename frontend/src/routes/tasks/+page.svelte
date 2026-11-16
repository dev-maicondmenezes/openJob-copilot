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

	// CP10 §4.4: abas por status + contadores, atualização em tempo real via
	// SSE (fallback polling), detalhe e ações (Cancelar / Retry).
	type Tab = 'all' | TaskStatus;
	const tabs: { key: Tab; label: string }[] = [
		{ key: 'all', label: 'Todas' },
		{ key: 'queued', label: 'Na fila' },
		{ key: 'running', label: 'Rodando' },
		{ key: 'done', label: 'Concluídas' },
		{ key: 'failed', label: 'Falharam' },
		{ key: 'cancelled', label: 'Canceladas' }
	];

	let tab: Tab = $state('all');
	let detail: TaskItem | null = $state(null);

	const typeLabel: Record<string, string> = {
		score_job: 'Score de vaga',
		extract_resume_keywords: 'Palavras-chave do currículo',
		tailor_resume: 'Adaptação de currículo'
	};

	const statusFilter = $derived(tab === 'all' ? '' : tab);
	const visible = $derived(
		tab === 'all' ? $tasks : $tasks.filter((t) => t.status === tab)
	);
	function countFor(key: TaskStatus): number {
		return $taskCounts[key] ?? 0;
	}

	onMount(() => {
		startTasksStream({ status: statusFilter });
		return () => stopTasksStream();
	});

	function switchTab(next: Tab): void {
		tab = next;
		stopTasksStream();
		startTasksStream({ status: statusFilter });
	}

	async function cancel(item: TaskItem): Promise<void> {
		if (await cancelTask(item.id)) void refreshQuiet();
	}
	async function retry(item: TaskItem): Promise<void> {
		if (await retryTask(item.id)) void refreshQuiet();
	}
	async function refreshQuiet(): Promise<void> {
		// startTasksStream já agenda refresh; força um imediato.
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

	const statusPill: Record<string, string> = {
		queued: 'Na fila',
		running: 'Rodando',
		done: 'Concluída',
		failed: 'Falhou',
		cancelled: 'Cancelada'
	};
</script>

<svelte:head>
	<title>Tarefas de IA · openJob</title>
</svelte:head>

<h1 class="page-title">Tarefas de IA</h1>
<p class="page-subtitle">
	Fila assíncrona: scoring de vagas, extração de palavras-chave e futuras tarefas.
	{#if $tasksStreamConnected}
		<span class="live">● em tempo real</span>
	{:else}
		<span class="live fallback">○ via polling</span>
	{/if}
</p>

<nav class="tabs" aria-label="Filtrar por status">
	{#each tabs as t (t.key)}
		<button class="tab" class:active={tab === t.key} onclick={() => switchTab(t.key)}>
			{t.label}
			{#if t.key !== 'all'}
				<span class="count">{countFor(t.key as TaskStatus)}</span>
			{/if}
		</button>
	{/each}
</nav>

{#if visible.length === 0}
	<p class="empty">
		Nenhuma tarefa {tab === 'all' ? '' : 'neste estado'}. Rodar “Buscar agora” em
		<a href="/jobs">Vagas</a> enfileira o scoring das vagas novas.
	</p>
{:else}
	<div class="list">
		{#each visible as task (task.id)}
			<article class="card">
				<header class="card-head">
					<div>
						<h2>{label(task)}</h2>
						<p class="muted">
							{elapsed(task)} · tentativas {task.attempts}/{task.max_attempts}
						</p>
					</div>
					<span class="pill st-{task.status}">{statusPill[task.status] ?? task.status}</span>
				</header>

				{#if task.status === 'running'}
					<progress value={task.progress} max="100"></progress>
				{:else if task.status === 'failed' && task.error}
					<p class="err">{task.error}</p>
				{/if}

				<footer class="actions">
					{#if task.status === 'queued' || task.status === 'running'}
						<button class="mini" onclick={() => cancel(task)}>Cancelar</button>
					{/if}
					{#if task.status === 'failed' || task.status === 'cancelled'}
						<button class="mini" onclick={() => retry(task)}>Retry</button>
					{/if}
					<button class="mini" onclick={() => (detail = task)}>Ver detalhe</button>
				</footer>
			</article>
		{/each}
	</div>
{/if}

{#if detail}
	<div
		class="backdrop"
		role="presentation"
		onclick={() => (detail = null)}
		onkeydown={(e) => e.key === 'Escape' && (detail = null)}
	>
		<div
			class="drawer"
			role="dialog"
			aria-label="Detalhe da tarefa"
			tabindex="-1"
			onclick={(e) => e.stopPropagation()}
			onkeydown={(e) => e.stopPropagation()}
		>
			<header class="drawer-head">
				<strong>{label(detail)}</strong>
				<button class="mini" onclick={() => (detail = null)}>Fechar</button>
			</header>
			<p class="muted">
				{statusPill[detail.status] ?? detail.status} · tentativas
				{detail.attempts}/{detail.max_attempts} · criada em
				{new Date(detail.created_at).toLocaleString('pt-BR')}
			</p>
			{#if detail.error}
				<h3>Erro</h3>
				<pre class="block err">{detail.error}</pre>
			{/if}
			<h3>Payload</h3>
			<pre class="block">{JSON.stringify(detail.payload, null, 2)}</pre>
			{#if detail.result !== null && detail.result !== undefined}
				<h3>Resultado</h3>
				<pre class="block">{fmtResult(detail.result)}</pre>
			{/if}
		</div>
	</div>
{/if}

<style>
	.live {
		font-size: 0.72rem;
		color: #047857;
		margin-left: 0.4rem;
	}

	.live.fallback {
		color: var(--text-muted);
	}

	.tabs {
		display: flex;
		gap: 0.35rem;
		flex-wrap: wrap;
		margin-bottom: 1rem;
	}

	.tab {
		display: inline-flex;
		align-items: center;
		gap: 0.35rem;
		border: 1px solid var(--border);
		background: var(--surface);
		color: var(--text-muted);
		border-radius: 999px;
		padding: 0.3rem 0.75rem;
		font-size: 0.8rem;
		cursor: pointer;
	}

	.tab.active {
		background: var(--accent-soft);
		color: var(--accent);
		border-color: var(--accent);
		font-weight: 600;
	}

	.count {
		background: var(--surface-soft);
		border-radius: 999px;
		padding: 0 0.4rem;
		font-size: 0.7rem;
	}

	.card {
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: var(--radius);
		padding: 0.9rem 1rem;
		margin-bottom: 0.75rem;
	}

	.card-head {
		display: flex;
		align-items: flex-start;
		justify-content: space-between;
		gap: 0.75rem;
	}

	.card-head h2 {
		font-size: 0.95rem;
		margin: 0;
	}

	.pill {
		font-size: 0.72rem;
		font-weight: 600;
		border-radius: 999px;
		padding: 0.15rem 0.6rem;
		background: var(--accent-soft);
		color: var(--accent);
		flex-shrink: 0;
	}

	.pill.st-running { background: #dbeafe; color: #1d4ed8; }
	.pill.st-done { background: #ecfdf5; color: #047857; }
	.pill.st-failed { background: #fef2f2; color: #b91c1c; }
	.pill.st-cancelled { background: #f3f4f6; color: #4b5563; }

	progress {
		width: 100%;
		height: 0.5rem;
		margin-top: 0.5rem;
	}

	.err {
		color: #b91c1c;
		font-size: 0.82rem;
		margin: 0.4rem 0 0;
		overflow-wrap: anywhere;
	}

	.actions {
		display: flex;
		gap: 0.5rem;
		margin-top: 0.6rem;
	}

	.mini {
		border: 1px solid var(--border);
		background: var(--surface);
		color: var(--text);
		border-radius: var(--radius);
		padding: 0.3rem 0.7rem;
		font-size: 0.78rem;
		cursor: pointer;
	}

	.muted {
		color: var(--text-muted);
		font-size: 0.78rem;
		margin: 0.15rem 0 0;
	}

	.empty {
		color: var(--text-muted);
		text-align: center;
		padding: 1.5rem 0;
	}

	.empty a {
		color: var(--accent);
		font-weight: 600;
	}

	.backdrop {
		position: fixed;
		inset: 0;
		background: rgba(0, 0, 0, 0.4);
		display: flex;
		align-items: flex-end;
		justify-content: center;
		z-index: 90;
	}

	.drawer {
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: var(--radius) var(--radius) 0 0;
		width: min(40rem, 100vw);
		max-height: 80vh;
		overflow-y: auto;
		padding: 1rem;
	}

	.drawer-head {
		display: flex;
		align-items: center;
		justify-content: space-between;
		margin-bottom: 0.25rem;
	}

	h3 {
		font-size: 0.8rem;
		text-transform: uppercase;
		color: var(--text-muted);
		margin: 0.9rem 0 0.25rem;
	}

	.block {
		background: var(--surface-soft);
		border: 1px solid var(--border);
		border-radius: var(--radius);
		padding: 0.6rem;
		font-size: 0.78rem;
		overflow-x: auto;
		white-space: pre-wrap;
		overflow-wrap: anywhere;
	}

	.block.err {
		color: #b91c1c;
	}
</style>
