// Fluxo das tarefas de IA (CP10 §4.4): assina o SSE /api/tasks/stream com
// fallback automático para polling em /api/tasks (o EventSource do navegador
// já reconecta com backoff próprio — critério §5).
import { writable } from 'svelte/store';

export type TaskStatus = 'queued' | 'running' | 'done' | 'failed' | 'cancelled';

export interface TaskItem {
	id: string;
	type: string;
	status: TaskStatus;
	payload: unknown;
	result: unknown;
	error: string | null;
	progress: number;
	attempts: number;
	max_attempts: number;
	priority: number;
	created_at: string;
	started_at: string | null;
	finished_at: string | null;
}

export interface TaskCounts {
	queued?: number;
	running?: number;
	done?: number;
	failed?: number;
	cancelled?: number;
}

export const tasks = writable<TaskItem[]>([]);
export const taskCounts = writable<TaskCounts>({});
export const tasksStreamConnected = writable(false);

let es: EventSource | null = null;
let pollTimer: ReturnType<typeof setInterval> | null = null;
let started = false;

export interface TasksQuery {
	status?: string; // "queued,running"
	type?: string;
}

function buildQuery(q: TasksQuery): string {
	const params = new URLSearchParams();
	if (q.status) params.set('status', q.status);
	if (q.type) params.set('type', q.type);
	const qs = params.toString();
	return qs ? `?${qs}` : '';
}

export async function refreshTasks(q: TasksQuery = {}): Promise<void> {
	try {
		const res = await fetch(`/api/tasks${buildQuery(q)}`);
		if (res.ok) {
			const data = await res.json();
			tasks.set(data.tasks ?? []);
			taskCounts.set(data.status_counts ?? {});
		}
	} catch {
		// silencioso (fallback de rede)
	}
}

export async function cancelTask(id: string): Promise<boolean> {
	const res = await fetch(`/api/tasks/${id}/cancel`, { method: 'POST' });
	return res.ok;
}

export async function retryTask(id: string): Promise<boolean> {
	const res = await fetch(`/api/tasks/${id}/retry`, { method: 'POST' });
	return res.ok;
}

/**
 * startTasksStream assina o SSE (mudanças de estado + progresso) e agenda um
 * polling de refresco; se a conexão cair, o EventSource reconecta e, enquanto
 * reconecta, o polling mantém a UI coerente.
 */
export function startTasksStream(q: TasksQuery = {}, pollMs = 2000): void {
	if (started || typeof EventSource === 'undefined') return;
	started = true;

	void refreshTasks(q);

	es = new EventSource('/api/tasks/stream');
	es.addEventListener('open', () => tasksStreamConnected.set(true));
	es.addEventListener('error', () => {
		// EventSource reconecta sozinho; enquanto isso, polling cobre o gap.
		tasksStreamConnected.set(false);
	});
	es.addEventListener('task', () => {
		// Qualquer mudança de estado/progresso → reconsulta a lista filtrada.
		void refreshTasks(q);
	});

	pollTimer = setInterval(() => void refreshTasks(q), pollMs * 3);
}

export function stopTasksStream(): void {
	if (es) {
		es.close();
		es = null;
	}
	if (pollTimer) clearInterval(pollTimer);
	pollTimer = null;
	tasksStreamConnected.set(false);
	started = false;
}
