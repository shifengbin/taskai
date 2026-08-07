import {createRequire} from 'node:module'
import {describe, expect, it} from 'vitest'

const require = createRequire(import.meta.url)
const config = require('./tailwind.config.cjs') as {
  theme: {
    extend: {
      colors: {snap: Record<string, string>}
      fontFamily: Record<string, string[]>
      borderRadius: Record<string, string>
    }
  }
}

describe('Nebula visual tokens', () => {
  it('uses the Nebula font stack for body, display, and terminal content', () => {
    expect(config.theme.extend.fontFamily).toMatchObject({
      sans: ['"Sora"', '"Noto Sans SC"', 'sans-serif'],
      display: ['"Chakra Petch"', '"Noto Sans SC"', 'sans-serif'],
      mono: ['"JetBrains Mono Variable"', 'ui-monospace', 'monospace'],
    })
  })

  it('keeps the Nebula surface aliases available to existing UI classes', () => {
    expect(config.theme.extend.colors.snap).toMatchObject({
      canvas: 'var(--snap-canvas)',
      surface: 'var(--snap-surface)',
      'surface-2': 'var(--snap-surface-2)',
      detail: 'var(--snap-detail)',
      overlay: 'var(--snap-overlay)',
      ink: 'var(--snap-ink)',
      muted: 'var(--snap-muted)',
      outline: 'var(--snap-outline)',
      cobalt: 'var(--snap-cobalt)',
      violet: 'var(--snap-violet)',
    })
  })

  it('uses Nebula geometry for compact, card, and overlay surfaces', () => {
    expect(config.theme.extend.borderRadius).toMatchObject({
      'snap-sm': '8px',
      snap: '10px',
      'snap-lg': '10px',
    })
  })
})
