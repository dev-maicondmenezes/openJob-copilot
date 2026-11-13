<script lang="ts">
	import { onMount } from 'svelte';

	interface LLMConfig {
		configured: boolean;
		base_url: string;
		model: string;
	}

	interface PingResult {
		ok: boolean;
		message?: string;
		error?: string;
	}

	let baseUrl = $state('');
	let model = $state('');
	let apiKey = $state('');
	let showKey = $state(false);
	let configured = $state(false);
	let saving = $state(false);
	let testing = $state(false);
	let status: { kind: 'success' | 'error'; text: string } | null = $state(null);

	onMount(async () => {
		try {
			const res = await fetch('/api/settings/llm');
			if (res.ok) {
				const cfg: LLMConfig = await res.json();
				baseUrl = cfg.base_url;
				model = cfg.model;
				configured = cfg.configured;
			}
		} catch {
			status = { kind: 'error', text: 'Não foi possível carregar a configuração atual.' };
		}
	});

	async function save() {
		if (!apiKey.trim()) {
			status = { kind: 'error', text: 'Informe a API key.' };
			return;
		}
		saving = true;
		status = null;
		try {
			const res = await fetch('/api/settings/llm-key', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ base_url: baseUrl, model, api_key: apiKey })
			});
			if (res.status === 204) {
				configured = true;
				apiKey = '';
				status = {
					kind: 'success',
					text: 'Chave salva com sucesso — armazenada cifrada no seu banco local.'
				};
			} else {
				const body = await res.json().catch(() => ({}));
				status = { kind: 'error', text: body.error ?? `Erro ${res.status} ao salvar.` };
			}
		} catch {
			status = { kind: 'error', text: 'Falha de rede ao salvar.' };
		} finally {
			saving = false;
		}
	}

	async function testConnection() {
		testing = true;
		status = null;
		try {
			const res = await fetch('/api/settings/llm-ping', { method: 'POST' });
			const result: PingResult = await res.json();
			if (result.ok) {
				status = { kind: 'success', text: result.message ?? 'Conexão estabelecida.' };
			} else {
				status = { kind: 'error', text: result.error ?? 'Falha na conexão.' };
			}
		} catch {
			status = { kind: 'error', text: 'Falha de rede ao testar a conexão.' };
		} finally {
			testing = false;
		}
	}
</script>

<svelte:head>
	<title>Configurações · openJob</title>
</svelte:head>

<h1 class="page-title">Configurações</h1>
<p class="page-subtitle">
	Provedor de IA (BYOK) — sua chave fica no seu computador, cifrada, e nunca volta para o navegador.
</p>

{#if status}
	<div class="status {status.kind}" role="status">{status.text}</div>
{/if}

<section class="card" aria-label="Provedor de IA">
	<header class="card-head">
		<h2>Provedor de IA</h2>
		<span class="pill {configured ? 'on' : 'off'}">
			{configured ? 'configurado' : 'não configurado'}
		</span>
	</header>

	<form onsubmit={(e) => { e.preventDefault(); save(); }}>
		<label for="base_url">Base URL</label>
		<input
			id="base_url"
			type="url"
			placeholder="https://api.openai.com/v1"
			bind:value={baseUrl}
			autocomplete="off"
			spellcheck="false"
		/>
		<p class="hint">Compatível com OpenAI, Groq, Ollama, Together, Fireworks…</p>

		<label for="model">Modelo</label>
		<input
			id="model"
			type="text"
			placeholder="gpt-4o-mini"
			bind:value={model}
			autocomplete="off"
			spellcheck="false"
		/>

		<label for="api_key">API key</label>
		<div class="key-row">
			<input
				id="api_key"
				type={showKey ? 'text' : 'password'}
				placeholder={configured ? '•••••••• (salva — digite para trocar)' : 'sk-…'}
				bind:value={apiKey}
				autocomplete="off"
				spellcheck="false"
			/>
			<button
				type="button"
				class="ghost"
				aria-label={showKey ? 'Ocultar chave' : 'Mostrar chave'}
				onclick={() => (showKey = !showKey)}
			>
				{showKey ? '🙈' : '👁'}
			</button>
		</div>
		<p class="hint">
			A chave é cifrada (AES-GCM) antes de ir para o SQLite e não pode ser recuperada pela UI.
		</p>

		<div class="actions">
			<button type="submit" class="primary" disabled={saving}>
				{saving ? 'Salvando…' : 'Salvar'}
			</button>
			<button type="button" class="secondary" disabled={testing} onclick={testConnection}>
				{testing ? 'Testando…' : 'Testar conexão'}
			</button>
		</div>
	</form>
</section>

<style>
	.card {
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: var(--radius);
		padding: 1rem;
	}

	.card-head {
		display: flex;
		align-items: center;
		justify-content: space-between;
		margin-bottom: 0.75rem;
	}

	.card-head h2 {
		font-size: 1rem;
		margin: 0;
	}

	.pill {
		font-size: 0.75rem;
		font-weight: 600;
		border-radius: 999px;
		padding: 0.125rem 0.625rem;
	}

	.pill.on {
		background: #ecfdf5;
		color: #047857;
	}

	.pill.off {
		background: var(--accent-soft);
		color: var(--accent);
	}

	label {
		display: block;
		font-size: 0.85rem;
		font-weight: 600;
		margin: 0.75rem 0 0.25rem;
	}

	input {
		width: 100%;
		border: 1px solid var(--border);
		border-radius: var(--radius);
		padding: 0.6rem 0.75rem;
		font-size: 1rem;
		background: var(--bg);
		color: var(--text);
	}

	input:focus {
		outline: 2px solid var(--accent);
		outline-offset: 1px;
		border-color: var(--accent);
	}

	.key-row {
		display: flex;
		gap: 0.5rem;
	}

	.key-row input {
		flex: 1;
	}

	.ghost {
		border: 1px solid var(--border);
		background: var(--surface);
		border-radius: var(--radius);
		padding: 0 0.75rem;
		font-size: 1rem;
		cursor: pointer;
	}

	.hint {
		color: var(--text-muted);
		font-size: 0.8rem;
		margin: 0.25rem 0 0;
	}

	.actions {
		display: flex;
		gap: 0.75rem;
		margin-top: 1.25rem;
	}

	.primary,
	.secondary {
		flex: 1;
		border: 0;
		border-radius: var(--radius);
		padding: 0.7rem 1rem;
		font-size: 0.95rem;
		font-weight: 600;
		cursor: pointer;
	}

	.primary {
		background: var(--accent);
		color: #fff;
	}

	.primary:disabled,
	.secondary:disabled {
		opacity: 0.6;
		cursor: default;
	}

	.secondary {
		background: var(--accent-soft);
		color: var(--accent);
	}

	.status {
		border-radius: var(--radius);
		padding: 0.6rem 0.8rem;
		font-size: 0.9rem;
		margin-bottom: 0.75rem;
	}

	.status.success {
		background: #ecfdf5;
		color: #047857;
	}

	.status.error {
		background: #fef2f2;
		color: #b91c1c;
	}
</style>
