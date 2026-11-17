import { test, expect } from '@playwright/test';
import { mockApi, gotoReady } from '../fixtures/mock-api';

/** Redimensionamento 375 → 768 → 1024 → 1440 sem perder estado de formulário. */
test('estado de formulário sobrevive à troca de viewport', async ({ page }) => {
	await mockApi(page);
	await page.setViewportSize({ width: 375, height: 667 });
	await gotoReady(page, '/settings/discovery');

	const input = page.locator('#new_keyword');
	await input.fill('Golang');
	await expect(input).toHaveValue('Golang');

	for (const width of [768, 1024, 1440, 375]) {
		await page.setViewportSize({ width, height: 900 });
		await expect(input).toHaveValue('Golang');
	}
});

test('navegação troca de bottom nav para sidebar sem duplicar elementos', async ({ page }) => {
	await mockApi(page);
	await page.setViewportSize({ width: 375, height: 667 });
	await gotoReady(page, '/jobs');
	await expect(page.locator('nav[aria-label="Navegação principal"]')).toBeVisible();

	await page.setViewportSize({ width: 1440, height: 900 });
	await expect(page.locator('aside[aria-label="Navegação lateral"]')).toBeVisible();
	await expect(page.locator('nav.bottom-nav')).toBeHidden();
	await expect(page.locator('a[aria-current="page"]')).toHaveCount(1);
});
