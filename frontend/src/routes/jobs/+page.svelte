<script lang="ts">
	import { t } from '#lib/i18n/t.js';
	import JobFeed from '#lib/components/JobFeed.svelte';
</script>

<svelte:head>
	<title>{t('jobs.feed_title')} · {t('common.app_name')}</title>
</svelte:head>

<JobFeed />
