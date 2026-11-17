<script lang="ts">
	import { onMount } from 'svelte';
	import SettingsNav from '#lib/components/SettingsNav.svelte';
	import PageHeader from '#lib/ui/PageHeader.svelte';
	import Card from '#lib/ui/Card.svelte';
	import Field from '#lib/ui/Field.svelte';
	import Input from '#lib/ui/Input.svelte';
	import Button from '#lib/ui/Button.svelte';
	import IconButton from '#lib/ui/IconButton.svelte';
	import Alert from '#lib/ui/Alert.svelte';
	import Badge from '#lib/ui/Badge.svelte';

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
				status = { kind: 'success', text: 'Chave salva com sucesso — armazenada cifrada no seu banco local.' };
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
			status = result.ok
				? { kind: 'success', text: result.message ?? 'Conexão estabelecida.' }
				: { kind: 'error', text: result.error ?? 'Falha na conexão.' };
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

<PageHeader
	title="Configurações"
	subtitle="Chaves e preferências — sua chave fica no seu computador, cifrada, e nunca volta para o navegador."
	breadcrumb={[{ label: 'Início', href: '/' }, { label: 'Configurações' }]}
/>

<SettingsNav active="llm" />

{#if status}
	<Alert severity={status.kind}>{status.text}</Alert>
{/if}

<Card title="Provedor de IA (BYOK)" subtitle="Compatível com OpenAI, Groq, Ollama, Together e Fireworks.">
	{#snippet actions()}
		<Badge tone={configured ? 'success' : 'warning'}>
			{configured ? 'configurado' : 'não configurado'}
		</Badge>
	{/snippet}
	<form onsubmit={(e) => { e.preventDefault(); save(); }}>
		<Field label="Base URL" for="base_url">
			<Input id="base_url" type="url" bind:value={baseUrl} placeholder="https://api.openai.com/v1" autocomplete="off" spellcheck="false" />
		</Field>
		<Field label="Modelo" for="model">
			<Input id="model" bind:value={model} placeholder="gpt-4o-mini" autocomplete="off" spellcheck="false" />
		</Field>
		<Field label="API key" for="api_key" hint="A chave é cifrada (AES-GCM) antes de ir para o SQLite e não pode ser recuperada pela UI.">
			<div class="key-row">
				<Input
					id="api_key"
					type={showKey ? 'text' : 'password'}
					placeholder={configured ? '•••••••• (salva — digite para trocar)' : 'sk-…'}
					bind:value={apiKey}
					autocomplete="off"
					spellcheck="false"
				/>
				<IconButton
					icon={showKey ? 'eye-off' : 'eye'}
					label={showKey ? 'Ocultar chave' : 'Revelar chave'}
					variant="ghost"
					onclick={() => (showKey = !showKey)}
				/>
			</div>
		</Field>
		<div class="form-actions">
			<Button type="submit" loading={saving}>Salvar</Button>
			<Button type="button" variant="secondary" loading={testing} onclick={testConnection}>Testar conexão</Button>
		</div>
	</form>
</Card>

<style>
	form {
		display: flex;
		flex-direction: column;
		gap: var(--space-4);
	}

	.key-row {
		display: flex;
		gap: var(--space-2);
	}

	.key-row :global(.input) {
		flex: 1;
		min-width: 0;
	}

	.form-actions {
		margin-top: var(--space-2);
	}
</style>
