import {describe, expect, it} from 'vitest'

import {
  defaultTerminalFontSize,
  maximumTerminalFontSize,
  minimumTerminalFontSize,
  normalizeTerminalFontSize,
  terminalFontSizePercent,
} from './terminal-font-size'

describe('terminal font size', () => {
  it('缺失字号时使用默认值，并将范围外数值限制在支持区间', () => {
    expect(defaultTerminalFontSize).toBe(13)
    expect(minimumTerminalFontSize).toBe(10)
    expect(maximumTerminalFontSize).toBe(24)
    expect(normalizeTerminalFontSize()).toBe(13)
    expect(normalizeTerminalFontSize(8)).toBe(10)
    expect(normalizeTerminalFontSize(27)).toBe(24)
    expect(normalizeTerminalFontSize(16)).toBe(16)
  })

  it('显示相对于默认字号的四舍五入比例', () => {
    expect(terminalFontSizePercent(13)).toBe(100)
    expect(terminalFontSizePercent(16)).toBe(123)
    expect(terminalFontSizePercent(10)).toBe(77)
  })
})
