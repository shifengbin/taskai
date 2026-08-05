/** @type {import('tailwindcss').Config} */
// 快门波普（Snap）令牌：颜色/字体/圆角/硬投影都引用 CSS 变量 var(--snap-*)，
// 由 style.css 的 :root（亮色默认）与 .dark（暗色覆盖）提供具体值。
// 字体由 @fontsource-variable 自托管（woff2，离线），family 名带 " Variable" 后缀。
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
        display: ['"Hanken Grotesk Variable"', '"Noto Sans SC"', 'sans-serif'],
        sans: ['"Plus Jakarta Sans Variable"', '"Noto Sans SC"', 'sans-serif'],
        mono: ['"JetBrains Mono Variable"', 'ui-monospace', 'monospace'],
      },
      borderRadius: {
        'snap-sm': '6px',
        snap: '7px',
        'snap-lg': '9px',
      },
      boxShadow: {
        'snap-sm': '2px 2px 0 var(--snap-outline)',
        snap: '3px 3px 0 var(--snap-outline)',
        'snap-lg': '4px 4px 0 var(--snap-outline)',
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
