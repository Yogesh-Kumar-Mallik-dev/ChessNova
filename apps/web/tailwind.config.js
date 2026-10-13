/** @type {import('tailwindcss').Config} */
export default {
	content: ['./src/**/*.{html,js,svelte,ts}'],
	theme: {
		extend: {
			colors: {
				border: 'hsl(var(--border))',
				input: 'hsl(var(--input))',
				ring: 'hsl(var(--ring))',
				background: 'hsl(var(--background))',
				foreground: 'hsl(var(--foreground))',
				primary: {
					DEFAULT: 'hsl(var(--primary))',
					foreground: 'hsl(var(--primary-foreground))'
				},
				secondary: {
					DEFAULT: 'hsl(var(--secondary))',
					foreground: 'hsl(var(--secondary-foreground))'
				},
				destructive: {
					DEFAULT: 'hsl(var(--destructive))',
					foreground: 'hsl(var(--destructive-foreground))'
				},
				muted: {
					DEFAULT: 'hsl(var(--muted))',
					foreground: 'hsl(var(--muted-foreground))'
				},
				accent: {
					DEFAULT: 'hsl(var(--accent))',
					foreground: 'hsl(var(--accent-foreground))'
				},
				popover: {
					DEFAULT: 'hsl(var(--popover))',
					foreground: 'hsl(var(--popover-foreground))'
				},
				card: {
					DEFAULT: 'hsl(var(--card))',
					foreground: 'hsl(var(--card-foreground))'
				},
				slate: {
					50: '#f8fafc',
					100: '#f1f5f9',
					200: '#e2e8f0',
					300: '#cbd5e1',
					400: '#94a3b8',
					500: '#64748b',
					600: 'hsl(var(--app-slate-600, 215 20% 30%) / <alpha-value>)',
					700: 'hsl(var(--app-slate-700, 215 25% 20%) / <alpha-value>)',
					750: 'hsl(var(--app-slate-750, 217 30% 16%) / <alpha-value>)',
					800: 'hsl(var(--app-slate-800, 217 33% 13%) / <alpha-value>)',
					850: 'hsl(var(--app-slate-850, 220 38% 10%) / <alpha-value>)',
					900: 'hsl(var(--app-slate-900, 222 47% 7%) / <alpha-value>)',
					950: 'hsl(var(--app-slate-950, 222 47% 4%) / <alpha-value>)'
				},
				nova: {
					void: '#06080d',
					space: '#0a0d14',
					surface: '#0f141f',
					surfaceHover: '#151c2b',
					surfaceElevated: '#1a2235',
					border: '#1e293b',
					borderLight: '#334155',
					text: '#f8fafc',
					textMuted: '#94a3b8',
					textSubtle: '#64748b'
				}
			},
			borderRadius: {
				lg: 'var(--radius)',
				md: 'calc(var(--radius) - 2px)',
				sm: 'calc(var(--radius) - 4px)'
			},
			fontFamily: {
				sans: ['Inter', '-apple-system', 'BlinkMacSystemFont', 'Segoe UI', 'Roboto', 'sans-serif'],
				mono: ['JetBrains Mono', 'Fira Code', 'monospace']
			},
			keyframes: {
				'accordion-down': {
					from: { height: '0' },
					to: { height: 'var(--radix-accordion-content-height)' }
				},
				'accordion-up': {
					from: { height: 'var(--radix-accordion-content-height)' },
					to: { height: '0' }
				},
				'fade-in': {
					from: { opacity: '0', transform: 'translateY(8px)' },
					to: { opacity: '1', transform: 'translateY(0)' }
				},
				'glow-pulse': {
					'0%, 100%': { opacity: '0.4' },
					'50%': { opacity: '0.85' }
				},
				'shimmer-slide': {
					to: {
						transform: 'translate(calc(100cqw - 100%), 0)'
					}
				},
				'spin-around': {
					'0%': {
						transform: 'translateZ(0) rotate(0)'
					},
					'15%, 35%': {
						transform: 'translateZ(0) rotate(90deg)'
					},
					'65%, 85%': {
						transform: 'translateZ(0) rotate(270deg)'
					},
					'100%': {
						transform: 'translateZ(0) rotate(360deg)'
					}
				},
				'border-beam': {
					'100%': {
						'offset-distance': '100%'
					}
				},
				'spotlight': {
					'0%': {
						opacity: 0,
						transform: 'translate(-72%, -62%) scale(0.5)'
					},
					'100%': {
						opacity: 1,
						transform: 'translate(-50%,-40%) scale(1)'
					}
				}
			},
			animation: {
				'accordion-down': 'accordion-down 0.2s ease-out',
				'accordion-up': 'accordion-up 0.2s ease-out',
				'fade-in': 'fade-in 0.35s cubic-bezier(0.16, 1, 0.3, 1)',
				'glow-pulse': 'glow-pulse 3s cubic-bezier(0.4, 0, 0.6, 1) infinite',
				'shimmer-slide': 'shimmer-slide var(--speed) ease-in-out infinite alternate',
				'spin-around': 'spin-around calc(var(--speed) * 2) infinite linear',
				'border-beam': 'border-beam calc(var(--duration)*1s) infinite linear',
				'spotlight': 'spotlight 2s ease .75s 1 forwards'
			}
		}
	},
	plugins: []
};
