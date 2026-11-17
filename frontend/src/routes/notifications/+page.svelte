<script lang="ts">
	import { onMount } from 'svelte';
	import { markNotificationRead, markAllNotificationsRead, unreadCount, type NotificationItem } from '#lib/notifications.svelte.ts';
	import PageHeader from '#lib/ui/PageHeader.svelte';
	import Card from '#lib/ui/Card.svelte';
	import Section from '#lib/ui/Section.svelte';
	import Field from '#lib/ui/Field.svelte';
	import Select from '#lib/ui/Select.svelte';
	import Checkbox from '#lib/ui/Checkbox.svelte';
	import SegmentedControl from '#lib/ui/SegmentedControl.svelte';
	import Button from '#lib/ui/Button.svelte';
	import IconButton from '#lib/ui/IconButton.svelte';
	import Badge from '#lib/ui/Badge.svelte';
	import Alert from '#lib/ui/Alert.svelte';
	import EmptyState from '#lib/ui/EmptyState.svelte';
	import Icon from '#lib/ui/Icon.svelte';

	const types = [
		{ value: '', label: 'Todos os tipos' },
		{ value: 'application_status', label: 'Candidaturas' },
		{ value: 'discovery_result', label: 'Busca de vagas' },
		{ value: 'task_completed', label: 'Tarefas concluídas' },
		{ value: 'task_failed', label: 'Tarefas com falha' },
		{ value: 'source_error', label: 'Erros de fonte' }
	];
	const severities = [
		{ value: '', label: 'Todas' },
		{ value: 'info', label: 'Info' },
		{ value: 'success', label: 'Sucesso' },
		{ value: 'warning', label: 'Aviso' },
		{ value: 'error', label: 'Erro' }
	];

	let items: NotificationItem[] = $state([]);
	let cursor = $state<string | null>(null);
	let loading = $state(true);
	let error = $state('');
	let typeFilter = $state('');
	let severityFilter = $state('');
	let unreadOnly = $state(false);

	const sevIcon: Record<string, string> = {
		success: 'circle-check',
		warning: 'warning',
		error: 'circle-alert',
		info: 'info'
	};

	const grouped = $derived.by(() => {
		const groups: { label: string; items: NotificationItem[] }[] = [];
		for (const item of items) {
			const label = dayLabel(item.created_at);
			const last = groups[groups.length - 1];
			if (last && last.label === label) last.items.push(item);
			else groups.push({ label, items: [item] });
		}
		return groups;
	});

	async function load(reset = true): Promise<void> {
		if (reset) {
			cursor = null;
			items = [];
		}
		loading = true;
		error = '';
		try {
			const params = new URLSearchParams();
			if (typeFilter) params.set('type', typeFilter);
			if (severityFilter) params.set('severity', severityFilter);
			if (unreadOnly) params.set('unread', '1');
			if (cursor) params.set('cursor', cursor);
			params.set('limit', '50');
			const res = await fetch(`/api/notifications?${params}`);
			if (res.ok) {
				const data = await res.json();
				items = reset ? (data.notifications ?? []) : [...items, ...(data.notifications ?? [])];
				cursor = data.next_cursor || null;
			} else {
				const err = await res.json().catch(() => ({}));
				error = err.error ?? `Erro ${res.status} ao carregar notificações.`;
			}
		} catch {
			error = 'Falha de rede ao carregar notificações.';
		} finally {
			loading = false;
		}
	}

	async function openItem(item: NotificationItem): Promise<void> {
		if (item.read_at === null) {
			await markNotificationRead(item.id);
			item.read_at = new Date().toISOString();
		}
		if (item.link) window.location.assign(item.link);
	}

	async function clearRead(): Promise<void> {
		await fetch('/api/notifications', { method: 'DELETE' });
		await load();
	}

	function fmtWhen(iso: string): string {
		return new Date(iso).toLocaleString('pt-BR', { day: '2-digit', month: 'short', hour: '2-digit', minute: '2-digit' });
	}

	function dayLabel(iso: string): string {
		const d = new Date(iso);
		const today = new Date();
		const yesterday = new Date(today);
		yesterday.setDate(today.getDate() - 1);
		if (d.toDateString() === today.toDateString()) return 'Hoje';
		if (d.toDateString() === yesterday.toDateString()) return 'Ontem';
		return d.toLocaleDateString('pt-BR', { day: '2-digit', month: 'long' });
	}

	onMount(() => void load());
</script>

<svelte:head>
	<title>Notificações · openJob</title>
</svelte:head>

<PageHeader
	title="Notificações"
	subtitle="Tudo que aconteceu com suas candidaturas, buscas e tarefas de IA."
	breadcrumb={[{ label: 'Início', href: '/' }, { label: 'Notificações' }]}
>
	{#snippet actions()}
		<Badge tone="primary">{$unreadCount} não lida{$unreadCount === 1 ? '' : 's'}</Badge>
	{/snippet}
</PageHeader>

{#snippet filterActions()}
	<div class="filter-actions">
		<IconButton
			icon="check"
			label="Marcar todas como lidas"
			variant="ghost"
			size="sm"
			onclick={() => markAllNotificationsRead().then(() => load())}
		/>
		<IconButton
			icon="trash"
			label="Limpar lidas"
			variant="ghost"
			size="sm"
			onclick={clearRead}
		/>
	</div>
{/snippet}

<div class="filters-mobile">
	<SegmentedControl options={severities} value={severityFilter} label="Filtrar por severidade" onchange={(value) => { severityFilter = value; void load(); }} />
	{@render filterActions()}
</div>

<div class="filters-inline">
	<Field label="Tipo" for="filter-type">
		<Select id="filter-type" options={types} value={typeFilter} onchange={(e) => { typeFilter = e.currentTarget.value; void load(); }} />
	</Field>
	<Field label="Severidade" for="filter-severity">
		<Select id="filter-severity" options={severities} value={severityFilter} onchange={(e) => { severityFilter = e.currentTarget.value; void load(); }} />
	</Field>
	<Checkbox label="Somente não lidas" checked={unreadOnly} onchange={() => void load()} />
	{@render filterActions()}
</div>

{#if error}
	<Alert severity="error" title="Não foi possível carregar">{error}</Alert>
{/if}

{#if loading && items.length === 0}
	<Card><p class="hint">Carregando notificações…</p></Card>
{:else if items.length === 0}
	<Card>
		<EmptyState
			title="Nenhuma notificação por aqui"
			description="Elas aparecem quando uma candidatura muda de status, uma busca termina ou uma tarefa de IA conclui/falha."
		>
			<a class="cta" href="/settings/notifications">Configurar notificações</a>
		</EmptyState>
	</Card>
{:else}
	{#each grouped as group (group.label)}
		<Section title={group.label}>
			<div class="feed">
				{#each group.items as item (item.id)}
					<Card
						class="feed-card sev-{item.severity} {item.read_at === null ? 'unread' : ''}"
						onclick={() => openItem(item)}
					>
						<span class="sev-icon"><Icon name={sevIcon[item.severity] ?? 'info'} size={20} /></span>
						<span class="body">
							<strong>{item.title}</strong>
							<span class="text">{item.body}</span>
							<span class="when">{fmtWhen(item.created_at)}</span>
						</span>
						{#if item.read_at === null}<span class="dot" role="img" aria-label="Não lida"></span>{/if}
					</Card>
				{/each}
			</div>
		</Section>
	{/each}
	{#if cursor}
		<Button variant="secondary" loading={loading} onclick={() => load(false)}>Carregar mais</Button>
	{/if}
{/if}

<style>
	.filters-mobile {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: var(--space-3);
		flex-wrap: wrap;
		margin-bottom: var(--space-4);
	}

	.filters-inline {
		display: none;
		gap: var(--space-3);
		align-items: end;
		margin-bottom: var(--space-4);
	}

	.filter-actions {
		display: flex;
		align-items: center;
		gap: var(--space-1);
		margin-left: auto;
	}

	.filters-inline > :global(.field) {
		min-width: 12rem;
	}

	.feed {
		display: flex;
		flex-direction: column;
		gap: var(--space-2);
	}

	:global(.card.feed-card) {
		cursor: pointer;
	}

	:global(.card.feed-card .card-body) {
		display: flex;
		align-items: flex-start;
		gap: var(--space-3);
		text-align: left;
	}

	:global(.card.feed-card.unread) {
		border-color: var(--color-primary);
		background: var(--color-info-soft);
	}

	.sev-icon {
		color: var(--color-text-muted);
	}

	:global(.sev-success) .sev-icon { color: var(--color-success); }
	:global(.sev-warning) .sev-icon { color: var(--color-warning); }
	:global(.sev-error) .sev-icon { color: var(--color-error); }

	.body {
		display: flex;
		flex-direction: column;
		gap: var(--space-1);
		flex: 1;
		min-width: 0;
	}

	.body strong {
		font-size: var(--text-body);
	}

	.text,
	.when {
		font-size: var(--text-caption);
		line-height: var(--leading-caption);
		color: var(--color-text-muted);
	}

	.dot {
		width: 8px;
		height: 8px;
		margin-top: var(--space-2);
		border-radius: var(--radius-pill);
		background: var(--color-primary);
		flex-shrink: 0;
	}

	.cta {
		display: inline-flex;
		align-items: center;
		min-height: var(--touch-target);
		font-weight: var(--weight-semibold);
	}

	@media (min-width: 1024px) {
		.filters-mobile {
			display: none;
		}

		.filters-inline {
			display: flex;
		}
	}
</style>
