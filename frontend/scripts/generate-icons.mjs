// Gera os ícones PNG do manifest PWA sem dependências externas
// (encoder PNG mínimo: IHDR + IDAT(deflate) + IEND).
// Uso: node scripts/generate-icons.mjs
import zlib from 'node:zlib';
import fs from 'node:fs';
import path from 'node:path';

const outDir = path.resolve(process.cwd(), 'static');
fs.mkdirSync(outDir, { recursive: true });

// --- PNG mínimo ------------------------------------------------------------
const CRC_TABLE = (() => {
	const table = new Uint32Array(256);
	for (let n = 0; n < 256; n++) {
		let c = n;
		for (let k = 0; k < 8; k++) c = c & 1 ? 0xedb88320 ^ (c >>> 1) : c >>> 1;
		table[n] = c >>> 0;
	}
	return table;
})();

function crc32(buf) {
	let crc = 0xffffffff;
	for (const byte of buf) crc = CRC_TABLE[(crc ^ byte) & 0xff] ^ (crc >>> 8);
	return (crc ^ 0xffffffff) >>> 0;
}

function chunk(type, data) {
	const len = Buffer.alloc(4);
	len.writeUInt32BE(data.length);
	const body = Buffer.concat([Buffer.from(type, 'ascii'), data]);
	const crc = Buffer.alloc(4);
	crc.writeUInt32BE(crc32(body));
	return Buffer.concat([len, body, crc]);
}

/** pixelFn(x, y) -> [r, g, b, a] */
function encode_png(width, height, pixel_fn) {
	const ihdr = Buffer.alloc(13);
	ihdr.writeUInt32BE(width, 0);
	ihdr.writeUInt32BE(height, 4);
	ihdr[8] = 8; // bit depth
	ihdr[9] = 6; // RGBA
	const stride = width * 4 + 1;
	const raw = Buffer.alloc(height * stride);
	for (let y = 0; y < height; y++) {
		raw[y * stride] = 0; // filtro "none"
		for (let x = 0; x < width; x++) {
			const [r, g, b, a] = pixel_fn(x, y);
			const off = y * stride + 1 + x * 4;
			raw[off] = r;
			raw[off + 1] = g;
			raw[off + 2] = b;
			raw[off + 3] = a;
		}
	}
	const signature = Buffer.from([0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a]);
	return Buffer.concat([
		signature,
		chunk('IHDR', ihdr),
		chunk('IDAT', zlib.deflateSync(raw, { level: 9 })),
		chunk('IEND', Buffer.alloc(0))
	]);
}

// --- desenho: fundo escuro + anel branco ("O" de openJob) -------------------
const BG = [15, 23, 42]; // #0f172a
const FG = [255, 255, 255];

function ring(x, y, size, outer_r, thickness) {
	const c = (size - 1) / 2;
	const d = Math.hypot(x - c, y - c);
	const inner_r = outer_r - thickness;
	return d <= outer_r && d >= inner_r;
}

/** maskable: conteúdo dentro da zona segura (raio < 80% do tamanho) */
function icon(size, maskable) {
	const outer = maskable ? size * 0.3 : size * 0.34;
	const thickness = size * 0.09;
	return encode_png(size, size, (x, y) => {
		const is_fg = ring(x, y, size, outer, thickness);
		return [...(is_fg ? FG : BG), 255];
	});
}

const files = [
	['pwa-192.png', icon(192, false)],
	['pwa-512.png', icon(512, false)],
	['pwa-maskable-512.png', icon(512, true)]
];

for (const [name, buf] of files) {
	fs.writeFileSync(path.join(outDir, name), buf);
	console.log(`OK ${name} (${buf.length} bytes)`);
}
