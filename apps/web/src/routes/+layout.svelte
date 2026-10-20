<script lang="ts">
	import '../app.css';
	import { onMount } from 'svelte';
	import { authStore } from '$lib/stores/auth';
	import { appThemeStore } from '$lib/stores/preferences';
	import { soundEffects } from '$lib/audio/sounds';
	import Navbar from '$lib/components/ui/Navbar.svelte';
	import ChallengeNotificationBanner from '$lib/components/friends/ChallengeNotificationBanner.svelte';

	onMount(() => {
		authStore.init();
		const unsub = appThemeStore.subscribe((theme) => {
			if (typeof document !== 'undefined') {
				document.documentElement.setAttribute('data-app-theme', theme);
			}
		});

		const handleWake = () => {
			soundEffects.unlockAudio();
		};

		const handleVisibilityChange = () => {
			if (document.visibilityState === 'visible') {
				handleWake();
			}
		};

		if (typeof window !== 'undefined') {
			document.addEventListener('visibilitychange', handleVisibilityChange);
			window.addEventListener('focus', handleWake);
			window.addEventListener('online', handleWake);
		}

		return () => {
			unsub();
			if (typeof window !== 'undefined') {
				document.removeEventListener('visibilitychange', handleVisibilityChange);
				window.removeEventListener('focus', handleWake);
				window.removeEventListener('online', handleWake);
			}
		};
	});
</script>

<div class="min-h-screen flex flex-col bg-background text-foreground antialiased selection:bg-foreground selection:text-background relative overflow-x-hidden">
	<Navbar />
	<ChallengeNotificationBanner />
	<main class="flex-1 flex flex-col min-h-0">
		<slot />
	</main>
</div>
