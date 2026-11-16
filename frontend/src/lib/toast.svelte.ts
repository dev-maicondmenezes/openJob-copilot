// Toast client-side (CP10 §4.4): efêmero, não persiste — o servidor persiste
// notificações de verdade via event bus; isto é apenas feedback imediato.
import { writable } from 'svelte/store';

export interface Toast {
	id: number;
	kind: 'success' | 'error' | 'info';
	text: string;
}

let nextId = 1;

export const toasts = writable<Toast[]>([]);

export function toast(kind: Toast['kind'], text: string, ttlMs = 4000): void {
	const id = nextId++;
	toasts.update((all) => [...all, { id, kind, text }]);
	setTimeout(() => {
		toasts.update((all) => all.filter((t) => t.id !== id));
	}, ttlMs);
}
