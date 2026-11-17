<script lang="ts">
	interface Option {
		value: string;
		label: string;
	}

	interface Props {
		label: string;
		value?: string;
		options: readonly Option[];
		placeholder?: string;
		emptyLabel?: string;
		onSelect?: (value: string) => void;
	}

	let {
		label,
		value = $bindable(''),
		options,
		placeholder = 'Buscar…',
		emptyLabel = 'Nenhuma opção',
		onSelect
	}: Props = $props();

	let query = $state('');
	let open = $state(false);

	const filtered = $derived(
		options.filter((option) => option.label.toLocaleLowerCase('pt-BR').includes(query.toLocaleLowerCase('pt-BR')))
	);
	const selectedLabel = $derived(options.find((option) => option.value === value)?.label ?? '');

	function choose(option: Option): void {
		value = option.value;
		query = '';
		open = false;
		onSelect?.(option.value);
	}
</script>

<div class="combobox">
	<input
		type="text"
		role="combobox"
		aria-label={label}
		aria-expanded={open}
		aria-controls="combobox-list"
		autocomplete="off"
		{placeholder}
		bind:value={query}
		onfocus={() => (open = true)}
		oninput={() => (open = true)}
		onkeydown={(e) => {
			if (e.key === 'Escape') open = false;
			if (e.key === 'Enter' && filtered[0]) {
				e.preventDefault();
				choose(filtered[0]);
			}
		}}
	/>
	{#if value && !open}
		<span class="selection" aria-live="polite">Selecionado: {selectedLabel}</span>
	{/if}
	{#if open}
		<ul id="combobox-list" class="options" role="listbox" aria-label={label}>
			{#each filtered as option (option.value)}
				<li role="option" aria-selected={value === option.value}>
					<button type="button" onclick={() => choose(option)}>{option.label}</button>
				</li>
			{:else}
				<li class="empty">{emptyLabel}</li>
			{/each}
		</ul>
	{/if}
</div>

<style>
	.combobox {
		position: relative;
	}

	input {
		width: 100%;
		min-height: var(--touch-target);
		padding: var(--space-2) var(--space-3);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-md);
		background: var(--color-surface);
		color: var(--color-text-primary);
	}

	input:focus {
		outline: 2px solid var(--color-focus);
		outline-offset: 1px;
	}

	.selection {
		display: block;
		margin-top: var(--space-1);
		font-size: var(--text-caption);
		color: var(--color-text-muted);
	}

	.options {
		position: absolute;
		top: calc(100% + var(--space-1));
		left: 0;
		right: 0;
		z-index: 20;
		max-height: 240px;
		overflow-y: auto;
		margin: 0;
		padding: var(--space-1);
		list-style: none;
		background: var(--color-surface);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-md);
		box-shadow: var(--shadow-md);
	}

	.options button {
		width: 100%;
		min-height: var(--touch-target);
		padding: var(--space-2) var(--space-3);
		text-align: left;
		background: none;
		border: 0;
		border-radius: var(--radius-sm);
		color: var(--color-text-primary);
		cursor: pointer;
	}

	.options button:hover {
		background: var(--color-info-soft);
	}

	.empty {
		padding: var(--space-3);
		color: var(--color-text-muted);
		font-size: var(--text-caption);
	}
</style>
