import type {TerminalShortcut, TerminalShortcutStep} from './types'

const modifierOrder = ['Ctrl', 'Alt', 'Shift', 'Command'] as const

export function keyboardEventShortcut(event: KeyboardEvent): string | undefined {
  if (event.type !== 'keydown') {
    return undefined
  }
  const key = normalizeKey(event.key)
  if (!key || modifierOrder.some((modifier) => key === modifier)) {
    return undefined
  }
  const modifiers: string[] = []
  if (event.ctrlKey) modifiers.push('Ctrl')
  if (event.altKey) modifiers.push('Alt')
  if (event.shiftKey) modifiers.push('Shift')
  if (event.metaKey) modifiers.push('Command')
  return [...modifiers, key].join('+')
}

export function findTerminalShortcut(shortcuts: TerminalShortcut[], event: KeyboardEvent): TerminalShortcut | undefined {
  const binding = keyboardEventShortcut(event)
  return binding ? shortcuts.find((shortcut) => shortcut.shortcut === binding) : undefined
}

export function terminalShortcutApplies(shortcut: TerminalShortcut, command?: string): boolean {
  const scope = shortcut.includePrograms
  if (!scope || scope.length === 0) {
    return true
  }
  if (!command) {
    return false
  }
  const normalizedCommand = normalizeProgramName(command)
  return scope.some((program) => normalizeProgramName(program) === normalizedCommand)
}

export function normalizeProgramName(value: string): string {
  const trimmed = value.trim()
  const segments = trimmed.split(/[\\/]/)
  const basename = segments[segments.length - 1] ?? trimmed
  return basename.replace(/\.(?:exe|com)$/i, '').toLowerCase()
}

// Normalizes a set of launch commands into deduplicated program basenames,
// preserving first-seen order. Used to build the candidate list of programs
// that can display a terminal (shell + show-terminal task menu commands).
export function uniqueProgramNames(commands: Array<string | undefined | null>): string[] {
  const seen = new Set<string>()
  const result: string[] = []
  for (const command of commands) {
    if (!command) {
      continue
    }
    const normalized = normalizeProgramName(command)
    if (!normalized || seen.has(normalized)) {
      continue
    }
    seen.add(normalized)
    result.push(normalized)
  }
  return result
}

export function terminalShortcutInput(steps: TerminalShortcutStep[]): string {
  return steps.map(terminalShortcutStepInput).join('')
}

export function terminalShortcutStepInput(step: TerminalShortcutStep): string {
  if (step.kind === 'text') {
    return step.text
  }
  if (step.kind === 'enter') {
    return '\r'
  }
  return keyInput(step.key, step.modifiers ?? [])
}

export function keyboardEventStep(event: KeyboardEvent): {kind: 'key', key: string, modifiers: string[]} | undefined {
  if (event.type !== 'keydown') {
    return undefined
  }
  const key = normalizeKeyName(event.key)
  if (!key || modifierOrder.some((modifier) => key === modifier)) {
    return undefined
  }
  const modifiers = keyboardEventModifiers(event)
  return {kind: 'key', key, modifiers}
}

export function normalizeKeyName(value: string): string | undefined {
  const trimmed = value.trim()
  const lower = trimmed.toLocaleLowerCase()
  const namedKeys: Record<string, string> = {
    enter: 'Enter',
    return: 'Enter',
    tab: 'Tab',
    escape: 'Escape',
    esc: 'Escape',
    space: 'Space',
    backspace: 'Backspace',
    delete: 'Delete',
    arrowup: 'ArrowUp',
    arrowdown: 'ArrowDown',
    arrowleft: 'ArrowLeft',
    arrowright: 'ArrowRight',
    home: 'Home',
    end: 'End',
    pageup: 'PageUp',
    pagedown: 'PageDown',
    insert: 'Insert',
  }
  if (value === ' ') return 'Space'
  if (namedKeys[lower]) return namedKeys[lower]
  if (/^f(?:[1-9]|1[0-2])$/i.test(trimmed)) return trimmed.toUpperCase()
  if (Array.from(trimmed).length === 1 && /^[\p{L}\p{N}]$/u.test(trimmed)) return trimmed.toUpperCase()
  if (Array.from(trimmed).length === 1 && /^[^\p{C}\p{Z}]$/u.test(trimmed)) return trimmed
  return undefined
}

function normalizeKey(value: string): string | undefined {
  return normalizeKeyName(value)
}

function keyboardEventModifiers(event: KeyboardEvent): string[] {
  const modifiers: string[] = []
  if (event.ctrlKey) modifiers.push('Ctrl')
  if (event.altKey) modifiers.push('Alt')
  if (event.shiftKey) modifiers.push('Shift')
  if (event.metaKey) modifiers.push('Command')
  return modifiers
}

function keyInput(key: string, modifiers: string[]): string {
  const has = (modifier: string) => modifiers.includes(modifier)
  const shifted = has('Shift')
  const alt = has('Alt')
  const ctrl = has('Ctrl')
  const command = has('Command')
  if (/^[A-Z]$/.test(key)) {
    const character = shifted ? key : key.toLocaleLowerCase()
    if (ctrl) {
      return String.fromCharCode(character.toLocaleUpperCase().charCodeAt(0) - 64)
    }
    return `${alt || command ? '\u001b' : ''}${character}`
  }
  if (ctrl) {
    const controlCharacter = controlInput(key)
    if (controlCharacter !== undefined) return controlCharacter
  }
  const modifier = 1 + (shifted ? 1 : 0) + (alt ? 2 : 0) + (ctrl ? 4 : 0) + (command ? 8 : 0)
  if (key === 'Space') {
    return `${alt || command ? '\u001b' : ''} `
  }
  if (key === 'Enter') return `${alt || command ? '\u001b' : ''}\r`
  if (key === 'Tab') {
    if (modifier === 1) return '\t'
    if (modifier === 2) return '\u001b[Z'
    return `\u001b[1;${modifier}Z`
  }
  if (key === 'Escape') return `${alt || command ? '\u001b' : ''}\u001b`
  if (key === 'Backspace') return `${alt || command ? '\u001b' : ''}\u007f`
  const navigation: Record<string, string> = {
    ArrowUp: 'A',
    ArrowDown: 'B',
    ArrowRight: 'C',
    ArrowLeft: 'D',
  }
  if (navigation[key]) {
    return modifier === 1 ? `\u001b[${navigation[key]}` : `\u001b[1;${modifier}${navigation[key]}`
  }
  const tildeKeys: Record<string, number> = {Insert: 2, Delete: 3, PageUp: 5, PageDown: 6}
  if (tildeKeys[key]) return `\u001b[${tildeKeys[key]}~`
  if (key === 'Home') return modifier === 1 ? '\u001b[H' : `\u001b[1;${modifier}H`
  if (key === 'End') return modifier === 1 ? '\u001b[F' : `\u001b[1;${modifier}F`
  const functionKey = /^F([1-9]|1[0-2])$/.exec(key)
  if (functionKey) {
    const number = Number(functionKey[1])
    const legacySequences: Record<number, string> = {1: '\u001bOP', 2: '\u001bOQ', 3: '\u001bOR', 4: '\u001bOS', 5: '\u001b[15~', 6: '\u001b[17~', 7: '\u001b[18~', 8: '\u001b[19~', 9: '\u001b[20~', 10: '\u001b[21~', 11: '\u001b[23~', 12: '\u001b[24~'}
    if (modifier === 1) return legacySequences[number]
    const ss3Suffixes: Record<number, string> = {1: 'P', 2: 'Q', 3: 'R', 4: 'S'}
    if (ss3Suffixes[number]) return `\u001b[1;${modifier}${ss3Suffixes[number]}`
    const csiCodes: Record<number, number> = {5: 15, 6: 17, 7: 18, 8: 19, 9: 20, 10: 21, 11: 23, 12: 24}
    return `\u001b[${csiCodes[number]};${modifier}~`
  }
  return `${alt || command ? '\u001b' : ''}${key}`
}

function controlInput(key: string): string | undefined {
  if (/^[A-Z]$/.test(key)) return String.fromCharCode(key.charCodeAt(0) - 64)
  const controls: Record<string, string> = {
    Space: '\u0000',
    '@': '\u0000',
    '[': '\u001b',
    '\\': '\u001c',
    ']': '\u001d',
    '^': '\u001e',
    '_': '\u001f',
    '?': '\u007f',
  }
  return controls[key]
}
