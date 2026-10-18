import { writable } from 'svelte/store';
import { soundEffects } from '$lib/audio/sounds';

export type BoardTheme = 'monochrome' | 'slate' | 'charcoal' | 'wood' | 'emerald' | 'ocean' | 'coral' | 'cyber';
export type AppTheme = 'dark' | 'light' | 'oled' | 'dark-slate' | 'chess-dark' | 'oled-black' | 'emerald-studio';

function createBoardThemeStore() {
	const initial: BoardTheme =
		typeof window !== 'undefined'
			? ((localStorage.getItem('chess_board_theme') as BoardTheme) || 'monochrome')
			: 'monochrome';

	const { subscribe, set } = writable<BoardTheme>(initial);

	return {
		subscribe,
		set: (theme: BoardTheme) => {
			if (typeof window !== 'undefined') {
				localStorage.setItem('chess_board_theme', theme);
				window.dispatchEvent(new CustomEvent('chess-theme-changed', { detail: theme }));
			}
			set(theme);
		}
	};
}

function createAppThemeStore() {
	const initial: AppTheme =
		typeof window !== 'undefined'
			? ((localStorage.getItem('chess_app_theme') as AppTheme) || 'dark')
			: 'dark';

	const { subscribe, set } = writable<AppTheme>(initial);

	if (typeof window !== 'undefined') {
		document.documentElement.setAttribute('data-app-theme', initial);
	}

	return {
		subscribe,
		set: (theme: AppTheme) => {
			if (typeof window !== 'undefined') {
				localStorage.setItem('chess_app_theme', theme);
				document.documentElement.setAttribute('data-app-theme', theme);
			}
			set(theme);
		}
	};
}

function createVolumeStore() {
	const initial = typeof window !== 'undefined' ? soundEffects.getVolume() : 0.85;
	const { subscribe, set } = writable<number>(initial);

	return {
		subscribe,
		set: (vol: number) => {
			const bounded = Math.max(0, Math.min(1, vol));
			soundEffects.setVolume(bounded);
			set(bounded);
		}
	};
}

function createSoundStore() {
	const initial = typeof window !== 'undefined' ? soundEffects.isEnabled() : true;
	const { subscribe, set } = writable<boolean>(initial);

	return {
		subscribe,
		set: (enabled: boolean) => {
			soundEffects.setEnabled(enabled);
			set(enabled);
		},
		toggle: () => {
			const next = soundEffects.toggleSound();
			set(next);
			return next;
		}
	};
}

export const themeStore = createBoardThemeStore();
export const appThemeStore = createAppThemeStore();
export const volumeStore = createVolumeStore();
export const soundStore = createSoundStore();
