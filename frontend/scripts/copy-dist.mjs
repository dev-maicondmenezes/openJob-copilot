// Copia o output do build SvelteKit (build/) para backend/web/dist/,
// alvo do //go:embed do binário Go (spec §5).
// Preserva o .gitkeep para o embed e para o versionamento.
// Uso: npm run build (chamado automaticamente após o `vite build`).
import fs from 'node:fs';
import path from 'node:path';

const src = path.resolve(process.cwd(), 'build');
const dest = path.resolve(process.cwd(), '..', 'backend', 'web', 'dist');

if (!fs.existsSync(src)) {
	console.error('[copy-dist] build/ não encontrado — rode `npm run build` (não `vite build` direto).');
	process.exit(1);
}

// Limpa o conteúdo anterior, preservando .gitkeep.
fs.mkdirSync(dest, { recursive: true });
for (const entry of fs.readdirSync(dest)) {
	if (entry === '.gitkeep') continue;
	fs.rmSync(path.join(dest, entry), { recursive: true, force: true });
}

fs.cpSync(src, dest, { recursive: true });

const files = fs.readdirSync(path.join(dest, '_app', 'immutable'), { recursive: true });
console.log(`[copy-dist] ${files.length} assets imutáveis publicados em backend/web/dist/`);
