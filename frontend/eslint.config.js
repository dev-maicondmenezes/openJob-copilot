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
	},
	{
		files: ['src/**/*.ts'],
		languageOptions: { parser: tsParser }
	},
	{
		// CP13: único ponto de import do paraglide é src/lib/i18n/.
		files: ['src/**/*.{ts,svelte}'],
		ignores: ['src/lib/i18n/**', 'src/paraglide/**'],
		rules: {
			'no-restricted-imports': [
				'error',
				{
					patterns: [
						{
							group: ['@inlang/paraglide-js', '**/paraglide/**', '../paraglide/**', './paraglide/**'],
							message: 'Use o helper único `src/lib/i18n/t.ts` (CP13 §4.2).'
						}
					]
				}
			]
		}
	}
];
