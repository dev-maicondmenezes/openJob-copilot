<script lang="ts">
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

	const columns: DataColumn<Source>[] = [
		{ key: 'name', header: 'Nome / ID', cardLabel: 'Nome', value: (src) => src.name },
		{ key: 'adapter', header: 'Adapter', value: (src) => src.adapter },
		{ key: 'enabled', header: 'Ativa', value: (src) => (src.enabled ? 'Ativa' : 'Inativa') },
		{ key: 'auth', header: 'Autenticação', value: (src) => (src.auth_type === 'api_key' ? (src.has_api_key ? 'chave salva' : 'sem chave') : 'pública') },
		{ key: 'rate', header: 'Rate limit', value: (src) => (src.capabilities.rate_limit.requests > 0 ? `${src.capabilities.rate_limit.requests}/${src.capabilities.rate_limit.period}` : 'Padrão') },
		{ key: 'updated', header: 'Atualizada em', value: (src) => new Date(src.updated_at).toLocaleDateString('pt-BR') },
		{ key: 'actions', header: 'Ações', value: () => '' }
	];

	onMount(async () => {
		await Promise.all([loadSources(), loadAdapters()]);
	});

	async function loadSources() {
		try {
			const res = await fetch('/api/sources');
			if (res.ok) sources = await res.json();
			else status = { kind: 'error', text: 'Falha ao carregar fontes configuradas.' };
		} catch {
			status = { kind: 'error', text: 'Falha de rede ao listar fontes.' };
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
				status = { kind: 'error', text: err.error ?? 'Falha ao alterar status da fonte.' };
			}
		} catch {
			status = { kind: 'error', text: 'Falha de rede ao alterar status.' };
		}
	}

	async function testSource(id: string) {
		testingId = id;
		status = null;
		try {
			const res = await fetch(`/api/sources/${id}/test`, { method: 'POST' });
			const result = await res.json().catch(() => ({ ok: false, error: 'Erro desconhecido' }));
			testResults[id] = result;
			status = result.ok
				? { kind: 'success', text: `Teste concluído (${result.latency_ms}ms, ${result.count ?? 1} vaga).` }
				: { kind: 'error', text: `Falha no teste: ${result.error}` };
		} catch {
			status = { kind: 'error', text: 'Falha de rede ao testar fonte.' };
		} finally {
			testingId = null;
		}
	}

	async function deleteSource(id: string) {
		if (!confirm(`Deseja realmente remover a fonte "${id}"? As vagas já descobertas serão mantidas.`)) return;
		status = null;
		try {
			const res = await fetch(`/api/sources/${id}`, { method: 'DELETE' });
			if (res.ok) {
				status = { kind: 'success', text: 'Fonte removida com sucesso.' };
				await loadSources();
			} else {
				const err = await res.json().catch(() => ({}));
				status = { kind: 'error', text: err.error ?? 'Falha ao remover fonte.' };
			}
		} catch {
			status = { kind: 'error', text: 'Falha de rede ao remover fonte.' };
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
					status = { kind: 'success', text: 'Fonte adicionada com sucesso!' };
				} else {
					const err = await res.json().catch(() => ({}));
					status = { kind: 'error', text: err.error ?? 'Falha ao criar fonte.' };
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
					status = { kind: 'success', text: 'Fonte atualizada com sucesso!' };
				} else {
					const err = await res.json().catch(() => ({}));
					status = { kind: 'error', text: err.error ?? 'Falha ao atualizar fonte.' };
				}
			}
			await loadSources();
		} catch {
			status = { kind: 'error', text: 'Falha de rede ao salvar fonte.' };
		} finally {
			saving = false;
		}
	}
</script>

<svelte:head>
	<title>Fontes de Vagas · openJob</title>
</svelte:head>

{#snippet renderCell(src: Source, column: DataColumn<Source>)}
	{#if column.key === 'name'}
		<strong>{src.name}</strong>
		<span class="cell-note">{src.id}</span>
		{#if src.capabilities.attribution_required}<Badge tone="info">atribuição</Badge>{/if}
		{#if src.capabilities.experimental}<Badge tone="warning">experimental</Badge>{/if}
	{:else if column.key === 'adapter'}
		<code class="mono">{src.adapter}</code>
	{:else if column.key === 'enabled'}
		<Switch
			label={src.enabled ? 'Ativa' : 'Inativa'}
			checked={src.enabled}
			onchange={() => toggleEnabled(src)}
		/>
	{:else if column.key === 'auth'}
		<Badge tone={src.auth_type === 'api_key' ? (src.has_api_key ? 'success' : 'warning') : 'neutral'}>
			{src.auth_type === 'api_key' ? (src.has_api_key ? 'chave salva' : 'sem chave') : 'pública'}
		</Badge>
	{:else if column.key === 'rate'}
		{src.capabilities.rate_limit.requests > 0
			? `${src.capabilities.rate_limit.requests}/${src.capabilities.rate_limit.period === 'day' ? 'dia' : src.capabilities.rate_limit.period === 'hour' ? 'hora' : 'min'}`
			: 'Padrão'}
	{:else if column.key === 'actions'}
		<div class="row-actions">
			<Button variant="ghost" size="sm" loading={testingId === src.id} onclick={() => testSource(src.id)}>Testar</Button>
			<Button variant="ghost" size="sm" onclick={() => openEditModal(src)}>Editar</Button>
			<Button variant="ghost" size="sm" onclick={() => deleteSource(src.id)}>Excluir</Button>
		</div>
		{#if testResults[src.id]}
			<Alert severity={testResults[src.id].ok ? 'success' : 'error'}>
				{testResults[src.id].ok
					? `OK (${testResults[src.id].latency_ms}ms)`
					: testResults[src.id].error}
			</Alert>
		{/if}
	{:else}
		{column.value?.(src) ?? ''}
	{/if}
{/snippet}

<PageHeader
	title="Configurações"
	subtitle="Gestão de fontes públicas de vagas — APIs gratuitas oficiais, sem webscraping."
	breadcrumb={[{ label: 'Início', href: '/' }, { label: 'Configurações', href: '/settings' }, { label: 'Fontes' }]}
/>

<SettingsNav active="sources" />

{#if status}
	<Alert severity={status.kind}>{status.text}</Alert>
{/if}

<Card title="Fontes Configuradas" subtitle="Fontes ativas são consultadas na descoberta automática ou pelo botão “Buscar agora”.">
	{#snippet actions()}
		<Button onclick={openAddModal}>+ Adicionar Fonte</Button>
	{/snippet}
	{#if loading}
		<p class="hint">Carregando fontes…</p>
	{:else if sources.length === 0}
		<p class="hint">Nenhuma fonte cadastrada.</p>
	{:else}
		<DataView
			columns={columns}
			rows={sources}
			getKey={(src) => src.id}
			caption="Fontes de vagas configuradas"
			render={renderCell}
		/>
	{/if}
</Card>

<Dialog open={showModal} onclose={() => (showModal = false)} title={modalMode === 'add' ? 'Adicionar nova fonte' : `Editar fonte: ${formId}`}>
	<form id="source-form" onsubmit={(e) => { e.preventDefault(); saveModal(); }}>
		{#if modalMode === 'add'}
			<Field label="Identificador único (slug)" for="form_id">
				<Input id="form_id" bind:value={formId} placeholder="ex: himalayas-br" required />
			</Field>
		{/if}
		<Field label="Nome exibido" for="form_name">
			<Input id="form_name" bind:value={formName} placeholder="ex: Himalayas Brasil" required />
		</Field>
		<Field label="Adapter em código" for="form_adapter">
			<Select id="form_adapter" bind:value={formAdapter}>
				{#each adapters as adapter (adapter.slug)}
					<option value={adapter.slug}>{adapter.name} ({adapter.slug})</option>
				{/each}
			</Select>
		</Field>
		<div class="grid grid-2">
			<Field label="Prioridade (menor = primeiro)" for="form_priority">
				<Input id="form_priority" type="number" min="1" max="999" bind:value={formPriority} />
			</Field>
			<Field label="Tipo de autenticação" for="form_auth">
				<Select id="form_auth" bind:value={formAuthType}>
					<option value="none">Nenhuma (pública/gratuita)</option>
					<option value="api_key">API Key</option>
				</Select>
			</Field>
		</div>
		{#if formAuthType === 'api_key'}
			<Field label="API Key" for="form_api_key" hint="Gravada cifrada com a sua master key e nunca exposta ao frontend.">
				<Input id="form_api_key" type="password" bind:value={formApiKey} placeholder={modalMode === 'edit' ? '•••••••• (deixe vazio para manter)' : 'Cole a chave da API'} />
			</Field>
		{/if}
		<Checkbox id="form_enabled" label="Habilitar para busca automática" bind:checked={formEnabled} />
	</form>
	{#snippet footer()}
		<Button variant="ghost" onclick={() => (showModal = false)}>Cancelar</Button>
		<Button type="submit" form="source-form" loading={saving}>Salvar Fonte</Button>
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
