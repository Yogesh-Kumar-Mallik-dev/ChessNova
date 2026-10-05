// ==============================================================================
// ChessNova Authentic Studio Audio Player
// Uses authentic, studio-recorded acoustic chess sound effects (clean, pleasant wood impacts)
// Pre-cached audio pool with zero-latency playback across all browsers
// ==============================================================================

export type MoveCategory = 'normal' | 'capture' | 'castle' | 'en_passant' | 'promote' | 'check' | 'checkmate';

class ChessSoundPlayer {
	private enabled = true;
	private audioCache = new Map<string, HTMLAudioElement[]>();

	constructor() {
		if (typeof window !== 'undefined') {
			this.preload('move', '/sounds/move.mp3');
			this.preload('capture', '/sounds/capture.mp3');
			this.preload('castle', '/sounds/castle.mp3');
			this.preload('check', '/sounds/check.mp3');
			this.preload('promote', '/sounds/promote.mp3');
			this.preload('en-passant', '/sounds/en-passant.mp3');
			this.preload('game-end', '/sounds/game-end.mp3');
			this.preload('game-start', '/sounds/game-start.mp3');
		}
	}

	private preload(key: string, url: string, poolSize = 4) {
		const pool: HTMLAudioElement[] = [];
		for (let i = 0; i < poolSize; i++) {
			const a = new Audio(url);
			a.preload = 'auto';
			pool.push(a);
		}
		this.audioCache.set(key, pool);
	}

	private play(key: string, volume = 0.85) {
		if (!this.enabled || typeof window === 'undefined') return;
		const pool = this.audioCache.get(key);
		if (!pool || pool.length === 0) {
			const a = new Audio(`/sounds/${key}.mp3`);
			a.volume = volume;
			a.play().catch(() => {});
			return;
		}

		// Find an idle instance in the pool or cycle the first one
		const audio = pool.find((a) => a.paused || a.ended) || pool[0];
		audio.currentTime = 0;
		audio.volume = volume;
		audio.play().catch(() => {});
	}

	public playNormalMove() {
		this.play('move', 0.85);
	}

	public playCapture() {
		this.play('capture', 0.9);
	}

	public playCastle() {
		this.play('castle', 0.85);
	}

	public playEnPassant() {
		this.play('en-passant', 0.95);
	}

	public playCheck() {
		this.play('check', 0.9);
	}

	public playCheckmate() {
		this.play('game-end', 1.0);
	}

	public playPromotion() {
		this.play('promote', 0.9);
	}

	public playGameStart() {
		this.play('game-start', 0.8);
	}

	public playGameEnd(won: boolean = true) {
		this.play('game-end', 0.9);
	}

	public toggleSound(): boolean {
		this.enabled = !this.enabled;
		return this.enabled;
	}

	public isEnabled(): boolean {
		return this.enabled;
	}

	public setEnabled(val: boolean) {
		this.enabled = val;
	}
}

export const soundEffects = new ChessSoundPlayer();

/**
 * Universal move sound dispatcher based on categorized move type
 */
export function playMoveSoundByCategory(category: MoveCategory) {
	switch (category) {
		case 'checkmate':
			soundEffects.playCheckmate();
			break;
		case 'promote':
			soundEffects.playPromotion();
			break;
		case 'check':
			soundEffects.playCheck();
			break;
		case 'en_passant':
			soundEffects.playEnPassant();
			break;
		case 'castle':
			soundEffects.playCastle();
			break;
		case 'capture':
			soundEffects.playCapture();
			break;
		case 'normal':
		default:
			soundEffects.playNormalMove();
			break;
	}
}

/**
 * Fallback compatibility helper
 */
export function playMoveSound(san: string, isCapture = false, isCheck = false) {
	if (san.includes('#')) {
		soundEffects.playCheckmate();
	} else if (san.includes('=')) {
		soundEffects.playPromotion();
	} else if (isCheck || san.includes('+')) {
		soundEffects.playCheck();
	} else if (san.startsWith('O-O')) {
		soundEffects.playCastle();
	} else if (isCapture || san.includes('x')) {
		soundEffects.playCapture();
	} else {
		soundEffects.playNormalMove();
	}
}
