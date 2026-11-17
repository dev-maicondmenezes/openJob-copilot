<script lang="ts">
	import { t } from '#lib/i18n/t.js';
	import { onMount } from 'svelte';
	import PageHeader from '#lib/ui/PageHeader.svelte';
	import Card from '#lib/ui/Card.svelte';
	import Field from '#lib/ui/Field.svelte';
	import Select from '#lib/ui/Select.svelte';
	import SegmentedControl from '#lib/ui/SegmentedControl.svelte';
	import Alert from '#lib/ui/Alert.svelte';
	import SettingsNav from '#lib/components/SettingsNav.svelte';
	import {
		themeState,
		setTheme,
		type ThemeMode
	} from '#lib/i18n/theme.svelte.js';
	import {
		applyLocale,
		localeState,
		LOCALE_OPTIONS,
		type Locale
	} from '#lib/i18n/locale.svelte.js';

	let status: { kind: 'success' | 'error'; text: string } | null = $state(null);
	let loading = $state(true);

	const themeOptions = $derived([
		{ value: 'light', label: t('appearance.theme.light'), icon: 'sun' },
		{ value: 'dark', label: t('appearance.theme.dark'), icon: 'moon' },
		{ value: 'system', label: t('appearance.theme.system'), icon: 'monitor' }
	]);

	onMount(() => {
		loading = false;
	});

	function changeTheme(mode: string): void {
		setTheme(mode as ThemeMode);
	}

	async function changeLocale(next: string): Promise<void> {
		const locale = next as Locale;
		applyLocale(locale);
		status = null;
		try {
			const res = await fetch('/api/settings/appearance', {
				method: 'PUT',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ locale })
			});
			if (!res.ok) {
				status = { kind: 'error', text: t('appearance.error.save') };
				return;
			}
			status = { kind: 'success', text: t('appearance.saved') };
		} catch {
			// Preferência local já aplicada; backend indisponível é avisado.
			status = { kind: 'error', text: t('appearance.error.save_network') };
		}
	}
</script>

<svelte:head>
	<title>{t('appearance.title')} · {t('common.app_name')}</title>
</svelte:head>

<PageHeader
	title={t('appearance.title')}
	subtitle={t('appearance.subtitle')}
	breadcrumb={[
		{ label: t('common.home'), href: '/' },
		{ label: t('common.settings'), href: '/settings' },
		{ label: t('appearance.title') }
	]}
/>

<SettingsNav active="appearance" />

{#if status}
	<Alert severity={status.kind}>{status.text}</Alert>
{/if}

<div class="stack">
	<Card title={t('appearance.theme.label')} subtitle={t('appearance.theme.description')}>
		<SegmentedControl
			label={t('appearance.theme.label')}
			options={themeOptions}
			value={themeState.mode}
			onchange={changeTheme}
		/>
	</Card>

	<Card title={t('appearance.locale.label')} subtitle={t('appearance.locale.description')}>
		{#if !loading}
			<Field label={t('appearance.locale.label')} for="appearance-locale">
				<Select
					id="appearance-locale"
					value={localeState.current}
					options={LOCALE_OPTIONS.map((item) => ({ value: item.value, label: item.label }))}
					onchange={(event) => void changeLocale(event.currentTarget.value)}
				/>
			</Field>
		{/if}
	</Card>
</div>
