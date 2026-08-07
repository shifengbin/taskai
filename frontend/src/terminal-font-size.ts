export const defaultTerminalFontSize = 13
export const minimumTerminalFontSize = 10
export const maximumTerminalFontSize = 24

export function normalizeTerminalFontSize(fontSize?: number): number {
  if (fontSize === undefined || !Number.isFinite(fontSize) || fontSize === 0) {
    return defaultTerminalFontSize
  }
  return Math.min(Math.max(Math.round(fontSize), minimumTerminalFontSize), maximumTerminalFontSize)
}

export function terminalFontSizePercent(fontSize?: number): number {
  return Math.round(normalizeTerminalFontSize(fontSize) / defaultTerminalFontSize * 100)
}
