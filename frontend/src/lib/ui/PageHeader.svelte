<script lang="ts">
	import { t } from '#lib/i18n/t.js';
	import type { Snippet } from 'svelte';
	import Icon from './Icon.svelte';

	interface Crumb {
		label: string;
		href?: string;
	}

	interface Props {
		title: string;
		subtitle?: string;
		breadcrumb?: Crumb[];
		actions?: Snippet;
	}

	let { title, subtitle, breadcrumb = [], actions }: Props = $props();
</script>

<header class="page-header">
	{#if breadcrumb.length > 0}
		<nav class="breadcrumb" aria-label={t('nav.breadcrumb')}>
			{#each breadcrumb as crumb, i (crumb.label)}
				{#if i > 0}
					<Icon name="chevron-right" size={16} />
				{/if}
				{#if crumb.href}
					<a href={crumb.href}>{crumb.label}</a>
				{:else}
					<span aria-current="page">{crumb.label}</span>
				{/if}
			{/each}
		</nav>
	{/if}

	<div class="heading-row">
		<div class="heading">
			<h1>{title}</h1>
			{#if subtitle}<p class="subtitle">{subtitle}</p>{/if}
		</div>
		{#if actions}<div class="actions">{@render actions()}</div>{/if}
	</div>
</header>

<style>
	.page-header {
		margin-bottom: var(--space-6);
	}

	.breadcrumb {
		display: none;
		align-items: center;
		gap: var(--space-2);
		margin-bottom: var(--space-3);
		font-size: var(--text-caption);
		line-height: var(--leading-caption);
		color: var(--color-text-muted);
	}

	.breadcrumb a {
		color: var(--color-text-muted);
		text-decoration: none;
	}

	.breadcrumb a:hover {
		color: var(--color-primary);
		text-decoration: underline;
	}

	.breadcrumb [aria-current='page'] {
		color: var(--color-text-primary);
		font-weight: var(--weight-semibold);
	}

	.heading-row {
		display: flex;
		flex-direction: column;
		gap: var(--space-3);
	}

	.heading {
		min-width: 0;
	}

	h1 {
		margin: 0;
	}

	.subtitle {
		margin: var(--space-1) 0 0;
		font-size: var(--text-body);
		line-height: var(--leading-body);
		color: var(--color-text-muted);
	}

	.actions {
		display: flex;
		align-items: center;
		gap: var(--space-2);
		flex-wrap: nowrap;
		white-space: nowrap;
		flex-shrink: 0;
	}

	@media (min-width: 768px) {
		.heading-row {
			flex-direction: row;
			align-items: flex-start;
			justify-content: space-between;
		}

		.actions {
			justify-content: flex-end;
		}
	}

	@media (min-width: 1024px) {
		.breadcrumb {
			display: flex;
		}
	}
</style>
