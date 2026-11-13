import adapter from '@sveltejs/adapter-static';
import { withPwa } from '@vite-pwa/sveltekit';
import { defineConfig } from 'vite';

export default defineConfig({
	// Dev server: HMR da SPA. O proxy /api evita CORS durante o desenvolvimento —
	// em produção SPA e API são servidas na mesma origem pelo binário Go.
	server: {
		// Acessível pelo celular na LAN (ex.: openjob.copilot:5173).
		host: true,
		// hostname mDNS no dev (Vite 6+ valida o Host por padrão).
		allowedHosts: true,
		proxy: {
			'/api': 'http://127.0.0.1:8080'
		}
	},
	// withPwa: sveltekit() + plugins do PWA (SvelteKit 3 / @vite-pwa/sveltekit v2).
	// 1º argumento: opções do sveltekit (adapter incluído); 2º: opções do PWA.
	plugins: withPwa(
		{
			compilerOptions: {
				// Force runes mode for the project, except for libraries. Can be removed in svelte 6.
				runes: ({ filename }) =>
					filename.split(/[/\\]/).includes('node_modules') ? undefined : true
			},

			// adapter-static em modo SPA puro: sem pré-render, fallback `index.html`
			// para deep links (ex.: /jobs recarregado no celular não dá 404).
			// Nota: no SvelteKit 3 o adapter é configurado aqui (o `svelte.config.js`
			// do plano foi escrito para o Kit 2). Ver docs/adr/0003-adapter-static-spa.md
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
				// App shell offline: pré-cacheia todos os assets do build.
				globPatterns: ['**/*.{js,css,html,svg,png,woff,woff2}'],
				// SPA pura: não há páginas pré-renderizadas, então o glob
				// padrão `prerendered/**` não casa com nada.
				globStrict: false,
				navigateFallback: '/'
			}
		}
	)
});
