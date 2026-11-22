<script lang="ts">
	import { t } from '#lib/i18n/t.js';
	import JobFeed from '#lib/components/JobFeed.svelte';
</script>

<svelte:head>
	<title>{t('common.app_name')}</title>
	<meta name="description" content={t('home.meta_description')} />
</svelte:head>

<JobFeed />
