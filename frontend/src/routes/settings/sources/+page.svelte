<script lang="ts">
	import { onMount } from 'svelte';
	import SettingsNav from '#lib/components/SettingsNav.svelte';

	interface Capabilities {
		supports_keyword_search: boolean;
		supports_pagination: boolean;
		max_results_per_run: number;
		rate_limit: {
			requests: number;
			period: string;
		};
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

	// Modal / Form state
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

	onMount(async () => {
		await Promise.all([loadSources(), loadAdapters()]);
	});

	async function loadSources() {
		try {
			const res = await fetch('/api/sources');
			if (res.ok) {
				sources = await res.json();
			} else {
				status = { kind: 'error', text: 'Falha ao carregar fontes configuradas.' };
			}
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
				if (adapters.length > 0 && !formAdapter) {
					formAdapter = adapters[0].slug;
				}
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
			if (res.ok) {
				src.enabled = !src.enabled;
			} else {
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
			if (result.ok) {
				status = { kind: 'success', text: `Teste da fonte concluído com sucesso (${result.latency_ms}ms, ${result.count ?? 1} vaga encontrada).` };
			} else {
				status = { kind: 'error', text: `Falha no teste da fonte: ${result.error}` };
			}
		} catch {
			status = { kind: 'error', text: 'Falha de rede ao testar fonte.' };
		} finally {
			testingId = null;
		}
	}

	async function deleteSource(id: string) {
		if (!confirm(`Deseja realmente remover a fonte "${id}"? As vagas já descobertas serão mantidas.`)) {
			return;
		}
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
					await loadSources();
				} else {
					const err = await res.json().catch(() => ({}));
					status = { kind: 'error', text: err.error ?? 'Falha ao criar fonte.' };
				}
			} else {
				const payload: Record<string, any> = {
					name: formName,
					adapter: formAdapter,
					enabled: formEnabled,
					priority: Number(formPriority),
					auth_type: formAuthType
				};
				if (formApiKey.trim()) {
					payload.api_key = formApiKey.trim();
				}
				const res = await fetch(`/api/sources/${formId}`, {
					method: 'PATCH',
					headers: { 'Content-Type': 'application/json' },
					body: JSON.stringify(payload)
				});
				if (res.ok) {
					showModal = false;
					status = { kind: 'success', text: 'Fonte atualizada com sucesso!' };
					await loadSources();
				} else {
					const err = await res.json().catch(() => ({}));
					status = { kind: 'error', text: err.error ?? 'Falha ao atualizar fonte.' };
				}
			}
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

<h1 class="page-title">Configurações</h1>
<p class="page-subtitle">
	Gestão de fontes públicas de vagas — APIs gratuitas oficiais, sem webscraping.
</p>

<SettingsNav active="sources" />

{#if status}
	<div class="status {status.kind}" role="status">{status.text}</div>
{/if}

<section class="card" aria-label="Fontes de Vagas">
	<header class="card-head">
		<div>
			<h2>Fontes Configuradas</h2>
			<p class="hint">Fontes ativas são consultadas na descoberta automática ou pelo botão "Buscar agora".</p>
		</div>
		<button type="button" class="primary" onclick={openAddModal}>+ Adicionar Fonte</button>
	</header>

	{#if loading}
		<p class="empty">Carregando fontes…</p>
	{:else if sources.length === 0}
		<p class="empty">Nenhuma fonte cadastrada.</p>
	{:else}
		<div class="table-container">
			<table class="sources-table">
				<thead>
					<tr>
						<th>Nome / ID</th>
						<th>Adapter</th>
						<th>Ativa</th>
						<th>Prioridade</th>
						<th>Rate Limit</th>
						<th>Autenticação</th>
						<th>Ações</th>
					</tr>
				</thead>
				<tbody>
					{#each sources as src (src.id)}
						<tr class={!src.enabled ? 'disabled-row' : ''}>
							<td>
								<div class="source-info">
									<span class="source-name">{src.name}</span>
									<span class="source-id">{src.id}</span>
									{#if src.capabilities.attribution_required}
										<span class="badge info" title="Exige exibição de link para a fonte original">atribuição</span>
									{/if}
									{#if src.capabilities.experimental}
										<span class="badge warn" title="Fonte sem contrato público garantido">experimental</span>
									{/if}
								</div>
							</td>
							<td>
								<code class="adapter-tag">{src.adapter}</code>
							</td>
							<td>
								<button
									type="button"
									class="toggle-btn {src.enabled ? 'on' : 'off'}"
									onclick={() => toggleEnabled(src)}
									aria-label={src.enabled ? 'Desabilitar fonte' : 'Habilitar fonte'}
								>
									{src.enabled ? 'Ativa' : 'Inativa'}
								</button>
							</td>
							<td>
								<span class="priority-val">{src.priority}</span>
							</td>
							<td>
								{#if src.capabilities.rate_limit.requests > 0}
									<span class="rate-limit">
										{src.capabilities.rate_limit.requests}/{src.capabilities.rate_limit.period === 'day' ? 'dia' : src.capabilities.rate_limit.period === 'hour' ? 'hora' : 'min'}
									</span>
								{:else}
									<span class="rate-limit muted">Padrão</span>
								{/if}
							</td>
							<td>
								{#if src.auth_type === 'api_key'}
									<span class="auth-pill on">{src.has_api_key ? 'chave salva' : 'sem chave'}</span>
								{:else}
									<span class="auth-pill off">pública</span>
								{/if}
							</td>
							<td>
								<div class="row-actions">
									<button
										type="button"
										class="ghost small"
										disabled={testingId === src.id}
										onclick={() => testSource(src.id)}
									>
										{testingId === src.id ? 'Testando…' : 'Testar'}
									</button>
									<button
										type="button"
										class="ghost small"
										onclick={() => openEditModal(src)}
									>
										Editar
									</button>
									<button
										type="button"
										class="ghost small danger"
										onclick={() => deleteSource(src.id)}
									>
										Excluir
									</button>
								</div>
								{#if testResults[src.id]}
									<div class="test-result {testResults[src.id].ok ? 'ok' : 'err'}">
										{#if testResults[src.id].ok}
											✓ OK ({testResults[src.id].latency_ms}ms)
										{:else}
											✕ {testResults[src.id].error}
										{/if}
									</div>
								{/if}
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	{/if}
</section>

{#if showModal}
	<div
		class="modal-overlay"
		onclick={() => (showModal = false)}
		onkeydown={(e) => { if (e.key === 'Escape') showModal = false; }}
		role="presentation"
	>
		<div
			class="modal-card"
			tabindex="-1"
			onclick={(e) => e.stopPropagation()}
			onkeydown={(e) => e.stopPropagation()}
			role="dialog"
			aria-modal="true"
		>
			<header class="modal-head">
				<h3>{modalMode === 'add' ? 'Adicionar Nova Fonte' : `Editar Fonte: ${formId}`}</h3>
				<button type="button" class="close-btn" onclick={() => (showModal = false)}>✕</button>
			</header>

			<form onsubmit={(e) => { e.preventDefault(); saveModal(); }}>
				{#if modalMode === 'add'}
					<label for="form_id">Identificador único (slug)</label>
					<input
						id="form_id"
						type="text"
						placeholder="ex: himalayas-br"
						bind:value={formId}
						required
					/>
				{/if}

				<label for="form_name">Nome exibido</label>
				<input
					id="form_name"
					type="text"
					placeholder="ex: Himalayas Brasil"
					bind:value={formName}
					required
				/>

				<label for="form_adapter">Adapter em código</label>
				<select id="form_adapter" bind:value={formAdapter}>
					{#each adapters as a}
						<option value={a.slug}>{a.name} ({a.slug})</option>
					{/each}
				</select>

				<div class="form-grid">
					<div>
						<label for="form_priority">Prioridade (menor = primeiro)</label>
						<input
							id="form_priority"
							type="number"
							min="1"
							max="999"
							bind:value={formPriority}
						/>
					</div>
					<div>
						<label for="form_auth">Tipo de autenticação</label>
						<select id="form_auth" bind:value={formAuthType}>
							<option value="none">Nenhuma (Pública / Gratuita)</option>
							<option value="api_key">API Key (Chave opcional/gratuita)</option>
						</select>
					</div>
				</div>

				{#if formAuthType === 'api_key'}
					<label for="form_api_key">API Key</label>
					<input
						id="form_api_key"
						type="password"
						placeholder={modalMode === 'edit' ? '•••••••• (deixe vazio para manter)' : 'Cole a chave da API'}
						bind:value={formApiKey}
					/>
					<p class="hint">A chave é gravada cifrada com a sua master key e nunca exposta ao frontend.</p>
				{/if}

				<div class="checkbox-row">
					<input id="form_enabled" type="checkbox" bind:checked={formEnabled} />
					<label for="form_enabled">Habilitar esta fonte para busca automática</label>
				</div>

				<div class="modal-actions">
					<button type="button" class="secondary" onclick={() => (showModal = false)}>Cancelar</button>
					<button type="submit" class="primary" disabled={saving}>
						{saving ? 'Salvando…' : 'Salvar Fonte'}
					</button>
				</div>
			</form>
		</div>
	</div>
{/if}

<style>
	.card {
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: var(--radius);
		padding: 1.25rem;
	}

	.card-head {
		display: flex;
		align-items: flex-start;
		justify-content: space-between;
		margin-bottom: 1.25rem;
		flex-wrap: wrap;
		gap: 0.75rem;
	}

	.card-head h2 {
		font-size: 1.125rem;
		margin: 0 0 0.25rem 0;
	}

	.table-container {
		overflow-x: auto;
	}

	.sources-table {
		width: 100%;
		border-collapse: collapse;
		font-size: 0.875rem;
		text-align: left;
	}

	.sources-table th,
	.sources-table td {
		padding: 0.75rem 0.5rem;
		border-bottom: 1px solid var(--border);
	}

	.sources-table th {
		font-weight: 600;
		color: var(--text-muted);
		font-size: 0.75rem;
		text-transform: uppercase;
		letter-spacing: 0.05em;
	}

	.disabled-row {
		opacity: 0.6;
	}

	.source-info {
		display: flex;
		flex-direction: column;
		gap: 0.125rem;
	}

	.source-name {
		font-weight: 600;
	}

	.source-id {
		font-size: 0.75rem;
		color: var(--text-muted);
	}

	.badge {
		display: inline-block;
		font-size: 0.65rem;
		font-weight: 600;
		padding: 0.1rem 0.35rem;
		border-radius: 4px;
		margin-top: 0.125rem;
		width: fit-content;
	}

	.badge.info {
		background: #eff6ff;
		color: #1d4ed8;
	}

	.badge.warn {
		background: #fffbeb;
		color: #b45309;
	}

	.adapter-tag {
		font-family: monospace;
		font-size: 0.8rem;
		background: var(--surface-soft, #f1f5f9);
		padding: 0.15rem 0.4rem;
		border-radius: 4px;
	}

	.toggle-btn {
		font-size: 0.75rem;
		font-weight: 600;
		padding: 0.2rem 0.5rem;
		border-radius: 999px;
		border: 1px solid transparent;
		cursor: pointer;
	}

	.toggle-btn.on {
		background: #ecfdf5;
		color: #047857;
		border-color: #a7f3d0;
	}

	.toggle-btn.off {
		background: #f1f5f9;
		color: #64748b;
	}

	.priority-val {
		font-weight: 600;
		color: var(--text-muted);
	}

	.rate-limit {
		font-size: 0.8rem;
	}

	.rate-limit.muted {
		color: var(--text-muted);
	}

	.auth-pill {
		font-size: 0.75rem;
		padding: 0.125rem 0.4rem;
		border-radius: 4px;
	}

	.auth-pill.on {
		background: #ecfdf5;
		color: #047857;
	}

	.auth-pill.off {
		background: #f1f5f9;
		color: #64748b;
	}

	.row-actions {
		display: flex;
		gap: 0.25rem;
	}

	.row-actions button.small {
		padding: 0.2rem 0.4rem;
		font-size: 0.75rem;
	}

	.row-actions button.danger {
		color: #dc2626;
	}

	.test-result {
		font-size: 0.7rem;
		margin-top: 0.25rem;
	}

	.test-result.ok {
		color: #047857;
	}

	.test-result.err {
		color: #dc2626;
	}

	.empty {
		text-align: center;
		padding: 2rem;
		color: var(--text-muted);
	}

	/* Modal */
	.modal-overlay {
		position: fixed;
		inset: 0;
		background: rgba(0, 0, 0, 0.5);
		display: flex;
		align-items: center;
		justify-content: center;
		z-index: 100;
		padding: 1rem;
	}

	.modal-card {
		background: var(--surface);
		border-radius: var(--radius);
		padding: 1.5rem;
		width: 100%;
		max-width: 480px;
		box-shadow: 0 10px 25px rgba(0, 0, 0, 0.2);
	}

	.modal-head {
		display: flex;
		justify-content: space-between;
		align-items: center;
		margin-bottom: 1rem;
	}

	.modal-head h3 {
		margin: 0;
		font-size: 1.125rem;
	}

	.close-btn {
		background: transparent;
		border: none;
		font-size: 1.25rem;
		cursor: pointer;
		color: var(--text-muted);
	}

	.form-grid {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 1rem;
	}

	.checkbox-row {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		margin: 1rem 0;
	}

	.modal-actions {
		display: flex;
		justify-content: flex-end;
		gap: 0.75rem;
		margin-top: 1.5rem;
	}
</style>
