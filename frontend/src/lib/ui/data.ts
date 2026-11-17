import type { Snippet } from 'svelte';

/** Definição de coluna compartilhada entre a tabela (desktop) e o card-list. */
export interface DataColumn<T> {
	key: string;
	header: string;
	/** Texto simples; usado como fallback quando não há snippet. */
	value?: (row: T) => string | number | null | undefined;
	/** Conteúdo rico na tabela. */
	cell?: Snippet<[T]>;
	/** Conteúdo rico no card. */
	card?: Snippet<[T]>;
	/** Rótulo do campo no card (padrão: `header`). */
	cardLabel?: string;
	/** Omite o campo no card (ex.: ações já presentes no rodapé). */
	hideInCard?: boolean;
	class?: string;
}
