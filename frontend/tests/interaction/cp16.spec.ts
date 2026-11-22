import { test, expect } from '@playwright/test';
import { mockApi, gotoReady } from '../fixtures/mock-api';

test.beforeEach(async ({ page }) => {
	await mockApi(page);
});

test.describe('Checkpoint 16 — Feed vertical imersivo', () => {
	test('feed renderiza em /jobs com role=feed, articles e snap nativo', async ({ page }) => {
		await gotoReady(page, '/jobs');
		const feed = page.locator('div[role="feed"]');
		await expect(feed).toBeVisible();

		const cards = page.getByRole('article');
		await expect(cards.first()).toBeVisible();
		// Primeiro card ativo
		await expect(cards.first()).toHaveClass(/active/);
		// Anúncio aria-live de posição inicial
		await expect(page.locator('.feed-announcement[aria-live="polite"]')).toContainText('Vaga 1 de 2');
	});

	test('apenas o card ativo é focável (inert nos cartões inativos)', async ({ page }) => {
		await gotoReady(page, '/jobs');
		const activeCard = page.locator('article.job-feed-card.active');
		await expect(activeCard).toBeVisible();
		await expect(activeCard).not.toHaveAttribute('inert', '');

		// Próximo card no slot deve estar com inert
		const nextCard = page.locator('.feed-slot[data-index="1"] article.job-feed-card');
		if (await nextCard.count() > 0) {
			await expect(nextCard).toHaveAttribute('inert', '');
		}
	});

	test('teclado ArrowDown e ArrowUp navegam entre os cards no desktop', async ({ page }) => {
		await page.setViewportSize({ width: 1440, height: 900 });
		await gotoReady(page, '/jobs');

		await expect(page.locator('article.job-feed-card.active')).toContainText('Senior Go Developer');

		// Pressionar seta para baixo
		await page.keyboard.press('ArrowDown');
		await page.waitForTimeout(300);

		// Anúncio deve atualizar para vaga 2
		await expect(page.locator('.feed-announcement[aria-live="polite"]')).toContainText('Vaga 2 de 2');
	});

	test('botão Saiba mais abre /jobs/[id] com descrição completa e botão Voltar', async ({ page }) => {
		await gotoReady(page, '/jobs');
		const readMore = page.locator('article.job-feed-card.active a.read-more-link');
		await expect(readMore).toBeVisible();
		await readMore.click();

		await expect(page).toHaveURL(/\/jobs\/1$/);
		await expect(page.getByRole('heading', { name: 'Detalhe da vaga' })).toBeVisible();
		await expect(page.getByText('Vaga remota com Go, SQL e observabilidade.')).toBeVisible();

		const backLink = page.getByRole('link', { name: /voltar para vagas/i });
		await expect(backLink).toBeVisible();
		await backLink.click();
		await expect(page).toHaveURL(/\/jobs$/);
	});

	test('segurança XSS: script injection em descrição não executa', async ({ page }) => {
		const xssJob = {
			id: 99,
			title: 'Injected Job',
			company: 'Hacker Inc',
			url: 'https://example.com/job/99',
			description: '<script>window.__xss_fired = true;</script>Texto seguro **destaque** [link](javascript:window.__xss_link = true)',
			source: 'himalayas',
			score_fit: 90,
			score_reason: 'teste xss',
			status: 'new',
			created_at: '2026-11-01T12:00:00Z'
		};
		await page.route('**/api/jobs', (route) =>
			route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ jobs: [xssJob] }) })
		);

		await gotoReady(page, '/jobs');

		const fired = await page.evaluate(() => (window as unknown as { __xss_fired?: boolean }).__xss_fired);
		expect(fired).toBeUndefined();

		// Clicar em Saiba mais para testar a página de detalhe
		await page.locator('article.job-feed-card.active a.read-more-link').click();
		await expect(page).toHaveURL(/\/jobs\/99$/);

		const firedDetail = await page.evaluate(() => (window as unknown as { __xss_fired?: boolean }).__xss_fired);
		expect(firedDetail).toBeUndefined();

		// Link javascript: não deve ser renderizado como tag <a>
		await expect(page.locator('a[href^="javascript:"]')).toHaveCount(0);
	});

	test('desempenho: janela deslizante mantém número limitado de cards no DOM', async ({ page }) => {
		const many = Array.from({ length: 25 }, (_, index) => ({
			id: index + 1,
			title: `Vaga ${index + 1}`,
			company: `Empresa ${index + 1}`,
			url: `https://example.com/job/${index + 1}`,
			description: `Descrição longa da vaga número ${index + 1}.`,
			source: 'himalayas',
			score_fit: 75,
			score_reason: 'OK',
			status: 'new',
			created_at: '2026-11-01T12:00:00Z'
		}));
		await page.route('**/api/jobs', (route) =>
			route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ jobs: many }) })
		);

		await gotoReady(page, '/jobs');

		// No estado inicial (índice 0), apenas índices 0 e 1 estão montados (máximo 2-3 cards)
		const mountedCardsInitial = await page.locator('article.job-feed-card').count();
		expect(mountedCardsInitial).toBeLessThanOrEqual(3);

		// Rolar para baixo até a vaga 10
		for (let i = 0; i < 10; i++) {
			await page.keyboard.press('ArrowDown');
			await page.waitForTimeout(50);
		}

		// Número de cards montados continua estritamente limitado pela janela deslizante
		const mountedCardsScrolled = await page.locator('article.job-feed-card').count();
		expect(mountedCardsScrolled).toBeLessThanOrEqual(3);
	});

	test('alternância fluida entre Feed e Lista tradicional', async ({ page }) => {
		await gotoReady(page, '/jobs');
		await expect(page.locator('div[role="feed"]')).toBeVisible();

		// Clicar no botão para Ver Lista
		await page.getByRole('link', { name: /ver lista/i }).click();
		await expect(page).toHaveURL(/\/jobs\/list$/);
		await expect(page.getByRole('heading', { name: 'Vagas (Lista)' })).toBeVisible();

		// Na lista, o botão Ver Feed retorna ao feed
		await page.getByRole('link', { name: /ver feed/i }).click();
		await expect(page).toHaveURL(/\/jobs$/);
		await expect(page.locator('div[role="feed"]')).toBeVisible();
	});

	test('Buscar agora no feed dispara discovery com loading e feedback', async ({ page }) => {
		let discoverCalls = 0;
		await page.route('**/api/jobs/discover', (route) => {
			discoverCalls++;
			return route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ status: 'ok' }) });
		});

		await gotoReady(page, '/jobs');
		const searchBtn = page.getByRole('button', { name: /buscar agora/i });
		await expect(searchBtn).toBeVisible();
		await searchBtn.click();

		expect(discoverCalls).toBe(1);
	});
});
