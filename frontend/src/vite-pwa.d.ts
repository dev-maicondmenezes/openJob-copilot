// Módulo virtual do @vite-pwa/sveltekit (registro do service worker).
// Assinatura espelhada de @unplugin-pwa/core/vanillajs.d.ts — declarado aqui
// porque o arquivo precisa ser ambiental (sem export/import no top level).
declare module 'virtual:pwa-register' {
	import type { RegisterSWOptions } from '@unplugin-pwa/core/types';

	export type { RegisterSWOptions };

	export function registerSW(options?: RegisterSWOptions): () => Promise<void>;
}
