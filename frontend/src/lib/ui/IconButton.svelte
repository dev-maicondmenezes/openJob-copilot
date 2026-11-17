<script lang="ts">
	import type { HTMLButtonAttributes } from 'svelte/elements';
	import Button from './Button.svelte';
	import Icon from './Icon.svelte';

	interface Props extends HTMLButtonAttributes {
		/** Nome do ícone no conjunto lucide centralizado. */
		icon: string;
		/** Obrigatório: nome acessível do botão só-ícone (usado como tooltip). */
		label: string;
		variant?: 'primary' | 'secondary' | 'ghost' | 'danger' | 'link';
		size?: 'sm' | 'md' | 'lg';
		/** Tamanho do glifo — 24 em cabeçalhos mobile. */
		iconSize?: 20 | 24;
		loading?: boolean;
	}

	let {
		icon,
		label,
		variant = 'ghost',
		size = 'md',
		iconSize = 20,
		loading = false,
		...rest
	}: Props = $props();
</script>

<Button
	{...rest}
	{variant}
	{size}
	{loading}
	iconOnly
	aria-label={label}
	title={label}
>
	<Icon name={icon} size={iconSize} />
</Button>
