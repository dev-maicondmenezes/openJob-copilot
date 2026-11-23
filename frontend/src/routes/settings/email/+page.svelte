<script lang="ts">
	import { t } from '#lib/i18n/t.js';
	import { errorText } from '#lib/i18n/errors.js';
	import { onMount } from 'svelte';
	import { toast } from '#lib/toast.svelte.ts';
	import { formatDate } from '#lib/i18n/format.js';
	import SettingsNav from '#lib/components/SettingsNav.svelte';
	import PageHeader from '#lib/ui/PageHeader.svelte';
	import Card from '#lib/ui/Card.svelte';
	import Field from '#lib/ui/Field.svelte';
	import Button from '#lib/ui/Button.svelte';
	import Alert from '#lib/ui/Alert.svelte';
	import Badge from '#lib/ui/Badge.svelte';
	import Checkbox from '#lib/ui/Checkbox.svelte';
	import SegmentedControl from '#lib/ui/SegmentedControl.svelte';

	interface EmailAccountData {
		id: string;
		provider: string;
		address: string;
		folder: string;
		enabled: boolean;
		has_auth_config: boolean;
		last_sync_at: string | null;
		last_error: string;
	}

	let provider = $state<'imap' | 'gmail'>('imap');
	let address = $state('');
	let folder = $state('INBOX');
	let enabled = $state(false);
	let interval = $state('5m');
	let hasAuthConfig = $state(false);
	let lastSyncAt = $state<string | null>(null);
	let lastError = $state('');

	// IMAP specific
	let imapHost = $state('');
	let imapPort = $state(993);
	let imapUser = $state('');
	let imapPassword = $state('');
	let imapUseTLS = $state(true);

	// Gmail specific
	let gmailClientID = $state('');
	let gmailClientSecret = $state('');
	let gmailRefreshToken = $state('');

	let loading = $state(true);
	let saving = $state(false);
	let testing = $state(false);
	let syncing = $state(false);
	let statusMsg = $state<{ kind: 'success' | 'error'; text: string } | null>(null);

	const providerOptions = [
		{ value: 'imap', label: 'IMAP' },
		{ value: 'gmail', label: 'Gmail (OAuth)' }
	];

	const intervalOptions = [
		{ value: '5m', label: '5 min' },
		{ value: '15m', label: '15 min' },
		{ value: '30m', label: '30 min' },
		{ value: '1h', label: '1 h' },
		{ value: 'off', label: 'Desligado' }
	];

	onMount(async () => {
		try {
			const res = await fetch('/api/settings/email');
			if (res.ok) {
				const data: { account: EmailAccountData | null; interval: string } = await res.json();
				if (data.interval) {
					interval = data.interval;
				}
				if (data.account) {
					provider = data.account.provider === 'gmail' ? 'gmail' : 'imap';
					address = data.account.address;
					folder = data.account.folder || 'INBOX';
					enabled = data.account.enabled;
					hasAuthConfig = data.account.has_auth_config;
					lastSyncAt = data.account.last_sync_at;
					lastError = data.account.last_error;
				}
			}
		} catch {
			statusMsg = { kind: 'error', text: t('settings.error.load') };
		} finally {
			loading = false;
		}
	});

	async function saveSettings() {
		saving = true;
		statusMsg = null;

		let authPayload: Record<string, unknown> | null = null;
		if (provider === 'imap') {
			if (imapHost.trim() || imapUser.trim() || imapPassword.trim()) {
				authPayload = {
					host: imapHost.trim(),
					port: Number(imapPort) || 993,
					username: imapUser.trim() || address.trim(),
					password: imapPassword,
					use_tls: imapUseTLS
				};
			}
		} else {
			if (gmailClientID.trim() || gmailRefreshToken.trim()) {
				authPayload = {
					client_id: gmailClientID.trim(),
					client_secret: gmailClientSecret.trim(),
					refresh_token: gmailRefreshToken.trim()
				};
			}
		}

		const body = {
			provider,
			address: address.trim(),
			folder: folder.trim() || 'INBOX',
			enabled,
			interval,
			auth: authPayload
		};

		try {
			const res = await fetch('/api/settings/email', {
				method: 'PUT',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify(body)
			});

			if (res.ok) {
				statusMsg = { kind: 'success', text: t('settings.email.saved') };
				hasAuthConfig = hasAuthConfig || authPayload !== null;
				toast('success', t('settings.email.saved'));
			} else {
				const errBody = await res.json().catch(() => ({}));
				statusMsg = { kind: 'error', text: errorText(errBody, 'Falha ao salvar configurações') };
			}
		} catch {
			statusMsg = { kind: 'error', text: t('jobs.error.save_network') };
		} finally {
			saving = false;
		}
	}

	async function testConnection() {
		testing = true;
		statusMsg = null;

		let authPayload: Record<string, unknown> | null = null;
		if (provider === 'imap' && (imapHost.trim() || imapPassword.trim())) {
			authPayload = {
				host: imapHost.trim(),
				port: Number(imapPort) || 993,
				username: imapUser.trim() || address.trim(),
				password: imapPassword,
				use_tls: imapUseTLS
			};
		} else if (provider === 'gmail' && (gmailClientID.trim() || gmailRefreshToken.trim())) {
			authPayload = {
				client_id: gmailClientID.trim(),
				client_secret: gmailClientSecret.trim(),
				refresh_token: gmailRefreshToken.trim()
			};
		}

		try {
			const res = await fetch('/api/settings/email/test', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ provider, auth: authPayload })
			});

			const data = await res.json().catch(() => ({ ok: false, error: 'Resposta inválida' }));
			if (data.ok) {
				statusMsg = { kind: 'success', text: data.message || 'Conexão estabelecida com sucesso!' };
				toast('success', data.message || 'Conexão OK');
			} else {
				statusMsg = { kind: 'error', text: data.error || 'Falha ao testar conexão' };
				toast('error', data.error || 'Falha no teste');
			}
		} catch {
			statusMsg = { kind: 'error', text: 'Falha de rede ao testar conexão' };
		} finally {
			testing = false;
		}
	}

	async function syncNow() {
		syncing = true;
		try {
			const res = await fetch('/api/settings/email/sync', { method: 'POST' });
			if (res.status === 202) {
				toast('info', 'Sincronização iniciada!');
			} else {
				toast('error', 'Falha ao iniciar sincronização');
			}
		} catch {
			toast('error', 'Erro de rede ao sincronizar');
		} finally {
			syncing = false;
		}
	}
</script>

<svelte:head>
	<title>{t('settings.email.title')} — openJob</title>
</svelte:head>

<PageHeader
	title={t('common.settings')}
	subtitle={t('settings.subtitle')}
	breadcrumb={[{ label: t('common.home'), href: '/' }, { label: t('common.settings'), href: '/settings' }, { label: t('nav.settings_email') }]}
/>

<SettingsNav active="email" />

<div class="email-settings">
	<Card>
		<header class="section-head">
			<div>
				<h2>{t('settings.email.title')}</h2>
				<p class="subtitle">{t('settings.email.subtitle')}</p>
			</div>
			<div class="head-badges">
				{#if hasAuthConfig}
					<Badge tone="success">Configurado</Badge>
				{:else}
					<Badge tone="neutral">Não configurado</Badge>
				{/if}
				{#if enabled}
					<Badge tone="primary">Ativo</Badge>
				{/if}
			</div>
		</header>

		{#if statusMsg}
			<Alert severity={statusMsg.kind}>{statusMsg.text}</Alert>
		{/if}

		{#if lastError}
			<Alert severity="warning" title="Último aviso">{lastError}</Alert>
		{/if}

		<form onsubmit={(e) => { e.preventDefault(); saveSettings(); }} class="form-body">
			<Field label={t('settings.email.provider')}>
				<SegmentedControl
					label={t('settings.email.provider')}
					options={providerOptions}
					value={provider}
					onchange={(val) => { provider = val as 'imap' | 'gmail'; }}
				/>
			</Field>

			<Field label={t('settings.email.address')} for="email-address">
				<input
					id="email-address"
					type="email"
					bind:value={address}
					placeholder={t('settings.email.address_placeholder')}
					required
				/>
			</Field>

			<Field label={t('settings.email.folder')} for="email-folder">
				<input
					id="email-folder"
					type="text"
					bind:value={folder}
					placeholder="INBOX"
				/>
			</Field>

			{#if provider === 'imap'}
				<div class="provider-fields">
					<div class="grid grid-2">
						<Field label={t('settings.email.imap_host')} for="imap-host">
							<input
								id="imap-host"
								type="text"
								bind:value={imapHost}
								placeholder="imap.gmail.com / imap.mail.me.com"
							/>
						</Field>

						<Field label={t('settings.email.imap_port')} for="imap-port">
							<input
								id="imap-port"
								type="number"
								bind:value={imapPort}
								placeholder="993"
							/>
						</Field>
					</div>

					<div class="grid grid-2">
						<Field label={t('settings.email.imap_user')} for="imap-user">
							<input
								id="imap-user"
								type="text"
								bind:value={imapUser}
								placeholder="Deixe em branco para usar o endereço"
							/>
						</Field>

						<Field label={t('settings.email.imap_password')} for="imap-password">
							<input
								id="imap-password"
								type="password"
								bind:value={imapPassword}
								placeholder={hasAuthConfig ? '•••••••• (manter atual)' : 'Senha de app'}
							/>
						</Field>
					</div>

					<Field label="">
						<Checkbox label={t('settings.email.use_tls')} bind:checked={imapUseTLS} />
					</Field>
				</div>
			{:else}
				<div class="provider-fields">
					<Field label={t('settings.email.gmail_client_id')} for="gmail-client-id">
						<input
							id="gmail-client-id"
							type="text"
							bind:value={gmailClientID}
							placeholder="xxx.apps.googleusercontent.com"
						/>
					</Field>

					<Field label={t('settings.email.gmail_client_secret')} for="gmail-client-secret">
						<input
							id="gmail-client-secret"
							type="password"
							bind:value={gmailClientSecret}
							placeholder={hasAuthConfig ? '••••••••' : 'Client secret'}
						/>
					</Field>

					<Field label={t('settings.email.gmail_refresh_token')} for="gmail-refresh-token">
						<input
							id="gmail-refresh-token"
							type="password"
							bind:value={gmailRefreshToken}
							placeholder={hasAuthConfig ? '••••••••' : 'Refresh token'}
						/>
					</Field>
				</div>
			{/if}

			<div class="sync-options">
				<Field label="">
					<Checkbox label={t('settings.email.enabled')} bind:checked={enabled} />
				</Field>

				<Field label={t('settings.email.interval')}>
					<SegmentedControl
						label={t('settings.email.interval')}
						options={intervalOptions}
						value={interval}
						onchange={(val) => { interval = val; }}
					/>
				</Field>
			</div>

			<div class="actions">
				<Button type="submit" variant="primary" disabled={saving}>
					{saving ? 'Salvando...' : t('settings.email.save')}
				</Button>

				<Button type="button" variant="secondary" onclick={testConnection} disabled={testing}>
					{testing ? t('settings.email.testing') : t('settings.email.test')}
				</Button>

				<Button type="button" variant="secondary" onclick={syncNow} disabled={syncing || !hasAuthConfig}>
					{syncing ? t('settings.email.syncing') : t('settings.email.sync_now')}
				</Button>
			</div>

			{#if lastSyncAt}
				<p class="sync-meta">
					{t('settings.email.last_sync', { date: formatDate(lastSyncAt) })}
				</p>
			{:else}
				<p class="sync-meta muted">{t('settings.email.never_synced')}</p>
			{/if}
		</form>

		<footer class="privacy-footer">
			<p>{t('settings.email.privacy_notice')}</p>
		</footer>
	</Card>
</div>

<style>
	.email-settings {
		max-width: 800px;
	}

	.section-head {
		display: flex;
		justify-content: space-between;
		align-items: flex-start;
		gap: var(--space-4);
		margin-bottom: var(--space-4);
	}

	.head-badges {
		display: flex;
		gap: var(--space-2);
	}

	.subtitle {
		color: var(--color-text-muted);
		font-size: var(--text-caption);
		margin-top: var(--space-1);
	}

	.form-body {
		display: flex;
		flex-direction: column;
		gap: var(--space-4);
		margin-top: var(--space-4);
	}

	.provider-fields {
		display: flex;
		flex-direction: column;
		gap: var(--space-3);
		padding: var(--space-4);
		background: var(--color-surface-elevated);
		border-radius: var(--radius-md);
		border: 1px solid var(--color-border);
	}

	.sync-options {
		display: flex;
		flex-direction: column;
		gap: var(--space-3);
		padding-top: var(--space-2);
		border-top: 1px solid var(--color-border);
	}

	.actions {
		display: flex;
		gap: var(--space-3);
		flex-wrap: wrap;
		margin-top: var(--space-2);
	}

	.sync-meta {
		font-size: var(--text-caption);
		color: var(--color-text-muted);
		margin-top: var(--space-2);
	}

	.privacy-footer {
		margin-top: var(--space-6);
		padding-top: var(--space-4);
		border-top: 1px solid var(--color-border);
		font-size: var(--text-caption);
		color: var(--color-text-muted);
		line-height: var(--line-height-relaxed);
	}
</style>
