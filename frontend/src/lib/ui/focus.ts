/**
 * Focus trap compartilhado por Dialog/Sheet/Modal/Drawer (CP12 §6.4).
 * - foca o primeiro elemento no mount
 * - Tab/Shift+Tab cicla dentro do overlay
 * - Escape fecha (listener de captura, antes dos atalhos globais)
 * - ao desmontar, devolve o foco ao disparador
 */
export function trapFocus(node: HTMLElement, onEscape: () => void) {
	const previous = document.activeElement as HTMLElement | null;

	const focusables = (): HTMLElement[] =>
		Array.from(
			node.querySelectorAll<HTMLElement>(
				'a[href], button:not([disabled]), input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])'
			)
		).filter((el) => el.getClientRects().length > 0);

	queueMicrotask(() => (focusables()[0] ?? node).focus());

	function onKeydown(event: KeyboardEvent): void {
		if (event.key === 'Escape') {
			event.preventDefault();
			event.stopPropagation();
			onEscape();
			return;
		}
		if (event.key !== 'Tab') return;
		const items = focusables();
		if (items.length === 0) {
			event.preventDefault();
			node.focus();
			return;
		}
		const first = items[0];
		const last = items[items.length - 1];
		const active = document.activeElement;
		if (event.shiftKey && (active === first || !node.contains(active))) {
			event.preventDefault();
			last.focus();
		} else if (!event.shiftKey && (active === last || !node.contains(active))) {
			event.preventDefault();
			first.focus();
		}
	}

	document.addEventListener('keydown', onKeydown, true);
	return {
		destroy() {
			document.removeEventListener('keydown', onKeydown, true);
			if (previous && previous.isConnected) previous.focus();
		}
	};
}
