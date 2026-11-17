import { test, expect } from '@playwright/test';
import { mockApi, gotoReady } from '../fixtures/mock-api';

test.beforeEach(async ({ page }) => {
	await mockApi(page);
});

test.describe('correções de UI/UX', () => {
	test('Nova Vaga abre em popup e o card saiu da página', async ({ page }) => {
		await gotoReady(page, '/jobs');
		await expect(page.getByText('Cole o link — os dados são preenchidos')).toHaveCount(0);

		await page.getByRole('button', { name: /nova vaga/i }).first().click();
		const dialog = page.getByRole('dialog');
		await expect(dialog).toBeVisible();
		await expect(dialog.getByLabel('URL da vaga')).toBeVisible();
		await expect(dialog.getByLabel('Descrição da vaga (JD)')).toBeVisible();
	});

	test('resumo de palavras-chave e ícones de configuração dos filtros', async ({ page }) => {
		await gotoReady(page, '/jobs');
		await expect(page.getByText('Palavras-chave configuradas')).toBeVisible();
		await expect(page.locator('a[title="Configurar busca e filtros"]')).toBeVisible();
		await expect(page.locator('a[title="Configurar fontes de vagas"]:visible')).toBeVisible();
		await expect(page.locator('button[title="Limpar filtros"]:visible')).toHaveCount(0);

		await page.locator('#inline-status').selectOption('new');
		const clear = page.locator('button[title="Limpar filtros"]:visible');
		await expect(clear).toBeVisible();
		// "Limpar filtros" vem antes do ícone de fontes.
		const order = await page.evaluate(() => {
			const root = document.querySelector('.list-actions')!;
			return Array.from(root.querySelectorAll('button, a')).map((el) => el.getAttribute('title'));
		});
		expect(order.indexOf('Limpar filtros')).toBeLessThan(order.indexOf('Configurar fontes de vagas'));
		await clear.click();
		await expect(page.locator('#inline-status')).toHaveValue('all');
	});

	test('paginação flutuante navega entre páginas', async ({ page }) => {
		const many = Array.from({ length: 12 }, (_, index) => ({
			id: index + 1,
			title: `Vaga ${index + 1}`,
			company: 'Acme Corp',
			url: `https://jobs.example.com/${index + 1}`,
			description: 'Descrição',
			source: 'himalayas',
			score_fit: 70,
			score_reason: '',
			status: 'new',
			created_at: '2026-11-01T12:00:00Z'
		}));
		await page.route('**/api/jobs', (route) =>
			route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ jobs: many }) })
		);
		await gotoReady(page, '/jobs');

		const pagination = page.getByRole('navigation', { name: 'Paginação e ações de vagas' });
		await expect(pagination).toBeVisible();
		await expect(pagination).toContainText('1 / 2');
		await expect(page.getByText('1..10 de 12 vagas')).toBeVisible();

		await pagination.getByRole('button', { name: 'Próxima página' }).click();
		await expect(pagination).toContainText('2 / 2');
		await expect(pagination.getByRole('button', { name: 'Próxima página' })).toBeDisabled();
	});

	test('Aprovar apenas mostra toast: sem redirect e sem recarregar a lista', async ({ page }) => {
		await gotoReady(page, '/jobs');
		let jobListRequests = 0;
		page.on('request', (request) => {
			if (request.url().endsWith('/api/jobs') && request.method() === 'GET') jobListRequests++;
		});
		const baseline = jobListRequests;

		await page.locator('tbody tr').first().locator('input[type="checkbox"]').check();
		await page.getByRole('button', { name: 'Aprovar vagas selecionadas' }).click();
		await expect(page.getByRole('status').filter({ hasText: /aprovada: 1 de 1/i })).toBeVisible();
		await expect(page).toHaveURL(/\/jobs$/);
		expect(jobListRequests).toBe(baseline);
	});

	test('/settings mostra apenas as abas (sem cards de seção)', async ({ page }) => {
		await gotoReady(page, '/settings');
		const nav = page.locator('nav[aria-label="Seções de configuração"]');
		await expect(nav).toBeVisible();
		await expect(nav.getByRole('link', { name: 'Fontes de Vagas' })).toBeVisible();
		await expect(page.locator('a[href="/settings/sources"]')).toHaveCount(1);
		await expect(page.locator('a[href="/settings/discovery"]')).toHaveCount(1);
		await expect(page.locator('a[href="/settings/notifications"]')).toHaveCount(1);
	});

	test('currículo oferece seletor de vagas pendentes com resumo', async ({ page }) => {
		await gotoReady(page, '/curriculum');
		const select = page.locator('#tailoring-job');
		await expect(select).toBeVisible();
		await expect(select.locator('option')).toHaveCount(3); // placeholder + 2 vagas "new"

		await select.selectOption('1');
		const summary = page.locator('.selected-job');
		await expect(summary.getByText('Senior Go Developer')).toBeVisible();
		await expect(summary.getByText('Acme Corp')).toBeVisible();
		await expect(page.locator('#jd_text')).toHaveValue(/Vaga remota com Go/);
	});

	test('ordenação única, filtro múltiplo e seleção em lote', async ({ page }) => {
		await gotoReady(page, '/jobs');

		// Ordenação padrão: score descendente.
		await expect(page.locator('tbody tr').first()).toContainText('Senior Go Developer');

		// Clicar em outra coluna troca a ordenação; clicar de novo inverte.
		const sortTitle = page.getByRole('button', { name: 'Ordenar por Vaga' });
		await sortTitle.click();
		await expect(page.locator('tbody tr').first()).toContainText('Backend Engineer');
		await sortTitle.click();
		await expect(page.locator('tbody tr').first()).toContainText('Senior Go Developer');

		// Filtro de fontes com múltipla escolha.
		const fontes = page.locator('.multi button.trigger:visible');
		await fontes.click();
		const popup = page.locator('[role="listbox"]:visible');
		await popup.locator('label').filter({ hasText: 'Remotive' }).locator('input').check();
		await fontes.click();
		await expect(page.locator('tbody tr')).toHaveCount(1);
		await expect(page.locator('tbody tr').first()).toContainText('Backend Engineer');

		// "Marcar todas" seleciona as vagas filtradas.
		await page.locator('button[title="Marcar todas as vagas"]:visible').click();
		await expect(page.locator('tbody input[type="checkbox"]:checked')).toHaveCount(1);
	});

	test('paginação de candidaturas em várias páginas', async ({ page }) => {
		const many = Array.from({ length: 10 }, (_, index) => ({
			id: index + 1,
			job_id: index + 1,
			status: 'pending',
			sent_at: null,
			updated_at: '2026-11-02T10:00:00Z',
			has_tailored: false,
			job: { title: `Vaga ${index + 1}`, company: 'Acme Corp', url: '', score_fit: 70 }
		}));
		await page.route('**/api/applications', (route) =>
			route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ applications: many }) })
		);
		await gotoReady(page, '/applications');

		const pagination = page.getByRole('navigation', { name: 'Paginação de candidaturas' });
		await expect(pagination).toBeVisible();
		await expect(page.getByText('1..8 de 10 candidaturas')).toBeVisible();
		await pagination.getByRole('button', { name: 'Próxima página' }).click();
		await expect(page.getByText('9..10 de 10 candidaturas')).toBeVisible();
	});

	test('smartphone navega por rolagem contínua, sem paginação', async ({ page }) => {
		await page.setViewportSize({ width: 375, height: 667 });
		await gotoReady(page, '/jobs');
		await expect(page.locator('footer.app-footer')).toHaveCount(0);
		await expect(page.getByRole('navigation', { name: /Paginação/ })).toHaveCount(0);
		await expect(page.locator('.card-list.scrollable').first()).toBeVisible();

		await gotoReady(page, '/applications');
		await expect(page.locator('footer.app-footer')).toHaveCount(0);
		await expect(page.getByRole('navigation', { name: /Paginação/ })).toHaveCount(0);
		await expect(page.locator('.apps-scroll')).toBeVisible();
	});

	test('rolagem contínua carrega mais vagas no tablet', async ({ page }) => {
		const many = Array.from({ length: 14 }, (_, index) => ({
			id: index + 1,
			title: `Vaga ${index + 1}`,
			company: 'Acme Corp',
			url: '',
			description: 'Descrição',
			source: 'himalayas',
			score_fit: 70,
			score_reason: '',
			status: 'new',
			created_at: '2026-11-01T12:00:00Z'
		}));
		await page.route('**/api/jobs', (route) =>
			route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ jobs: many }) })
		);
		await page.setViewportSize({ width: 768, height: 1024 });
		await gotoReady(page, '/jobs');
		await expect(page.locator('.record')).toHaveCount(8);
		await page.locator('.card-list').evaluate((el) => (el.scrollTop = el.scrollHeight));
		await expect(page.locator('.record')).toHaveCount(14);
	});

	test('configurações e atalhos vivem no menu do cabeçalho', async ({ page }) => {
		await gotoReady(page, '/');
		const sidebar = page.locator('aside[aria-label="Navegação lateral"]');
		await expect(sidebar.getByRole('link', { name: 'Notificações' })).toHaveCount(0);
		await expect(sidebar.getByRole('link', { name: 'Config' })).toHaveCount(0);

		const settings = page.locator('.head-actions button[aria-label="Configurações"]');
		await expect(settings).toBeVisible();
		await settings.click();
		const menu = page.getByRole('menu', { name: 'Configurações' });
		await expect(menu).toBeVisible();
		for (const label of [
			'Provedor IA (BYOK)',
			'Fontes de Vagas',
			'Busca & Filtros',
			'Notificações & Tarefas',
			'Aparência'
		]) {
			await expect(menu.getByRole('menuitem', { name: label })).toBeVisible();
		}
		await menu.getByRole('menuitem', { name: 'Atalhos de teclado' }).click();
		await expect(page.getByRole('dialog', { name: /atalhos de teclado/i })).toBeVisible();
	});

	test('menu de configurações navega entre as abas internas', async ({ page }) => {
		await gotoReady(page, '/jobs');
		await page.locator('.head-actions button[aria-label="Configurações"]').click();
		await page.getByRole('menuitem', { name: 'Aparência' }).click();
		await expect(page).toHaveURL(/\/settings\/appearance$/);
		await expect(page.getByRole('heading', { name: 'Aparência' })).toBeVisible();
	});

	test('mobile: bottom nav tem 4 destinos e configuração fica no header', async ({ page }) => {
		await page.setViewportSize({ width: 375, height: 667 });
		await gotoReady(page, '/');
		const bottom = page.locator('nav.bottom-nav');
		await expect(bottom.getByRole('link')).toHaveCount(4);
		await expect(bottom.getByRole('link', { name: 'Config' })).toHaveCount(0);
		await page.locator('.head-actions button[aria-label="Configurações"]').click();
		await expect(page.getByRole('menuitem', { name: 'Aparência' })).toBeVisible();
	});

	test('ações de notificação são ícones com tooltip', async ({ page }) => {
		await gotoReady(page, '/notifications');
		await expect(page.locator('button[title="Marcar todas como lidas"]:visible')).toBeVisible();
		await expect(page.locator('button[title="Limpar lidas"]:visible')).toBeVisible();
	});

	test('applications: barra de filtros e cards dentro das colunas', async ({ page }) => {
		await gotoReady(page, '/applications');
		await expect(page.locator('.filters-inline')).toBeVisible();

		const card = page.locator('.app-card').first();
		const column = page.locator('.column').first();
		const cardBox = await card.boundingBox();
		const columnBox = await column.boundingBox();
		expect(cardBox!.x).toBeGreaterThanOrEqual(columnBox!.x);
		expect(cardBox!.x + cardBox!.width).toBeLessThanOrEqual(columnBox!.x + columnBox!.width + 1);

		await page.locator('.multi button.trigger:visible').click();
		const popup = page.locator('[role="listbox"]:visible');
		await popup.locator('label').filter({ hasText: 'Entrevista' }).locator('input').check();
		await page.locator('.multi button.trigger:visible').click();
		await expect(page.getByText('Nenhuma candidatura com esses filtros')).toBeVisible();
	});

	test('cards de candidaturas são clicáveis e abrem detalhe', async ({ page }) => {
		await gotoReady(page, '/applications');
		await page.locator('.app-card').first().click();
		await expect(page.getByRole('dialog')).toBeVisible();
	});
});
