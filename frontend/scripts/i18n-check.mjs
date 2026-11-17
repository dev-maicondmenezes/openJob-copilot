/**
 * Validação de completude dos catálogos (CP13 §4.2).
 *
 * `en-US` é o canônico estrutural: nenhum locale pode ter chave faltando
 * ou extra em relação a ele. Também detecta placeholders divergentes
 * (`{name}`), que causariam runtime errors com mensagens ICU.
 */
import { readFileSync, readdirSync } from 'node:fs';
import { join } from 'node:path';

const dir = new URL('../messages/', import.meta.url).pathname;
const locales = readdirSync(dir)
	.filter((f) => f.endsWith('.json'))
	.map((f) => f.replace(/\.json$/, ''))
	.sort();

if (!locales.includes('en-US')) {
	console.error('i18n:check: en-US.json não encontrado (canônico estrutural).');
	process.exit(1);
}

const catalogs = Object.fromEntries(
	locales.map((locale) => [locale, JSON.parse(readFileSync(join(dir, `${locale}.json`), 'utf8'))])
);

const keys = (obj) => Object.keys(obj).filter((k) => !k.startsWith('$')).sort();
const placeholders = (text) => [...String(text).matchAll(/\{([a-zA-Z_][\w]*)/g)].map((m) => m[1]).sort().join(',');

const canonical = keys(catalogs['en-US']);
let errors = 0;

for (const locale of locales) {
	const found = keys(catalogs[locale]);
	const missing = canonical.filter((k) => !found.includes(k));
	const extra = found.filter((k) => !canonical.includes(k));
	if (missing.length) {
		errors++;
		console.error(`i18n:check: ${locale} — ${missing.length} chave(s) faltando: ${missing.slice(0, 10).join(', ')}`);
	}
	if (extra.length) {
		errors++;
		console.error(`i18n:check: ${locale} — ${extra.length} chave(s) extra: ${extra.slice(0, 10).join(', ')}`);
	}
	for (const key of canonical) {
		if (!(key in catalogs[locale])) continue;
		const expected = placeholders(catalogs['en-US'][key]);
		const actual = placeholders(catalogs[locale][key]);
		if (expected !== actual) {
			errors++;
			console.error(`i18n:check: ${locale}.${key} — placeholders divergentes (${actual} !== ${expected})`);
		}
	}
}

if (errors) {
	console.error(`i18n:check: FALHOU (${errors} problema(s)).`);
	process.exit(1);
}
console.log(`i18n:check: OK (${canonical.length} chaves × ${locales.length} locales).`);
