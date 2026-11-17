import { test, expect } from '@playwright/test';
import { mockApi, gotoReady } from '../fixtures/mock-api';

test.beforeEach(async ({ page }) => {
	await mockApi(page);
});

test.describe('CP13 — dark mode + i18n', () => {
	test('anti-FOUC: data-theme aplicado antes do DOMContentLoaded', async ({ page }) => {
		await page.addInitScript(() => {
			localStorage.setItem('openjob.theme', 'dark');
			window.addEventListener('DOMContentLoaded', () => {
				(window as any).__themeAtDomReady = document.documentElement.dataset.theme;
			});
		});
		await page.goto('/jobs');
		expect(await page.evaluate(() => (window as any).__themeAtDomReady)).toBe('dark');
		await expect(page.locator('html')).toHaveAttribute('data-theme', 'dark');
	});

	test('modo system acompanha prefers-color-scheme sem reload', async ({ page }) => {
		await gotoReady(page, '/settings/appearance');
		await page.getByRole('button', { name: /sistema/i }).click();
		await expect(page.locator('html')).toHaveAttribute('data-theme', 'light');
		await page.emulateMedia({ colorScheme: 'dark' });
		await expect(page.locator('html')).toHaveAttribute('data-theme', 'dark');
		await page.emulateMedia({ colorScheme: 'light' });
		await expect(page.locator('html')).toHaveAttribute('data-theme', 'light');
	});

	test('trocar locale re-renderiza UI e histórico de notificações sem reload', async ({ page }) => {
		await gotoReady(page, '/notifications');
		// Notificação legada (title/body resolvidos) continua legível.
		await expect(page.getByText('Vaga pontuada')).toBeVisible();
		// Notificação nova (i18n_key) usa o locale atual.
		await expect(page.getByText(/mudou de pending para interview/)).toBeVisible();

		await gotoReady(page, '/settings/appearance');
		await page.locator('#appearance-locale').selectOption('es-419');
		await expect(page.locator('html')).toHaveAttribute('lang', 'es-419');
		await expect(page.getByRole('heading', { name: 'Apariencia' })).toBeVisible();

		// Navegação SPA (sem reload): o script de teste não deve re-fixar o locale.
		await page.getByRole('button', { name: /notificaciones|notificações/i }).first().click();
		await page.locator('a[href="/notifications"]').last().click();
		await expect(page.getByText('Vaga pontuada')).toBeVisible(); // legado intacto
		await expect(page.getByText(/cambió de pending a interview/)).toBeVisible();
	});

	test('fallback chain: es-MX → es-419, pt-PT → pt-BR, fr-FR → en-US', async ({ browser }) => {
		for (const [locale, expected] of [
			['es-MX', 'es-419'],
			['pt-PT', 'pt-BR'],
			['fr-FR', 'en-US']
		]) {
			const context = await browser.newContext({ locale });
			const page = await context.newPage();
			await mockApi(page);
			await page.addInitScript(() => localStorage.removeItem('openjob.locale'));
			await page.goto('/');
			await expect(page.locator('html')).toHaveAttribute('lang', expected);
			await context.close();
		}
	});

	test('deep link mantém o locale aplicado', async ({ page }) => {
		await page.addInitScript(() => localStorage.setItem('openjob.locale', 'en-US'));
		await gotoReady(page, '/jobs/1');
		await expect(page.locator('html')).toHaveAttribute('lang', 'en-US');
		await expect(page.getByRole('heading', { name: 'Job details' })).toBeVisible();
	});
});
