import { test, expect } from '@playwright/test';
import { mockApi, ROUTES, gotoReady } from '../fixtures/mock-api';

/**
 * Snapshots por rota × viewport canônico (375×667, 768×1024, 1440×900).
 * Baseline em tests/visual/**. Mudanças exigem revisão manual (CP12 §8).
 */
test.describe('snapshots visuais', () => {
	for (const route of ROUTES) {
		test(`rota ${route}`, async ({ page }) => {
			await page.addInitScript(() => document.body.setAttribute('data-test-animations', 'off'));
			await page.emulateMedia({ reducedMotion: 'reduce' });
			await mockApi(page);
			await gotoReady(page, route);
			await expect(page.locator('main')).toBeVisible();
			await page.waitForLoadState('networkidle');
			const name = route === '/' ? 'home' : route.replace(/^\//, '').replace(/\//g, '-');
			await expect(page).toHaveScreenshot(`${name}.png`, {
				animations: 'disabled',
				fullPage: true
			});
		});
	}
});
