/** @type {import('tailwindcss').Config} */
// Nebula 令牌：颜色/字体/圆角/阴影都引用 CSS 变量 var(--snap-*)，
// 由 style.css 的 :root（亮色默认）与 .dark（暗色覆盖）提供具体值。
module.exports = {
  darkMode: 'class',
  content: ['./index.html', './src/**/*.{ts,tsx}'],
  theme: {
    extend: {
      colors: {
        snap: {
          canvas: 'var(--snap-canvas)',
          surface: 'var(--snap-surface)',
          'surface-2': 'var(--snap-surface-2)',
          detail: 'var(--snap-detail)',
          overlay: 'var(--snap-overlay)',
          ink: 'var(--snap-ink)',
          muted: 'var(--snap-muted)',
          outline: 'var(--snap-outline)',
          coral: 'var(--snap-coral)',
          cobalt: 'var(--snap-cobalt)',
          amber: 'var(--snap-amber)',
          violet: 'var(--snap-violet)',
          error: 'var(--snap-error)',
        },
      },
      fontFamily: {
        display: ['"Chakra Petch"', '"Noto Sans SC"', 'sans-serif'],
        sans: ['"Sora"', '"Noto Sans SC"', 'sans-serif'],
        mono: ['"JetBrains Mono Variable"', 'ui-monospace', 'monospace'],
      },
      borderRadius: {
        'snap-sm': '8px',
        snap: '10px',
        'snap-lg': '10px',
      },
      boxShadow: {
        'snap-sm': '0 8px 22px -16px var(--snap-shadow)',
        snap: '0 14px 34px -20px var(--snap-shadow)',
        'snap-lg': '0 22px 48px -26px var(--snap-shadow)',
      },
      backgroundImage: {
        'snap-dots': 'radial-gradient(var(--snap-dots) 1.3px, transparent 1.3px)',
      },
      backgroundSize: {
        'snap-dots': '15px 15px',
      },
      keyframes: {
        'accordion-down': {
          from: {height: '0'},
          to: {height: 'var(--radix-accordion-content-height)'},
        },
        'accordion-up': {
          from: {height: 'var(--radix-accordion-content-height)'},
          to: {height: '0'},
        },
      },
      animation: {
        'accordion-down': 'accordion-down 0.2s ease-out',
        'accordion-up': 'accordion-up 0.2s ease-out',
      },
    },
  },
  plugins: [require('tailwindcss-animate')],
}
