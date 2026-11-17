/** Hierarquia única de navegação — usada pela sidebar e pela bottom nav. */
export interface NavItem {
	href: string;
	label: string;
	icon: string;
}

export const PRIMARY_NAV: readonly NavItem[] = [
	{ href: '/', label: 'Início', icon: 'home' },
	{ href: '/jobs', label: 'Vagas', icon: 'briefcase' },
	{ href: '/applications', label: 'Candidaturas', icon: 'clipboard' },
	{ href: '/curriculum', label: 'Currículo', icon: 'file' }
] as const;

/** Tarefas fica no grupo secundário da sidebar (Notificações saiu da sidebar). */
export const SECONDARY_NAV: readonly NavItem[] = [
	{ href: '/tasks', label: 'Tarefas', icon: 'sparkles' }
] as const;

/** Config compartilha o separador do antigo "Atalhos" no rodapé da sidebar. */
export const FOOTER_NAV: readonly NavItem[] = [
	{ href: '/settings', label: 'Config', icon: 'settings' }
] as const;

/** Bottom nav mantém Config como 5º destino (mesma hierarquia, agrupamento mobile). */
export const BOTTOM_NAV: readonly NavItem[] = [...PRIMARY_NAV, ...FOOTER_NAV] as const;

export function isActive(href: string, pathname: string): boolean {
	return href === '/' ? pathname === '/' : pathname.startsWith(href);
}
