<script lang="ts" generics="T">
	import { t } from '#lib/i18n/t.js';
	import type { Snippet } from 'svelte';
	import type { DataColumn } from './data.js';
	import { useViewport, viewport } from './media.svelte.js';
	import EmptyState from './EmptyState.svelte';
	import Icon from './Icon.svelte';
	import Spinner from './Spinner.svelte';

	interface SortState {
		key: string;
		dir: 'asc' | 'desc';
	}

	interface Props {
		columns: readonly DataColumn<T>[];
		rows: readonly T[];
		getKey: (row: T) => string | number;
		caption?: string;
		emptyTitle?: string;
		emptyDescription?: string;
		empty?: Snippet;
		onrowclick?: (row: T) => void;
		/** Renderizador único com acesso à coluna — evita declarar snippets por coluna. */
		render?: Snippet<[T, DataColumn<T>]>;
		/** Seleção em lote */
		selectable?: boolean;
		selectedIds?: readonly (string | number)[];
		onselect?: (id: string | number, checked: boolean) => void;
		onselectall?: (checked: boolean) => void;
		/** Ordenação (uma coluna por vez) */
		sort?: SortState;
		onsort?: (key: string) => void;
		/** Scroll interno na lista, em vez de rolar a página. */
		scrollable?: boolean;
		/** Infinite scroll (mobile/tablet): chamado ao chegar perto do fim. */
		onreachEnd?: () => void;
		loadingMore?: boolean;
	}

	let {
		columns,
		rows,
		getKey,
		caption,
		emptyTitle,
		emptyDescription,
		empty,
		onrowclick,
		render,
		selectable = false,
		selectedIds = [],
		onselect,
		onselectall,
		sort,
		onsort,
		scrollable = false,
		onreachEnd,
		loadingMore = false
	}: Props = $props();

	useViewport();

	const allSelected = $derived(rows.length > 0 && rows.every((row) => selectedIds.includes(getKey(row))));

	function onScroll(event: Event): void {
		const element = event.currentTarget as HTMLElement;
		if (element.scrollHeight - element.scrollTop - element.clientHeight < 80) onreachEnd?.();
	}
</script>

{#if rows.length === 0}
	{#if empty}
		{@render empty()}
	{:else}
		<EmptyState title={emptyTitle ?? t('ui.empty_title')} description={emptyDescription} />
	{/if}
{:else if viewport.isDesktop}
	<div class="table-wrap" class:scrollable onscroll={onreachEnd ? onScroll : undefined}>
		<table>
			<caption class="visually-hidden">{caption ?? t('ui.empty_records')}</caption>
			<thead>
				<tr>
					{#if selectable}
						<th scope="col" class="check-col">
							<input
								type="checkbox"
								aria-label={t('ui.select_all_rows')}
								checked={allSelected}
								onchange={(event) => onselectall?.(event.currentTarget.checked)}
							/>
						</th>
					{/if}
					{#each columns as column (column.key)}
						<th scope="col" class={column.class}>
							{#if onsort}
								<button
									type="button"
									class="sort-button"
									aria-label={t('ui.sort_by', { column: column.header })}
									onclick={(event) => { event.stopPropagation(); onsort(column.key); }}
								>
									{column.header}
									{#if sort?.key === column.key}
										<Icon name={sort.dir === 'asc' ? 'arrow-up' : 'arrow-down'} size={16} />
									{/if}
								</button>
							{:else}
								{column.header}
							{/if}
						</th>
					{/each}
				</tr>
			</thead>
			<tbody>
				{#each rows as row (getKey(row))}
					<tr class:clickable={Boolean(onrowclick)} class:selected={selectedIds.includes(getKey(row))} onclick={() => onrowclick?.(row)}>
						{#if selectable}
							<td class="check-col">
								<input
									type="checkbox"
									aria-label={t('ui.select_row', { id: getKey(row) })}
									checked={selectedIds.includes(getKey(row))}
									onclick={(event) => event.stopPropagation()}
									onchange={(event) => onselect?.(getKey(row), event.currentTarget.checked)}
								/>
							</td>
						{/if}
						{#each columns as column (column.key)}
							<td class={column.class}>
								{#if column.cell}
									{@render column.cell(row)}
								{:else if render}
									{@render render(row, column)}
								{:else}
									{column.value?.(row) ?? ''}
								{/if}
							</td>
						{/each}
					</tr>
				{/each}
			</tbody>
		</table>
	</div>
{:else}
	<div class="card-list" class:scrollable onscroll={onreachEnd ? onScroll : undefined}>
		{#each rows as row (getKey(row))}
			<!-- svelte-ignore a11y_no_noninteractive_tabindex -->
			<div
				class="record"
				class:clickable={Boolean(onrowclick)}
				class:selected={selectedIds.includes(getKey(row))}
				role={onrowclick ? 'link' : undefined}
				tabindex={onrowclick ? 0 : undefined}
				onclick={() => onrowclick?.(row)}
				onkeydown={(e) => {
					if (onrowclick && (e.key === 'Enter' || e.key === ' ')) {
						e.preventDefault();
						onrowclick(row);
					}
				}}
			>
				<dl>
					{#if selectable}
						<div class="field field-select">
							<dt class="visually-hidden">{t('ui.select_column')}</dt>
							<dd>
								<input
									type="checkbox"
									aria-label={t('ui.select_row', { id: getKey(row) })}
									checked={selectedIds.includes(getKey(row))}
									onclick={(event) => event.stopPropagation()}
									onchange={(event) => onselect?.(getKey(row), event.currentTarget.checked)}
								/>
							</dd>
						</div>
					{/if}
					{#each columns as column (column.key)}
						{#if !column.hideInCard}
							<div class="field field-{column.key}">
								<dt>
									{#if onsort}
										<button
											type="button"
											class="sort-button"
											onclick={(event) => { event.stopPropagation(); onsort(column.key); }}
										>
											{column.cardLabel ?? column.header}
											{#if sort?.key === column.key}
												<Icon name={sort.dir === 'asc' ? 'arrow-up' : 'arrow-down'} size={16} />
											{/if}
										</button>
									{:else}
										{column.cardLabel ?? column.header}
									{/if}
								</dt>
								<dd>
									{#if column.card}
										{@render column.card(row)}
									{:else if column.cell}
										{@render column.cell(row)}
									{:else if render}
										{@render render(row, column)}
									{:else}
										{column.value?.(row) ?? ''}
									{/if}
								</dd>
							</div>
						{/if}
					{/each}
				</dl>
			</div>
		{/each}
	</div>
{/if}

{#if loadingMore}
	<div class="list-loading" role="status">
		<Spinner size={16} />
		<span>{t('ui.loading_more')}</span>
	</div>
{/if}

<style>
	.table-wrap {
		overflow-x: auto;
		border: 1px solid var(--color-border);
		border-radius: var(--radius-md);
	}

	.table-wrap.scrollable,
	.card-list.scrollable {
		flex: 1 1 auto;
		min-height: 0;
		max-height: none;
		overflow-y: auto;
		-webkit-overflow-scrolling: touch;
	}

	table {
		width: 100%;
		border-collapse: collapse;
		font-size: var(--text-caption);
		text-align: left;
	}

	th {
		position: sticky;
		top: 0;
		z-index: 1;
		background: var(--color-surface-soft);
		color: var(--color-text-muted);
		font-size: 11px;
		font-weight: var(--weight-semibold);
		text-transform: uppercase;
		letter-spacing: 0.04em;
		white-space: nowrap;
	}

	th,
	td {
		padding: var(--space-4) var(--space-6);
		border-bottom: 1px solid var(--color-border);
		vertical-align: middle;
	}

	tbody tr:last-child td {
		border-bottom: 0;
	}

	tr.clickable,
	div.clickable {
		cursor: pointer;
	}

	tbody tr.clickable:hover,
	div.record.clickable:hover {
		background: var(--color-surface-soft);
	}

	tr.selected,
	div.record.selected {
		background: var(--color-info-soft);
	}

	.check-col {
		width: 48px;
		text-align: center;
	}

	.check-col input,
	.field-select input {
		width: 18px;
		height: 18px;
		accent-color: var(--color-primary);
		cursor: pointer;
	}

	.sort-button {
		display: inline-flex;
		align-items: center;
		gap: var(--space-1);
		border: 0;
		background: none;
		padding: 0;
		color: inherit;
		font: inherit;
		text-transform: inherit;
		letter-spacing: inherit;
		cursor: pointer;
	}

	.sort-button:hover {
		color: var(--color-primary);
	}

	.card-list {
		display: flex;
		flex-direction: column;
		gap: var(--space-4);
	}

	.record {
		background: var(--color-surface);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-md);
		padding: var(--space-6);
		box-shadow: var(--shadow-sm);
	}

	dl {
		display: flex;
		flex-direction: column;
		gap: var(--space-4);
		margin: 0;
	}

	.field {
		display: grid;
		grid-template-columns: minmax(96px, 35%) 1fr;
		gap: var(--space-4);
		align-items: start;
	}

	.field-select {
		grid-template-columns: 1fr;
	}

	.field-select dd {
		margin: 0;
	}

	dt {
		font-size: 11px;
		font-weight: var(--weight-semibold);
		text-transform: uppercase;
		letter-spacing: 0.04em;
		color: var(--color-text-muted);
	}

	dd {
		margin: 0;
		min-width: 0;
		overflow-wrap: anywhere;
	}

	/* Tablet: cards em duas colunas quando a largura permite. */
	@media (min-width: 768px) and (max-width: 1023px) {
		.card-list {
			display: grid;
			grid-template-columns: repeat(2, minmax(0, 1fr));
			align-items: start;
		}
	}

	.list-loading {
		display: flex;
		align-items: center;
		justify-content: center;
		gap: var(--space-2);
		padding: var(--space-3);
		font-size: var(--text-caption);
		color: var(--color-text-muted);
	}

	/* Phone: mais espaço útil no card e alvos de toque confortáveis. */
	@media (max-width: 767px) {
		.record {
			padding: var(--space-4);
		}

		.check-col input,
		.field-select input {
			width: 20px;
			height: 20px;
		}
	}
</style>
