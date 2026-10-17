import { writable } from 'svelte/store';
import { api } from '$lib/api/client';

export interface User {
	id: string;
	username: string;
	email: string;
	avatar?: string;
	bio?: string;
	country?: string;
	chessComUsername?: string;
	lichessUsername?: string;
}

interface AuthState {
	user: User | null;
	ratings: Record<string, number>;
	token: string | null;
	loading: boolean;
}

const initialState: AuthState = {
	user: null,
	ratings: {},
	token: typeof window !== 'undefined' ? localStorage.getItem('access_token') : null,
	loading: true
};

function createAuthStore() {
	const { subscribe, set, update } = writable<AuthState>(initialState);

	return {
		subscribe,
		init: async () => {
			if (typeof window === 'undefined') return;
			const token = localStorage.getItem('access_token');
			const refreshToken = localStorage.getItem('refresh_token');
			if (!token && !refreshToken) {
				update((s) => ({ ...s, loading: false }));
				return;
			}

			try {
				const res = await api.users.getMe();
				set({
					user: res.user,
					ratings: res.ratings || {},
					token: localStorage.getItem('access_token') || token,
					loading: false
				});
			} catch (err) {
				if (refreshToken) {
					try {
						const refreshed = await api.auth.refresh(refreshToken);
						const newAccess = (refreshed as any).accessToken;
						const newRefresh = (refreshed as any).refreshToken;
						if (newAccess) {
							localStorage.setItem('access_token', newAccess);
							if (newRefresh) localStorage.setItem('refresh_token', newRefresh);
							const res = await api.users.getMe();
							set({
								user: res.user,
								ratings: res.ratings || {},
								token: newAccess,
								loading: false
							});
							return;
						}
					} catch (_) {
						// refresh failed as well
					}
				}
				localStorage.removeItem('access_token');
				localStorage.removeItem('refresh_token');
				set({ user: null, ratings: {}, token: null, loading: false });
			}
		},
		loginSuccess: (user: User, tokens: { accessToken: string; refreshToken: string }, ratings: Record<string, number> = {}) => {
			if (typeof window !== 'undefined') {
				localStorage.setItem('access_token', tokens.accessToken);
				localStorage.setItem('refresh_token', tokens.refreshToken);
			}
			set({
				user,
				ratings,
				token: tokens.accessToken,
				loading: false
			});
		},
		updateUser: (fields: Partial<User>) => {
			update((state) => ({
				...state,
				user: state.user ? { ...state.user, ...fields } : null
			}));
		},
		updateRating: (category: string, newRating: number) => {
			update((state) => {
				const ratings = { ...state.ratings, [category]: newRating };
				if (category === 'puzzles') ratings['puzzle'] = newRating;
				if (category === 'puzzle') ratings['puzzles'] = newRating;
				return { ...state, ratings };
			});
		},
		logout: () => {
			if (typeof window !== 'undefined') {
				localStorage.removeItem('access_token');
				localStorage.removeItem('refresh_token');
			}
			set({
				user: null,
				ratings: {},
				token: null,
				loading: false
			});
		}
	};
}

export const authStore = createAuthStore();
