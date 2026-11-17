<script lang="ts">
	import { t } from '#lib/i18n/t.js';
	import { errorText } from '#lib/i18n/errors.js';
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
				message = { kind: 'success', text: t('notif_settings.saved') };
			} else {
				const err = await (nRes.ok ? tRes : nRes).json().catch(() => ({}));
				message = { kind: 'error', text: errorText(err, t('discovery.error.save')) };
			}
		} catch {
			message = { kind: 'error', text: t('discovery.error.save_network') };
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
			new Notification(t('common.app_name'), { body: t('notif_settings.browser_enabled') });
		}
	}
</script>

<svelte:head>
	<title>{t('notif_settings.meta_title')}</title>
</svelte:head>

<PageHeader
	title={t('common.settings')}
	subtitle={t('notif_settings.subtitle_full')}
	breadcrumb={[{ label: t('common.home'), href: '/' }, { label: t('common.settings'), href: '/settings' }, { label: t('nav.settings_notifications') }]}
/>

<SettingsNav active="notifications" />

{#if message}
	<Alert severity={message.kind}>{message.text}</Alert>
{/if}

<div class="settings-grid">
	<Card title={t('common.notifications')} subtitle={t('notif_settings.subtitle')}>
		<div class="switches">
			<Switch
				label={t('notif_settings.in_app')}
				description={t('notif_settings.in_app_desc')}
				bind:checked={notif.in_app_enabled}
			/>
			<Field label={t('notif_settings.retention')} for="retention_days" hint={t('notif_settings.retention_hint_full')}>
				<input id="retention_days" type="number" min="1" max="3650" bind:value={notif.retention_days} />
			</Field>
			<Switch
				label={t('notif_settings.browser')}
				description={browserPerm === 'unsupported'
					? t('notif_settings.browser.unsupported')
					: browserPerm === 'denied'
						? t('notif_settings.browser.denied')
						: browserPerm === 'granted'
							? t('notif_settings.browser.background')
							: t('notif_settings.browser.needs_permission')}
				bind:checked={notif.browser_notifications}
				disabled={browserPerm === 'unsupported' || browserPerm === 'denied'}
			/>
		</div>
		{#if browserPerm === 'default'}
			<Button variant="secondary" onclick={enableBrowserNotifications}>{t('notif_settings.browser.allow')}</Button>
		{/if}
		{#if !notif.in_app_enabled}
			<Alert severity="warning" title={t('notif_settings.in_app_off_title')}>{t('notif_settings.in_app_off_desc')}</Alert>
		{/if}
	</Card>

	<Card title={t('notif_settings.tasks_title')} subtitle={t('notif_settings.tasks_subtitle_full')}>
		<div class="switches">
			<Field label={t('notif_settings.concurrency')} for="max_concurrency" hint={t('notif_settings.concurrency_hint_full')}>
				<input id="max_concurrency" type="number" min="1" max="8" bind:value={tasksCfg.max_concurrency} />
			</Field>
			<Field label={t('notif_settings.poll_interval')} for="worker_poll_ms" hint={t('notif_settings.poll_hint_full')}>
				<input id="worker_poll_ms" type="number" min="100" max="5000" step="100" bind:value={tasksCfg.worker_poll_ms} />
			</Field>
		</div>
		<p class="hint">{t('notif_settings.rate_limit_note')}</p>
	</Card>
</div>

<div class="form-actions">
	<Button loading={saving} onclick={saveAll}>{t('common.save')}</Button>
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
