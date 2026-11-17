<script lang="ts">
	import { t } from '#lib/i18n/t.js';
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
	let segment = $state(t('sources.cell.day'));

	interface Sample {
		id: number;
		nome: string;
		tipo: string;
		valor: string;
	}
	const sample = $derived<Sample[]>([
		{ id: 1, nome: 'Himalayas', tipo: t('devui.demo.public_api'), valor: '60/min' },
		{ id: 2, nome: 'Remotive', tipo: t('devui.demo.public_api'), valor: t('sources.cell.default') }
	]);
	const sampleColumns = $derived<DataColumn<Sample>[]>([
		{ key: 'nome', header: t('sources.col.name'), value: (row) => row.nome },
		{ key: 'tipo', header: t('notifications.filter.type'), value: (row) => row.tipo },
		{ key: 'valor', header: t('sources.col.rate_limit'), value: (row) => row.valor }
	]);
</script>

<svelte:head><title>{t('devui.title')} · {t('common.app_name')}</title></svelte:head>

{#if !enabled}
	<Card title={t('devui.unavailable')}>
		<p>{t('devui.unavailable_desc')}</p>
	</Card>
{:else}
	<PageHeader
		title={t('devui.title')}
		subtitle={t('devui.subtitle')}
		breadcrumb={[{ label: t('common.home'), href: '/' }, { label: t('devui.title') }]}
	/>

	<Section title={t('tasks.col.actions')} description={t('devui.demo.actions_desc')}>
		<Card>
			<div class="demo-row">
				<Button>{t('devui.demo.primary')}</Button>
				<Button variant="secondary">{t('devui.demo.secondary')}</Button>
				<Button variant="ghost">{t('devui.demo.ghost')}</Button>
				<Button variant="danger">{t('devui.demo.danger')}</Button>
				<Button loading>{t('devui.demo.loading')}</Button>
				<Button size="sm">{t('devui.demo.small')}</Button>
				<Button size="lg">{t('devui.demo.large')}</Button>
				<IconButton icon="plus" label={t('devui.demo.add')} />
				<LinkButton href="/settings" variant="secondary">{t('devui.demo.link')}</LinkButton>
			</div>
		</Card>
	</Section>

	<Section title={t('devui.section.form')} description={t('devui.section.form_desc_rich')}>
		<div class="grid grid-2">
			<Card>
				<Field label={t('devui.demo.input')} for="demo-input" hint={t('devui.demo.textarea_hint')}>
					<input id="demo-input" bind:value={inputValue} />
				</Field>
				<Field label={t('devui.demo.textarea')} for="demo-textarea">
					<Textarea id="demo-textarea" bind:value={textareaValue} rows={3} />
				</Field>
				<Field label={t('devui.demo.select')} for="demo-select">
					<Select id="demo-select" bind:value={selectValue} options={[{ value: 'a', label: t('devui.demo.option_a') }, { value: 'b', label: t('devui.demo.option_b') }]} />
				</Field>
			</Card>
			<Card>
				<Checkbox label={t('devui.demo.checkbox')} bind:checked />
				<Switch label={t('devui.demo.switch')} description="Estado persistido separadamente." bind:checked={switched} />
				<RadioGroup
					name="demo-radio"
					label={t('devui.demo.radio')}
					bind:value={radio}
					options={[{ value: 'a', label: t('devui.demo.first') }, { value: 'b', label: t('devui.demo.second') }]}
				/>
				<Combobox
					label={t('devui.demo.combobox')}
					bind:value={combobox}
					options={[{ value: 'go', label: t('devui.demo.golang') }, { value: 'ts', label: t('devui.demo.typescript') }]}
				/>
			</Card>
		</div>
	</Section>

	<Section title={t('devui.section.feedback')} description="Alert, Badge, Spinner, ProgressBar, EmptyState e Skeleton.">
		<div class="grid grid-2">
			<Card>
				<div class="demo-row"><Alert severity="info" title={t('common.info')}>{t('devui.demo.info_msg')}</Alert></div>
				<div class="demo-row"><Alert severity="success" title={t('common.success')}>{t('devui.demo.success_msg')}</Alert></div>
				<div class="demo-row"><Alert severity="warning" title={t('common.warning')}>{t('devui.demo.warning_msg')}</Alert></div>
				<div class="demo-row"><Alert severity="error" title={t('common.error')}>{t('errors.unknown')}</Alert></div>
			</Card>
			<Card>
				<div class="demo-row">
					<Badge>neutral</Badge><Badge tone="primary">primary</Badge><Badge tone="success">success</Badge>
					<Badge tone="warning">warning</Badge><Badge tone="error">error</Badge><Badge tone="info">info</Badge>
				</div>
				<div class="demo-row"><Spinner size={24} label="Carregando" /></div>
				<ProgressBar value={65} label={t('common.progress')} showValue />
				<Skeleton width="100%" height="48px" />
				<EmptyState title={t('devui.demo.empty')} description={t('devui.demo.empty_desc')} />
			</Card>
		</div>
	</Section>

	<Section title={t('devui.section.structure')} description="Card, {t('devui.demo.panel')}, Tabs e SegmentedControl.">
		<div class="grid grid-2">
			<Card title={t('devui.demo.card_title')} subtitle={t('devui.demo.card_subtitle')}>
				<Panel title={t('devui.demo.panel')}>{t('devui.demo.panel_content')}</Panel>
			</Card>
			<Card>
				<Tabs items={[{ value: 'um', label: t('devui.demo.one'), count: 2 }, { value: 'dois', label: t('devui.demo.two') }]} bind:value={tab} label={t('devui.demo.tabs')} />
				<SegmentedControl options={[{ value: t('sources.cell.day'), label: t('devui.demo.day') }, { value: 'semana', label: t('devui.demo.week') }]} bind:value={segment} label={t('devui.demo.period')} />
			</Card>
		</div>
	</Section>

	<Section title={t('devui.section.data')} description={t('devui.section.data_desc')}>
		<Card>
			<DataView columns={sampleColumns} rows={sample} getKey={(row) => row.id} caption={t('devui.demo.caption')} />
			<div class="demo-row">
				<Button onclick={() => (dialogOpen = true)}>{t('devui.demo.open_dialog')}</Button>
				<Button variant="secondary" onclick={() => (drawerOpen = true)}>{t('devui.demo.open_drawer')}</Button>
			</div>
		</Card>
	</Section>
{/if}

<Dialog open={dialogOpen} onclose={() => (dialogOpen = false)} title={t('devui.demo.dialog_example')} description={t('devui.demo.dialog_desc')}>
	<p>{t('devui.demo.dialog_body')}</p>
	{#snippet footer()}<Button onclick={() => (dialogOpen = false)}>{t('common.close')}</Button>{/snippet}
</Dialog>

{#if drawerOpen}
	<Drawer title={t('devui.demo.drawer_example')} onclose={() => (drawerOpen = false)}>
		<p>{t('devui.demo.drawer_body')}</p>
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

</style>
