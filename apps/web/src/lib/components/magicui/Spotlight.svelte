<script lang="ts">
	import { cn } from '$lib/utils';

	interface Props {
		class?: string;
		fill?: string;
	}

	let { class: className = '', fill = 'white' }: Props = $props();
</script>

<svg
	class={cn(
		'pointer-events-none absolute z-0 h-[170%] w-[140%] animate-spotlight opacity-0',
		className
	)}
	xmlns="http://www.w3.org/2000/svg"
	viewBox="0 0 3787 2842"
	fill="none"
>
	<g filter="url(#filter)">
		<ellipse
			cx="1924.71"
			cy="273.501"
			rx="1924.71"
			ry="273.501"
			transform="matrix(-0.822377 -0.568943 -0.568943 0.822377 3631.88 2291.09)"
			fill={fill}
			fill-opacity="0.18"
		></ellipse>
	</g>
	<defs>
		<filter
			id="filter"
			x="0.860352"
			y="0.838989"
			width="3785.16"
			height="2840.26"
			filterUnits="userSpaceOnUse"
			color-interpolation-filters="sRGB"
		>
			<feFlood flood-opacity="0" result="BackgroundImageFix"></feFlood>
			<feBlend mode="normal" in="SourceGraphic" in2="BackgroundImageFix" result="shape"></feBlend>
			<feGaussianBlur stdDeviation="151" result="effect1_foregroundBlur_1065_8"></feGaussianBlur>
		</filter>
	</defs>
</svg>
