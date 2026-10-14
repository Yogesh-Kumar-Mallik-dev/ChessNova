/**
 * apps/web/src/lib/utils/desktop.ts
 * Cross-platform desktop bridge for Tauri v2 native integrations.
 * Gracefully no-ops or falls back to Web APIs when running in standard browser.
 */

export function isTauri(): boolean {
	if (typeof window === 'undefined') return false;
	return '__TAURI_INTERNALS__' in window || '__TAURI__' in window;
}

export async function getTauriWindow() {
	if (!isTauri()) return null;
	try {
		const { getCurrentWindow } = await import('@tauri-apps/api/window');
		return getCurrentWindow();
	} catch (e) {
		console.warn('Failed to get Tauri window:', e);
		return null;
	}
}

export async function minimizeWindow(): Promise<void> {
	const win = await getTauriWindow();
	if (win) {
		await win.minimize();
	}
}

export async function toggleMaximizeWindow(): Promise<boolean> {
	const win = await getTauriWindow();
	if (win) {
		await win.toggleMaximize();
		return win.isMaximized();
	}
	return false;
}

export async function closeWindow(): Promise<void> {
	const win = await getTauriWindow();
	if (win) {
		await win.close();
	}
}

export async function isWindowMaximized(): Promise<boolean> {
	const win = await getTauriWindow();
	if (win) {
		return win.isMaximized();
	}
	return false;
}

/**
 * Sends a native system notification via Tauri or HTML5 Notification API.
 */
export async function sendDesktopNotification(title: string, body: string): Promise<void> {
	if (isTauri()) {
		try {
			const { isPermissionGranted, requestPermission, sendNotification } = await import(
				'@tauri-apps/plugin-notification'
			);
			let granted = await isPermissionGranted();
			if (!granted) {
				const perm = await requestPermission();
				granted = perm === 'granted';
			}
			if (granted) {
				sendNotification({ title, body });
				return;
			}
		} catch (err) {
			console.warn('Failed to send native Tauri notification:', err);
		}
	}

	// Browser Notification API fallback
	if (typeof window !== 'undefined' && 'Notification' in window) {
		if (Notification.permission === 'granted') {
			new Notification(title, { body, icon: '/favicon.png' });
		} else if (Notification.permission !== 'denied') {
			Notification.requestPermission().then((perm) => {
				if (perm === 'granted') {
					new Notification(title, { body, icon: '/favicon.png' });
				}
			});
		}
	}
}
