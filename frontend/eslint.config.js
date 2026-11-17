import svelte from 'eslint-plugin-svelte';
import tsParser from '@typescript-eslint/parser';
import uiPlugin from './scripts/eslint-ui-plugin.mjs';

/** Configuração mínima: parser Svelte + TypeScript e a regra de UI do CP12. */
export default [
	{ ignores: ['build/**', '.svelte-kit/**', 'node_modules/**', '../backend/web/**'] },
	...svelte.configs['flat/base'],
	{
		files: ['src/**/*.svelte'],
		languageOptions: {
			parserOptions: {
				parser: tsParser,
				extraFileExtensions: ['.svelte']
			}
		}
	},
	{
		files: ['src/routes/**/*.svelte', 'src/lib/ui/**/*.svelte'],
		plugins: { ui: uiPlugin },
		rules: {
			'ui/no-ad-hoc-style': 'error'
		}
	}
];
