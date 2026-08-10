export interface TerminalTheme {
  background: string
  foreground: string
  cursor: string
  cursorAccent: string
  selectionBackground: string
  selectionForeground: string
  black: string
  red: string
  green: string
  yellow: string
  blue: string
  magenta: string
  cyan: string
  white: string
  brightBlack: string
  brightRed: string
  brightGreen: string
  brightYellow: string
  brightBlue: string
  brightMagenta: string
  brightCyan: string
  brightWhite: string
}

export const defaultTerminalTheme: TerminalTheme = {
  background: '#070A16',
  foreground: '#E8ECFF',
  cursor: '#2DE2E6',
  cursorAccent: '#070A16',
  selectionBackground: '#2DE2E640',
  selectionForeground: '#FFFFFF',
  black: '#070A16',
  red: '#FF5D73',
  green: '#34F5C5',
  yellow: '#F3C969',
  blue: '#2DE2E6',
  magenta: '#B06BFF',
  cyan: '#2DE2E6',
  white: '#8A93C2',
  brightBlack: '#8A93C2',
  brightRed: '#FF7A6E',
  brightGreen: '#34F5C5',
  brightYellow: '#F3C969',
  brightBlue: '#2DE2E6',
  brightMagenta: '#C08BFF',
  brightCyan: '#6FF5F2',
  brightWhite: '#E8ECFF',
}

export function normalizeTerminalTheme(theme?: Partial<TerminalTheme>): TerminalTheme {
  return {
    background: normalizeTerminalThemeColor(theme?.background, defaultTerminalTheme.background),
    foreground: normalizeTerminalThemeColor(theme?.foreground, defaultTerminalTheme.foreground),
    cursor: normalizeTerminalThemeColor(theme?.cursor, defaultTerminalTheme.cursor),
    cursorAccent: normalizeTerminalThemeColor(theme?.cursorAccent, defaultTerminalTheme.cursorAccent),
    selectionBackground: normalizeTerminalThemeColor(theme?.selectionBackground, defaultTerminalTheme.selectionBackground, true),
    selectionForeground: normalizeTerminalThemeColor(theme?.selectionForeground, defaultTerminalTheme.selectionForeground),
    black: normalizeTerminalThemeColor(theme?.black, defaultTerminalTheme.black),
    red: normalizeTerminalThemeColor(theme?.red, defaultTerminalTheme.red),
    green: normalizeTerminalThemeColor(theme?.green, defaultTerminalTheme.green),
    yellow: normalizeTerminalThemeColor(theme?.yellow, defaultTerminalTheme.yellow),
    blue: normalizeTerminalThemeColor(theme?.blue, defaultTerminalTheme.blue),
    magenta: normalizeTerminalThemeColor(theme?.magenta, defaultTerminalTheme.magenta),
    cyan: normalizeTerminalThemeColor(theme?.cyan, defaultTerminalTheme.cyan),
    white: normalizeTerminalThemeColor(theme?.white, defaultTerminalTheme.white),
    brightBlack: normalizeTerminalThemeColor(theme?.brightBlack, defaultTerminalTheme.brightBlack),
    brightRed: normalizeTerminalThemeColor(theme?.brightRed, defaultTerminalTheme.brightRed),
    brightGreen: normalizeTerminalThemeColor(theme?.brightGreen, defaultTerminalTheme.brightGreen),
    brightYellow: normalizeTerminalThemeColor(theme?.brightYellow, defaultTerminalTheme.brightYellow),
    brightBlue: normalizeTerminalThemeColor(theme?.brightBlue, defaultTerminalTheme.brightBlue),
    brightMagenta: normalizeTerminalThemeColor(theme?.brightMagenta, defaultTerminalTheme.brightMagenta),
    brightCyan: normalizeTerminalThemeColor(theme?.brightCyan, defaultTerminalTheme.brightCyan),
    brightWhite: normalizeTerminalThemeColor(theme?.brightWhite, defaultTerminalTheme.brightWhite),
  }
}

function normalizeTerminalThemeColor(value: string | undefined, fallback: string, allowAlpha = false): string {
  const normalized = value?.trim().toUpperCase() ?? ''
  const hexadecimalLength = allowAlpha ? '(?:[0-9A-F]{6}|[0-9A-F]{8})' : '[0-9A-F]{6}'
  return new RegExp(`^#${hexadecimalLength}$`).test(normalized) ? normalized : fallback
}
