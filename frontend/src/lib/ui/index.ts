/**
 * Biblioteca de design system do openJob (CP12).
 * Ponto único de importação: use `#lib/ui` em vez de caminhos internos.
 */

// Ação
export { default as Button } from './Button.svelte';
export { default as IconButton } from './IconButton.svelte';
export { default as LinkButton } from './LinkButton.svelte';
export { default as IconLinkButton } from './IconLinkButton.svelte';

// Formulário
export { default as Field } from './Field.svelte';
export { default as Input } from './Input.svelte';
export { default as Textarea } from './Textarea.svelte';
export { default as Select } from './Select.svelte';
export { default as MultiSelect } from './MultiSelect.svelte';
export { default as Combobox } from './Combobox.svelte';
export { default as Checkbox } from './Checkbox.svelte';
export { default as RadioGroup } from './RadioGroup.svelte';
export { default as Switch } from './Switch.svelte';

// Feedback
export { default as Alert } from './Alert.svelte';
export { default as Badge } from './Badge.svelte';
export { default as Toast } from './Toast.svelte';
export { default as EmptyState } from './EmptyState.svelte';
export { default as Spinner } from './Spinner.svelte';
export { default as Skeleton } from './Skeleton.svelte';
export { default as ProgressBar } from './ProgressBar.svelte';

// Estrutura
export { default as Card } from './Card.svelte';
export { default as Panel } from './Panel.svelte';
export { default as PageHeader } from './PageHeader.svelte';
export { default as Section } from './Section.svelte';
export { default as Tabs } from './Tabs.svelte';
export { default as SegmentedControl } from './SegmentedControl.svelte';

// Navegação
export { default as Icon } from './Icon.svelte';
export { default as BottomNav } from './BottomNav.svelte';
export { default as Sidebar } from './Sidebar.svelte';
export { default as Menu } from './Menu.svelte';
export { primaryNav, secondaryNav, bottomNav, isActive } from './nav.js';
export type { NavItem } from './nav.js';

// Overlay
export { default as Dialog } from './Dialog.svelte';
export { default as Modal } from './Modal.svelte';
export { default as Sheet } from './Sheet.svelte';
export { default as Drawer } from './Drawer.svelte';
export { trapFocus } from './focus.js';

// Dados e paginação
export { default as DataView } from './DataView.svelte';
export { default as Pagination } from './Pagination.svelte';
export type { DataColumn } from './data.js';

// Shell
export { default as PageFooter } from './PageFooter.svelte';
export { default as ShortcutsHelp } from './ShortcutsHelp.svelte';
export { viewport, useViewport, matches, breakpoints } from './media.svelte.js';
export { sidebar } from './sidebar.svelte.js';
export { pageFooter } from './footer.svelte.js';

// Tokens tipados
export {
	BREAKPOINTS,
	MEDIA,
	SPACING,
	RADIUS,
	MOTION,
	LAYOUT,
	classifyWidth
} from '../styles/tokens.js';
export type { Breakpoint, DeviceClass } from '../styles/tokens.js';
