<script lang="ts">
	import '../app.css';
	import { onMount } from 'svelte';
	import { authStore } from '$lib/stores/auth';
	import { appThemeStore } from '$lib/stores/preferences';
	import Navbar from '$lib/components/ui/Navbar.svelte';

	onMount(() => {
		authStore.init();
		const unsub = appThemeStore.subscribe((theme) => {
			if (typeof document !== 'undefined') {
				document.documentElement.setAttribute('data-app-theme', theme);
			}
		});
		return () => unsub();
	});
</script>

<div class="min-h-screen flex flex-col bg-background text-foreground antialiased selection:bg-emerald-500/20 selection:text-emerald-300 relative overflow-x-hidden">
	<Navbar />
	<main class="flex-1 flex flex-col">
		<slot />
	</main>
</div>
