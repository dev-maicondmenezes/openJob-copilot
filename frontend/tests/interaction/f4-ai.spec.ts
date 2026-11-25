import { test, expect } from '@playwright/test';
import { mockApi, gotoReady } from '../fixtures/mock-api';

test.beforeEach(async ({ page }) => {
	await mockApi(page);
});

test.describe('Fase 4 — UX de IA', () => {
	test('preset controla o seletor de modelos e o card de ajuda do provedor', async ({ page }) => {
		await gotoReady(page, '/settings');
		await page.getByRole('button', { name: 'Adicionar provedor' }).click();

		const presetSelect = page.getByLabel('Provedor', { exact: true });
		const modelField = page.getByLabel('Modelo', { exact: true });

		// Preset homologado: modelos vêm do catálogo em um select.
		await expect(presetSelect).toHaveValue('gemini');
		await expect(modelField).toHaveJSProperty('tagName', 'SELECT');
		await expect(modelField.locator('option')).toHaveCount(2);
		await expect(modelField).toHaveValue('gemini-2.5-flash');

		// Card de ajuda: documentação + página oficial de criação da chave.
		await expect(page.getByText('Ajuda do provedor')).toBeVisible();
		await expect(page.getByRole('link', { name: 'Documentação' })).toHaveAttribute(
			'href',
			'https://example.com/gemini/docs'
		);
		await expect(page.getByRole('link', { name: 'Criar API key' })).toHaveAttribute(
			'href',
			'https://aistudio.google.com/apikey'
		);

		// Trocar de preset atualiza base URL, modelos e links oficiais.
		await presetSelect.selectOption('openai');
		await expect(modelField).toHaveValue('gpt-4o-mini');
		await expect(modelField.locator('option')).toHaveCount(2);
		await expect(page.getByRole('link', { name: 'Criar API key' })).toHaveAttribute(
			'href',
			'https://example.com/openai/keys'
		);

		// Custom é internacionalizado e volta a ser texto simples, sem card.
		await presetSelect.selectOption('custom');
		await expect(presetSelect.locator('option[value="custom"]')).toHaveText('Personalizado');
		await expect(page.getByLabel('Modelo', { exact: true })).toHaveJSProperty('tagName', 'INPUT');
		await expect(page.getByLabel('Base URL', { exact: true })).toBeVisible();
		await expect(page.getByText('Ajuda do provedor')).toHaveCount(0);
	});

	test('cadastra outro provedor e troca o perfil ativo', async ({ page }) => {
		const profiles = [
			{
				id: 'p_groq',
				name: 'Groq',
				preset_id: 'groq',
				base_url: 'https://api.groq.com/openai/v1',
				model: 'llama-3.3-70b-versatile',
				active: true,
				local: false
			},
			{
				id: 'p_openai',
				name: 'OpenAI (trabalho)',
				preset_id: 'openai',
				base_url: 'https://api.openai.com/v1',
				model: 'gpt-4o-mini',
				active: false,
				local: false
			}
		];
		let activateBody: Record<string, unknown> | null = null;
		let createBody: Record<string, unknown> | null = null;

		await page.route('**/api/settings/llm/profiles', async (route) => {
			const method = route.request().method();
			if (method === 'GET') {
				await route.fulfill({
					status: 200,
					contentType: 'application/json',
					body: JSON.stringify({ profiles })
				});
				return;
			}
			if (method === 'POST') {
				createBody = route.request().postDataJSON();
				profiles.push({
					id: 'p_new',
					name: 'Gemini pessoal',
					preset_id: 'gemini',
					base_url: 'https://generativelanguage.googleapis.com/v1beta/openai',
					model: 'gemini-2.5-flash',
					active: false,
					local: false
				});
				await route.fulfill({
					status: 201,
					contentType: 'application/json',
					body: JSON.stringify({ profile: profiles[profiles.length - 1] })
				});
				return;
			}
			await route.fallback();
		});
		await page.route('**/api/settings/llm/activate', async (route) => {
			activateBody = route.request().postDataJSON();
			const id = String(activateBody?.id ?? '');
			for (const profile of profiles) profile.active = profile.id === id;
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({ profile: profiles.find((p) => p.id === id) })
			});
		});

		await gotoReady(page, '/settings');
		const groqRow = page.getByRole('listitem').filter({ hasText: 'Groq' });
		await expect(groqRow).toContainText('Ativa');

		await page.getByRole('button', { name: 'Adicionar provedor' }).click();
		await page.getByLabel('Nome do perfil').fill('Gemini pessoal');
		await page.getByLabel('API key').fill('sk-test-123');
		await page.getByRole('button', { name: /^Salvar$/ }).click();

		await expect.poll(() => createBody).not.toBeNull();
		expect(createBody).toMatchObject({
			name: 'Gemini pessoal',
			preset_id: 'gemini',
			model: 'gemini-2.5-flash'
		});
		await expect(page.getByText('Provedor adicionado.')).toBeVisible();
		await expect(page.getByText('Gemini pessoal')).toBeVisible();

		const openaiRow = page.getByRole('listitem').filter({ hasText: 'OpenAI (trabalho)' });
		await openaiRow.getByRole('button', { name: 'Ativar' }).click();

		await expect.poll(() => activateBody).not.toBeNull();
		expect(activateBody).toMatchObject({ id: 'p_openai' });
		await expect(openaiRow).toContainText('Ativa');
		await expect(page.getByText('Provedor ativo atualizado.')).toBeVisible();
	});
});
