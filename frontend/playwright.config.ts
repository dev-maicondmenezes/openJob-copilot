import { defineConfig, devices } from '@playwright/test';

/**
 * Testes do CP12: snapshots por rota × viewport canônico, axe por viewport,
 * interações (atalhos, troca de viewport, deep link). Ver docs/design/tokens.md.
 */
const PORT = 5173;
const baseURL = `http://localhost:${PORT}`;

export default defineConfig({
	testDir: './tests',
	fullyParallel: true,
	forbidOnly: Boolean(process.env.CI),
	retries: process.env.CI ? 2 : 0,
	workers: process.env.CI ? 2 : undefined,
	reporter: process.env.CI
		? [['github'], ['html', { open: 'never' }]]
		: [['list']],
	expect: { timeout: 10_000 },
	use: {
		baseURL,
		trace: 'retain-on-failure',
		screenshot: 'only-on-failure',
		launchOptions: {
			// Snapshots visuais precisam ser idênticos entre a máquina local e o
			// CI. Sem isso, hinting/LCD variam por distro e quebram os baselines.
			args: ['--disable-lcd-text', '--font-render-hinting=none']
		},
		...devices['Desktop Chrome']
	},
	webServer: {
		command: `npm run dev -- --port ${PORT} --strictPort`,
		url: baseURL,
		reuseExistingServer: !process.env.CI,
		timeout: 120_000
	},
	projects: [
		{
			name: 'visual-phone',
			testMatch: /tests\/visual\/.*\.spec\.ts/,
			use: { viewport: { width: 375, height: 667 }, deviceScaleFactor: 1 }
		},
		{
			name: 'visual-tablet',
			testMatch: /tests\/visual\/.*\.spec\.ts/,
			use: { viewport: { width: 768, height: 1024 }, deviceScaleFactor: 1 }
		},
		{
			name: 'visual-desktop',
			testMatch: /tests\/visual\/.*\.spec\.ts/,
			use: { viewport: { width: 1440, height: 900 }, deviceScaleFactor: 1 }
		},
		{
			name: 'a11y-phone',
			testMatch: /tests\/a11y\/.*\.spec\.ts/,
			use: { viewport: { width: 375, height: 667 } }
		},
		{
			name: 'a11y-tablet',
			testMatch: /tests\/a11y\/.*\.spec\.ts/,
			use: { viewport: { width: 768, height: 1024 } }
		},
		{
			name: 'a11y-desktop',
			testMatch: /tests\/a11y\/.*\.spec\.ts/,
			use: { viewport: { width: 1440, height: 900 } }
		},
		{
			name: 'e2e',
			testMatch: /tests\/interaction\/.*\.spec\.ts/,
			use: { viewport: { width: 1440, height: 900 } }
		}
	]
});
