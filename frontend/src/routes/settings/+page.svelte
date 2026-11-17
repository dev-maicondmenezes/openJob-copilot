<script lang="ts">
	import { t } from '#lib/i18n/t.js';
	import { errorText } from '#lib/i18n/errors.js';
	import { onMount } from 'svelte';
	import SettingsNav from '#lib/components/SettingsNav.svelte';
	import PageHeader from '#lib/ui/PageHeader.svelte';
	import Card from '#lib/ui/Card.svelte';
	import Field from '#lib/ui/Field.svelte';
	import Input from '#lib/ui/Input.svelte';
	import Button from '#lib/ui/Button.svelte';
	import IconButton from '#lib/ui/IconButton.svelte';
	import Alert from '#lib/ui/Alert.svelte';
	import Badge from '#lib/ui/Badge.svelte';

	interface LLMConfig {
		configured: boolean;
		base_url: string;
		model: string;
	}
	interface PingResult {
		ok: boolean;
		message?: string;
		error?: string;
	}

	let baseUrl = $state('');
	let model = $state('');
	let apiKey = $state('');
	let showKey = $state(false);
	let configured = $state(false);
	let saving = $state(false);
	let testing = $state(false);
	let status: { kind: 'success' | 'error'; text: string } | null = $state(null);

	onMount(async () => {
		try {
			const res = await fetch('/api/settings/llm');
			if (res.ok) {
				const cfg: LLMConfig = await res.json();
				baseUrl = cfg.base_url;
				model = cfg.model;
				configured = cfg.configured;
			}
		} catch {
			status = { kind: 'error', text: t('settings.error.load') };
		}
	});

	async function save() {
		if (!apiKey.trim()) {
			status = { kind: 'error', text: t('settings.error.key_required') };
			return;
		}
		saving = true;
		status = null;
		try {
			const res = await fetch('/api/settings/llm-key', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ base_url: baseUrl, model, api_key: apiKey })
			});
			if (res.status === 204) {
				configured = true;
				apiKey = '';
				status = { kind: 'success', text: t('settings.error.saved') };
			} else {
				const body = await res.json().catch(() => ({}));
				status = { kind: 'error', text: errorText(body, t('settings.error.save_http', { status: res.status })) };
			}
		} catch {
			status = { kind: 'error', text: t('jobs.error.save_network') };
		} finally {
			saving = false;
		}
	}

	async function testConnection() {
		testing = true;
		status = null;
		try {
			const res = await fetch('/api/settings/llm-ping', { method: 'POST' });
			const result: PingResult = await res.json();
			status = result.ok
				? { kind: 'success', text: result.message ?? t('settings.error.ping_ok') }
				: { kind: 'error', text: errorText(result, t('settings.error.ping_fail')) };
		} catch {
			status = { kind: 'error', text: t('settings.error.ping_network') };
		} finally {
			testing = false;
		}
	}
</script>

<svelte:head>
	<title>{t('settings.meta_title')}</title>
</svelte:head>

<PageHeader
	title={t('common.settings')}
	subtitle={t('settings.subtitle')}
	breadcrumb={[{ label: t('common.home'), href: '/' }, { label: t('common.settings') }]}
/>

<SettingsNav active="llm" />

{#if status}
	<Alert severity={status.kind}>{status.text}</Alert>
{/if}

<Card title={t('settings.card.llm_title')} subtitle={t('settings.card.llm_desc')}>
	{#snippet actions()}
		<Badge tone={configured ? 'success' : 'warning'}>
			{configured ? t('settings.badge.configured') : t('settings.badge.not_configured')}
		</Badge>
	{/snippet}
	<form onsubmit={(e) => { e.preventDefault(); save(); }}>
		<Field label={t('settings.field.base_url')} for="base_url">
			<input id="base_url" type="url" bind:value={baseUrl} placeholder="https://api.openai.com/v1" autocomplete="off" spellcheck="false" />
		</Field>
		<Field label={t('settings.field.model')} for="model">
			<input id="model" bind:value={model} placeholder="gpt-4o-mini" autocomplete="off" spellcheck="false" />
		</Field>
		<Field label={t('settings.field.api_key')} for="api_key" hint={t('settings.field.api_key_hint')}>
			<div class="key-row">
				<input
					id="api_key"
					type={showKey ? 'text' : 'password'}
					placeholder={configured ? t('settings.field.api_key_saved') : t('settings.field.api_key_placeholder')}
					bind:value={apiKey}
					autocomplete="off"
					spellcheck="false"
				/>
				<IconButton
					icon={showKey ? 'eye-off' : 'eye'}
					label={showKey ? t('ui.hide_password') : t('ui.show_password')}
					variant="ghost"
					onclick={() => (showKey = !showKey)}
				/>
			</div>
		</Field>
		<div class="form-actions">
			<Button type="submit" loading={saving}>{t('common.save')}</Button>
			<Button type="button" variant="secondary" loading={testing} onclick={testConnection}>{t('settings.button.test_connection')}</Button>
		</div>
	</form>
</Card>

<style>
	form {
		display: flex;
		flex-direction: column;
		gap: var(--space-4);
	}

	.key-row {
		display: flex;
		gap: var(--space-2);
	}

	.key-row :global(.input) {
		flex: 1;
		min-width: 0;
	}

	.form-actions {
		margin-top: var(--space-2);
	}
</style>
