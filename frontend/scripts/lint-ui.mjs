#!/usr/bin/env node
import { readFileSync, readdirSync, statSync } from 'node:fs';
import { join, relative } from 'node:path';
import { detectViolations } from './ui-lint-core.mjs';

const ROOTS = [
	new URL('../src/routes', import.meta.url).pathname,
	new URL('../src/lib/ui', import.meta.url).pathname
];

function collect(dir) {
	const files = [];
	for (const entry of readdirSync(dir)) {
		const path = join(dir, entry);
		if (statSync(path).isDirectory()) files.push(...collect(path));
		else if (entry.endsWith('.svelte')) files.push(path);
	}
	return files;
}

let total = 0;
for (const file of ROOTS.flatMap(collect)) {
	const source = readFileSync(file, 'utf8');
	const violations = detectViolations(source);
	const name = relative(process.cwd(), file);
	for (const violation of violations) {
		total++;
		console.error(`${name}:${violation.line}: ${violation.message}`);
	}
}

if (total > 0) {
	console.error(`\n[lint:ui] ${total} violação(ões). Use tokens de lib/styles/tokens.css ou o escape hatch documentado.`);
	process.exit(1);
}
console.log('[lint:ui] ✅ nenhum estilo ad-hoc em routes/ ou lib/ui/');
