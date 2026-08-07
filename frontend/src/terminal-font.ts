export const defaultTerminalFontFamily = 'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace'

export function resolveTerminalFontFamily(fontFamily: string | undefined): string {
  const family = fontFamily?.trim()
  if (!family) {
    return defaultTerminalFontFamily
  }
  const escapedFamily = family.replace(/\\/g, '\\\\').replace(/"/g, '\\"')
  return `"${escapedFamily}", ${defaultTerminalFontFamily}`
}
