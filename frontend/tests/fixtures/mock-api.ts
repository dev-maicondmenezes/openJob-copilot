import type { Page, Route } from '@playwright/test';

/** Dados determinísticos para snapshots e auditorias — sem backend real. */
const job = (id: number) => ({
	id,
	title: id === 1 ? 'Senior Go Developer' : 'Backend Engineer',
	company: id === 1 ? 'Acme Corp' : 'Example Ltda',
	url: `https://jobs.example.com/vaga/${id}`,
	description: 'Vaga remota com Go, SQL e observabilidade.',
	source: id === 1 ? 'himalayas' : 'remotive',
	score_fit: id === 1 ? 82 : 61,
	score_reason: 'Aderência forte com Go e backend.',
	status: 'new',
	created_at: '2026-11-01T12:00:00Z',
	matched_keywords: ['Go', 'Backend'],
	matched_resume_terms: ['SQL']
});

const responses: Record<string, unknown> = {
	'/api/settings': { settings: { locale: 'pt-BR' }, locale: 'pt-BR' },
	'/api/settings/llm-key': { configured: true },
	'/api/settings/llm': { configured: true, base_url: 'https://api.openai.com/v1', model: 'gpt-4o-mini' },
	'/api/settings/discovery': {
		keywords: ['Go', 'Backend'],
		resume_filter: { enabled: true, min_matches: 2, match_mode: 'any' },
		discovery_interval: 'off',
		resume_keywords: ['Go', 'SQL']
	},
	'/api/settings/notifications': { in_app_enabled: true, retention_days: 90, browser_notifications: false },
	'/api/settings/tasks': { max_concurrency: 2, worker_poll_ms: 500 },
	'/api/jobs': { jobs: [job(1), job(2)] },
	'/api/jobs/discover/status': { running: false, latest_run: null },
	'/api/applications': {
		applications: [
			{
				id: 10,
				job_id: 1,
				status: 'pending',
				sent_at: null,
				updated_at: '2026-11-02T10:00:00Z',
				has_tailored: true,
				job: { title: 'Senior Go Developer', company: 'Acme Corp', url: 'https://jobs.example.com/vaga/1', score_fit: 82 }
			}
		]
	},
	'/api/notifications': {
		notifications: [
			{
				id: 'n1',
				title: 'Vaga pontuada',
				body: 'A vaga Senior Go Developer recebeu score 82.',
				severity: 'success',
				type: 'application_status',
				read_at: null,
				link: '/jobs/1',
				created_at: '2026-11-03T09:30:00Z'
			},
			{
				id: 'n2',
				title: '',
				body: '',
				i18n_key: 'notifications.application.status_changed',
				i18n_params: '{"job_label":"Senior Go Developer","from":"pending","to":"interview"}',
				severity: 'info',
				type: 'application_status',
				read_at: null,
				link: '/applications',
				created_at: '2026-11-03T10:00:00Z'
			}
		],
		next_cursor: null
	},
	'/api/tasks': { tasks: [] },
	'/api/sources': [
		{
			id: 'himalayas',
			name: 'Himalayas',
			adapter: 'himalayas',
			enabled: true,
			priority: 10,
			auth_type: 'none',
			has_api_key: false,
			capabilities: {
				supports_keyword_search: true,
				supports_pagination: true,
				max_results_per_run: 50,
				rate_limit: { requests: 60, period: 'minute' },
				attribution_required: false,
				experimental: false,
				recent_window_hours: 720
			},
			created_at: '2026-10-01T00:00:00Z',
			updated_at: '2026-11-01T00:00:00Z'
		}
	],
	'/api/sources/adapters': [{ slug: 'himalayas', name: 'Himalayas', capabilities: { rate_limit: { requests: 60, period: 'minute' } } }],
	'/api/curriculum': {
		curriculum: {
			id: 1,
			content_md: '# Currículo\n\nExperiência com Go, SQL e backend.',
			source_filename: 'curriculo.md',
			uploaded_at: '2026-10-15T00:00:00Z'
		}
	}
};

/** Intercepta chamadas de API com respostas fixas. */
export async function mockApi(page: Page): Promise<void> {
	// CP13: fixa o locale/tema para que os testes legados (que assertam
	// texto pt-BR e o visual snapshot light) permaneçam determinísticos.
	// Testes específicos de i18n/dark mode podem sobrescrever via
	// `page.evaluate` + `page.reload()`.
	await page.addInitScript(() => {
		localStorage.setItem('openjob.locale', 'pt-BR');
		localStorage.setItem('openjob.theme', 'light');
	});
	await page.route('**/api/**', async (route: Route) => {
		const url = new URL(route.request().url());
		const key = url.pathname;
		const method = route.request().method();

		if (key.startsWith('/api/tasks/stream')) {
			// EventSource: encerra imediatamente; a UI cai no fallback polling.
			await route.fulfill({ status: 200, contentType: 'text/event-stream', body: '' });
			return;
		}
		if (method !== 'GET') {
			await route.fulfill({ status: 204, body: '' });
			return;
		}
		const body = responses[key];
		if (body === undefined) {
			await route.fulfill({ status: 200, contentType: 'application/json', body: '{}' });
			return;
		}
		await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(body) });
	});
}

/** Navega e espera a hidratação do shell (atalhos e stores prontos). */
export async function gotoReady(page: Page, path: string): Promise<void> {
	await page.goto(path);
	await page.waitForSelector('html[data-app-ready="1"]');
}

/** Rotas canônicas usadas por visual/a11y. */
export const ROUTES = [
	'/',
	'/jobs',
	'/applications',
	'/curriculum',
	'/settings',
	'/settings/sources',
	'/settings/discovery',
	'/settings/notifications',
	'/notifications',
	'/tasks'
];
