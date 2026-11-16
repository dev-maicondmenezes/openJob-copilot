<script lang="ts">
	import { onMount } from 'svelte';
	import SettingsNav from '#lib/components/SettingsNav.svelte';

	interface ResumeFilter {
		enabled: boolean;
		min_matches: number;
		match_mode: string;
		source_of_keywords: string;
	}

	interface DiscoverySettings {
		keywords: string[];
		resume_filter: ResumeFilter;
		discovery_interval: string;
		resume_keywords: string[];
	}

	let keywords = $state<string[]>([]);
	let newKeyword = $state('');
	let resumeFilterEnabled = $state(true);
	let minMatches = $state(3);
	let matchMode = $state('any');
	let discoveryInterval = $state('off');
	let resumeKeywords = $state<string[]>([]);

	let loading = $state(true);
	let saving = $state(false);
	let refreshingResume = $state(false);
	let status: { kind: 'success' | 'error'; text: string } | null = $state(null);

	async function loadConfig(): Promise<void> {
		const res = await fetch('/api/settings/discovery');
		if (res.ok) {
			const data: DiscoverySettings = await res.json();
			keywords = data.keywords ?? [];
			if (data.resume_filter) {
				resumeFilterEnabled = data.resume_filter.enabled ?? true;
				minMatches = data.resume_filter.min_matches ?? 3;
				matchMode = data.resume_filter.match_mode ?? 'any';
			}
			discoveryInterval = data.discovery_interval ?? 'off';
			resumeKeywords = data.resume_keywords ?? [];
		} else {
			status = { kind: 'error', text: 'Falha ao carregar configurações de busca.' };
		}
	}

	onMount(async () => {
		try {
			await loadConfig();
		} catch {
			status = { kind: 'error', text: 'Falha de rede ao carregar configurações.' };
		} finally {
			loading = false;
		}
	});

	function addKeyword() {
		const term = newKeyword.trim();
		if (!term) return;
		if (keywords.length >= 5) {
			status = { kind: 'error', text: 'Limite máximo de 5 palavras-chave atingido.' };
			return;
		}
		if (keywords.some((k) => k.toLowerCase() === term.toLowerCase())) {
			status = { kind: 'error', text: `A palavra-chave "${term}" já foi adicionada.` };
			return;
		}
		keywords = [...keywords, term];
		newKeyword = '';
		status = null;
	}

	function removeKeyword(index: number) {
		keywords = keywords.filter((_, i) => i !== index);
		status = null;
	}

	async function saveSettings() {
		saving = true;
		status = null;
		try {
			const res = await fetch('/api/settings/discovery', {
				method: 'PUT',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({
					keywords,
					resume_filter: {
						enabled: resumeFilterEnabled,
						min_matches: Number(minMatches),
						match_mode: matchMode,
						source_of_keywords: 'resume'
					},
					discovery_interval: discoveryInterval
				})
			});
			if (res.ok) {
				const updated: DiscoverySettings = await res.json();
				keywords = updated.keywords ?? [];
				status = { kind: 'success', text: 'Configurações de busca salvas com sucesso!' };
			} else {
				const err = await res.json().catch(() => ({}));
				status = { kind: 'error', text: err.error ?? 'Falha ao salvar configurações.' };
			}
		} catch {
			status = { kind: 'error', text: 'Falha de rede ao salvar configurações.' };
		} finally {
			saving = false;
		}
	}

	async function refreshResume() {
		refreshingResume = true;
		status = null;
		try {
			// CP10: o refresh virou tarefa assíncrona na fila (202 + task_id);
			// acompanhamos a tarefa em /api/tasks/{id} até ela terminar.
			const res = await fetch('/api/settings/discovery/resume-keywords/refresh', { method: 'POST' });
			if (!res.ok) {
				const err = await res.json().catch(() => ({}));
				status = { kind: 'error', text: err.error ?? 'Falha ao enfileirar extração de palavras.' };
				refreshingResume = false;
				return;
			}
			const body = await res.json();
			const taskId = body.task_id as string | undefined;
			if (!taskId) {
				// fallback síncrono (resposta legada)
				resumeKeywords = body.resume_keywords ?? [];
				status = { kind: 'success', text: `Extração concluída: ${resumeKeywords.length} palavras-chave identificadas.` };
				return;
			}

			for (let i = 0; i < 100; i++) {
				await new Promise((r) => setTimeout(r, 300));
				const tRes = await fetch(`/api/tasks/${taskId}`);
				if (!tRes.ok) continue;
				const task = await tRes.json();
				if (task.status === 'done') {
					await loadConfig();
					status = { kind: 'success', text: `Extração concluída: ${(task.result?.count ?? 0)} palavras-chave identificadas. Acompanhe em /tasks.` };
					return;
				}
				if (task.status === 'failed' || task.status === 'cancelled') {
					status = { kind: 'error', text: task.error ?? 'Extração de palavras-chave falhou.' };
					return;
				}
			}
			status = { kind: 'error', text: 'Extração demorou demais — acompanhe em /tasks.' };
		} catch {
			status = { kind: 'error', text: 'Falha de rede ao extrair palavras do currículo.' };
		} finally {
			refreshingResume = false;
		}
	}
</script>

<svelte:head>
	<title>Busca & Filtros · openJob</title>
</svelte:head>

<h1 class="page-title">Configurações</h1>
<p class="page-subtitle">
	Parâmetros da descoberta automática de vagas e filtragem prévia por currículo.
</p>

<SettingsNav active="discovery" />

{#if status}
	<div class="status {status.kind}" role="status">{status.text}</div>
{/if}

{#if loading}
	<p class="empty">Carregando configurações…</p>
{:else}
	<form onsubmit={(e) => { e.preventDefault(); saveSettings(); }}>
		<!-- Seção 1: Palavras-chave -->
		<section class="card" aria-label="Palavras-chave">
			<header class="card-head">
				<div>
					<h2>Palavras-chave de Busca</h2>
					<p class="hint">
						Termos consultados nas APIs das fontes e aplicados como filtro sobre o título e descrição das vagas.
					</p>
				</div>
				<span class="count-badge {keywords.length === 5 ? 'max' : ''}">
					{keywords.length}/5 termos
				</span>
			</header>

			<div class="keywords-input-row">
				<input
					type="text"
					placeholder={keywords.length >= 5 ? 'Limite de 5 termos atingido' : 'Ex: Golang, Backend, Remote…'}
					bind:value={newKeyword}
					disabled={keywords.length >= 5}
					onkeydown={(e) => { if (e.key === 'Enter') { e.preventDefault(); addKeyword(); } }}
				/>
				<button
					type="button"
					class="secondary"
					disabled={keywords.length >= 5 || !newKeyword.trim()}
					onclick={addKeyword}
				>
					+ Adicionar
				</button>
			</div>

			<div class="chips-list">
				{#each keywords as kw, i}
					<span class="keyword-chip">
						{kw}
						<button type="button" class="remove-chip" onclick={() => removeKeyword(i)} aria-label="Remover">✕</button>
					</span>
				{/each}
				{#if keywords.length === 0}
					<span class="hint-empty">Nenhum termo configurado — todas as vagas retornadas pelas fontes serão avaliadas.</span>
				{/if}
			</div>
		</section>

		<!-- Seção 2: Filtro por Currículo -->
		<section class="card" aria-label="Filtro por Currículo">
			<header class="card-head">
				<div>
					<h2>Filtro de Aderência ao Currículo</h2>
					<p class="hint">
						Descarta automaticamente vagas sem correspondência mínima com suas competências antes do scoring de IA.
					</p>
				</div>
				<div class="toggle-wrap">
					<input id="resume_enabled" type="checkbox" bind:checked={resumeFilterEnabled} />
					<label for="resume_enabled">Ativo</label>
				</div>
			</header>

			{#if resumeFilterEnabled}
				<div class="form-grid">
					<div>
						<label for="min_matches">Mínimo de correspondências necessárias</label>
						<input
							id="min_matches"
							type="number"
							min="1"
							max="10"
							bind:value={minMatches}
						/>
						<p class="hint">Recomendado: 2 a 3 termos para equilibrar relevância e volume.</p>
					</div>

					<div>
						<label for="match_mode">Modo de correspondência</label>
						<select id="match_mode" bind:value={matchMode}>
							<option value="any">Qualquer termo (≥ mínimo configurado)</option>
							<option value="all">Todos os termos do currículo</option>
						</select>
					</div>
				</div>

				<div class="resume-keywords-box">
					<div class="box-head">
						<span class="box-title">Palavras-chave extraídas do seu currículo ({resumeKeywords.length})</span>
						<button
							type="button"
							class="ghost small"
							disabled={refreshingResume}
							onclick={refreshResume}
						>
							{refreshingResume ? 'Extraindo…' : '↻ Reextrair do currículo'}
						</button>
					</div>

					{#if resumeKeywords.length > 0}
						<div class="resume-chips">
							{#each resumeKeywords as term}
								<span class="resume-chip">{term}</span>
							{/each}
						</div>
					{:else}
						<p class="hint-empty">
							Nenhuma palavra extraída ainda. Faça upload do seu currículo em <a href="/curriculum">Currículo</a> ou clique em "Reextrair".
						</p>
					{/if}
				</div>
			{/if}
		</section>

		<!-- Seção 3: Ticker / Busca periódica -->
		<section class="card" aria-label="Busca Periódica">
			<header class="card-head">
				<div>
					<h2>Busca Periódica Automática (Ticker)</h2>
					<p class="hint">
						Executa a descoberta em segundo plano dentro do próprio processo openJob (sem cron do sistema).
					</p>
				</div>
			</header>

			<label for="interval">Frequência de execução</label>
			<select id="interval" bind:value={discoveryInterval}>
				<option value="off">Desativada (apenas sob demanda no botão "Buscar agora")</option>
				<option value="1h">A cada 1 hora</option>
				<option value="6h">A cada 6 horas (Recomendado se ativo)</option>
				<option value="24h">A cada 24 horas</option>
			</select>
			<p class="hint">
				Mantenha desativado caso queira total controle de quando e quanto consultar as APIs.
			</p>
		</section>

		<div class="form-actions">
			<button type="submit" class="primary" disabled={saving}>
				{saving ? 'Salvando alterações…' : 'Salvar Configurações'}
			</button>
		</div>
	</form>
{/if}

<style>
	.card {
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: var(--radius);
		padding: 1.25rem;
		margin-bottom: 1.5rem;
	}

	.card-head {
		display: flex;
		align-items: flex-start;
		justify-content: space-between;
		margin-bottom: 1.25rem;
		gap: 1rem;
	}

	.card-head h2 {
		font-size: 1.125rem;
		margin: 0 0 0.25rem 0;
	}

	.count-badge {
		font-size: 0.8rem;
		font-weight: 600;
		background: var(--surface-soft, #f1f5f9);
		padding: 0.25rem 0.6rem;
		border-radius: 999px;
		color: var(--text-muted);
		white-space: nowrap;
	}

	.count-badge.max {
		background: #fee2e2;
		color: #b91c1c;
	}

	.keywords-input-row {
		display: flex;
		gap: 0.5rem;
		margin-bottom: 1rem;
	}

	.chips-list {
		display: flex;
		flex-wrap: wrap;
		gap: 0.5rem;
		min-height: 2.25rem;
		align-items: center;
	}

	.keyword-chip {
		display: inline-flex;
		align-items: center;
		gap: 0.35rem;
		background: var(--accent-soft, #eff6ff);
		color: var(--primary, #1d4ed8);
		border: 1px solid var(--border-soft, #bfdbfe);
		padding: 0.25rem 0.6rem;
		border-radius: 999px;
		font-size: 0.875rem;
		font-weight: 500;
	}

	.remove-chip {
		background: transparent;
		border: none;
		cursor: pointer;
		color: inherit;
		font-size: 0.8rem;
		padding: 0;
		line-height: 1;
	}

	.toggle-wrap {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		font-weight: 600;
		font-size: 0.875rem;
	}

	.form-grid {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 1.25rem;
		margin-bottom: 1.25rem;
	}

	.resume-keywords-box {
		background: var(--surface-soft, #f8fafc);
		border: 1px solid var(--border);
		border-radius: var(--radius);
		padding: 1rem;
	}

	.box-head {
		display: flex;
		justify-content: space-between;
		align-items: center;
		margin-bottom: 0.75rem;
	}

	.box-title {
		font-size: 0.85rem;
		font-weight: 600;
		color: var(--text-muted);
	}

	.resume-chips {
		display: flex;
		flex-wrap: wrap;
		gap: 0.35rem;
	}

	.resume-chip {
		background: var(--surface);
		border: 1px solid var(--border);
		font-size: 0.75rem;
		padding: 0.15rem 0.45rem;
		border-radius: 4px;
		color: var(--text);
	}

	.hint-empty {
		font-size: 0.85rem;
		color: var(--text-muted);
		font-style: italic;
	}

	.form-actions {
		display: flex;
		justify-content: flex-end;
		margin-top: 1rem;
		margin-bottom: 2rem;
	}

	.empty {
		text-align: center;
		padding: 2rem;
		color: var(--text-muted);
	}
</style>
