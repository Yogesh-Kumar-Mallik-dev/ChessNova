import { gameStore } from '$lib/stores/game';
import { soundEffects } from '$lib/audio/sounds';

export class GameSocket {
	private ws: WebSocket | null = null;
	private gameId: string;
	private token: string | null;
	private reconnectTimer: any = null;
	private pingTimer: any = null;
	private isExplicitlyClosed = false;

	constructor(gameId: string, token: string | null) {
		this.gameId = gameId;
		this.token = token;
	}

	public connect() {
		this.isExplicitlyClosed = false;
		const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
		const host = window.location.host;
		const tokenParam = this.token ? `?token=${encodeURIComponent(this.token)}` : '';
		const url = `${protocol}//${host}/ws/game/${this.gameId}${tokenParam}`;

		gameStore.update((s) => ({ ...s, connectionStatus: 'connecting' }));

		this.ws = new WebSocket(url);

		this.ws.onopen = () => {
			gameStore.update((s) => ({ ...s, connectionStatus: 'connected' }));
			this.startPing();
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
				this.reconnectTimer = setTimeout(() => this.connect(), 2000);
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
					isCheck: data.isCheck
				}));
				break;

			case 'move':
				gameStore.update((s) => {
					const newMoves = [...s.moves];
					if (data.san) {
						newMoves.push({
							from: data.move.from,
							to: data.move.to,
							san: data.san,
							fen: data.fen
						});
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
						lastMove: data.move
							? {
									from: data.move.from,
									to: data.move.to,
									san: data.san,
									isCapture: data.san?.includes('x'),
									isPromotion: data.san?.includes('=') || !!data.move.promotion
								}
							: null,
						moves: newMoves,
						selectedSquare: null,
						legalMoves: []
					};
				});
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
				gameStore.update((s) => ({
					...s,
					status: data.status,
					outcome: data.outcome,
					result: data.result,
					winner: data.winner || s.winner
				}));
				break;

			case 'draw_offer':
				gameStore.update((s) => ({
					...s,
					drawOfferBy: data.offeredBy
				}));
				break;

			case 'draw_accepted':
				gameStore.update((s) => ({
					...s,
					status: data.status,
					outcome: data.outcome,
					result: data.result,
					drawOfferBy: null
				}));
				break;

			case 'draw_declined':
				gameStore.update((s) => ({
					...s,
					drawOfferBy: null
				}));
				break;

			case 'error':
				console.warn('Game error from server:', data.error);
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
			this.ws.send(JSON.stringify(msg));
		}
	}

	private startPing() {
		this.pingTimer = setInterval(() => {
			this.send({ type: 'ping' });
		}, 5000);
	}

	private stopPing() {
		if (this.pingTimer) clearInterval(this.pingTimer);
	}

	public disconnect() {
		this.isExplicitlyClosed = true;
		this.stopPing();
		if (this.reconnectTimer) clearTimeout(this.reconnectTimer);
		if (this.ws) {
			this.ws.close();
			this.ws = null;
		}
	}
}
