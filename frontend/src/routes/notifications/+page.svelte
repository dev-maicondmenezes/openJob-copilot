<script lang="ts">
	import { onMount } from 'svelte';
	import { markNotificationRead, markAllNotificationsRead, unreadCount, type NotificationItem } from '#lib/notifications.svelte.ts';

	// CP10 §4.4: feed completo com filtros por type/severity, "somente não
	// lidas" e botão de limpar lidas.
	const types = [
		{ value: '', label: 'Todos os tipos' },
		{ value: 'application_status', label: 'Candidaturas' },
		{ value: 'discovery_result', label: 'Busca de vagas' },
		{ value: 'task_completed', label: 'Tarefas concluídas' },
		{ value: 'task_failed', label: 'Tarefas com falha' },
		{ value: 'source_error', label: 'Erros de fonte' }
	];
	const severities = [
		{ value: '', label: 'Toda severidade' },
		{ value: 'info', label: 'Info' },
		{ value: 'success', label: 'Sucesso' },
		{ value: 'warning', label: 'Aviso' },
		{ value: 'error', label: 'Erro' }
	];

	let items: NotificationItem[] = $state([]);
	let cursor = $state<string | null>(null);
	let loading = $state(true);
	let error = $state('');
	let typeFilter = $state('');
	let severityFilter = $state('');
	let unreadOnly = $state(false);

	const sevIcon: Record<string, string> = { success: '✔', warning: '⚠', error: '✖', info: 'ℹ' };

	async function load(reset = true): Promise<void> {
		if (reset) {
			cursor = null;
			items = [];
		}
		loading = true;
		error = '';
		try {
			const params = new URLSearchParams();
			if (typeFilter) params.set('type', typeFilter);
			if (severityFilter) params.set('severity', severityFilter);
			if (unreadOnly) params.set('unread', '1');
			if (cursor) params.set('cursor', cursor);
			params.set('limit', '50');
			const res = await fetch(`/api/notifications?${params}`);
			if (res.ok) {
				const data = await res.json();
				items = reset ? (data.notifications ?? []) : [...items, ...(data.notifications ?? [])];
				cursor = data.next_cursor || null;
			} else {
				const err = await res.json().catch(() => ({}));
				error = err.error ?? `Erro ${res.status} ao carregar notificações.`;
			}
		} catch {
			error = 'Falha de rede ao carregar notificações.';
		} finally {
			loading = false;
		}
	}

	async function openItem(item: NotificationItem): Promise<void> {
		if (item.read_at === null) {
			await markNotificationRead(item.id);
			item.read_at = new Date().toISOString();
		}
		if (item.link) window.location.assign(item.link);
	}

	async function clearRead(): Promise<void> {
		await fetch('/api/notifications', { method: 'DELETE' });
		await load();
	}

	function fmtWhen(iso: string): string {
		return new Date(iso).toLocaleString('pt-BR', {
			day: '2-digit',
			month: 'short',
			hour: '2-digit',
			minute: '2-digit'
		});
	}

	function refilter(): void {
		void load();
	}

	onMount(() => void load());
</script>

<svelte:head>
	<title>Notificações · openJob</title>
</svelte:head>

<h1 class="page-title">Notificações</h1>
<p class="page-subtitle">
	Tudo que aconteceu com suas candidaturas, buscas e tarefas de IA.
</p>

<div class="filters">
	<select bind:value={typeFilter} onchange={refilter}>
		{#each types as t}<option value={t.value}>{t.label}</option>{/each}
	</select>
	<select bind:value={severityFilter} onchange={refilter}>
		{#each severities as s}<option value={s.value}>{s.label}</option>{/each}
	</select>
	<label class="check">
		<input type="checkbox" bind:checked={unreadOnly} onchange={refilter} />
		Somente não lidas
	</label>
</div>

<div class="actions">
	<span class="pill">{$unreadCount} não lida{$unreadCount === 1 ? '' : 's'}</span>
	<button class="btn ghost" onclick={() => markAllNotificationsRead().then(() => load())}>
		Marcar todas como lidas
	</button>
	<button class="btn ghost" onclick={clearRead}>Limpar lidas</button>
</div>

{#if error}
	<div class="status error" role="status">{error}</div>
{/if}

{#if loading && items.length === 0}
	<p class="empty">Carregando notificações…</p>
{:else if items.length === 0}
	<div class="empty-state">
		<p><strong>Nenhuma notificação por aqui.</strong></p>
		<p>
			Elas aparecem quando uma candidatura muda de status, uma busca automática
			termina ou uma tarefa de IA conclui/falha.
		</p>
		<p><a href="/settings/notifications">Configurar notificações</a></p>
	</div>
{:else}
	<div class="list">
		{#each items as item (item.id)}
			<button class="card sev-{item.severity}" class:unread={item.read_at === null} onclick={() => openItem(item)}>
				<header class="card-head">
					<div>
						<h2>{sevIcon[item.severity] ?? 'ℹ'} {item.title}</h2>
						<p class="muted">{fmtWhen(item.created_at)}</p>
					</div>
					{#if item.read_at === null}
						<span class="dot" title="Não lida"></span>
					{/if}
				</header>
				<p class="body">{item.body}</p>
			</button>
		{/each}
	</div>

	{#if cursor}
		<button class="btn" disabled={loading} onclick={() => load(false)}>
			{loading ? 'Carregando…' : 'Carregar mais'}
		</button>
	{/if}
{/if}

<style>
	.filters {
		display: flex;
		gap: 0.5rem;
		align-items: center;
		flex-wrap: wrap;
		margin-bottom: 0.75rem;
	}

	.filters select {
		padding: 0.45rem 0.6rem;
		border: 1px solid var(--border);
		border-radius: var(--radius);
		background: var(--surface);
		color: var(--text);
		font-size: 0.85rem;
	}

	.check {
		display: inline-flex;
		gap: 0.35rem;
		align-items: center;
		font-size: 0.85rem;
		color: var(--text-muted);
	}

	.actions {
		display: flex;
		gap: 0.5rem;
		align-items: center;
		margin-bottom: 1rem;
		flex-wrap: wrap;
	}

	.btn {
		border: 1px solid var(--border);
		background: var(--surface);
		color: var(--text);
		border-radius: var(--radius);
		padding: 0.45rem 0.8rem;
		font-size: 0.85rem;
		cursor: pointer;
	}

	.btn.ghost {
		border: none;
		background: none;
		color: var(--accent);
		padding: 0.2rem 0;
	}

	.card {
		display: block;
		width: 100%;
		text-align: left;
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: var(--radius);
		padding: 0.9rem 1rem;
		margin-bottom: 0.75rem;
		cursor: pointer;
		color: var(--text);
	}

	.card.unread {
		border-color: var(--accent);
		background: var(--accent-soft);
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

	.dot {
		width: 0.55rem;
		height: 0.55rem;
		border-radius: 999px;
		background: var(--accent);
		flex-shrink: 0;
		margin-top: 0.3rem;
	}

	.body {
		color: var(--text-muted);
		font-size: 0.85rem;
		margin: 0.3rem 0 0;
	}

	.muted {
		color: var(--text-muted);
		font-size: 0.75rem;
		margin: 0.1rem 0 0;
	}

	.status.error {
		background: #fef2f2;
		color: #b91c1c;
		border-radius: var(--radius);
		padding: 0.6rem 0.8rem;
		margin-bottom: 0.75rem;
	}

	.empty-state {
		text-align: center;
		color: var(--text-muted);
		padding: 1.5rem 0;
		font-size: 0.9rem;
	}

	.empty-state a {
		color: var(--accent);
		font-weight: 600;
	}

	.empty {
		color: var(--text-muted);
		text-align: center;
		padding: 1.5rem 0;
	}

	.pill {
		background: var(--accent-soft);
		color: var(--accent);
		border-radius: 999px;
		padding: 0.15rem 0.6rem;
		font-size: 0.75rem;
		font-weight: 600;
	}
</style>
