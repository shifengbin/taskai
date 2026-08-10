import {describe, expect, it} from 'vitest'

import {defaultTerminalTheme, normalizeTerminalTheme} from './terminal-theme'

describe('terminal theme', () => {
  it('uses the current dark terminal palette by default', () => {
    expect(defaultTerminalTheme).toMatchObject({
      background: '#070A16',
      foreground: '#E8ECFF',
      cursor: '#2DE2E6',
      selectionBackground: '#2DE2E640',
      brightCyan: '#6FF5F2',
    })
  })

  it('preserves valid theme colors and restores missing fields', () => {
    const normalized = normalizeTerminalTheme({
      background: '#102030',
      cursor: '#abcdef',
      selectionBackground: '#1234567f',
    })

    expect(normalized.background).toBe('#102030')
    expect(normalized.cursor).toBe('#ABCDEF')
    expect(normalized.selectionBackground).toBe('#1234567F')
    expect(normalized.foreground).toBe(defaultTerminalTheme.foreground)
  })
})
