<script lang="ts">
	import { onMount } from 'svelte';
	import Icon from '$lib/components/icons/Icon.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import Badge from '$lib/components/ui/Badge.svelte';
	import Card from '$lib/components/ui/Card.svelte';

	const GITHUB_REPO = 'https://github.com/Yogesh-Kumar-Mallik-dev/ChessNova';
	const RELEASE_TAG = 'v0.1.0';
	const RELEASE_BASE = `${GITHUB_REPO}/releases/download/${RELEASE_TAG}`;
	const RELEASES_PAGE = `${GITHUB_REPO}/releases`;

	// Download artifacts map
	const downloads = {
		linux: {
			appimage: `${RELEASE_BASE}/chessnova_0.1.0_amd64.AppImage`,
			deb: `${RELEASE_BASE}/chessnova_0.1.0_amd64.deb`
		},
		windows: {
			exe: `${RELEASE_BASE}/ChessNova_0.1.0_x64-setup.exe`,
			msi: `${RELEASE_BASE}/ChessNova_0.1.0_x64_en-US.msi`
		},
		macos: {
			dmg: `${RELEASE_BASE}/ChessNova_0.1.0_universal.dmg`
		}
	};

	let detectedOs: 'linux' | 'windows' | 'macos' | 'other' = 'other';

	onMount(() => {
		if (typeof window !== 'undefined' && navigator) {
			const ua = navigator.userAgent.toLowerCase();
			if (ua.includes('win')) {
				detectedOs = 'windows';
			} else if (ua.includes('mac') || ua.includes('darwin')) {
				detectedOs = 'macos';
			} else if (ua.includes('linux') || ua.includes('x11')) {
				detectedOs = 'linux';
			}
		}
	});
</script>

<svelte:head>
	<title>Download ChessNova Desktop — Linux, Windows & macOS</title>
	<meta name="description" content="Download ChessNova desktop application for Linux, Windows, and macOS. Native performance, embedded Stockfish 19 NNUE, offline analysis, and Studio Noir design." />
</svelte:head>

<div class="max-w-6xl mx-auto px-4 sm:px-6 py-10 sm:py-16 w-full flex-1 flex flex-col justify-center">
	<!-- Hero Header -->
	<div class="text-center max-w-3xl mx-auto mb-12 sm:mb-16">
		<Badge variant="outline" className="mb-4 px-3 py-1 font-mono text-xs tracking-wider uppercase border-border text-muted-foreground">
			<span class="inline-block w-1.5 h-1.5 rounded-full bg-emerald-500 mr-2 animate-pulse"></span>
			<span>Official Release {RELEASE_TAG}</span>
		</Badge>

		<h1 class="text-3xl sm:text-5xl lg:text-6xl font-black text-foreground tracking-tight mb-4">
			ChessNova for Desktop
		</h1>

		<p class="text-base sm:text-lg text-muted-foreground leading-relaxed">
			The complete ChessNova platform packaged into an ultra-fast, native desktop application engineered with Rust and Tauri v2. Offline Stockfish 19 sparring, grandmaster tactical review coach, and frameless Studio Noir design.
		</p>
	</div>

	<!-- Operating Systems Grid -->
	<div class="grid grid-cols-1 md:grid-cols-3 gap-6 mb-14">
		<!-- Linux Card -->
		<Card className="p-6 sm:p-7 flex flex-col justify-between relative border-border hover:border-foreground/30 transition-all {detectedOs === 'linux' ? 'ring-2 ring-emerald-500/50 bg-secondary/30' : 'bg-card'}">
			{#if detectedOs === 'linux'}
				<div class="absolute -top-3 left-6">
					<span class="bg-emerald-500 text-black text-[10px] font-black uppercase tracking-wider px-2 py-0.5 rounded-full shadow-xs">
						Detected OS
					</span>
				</div>
			{/if}

			<div>
				<div class="flex items-center justify-between mb-4">
					<div class="w-12 h-12 rounded-xl bg-secondary flex items-center justify-center text-foreground font-bold text-xl">
						🐧
					</div>
					<span class="text-xs font-mono text-muted-foreground">x86_64</span>
				</div>

				<h2 class="text-xl font-bold text-foreground mb-1">Linux</h2>
				<p class="text-xs text-muted-foreground mb-6">
					Universal AppImage and Debian/Ubuntu packages. Runs seamlessly on Arch, Fedora, Ubuntu, Debian, and Linux Mint.
				</p>
			</div>

			<div class="space-y-2.5 pt-4 border-t border-border">
				<a href={downloads.linux.appimage} class="block">
					<Button variant={detectedOs === 'linux' ? 'primary' : 'secondary'} size="default" className="w-full justify-between">
						<span>Download .AppImage</span>
						<Icon name="download" size={15} />
					</Button>
				</a>
				<a href={downloads.linux.deb} class="block">
					<Button variant="outline" size="sm" className="w-full justify-between text-xs">
						<span>Download .deb Package</span>
						<Icon name="download" size={13} />
					</Button>
				</a>
			</div>
		</Card>

		<!-- Windows Card -->
		<Card className="p-6 sm:p-7 flex flex-col justify-between relative border-border hover:border-foreground/30 transition-all {detectedOs === 'windows' ? 'ring-2 ring-emerald-500/50 bg-secondary/30' : 'bg-card'}">
			{#if detectedOs === 'windows'}
				<div class="absolute -top-3 left-6">
					<span class="bg-emerald-500 text-black text-[10px] font-black uppercase tracking-wider px-2 py-0.5 rounded-full shadow-xs">
						Detected OS
					</span>
				</div>
			{/if}

			<div>
				<div class="flex items-center justify-between mb-4">
					<div class="w-12 h-12 rounded-xl bg-secondary flex items-center justify-center text-foreground font-bold text-xl">
						🪟
					</div>
					<span class="text-xs font-mono text-muted-foreground">x64 (64-bit)</span>
				</div>

				<h2 class="text-xl font-bold text-foreground mb-1">Windows</h2>
				<p class="text-xs text-muted-foreground mb-6">
					Installer executable (.exe) and enterprise MSI package for Windows 10 and Windows 11.
				</p>
			</div>

			<div class="space-y-2.5 pt-4 border-t border-border">
				<a href={downloads.windows.exe} class="block">
					<Button variant={detectedOs === 'windows' ? 'primary' : 'secondary'} size="default" className="w-full justify-between">
						<span>Download .exe Installer</span>
						<Icon name="download" size={15} />
					</Button>
				</a>
				<a href={downloads.windows.msi} class="block">
					<Button variant="outline" size="sm" className="w-full justify-between text-xs">
						<span>Download .msi Package</span>
						<Icon name="download" size={13} />
					</Button>
				</a>
			</div>
		</Card>

		<!-- macOS Card -->
		<Card className="p-6 sm:p-7 flex flex-col justify-between relative border-border hover:border-foreground/30 transition-all {detectedOs === 'macos' ? 'ring-2 ring-emerald-500/50 bg-secondary/30' : 'bg-card'}">
			{#if detectedOs === 'macos'}
				<div class="absolute -top-3 left-6">
					<span class="bg-emerald-500 text-black text-[10px] font-black uppercase tracking-wider px-2 py-0.5 rounded-full shadow-xs">
						Detected OS
					</span>
				</div>
			{/if}

			<div>
				<div class="flex items-center justify-between mb-4">
					<div class="w-12 h-12 rounded-xl bg-secondary flex items-center justify-center text-foreground font-bold text-xl">
						🍏
					</div>
					<span class="text-xs font-mono text-muted-foreground">Universal Binary</span>
				</div>

				<h2 class="text-xl font-bold text-foreground mb-1">macOS</h2>
				<p class="text-xs text-muted-foreground mb-6">
					Single Universal DMG disk image natively optimized for both Apple Silicon (M1/M2/M3/M4) and Intel Macs.
				</p>
			</div>

			<div class="space-y-2.5 pt-4 border-t border-border">
				<a href={downloads.macos.dmg} class="block">
					<Button variant={detectedOs === 'macos' ? 'primary' : 'secondary'} size="default" className="w-full justify-between">
						<span>Download .dmg</span>
						<Icon name="download" size={15} />
					</Button>
				</a>
				<div class="text-[11px] text-center text-muted-foreground pt-1">
					macOS 11 Big Sur or newer
				</div>
			</div>
		</Card>
	</div>

	<!-- GitHub Release Banner & Verification -->
	<Card className="p-6 sm:p-8 border-border bg-card/60 backdrop-blur-xs mb-14">
		<div class="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-6">
			<div>
				<div class="flex items-center gap-2 mb-2">
					<span class="text-base font-bold text-foreground">GitHub Release Repository</span>
					<Badge variant="outline" className="text-[10px] font-mono">Open Source</Badge>
				</div>
				<p class="text-sm text-muted-foreground max-w-xl">
					All release binaries are automatically compiled and published through GitHub Actions. You can inspect SHA-256 checksums, full changelogs, and release assets on GitHub.
				</p>
			</div>

			<a href={RELEASES_PAGE} target="_blank" rel="noopener noreferrer" class="shrink-0 w-full sm:w-auto">
				<Button variant="outline" size="default" className="w-full sm:w-auto gap-2">
					<span>View on GitHub</span>
					<Icon name="external-link" size={14} />
				</Button>
			</a>
		</div>
	</Card>

	<!-- Key Desktop Features Grid -->
	<div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-6">
		<div class="p-5 rounded-xl border border-border bg-card/40">
			<div class="w-8 h-8 rounded-lg bg-secondary flex items-center justify-center mb-3">
				<Icon name="zap" size={16} className="text-foreground" />
			</div>
			<h3 class="text-sm font-bold text-foreground mb-1">Stockfish 19 with NNUE</h3>
			<p class="text-xs text-muted-foreground leading-relaxed">
				Integrated neural network engine calculation at depth 14–16 for grandmaster tactical precision.
			</p>
		</div>

		<div class="p-5 rounded-xl border border-border bg-card/40">
			<div class="w-8 h-8 rounded-lg bg-secondary flex items-center justify-center mb-3">
				<Icon name="wifi-off" size={16} className="text-foreground" />
			</div>
			<h3 class="text-sm font-bold text-foreground mb-1">100% Offline Play</h3>
			<p class="text-xs text-muted-foreground leading-relaxed">
				Full game reviews, bot sparring, and pass-and-play matches function with zero internet connection.
			</p>
		</div>

		<div class="p-5 rounded-xl border border-border bg-card/40">
			<div class="w-8 h-8 rounded-lg bg-secondary flex items-center justify-center mb-3">
				<Icon name="sparkles" size={16} className="text-foreground" />
			</div>
			<h3 class="text-sm font-bold text-foreground mb-1">Frameless Studio Noir</h3>
			<p class="text-xs text-muted-foreground leading-relaxed">
				Custom desktop titlebar, high-contrast Staunton SVG vector pieces, and dynamic multi-theme engine.
			</p>
		</div>

		<div class="p-5 rounded-xl border border-border bg-card/40">
			<div class="w-8 h-8 rounded-lg bg-secondary flex items-center justify-center mb-3">
				<Icon name="shield" size={16} className="text-foreground" />
			</div>
			<h3 class="text-sm font-bold text-foreground mb-1">Under 60 MB RAM</h3>
			<p class="text-xs text-muted-foreground leading-relaxed">
				Native Tauri v2 + Rust architecture replaces heavy Electron runtimes with ultra-light native webviews.
			</p>
		</div>
	</div>
</div>
