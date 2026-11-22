<script lang="ts">
	import { t } from '#lib/i18n/t.js';
	import { errorText } from '#lib/i18n/errors.js';
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
			status = { kind: 'error', text: t('discovery.error.load') };
		}
	}

	onMount(async () => {
		try {
			await loadConfig();
		} catch {
			status = { kind: 'error', text: t('discovery.error.load_network') };
		} finally {
			loading = false;
		}
	});

	function addKeyword() {
		const term = newKeyword.trim();
		if (!term) return;
		if (keywords.length >= 5) {
			keywordError = t('discovery.error.keyword_limit');
			return;
		}
		if (keywords.some((k) => k.toLowerCase() === term.toLowerCase())) {
			keywordError = t('discovery.error.keyword_duplicate', { term });
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
				status = { kind: 'success', text: t('discovery.error.saved') };
			} else {
				const err = await res.json().catch(() => ({}));
				status = { kind: 'error', text: errorText(err, t('discovery.error.save')) };
			}
		} catch {
			status = { kind: 'error', text: t('discovery.error.save_network') };
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
				status = { kind: 'error', text: errorText(err, t('discovery.error.enqueue')) };
				return;
			}
			const body = await res.json();
			const taskId = body.task_id as string | undefined;
			if (!taskId) {
				resumeKeywords = body.resume_keywords ?? [];
				status = { kind: 'success', text: t('discovery.error.extract_done', { count: resumeKeywords.length }) };
				return;
			}
			for (let i = 0; i < 100; i++) {
				await new Promise((r) => setTimeout(r, 300));
				const tRes = await fetch(`/api/tasks/${taskId}`);
				if (!tRes.ok) continue;
				const task = await tRes.json();
				if (task.status === 'done') {
					await loadConfig();
					status = { kind: 'success', text: t('discovery.error.extract_done_tasks', { count: task.result?.count ?? 0 }) };
					return;
				}
				if (task.status === 'failed' || task.status === 'cancelled') {
					status = { kind: 'error', text: errorText(task, t('discovery.error.extract_failed')) };
					return;
				}
			}
			status = { kind: 'error', text: t('discovery.error.extract_timeout') };
		} catch {
			status = { kind: 'error', text: t('discovery.error.extract_network') };
		} finally {
			refreshingResume = false;
		}
	}
</script>

<svelte:head>
	<title>{t('discovery.meta_title')}</title>
</svelte:head>

<PageHeader
	title={t('common.settings')}
	subtitle={t('discovery.subtitle')}
	breadcrumb={[{ label: t('common.home'), href: '/' }, { label: t('common.settings'), href: '/settings' }, { label: t('nav.settings_discovery') }]}
/>

<SettingsNav active="discovery" />

{#if status}
	<Alert severity={status.kind}>{status.text}</Alert>
{/if}

{#if loading}
	<Card><p class="hint">{t('discovery.loading')}</p></Card>
{:else}
	<form onsubmit={(e) => { e.preventDefault(); saveSettings(); }} class="discovery-form">
		<Card title={t('discovery.card.keywords')} class="spacious-card" subtitle={t('discovery.card.keywords_subtitle')}>
			{#snippet actions()}
				<Badge tone={keywords.length === 5 ? 'warning' : 'neutral'}>{t('discovery.keywords.count', { count: keywords.length })}</Badge>
			{/snippet}
			<Field label={t('discovery.keywords.add')} for="new_keyword" error={keywordError}>
				<div class="keyword-row">
					<input
						id="new_keyword"
						bind:value={newKeyword}
						disabled={keywords.length >= 5}
						placeholder={keywords.length >= 5 ? t('discovery.keywords.limit') : t('discovery.keywords.placeholder')}
						onkeydown={(e) => { if (e.key === 'Enter') { e.preventDefault(); addKeyword(); } }}
					/>
					<Button type="button" variant="secondary" disabled={keywords.length >= 5 || !newKeyword.trim()} onclick={addKeyword}>
						{t('discovery.keywords.add_button')}
					</Button>
				</div>
			</Field>
			<div class="chips">
				{#each keywords as keyword, i (keyword)}
					<span class="chip">
						{keyword}
						<IconButton icon="x" label={`${t('common.remove')} ${keyword}`} variant="ghost" size="sm" onclick={() => removeKeyword(i)} />
					</span>
				{:else}
					<p class="hint">{t('discovery.keywords.empty')}</p>
				{/each}
			</div>
		</Card>

		<Card title={t('discovery.card.resume_filter')} class="spacious-card" subtitle={t('discovery.card.resume_filter_subtitle')}>
			<Switch label={t('discovery.resume_filter.active')} bind:checked={resumeFilterEnabled} />
			{#if resumeFilterEnabled}
				<div class="grid grid-2 form-grid">
					<Field label={t('discovery.resume_filter.min_matches')} for="min_matches" hint={t('discovery.resume_filter.min_hint_detail')}>
						<input id="min_matches" type="number" min="1" max="10" bind:value={minMatches} />
					</Field>
					<Field label={t('discovery.resume_filter.mode')} for="match_mode">
						<Select id="match_mode" bind:value={matchMode}>
							<option value="any">{t('discovery.resume_filter.mode_any')}</option>
							<option value="all">{t('discovery.resume_filter.mode_all')}</option>
						</Select>
					</Field>
				</div>
				<Panel title={t('discovery.resume_filter.extracted', { count: resumeKeywords.length })}>
					{#snippet actions()}
						<Button type="button" variant="ghost" size="sm" loading={refreshingResume} onclick={refreshResume}>
							{t('discovery.resume_filter.reextract')}
						</Button>
					{/snippet}
					{#if resumeKeywords.length > 0}
						<div class="chips">
							{#each resumeKeywords as term (term)}<span class="chip neutral">{term}</span>{/each}
						</div>
					{:else}
						<p class="hint">{t('discovery.resume_filter.no_extracted_rich')}</p>
					{/if}
				</Panel>
			{/if}
		</Card>

		<Card title={t('discovery.card.ticker')} class="spacious-card" subtitle={t('discovery.card.ticker_subtitle', { app: t('common.app_name') })}>
			<Field label={t('discovery.ticker.frequency')} for="interval" hint={t('discovery.ticker.hint')}>
				<Select id="interval" bind:value={discoveryInterval}>
					<option value="off">{t('discovery.ticker.off')}</option>
					<option value="1h">{t('discovery.ticker.hourly')}</option>
					<option value="6h">{t('discovery.ticker.every_6h')}</option>
					<option value="24h">{t('discovery.ticker.daily')}</option>
				</Select>
			</Field>
		</Card>

		<div class="form-actions">
			<Button type="submit" loading={saving}>{t('discovery.button.save')}</Button>
		</div>
	</form>
{/if}

<style>
	.discovery-form {
		display: flex;
		flex-direction: column;
		gap: var(--space-6);
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
