<script lang="ts">
	import { onMount } from 'svelte';
	import MarkdownView from '#lib/components/MarkdownView.svelte';
	import { computeUnifiedDiff, computeSideBySideDiff, type DiffLine, type SideBySideRow } from '../../lib/diff';

	interface ActiveCurriculum {
		id: number;
		content_md: string;
		source_filename: string;
		uploaded_at: string;
	}

	let activeCV: ActiveCurriculum | null = $state(null);
	let llmConfigured = $state(false);
	let loadingCV = $state(true);

	// Upload state
	let uploading = $state(false);
	let fileInput: HTMLInputElement | null = $state(null);
	let showBasePreview = $state(false);
	let rawBaseView = $state(false);

	// Tailoring state
	let jdText = $state('');
	let tailoring = $state(false);
	let tailoredMD = $state('');
	let promptVersion = $state('');
	let status: { kind: 'success' | 'error'; text: string } | null = $state(null);

	// Diff view modes: 'diff' | 'side-by-side' | 'preview'
	let diffMode: 'diff' | 'side-by-side' | 'preview' = $state('diff');
	let copyFeedback = $state(false);

	onMount(async () => {
		await Promise.all([loadCurriculum(), checkLLM()]);
		loadingCV = false;
	});

	async function checkLLM() {
		try {
			const res = await fetch('/api/settings/llm-key');
			if (res.ok) {
				const data = await res.json();
				llmConfigured = !!data.configured;
			}
		} catch {
			// ignore network check error on init
		}
	}

	async function loadCurriculum() {
		try {
			const res = await fetch('/api/curriculum');
			if (res.ok) {
				const data = await res.json();
				activeCV = data.curriculum;
			}
		} catch {
			status = { kind: 'error', text: 'Falha ao carregar currículo ativo.' };
		}
	}

	async function handleFileUpload(e: Event) {
		const target = e.target as HTMLInputElement;
		const file = target.files?.[0];
		if (!file) return;

		uploading = true;
		status = null;

		const form = new FormData();
		form.append('file', file);

		try {
			const res = await fetch('/api/curriculum/upload', {
				method: 'POST',
				body: form
			});

			const body = await res.json().catch(() => ({}));
			if (res.ok) {
				activeCV = {
					id: body.id,
					content_md: body.content_md,
					source_filename: body.source_filename,
					uploaded_at: body.uploaded_at ?? new Date().toISOString()
				};
				showBasePreview = true;
				status = { kind: 'success', text: `Currículo "${file.name}" processado e salvo com sucesso!` };
			} else {
				status = { kind: 'error', text: body.error ?? `Erro ${res.status} no upload.` };
			}
		} catch {
			status = { kind: 'error', text: 'Falha de rede ao enviar currículo.' };
		} finally {
			uploading = false;
			if (fileInput) fileInput.value = '';
		}
	}

	async function generateTailored() {
		if (!activeCV) {
			status = { kind: 'error', text: 'Faça upload de um currículo-base primeiro.' };
			return;
		}
		if (!jdText.trim()) {
			status = { kind: 'error', text: 'Cole a descrição da vaga (JD) para adaptar o currículo.' };
			return;
		}
		if (!llmConfigured) {
			status = {
				kind: 'error',
				text: 'Chave de IA não configurada. Acesse Configurações para adicionar sua chave de IA.'
			};
			return;
		}

		tailoring = true;
		status = null;

		try {
			const res = await fetch('/api/curriculum/tailor', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ jd_text: jdText })
			});

			const body = await res.json().catch(() => ({}));
			if (res.ok) {
				tailoredMD = body.tailored_md;
				promptVersion = body.prompt_version ?? 'v1';
				status = { kind: 'success', text: 'Currículo adaptado com sucesso para a vaga!' };
			} else {
				status = { kind: 'error', text: body.error ?? `Erro ${res.status} ao adaptar currículo.` };
			}
		} catch {
			status = { kind: 'error', text: 'Falha de rede ao conectar ao serviço de IA.' };
		} finally {
			tailoring = false;
		}
	}

	let unifiedDiff = $derived.by<DiffLine[]>(() => {
		if (!activeCV?.content_md || !tailoredMD) return [];
		return computeUnifiedDiff(activeCV.content_md, tailoredMD);
	});

	let sideBySideDiff = $derived.by<SideBySideRow[]>(() => {
		if (!activeCV?.content_md || !tailoredMD) return [];
		return computeSideBySideDiff(activeCV.content_md, tailoredMD);
	});

	function copyTailoredMarkdown() {
		if (!tailoredMD) return;
		navigator.clipboard.writeText(tailoredMD);
		copyFeedback = true;
		setTimeout(() => {
			copyFeedback = false;
		}, 2000);
	}

	function downloadTailoredMarkdown() {
		if (!tailoredMD) return;
		const blob = new Blob([tailoredMD], { type: 'text/markdown;charset=utf-8' });
		const url = URL.createObjectURL(blob);
		const a = document.createElement('a');
		a.href = url;
		a.download = `curriculo_adaptado_${promptVersion || 'v1'}.md`;
		a.click();
		URL.revokeObjectURL(url);
	}
</script>

<svelte:head>
	<title>Currículo · openJob</title>
</svelte:head>

<h1 class="page-title">Currículo & Adaptação</h1>
<p class="page-subtitle">
	Upload do currículo-base (PDF, MD ou TXT) e geração de versões adaptadas com IA para cada vaga.
</p>

{#if status}
	<div class="status {status.kind}" role="status">
		{status.text}
	</div>
{/if}

{#if !llmConfigured && !loadingCV}
	<div class="warning-banner">
		⚠️ <strong>Atenção:</strong> Provedor de IA não configurado. Você pode enviar seu currículo, mas precisará configurar a chave em
		<a href="/settings">Configurações</a> para gerar adaptações.
	</div>
{/if}

<!-- 1. Currículo Base Card -->
<section class="card" aria-label="Currículo Base">
	<header class="card-head">
		<div>
			<h2>Currículo Base</h2>
			<p class="hint">Uma única versão ativa por vez (novo upload substitui a anterior).</p>
		</div>
		<span class="pill {activeCV ? 'on' : 'off'}">
			{activeCV ? 'ativo' : 'nenhum'}
		</span>
	</header>

	<div class="upload-zone">
		<input
			type="file"
			id="cv-file"
			accept=".pdf,.md,.markdown,.txt"
			bind:this={fileInput}
			onchange={handleFileUpload}
			style="display: none;"
		/>

		{#if activeCV}
			<div class="cv-info">
				<div class="cv-details">
					<span class="file-icon">📄</span>
					<div>
						<strong>{activeCV.source_filename || 'curriculo.md'}</strong>
						<span class="meta">
							{new Date(activeCV.uploaded_at).toLocaleString('pt-BR')} · {activeCV.content_md.length} caracteres
						</span>
					</div>
				</div>

				<div class="cv-actions">
					<button
						type="button"
						class="secondary"
						disabled={uploading}
						onclick={() => fileInput?.click()}
					>
						{uploading ? 'Enviando…' : 'Substituir arquivo'}
					</button>

					<button
						type="button"
						class="ghost"
						onclick={() => (showBasePreview = !showBasePreview)}
					>
						{showBasePreview ? 'Ocultar preview' : 'Visualizar base'}
					</button>
				</div>
			</div>
		{:else}
			<div class="empty-upload">
				<p>Arraste ou selecione seu currículo em <strong>PDF</strong>, <strong>Markdown (.md)</strong> ou <strong>TXT</strong>.</p>
				<button
					type="button"
					class="primary"
					disabled={uploading}
					onclick={() => fileInput?.click()}
				>
					{uploading ? 'Processando…' : 'Selecionar currículo'}
				</button>
			</div>
		{/if}
	</div>

	{#if activeCV && showBasePreview}
		<div class="preview-box">
			<div class="preview-header">
				<span>Preview do Currículo Base</span>
				<button
					type="button"
					class="text-btn"
					onclick={() => (rawBaseView = !rawBaseView)}
				>
					{rawBaseView ? 'Modo Renderizado' : 'Modo Markdown Bruto'}
				</button>
			</div>

			<div class="preview-content">
				{#if rawBaseView}
					<pre class="raw-markdown">{activeCV.content_md}</pre>
				{:else}
					<MarkdownView content={activeCV.content_md} />
				{/if}
			</div>
		</div>
	{/if}
</section>

<!-- 2. Formulário de Adaptação (Tailoring) -->
<section class="card" aria-label="Adaptação para Vaga">
	<header class="card-head">
		<div>
			<h2>Adaptação com IA (Tailoring)</h2>
			<p class="hint">Cole a descrição da vaga (JD) para realinhar realizações e palavras-chave.</p>
		</div>
		{#if promptVersion}
			<span class="pill prompt-tag">prompt {promptVersion}</span>
		{/if}
	</header>

	<label for="jd_text">Descrição da vaga (Job Description)</label>
	<textarea
		id="jd_text"
		rows="6"
		placeholder="Cole aqui o texto da vaga (requisitos, responsabilidades, tecnologias desejadas)..."
		bind:value={jdText}
		disabled={tailoring}
	></textarea>

	<div class="actions">
		<button
			type="button"
			class="primary"
			disabled={tailoring || !activeCV || !jdText.trim() || !llmConfigured}
			onclick={generateTailored}
		>
			{#if tailoring}
				<span class="spinner"></span> Adaptando com IA…
			{:else}
				Gerar versão adaptada
			{/if}
		</button>
	</div>
</section>

<!-- 3. Resultado e Diff Visual -->
{#if tailoredMD}
	<section class="card result-card" aria-label="Currículo Adaptado">
		<header class="card-head">
			<div>
				<h2>Resultado Adaptado & Comparação</h2>
				<p class="hint">Compare as alterações sugeridas pela IA com seu currículo original.</p>
			</div>

			<div class="mode-tabs">
				<button
					type="button"
					class="tab {diffMode === 'diff' ? 'active' : ''}"
					onclick={() => (diffMode = 'diff')}
				>
					Diff Visual
				</button>
				<button
					type="button"
					class="tab {diffMode === 'side-by-side' ? 'active' : ''}"
					onclick={() => (diffMode = 'side-by-side')}
				>
					Lado a Lado
				</button>
				<button
					type="button"
					class="tab {diffMode === 'preview' ? 'active' : ''}"
					onclick={() => (diffMode = 'preview')}
				>
					Renderizado
				</button>
			</div>
		</header>

		<!-- Toolbar -->
		<div class="result-toolbar">
			<button type="button" class="action-btn" onclick={copyTailoredMarkdown}>
				{copyFeedback ? '✓ Copiado!' : '📋 Copiar Markdown'}
			</button>
			<button type="button" class="action-btn" onclick={downloadTailoredMarkdown}>
				⬇ Baixar .md
			</button>
		</div>

		<!-- Diff or Preview Container -->
		<div class="diff-container">
			{#if diffMode === 'diff'}
				<div class="unified-diff">
					{#each unifiedDiff as line}
						<div class="diff-row {line.type}">
							<span class="sign">
								{line.type === 'added' ? '+' : line.type === 'removed' ? '-' : ' '}
							</span>
							<span class="line-content">{line.text || ' '}</span>
						</div>
					{/each}
				</div>
			{:else if diffMode === 'side-by-side'}
				<div class="side-by-side-wrapper">
					<div class="sbs-header">
						<span>Currículo Base</span>
						<span>Currículo Adaptado</span>
					</div>
					<div class="sbs-body">
						{#each sideBySideDiff as row}
							<div class="sbs-row">
								<div class="sbs-col left {row.left?.type ?? ''}">
									<span class="lineno">{row.left?.lineNo ?? ''}</span>
									<span class="line-text">{row.left?.text ?? ''}</span>
								</div>
								<div class="sbs-col right {row.right?.type ?? ''}">
									<span class="lineno">{row.right?.lineNo ?? ''}</span>
									<span class="line-text">{row.right?.text ?? ''}</span>
								</div>
							</div>
						{/each}
					</div>
				</div>
			{:else}
				<div class="adapted-preview">
					<MarkdownView content={tailoredMD} />
				</div>
			{/if}
		</div>
	</section>
{/if}

<style>
	.card {
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: var(--radius);
		padding: 1.25rem;
		margin-bottom: 1.25rem;
	}

	.card-head {
		display: flex;
		align-items: flex-start;
		justify-content: space-between;
		gap: 0.5rem;
		margin-bottom: 0.75rem;
	}

	.card-head h2 {
		font-size: 1.1rem;
		margin: 0;
	}

	.pill {
		font-size: 0.75rem;
		font-weight: 600;
		border-radius: 999px;
		padding: 0.2rem 0.65rem;
		white-space: nowrap;
	}

	.pill.on {
		background: #ecfdf5;
		color: #047857;
	}

	.pill.off {
		background: var(--accent-soft);
		color: var(--accent);
	}

	.pill.prompt-tag {
		background: #f3e8ff;
		color: #7e22ce;
	}

	.hint {
		color: var(--text-muted);
		font-size: 0.8rem;
		margin: 0.25rem 0 0;
	}

	.warning-banner {
		background: #fffbeb;
		border: 1px solid #fde68a;
		color: #92400e;
		border-radius: var(--radius);
		padding: 0.75rem 1rem;
		font-size: 0.9rem;
		margin-bottom: 1rem;
	}

	.warning-banner a {
		color: #b45309;
		font-weight: 600;
		text-decoration: underline;
	}

	.status {
		border-radius: var(--radius);
		padding: 0.6rem 0.8rem;
		font-size: 0.9rem;
		margin-bottom: 1rem;
	}

	.status.success {
		background: #ecfdf5;
		color: #047857;
	}

	.status.error {
		background: #fef2f2;
		color: #b91c1c;
	}

	.upload-zone {
		margin-top: 0.5rem;
	}

	.empty-upload {
		border: 2px dashed var(--border);
		border-radius: var(--radius);
		padding: 1.5rem 1rem;
		text-align: center;
		background: var(--bg);
	}

	.empty-upload p {
		margin: 0 0 1rem;
		font-size: 0.95rem;
		color: var(--text-muted);
	}

	.cv-info {
		display: flex;
		flex-direction: column;
		gap: 0.75rem;
		padding: 0.75rem;
		background: var(--bg);
		border-radius: var(--radius);
		border: 1px solid var(--border);
	}

	.cv-details {
		display: flex;
		align-items: center;
		gap: 0.75rem;
	}

	.file-icon {
		font-size: 1.75rem;
	}

	.meta {
		display: block;
		font-size: 0.8rem;
		color: var(--text-muted);
	}

	.cv-actions {
		display: flex;
		gap: 0.5rem;
		flex-wrap: wrap;
	}

	label {
		display: block;
		font-size: 0.85rem;
		font-weight: 600;
		margin: 0.75rem 0 0.25rem;
	}

	textarea {
		width: 100%;
		border: 1px solid var(--border);
		border-radius: var(--radius);
		padding: 0.75rem;
		font-size: 0.95rem;
		background: var(--bg);
		color: var(--text);
		font-family: inherit;
		resize: vertical;
		min-height: 7rem;
	}

	textarea:focus {
		outline: 2px solid var(--accent);
		outline-offset: 1px;
		border-color: var(--accent);
	}

	.actions {
		display: flex;
		gap: 0.75rem;
		margin-top: 1rem;
	}

	.primary,
	.secondary,
	.ghost {
		border: 0;
		border-radius: var(--radius);
		padding: 0.65rem 1rem;
		font-size: 0.9rem;
		font-weight: 600;
		cursor: pointer;
		display: inline-flex;
		align-items: center;
		justify-content: center;
		gap: 0.5rem;
	}

	.primary {
		background: var(--accent);
		color: #fff;
	}

	.primary:disabled {
		opacity: 0.6;
		cursor: default;
	}

	.secondary {
		background: var(--accent-soft);
		color: var(--accent);
	}

	.ghost {
		background: var(--surface);
		border: 1px solid var(--border);
		color: var(--text);
	}

	.text-btn {
		background: none;
		border: none;
		color: var(--accent);
		font-size: 0.85rem;
		cursor: pointer;
		text-decoration: underline;
		padding: 0;
	}

	.preview-box {
		margin-top: 1rem;
		border-top: 1px solid var(--border);
		padding-top: 0.75rem;
	}

	.preview-header {
		display: flex;
		justify-content: space-between;
		font-size: 0.85rem;
		font-weight: 600;
		color: var(--text-muted);
		margin-bottom: 0.5rem;
	}

	.preview-content {
		max-height: 20rem;
		overflow-y: auto;
		background: var(--bg);
		padding: 0.75rem;
		border-radius: var(--radius);
		border: 1px solid var(--border);
	}

	.raw-markdown {
		margin: 0;
		font-family: monospace;
		font-size: 0.85rem;
		white-space: pre-wrap;
		word-break: break-word;
	}

	/* Diff Container & Tabs */
	.mode-tabs {
		display: flex;
		gap: 0.25rem;
		background: var(--bg);
		padding: 0.2rem;
		border-radius: var(--radius);
		border: 1px solid var(--border);
	}

	.tab {
		border: none;
		background: transparent;
		padding: 0.35rem 0.65rem;
		font-size: 0.8rem;
		font-weight: 600;
		border-radius: calc(var(--radius) - 2px);
		cursor: pointer;
		color: var(--text-muted);
	}

	.tab.active {
		background: var(--surface);
		color: var(--text);
		box-shadow: 0 1px 2px rgba(0, 0, 0, 0.05);
	}

	.result-toolbar {
		display: flex;
		gap: 0.5rem;
		margin-bottom: 0.75rem;
	}

	.action-btn {
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: var(--radius);
		padding: 0.4rem 0.75rem;
		font-size: 0.85rem;
		font-weight: 500;
		cursor: pointer;
	}

	.diff-container {
		background: var(--bg);
		border: 1px solid var(--border);
		border-radius: var(--radius);
		overflow-x: auto;
		max-height: 30rem;
		overflow-y: auto;
	}

	.unified-diff {
		font-family: monospace;
		font-size: 0.85rem;
		line-height: 1.45;
	}

	.diff-row {
		display: flex;
		padding: 0.1rem 0.5rem;
		white-space: pre-wrap;
		word-break: break-word;
	}

	.diff-row.added {
		background: #dcfce7;
		color: #14532d;
	}

	.diff-row.removed {
		background: #fee2e2;
		color: #991b1b;
	}

	.sign {
		width: 1.25rem;
		user-select: none;
		font-weight: bold;
	}

	.line-content {
		flex: 1;
	}

	/* Side by Side Diff */
	.side-by-side-wrapper {
		min-width: 35rem;
		font-family: monospace;
		font-size: 0.8rem;
	}

	.sbs-header {
		display: grid;
		grid-template-columns: 1fr 1fr;
		background: #f1f5f9;
		font-weight: 600;
		padding: 0.4rem 0.75rem;
		border-bottom: 1px solid var(--border);
	}

	.sbs-row {
		display: grid;
		grid-template-columns: 1fr 1fr;
		border-bottom: 1px solid #f1f5f9;
	}

	.sbs-col {
		display: flex;
		padding: 0.1rem 0.5rem;
		overflow-x: hidden;
		white-space: pre-wrap;
		word-break: break-word;
	}

	.sbs-col.left {
		border-right: 1px solid var(--border);
	}

	.sbs-col.added {
		background: #dcfce7;
		color: #14532d;
	}

	.sbs-col.removed {
		background: #fee2e2;
		color: #991b1b;
	}

	.lineno {
		width: 2rem;
		color: var(--text-muted);
		user-select: none;
		font-size: 0.75rem;
	}

	.line-text {
		flex: 1;
	}

	.adapted-preview {
		padding: 1rem;
		background: var(--surface);
	}

	.spinner {
		display: inline-block;
		width: 1rem;
		height: 1rem;
		border: 2px solid rgba(255, 255, 255, 0.4);
		border-radius: 50%;
		border-top-color: #fff;
		animation: spin 0.8s linear infinite;
	}

	@keyframes spin {
		to {
			transform: rotate(360deg);
		}
	}
</style>
