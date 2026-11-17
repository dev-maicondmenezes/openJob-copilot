import adapter from '@sveltejs/adapter-static';
import { paraglideVitePlugin } from '@inlang/paraglide-js';
import { withPwa } from '@vite-pwa/sveltekit';
import { defineConfig } from 'vite';

export default defineConfig({
	// Página de referência /dev/ui: só existe quando OPENJOB_DEV_UI=1 no build/dev.
	// (globalThis.process evita depender de @types/node neste projeto.)
	define: {
		'import.meta.env.OPENJOB_DEV_UI': JSON.stringify(
			String(
				(globalThis as any).process?.env?.OPENJOB_DEV_UI ??
					(globalThis as any).process?.env?.VITE_OPENJOB_DEV_UI ??
					''
			)
		)
	},
	// Dev server: HMR da SPA. O proxy /api evita CORS durante o desenvolvimento —
	// em produção SPA e API são servidas na mesma origem pelo binário Go.
	server: {
		host: true,
		allowedHosts: true,
		proxy: {
			'/api': 'http://127.0.0.1:8080'
		}
	},
	plugins: [
		// Paraglide compila as mensagens em funções tipadas por chave (ADR 0017).
		// `localStorage` usa a mesma chave do script anti-FOUC em app.html.
		paraglideVitePlugin({
			project: './project.inlang',
			outdir: './src/paraglide',
			outputStructure: 'message-modules',
			strategy: ['localStorage', 'baseLocale'],
			localStorageKey: 'openjob.locale'
		}),
		// withPwa: sveltekit() + plugins do PWA (SvelteKit 3 / @vite-pwa/sveltekit v2).
		...withPwa(
			{
				compilerOptions: {
					// Force runes mode for the project, except for libraries.
					runes: ({ filename }) =>
						filename.split(/[/\\]/).includes('node_modules') ? undefined : true
				},
				// adapter-static em modo SPA puro: sem pré-render, fallback `index.html`.
				// Ver docs/adr/0003-adapter-static-spa.md.
				adapter: adapter({
					pages: 'build',
					assets: 'build',
					fallback: 'index.html',
					precompress: false,
					strict: false
				})
			},
			{
				registerType: 'autoUpdate',
				includeAssets: ['favicon.svg', 'robots.txt'],
				manifest: {
					name: 'openJob',
					short_name: 'openJob',
					description: 'Assistente local-first para a busca de emprego',
					lang: 'pt-BR',
					start_url: '/',
					scope: '/',
					display: 'standalone',
					background_color: '#0f172a',
					theme_color: '#0f172a',
					icons: [
						{ src: 'pwa-192.png', sizes: '192x192', type: 'image/png' },
						{ src: 'pwa-512.png', sizes: '512x512', type: 'image/png' },
						{
							src: 'pwa-maskable-512.png',
							sizes: '512x512',
							type: 'image/png',
							purpose: 'maskable'
						}
					]
				},
				generateSW: {
					globPatterns: ['**/*.{js,css,html,svg,png,woff,woff2}'],
					globStrict: false,
					navigateFallback: '/'
				}
			}
		)
	]
});
