<script lang="ts">
	import '../app.css';
	import { onMount } from 'svelte';
	import { authStore } from '$lib/stores/auth';
	import Navbar from '$lib/components/ui/Navbar.svelte';

	onMount(() => {
		authStore.init();
	});
</script>

<div class="min-h-screen flex flex-col bg-[#161512] text-[#bababa]">
	<Navbar />
	<main class="flex-1 flex flex-col">
		<slot />
	</main>
</div>
