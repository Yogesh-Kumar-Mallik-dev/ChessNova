import { gameStore } from '$lib/stores/game';
import { authStore } from '$lib/stores/auth';
import { soundEffects } from '$lib/audio/sounds';
import { sendDesktopNotification } from '$lib/utils/desktop';

function getCategory(initialSeconds: number): string {
	const mins = initialSeconds / 60;
	if (mins < 3) return 'bullet';
	if (mins < 10) return 'blitz';
	if (mins < 30) return 'rapid';
	return 'classical';
}

function toSquareNotation(val: any): string {
	if (typeof val === 'string' && val.length === 2 && val[0] >= 'a' && val[0] <= 'h' && val[1] >= '1' && val[1] <= '8') {
		return val;
	}
	if (typeof val === 'number' && val >= 0 && val < 64) {
		const file = String.fromCharCode('a'.charCodeAt(0) + (val % 8));
		const rank = String(Math.floor(val / 8) + 1);
		return `${file}${rank}`;
	}
	return String(val || '');
}

export class GameSocket {
	private ws: WebSocket | null = null;
	private gameId: string;
	private token: string | null;
	private reconnectTimer: any = null;
	private pingTimer: any = null;
	private isExplicitlyClosed = false;
	private errorCallback: ((err: string) => void) | null = null;
	private messageQueue: any[] = [];

	constructor(gameId: string, token: string | null) {
		this.gameId = gameId;
		this.token = token;
	}

	public isConnected(): boolean {
		return !!(this.ws && this.ws.readyState === WebSocket.OPEN);
	}

	public onError(cb: (err: string) => void) {
		this.errorCallback = cb;
	}

	public connect() {
		this.isExplicitlyClosed = false;
		if (typeof window === 'undefined') return;

		const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
		const host = window.location.host;
		const tokenParam = this.token ? `?token=${encodeURIComponent(this.token)}` : '';
		const url = `${protocol}//${host}/ws/game/${this.gameId}${tokenParam}`;

		gameStore.update((s) => ({ ...s, connectionStatus: 'connecting' }));

		this.ws = new WebSocket(url);

		this.ws.onopen = () => {
			gameStore.update((s) => ({ ...s, connectionStatus: 'connected' }));
			this.startPing();

			// Flush buffered outbound messages upon connection
			while (this.messageQueue.length > 0) {
				const msg = this.messageQueue.shift();
				try {
					this.ws?.send(JSON.stringify(msg));
				} catch (e) {
					console.error('Failed to flush message to socket:', e);
				}
			}
		};

		this.ws.onmessage = (event) => {
			try {
				const data = JSON.parse(event.data);
				this.handleMessage(data);
			} catch (e) {
				console.error('Failed to parse websocket message:', e);
			}
		};

		this.ws.onclose = () => {
			this.stopPing();
			gameStore.update((s) => ({ ...s, connectionStatus: 'disconnected' }));
			if (!this.isExplicitlyClosed) {
				this.reconnectTimer = setTimeout(() => this.connect(), 1500);
			}
		};

		this.ws.onerror = (err) => {
			console.error('WebSocket error:', err);
		};
	}

	private handleMessage(data: any) {
		switch (data.type) {
			case 'game_started':
				soundEffects.playGameStart();
				sendDesktopNotification('Game Started!', 'Your chess match is ready. Good luck!');
				gameStore.update((s) => ({
					...s,
					gameId: data.gameId,
					fen: data.fen,
					turn: data.turn,
					whiteTime: data.whiteTime,
					blackTime: data.blackTime,
					status: data.status,
					whitePlayer: data.white,
					blackPlayer: data.black,
					timeControl: data.timeControl,
					isCheck: data.isCheck,
					drawOfferBy: null
				}));
				break;

			case 'move': {
				const moveFrom = toSquareNotation(data.move?.from);
				const moveTo = toSquareNotation(data.move?.to);

				gameStore.update((s) => {
					const newMoves = [...s.moves];
					const lastM = newMoves.length > 0 ? newMoves[newMoves.length - 1] : null;
					const isOptimisticMatch = lastM && lastM.from === moveFrom && lastM.to === moveTo;

					if (data.san) {
						if (isOptimisticMatch) {
							newMoves[newMoves.length - 1] = {
								from: moveFrom,
								to: moveTo,
								san: data.san,
								fen: data.fen
							};
						} else {
							newMoves.push({
								from: moveFrom,
								to: moveTo,
								san: data.san,
								fen: data.fen
							});
						}
					}

					// Sound effect for move event (skip if already played locally during optimistic update)
					if (data.status === 'playing' && !isOptimisticMatch) {
						if (data.isCheck) {
							soundEffects.playCheck();
						} else if (data.san?.includes('x')) {
							soundEffects.playCapture();
						} else if (data.san?.startsWith('O-O')) {
							soundEffects.playCastle();
						} else if (data.san?.includes('=')) {
							soundEffects.playPromotion();
						} else {
							soundEffects.playNormalMove();
						}
					}

					return {
						...s,
						fen: data.fen,
						turn: data.turn,
						whiteTime: data.whiteTime,
						blackTime: data.blackTime,
						status: data.status,
						outcome: data.outcome,
						result: data.result,
						winner: data.winner,
						isCheck: data.isCheck,
						drawOfferBy: null, // Any subsequent move implicitly declines an outstanding draw offer
						lastMove: moveFrom && moveTo
							? {
									from: moveFrom,
									to: moveTo,
									san: data.san,
									isCapture: data.san?.includes('x'),
									isPromotion: data.san?.includes('=') || !!data.move?.promotion
								}
							: null,
						moves: newMoves,
						selectedSquare: null,
						legalMoves: []
					};
				});
				break;
			}

			case 'player_disconnected':
				gameStore.update((s) => ({
					...s,
					opponentConnected: false,
					opponentDisconnectTime: Date.now()
				}));
				break;

			case 'player_connected':
				gameStore.update((s) => ({
					...s,
					opponentConnected: true,
					opponentDisconnectTime: null
				}));
				break;

			case 'clock_update':
				gameStore.update((s) => ({
					...s,
					whiteTime: data.whiteTime,
					blackTime: data.blackTime
				}));
				break;

			case 'resignation':
			case 'game_finished':
			case 'draw_accepted':
				soundEffects.playGameEnd();
				sendDesktopNotification('Game Over', `Match finished: ${data.outcome || data.result || 'Game ended'}.`);
				gameStore.update((s) => {
					const wDiff = data.whiteDiff !== undefined ? data.whiteDiff : s.whiteDiff;
					const bDiff = data.blackDiff !== undefined ? data.blackDiff : s.blackDiff;

					// Sync authStore ratings immediately if the current player participated
					if (s.timeControl?.initial) {
						const cat = getCategory(s.timeControl.initial);
						let authState: any = null;
						const unsub = authStore.subscribe((val) => (authState = val));
						unsub();
						if (authState?.user) {
							const uid = authState.user.id;
							const uname = authState.user.username;
							const isWhite = (s.whitePlayer?.userId && s.whitePlayer.userId === uid) || (uname && s.whitePlayer?.username?.toLowerCase() === uname.toLowerCase());
							const isBlack = (s.blackPlayer?.userId && s.blackPlayer.userId === uid) || (uname && s.blackPlayer?.username?.toLowerCase() === uname.toLowerCase());
							const diff = isWhite ? wDiff : isBlack ? bDiff : null;
							if (diff !== null && diff !== undefined && authState.ratings?.[cat] !== undefined) {
								authStore.updateRating(cat, Math.max(100, authState.ratings[cat] + diff));
							}
						}
					}

					return {
						...s,
						status: data.status || 'finished',
						outcome: data.outcome || s.outcome,
						result: data.result || s.result,
						winner: data.winner || s.winner,
						whiteDiff: wDiff,
						blackDiff: bDiff,
						drawOfferBy: null
					};
				});
				break;

			case 'draw_offer':
				sendDesktopNotification('Draw Offer', 'Your opponent offered a draw.');
				gameStore.update((s) => ({
					...s,
					drawOfferBy: data.offeredBy
				}));
				break;

			case 'draw_declined':
				sendDesktopNotification('Draw Declined', 'Opponent declined the draw offer.');
				gameStore.update((s) => ({
					...s,
					drawOfferBy: null
				}));
				break;

			case 'error':
				console.warn('Game error from server:', data.error);
				if (this.errorCallback) {
					this.errorCallback(data.error);
				}
				break;
		}
	}

	public sendMove(from: string, to: string, promotion?: string) {
		this.send({
			type: 'move',
			from,
			to,
			promotion
		});
	}

	public resign() {
		this.send({ type: 'resign' });
	}

	public offerDraw() {
		this.send({ type: 'draw_offer' });
	}

	public acceptDraw() {
		this.send({ type: 'draw_accept' });
	}

	public declineDraw() {
		this.send({ type: 'draw_decline' });
	}

	private send(msg: any) {
		if (this.ws && this.ws.readyState === WebSocket.OPEN) {
			try {
				this.ws.send(JSON.stringify(msg));
			} catch (e) {
				console.warn('Failed to send on open socket, queueing:', e);
				this.messageQueue.push(msg);
			}
		} else {
			this.messageQueue.push(msg);
			if (!this.ws || this.ws.readyState === WebSocket.CLOSED) {
				this.connect();
			}
		}
	}

	private startPing() {
		this.pingTimer = setInterval(() => {
			this.send({ type: 'ping' });
		}, 5000);
	}

	private stopPing() {
		if (this.pingTimer) {
			clearInterval(this.pingTimer);
			this.pingTimer = null;
		}
	}

	public disconnect() {
		this.isExplicitlyClosed = true;
		this.stopPing();
		if (this.reconnectTimer) {
			clearTimeout(this.reconnectTimer);
			this.reconnectTimer = null;
		}
		if (this.ws) {
			this.ws.close();
			this.ws = null;
		}
		this.messageQueue = [];
	}
}
