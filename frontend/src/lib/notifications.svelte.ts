// Estado global do sino de notificações (CP10 §4.4): badge de não lidas com
// polling de 30s que pausa quando a aba está em background (Page Visibility).
import { writable } from 'svelte/store';

export interface NotificationItem {
	id: number;
	type: string;
	severity: string;
	title: string;
	body: string;
	link: string | null;
	read_at: string | null;
	created_at: string;
}

export const unreadCount = writable(0);

let timer: ReturnType<typeof setInterval> | null = null;
let started = false;

export function startNotificationPolling(): void {
	if (started || typeof document === 'undefined') return;
	started = true;

	const refresh = async () => {
		// Pausa em aba em background: o badge só interessa a quem está vendo.
		if (document.hidden) return;
		try {
			const res = await fetch('/api/notifications/unread-count');
			if (res.ok) {
				const data = await res.json();
				unreadCount.set(data.unread ?? 0);
			}
		} catch {
			// silencioso: rede cai, badge mantém o último valor
		}
	};

	refresh();
	timer = setInterval(refresh, 30_000);
	document.addEventListener('visibilitychange', () => {
		if (!document.hidden) void refresh();
	});
}

export function stopNotificationPolling(): void {
	if (timer) clearInterval(timer);
	timer = null;
	started = false;
}

export async function markNotificationRead(id: number): Promise<void> {
	await fetch(`/api/notifications/${id}/read`, { method: 'POST' });
	unreadCount.update((n) => Math.max(0, n - 1));
}

export async function markAllNotificationsRead(): Promise<void> {
	await fetch('/api/notifications/read-all', { method: 'POST' });
	unreadCount.set(0);
}
