<script lang="ts">
	import { onMount } from 'svelte';
	import {
		unreadCount,
		startNotificationPolling,
		markNotificationRead,
		markAllNotificationsRead,
		type NotificationItem
	} from '#lib/notifications.svelte.ts';

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

	// Agrupa por dia, em ordem de inserção (feed já vem mais novo primeiro).
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

	const sevIcon = $derived.by<Record<string, string>>(() => ({
		success: '✔',
		warning: '⚠',
		error: '✖',
		info: 'ℹ'
	}));

	function onWindowClick(e: MouseEvent): void {
		if (open && panelRef && !panelRef.contains(e.target as Node)) open = false;
	}
</script>

<svelte:window onclick={onWindowClick} />

<div class="bell-wrap" bind:this={panelRef}>
	<button
		class="bell"
		aria-label="Notificações"
		title="Notificações"
		onclick={toggle}
	>
		<svg viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
			<path d="M18 8a6 6 0 0 0-12 0c0 7-3 9-3 9h18s-3-2-3-9" />
			<path d="M13.7 21a2 2 0 0 1-3.4 0" />
		</svg>
		{#if $unreadCount > 0}
			<span class="badge">{$unreadCount > 99 ? '99+' : $unreadCount}</span>
		{/if}
	</button>

	{#if open}
		<div class="panel" role="dialog" aria-label="Notificações recentes">
			<header class="panel-head">
				<strong>Notificações</strong>
				<div class="head-actions">
					<button class="ghost" onclick={() => markAllNotificationsRead().then(loadRecent)}>
						Marcar todas como lidas
					</button>
					<a class="ghost" href="/notifications" onclick={() => (open = false)}>Ver todas</a>
				</div>
			</header>

			{#if items.length === 0}
				<p class="empty">Nada por aqui ainda.</p>
			{:else}
				{#each grouped as group (group.label)}
					<p class="day">{group.label}</p>
					{#each group.items as item (item.id)}
						<button
							class="item sev-{item.severity}"
							class:unread={item.read_at === null}
							onclick={() => clickItem(item)}
						>
							<span class="icon">{sevIcon[item.severity] ?? 'ℹ'}</span>
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

	.bell {
		position: relative;
		display: inline-flex;
		align-items: center;
		justify-content: center;
		width: 2.4rem;
		height: 2.4rem;
		border: 1px solid var(--border);
		border-radius: 999px;
		background: var(--surface);
		color: var(--text);
		cursor: pointer;
	}

	.badge {
		position: absolute;
		top: -4px;
		right: -6px;
		background: #dc2626;
		color: #fff;
		font-size: 0.65rem;
		font-weight: 700;
		border-radius: 999px;
		padding: 0.05rem 0.35rem;
		min-width: 1rem;
		text-align: center;
	}

	.panel {
		position: absolute;
		right: 0;
		top: calc(100% + 0.5rem);
		width: min(22rem, calc(100vw - 2rem));
		max-height: 24rem;
		overflow-y: auto;
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: var(--radius);
		box-shadow: 0 8px 24px rgba(0, 0, 0, 0.12);
		z-index: 50;
		padding: 0.5rem;
	}

	.panel-head {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 0.5rem;
		padding: 0.25rem 0.25rem 0.5rem;
	}

	.head-actions {
		display: flex;
		gap: 0.5rem;
		align-items: center;
	}

	.ghost {
		background: none;
		border: none;
		color: var(--accent);
		font-size: 0.75rem;
		cursor: pointer;
		padding: 0;
		text-decoration: none;
	}

	.day {
		font-size: 0.7rem;
		font-weight: 700;
		color: var(--text-muted);
		text-transform: uppercase;
		margin: 0.5rem 0.25rem 0.25rem;
	}

	.item {
		display: flex;
		gap: 0.5rem;
		width: 100%;
		text-align: left;
		background: none;
		border: none;
		border-radius: var(--radius);
		padding: 0.5rem;
		cursor: pointer;
		color: var(--text);
	}

	.item:hover {
		background: var(--surface-soft);
	}

	.item.unread {
		background: var(--accent-soft);
	}

	.icon {
		flex-shrink: 0;
		width: 1.1rem;
		text-align: center;
		color: var(--text-muted);
	}

	.item.sev-success .icon { color: #047857; }
	.item.sev-warning .icon { color: #b45309; }
	.item.sev-error .icon { color: #b91c1c; }

	.text {
		display: flex;
		flex-direction: column;
		gap: 0.1rem;
		min-width: 0;
	}

	.text strong {
		font-size: 0.85rem;
	}

	.body {
		font-size: 0.78rem;
		color: var(--text-muted);
		overflow: hidden;
		display: -webkit-box;
		-webkit-line-clamp: 2;
		line-clamp: 2;
		-webkit-box-orient: vertical;
	}

	.empty {
		color: var(--text-muted);
		text-align: center;
		padding: 1rem 0;
		font-size: 0.85rem;
	}
</style>
