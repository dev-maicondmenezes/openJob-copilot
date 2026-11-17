import { goto } from '$app/navigation';
import { t } from '#lib/i18n/t.js';

/**
 * Atalhos de teclado desktop (CP12 §6.5). Instalado uma única vez pelo layout.
 * Nunca dispara quando o foco está em campo de texto ou conteúdo editável.
 */

export type ShortcutId =
	| 'jobs'
	| 'applications'
	| 'settings'
	| 'notifications'
	| 'tasks'
	| 'help'
	| 'search'
	| 'rail'
	| 'escape';

export interface ShortcutDefinition {
	id: ShortcutId;
	keys: string;
	label: string;
	description: string;
}

export function getShortcuts(): readonly ShortcutDefinition[] {
	return [
		{ id: 'jobs', keys: 'g j', label: t('shortcut.jobs'), description: t('shortcut.jobs_desc') },
		{ id: 'applications', keys: 'g a', label: t('shortcut.applications'), description: t('shortcut.applications_desc') },
		{ id: 'settings', keys: 'g s', label: t('shortcut.settings'), description: t('shortcut.settings_desc') },
		{ id: 'notifications', keys: 'g n', label: t('shortcut.notifications'), description: t('shortcut.notifications_desc') },
		{ id: 'tasks', keys: 'g t', label: t('shortcut.tasks'), description: t('shortcut.tasks_desc') },
		{ id: 'search', keys: '/', label: t('shortcut.search'), description: t('shortcut.search_desc') },
		{ id: 'help', keys: '?', label: t('shortcut.help'), description: t('shortcut.help_desc') },
		{ id: 'rail', keys: '[', label: t('shortcut.collapse'), description: t('shortcut.collapse_desc') },
		{ id: 'escape', keys: 'Esc', label: t('shortcut.close'), description: t('shortcut.close_desc') }
	];
}

const ROUTES: Partial<Record<ShortcutId, string>> = {
	jobs: '/jobs',
	applications: '/applications',
	settings: '/settings',
	notifications: '/notifications',
	tasks: '/tasks'
};

type Handler = () => void;
const handlers = new Map<ShortcutId, Set<Handler>>();

/** Registra um handler para um atalho; retorna a função de remoção. */
export function onShortcut(id: ShortcutId, handler: Handler): () => void {
	let set = handlers.get(id);
	if (!set) {
		set = new Set();
		handlers.set(id, set);
	}
	set.add(handler);
	return () => set!.delete(handler);
}

function emit(id: ShortcutId): void {
	handlers.get(id)?.forEach((fn) => fn());
}

/** True quando o alvo é um campo onde digitar não deve disparar atalhos. */
export function isEditable(target: EventTarget | null): boolean {
	if (!(target instanceof HTMLElement)) return false;
	if (target.isContentEditable) return true;
	return Boolean(
		target.closest('input, textarea, select, [contenteditable="true"], [role="textbox"]')
	);
}

let pendingG = 0;
const SEQUENCE_WINDOW_MS = 900;

export function handleKeydown(event: KeyboardEvent): void {
	if (event.defaultPrevented || event.altKey || event.ctrlKey || event.metaKey) return;
	if (isEditable(event.target)) return;

	const key = event.key;

	// Escape deve funcionar mesmo se houver um "g" pendente.
	if (key === 'Escape') {
		pendingG = 0;
		emit('escape');
		return;
	}

	if (key === 'g' || key === 'G') {
		pendingG = Date.now();
		event.preventDefault();
		return;
	}

	if (Date.now() - pendingG < SEQUENCE_WINDOW_MS) {
		pendingG = 0;
		const target: Partial<Record<string, ShortcutId>> = {
			j: 'jobs',
			a: 'applications',
			s: 'settings',
			n: 'notifications',
			t: 'tasks'
		};
		const id = target[key.toLowerCase()];
		if (id) {
			event.preventDefault();
			void goto(ROUTES[id]!);
			return;
		}
	}

	switch (key) {
		case '?':
			event.preventDefault();
			emit('help');
			break;
		case '/':
			event.preventDefault();
			emit('search');
			break;
		case '[':
			event.preventDefault();
			emit('rail');
			break;
	}
}

export function installShortcuts(): () => void {
	if (typeof window === 'undefined') return () => {};
	window.addEventListener('keydown', handleKeydown);
	return () => window.removeEventListener('keydown', handleKeydown);
}
