<script lang="ts">
	/**
	 * Página de referência ("storybook mínimo") do design system.
	 * Protegida por OPENJOB_DEV_UI=1 — ver docs/design/tokens.md.
	 */
	import PageHeader from '#lib/ui/PageHeader.svelte';
	import Card from '#lib/ui/Card.svelte';
	import Panel from '#lib/ui/Panel.svelte';
	import Section from '#lib/ui/Section.svelte';
	import Field from '#lib/ui/Field.svelte';
	import Input from '#lib/ui/Input.svelte';
	import Textarea from '#lib/ui/Textarea.svelte';
	import Select from '#lib/ui/Select.svelte';
	import Checkbox from '#lib/ui/Checkbox.svelte';
	import Switch from '#lib/ui/Switch.svelte';
	import RadioGroup from '#lib/ui/RadioGroup.svelte';
	import Combobox from '#lib/ui/Combobox.svelte';
	import Button from '#lib/ui/Button.svelte';
	import IconButton from '#lib/ui/IconButton.svelte';
	import LinkButton from '#lib/ui/LinkButton.svelte';
	import Alert from '#lib/ui/Alert.svelte';
	import Badge from '#lib/ui/Badge.svelte';
	import Spinner from '#lib/ui/Spinner.svelte';
	import Skeleton from '#lib/ui/Skeleton.svelte';
	import ProgressBar from '#lib/ui/ProgressBar.svelte';
	import EmptyState from '#lib/ui/EmptyState.svelte';
	import Tabs from '#lib/ui/Tabs.svelte';
	import SegmentedControl from '#lib/ui/SegmentedControl.svelte';
	import Dialog from '#lib/ui/Dialog.svelte';
	import Drawer from '#lib/ui/Drawer.svelte';
	import DataView from '#lib/ui/DataView.svelte';
	import type { DataColumn } from '#lib/ui/data.js';

	const enabled = import.meta.env.OPENJOB_DEV_UI === '1' || import.meta.env.VITE_OPENJOB_DEV_UI === '1';

	let inputValue = $state('texto de exemplo');
	let textareaValue = $state('Linha 1\nLinha 2');
	let selectValue = $state('b');
	let checked = $state(true);
	let switched = $state(false);
	let radio = $state('a');
	let combobox = $state('');
	let dialogOpen = $state(false);
	let drawerOpen = $state(false);
	let tab = $state('um');
	let segment = $state('dia');

	interface Sample {
		id: number;
		nome: string;
		tipo: string;
		valor: string;
	}
	const sample: Sample[] = [
		{ id: 1, nome: 'Himalayas', tipo: 'API pública', valor: '60/min' },
		{ id: 2, nome: 'Remotive', tipo: 'API pública', valor: 'Padrão' }
	];
	const sampleColumns: DataColumn<Sample>[] = [
		{ key: 'nome', header: 'Nome', value: (row) => row.nome },
		{ key: 'tipo', header: 'Tipo', value: (row) => row.tipo },
		{ key: 'valor', header: 'Rate limit', value: (row) => row.valor }
	];
</script>

<svelte:head><title>Design System · openJob</title></svelte:head>

{#if !enabled}
	<Card title="Página indisponível">
		<p>Defina <code>OPENJOB_DEV_UI=1</code> para habilitar a página de referência.</p>
	</Card>
{:else}
	<PageHeader
		title="Design System"
		subtitle="Referência visual dos tokens e componentes — CP12."
		breadcrumb={[{ label: 'Início', href: '/' }, { label: 'Design System' }]}
	/>

	<Section title="Ações" description="Button (primary/secondary/ghost/danger), IconButton e LinkButton.">
		<Card>
			<div class="demo-row">
				<Button>Primary</Button>
				<Button variant="secondary">Secondary</Button>
				<Button variant="ghost">Ghost</Button>
				<Button variant="danger">Danger</Button>
				<Button loading>Loading</Button>
				<Button size="sm">Small</Button>
				<Button size="lg">Large</Button>
				<IconButton icon="plus" label="Adicionar" />
				<LinkButton href="/settings" variant="secondary">Link</LinkButton>
			</div>
		</Card>
	</Section>

	<Section title="Formulário" description="Field + controles. Todos usam os tokens de espaçamento e foco.">
		<div class="grid grid-2">
			<Card>
				<Field label="Input" for="demo-input" hint="Texto de ajuda do campo.">
					<Input id="demo-input" bind:value={inputValue} />
				</Field>
				<Field label="Textarea" for="demo-textarea">
					<Textarea id="demo-textarea" bind:value={textareaValue} rows={3} />
				</Field>
				<Field label="Select" for="demo-select">
					<Select id="demo-select" bind:value={selectValue} options={[{ value: 'a', label: 'Opção A' }, { value: 'b', label: 'Opção B' }]} />
				</Field>
			</Card>
			<Card>
				<Checkbox label="Checkbox" bind:checked />
				<Switch label="Switch" description="Estado persistido separadamente." bind:checked={switched} />
				<RadioGroup
					name="demo-radio"
					label="Radio group"
					bind:value={radio}
					options={[{ value: 'a', label: 'Primeira' }, { value: 'b', label: 'Segunda' }]}
				/>
				<Combobox
					label="Combobox"
					bind:value={combobox}
					options={[{ value: 'go', label: 'Golang' }, { value: 'ts', label: 'TypeScript' }]}
				/>
			</Card>
		</div>
	</Section>

	<Section title="Feedback" description="Alert, Badge, Spinner, ProgressBar, EmptyState e Skeleton.">
		<div class="grid grid-2">
			<Card>
				<div class="demo-row"><Alert severity="info" title="Info">Mensagem informativa.</Alert></div>
				<div class="demo-row"><Alert severity="success" title="Sucesso">Operação concluída.</Alert></div>
				<div class="demo-row"><Alert severity="warning" title="Aviso">Atenção necessária.</Alert></div>
				<div class="demo-row"><Alert severity="error" title="Erro">Algo deu errado.</Alert></div>
			</Card>
			<Card>
				<div class="demo-row">
					<Badge>neutral</Badge><Badge tone="primary">primary</Badge><Badge tone="success">success</Badge>
					<Badge tone="warning">warning</Badge><Badge tone="error">error</Badge><Badge tone="info">info</Badge>
				</div>
				<div class="demo-row"><Spinner size={24} label="Carregando" /></div>
				<ProgressBar value={65} label="Progresso" showValue />
				<Skeleton width="100%" height="48px" />
				<EmptyState title="Estado vazio" description="Usado quando não há registros." />
			</Card>
		</div>
	</Section>

	<Section title="Estrutura" description="Card, Panel, Tabs e SegmentedControl.">
		<div class="grid grid-2">
			<Card title="Card com cabeçalho" subtitle="Subtítulo do cartão">
				<Panel title="Panel">Conteúdo em painel.</Panel>
			</Card>
			<Card>
				<Tabs items={[{ value: 'um', label: 'Um', count: 2 }, { value: 'dois', label: 'Dois' }]} bind:value={tab} label="Abas de exemplo" />
				<SegmentedControl options={[{ value: 'dia', label: 'Dia' }, { value: 'semana', label: 'Semana' }]} bind:value={segment} label="Período" />
			</Card>
		</div>
	</Section>

	<Section title="Dados e overlays" description="DataView alterna card/tabela por breakpoint; Dialog alterna sheet/modal.">
		<Card>
			<DataView columns={sampleColumns} rows={sample} getKey={(row) => row.id} caption="Exemplo" />
			<div class="demo-row">
				<Button onclick={() => (dialogOpen = true)}>Abrir Dialog</Button>
				<Button variant="secondary" onclick={() => (drawerOpen = true)}>Abrir Drawer</Button>
			</div>
		</Card>
	</Section>
{/if}

<Dialog open={dialogOpen} onclose={() => (dialogOpen = false)} title="Diálogo de exemplo" description="Sheet no mobile/tablet, modal no desktop.">
	<p>Conteúdo do diálogo. Escape fecha e o foco volta ao disparador.</p>
	{#snippet footer()}<Button onclick={() => (dialogOpen = false)}>Fechar</Button>{/snippet}
</Dialog>

{#if drawerOpen}
	<Drawer title="Drawer de exemplo" onclose={() => (drawerOpen = false)}>
		<p>Conteúdo do drawer lateral.</p>
	</Drawer>
{/if}

<style>
	.demo-row {
		display: flex;
		align-items: center;
		gap: var(--space-2);
		flex-wrap: wrap;
		margin-bottom: var(--space-3);
	}

	code {
		font-family: var(--font-mono);
		background: var(--color-surface-elevated);
		padding: var(--space-1) var(--space-2);
		border-radius: var(--radius-sm);
	}
</style>
