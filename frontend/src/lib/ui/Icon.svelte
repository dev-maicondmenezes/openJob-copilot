<script lang="ts">
	import type { Component } from 'svelte';
	import {
		ArrowDown,
		ArrowLeft,
		ArrowUp,
		ArrowUpRight,
		CheckCheck,
		Bell,
		BriefcaseBusiness,
		CalendarClock,
		Check,
		ChevronRight,
		CircleAlert,
		CircleCheck,
		ClipboardList,
		Ellipsis,
		ExternalLink,
		Eye,
		EyeOff,
		FileText,
		House,
		Inbox,
		Info,
		KeyRound,
	Palette,
		Keyboard,
		LoaderCircle,
		Menu,
	Monitor,
	Moon,
		PanelLeftClose,
		PanelLeftOpen,
		Pencil,
		Plus,
		RefreshCw,
		Rss,
		Search,
		Settings,
		SlidersHorizontal,
	Sun,
		Sparkles,
		Trash2,
		TriangleAlert,
		X
	} from '@lucide/svelte';

	/** Conjunto único de ícones (lucide). Declarado aqui para permitir `name="..."`. */
	const icons: Record<string, Component<any>> = {
		'arrow-left': ArrowLeft,
		'arrow-up': ArrowUp,
		'arrow-down': ArrowDown,
		'arrow-up-right': ArrowUpRight,
		'check-double': CheckCheck,
		bell: Bell,
		briefcase: BriefcaseBusiness,
		calendar: CalendarClock,
		check: Check,
		'chevron-right': ChevronRight,
		'circle-alert': CircleAlert,
		'circle-check': CircleCheck,
		clipboard: ClipboardList,
		ellipsis: Ellipsis,
		external: ExternalLink,
		eye: Eye,
		'eye-off': EyeOff,
		file: FileText,
		home: House,
		inbox: Inbox,
		info: Info,
		key: KeyRound,
		keyboard: Keyboard,
		loader: LoaderCircle,
		menu: Menu,
		monitor: Monitor,
		moon: Moon,
		palette: Palette,
		sun: Sun,
		'panel-close': PanelLeftClose,
		'panel-open': PanelLeftOpen,
		pencil: Pencil,
		plus: Plus,
		refresh: RefreshCw,
		rss: Rss,
		search: Search,
		settings: Settings,
		sliders: SlidersHorizontal,
		sparkles: Sparkles,
		trash: Trash2,
		warning: TriangleAlert,
		x: X
	};

	interface Props {
		name: string;
		size?: 16 | 20 | 24;
		/** Texto alternativo. Quando ausente, o ícone é decorativo (aria-hidden). */
		label?: string;
		class?: string;
		strokeWidth?: number;
	}

	let {
		name,
		size = 20,
		label,
		class: className = '',
		strokeWidth = 1.8
	}: Props = $props();

	const Glyph = $derived(icons[name] ?? Info);
</script>

{#if label}
	<span class="icon {className}" role="img" aria-label={label}>
		<Glyph {size} {strokeWidth} aria-hidden="true" focusable="false" />
	</span>
{:else}
	<span class="icon {className}" aria-hidden="true">
		<Glyph {size} {strokeWidth} focusable="false" />
	</span>
{/if}

<style>
	.icon {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		flex-shrink: 0;
		line-height: 0;
	}
</style>
