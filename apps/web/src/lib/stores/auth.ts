import { writable } from 'svelte/store';
import { api } from '$lib/api/client';

export interface User {
	id: string;
	username: string;
	email: string;
	avatar?: string;
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
			if (!token) {
				update((s) => ({ ...s, loading: false }));
				return;
			}

			try {
				const res = await api.users.getMe();
				set({
					user: res.user,
					ratings: res.ratings || {},
					token,
					loading: false
				});
			} catch (err) {
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
