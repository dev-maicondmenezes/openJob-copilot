/** Hierarquia única de navegação — usada pela sidebar e pela bottom nav. */
import { t } from '#lib/i18n/t.js';

export interface NavItem {
	href: string;
	label: string;
	icon: string;
}

/** Funções (não constantes) para que os labels sejam reativos ao locale.
 * CP16: Feed (/jobs) é a home principal; o dashboard do CP15 vive em /dashboard. */
export function primaryNav(): readonly NavItem[] {
	return [
		{ href: '/jobs', label: t('common.jobs'), icon: 'briefcase' },
		{ href: '/applications', label: t('common.applications'), icon: 'clipboard' },
		{ href: '/curriculum', label: t('common.curriculum'), icon: 'file' },
		{ href: '/dashboard', label: t('home.dashboard'), icon: 'home' }
	];
}

/** Tarefas fica no grupo secundário da sidebar (Notificações saiu da sidebar). */
export function secondaryNav(): readonly NavItem[] {
	return [{ href: '/tasks', label: t('common.tasks'), icon: 'sparkles' }];
}

/** Bottom nav: os 4 destinos primários (Feed, Candidaturas, Currículo, Dashboard). */
export function bottomNav(): readonly NavItem[] {
	return primaryNav();
}

export function isActive(href: string, pathname: string): boolean {
	if (href === '/jobs') {
		return pathname === '/' || pathname === '/jobs';
	}
	return href === '/' ? pathname === '/' : pathname.startsWith(href);
}
