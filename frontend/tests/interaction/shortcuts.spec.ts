import { test, expect } from '@playwright/test';
import { mockApi, gotoReady } from '../fixtures/mock-api';

test.describe('atalhos de teclado (desktop)', () => {
	test.beforeEach(async ({ page }) => {
		await mockApi(page);
		await gotoReady(page, '/');
	});

	test('g j / g a / g s navegam', async ({ page }) => {
		await page.keyboard.press('g');
		await page.keyboard.press('j');
		await expect(page).toHaveURL(/\/jobs$/);

		await page.keyboard.press('g');
		await page.keyboard.press('a');
		await expect(page).toHaveURL(/\/applications$/);

		await page.keyboard.press('g');
		await page.keyboard.press('s');
		await expect(page).toHaveURL(/\/settings$/);
	});

	test('? abre a ajuda e Escape fecha', async ({ page }) => {
		await page.keyboard.press('?');
		await expect(page.getByRole('dialog', { name: /atalhos de teclado/i })).toBeVisible();
		await page.keyboard.press('Escape');
		await expect(page.getByRole('dialog', { name: /atalhos de teclado/i })).toHaveCount(0);
	});

	test('/ foca a busca', async ({ page }) => {
		await page.keyboard.press('/');
		await expect(page.locator('input[data-global-search]')).toBeFocused();
	});

	test('atalhos não disparam em campo de texto', async ({ page }) => {
		await gotoReady(page, '/jobs');
		await page.getByRole('button', { name: /nova vaga/i }).first().click();
		const input = page.locator('#title');
		await input.click();
		await input.pressSequentially('g j');
		await expect(input).toHaveValue('g j');
		await expect(page).toHaveURL(/\/jobs$/);
	});

	test('[ alterna o rail da sidebar e persiste', async ({ page }) => {
		await page.keyboard.press('[');
		await expect(page.locator('aside[aria-label="Navegação lateral"]')).toHaveClass(/collapsed/);
		await page.reload();
		await expect(page.locator('aside[aria-label="Navegação lateral"]')).toHaveClass(/collapsed/);
	});
});
