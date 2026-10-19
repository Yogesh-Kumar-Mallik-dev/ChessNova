import { getApiBase } from '$lib/config';

export const API_BASE = '/api/v1';

function getToken(): string | null {
	if (typeof window === 'undefined') return null;
	return localStorage.getItem('access_token');
}

let isRefreshing = false;
let refreshPromise: Promise<string | null> | null = null;

async function refreshAccessToken(): Promise<string | null> {
	if (typeof window === 'undefined') return null;
	const refreshToken = localStorage.getItem('refresh_token');
	if (!refreshToken) return null;

	if (isRefreshing && refreshPromise) {
		return refreshPromise;
	}

	isRefreshing = true;
	refreshPromise = (async () => {
		try {
			const res = await fetch(`${getApiBase()}/auth/refresh`, {
				method: 'POST',
				credentials: 'include',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ refreshToken })
			});
			if (!res.ok) {
				localStorage.removeItem('access_token');
				localStorage.removeItem('refresh_token');
				return null;
			}
			const data = await res.json();
			const newAccess = data.accessToken || data.tokens?.accessToken;
			const newRefresh = data.refreshToken || data.tokens?.refreshToken;
			if (newAccess) {
				localStorage.setItem('access_token', newAccess);
				if (newRefresh) localStorage.setItem('refresh_token', newRefresh);
				return newAccess;
			}
			return null;
		} catch (e) {
			console.error('Failed to refresh token:', e);
			return null;
		} finally {
			isRefreshing = false;
			refreshPromise = null;
		}
	})();

	return refreshPromise;
}

async function request<T>(endpoint: string, options: RequestInit = {}): Promise<T> {
	let token = getToken();
	const headers: Record<string, string> = {
		'Content-Type': 'application/json',
		...(options.headers as Record<string, string>)
	};

	if (token) {
		headers['Authorization'] = `Bearer ${token}`;
	}

	let res = await fetch(`${getApiBase()}${endpoint}`, {
		...options,
		credentials: 'include',
		headers
	});

	// If unauthorized (401), automatically refresh the access token and retry once
	if (res.status === 401 && !endpoint.startsWith('/auth/')) {
		const newAccess = await refreshAccessToken();
		if (newAccess) {
			headers['Authorization'] = `Bearer ${newAccess}`;
			res = await fetch(`${getApiBase()}${endpoint}`, {
				...options,
				credentials: 'include',
				headers
			});
		}
	}

	if (!res.ok) {
		let errMsg = `Request failed: ${res.statusText}`;
		try {
			const data = await res.json();
			if (data.error && data.error.message) {
				errMsg = data.error.message;
			}
		} catch {
			// ignore
		}
		throw new Error(errMsg);
	}

	return (await res.json()) as Promise<T>;
}

export const api = {
	auth: {
		sendRegisterOtp: (data: { username: string; email: string; password: string }) =>
			request<{ message: string; email: string; expiresIn: number }>('/auth/register/send-otp', {
				method: 'POST',
				body: JSON.stringify(data)
			}),
		verifyRegisterOtp: (data: { email: string; otp: string }) =>
			request<{ user: any; tokens: { accessToken: string; refreshToken: string } }>('/auth/register/verify-otp', {
				method: 'POST',
				body: JSON.stringify(data)
			}),
		resendRegisterOtp: (data: { email: string }) =>
			request<{ message: string; email: string; expiresIn: number }>('/auth/register/resend-otp', {
				method: 'POST',
				body: JSON.stringify(data)
			}),
		register: (data: { username: string; email: string; password: string }) =>
			request<{ user: any; tokens: { accessToken: string; refreshToken: string } }>('/auth/register', {
				method: 'POST',
				body: JSON.stringify(data)
			}),
		login: (data: { username: string; password: string }) =>
			request<{ user: any; tokens: { accessToken: string; refreshToken: string } }>('/auth/login', {
				method: 'POST',
				body: JSON.stringify(data)
			}),
		refresh: (refreshToken: string) =>
			request<{ accessToken: string; refreshToken: string }>('/auth/refresh', {
				method: 'POST',
				body: JSON.stringify({ refreshToken })
			}),
		logout: async () => {
			const accessToken = typeof window !== 'undefined' ? localStorage.getItem('access_token') : null;
			const refreshToken = typeof window !== 'undefined' ? localStorage.getItem('refresh_token') : null;
			try {
				if (accessToken || refreshToken) {
					await fetch(`${getApiBase()}/auth/logout`, {
						method: 'POST',
						credentials: 'include',
						headers: {
							'Content-Type': 'application/json',
							...(accessToken ? { Authorization: `Bearer ${accessToken}` } : {})
						},
						body: JSON.stringify({ refreshToken })
					});
				}
			} catch (e) {
				console.error('Failed to notify server of logout:', e);
			} finally {
				if (typeof window !== 'undefined') {
					localStorage.removeItem('access_token');
					localStorage.removeItem('refresh_token');
				}
			}
		},
		getWSTicket: () =>
			request<{ ticket: string; expiresIn: number }>('/ws/ticket', {
				method: 'POST'
			})
	},
	users: {
		getMe: () => request<{ user: any; ratings: Record<string, number> }>('/users/me'),
		getUser: (username: string) => request<any>(`/users/${username}`),
		updateProfile: (data: { bio: string; country: string }) =>
			request<{ message: string }>('/users/profile', {
				method: 'PUT',
				body: JSON.stringify(data)
			}),
		uploadAvatar: async (file: File) => {
			const token = getToken();
			const formData = new FormData();
			formData.append('avatar', file);
			const headers: Record<string, string> = {};
			if (token) headers['Authorization'] = `Bearer ${token}`;

			const res = await fetch(`${getApiBase()}/users/avatar`, {
				method: 'POST',
				credentials: 'include',
				headers,
				body: formData
			});
			if (!res.ok) {
				let errMsg = 'Failed to upload avatar';
				try {
					const data = await res.json();
					if (data.error?.message) errMsg = data.error.message;
				} catch {}
				throw new Error(errMsg);
			}
			return (await res.json()) as { avatar: string };
		},
		linkAccounts: (data: { chessComUsername: string; lichessUsername: string }) =>
			request<{ message: string; chessComUsername: string; lichessUsername: string }>('/users/link-accounts', {
				method: 'POST',
				body: JSON.stringify(data)
			}),
		getExternalStats: (username: string) => request<any>(`/users/${username}/external-stats`),
		getExternalGames: (username: string, platform = 'all', limit = 10) =>
			request<any[]>(`/users/${username}/external-games?platform=${platform}&limit=${limit}`)
	},
	games: {
		list: (params?: { userId?: string; limit?: number }) => {
			const query = new URLSearchParams();
			if (params?.userId) query.set('userId', params.userId);
			if (params?.limit) query.set('limit', params.limit.toString());
			return request<any[]>(`/games?${query.toString()}`);
		},
		create: (data: { initial: number; increment: number; opponentId?: string }) =>
			request<any>('/games', {
				method: 'POST',
				body: JSON.stringify(data)
			}),
		get: (id: string) => request<any>(`/games/${id}`),
		getActive: () => request<any>('/games/active'),
		makeMove: (gameId: string, move: { from: string; to: string; promotion?: string }) =>
			request<any>(`/games/${gameId}/moves`, {
				method: 'POST',
				body: JSON.stringify(move)
			}),
		resign: (gameId: string) =>
			request<any>(`/games/${gameId}/resign`, {
				method: 'POST'
			}),
		offerDraw: (gameId: string) =>
			request<any>(`/games/${gameId}/draw-offer`, {
				method: 'POST'
			}),
		acceptDraw: (gameId: string) =>
			request<any>(`/games/${gameId}/draw-accept`, {
				method: 'POST'
			}),
		declineDraw: (gameId: string) =>
			request<any>(`/games/${gameId}/draw-decline`, {
				method: 'POST'
			}),
		join: (gameId: string) =>
			request<any>(`/games/${gameId}/join`, {
				method: 'POST'
			}),
		review: (gameId: string) =>
			request<any>(`/games/${gameId}/review`, {
				method: 'POST'
			}),
		getPgn: async (gameId: string) => {
			const res = await fetch(`${getApiBase()}/games/${gameId}/pgn`);
			if (!res.ok) throw new Error('Failed to fetch game PGN');
			return res.text();
		}
	},
	chess: {
		legalMoves: (fen?: string, square?: string) =>
			request<{ fen: string; targets: { to: string; san: string; isCapture: boolean; category: string }[] }>('/chess/legal-moves', {
				method: 'POST',
				body: JSON.stringify({ fen, square })
			}),
		move: (fen: string, from: string, to: string, promotion?: string) =>
			request<{ valid: boolean; newFen: string; san: string; isCapture?: boolean; isPromotion?: boolean; isCheck: boolean; isCheckmate: boolean; isStalemate: boolean; isDraw: boolean; turn: string }>('/chess/move', {
				method: 'POST',
				body: JSON.stringify({ fen, from, to, promotion })
			}),
		botMove: (fen: string, level = 2) =>
			request<{ valid: boolean; uci: string; from: string; to: string; promotion?: string; san: string; newFen: string; isCapture?: boolean; isPromotion?: boolean; isCheck: boolean; isCheckmate: boolean; isStalemate: boolean; isDraw: boolean; turn: string }>('/chess/bot-move', {
				method: 'POST',
				body: JSON.stringify({ fen, level })
			})
	},
	analysis: {
		evaluate: (fen: string, depth = 12) =>
			request<{ fen: string; scoreCp: number; isMate: boolean; mateIn?: number; bestMove: string; pv: string[]; depth: number; winChance: number }>('/analysis/evaluate', {
				method: 'POST',
				body: JSON.stringify({ fen, depth })
			}),
		review: (
			payload:
				| string
				| { pgn?: string; moves?: { from: string; to: string; san: string; fen?: string }[] }
				| { from: string; to: string; san: string; fen?: string }[]
		) => {
			let body: Record<string, any>;
			if (typeof payload === 'string') {
				body = { pgn: payload };
			} else if (Array.isArray(payload)) {
				body = { moves: payload };
			} else {
				body = payload;
			}
			return request<any>('/analysis/review', {
				method: 'POST',
				body: JSON.stringify(body)
			});
		}
	},
	matchmaking: {
		join: (data: { initial: number; increment: number }) =>
			request<any>(`/matchmaking/join`, {
				method: 'POST',
				body: JSON.stringify(data)
			}),
		leave: () =>
			request<any>('/matchmaking/leave', {
				method: 'POST'
			}),
		status: () => request<any>('/matchmaking/status')
	},
	leaderboard: {
		get: (category: string = 'blitz') => request<any[]>(`/leaderboard?category=${category}`)
	},
	puzzles: {
		getRandom: (params?: { category?: string; tier?: number; minRating?: number; maxRating?: number }) => {
			const query = new URLSearchParams();
			if (params?.category && params.category !== 'all') query.set('category', params.category);
			if (params?.tier) query.set('tier', params.tier.toString());
			if (params?.minRating) query.set('minRating', params.minRating.toString());
			if (params?.maxRating) query.set('maxRating', params.maxRating.toString());
			const qs = query.toString();
			return request<any>(`/puzzles/random${qs ? `?${qs}` : ''}`);
		},
		getDaily: () => request<any>('/puzzles/daily'),
		getWeekly: () => request<any[]>('/puzzles/weekly'),
		getById: (id: string) => request<any>(`/puzzles/${id}`),
		getHint: (id: string, plyIndex: number = 0) =>
			request<{ plyIndex: number; from: string; to: string; hint: string }>(`/puzzles/${id}/hint?plyIndex=${plyIndex}`),
		getSolution: (id: string) =>
			request<{ id: string; solution: string[]; rating: number; themes: string[] }>(`/puzzles/${id}/solution`),
		solve: (
			id: string,
			payload:
				| { move?: string; plyIndex?: number; success?: boolean; isRetry?: boolean; hasAssistance?: boolean }
				| boolean
		) => {
			const body = typeof payload === 'boolean' ? { success: payload } : payload;
			return request<{ correct: boolean; finished: boolean; replyMove?: string; newRating: number; diff: number; solution?: string[] }>(`/puzzles/${id}/solve`, {
				method: 'POST',
				body: JSON.stringify(body)
			});
		}
	}
};
