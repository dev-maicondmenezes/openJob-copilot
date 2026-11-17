<script lang="ts">
	import { onMount } from 'svelte';
	import SettingsNav from '#lib/components/SettingsNav.svelte';
	import PageHeader from '#lib/ui/PageHeader.svelte';
	import Card from '#lib/ui/Card.svelte';
	import Panel from '#lib/ui/Panel.svelte';
	import Field from '#lib/ui/Field.svelte';
	import Input from '#lib/ui/Input.svelte';
	import Select from '#lib/ui/Select.svelte';
	import Switch from '#lib/ui/Switch.svelte';
	import Button from '#lib/ui/Button.svelte';
	import IconButton from '#lib/ui/IconButton.svelte';
	import Badge from '#lib/ui/Badge.svelte';
	import Alert from '#lib/ui/Alert.svelte';

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
	let keywordError = $state('');

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
			keywordError = 'Limite máximo de 5 palavras-chave atingido.';
			return;
		}
		if (keywords.some((k) => k.toLowerCase() === term.toLowerCase())) {
			keywordError = `A palavra-chave "${term}" já foi adicionada.`;
			return;
		}
		keywords = [...keywords, term];
		newKeyword = '';
		keywordError = '';
		status = null;
	}

	function removeKeyword(index: number) {
		keywords = keywords.filter((_, i) => i !== index);
		keywordError = '';
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
			const res = await fetch('/api/settings/discovery/resume-keywords/refresh', { method: 'POST' });
			if (!res.ok) {
				const err = await res.json().catch(() => ({}));
				status = { kind: 'error', text: err.error ?? 'Falha ao enfileirar extração de palavras.' };
				return;
			}
			const body = await res.json();
			const taskId = body.task_id as string | undefined;
			if (!taskId) {
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
					status = { kind: 'success', text: `Extração concluída: ${task.result?.count ?? 0} palavras-chave identificadas. Acompanhe em /tasks.` };
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

<PageHeader
	title="Configurações"
	subtitle="Parâmetros da descoberta automática de vagas e filtragem prévia por currículo."
	breadcrumb={[{ label: 'Início', href: '/' }, { label: 'Configurações', href: '/settings' }, { label: 'Busca & Filtros' }]}
/>

<SettingsNav active="discovery" />

{#if status}
	<Alert severity={status.kind}>{status.text}</Alert>
{/if}

{#if loading}
	<Card><p class="hint">Carregando configurações…</p></Card>
{:else}
	<form onsubmit={(e) => { e.preventDefault(); saveSettings(); }} class="discovery-form">
		<Card title="Palavras-chave de Busca" class="spacious-card" subtitle="Termos consultados nas APIs e aplicados como filtro sobre título e descrição das vagas.">
			{#snippet actions()}
				<Badge tone={keywords.length === 5 ? 'warning' : 'neutral'}>{keywords.length}/5 termos</Badge>
			{/snippet}
			<Field label="Adicionar termo" for="new_keyword" error={keywordError}>
				<div class="keyword-row">
					<Input
						id="new_keyword"
						bind:value={newKeyword}
						disabled={keywords.length >= 5}
						placeholder={keywords.length >= 5 ? 'Limite de 5 termos atingido' : 'Ex: Golang, Backend, Remote…'}
						onkeydown={(e) => { if (e.key === 'Enter') { e.preventDefault(); addKeyword(); } }}
					/>
					<Button type="button" variant="secondary" disabled={keywords.length >= 5 || !newKeyword.trim()} onclick={addKeyword}>
						+ Adicionar
					</Button>
				</div>
			</Field>
			<div class="chips">
				{#each keywords as keyword, i (keyword)}
					<span class="chip">
						{keyword}
						<IconButton icon="x" label={`Remover ${keyword}`} variant="ghost" size="sm" onclick={() => removeKeyword(i)} />
					</span>
				{:else}
					<p class="hint">Nenhum termo configurado — todas as vagas retornadas pelas fontes serão avaliadas.</p>
				{/each}
			</div>
		</Card>

		<Card title="Filtro de Aderência ao Currículo" class="spacious-card" subtitle="Descarta vagas sem correspondência mínima com suas competências antes do scoring de IA.">
			<Switch label="Filtro ativo" bind:checked={resumeFilterEnabled} />
			{#if resumeFilterEnabled}
				<div class="grid grid-2 form-grid">
					<Field label="Mínimo de correspondências" for="min_matches" hint="Recomendado: 2 a 3 termos para equilibrar relevância e volume.">
						<Input id="min_matches" type="number" min="1" max="10" bind:value={minMatches} />
					</Field>
					<Field label="Modo de correspondência" for="match_mode">
						<Select id="match_mode" bind:value={matchMode}>
							<option value="any">Qualquer termo (≥ mínimo configurado)</option>
							<option value="all">Todos os termos do currículo</option>
						</Select>
					</Field>
				</div>
				<Panel title={`Palavras-chave extraídas do currículo (${resumeKeywords.length})`}>
					{#snippet actions()}
						<Button type="button" variant="ghost" size="sm" loading={refreshingResume} onclick={refreshResume}>
							↻ Reextrair do currículo
						</Button>
					{/snippet}
					{#if resumeKeywords.length > 0}
						<div class="chips">
							{#each resumeKeywords as term (term)}<span class="chip neutral">{term}</span>{/each}
						</div>
					{:else}
						<p class="hint">Nenhuma palavra extraída ainda. Faça upload em <a href="/curriculum">Currículo</a> ou clique em “Reextrair”.</p>
					{/if}
				</Panel>
			{/if}
		</Card>

		<Card title="Busca Periódica Automática (Ticker)" class="spacious-card" subtitle="Executa a descoberta em segundo plano dentro do próprio processo openJob.">
			<Field label="Frequência de execução" for="interval" hint="Mantenha desativado para controlar totalmente quando e quanto consultar as APIs.">
				<Select id="interval" bind:value={discoveryInterval}>
					<option value="off">Desativada (apenas sob demanda)</option>
					<option value="1h">A cada 1 hora</option>
					<option value="6h">A cada 6 horas (recomendado)</option>
					<option value="24h">A cada 24 horas</option>
				</Select>
			</Field>
		</Card>

		<div class="form-actions">
			<Button type="submit" loading={saving}>Salvar Configurações</Button>
		</div>
	</form>
{/if}

<style>
	.discovery-form {
		display: flex;
		flex-direction: column;
	}

	.keyword-row {
		display: flex;
		gap: var(--space-2);
	}

	.keyword-row :global(.input) {
		flex: 1;
		min-width: 0;
	}

	.chips {
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-2);
		margin-top: var(--space-3);
	}

	.chip {
		display: inline-flex;
		align-items: center;
		gap: var(--space-2);
		padding: var(--space-1) var(--space-3);
		border-radius: var(--radius-pill);
		background: var(--color-info-soft);
		border: 1px solid var(--color-info-border);
		color: var(--color-primary);
		font-size: var(--text-caption);
		font-weight: var(--weight-medium);
	}

	.chip.neutral {
		background: var(--color-surface);
		border-color: var(--color-border);
		color: var(--color-text-secondary);
	}

	:global(.chip button) {
		border: 0;
		background: none;
		color: inherit;
		cursor: pointer;
		padding: 0;
		line-height: 1;
	}

	.form-grid {
		margin-top: var(--space-4);
	}

	:global(.spacious-card .card-body) {
		display: flex;
		flex-direction: column;
		gap: var(--space-6);
	}

	.form-actions {
		margin-top: var(--space-6);
		margin-bottom: var(--space-12);
	}
</style>
