/** Hierarquia única de navegação — usada pela sidebar e pela bottom nav. */
import { t } from '#lib/i18n/t.js';

export interface NavItem {
	href: string;
	label: string;
	icon: string;
}

/** Funções (não constantes) para que os labels sejam reativos ao locale. */
export function primaryNav(): readonly NavItem[] {
	return [
		{ href: '/', label: t('common.home'), icon: 'home' },
		{ href: '/jobs', label: t('common.jobs'), icon: 'briefcase' },
		{ href: '/applications', label: t('common.applications'), icon: 'clipboard' },
		{ href: '/curriculum', label: t('common.curriculum'), icon: 'file' }
	];
}

/** Tarefas fica no grupo secundário da sidebar (Notificações saiu da sidebar). */
export function secondaryNav(): readonly NavItem[] {
	return [{ href: '/tasks', label: t('common.tasks'), icon: 'sparkles' }];
}

/** Bottom nav: apenas os 4 destinos primários. Configurações ficam no menu
 * do cabeçalho (`SettingsMenu`), o mesmo padrão em desktop e mobile. */
export function bottomNav(): readonly NavItem[] {
	return primaryNav();
}

export function isActive(href: string, pathname: string): boolean {
	return href === '/' ? pathname === '/' : pathname.startsWith(href);
}
