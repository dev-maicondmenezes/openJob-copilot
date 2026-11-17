import { detectViolations } from './ui-lint-core.mjs';

/** Regra ESLint que reusa o detector do lint:ui dentro de arquivos Svelte. */
export default {
	meta: { name: 'openjob-ui', version: '1.0.0' },
	rules: {
		'no-ad-hoc-style': {
			meta: {
				type: 'problem',
				docs: {
					description: 'Proíbe cor/spacing/raio literais em <style> de src/routes/.'
				},
				schema: []
			},
			create(context) {
				return {
					Program(node) {
						const source = context.sourceCode?.text ?? context.getSourceCode().text;
						for (const violation of detectViolations(source)) {
							context.report({
								node,
								loc: { line: violation.line, column: 1 },
								message: violation.message
							});
						}
					}
				};
			}
		}
	}
};
