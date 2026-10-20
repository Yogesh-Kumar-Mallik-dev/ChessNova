// apps/web/src/lib/config.ts
// Centralized endpoint configuration for Web and Tauri Native Desktop

export const PRODUCTION_DOMAIN = 'chess.yogeshmallik.in';

export function isTauriEnvironment(): boolean {
	return (
		typeof window !== 'undefined' &&
		('__TAURI_INTERNALS__' in window ||
			'__TAURI__' in window ||
			window.location.protocol === 'tauri:' ||
			window.location.hostname === 'tauri.localhost')
	);
}

export function isNativeOrAppEnvironment(): boolean {
	if (typeof window === 'undefined') return false;
	if (isTauriEnvironment()) return true;
	if (window.matchMedia?.('(display-mode: standalone)').matches) return true;
	if ((window.navigator as any)?.standalone === true) return true;
	return false;
}

export function getApiBase(): string {
	if (typeof window === 'undefined') return '/api/v1';

	// In Vite dev mode (http://localhost:5173), use relative URL so Vite proxy forwards to localhost:8080
	if (window.location.hostname === 'localhost' && window.location.port === '5173') {
		return '/api/v1';
	}

	// In native desktop production app (tauri://localhost or http://tauri.localhost),
	// route all requests to the production cloud backend
	if (isTauriEnvironment()) {
		const custom = localStorage.getItem('chesspasant_api_base');
		if (custom) return custom.replace(/\/$/, '');
		return `https://${PRODUCTION_DOMAIN}/api/v1`;
	}

	// Standard web client (relative to current domain)
	return '/api/v1';
}

export function getWsBase(): string {
	if (typeof window === 'undefined') return '';

	// In Vite dev mode
	if (window.location.hostname === 'localhost' && window.location.port === '5173') {
		return `ws://${window.location.host}/ws`;
	}

	// In native desktop production app
	if (isTauriEnvironment()) {
		const custom = localStorage.getItem('chesspasant_ws_base');
		if (custom) return custom.replace(/\/$/, '');
		return `wss://${PRODUCTION_DOMAIN}/ws`;
	}

	// Standard web client
	const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
	return `${protocol}//${window.location.host}/ws`;
}

export function resolveMediaUrl(url: string | null | undefined): string {
	if (!url) return '';
	if (url.startsWith('http://') || url.startsWith('https://') || url.startsWith('data:') || url.startsWith('blob:')) {
		return url;
	}
	if (isTauriEnvironment()) {
		const base = `https://${PRODUCTION_DOMAIN}`;
		return url.startsWith('/') ? `${base}${url}` : `${base}/${url}`;
	}
	return url;
}

