import { goto } from '$app/navigation';

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

export const SHORTCUTS: readonly ShortcutDefinition[] = [
	{ id: 'jobs', keys: 'g j', label: 'Vagas', description: 'Ir para Vagas' },
	{ id: 'applications', keys: 'g a', label: 'Candidaturas', description: 'Ir para Candidaturas' },
	{ id: 'settings', keys: 'g s', label: 'Configurações', description: 'Ir para Configurações' },
	{ id: 'notifications', keys: 'g n', label: 'Notificações', description: 'Ir para Notificações' },
	{ id: 'tasks', keys: 'g t', label: 'Tarefas', description: 'Ir para Tarefas de IA' },
	{ id: 'search', keys: '/', label: 'Busca', description: 'Focar o campo de busca' },
	{ id: 'help', keys: '?', label: 'Ajuda', description: 'Abrir ajuda de atalhos' },
	{ id: 'rail', keys: '[', label: 'Recolher menu', description: 'Alternar sidebar/rail' },
	{ id: 'escape', keys: 'Esc', label: 'Fechar', description: 'Fechar overlay/detalhe' }
] as const;

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
