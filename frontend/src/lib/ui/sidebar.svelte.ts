const STORAGE_KEY = 'openjob:sidebar-collapsed';

/** Estado persistido do rail da sidebar (compartilhado por layout e Sidebar). */
class SidebarState {
	collapsed = $state(false);

	load(): void {
		if (typeof localStorage === 'undefined') return;
		this.collapsed = localStorage.getItem(STORAGE_KEY) === '1';
	}

	toggle(): void {
		this.collapsed = !this.collapsed;
		if (typeof localStorage !== 'undefined') {
			localStorage.setItem(STORAGE_KEY, this.collapsed ? '1' : '0');
		}
	}
}

export const sidebar = new SidebarState();
