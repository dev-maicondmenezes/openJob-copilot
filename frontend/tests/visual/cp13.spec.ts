import { test, expect } from '@playwright/test';
import { mockApi, gotoReady } from '../fixtures/mock-api';

/**
 * CP13 §4.8: cobertura visual por locale × tema sem explodir a suíte.
 * Desktop: 4 rotas core × 3 locales × 2 temas.
 * Mobile: 1 smoke por locale × 2 temas na rota /jobs.
 */
const LOCALES = ['pt-BR', 'en-US', 'es-419'] as const;
const THEMES = ['light', 'dark'] as const;
const CORE_ROUTES = ['/jobs', '/applications', '/settings', '/notifications'] as const;

for (const route of CORE_ROUTES) {
	for (const locale of LOCALES) {
		for (const theme of THEMES) {
			test(`desktop ${route} ${locale} ${theme}`, async ({ page }) => {
				const width = page.viewportSize()?.width ?? 0;
				test.skip(width !== 1440, 'cobertura desktop apenas');
				await mockApi(page);
				await page.addInitScript(
					({ locale, theme }) => {
						localStorage.setItem('openjob.locale', locale);
						localStorage.setItem('openjob.theme', theme);
					},
					{ locale, theme }
				);
				await page.emulateMedia({ reducedMotion: 'reduce' });
				await gotoReady(page, route);
				await expect(page.locator('main')).toBeVisible();
				await page.waitForLoadState('networkidle');
				const name = route.slice(1).replace(/\//g, '-');
				await expect(page).toHaveScreenshot(`cp13-${name}-${locale}-${theme}.png`, {
					animations: 'disabled',
					fullPage: true
				});
			});
		}
	}
}

for (const locale of LOCALES) {
	for (const theme of THEMES) {
		test(`mobile smoke /jobs ${locale} ${theme}`, async ({ page }) => {
			const width = page.viewportSize()?.width ?? 0;
			test.skip(width !== 375, 'smoke mobile apenas');
			await mockApi(page);
			await page.addInitScript(
				({ locale, theme }) => {
					localStorage.setItem('openjob.locale', locale);
					localStorage.setItem('openjob.theme', theme);
				},
				{ locale, theme }
			);
			await page.emulateMedia({ reducedMotion: 'reduce' });
			await gotoReady(page, '/jobs');
			await expect(page).toHaveScreenshot(`cp13-mobile-jobs-${locale}-${theme}.png`, {
				animations: 'disabled',
				fullPage: true
			});
		});
	}
}
