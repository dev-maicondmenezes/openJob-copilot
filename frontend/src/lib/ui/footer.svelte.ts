import type { Snippet } from 'svelte';

/**
 * Slot de rodapé do shell. Páginas que precisam de uma barra de ação/admin
 * (ex.: /jobs) registram seu conteúdo aqui; o layout renderiza o <footer>
 * depois do <main>, como 4ª área do layout.
 */
class FooterState {
	content: Snippet | null = $state(null);

	set(snippet: Snippet | null): void {
		this.content = snippet;
	}
}

export const pageFooter = new FooterState();
