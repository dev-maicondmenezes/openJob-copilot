import { test, expect } from '@playwright/test';
import { mockApi, gotoReady } from '../fixtures/mock-api';

test('deep link /jobs/123 abre a página cheia no desktop', async ({ page }) => {
	await mockApi(page);
	await gotoReady(page, '/jobs/1');

	await expect(page.getByRole('heading', { name: 'Detalhe da vaga' })).toBeVisible();
	await expect(page.locator('nav[aria-label="Trilha de navegação"]')).toBeVisible();
	await expect(page.getByRole('dialog')).toHaveCount(0);
	await expect(page.getByRole('link', { name: /voltar para vagas/i })).toBeVisible();
});
