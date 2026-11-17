/**
 * Espelho TypeScript dos tokens CSS (adr 0013).
 * Use estes valores em JS/TS — nunca literals de breakpoint ou spacing.
 */

export const BREAKPOINTS = {
	sm: 640,
	md: 768,
	lg: 1024,
	xl: 1280
} as const;

export type Breakpoint = keyof typeof BREAKPOINTS;

/** Consultas de mídia canônicas. `< md` = phone, `md..<lg` = tablet, `>= lg` = desktop. */
export const MEDIA = {
	sm: `(min-width: ${BREAKPOINTS.sm}px)`,
	md: `(min-width: ${BREAKPOINTS.md}px)`,
	lg: `(min-width: ${BREAKPOINTS.lg}px)`,
	xl: `(min-width: ${BREAKPOINTS.xl}px)`,
	tablet: `(min-width: ${BREAKPOINTS.md}px) and (max-width: ${BREAKPOINTS.lg - 1}px)`,
	desktop: `(min-width: ${BREAKPOINTS.lg}px)`,
	reducedMotion: '(prefers-reduced-motion: reduce)'
} as const;

/** Escala de espaçamento base 4px (mesma ordem dos custom properties). */
export const SPACING = [2, 4, 8, 12, 16, 24, 32, 48, 64] as const;

export const RADIUS = {
	sm: 6,
	md: 10,
	lg: 16,
	pill: 999
} as const;

export const MOTION = {
	micro: 120,
	transition: 200,
	route: 320
} as const;

export const LAYOUT = {
	contentMaxTablet: 1080,
	contentMaxDesktop: 1920,
	sidebarWidth: 240,
	sidebarRailWidth: 72,
	bottomNavHeight: 60,
	touchTarget: 48
} as const;

/** Classificação canônica de dispositivo usada pelo shell e pelo DataView. */
export type DeviceClass = 'phone' | 'tablet' | 'desktop';

export function classifyWidth(width: number): DeviceClass {
	if (width >= BREAKPOINTS.lg) return 'desktop';
	if (width >= BREAKPOINTS.md) return 'tablet';
	return 'phone';
}
