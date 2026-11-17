/**
 * Detector de estilo ad-hoc em arquivos .svelte de `src/routes/`.
 * Fonte única usada por `scripts/lint-ui.mjs` e pela regra ESLint `ui/no-ad-hoc-style`.
 *
 * Regras (CP12 / ADR 0013):
 *  - cores literais (hex/rgb/hsl/nomeadas) são proibidas; use tokens semânticos;
 *  - spacing literal (padding/margin/gap) e border-radius literal são proibidos;
 *  - valores 0, auto, %, unidades de viewport e `var(...)` são permitidos;
 *  - escape hatch: a marca ui-lint-ignore na linha ou ui-lint-disable no início
 *    do bloco <style> (documentado em docs/design/tokens.md).
 */

// Nomes de cores são verificados só após `:` para evitar falsos positivos
// como `white-space`, `color: var(--color-text-primary)` etc.
const NAMED_COLORS =
	/:\s*(white|black|red|green|blue|yellow|orange|purple|gray|grey|pink|cyan|magenta|brown|gold|silver|navy|teal|maroon|olive|lime|aqua|fuchsia)\b/i;
const COLOR_LITERAL = /#(?:[0-9a-f]{3,4}|[0-9a-f]{6}|[0-9a-f]{8})\b|\brgba?\(|\bhsla?\(/i;
const SPACING_PROPS =
	/\b(?:padding|padding-top|padding-right|padding-bottom|padding-left|margin|margin-top|margin-right|margin-bottom|margin-left|gap|row-gap|column-gap)\s*:\s*([^;]+);/gi;
const RADIUS_PROP = /\bborder-radius\s*:\s*([^;]+);/gi;
const RAW_LENGTH = /-?\d+(?:\.\d+)?(?:px|rem|em)\b/;

function allowedValue(value) {
	return value.includes('var(') || !RAW_LENGTH.test(value);
}

export function detectViolations(source) {
	const violations = [];
	const styleBlocks = /<style[^>]*>([\s\S]*?)<\/style>/g;
	let match;

	while ((match = styleBlocks.exec(source)) !== null) {
		const blockStart = match.index + match[0].indexOf(match[1]);
		const startLine = source.slice(0, blockStart).split('\n').length;
		const body = match[1];
		if (body.includes('ui-lint-disable')) continue;

		const lines = body.split('\n');
		let ignoreNext = false;
		for (let i = 0; i < lines.length; i++) {
			const line = lines[i];
			if (line.includes('ui-lint-ignore')) {
				ignoreNext = true;
				continue;
			}
			if (ignoreNext) {
				ignoreNext = false;
				continue;
			}
			const lineNumber = startLine + i;

			if (COLOR_LITERAL.test(line) || NAMED_COLORS.test(line)) {
				violations.push({
					line: lineNumber,
					message: 'Cor literal fora dos tokens. Use var(--color-*) ou um componente de lib/ui.'
				});
			}

			let prop;
			while ((prop = SPACING_PROPS.exec(line)) !== null) {
				if (!allowedValue(prop[1])) {
					violations.push({
						line: lineNumber,
						message: 'Espaçamento literal fora da escala. Use var(--space-*).'
					});
					break;
				}
			}
			SPACING_PROPS.lastIndex = 0;

			let radius;
			while ((radius = RADIUS_PROP.exec(line)) !== null) {
				if (!allowedValue(radius[1])) {
					violations.push({
						line: lineNumber,
						message: 'Raio literal fora dos tokens. Use var(--radius-*).'
					});
					break;
				}
			}
			RADIUS_PROP.lastIndex = 0;
		}
	}

	return violations;
}
