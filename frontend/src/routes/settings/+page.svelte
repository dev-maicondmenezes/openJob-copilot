<script lang="ts">
	import { t } from '#lib/i18n/t.js';
	import { errorText, readErrorBody } from '#lib/i18n/errors.js';
	import { onMount } from 'svelte';
	import SettingsNav from '#lib/components/SettingsNav.svelte';
	import PageHeader from '#lib/ui/PageHeader.svelte';
	import Card from '#lib/ui/Card.svelte';
	import Field from '#lib/ui/Field.svelte';
	import Input from '#lib/ui/Input.svelte';
	import Select from '#lib/ui/Select.svelte';
	import Button from '#lib/ui/Button.svelte';
	import LinkButton from '#lib/ui/LinkButton.svelte';
	import IconButton from '#lib/ui/IconButton.svelte';
	import Alert from '#lib/ui/Alert.svelte';
	import Badge from '#lib/ui/Badge.svelte';
	import EmptyState from '#lib/ui/EmptyState.svelte';
	import Icon from '#lib/ui/Icon.svelte';

	interface Preset {
		id: string;
		name: string;
		base_url: string;
		model: string;
		models: string[];
		free_tier: boolean;
		free_note?: string;
		key_url?: string;
		docs_url?: string;
		requires_key: boolean;
		local: boolean;
	}

	interface Profile {
		id: string;
		name: string;
		preset_id?: string;
		base_url: string;
		model: string;
		active: boolean;
		local: boolean;
	}

	interface PingResult {
		ok: boolean;
		message?: string;
		code?: string;
		error?: string;
	}

	const CUSTOM = 'custom';

	let presets = $state<Preset[]>([]);
	let defaultPresetId = $state(CUSTOM);
	let profiles = $state<Profile[]>([]);
	let loading = $state(true);
	let loadError = $state('');
	let status: { kind: 'success' | 'error'; text: string } | null = $state(null);

	let editing = $state<string | null>(null);
	let formPresetId = $state(CUSTOM);
	let formName = $state('');
	let formBaseUrl = $state('');
	let formModel = $state('');
	let formApiKey = $state('');
	let showKey = $state(false);
	let saving = $state(false);
	let testingId = $state('');
	let activatingId = $state('');
	let deletingId = $state('');

	const editingProfile = $derived(profiles.find((p) => p.id === editing) ?? null);
	const activeProfile = $derived(profiles.find((p) => p.active) ?? null);
	const selectedPreset = $derived(presets.find((p) => p.id === formPresetId) ?? null);
	const isCustom = $derived(formPresetId === CUSTOM);
	const modelOptions = $derived(selectedPreset?.models ?? []);
	const modelSelectOptions = $derived(modelOptions.map((model) => ({ value: model, label: model })));
	const presetOptions = $derived([
		...presets.map((preset) => ({ value: preset.id, label: preset.name })),
		{ value: CUSTOM, label: t('settings.preset.custom') }
	]);
	const providerLinksVisible = $derived(!isCustom && selectedPreset !== null);
	const providerNeedsKey = $derived(isCustom || Boolean(selectedPreset?.requires_key));

	async function load(initial = false) {
		if (initial) loading = true;
		try {
			const [presetRes, profileRes] = await Promise.all([
				fetch('/api/settings/llm/presets'),
				fetch('/api/settings/llm/profiles')
			]);
			if (!presetRes.ok || !profileRes.ok) throw new Error('load failed');
			const catalog = await presetRes.json();
			const list = await profileRes.json();
			presets = Array.isArray(catalog.presets) ? catalog.presets : [];
			defaultPresetId = catalog.default_id || presets[0]?.id || CUSTOM;
			profiles = Array.isArray(list.profiles) ? list.profiles : [];
			loadError = '';
		} catch {
			loadError = t('settings.error.load');
		} finally {
			if (initial) loading = false;
		}
	}

	function applyPreset(id: string) {
		formPresetId = id;
		const preset = presets.find((p) => p.id === id);
		if (preset) {
			formBaseUrl = preset.base_url;
			formModel = preset.model || preset.models?.[0] || '';
		} else {
			formBaseUrl = '';
			formModel = '';
		}
	}

	function startCreate(presetId = defaultPresetId) {
		status = null;
		editing = 'new';
		formName = '';
		formApiKey = '';
		showKey = false;
		applyPreset(presets.some((p) => p.id === presetId) ? presetId : CUSTOM);
	}

	function startEdit(profile: Profile) {
		status = null;
		editing = profile.id;
		formName = profile.name;
		formBaseUrl = profile.base_url;
		formModel = profile.model;
		formApiKey = '';
		showKey = false;
		formPresetId = presets.some((p) => p.id === profile.preset_id) ? profile.preset_id! : CUSTOM;
	}

	function cancelEdit() {
		editing = null;
		formApiKey = '';
		showKey = false;
	}

	async function save() {
		if (!formModel.trim()) {
			status = { kind: 'error', text: t('settings.error.model_required') };
			return;
		}
		if (!formBaseUrl.trim()) {
			status = { kind: 'error', text: t('settings.error.base_url_required') };
			return;
		}
		const keepsLocalPlaceholder = editingProfile?.local === true && !formApiKey.trim();
		if (providerNeedsKey && (!formApiKey.trim() && (editing === 'new' || keepsLocalPlaceholder))) {
			status = { kind: 'error', text: t('settings.error.key_required') };
			return;
		}
		saving = true;
		status = null;
		const wasNew = editing === 'new';
		try {
			const body = {
				name: formName.trim(),
				preset_id: isCustom ? '' : formPresetId,
				base_url: formBaseUrl.trim(),
				model: formModel.trim(),
				api_key: formApiKey,
				activate: profiles.length === 0
			};
			const res =
				editing === 'new'
					? await fetch('/api/settings/llm/profiles', {
							method: 'POST',
							headers: { 'Content-Type': 'application/json' },
							body: JSON.stringify(body)
						})
					: await fetch(`/api/settings/llm/profiles/${editing}`, {
							method: 'PUT',
							headers: { 'Content-Type': 'application/json' },
							body: JSON.stringify(body)
						});
			if (!res.ok) {
				status = {
					kind: 'error',
					text: errorText(await readErrorBody(res), t('settings.error.save_http', { status: res.status }))
				};
				return;
			}
			await load();
			editing = null;
			formApiKey = '';
			showKey = false;
			status = {
				kind: 'success',
				text: t(wasNew ? 'settings.profiles.created' : 'settings.profiles.saved')
			};
		} catch {
			status = { kind: 'error', text: t('settings.error.save_network') };
		} finally {
			saving = false;
		}
	}

	async function activate(id: string) {
		activatingId = id;
		status = null;
		try {
			const res = await fetch('/api/settings/llm/activate', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ id })
			});
			if (!res.ok) {
				status = {
					kind: 'error',
					text: errorText(await readErrorBody(res), t('settings.error.save_http', { status: res.status }))
				};
				return;
			}
			await load();
			status = { kind: 'success', text: t('settings.profiles.activated') };
		} catch {
			status = { kind: 'error', text: t('settings.error.save_network') };
		} finally {
			activatingId = '';
		}
	}

	async function test(id: string) {
		testingId = id;
		status = null;
		try {
			const res = await fetch(`/api/settings/llm/profiles/${id}/test`, { method: 'POST' });
			const result: PingResult = await res.json().catch(() => ({ ok: false }));
			status = result.ok
				? { kind: 'success', text: t('settings.error.ping_ok') }
				: { kind: 'error', text: errorText(result, t('settings.error.ping_fail')) };
		} catch {
			status = { kind: 'error', text: t('settings.error.ping_network') };
		} finally {
			testingId = '';
		}
	}

	async function remove(profile: Profile) {
		if (!window.confirm(t('settings.profiles.delete_confirm', { name: profile.name }))) return;
		deletingId = profile.id;
		status = null;
		try {
			const res = await fetch(`/api/settings/llm/profiles/${profile.id}`, { method: 'DELETE' });
			if (!res.ok) {
				status = {
					kind: 'error',
					text: errorText(await readErrorBody(res), t('settings.error.save_http', { status: res.status }))
				};
				return;
			}
			await load();
			status = { kind: 'success', text: t('settings.profiles.deleted') };
		} catch {
			status = { kind: 'error', text: t('settings.error.save_network') };
		} finally {
			deletingId = '';
		}
	}

	onMount(() => load(true));
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
		<Badge tone={activeProfile ? 'success' : 'warning'}>
			{activeProfile ? t('settings.badge.configured') : t('settings.badge.not_configured')}
		</Badge>
	{/snippet}

	{#if loading}
		<p class="loading" role="status">{t('settings.profiles.loading')}</p>
	{:else if loadError}
		<Alert severity="error">{loadError}</Alert>
		<div class="form-actions">
			<Button variant="secondary" onclick={() => load(true)}>{t('common.retry')}</Button>
		</div>
	{:else}
		{#if profiles.length === 0 && editing === null}
			<EmptyState
				icon="key"
				title={t('settings.onboarding.title')}
				description={t('settings.onboarding.subtitle')}
			>
				<Button onclick={() => startCreate(defaultPresetId)}>
					{t('settings.onboarding.start_free')}
				</Button>
				<Button variant="secondary" onclick={() => startCreate(CUSTOM)}>
					{t('settings.profiles.add')}
				</Button>
			</EmptyState>
		{:else}
			<div class="profile-list" role="list">
				{#each profiles as profile (profile.id)}
					<div class="profile-row" role="listitem">
						<div class="profile-main">
							<div class="profile-title">
								<span class="profile-name">{profile.name}</span>
								{#if profile.active}
									<Badge tone="success">{t('common.active')}</Badge>
								{/if}
								{#if profile.local}
									<Badge>{t('settings.profiles.local')}</Badge>
								{/if}
							</div>
							<p class="profile-meta">{profile.base_url} · {profile.model}</p>
						</div>
						<div class="profile-actions">
							{#if !profile.active}
								<Button
									size="sm"
									variant="secondary"
									loading={activatingId === profile.id}
									onclick={() => activate(profile.id)}
								>
									{t('settings.profiles.activate')}
								</Button>
							{/if}
							<Button
								size="sm"
								variant="ghost"
								loading={testingId === profile.id}
								onclick={() => test(profile.id)}
							>
								{t('common.test')}
							</Button>
							<Button size="sm" variant="ghost" onclick={() => startEdit(profile)}>
								{t('common.edit')}
							</Button>
							{#if !profile.active}
								<Button
									size="sm"
									variant="danger"
									loading={deletingId === profile.id}
									onclick={() => remove(profile)}
								>
									{t('common.remove')}
								</Button>
							{/if}
						</div>
					</div>
				{/each}
			</div>
			{#if editing === null}
				<div class="form-actions">
					<Button variant="secondary" onclick={() => startCreate()}>
						<Icon name="plus" size={16} />
						{t('settings.profiles.add')}
					</Button>
				</div>
			{/if}
		{/if}
	{/if}
</Card>

{#if editing !== null}
	<Card
		title={editing === 'new' ? t('settings.profiles.create_title') : t('settings.profiles.edit_title')}
		subtitle={t('settings.profiles.form_subtitle')}
	>
		<form onsubmit={(e) => { e.preventDefault(); save(); }}>
			<Field label={t('settings.profiles.name')} for="profile-name" hint={t('settings.profiles.name_hint')}>
				<Input
					id="profile-name"
					bind:value={formName}
					maxlength={80}
					placeholder={t('settings.profiles.name_placeholder')}
					autocomplete="off"
				/>
			</Field>

			<Field label={t('settings.field.preset')} for="profile-preset">
				<Select
					id="profile-preset"
					value={formPresetId}
					options={presetOptions}
					onchange={(e) => applyPreset(e.currentTarget.value)}
				/>
			</Field>

			{#if isCustom}
				<Field label={t('settings.field.base_url')} for="profile-base-url">
					<Input
						id="profile-base-url"
						type="url"
						bind:value={formBaseUrl}
						placeholder="https://api.example.com/v1"
						autocomplete="off"
						spellcheck="false"
					/>
				</Field>
			{/if}

			<Field label={t('settings.field.model')} for="profile-model">
				{#if !isCustom && modelSelectOptions.length > 0}
					<Select id="profile-model" bind:value={formModel} options={modelSelectOptions} />
				{:else}
					<Input
						id="profile-model"
						bind:value={formModel}
						placeholder="gpt-4o-mini"
						autocomplete="off"
						spellcheck="false"
					/>
				{/if}
			</Field>

			<Field
				label={t('settings.field.api_key')}
				for="profile-api-key"
				hint={selectedPreset?.local ? t('settings.profiles.local_hint') : t('settings.field.api_key_hint')}
			>
				<div class="key-row">
					<Input
						id="profile-api-key"
						type={showKey ? 'text' : 'password'}
						placeholder={editingProfile
							? t('settings.field.api_key_saved')
							: t('settings.field.api_key_placeholder')}
						bind:value={formApiKey}
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

			{#if providerLinksVisible && selectedPreset}
				<Card compact title={t('settings.provider.title')} subtitle={selectedPreset.name}>
					<div class="provider-links">
						{#if selectedPreset.docs_url}
							<LinkButton
								href={selectedPreset.docs_url}
								target="_blank"
								rel="noreferrer"
								variant="ghost"
								size="sm"
							>
								<Icon name="external" size={16} />
								{t('settings.provider.docs')}
							</LinkButton>
						{/if}
						{#if selectedPreset.key_url && selectedPreset.requires_key}
							<LinkButton
								href={selectedPreset.key_url}
								target="_blank"
								rel="noreferrer"
								variant="secondary"
								size="sm"
							>
								<Icon name="key" size={16} />
								{t('settings.provider.key')}
							</LinkButton>
						{/if}
					</div>
					{#if selectedPreset.requires_key}
						<p class="provider-hint">{t('settings.provider.key_hint')}</p>
					{/if}
					{#if selectedPreset.free_tier}
						<p class="provider-hint">
							<Badge tone="success">{t('settings.profiles.free')}</Badge>
							{selectedPreset.free_note === 'free_local'
								? t('settings.onboarding.local_runtime')
								: t('settings.onboarding.free_tier')}
						</p>
					{/if}
				</Card>
			{/if}

			<div class="form-actions">
				<Button type="submit" loading={saving}>{t('common.save')}</Button>
				<Button type="button" variant="ghost" onclick={cancelEdit}>{t('common.cancel')}</Button>
			</div>
		</form>
	</Card>
{/if}

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
		display: flex;
		gap: var(--space-2);
		flex-wrap: wrap;
		margin-top: var(--space-2);
	}

	.profile-list {
		display: flex;
		flex-direction: column;
		gap: var(--space-3);
	}

	.profile-row {
		display: flex;
		align-items: flex-start;
		justify-content: space-between;
		gap: var(--space-3);
		flex-wrap: wrap;
		padding: var(--space-3);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-md);
		background: var(--color-surface-elevated);
	}

	.profile-main {
		flex: 1 1 220px;
		min-width: 0;
	}

	.profile-title {
		display: flex;
		align-items: center;
		gap: var(--space-2);
		flex-wrap: wrap;
	}

	.profile-name {
		font-weight: var(--weight-semibold);
	}

	.profile-meta {
		margin: var(--space-1) 0 0;
		font-size: var(--text-caption);
		line-height: var(--leading-caption);
		color: var(--color-text-muted);
		overflow-wrap: anywhere;
	}

	.profile-actions {
		display: flex;
		gap: var(--space-2);
		flex-wrap: wrap;
	}

	.provider-links {
		display: flex;
		gap: var(--space-2);
		flex-wrap: wrap;
	}

	.provider-hint {
		display: flex;
		align-items: center;
		gap: var(--space-2);
		flex-wrap: wrap;
		margin: var(--space-2) 0 0;
		font-size: var(--text-caption);
		line-height: var(--leading-caption);
		color: var(--color-text-muted);
	}

	.loading {
		margin: 0;
		color: var(--color-text-muted);
	}

	@media (max-width: 640px) {
		.profile-row {
			flex-direction: column;
		}

		.profile-actions {
			width: 100%;
		}

		.form-actions :global(.btn) {
			flex: 1;
		}
	}
</style>
