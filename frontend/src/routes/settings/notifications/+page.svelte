<script lang="ts">
	import { onMount } from 'svelte';
	import SettingsNav from '#lib/components/SettingsNav.svelte';
	import PageHeader from '#lib/ui/PageHeader.svelte';
	import Card from '#lib/ui/Card.svelte';
	import Field from '#lib/ui/Field.svelte';
	import Input from '#lib/ui/Input.svelte';
	import Switch from '#lib/ui/Switch.svelte';
	import Button from '#lib/ui/Button.svelte';
	import Alert from '#lib/ui/Alert.svelte';

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
	let message: { kind: 'success' | 'error'; text: string } | null = $state(null);
	let browserPerm: NotificationPermission | 'unsupported' = $state('default');

	onMount(async () => {
		await Promise.all([loadNotif(), loadTasks()]);
		browserPerm = typeof Notification === 'undefined' ? 'unsupported' : Notification.permission;
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

<PageHeader
	title="Configurações"
	subtitle="Canais de notificação e parâmetros do worker de tarefas de IA."
	breadcrumb={[{ label: 'Início', href: '/' }, { label: 'Configurações', href: '/settings' }, { label: 'Notificações & Tarefas' }]}
/>

<SettingsNav active="notifications" />

{#if message}
	<Alert severity={message.kind}>{message.text}</Alert>
{/if}

<div class="settings-grid">
	<Card title="Notificações" subtitle="Sino in-app, retenção e notificações do navegador.">
		<div class="switches">
			<Switch
				label="Sino in-app ativado"
				description="Feed persistido em /notifications. Desativar não afeta o Telegram."
				bind:checked={notif.in_app_enabled}
			/>
			<Field label="Retenção (dias)" for="retention_days" hint="Notificações lidas há mais tempo que isso são apagadas. Não lidas nunca são.">
				<Input id="retention_days" type="number" min="1" max="3650" bind:value={notif.retention_days} />
			</Field>
			<Switch
				label="Notificações do navegador"
				description={browserPerm === 'unsupported'
					? 'Sem suporte neste navegador.'
					: browserPerm === 'denied'
						? 'Permissão negada nas configurações do navegador.'
						: browserPerm === 'granted'
							? 'Dispara só com a aba em background (best-effort).'
							: 'Precisa de permissão do navegador.'}
				bind:checked={notif.browser_notifications}
				disabled={browserPerm === 'unsupported' || browserPerm === 'denied'}
			/>
		</div>
		{#if browserPerm === 'default'}
			<Button variant="secondary" onclick={enableBrowserNotifications}>Permitir notificações do navegador</Button>
		{/if}
		{#if !notif.in_app_enabled}
			<Alert severity="warning" title="Sino in-app desligado">Novas notificações não serão gravadas.</Alert>
		{/if}
	</Card>

	<Card title="Tarefas de IA (worker)" subtitle="Concorrência e intervalo de varredura da fila.">
		<div class="switches">
			<Field label="Concorrência máxima" for="max_concurrency" hint="1 a 8 workers em paralelo (1–8). Afeta imediatamente, sem restart.">
				<Input id="max_concurrency" type="number" min="1" max="8" bind:value={tasksCfg.max_concurrency} />
			</Field>
			<Field label="Intervalo de polling (ms)" for="worker_poll_ms" hint="Entre 100 e 5000ms entre varreduras da fila.">
				<Input id="worker_poll_ms" type="number" min="100" max="5000" step="100" bind:value={tasksCfg.worker_poll_ms} />
			</Field>
		</div>
		<p class="hint">
			O scoring compartilha um único limite de requisições por provedor de IA — aumentar a concorrência
			não estoura o rate limit do seu BYOK.
		</p>
	</Card>
</div>

<div class="form-actions">
	<Button loading={saving} onclick={saveAll}>Salvar</Button>
</div>

<style>
	.settings-grid {
		display: grid;
		grid-template-columns: 1fr;
		gap: var(--space-4);
	}

	.switches {
		display: flex;
		flex-direction: column;
		gap: var(--space-4);
		margin-bottom: var(--space-4);
	}

	.form-actions {
		margin-top: var(--space-6);
		margin-bottom: var(--space-12);
	}

	@media (min-width: 768px) {
		.settings-grid {
			grid-template-columns: repeat(2, minmax(0, 1fr));
		}
	}
</style>
