<script lang="ts">
	import { t } from '#lib/i18n/t.js';
	import { errorText } from '#lib/i18n/errors.js';
	import { formatDate } from '#lib/i18n/format.js';
	import { onMount } from 'svelte';
	import SettingsNav from '#lib/components/SettingsNav.svelte';
	import PageHeader from '#lib/ui/PageHeader.svelte';
	import Card from '#lib/ui/Card.svelte';
	import Field from '#lib/ui/Field.svelte';
	import Input from '#lib/ui/Input.svelte';
	import Select from '#lib/ui/Select.svelte';
	import Checkbox from '#lib/ui/Checkbox.svelte';
	import Switch from '#lib/ui/Switch.svelte';
	import Button from '#lib/ui/Button.svelte';
	import Badge from '#lib/ui/Badge.svelte';
	import Alert from '#lib/ui/Alert.svelte';
	import Dialog from '#lib/ui/Dialog.svelte';
	import DataView from '#lib/ui/DataView.svelte';
	import type { DataColumn } from '#lib/ui/data.js';

	interface Capabilities {
		supports_keyword_search: boolean;
		supports_pagination: boolean;
		max_results_per_run: number;
		rate_limit: { requests: number; period: string };
		attribution_required: boolean;
		experimental: boolean;
		recent_window_hours: number;
	}

	interface Source {
		id: string;
		name: string;
		adapter: string;
		enabled: boolean;
		priority: number;
		auth_type: string;
		has_api_key: boolean;
		capabilities: Capabilities;
		created_at: string;
		updated_at: string;
	}

	interface AdapterInfo {
		slug: string;
		name: string;
		capabilities: Capabilities;
	}

	let sources: Source[] = $state([]);
	let adapters: AdapterInfo[] = $state([]);
	let loading = $state(true);
	let testingId = $state<string | null>(null);
	let status: { kind: 'success' | 'error'; text: string } | null = $state(null);
	let testResults: Record<string, { ok: boolean; latency_ms?: number; count?: number; error?: string }> = $state({});

	let showModal = $state(false);
	let modalMode = $state<'add' | 'edit'>('add');
	let formId = $state('');
	let formName = $state('');
	let formAdapter = $state('himalayas');
	let formEnabled = $state(true);
	let formPriority = $state(10);
	let formAuthType = $state('none');
	let formApiKey = $state('');
	let saving = $state(false);

	const columns = $derived<DataColumn<Source>[]>([
		{ key: 'name', header: t('sources.col.name_id'), cardLabel: t('sources.col.name'), value: (src) => src.name },
		{ key: 'adapter', header: t('sources.col.adapter'), value: (src) => src.adapter },
		{ key: 'enabled', header: t('common.active'), value: (src) => (src.enabled ? t('common.active') : t('common.inactive')) },
		{ key: 'auth', header: t('sources.col.auth'), value: (src) => (src.auth_type === 'api_key' ? (src.has_api_key ? t('sources.cell.key_saved') : t('sources.cell.no_key')) : t('sources.cell.public')) },
		{ key: 'rate', header: t('sources.col.rate_limit'), value: (src) => (src.capabilities.rate_limit.requests > 0 ? `${src.capabilities.rate_limit.requests}/${src.capabilities.rate_limit.period}` : t('sources.cell.default')) },
		{ key: 'updated', header: t('sources.col.updated'), value: (src) => formatDate(src.updated_at) },
		{ key: 'actions', header: t('tasks.col.actions'), value: () => '' }
	]);

	onMount(async () => {
		await Promise.all([loadSources(), loadAdapters()]);
	});

	async function loadSources() {
		try {
			const res = await fetch('/api/sources');
			if (res.ok) sources = await res.json();
			else status = { kind: 'error', text: t('sources.error.load') };
		} catch {
			status = { kind: 'error', text: t('sources.error.load_network') };
		} finally {
			loading = false;
		}
	}

	async function loadAdapters() {
		try {
			const res = await fetch('/api/sources/adapters');
			if (res.ok) {
				adapters = await res.json();
				if (adapters.length > 0 && !formAdapter) formAdapter = adapters[0].slug;
			}
		} catch {
			// silencioso
		}
	}

	async function toggleEnabled(src: Source) {
		status = null;
		try {
			const res = await fetch(`/api/sources/${src.id}`, {
				method: 'PATCH',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ enabled: !src.enabled })
			});
			if (res.ok) src.enabled = !src.enabled;
			else {
				const err = await res.json().catch(() => ({}));
				status = { kind: 'error', text: errorText(err, t('sources.error.toggle')) };
			}
		} catch {
			status = { kind: 'error', text: t('sources.error.toggle_network') };
		}
	}

	async function testSource(id: string) {
		testingId = id;
		status = null;
		try {
			const res = await fetch(`/api/sources/${id}/test`, { method: 'POST' });
			const result = await res.json().catch(() => ({ ok: false, error: t('sources.error.unknown') }));
			testResults[id] = result;
			status = result.ok
				? { kind: 'success', text: t('sources.flash.test', { ms: result.latency_ms, count: result.count ?? 1 }) }
				: { kind: 'error', text: t('sources.flash.test_fail', { error: result.error }) };
		} catch {
			status = { kind: 'error', text: t('sources.error.test_network') };
		} finally {
			testingId = null;
		}
	}

	async function deleteSource(id: string) {
		if (!confirm(t('sources.confirm_remove', { id }))) return;
		status = null;
		try {
			const res = await fetch(`/api/sources/${id}`, { method: 'DELETE' });
			if (res.ok) {
				status = { kind: 'success', text: t('sources.flash.removed') };
				await loadSources();
			} else {
				const err = await res.json().catch(() => ({}));
				status = { kind: 'error', text: errorText(err, t('sources.error.remove')) };
			}
		} catch {
			status = { kind: 'error', text: t('sources.error.remove_network') };
		}
	}

	function openAddModal() {
		modalMode = 'add';
		formId = '';
		formName = '';
		formAdapter = adapters[0]?.slug ?? 'himalayas';
		formEnabled = true;
		formPriority = 50;
		formAuthType = 'none';
		formApiKey = '';
		showModal = true;
	}

	function openEditModal(src: Source) {
		modalMode = 'edit';
		formId = src.id;
		formName = src.name;
		formAdapter = src.adapter;
		formEnabled = src.enabled;
		formPriority = src.priority;
		formAuthType = src.auth_type;
		formApiKey = '';
		showModal = true;
	}

	async function saveModal() {
		saving = true;
		status = null;
		try {
			if (modalMode === 'add') {
				const res = await fetch('/api/sources', {
					method: 'POST',
					headers: { 'Content-Type': 'application/json' },
					body: JSON.stringify({
						id: formId,
						name: formName,
						adapter: formAdapter,
						enabled: formEnabled,
						priority: Number(formPriority),
						auth_type: formAuthType,
						api_key: formApiKey
					})
				});
				if (res.ok) {
					showModal = false;
					status = { kind: 'success', text: t('sources.flash.added') };
				} else {
					const err = await res.json().catch(() => ({}));
					status = { kind: 'error', text: errorText(err, t('sources.error.create')) };
				}
			} else {
				const payload: Record<string, unknown> = {
					name: formName,
					adapter: formAdapter,
					enabled: formEnabled,
					priority: Number(formPriority),
					auth_type: formAuthType
				};
				if (formApiKey.trim()) payload.api_key = formApiKey.trim();
				const res = await fetch(`/api/sources/${formId}`, {
					method: 'PATCH',
					headers: { 'Content-Type': 'application/json' },
					body: JSON.stringify(payload)
				});
				if (res.ok) {
					showModal = false;
					status = { kind: 'success', text: t('sources.flash.updated') };
				} else {
					const err = await res.json().catch(() => ({}));
					status = { kind: 'error', text: errorText(err, t('sources.error.update')) };
				}
			}
			await loadSources();
		} catch {
			status = { kind: 'error', text: t('sources.error.save_network') };
		} finally {
			saving = false;
		}
	}
</script>

<svelte:head>
	<title>{t('sources.meta_title')}</title>
</svelte:head>

{#snippet renderCell(src: Source, column: DataColumn<Source>)}
	{#if column.key === 'name'}
		<strong>{src.name}</strong>
		<span class="cell-note">{src.id}</span>
		{#if src.capabilities.attribution_required}<Badge tone="info">{t('sources.badge.attribution')}</Badge>{/if}
		{#if src.capabilities.experimental}<Badge tone="warning">{t('sources.badge.experimental')}</Badge>{/if}
	{:else if column.key === 'adapter'}
		<code class="mono">{src.adapter}</code>
	{:else if column.key === 'enabled'}
		<Switch
			label={src.enabled ? t('common.active') : t('common.inactive')}
			checked={src.enabled}
			onchange={() => toggleEnabled(src)}
		/>
	{:else if column.key === 'auth'}
		<Badge tone={src.auth_type === 'api_key' ? (src.has_api_key ? 'success' : 'warning') : 'neutral'}>
			{src.auth_type === 'api_key' ? (src.has_api_key ? t('sources.cell.key_saved') : t('sources.cell.no_key')) : t('sources.cell.public')}
		</Badge>
	{:else if column.key === 'rate'}
		{src.capabilities.rate_limit.requests > 0
			? `${src.capabilities.rate_limit.requests}/${src.capabilities.rate_limit.period === 'day' ? t('sources.cell.day') : src.capabilities.rate_limit.period === 'hour' ? t('sources.cell.hour') : t('sources.cell.min')}`
			: t('sources.cell.default')}
	{:else if column.key === 'actions'}
		<div class="row-actions">
			<Button variant="ghost" size="sm" loading={testingId === src.id} onclick={() => testSource(src.id)}>{t('common.test')}</Button>
			<Button variant="ghost" size="sm" onclick={() => openEditModal(src)}>{t('common.edit')}</Button>
			<Button variant="ghost" size="sm" onclick={() => deleteSource(src.id)}>{t('common.remove')}</Button>
		</div>
		{#if testResults[src.id]}
			<Alert severity={testResults[src.id].ok ? 'success' : 'error'}>
				{testResults[src.id].ok
					? t('sources.button.test_ok', { ms: testResults[src.id].latency_ms })
					: testResults[src.id].error}
			</Alert>
		{/if}
	{:else}
		{column.value?.(src) ?? ''}
	{/if}
{/snippet}

<PageHeader
	title={t('common.settings')}
	subtitle={t('sources.subtitle')}
	breadcrumb={[{ label: t('common.home'), href: '/' }, { label: t('common.settings'), href: '/settings' }, { label: 'Fontes' }]}
/>

<SettingsNav active="sources" />

{#if status}
	<Alert severity={status.kind}>{status.text}</Alert>
{/if}

<Card title={t('sources.title')} subtitle={t('sources.card_subtitle', { search: t('jobs.search_now') })}>
	{#snippet actions()}
		<Button onclick={openAddModal}>{t('sources.add_button')}</Button>
	{/snippet}
	{#if loading}
		<p class="hint">{t('sources.loading')}</p>
	{:else if sources.length === 0}
		<p class="hint">{t('sources.empty')}</p>
	{:else}
		<DataView
			columns={columns}
			rows={sources}
			getKey={(src) => src.id}
			caption={t('sources.caption')}
			render={renderCell}
		/>
	{/if}
</Card>

<Dialog open={showModal} onclose={() => (showModal = false)} title={modalMode === 'add' ? t('sources.dialog.add') : t('sources.dialog.edit', { id: formId })}>
	<form id="source-form" onsubmit={(e) => { e.preventDefault(); saveModal(); }}>
		{#if modalMode === 'add'}
			<Field label={t('sources.field.slug')} for="form_id">
				<input id="form_id" bind:value={formId} placeholder={t('sources.field.slug_placeholder')} required />
			</Field>
		{/if}
		<Field label={t('sources.field.name')} for="form_name">
			<input id="form_name" bind:value={formName} placeholder={t('sources.field.name_placeholder')} required />
		</Field>
		<Field label={t('sources.field.adapter')} for="form_adapter">
			<Select id="form_adapter" bind:value={formAdapter}>
				{#each adapters as adapter (adapter.slug)}
					<option value={adapter.slug}>{adapter.name} ({adapter.slug})</option>
				{/each}
			</Select>
		</Field>
		<div class="grid grid-2">
			<Field label={t('sources.field.priority')} for="form_priority">
				<input id="form_priority" type="number" min="1" max="999" bind:value={formPriority} />
			</Field>
			<Field label={t('sources.field.auth')} for="form_auth">
				<Select id="form_auth" bind:value={formAuthType}>
					<option value="none">{t('sources.field.auth_none')}</option>
					<option value="api_key">{t('sources.field.api_key')}</option>
				</Select>
			</Field>
		</div>
		{#if formAuthType === 'api_key'}
			<Field label={t('sources.field.api_key')} for="form_api_key" hint="Gravada cifrada com a sua master key e nunca exposta ao frontend.">
				<input id="form_api_key" type="password" bind:value={formApiKey} placeholder={modalMode === 'edit' ? t('sources.field.api_key_saved') : t('sources.field.api_key_hint')} />
			</Field>
		{/if}
		<Checkbox id="form_enabled" label={t('sources.field.enabled')} bind:checked={formEnabled} />
	</form>
	{#snippet footer()}
		<Button variant="ghost" onclick={() => (showModal = false)}>{t('common.cancel')}</Button>
		<Button type="submit" form="source-form" loading={saving}>{t('sources.button.save')}</Button>
	{/snippet}
</Dialog>

<style>
	.cell-note {
		display: block;
		font-size: var(--text-caption);
		color: var(--color-text-muted);
	}

	.mono {
		font-family: var(--font-mono);
		font-size: var(--text-mono);
	}

	.row-actions {
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-1);
	}

</style>
