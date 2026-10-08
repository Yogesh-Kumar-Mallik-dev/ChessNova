import { writable } from 'svelte/store';
import { soundEffects } from '$lib/audio/sounds';

export type BoardTheme = 'cyber' | 'wood' | 'emerald' | 'ocean';

function createThemeStore() {
	const initial: BoardTheme =
		typeof window !== 'undefined'
			? ((localStorage.getItem('chess_board_theme') as BoardTheme) || 'cyber')
			: 'cyber';

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

export const themeStore = createThemeStore();
export const volumeStore = createVolumeStore();
export const soundStore = createSoundStore();
