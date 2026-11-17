import AxeBuilder from '@axe-core/playwright';
import { test, expect } from '@playwright/test';
import { mockApi, ROUTES, gotoReady } from '../fixtures/mock-api';

/** axe-core por rota × viewport canônico (CP12 §7). */
test.describe('acessibilidade (axe)', () => {
	for (const route of ROUTES) {
		test(`sem violações sérias em ${route}`, async ({ page }) => {
			await mockApi(page);
			await gotoReady(page, route);
			await expect(page.locator('main')).toBeVisible();

			const results = await new AxeBuilder({ page })
				.withTags(['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa'])
				.analyze();

			const serious = results.violations.filter(
				(violation) => violation.impact === 'serious' || violation.impact === 'critical'
			);
			expect(
				serious.map((violation) => `${violation.id}: ${violation.help}`),
				'violações sérias/críticas de acessibilidade'
			).toEqual([]);
		});
	}
});
