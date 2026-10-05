export const API_BASE = '/api/v1';

function getToken(): string | null {
	if (typeof window === 'undefined') return null;
	return localStorage.getItem('access_token');
}

async function request<T>(endpoint: string, options: RequestInit = {}): Promise<T> {
	const token = getToken();
	const headers: Record<string, string> = {
		'Content-Type': 'application/json',
		...(options.headers as Record<string, string>)
	};

	if (token) {
		headers['Authorization'] = `Bearer ${token}`;
	}

	const res = await fetch(`${API_BASE}${endpoint}`, {
		...options,
		headers
	});

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

	return res.json() as Promise<T>;
}

export const api = {
	auth: {
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
			})
	},
	users: {
		getMe: () => request<{ user: any; ratings: Record<string, number> }>('/users/me'),
		getUser: (username: string) => request<any>(`/users/${username}`)
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
		makeMove: (gameId: string, move: { from: string; to: string; promotion?: string }) =>
			request<any>(`/games/${gameId}/moves`, {
				method: 'POST',
				body: JSON.stringify(move)
			}),
		resign: (gameId: string) =>
			request<any>(`/games/${gameId}/resign`, {
				method: 'POST'
			}),
		review: (gameId: string) =>
			request<any>(`/games/${gameId}/review`, {
				method: 'POST'
			})
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
			})
	},
	analysis: {
		evaluate: (fen: string, depth = 12) =>
			request<{ fen: string; scoreCp: number; isMate: boolean; mateIn?: number; bestMove: string; pv: string[]; depth: number; winChance: number }>('/analysis/evaluate', {
				method: 'POST',
				body: JSON.stringify({ fen, depth })
			}),
		review: (moves: { from: string; to: string; san: string; fen?: string }[]) =>
			request<any>('/analysis/review', {
				method: 'POST',
				body: JSON.stringify({ moves })
			})
	},
	matchmaking: {
		join: (data: { initial: number; increment: number }) =>
			request<any>('/matchmaking/join', {
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
		getRandom: () => request<any>('/puzzles/random'),
		getById: (id: string) => request<any>(`/puzzles/${id}`)
	}
};
