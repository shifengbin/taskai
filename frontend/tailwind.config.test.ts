import {createRequire} from 'node:module'
import {describe, expect, it} from 'vitest'

const require = createRequire(import.meta.url)
const config = require('./tailwind.config.cjs') as {
  theme: {extend: {boxShadow: Record<string, string>}}
}

describe('Snap shadow tokens', () => {
  it('uses the original Snap hard-shadow hierarchy while preserving the outline color', () => {
    expect(config.theme.extend.boxShadow).toMatchObject({
      'snap-sm': '2px 2px 0 var(--snap-outline)',
      snap: '3px 3px 0 var(--snap-outline)',
      'snap-lg': '4px 4px 0 var(--snap-outline)',
    })
  })
})
