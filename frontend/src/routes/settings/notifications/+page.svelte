<script lang="ts">
	import { onMount } from 'svelte';
	import SettingsNav from '#lib/components/SettingsNav.svelte';

	// CP10 §4.3/§4.4: configurações dos canais de notificação (in-app +
	// navegador) e do worker de tarefas (concorrência / intervalo de polling).
	interface NotifConfig {
		in_app_enabled: boolean;
		retention_days: number;
		browser_notifications: boolean;
	}
	interface TasksConfig {
		max_concurrency: number;
		worker_poll_ms: number;
	}

	let notif: NotifConfig = $state({ in_app_enabled: true, retention_days: 90, browser_notifications: false });
	let tasksCfg: TasksConfig = $state({ max_concurrency: 2, worker_poll_ms: 500 });
	let saving = $state(false);
	let message = $state<{ kind: 'success' | 'error'; text: string } | null>(null);
	let browserPerm: NotificationPermission | 'unsupported' = $state('default');

	onMount(async () => {
		await Promise.all([loadNotif(), loadTasks()]);
		if (typeof Notification === 'undefined') {
			browserPerm = 'unsupported';
		} else {
			browserPerm = Notification.permission;
		}
	});

	async function loadNotif(): Promise<void> {
		try {
			const res = await fetch('/api/settings/notifications');
			if (res.ok) notif = await res.json();
		} catch {
			// silencioso
		}
	}

	async function loadTasks(): Promise<void> {
		try {
			const res = await fetch('/api/settings/tasks');
			if (res.ok) tasksCfg = await res.json();
		} catch {
			// silencioso
		}
	}

	async function saveAll(): Promise<void> {
		saving = true;
		message = null;
		try {
			const [nRes, tRes] = await Promise.all([
				fetch('/api/settings/notifications', {
					method: 'PUT',
					headers: { 'Content-Type': 'application/json' },
					body: JSON.stringify(notif)
				}),
				fetch('/api/settings/tasks', {
					method: 'PUT',
					headers: { 'Content-Type': 'application/json' },
					body: JSON.stringify(tasksCfg)
				})
			]);
			if (nRes.ok && tRes.ok) {
				message = { kind: 'success', text: 'Configurações salvas.' };
			} else {
				const err = await (nRes.ok ? tRes : nRes).json().catch(() => ({}));
				message = { kind: 'error', text: err.error ?? 'Falha ao salvar configurações.' };
			}
		} catch {
			message = { kind: 'error', text: 'Falha de rede ao salvar configurações.' };
		} finally {
			saving = false;
		}
	}

	async function enableBrowserNotifications(): Promise<void> {
		if (typeof Notification === 'undefined') return;
		const perm = await Notification.requestPermission();
		browserPerm = perm;
		if (perm === 'granted') {
			notif.browser_notifications = true;
			void saveAll();
			new Notification('openJob', { body: 'Notificações do navegador ativadas.' });
		}
	}
</script>

<svelte:head>
	<title>Notificações & Tarefas · openJob</title>
</svelte:head>

<h1 class="page-title">Configurações</h1>
<SettingsNav active="notifications" />

<section class="card">
	<h2>Notificações</h2>

	<label class="row">
		<input type="checkbox" bind:checked={notif.in_app_enabled} />
		<span>
			Sino in-app ativado
			<small>Feed persistido em /notifications. Desativar não afeta o Telegram (quando configurado).</small>
		</span>
	</label>

	<label class="row">
		<span>
			Retenção (dias)
			<small>Notificações lidas há mais tempo que isso são apagadas. Não lidas nunca são.</small>
		</span>
		<input
			type="number"
			min="1"
			max="3650"
			bind:value={notif.retention_days}
			class="num"
		/>
	</label>

	<label class="row">
		<input
			type="checkbox"
			bind:checked={notif.browser_notifications}
			disabled={browserPerm === 'unsupported' || browserPerm === 'denied'}
		/>
		<span>
			Notificações do navegador
			<small>
				{#if browserPerm === 'unsupported'}
					Sem suporte neste navegador.
				{:else if browserPerm === 'denied'}
					Permissão negada nas configurações do navegador.
				{:else if browserPerm === 'granted'}
					Dispara só quando a aba está em background (best-effort).
				{:else}
					Precisa de permissão do navegador.
				{/if}
			</small>
		</span>
	</label>

	{#if browserPerm === 'default'}
		<button class="btn" onclick={enableBrowserNotifications}>Permitir notificações do navegador</button>
	{/if}

	{#if !notif.in_app_enabled}
		<p class="warn">⚠ Sino in-app desligado — novas notificações não serão gravadas.</p>
	{/if}
</section>

<section class="card">
	<h2>Tarefas de IA (worker)</h2>

	<label class="row">
		<span>
			Concorrência máxima
			<small>1–8 workers em paralelo (afeta imediatamente, sem restart).</small>
		</span>
		<input type="number" min="1" max="8" bind:value={tasksCfg.max_concurrency} class="num" />
	</label>

	<label class="row">
		<span>
			Intervalo de polling (ms)
			<small>100–5000ms entre varreduras da fila.</small>
		</span>
		<input type="number" min="100" max="5000" step="100" bind:value={tasksCfg.worker_poll_ms} class="num" />
	</label>

	<p class="muted">
		Scoring de vagas compartilha um único limite de requisições por provedor de IA —
		aumentar a concorrência não estoura o rate limit do seu BYOK.
	</p>
</section>

<div class="save-area">
	<button class="btn primary" disabled={saving} onclick={saveAll}>
		{saving ? 'Salvando…' : 'Salvar'}
	</button>
	{#if message}
		<span class="msg {message.kind}">{message.text}</span>
	{/if}
</div>

<style>
	.card {
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: var(--radius);
		padding: 1rem;
		margin-bottom: 1rem;
	}

	.card h2 {
		font-size: 1rem;
		margin: 0 0 0.75rem;
	}

	.row {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 1rem;
		padding: 0.5rem 0;
		font-size: 0.9rem;
	}

	.row span {
		display: flex;
		flex-direction: column;
		gap: 0.15rem;
	}

	.row small {
		color: var(--text-muted);
		font-size: 0.75rem;
	}

	.num {
		width: 6rem;
		padding: 0.4rem 0.6rem;
		border: 1px solid var(--border);
		border-radius: var(--radius);
		background: var(--surface-soft);
		color: var(--text);
	}

	.btn {
		border: 1px solid var(--border);
		background: var(--surface);
		color: var(--text);
		border-radius: var(--radius);
		padding: 0.5rem 1rem;
		font-size: 0.9rem;
		cursor: pointer;
		margin-top: 0.5rem;
	}

	.btn.primary {
		background: var(--accent);
		border-color: var(--accent);
		color: #fff;
		font-weight: 600;
	}

	.btn:disabled {
		opacity: 0.6;
		cursor: default;
	}

	.warn {
		color: #b45309;
		background: #fffbeb;
		border-radius: var(--radius);
		padding: 0.5rem 0.7rem;
		font-size: 0.82rem;
		margin: 0.5rem 0 0;
	}

	.muted {
		color: var(--text-muted);
		font-size: 0.78rem;
		margin: 0.5rem 0 0;
	}

	.save-area {
		display: flex;
		align-items: center;
		gap: 0.75rem;
	}

	.msg.success {
		color: #047857;
		font-size: 0.85rem;
	}

	.msg.error {
		color: #b91c1c;
		font-size: 0.85rem;
	}
</style>
