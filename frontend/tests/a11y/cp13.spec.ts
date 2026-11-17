import AxeBuilder from '@axe-core/playwright';
import { test, expect } from '@playwright/test';
import { mockApi, gotoReady } from '../fixtures/mock-api';

/** CP13 §4.8: axe nos dois temas, nas rotas core (desktop + phone). */
const CORE_ROUTES = ['/jobs', '/applications', '/settings', '/notifications'] as const;

for (const route of CORE_ROUTES) {
	for (const theme of ['light', 'dark'] as const) {
		test(`${route} tema=${theme} sem violações sérias`, async ({ page }) => {
			await mockApi(page);
			await page.addInitScript((theme) => localStorage.setItem('openjob.theme', theme), theme);
			await gotoReady(page, route);
			await expect(page.locator('main')).toBeVisible();
			await expect(page.locator('html')).toHaveAttribute('data-theme', theme);

			const results = await new AxeBuilder({ page })
				.withTags(['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa'])
				.analyze();
			const serious = results.violations.filter(
				(violation) => violation.impact === 'serious' || violation.impact === 'critical'
			);
			expect(serious.map((violation) => `${violation.id}: ${violation.help}`)).toEqual([]);
		});
	}
}
