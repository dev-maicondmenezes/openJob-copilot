import { test, expect } from '@playwright/test';
import { mockApi, gotoReady } from '../fixtures/mock-api';

const appBase = {
	id: 10,
	job_id: 1,
	status: 'pending',
	stage_group: 'pending',
	stage_label: null,
	stage_source: 'manual',
	tags: [],
	sent_at: null,
	updated_at: '2026-11-02T10:00:00Z',
	has_tailored: true,
	job: {
		title: 'Senior Go Developer',
		company: 'Acme Corp',
		url: 'https://jobs.example.com/vaga/1',
		score_fit: 82
	}
};

test.beforeEach(async ({ page }) => {
	await mockApi(page);
});

test.describe('Checkpoint 17 — e-mail, pipeline e tags', () => {
	test('banner de sugestão de baixa confiança permite confirmar o avanço', async ({ page }) => {
		await page.route('**/api/email/messages**', (route) => {
			if (route.request().method() !== 'GET') return route.fallback();
			return route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({
					messages: [
						{
							id: 'msg-1',
							account_id: 'imap',
							message_id: '<msg-1@example.com>',
							thread_id: 'thread-1',
							from_address: 'no-reply@greenhouse.io',
							from_name: 'Greenhouse',
							subject: 'Next steps for your application',
							received_at: '2026-11-03T10:00:00Z',
							matched_application_id: 10,
							match_confidence: 0.6,
							stage_signal: 'interview',
							link_source: 'suggested'
						}
					]
				})
			});
		});

		let linkBody: Record<string, unknown> | null = null;
		await page.route('**/api/email/messages/*/link', async (route) => {
			linkBody = route.request().postDataJSON();
			await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ ok: true }) });
		});

		await gotoReady(page, '/applications');

		const banner = page.locator('.suggestions-banner');
		await expect(banner).toBeVisible();
		await expect(banner).toContainText('podem ter avançado');

		await page.getByRole('button', { name: /revisar sugestões/i }).click();
		await expect(page.getByRole('dialog')).toBeVisible();
		await page.getByRole('button', { name: /confirmar avanço/i }).click();

		await expect.poll(() => linkBody).not.toBeNull();
		expect(linkBody).toMatchObject({ application_id: 10, advance_stage: true });
		await expect(banner).toHaveCount(0);
	});

	test('cria tag de fase, aplica à candidatura e envia tag_ids no PATCH', async ({ page }) => {
		await page.route('**/api/tags**', async (route) => {
			if (route.request().method() === 'POST') {
				await route.fulfill({
					status: 200,
					contentType: 'application/json',
					body: JSON.stringify({ id: 5, namespace: 'phase', value: 'desafio_tecnico', label: 'Desafio técnico' })
				});
				return;
			}
			await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ tags: [] }) });
		});

		let patchBody: Record<string, unknown> | null = null;
		await page.route('**/api/applications/10', async (route) => {
			if (route.request().method() === 'PATCH') {
				patchBody = route.request().postDataJSON();
				await route.fulfill({
					status: 200,
					contentType: 'application/json',
					body: JSON.stringify({
						...appBase,
						tags: [{ id: 5, namespace: 'phase', value: 'desafio_tecnico', label: 'Desafio técnico' }]
					})
				});
				return;
			}
			await route.fallback();
		});

		await gotoReady(page, '/applications');
		await page.locator('.app-card').first().click();
		const dialog = page.getByRole('dialog');
		await expect(dialog).toBeVisible();

		await dialog.getByLabel(/nova tag/i).fill('Desafio técnico');
		await dialog.getByRole('button', { name: /criar tag/i }).click();
		await expect(dialog.getByText('Desafio técnico')).toBeVisible();
		await dialog.getByRole('button', { name: /^salvar$/i }).click();

		await expect.poll(() => patchBody).not.toBeNull();
		expect(patchBody).toMatchObject({ stage_group: 'pending', tag_ids: [5] });
		await expect(page.getByText('Tags atualizadas.')).toBeVisible();
	});

	test('filtra o pipeline por tag de fase', async ({ page }) => {
		const tagged = {
			...appBase,
			id: 11,
			tags: [{ id: 5, namespace: 'phase', value: 'desafio_tecnico', label: 'Desafio técnico' }]
		};
		await page.route('**/api/applications', (route) =>
			route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({ applications: [appBase, tagged] })
			})
		);
		await page.route('**/api/tags**', (route) =>
			route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({ tags: [{ id: 5, namespace: 'phase', value: 'desafio_tecnico', label: 'Desafio técnico' }] })
			})
		);

		await gotoReady(page, '/applications');
		await expect(page.locator('.app-card')).toHaveCount(2);

		const inline = page.locator('.filters-inline');
		await inline.getByRole('button', { name: /todas as tags/i }).click();
		await inline.getByRole('listbox').getByText('Desafio técnico').click();

		await expect(page.locator('.app-card')).toHaveCount(1);
		await expect(page.locator('.app-card').first()).toContainText('Desafio técnico');
	});

	test('/settings/email mostra provedores, privacidade e sincronização', async ({ page }) => {
		await gotoReady(page, '/settings/email');
		await expect(page.getByRole('heading', { name: /sincronização de e-mail/i })).toBeVisible();
		await expect(page.getByRole('button', { name: 'IMAP' })).toBeVisible();
		await expect(page.getByRole('button', { name: /gmail \(oauth\)/i })).toBeVisible();
		await expect(page.getByText(/credenciais são cifradas localmente/i)).toBeVisible();
		await expect(page.getByRole('button', { name: /sincronizar agora/i })).toBeDisabled();
	});
});
