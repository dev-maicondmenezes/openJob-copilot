<script lang="ts">
	import { onMount } from 'svelte';
	import {
		unreadCount,
		startNotificationPolling,
		markNotificationRead,
		markAllNotificationsRead,
		type NotificationItem
	} from '#lib/notifications.svelte.ts';
	import IconButton from '#lib/ui/IconButton.svelte';
	import Icon from '#lib/ui/Icon.svelte';
	import Alert from '#lib/ui/Alert.svelte';

	let open = $state(false);
	let items: NotificationItem[] = $state([]);
	let panelRef: HTMLDivElement | null = $state(null);

	onMount(() => {
		startNotificationPolling();
	});

	async function toggle(): Promise<void> {
		open = !open;
		if (open) await loadRecent();
	}

	async function loadRecent(): Promise<void> {
		try {
			const res = await fetch('/api/notifications?limit=10');
			if (res.ok) {
				const data = await res.json();
				items = data.notifications ?? [];
			}
		} catch {
			// silencioso
		}
	}

	async function clickItem(item: NotificationItem): Promise<void> {
		if (item.read_at === null) await markNotificationRead(item.id);
		open = false;
		if (item.link) window.location.assign(item.link);
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

	const sevIcon: Record<string, string> = {
		success: 'circle-check',
		warning: 'warning',
		error: 'circle-alert',
		info: 'info'
	};

	function onWindowClick(e: MouseEvent): void {
		if (open && panelRef && !panelRef.contains(e.target as Node)) open = false;
	}

	function onWindowKeydown(e: KeyboardEvent): void {
		if (open && e.key === 'Escape') open = false;
	}
</script>

<svelte:window onclick={onWindowClick} onkeydown={onWindowKeydown} />

<div class="bell-wrap" bind:this={panelRef}>
	<div class="bell-button">
		<IconButton
			icon="bell"
			label="Notificações"
			variant="ghost"
			iconSize={24}
			onclick={toggle}
			aria-expanded={open}
			aria-haspopup="dialog"
		/>
		{#if $unreadCount > 0}
			<span class="badge" aria-hidden="true">{$unreadCount > 99 ? '99+' : $unreadCount}</span>
		{/if}
	</div>

	{#if open}
		<div class="panel" role="dialog" aria-label="Notificações recentes">
			<header class="panel-head">
				<strong>Notificações</strong>
				<div class="head-actions">
					<IconButton
						icon="check"
						label="Marcar todas como lidas"
						variant="ghost"
						size="sm"
						onclick={() => markAllNotificationsRead().then(loadRecent)}
					/>
					<a
						href="/notifications"
						class="icon-link"
						aria-label="Ver todas"
						title="Ver todas"
						onclick={() => (open = false)}
					>
						<Icon name="arrow-up-right" size={20} />
					</a>
				</div>
			</header>

			{#if items.length === 0}
				<Alert title="Nada por aqui ainda" severity="info">
					Notificações aparecem quando uma candidatura muda de status ou uma busca termina.
				</Alert>
			{:else}
				{#each grouped as group (group.label)}
					<p class="day">{group.label}</p>
					{#each group.items as item (item.id)}
						<button
							type="button"
							class="item sev-{item.severity}"
							class:unread={item.read_at === null}
							onclick={() => clickItem(item)}
						>
							<Icon name={sevIcon[item.severity] ?? 'info'} size={20} />
							<span class="text">
								<strong>{item.title}</strong>
								<span class="body">{item.body}</span>
							</span>
						</button>
					{/each}
				{/each}
			{/if}
		</div>
	{/if}
</div>

<style>
	.bell-wrap {
		position: relative;
	}

	.bell-button {
		position: relative;
		display: inline-flex;
	}

	.badge {
		position: absolute;
		top: -4px;
		right: -4px;
		min-width: 18px;
		height: 18px;
		padding: 0 var(--space-1);
		border-radius: var(--radius-pill);
		background: var(--color-error);
		color: var(--color-on-primary);
		font-size: 11px;
		font-weight: var(--weight-bold);
		line-height: 18px;
		text-align: center;
		pointer-events: none;
	}

	.panel {
		position: absolute;
		right: 0;
		top: calc(100% + var(--space-2));
		width: min(22rem, calc(100vw - var(--space-8)));
		max-height: 24rem;
		overflow-y: auto;
		background: var(--color-surface);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-md);
		box-shadow: var(--shadow-md);
		z-index: 50;
		padding: var(--space-3);
	}

	.panel-head {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: var(--space-2);
		padding: var(--space-1) var(--space-2) var(--space-3);
	}

	.head-actions {
		display: flex;
		gap: var(--space-2);
		align-items: center;
		flex-wrap: wrap;
	}

	.icon-link {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		width: var(--touch-target);
		height: var(--touch-target);
		border-radius: var(--radius-md);
		color: var(--color-text-secondary);
	}
	.icon-link:hover {
		background: var(--color-surface-elevated);
	}

	.day {
		font-size: 11px;
		font-weight: var(--weight-bold);
		color: var(--color-text-muted);
		text-transform: uppercase;
		margin: var(--space-4) var(--space-3) var(--space-2);
	}

	.item {
		display: flex;
		gap: var(--space-3);
		width: 100%;
		text-align: left;
		background: none;
		border: 0;
		border-radius: var(--radius-md);
		padding: var(--space-3) var(--space-4);
		margin-bottom: var(--space-2);
		cursor: pointer;
		color: var(--color-text-primary);
	}

	.item:hover {
		background: var(--color-surface-elevated);
	}

	.item.unread {
		background: var(--color-info-soft);
	}

	.item.sev-success :global(.icon) {
		color: var(--color-success);
	}
	.item.sev-warning :global(.icon) {
		color: var(--color-warning);
	}
	.item.sev-error :global(.icon) {
		color: var(--color-error);
	}

	.text {
		display: flex;
		flex-direction: column;
		gap: 2px;
		min-width: 0;
	}

	.text strong {
		font-size: var(--text-caption);
	}

	.body {
		font-size: var(--text-caption);
		line-height: var(--leading-caption);
		color: var(--color-text-muted);
		overflow: hidden;
		display: -webkit-box;
		-webkit-line-clamp: 2;
		line-clamp: 2;
		-webkit-box-orient: vertical;
	}
</style>
