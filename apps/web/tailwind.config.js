/** @type {import('tailwindcss').Config} */
export default {
	content: ['./src/**/*.{html,js,svelte,ts}'],
	theme: {
		extend: {
			colors: {
				nova: {
					void: '#07090e',
					space: '#0c1017',
					surface: '#121824',
					surfaceHover: '#182030',
					surfaceElevated: '#1e283d',
					border: '#1f293d',
					borderLight: '#2c3a56',
					text: '#f1f5f9',
					textMuted: '#94a3b8',
					textSubtle: '#64748b',
					cyan: '#38bdf8',
					indigo: '#6366f1',
					emerald: '#10b981',
					amber: '#f59e0b',
					crimson: '#f43f5e',
					boardDark: '#2b3648',
					boardLight: '#cdd6e0',
					boardHighlight: 'rgba(99, 102, 241, 0.4)',
					boardLastMove: 'rgba(56, 189, 248, 0.35)',
					boardCheck: 'rgba(244, 63, 94, 0.65)'
				}
			},
			fontFamily: {
				sans: ['Inter', '-apple-system', 'BlinkMacSystemFont', 'Segoe UI', 'Roboto', 'sans-serif'],
				mono: ['JetBrains Mono', 'Fira Code', 'monospace']
			},
			backgroundImage: {
				'nova-gradient': 'linear-gradient(135deg, #38bdf8 0%, #6366f1 100%)',
				'nova-card': 'linear-gradient(180deg, rgba(18, 24, 36, 0.85) 0%, rgba(12, 16, 23, 0.95) 100%)',
				'nova-radial': 'radial-gradient(circle at 50% 0%, rgba(56, 189, 248, 0.12) 0%, transparent 70%)'
			}
		}
	},
	plugins: []
};
