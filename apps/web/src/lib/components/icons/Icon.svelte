<script lang="ts">
	export let name:
		| 'swords'
		| 'zap'
		| 'flame'
		| 'timer'
		| 'clock'
		| 'hourglass'
		| 'trophy'
		| 'shield'
		| 'rotate-cw'
		| 'rotate-ccw'
		| 'puzzle'
		| 'search'
		| 'user'
		| 'award'
		| 'chevron-left'
		| 'chevron-right'
		| 'chevrons-left'
		| 'chevrons-right'
		| 'check'
		| 'x'
		| 'flag'
		| 'handshake'
		| 'play'
		| 'sparkles'
		| 'crown'
		| 'bishop'
		| 'menu'
		| 'arrow-right'
		| 'log-out'
		| 'settings'
		| 'volume-2'
		| 'volume-x'
		| 'eye'
		| 'external-link'
		| 'camera'
		| 'upload'
		| 'edit'
		| 'wifi-off'
		| 'copy'
		| 'brand-arch'
		| 'brand-linux'
		| 'brand-windows'
		| 'brand-apple'
		| 'brand-debian'
		| 'download' = 'sparkles';
	export let size = 18;
	export let className = '';
</script>

<svg
	xmlns="http://www.w3.org/2000/svg"
	width={size}
	height={size}
	viewBox="0 0 24 24"
	fill="none"
	stroke="currentColor"
	stroke-width="2"
	stroke-linecap="round"
	stroke-linejoin="round"
	class="inline-block shrink-0 {className}"
	style="width: {size}px; height: {size}px; min-width: {size}px; min-height: {size}px;"
>
	{#if name === 'crown'}
		<path d="M11.562 3.266a.5.5 0 0 1 .876 0L15.39 8.87a1 1 0 0 0 1.516.294L21.183 5.5a.5.5 0 0 1 .798.519l-2.834 10.77a2 2 0 0 1-1.936 1.491H6.789a2 2 0 0 1-1.936-1.49L2.019 6.02a.5.5 0 0 1 .798-.519l4.276 3.664a1 1 0 0 0 1.516-.294z" />
		<path d="M5 21h14" />
	{:else if name === 'bishop'}
		<circle cx="12" cy="4" r="1.5" />
		<path d="M7 17.5c0-3 1.5-6 2.5-8.5a3.2 3.2 0 0 1 5 0c1 2.5 2.5 5.5 2.5 8.5" />
		<path d="m10 10.5 4 3.5" />
		<path d="M7 17.5l-1 3.5h12l-1-3.5Z" />
	{:else if name === 'menu'}
		<line x1="4" x2="20" y1="6" y2="6" />
		<line x1="4" x2="20" y1="12" y2="12" />
		<line x1="4" x2="20" y1="18" y2="18" />
	{:else if name === 'sparkles'}
		<path d="m12 3-1.912 5.813a2 2 0 0 1-1.275 1.275L3 12l5.813 1.912a2 2 0 0 1 1.275 1.275L12 21l1.912-5.813a2 2 0 0 1 1.275-1.275L21 12l-5.813-1.912a2 2 0 0 1-1.275-1.275z" />
		<path d="M5 3v4" /><path d="M19 17v4" /><path d="M3 5h4" /><path d="M17 19h4" />
	{:else if name === 'swords'}
		<polyline points="14.5 17.5 3 6 3 3 6 3 17.5 14.5" />
		<line x1="13" y1="19" x2="19" y2="13" />
		<line x1="16" y1="16" x2="20" y2="20" />
		<line x1="19" y1="21" x2="21" y2="19" />
		<polyline points="14.5 6.5 18 3 21 3 21 6 17.5 9.5" />
		<line x1="5" y1="14" x2="9" y2="18" />
		<line x1="7" y1="17" x2="4" y2="20" />
		<line x1="3" y1="19" x2="5" y2="21" />
	{:else if name === 'zap'}
		<polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2" />
	{:else if name === 'flame'}
		<path d="M8.5 14.5A2.5 2.5 0 0 0 11 12c0-1.38-.5-2-1-3-1.072-2.143-.224-4.054 2-6 .5 2.5 2 4.9 4 6.5 2 1.6 3 3.5 3 5.5a7 7 0 1 1-14 0c0-1.153.433-2.294 1-3a2.5 2.5 0 0 0 2.5 3.5z" />
	{:else if name === 'timer'}
		<line x1="10" y1="2" x2="14" y2="2" />
		<line x1="12" y1="14" x2="15" y2="11" />
		<circle cx="12" cy="14" r="8" />
	{:else if name === 'clock'}
		<circle cx="12" cy="12" r="10" />
		<polyline points="12 6 12 12 16 14" />
	{:else if name === 'hourglass'}
		<path d="M5 22h14" /><path d="M5 2h14" />
		<path d="M17 22v-4.172a2 2 0 0 0-.586-1.414L12 12l-4.414 4.414A2 2 0 0 0 7 17.828V22" />
		<path d="M7 2v4.172a2 2 0 0 0 .586 1.414L12 12l4.414-4.414A2 2 0 0 0 17 6.172V2" />
	{:else if name === 'trophy'}
		<path d="M6 9H4.5a2.5 2.5 0 0 1 0-5H6" />
		<path d="M18 9h1.5a2.5 2.5 0 0 0 0-5H18" />
		<path d="M4 22h16" />
		<path d="M10 14.66V17c0 .55-.45 1-1 1H7c-.55 0-1-.45-1-1v-2.34" />
		<path d="M14 14.66V17c0 .55.45 1 1 1h2c.55 0 1-.45 1-1v-2.34" />
		<path d="M18 2H6v7a6 6 0 0 0 12 0V2Z" />
	{:else if name === 'shield'}
		<path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z" />
	{:else if name === 'rotate-cw'}
		<path d="M21 12a9 9 0 1 1-9-9c2.52 0 4.93 1 6.74 2.74L21 8" />
		<polyline points="21 3 21 8 16 8" />
	{:else if name === 'rotate-ccw'}
		<path d="M3 12a9 9 0 1 0 9-9 9.75 9.75 0 0 0-6.74 2.74L3 8" />
		<polyline points="3 3 3 8 8 8" />
	{:else if name === 'puzzle'}
		<path d="M19.439 7.85c-.049-.322-.059-.659-.059-1a4 4 0 0 0-4-4c-1.396 0-2.617.712-3.327 1.792A4.004 4.004 0 0 0 8 3a4 4 0 0 0-4 4c0 .341.01.678.059 1A4.004 4.004 0 0 0 3 11a4 4 0 0 0 4 4c.341 0 .678-.01 1-.059A4.004 4.004 0 0 0 11 17a4 4 0 0 0 4-4c.341 0 .678-.01 1-.059A4.004 4.004 0 0 0 19 11a4 4 0 0 0-4-4c0-.341.01-.678.059-1A4.004 4.004 0 0 0 19.439 7.85z" />
	{:else if name === 'search'}
		<circle cx="11" cy="11" r="8" />
		<line x1="21" y1="21" x2="16.65" y2="16.65" />
	{:else if name === 'user'}
		<path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2" />
		<circle cx="12" cy="7" r="4" />
	{:else if name === 'award'}
		<circle cx="12" cy="8" r="7" />
		<polyline points="8.21 13.89 7 23 12 20 17 23 15.79 13.88" />
	{:else if name === 'chevron-left'}
		<polyline points="15 18 9 12 15 6" />
	{:else if name === 'chevron-right'}
		<polyline points="9 18 15 12 9 6" />
	{:else if name === 'chevrons-left'}
		<polyline points="11 17 6 12 11 7" />
		<polyline points="18 17 13 12 18 7" />
	{:else if name === 'chevrons-right'}
		<polyline points="13 17 18 12 13 7" />
		<polyline points="6 17 11 12 6 7" />
	{:else if name === 'check'}
		<polyline points="20 6 9 17 4 12" />
	{:else if name === 'x'}
		<line x1="18" y1="6" x2="6" y2="18" />
		<line x1="6" y1="6" x2="18" y2="18" />
	{:else if name === 'flag'}
		<path d="M4 15s1-1 4-1 5 2 8 2 4-1 4-1V3s-1 1-4 1-5-2-8-2-4 1-4 1z" />
		<line x1="4" y1="22" x2="4" y2="15" />
	{:else if name === 'handshake'}
		<path d="m11 17 2 2a1 1 0 0 0 1.4 0l4.3-4.3a1 1 0 0 0 0-1.4l-2-2" />
		<path d="m18 10 1.3-1.3a2.5 2.5 0 0 0 0-3.5v0a2.5 2.5 0 0 0-3.5 0L14 7" />
		<path d="m6 18 1.3-1.3a2.5 2.5 0 0 0 0-3.5v0a2.5 2.5 0 0 0-3.5 0L2 15" />
		<path d="m8 14 3-3" />
	{:else if name === 'play'}
		<polygon points="5 3 19 12 5 21 5 3" fill="currentColor" />
	{:else if name === 'arrow-right'}
		<line x1="5" y1="12" x2="19" y2="12" />
		<polyline points="12 5 19 12 12 19" />
	{:else if name === 'log-out'}
		<path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4" />
		<polyline points="16 17 21 12 16 7" />
		<line x1="21" y1="12" x2="9" y2="12" />
	{:else if name === 'settings'}
		<circle cx="12" cy="12" r="3" />
		<path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z" />
	{:else if name === 'volume-2'}
		<polygon points="11 5 6 9 2 9 2 15 6 15 11 19 11 5" />
		<path d="M15.54 8.46a5 5 0 0 1 0 7.07" />
		<path d="M19.07 4.93a10 10 0 0 1 0 14.14" />
	{:else if name === 'volume-x'}
		<polygon points="11 5 6 9 2 9 2 15 6 15 11 19 11 5" />
		<line x1="22" y1="9" x2="16" y2="15" />
		<line x1="16" y1="9" x2="22" y2="15" />
	{:else if name === 'download'}
		<path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" />
		<polyline points="7 10 12 15 17 10" />
		<line x1="12" y1="15" x2="12" y2="3" />
	{:else if name === 'eye'}
		<path d="M2 12s3-7 10-7 10 7 10 7-3 7-10 7-10-7-10-7Z" />
		<circle cx="12" cy="12" r="3" />
	{:else if name === 'external-link'}
		<path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6" />
		<polyline points="15 3 21 3 21 9" />
		<line x1="10" y1="14" x2="21" y2="3" />
	{:else if name === 'camera'}
		<path d="M14.5 4h-5L7 7H4a2 2 0 0 0-2 2v9a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2V9a2 2 0 0 0-2-2h-3l-2.5-3z" />
		<circle cx="12" cy="13" r="3" />
	{:else if name === 'upload'}
		<path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" />
		<polyline points="17 8 12 3 7 8" />
		<line x1="12" y1="3" x2="12" y2="15" />
	{:else if name === 'edit'}
		<path d="M17 3a2.85 2.83 0 1 1 4 4L7.5 20.5 2 22l1.5-5.5Z" />
		<path d="m15 5 4 4" />
	{:else if name === 'wifi-off'}
		<line x1="1" y1="1" x2="23" y2="23" />
		<path d="M16.72 11.06A10.94 10.94 0 0 1 19 12.55" />
		<path d="M5 12.55a10.94 10.94 0 0 1 5.17-2.39" />
		<path d="M10.71 5.05A16 16 0 0 1 22.58 9" />
		<path d="M1.42 9a15.91 15.91 0 0 1 4.7-2.88" />
		<path d="M8.53 16.11a6 6 0 0 1 6.95 0" />
		<line x1="12" y1="20" x2="12.01" y2="20" />
	{:else if name === 'copy'}
		<rect width="14" height="14" x="8" y="8" rx="2" ry="2" />
		<path d="M4 16c-1.1 0-2-.9-2-2V4c0-1.1.9-2 2-2h10c1.1 0 2 .9 2 2" />
	{:else if name === 'brand-arch'}
		<path
			d="m12 2.5c-.5 1.2-1.2 2.9-2.1 5 1.3.7 2.4 1.7 3.3 2.9.9-1.2 2.1-2.2 3.3-2.9-.8-2-1.5-3.7-2.1-5h-.4zm-4.1 9.9c-2.5 5.2-4.9 9.8-4.9 9.8s5.7-3.1 9-3.1c3.3 0 9 3.1 9 3.1s-2.4-4.6-4.9-9.9c-1.4.9-2.7 2.1-3.7 3.5-.3.4-.6.4-.9 0-1-1.4-2.3-2.6-3.7-3.4z"
			fill="currentColor"
			stroke="none"
		/>
	{:else if name === 'brand-linux'}
		<path
			d="M12 2c-2.8 0-4.7 2.2-4.7 5.4 0 1.3.3 2.7.7 3.9-.4.4-.8 1-1.1 1.6-.6 1.3-.9 2.9-.8 4.1.1 1.2.6 2.1 1.4 2.5.5.2 1.2.3 1.8.2.5.9 1.4 1.5 2.6 1.5 1 0 1.8-.5 2.3-1.2.5.7 1.3 1.2 2.3 1.2 1.1 0 2-.6 2.6-1.5.7.1 1.3.1 1.8-.2.8-.4 1.3-1.3 1.4-2.5.1-1.3-.1-2.8-.8-4.1-.3-.7-.7-1.2-1.1-1.6.5-1.2.7-2.6.7-3.9C21.5 4.2 19.5 2 12 2zm-1.9 4.8a.9.9 0 1 1 0 1.8.9.9 0 0 1 0-1.8zm3.8 0a.9.9 0 1 1 0 1.8.9.9 0 0 1 0-1.8z"
			fill="currentColor"
			stroke="none"
		/>
	{:else if name === 'brand-windows'}
		<path
			d="M2.5 3.5 10.5 2.4v8.8H2.5V3.5zm9.3-1.3L22 .7v10.5h-10.2V2.2zM2.5 12.8h8v8.8l-8-1.1v-7.7zm9.3 0H22v10.5l-10.2-1.5V12.8z"
			fill="currentColor"
			stroke="none"
		/>
	{:else if name === 'brand-apple'}
		<path
			d="M18.71 19.5c-.83 1.24-1.71 2.45-3.05 2.47-1.34.03-1.77-.79-3.29-.79-1.53 0-2 .77-3.27.82-1.31.05-2.3-1.32-3.14-2.53C4.25 17 2.94 12.45 4.7 9.39c.87-1.52 2.43-2.48 4.12-2.51 1.28-.02 2.5.87 3.29.87.78 0 2.26-1.07 3.81-.91.65.03 2.47.26 3.64 1.98-.09.06-2.17 1.28-2.15 3.81.03 3.02 2.65 4.03 2.68 4.04-.03.07-.42 1.44-1.38 2.83M15.97 6.36c.64-.78 1.08-1.86.96-2.95-1 .04-2.14.65-2.77 1.4-.56.65-1.04 1.74-.91 2.8 1.11.09 2.19-.58 2.72-1.25z"
			fill="currentColor"
			stroke="none"
		/>
	{:else if name === 'brand-debian'}
		<circle cx="12" cy="12" r="10" />
		<path d="M12 7c2.8 0 5 2.2 5 5s-2.2 5-5 5-5-2.2-5-5" />
	{/if}
</svg>
