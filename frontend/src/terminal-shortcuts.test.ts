import {describe, expect, it} from 'vitest'

import {findTerminalShortcut, keyboardEventShortcut, keyboardEventStep, terminalShortcutApplies, terminalShortcutInput, terminalShortcutStepInput, uniqueProgramNames} from './terminal-shortcuts'
import type {TerminalShortcut, TerminalShortcutStep} from './types'

describe('terminal shortcut matching', () => {
  it('normalizes keyboard modifiers and named keys', () => {
    expect(keyboardEventShortcut(new KeyboardEvent('keydown', {key: 'Enter', shiftKey: true}))).toBe('Shift+Enter')
    expect(keyboardEventShortcut(new KeyboardEvent('keydown', {key: 'p', ctrlKey: true, shiftKey: true}))).toBe('Ctrl+Shift+P')
  })

  it('finds only the configured binding', () => {
    const shortcut: TerminalShortcut = {id: 'shortcut-1', shortcut: 'Shift+Enter', steps: [{kind: 'text', text: '\\'}, {kind: 'enter'}]}
    expect(findTerminalShortcut([shortcut], new KeyboardEvent('keydown', {key: 'Enter', shiftKey: true}))).toEqual(shortcut)
    expect(findTerminalShortcut([shortcut], new KeyboardEvent('keydown', {key: 'Enter'}))).toBeUndefined()
  })

  it('converts text and Enter steps to direct terminal input', () => {
    expect(terminalShortcutInput([{kind: 'text', text: '\\'}, {kind: 'enter'}, {kind: 'text', text: 'next'}])).toBe('\\\rnext')
  })

  it('captures a key action with modifiers and maps common terminal keys', () => {
    expect(keyboardEventStep(new KeyboardEvent('keydown', {key: 'Tab', shiftKey: true}))).toEqual({kind: 'key', key: 'Tab', modifiers: ['Shift']})
    expect(keyboardEventStep(new KeyboardEvent('keydown', {key: '.', altKey: true}))).toEqual({kind: 'key', key: '.', modifiers: ['Alt']})
    expect(terminalShortcutStepInput({kind: 'key', key: 'ArrowUp', modifiers: []})).toBe('\u001b[A')
    expect(terminalShortcutStepInput({kind: 'key', key: 'C', modifiers: ['Ctrl']})).toBe('\u0003')
    expect(terminalShortcutStepInput({kind: 'key', key: 'Tab', modifiers: ['Shift']})).toBe('\u001b[Z')
    expect(terminalShortcutStepInput({kind: 'key', key: 'F5', modifiers: ['Ctrl']})).toBe('\u001b[15;5~')
  })

  it('maps a saved key action without modifiers as an unmodified key', () => {
    const savedStep = {kind: 'key', key: 'Enter'} as unknown as TerminalShortcutStep

    expect(terminalShortcutStepInput(savedStep)).toBe('\r')
  })
})

describe('terminal shortcut program scope', () => {
  const scoped: TerminalShortcut = {id: 'scoped', shortcut: 'Shift+Enter', steps: [{kind: 'enter'}], includePrograms: ['codex']}

  it('applies to all terminals when includePrograms is empty or missing', () => {
    const unscoped: TerminalShortcut = {id: 'unscoped', shortcut: 'Shift+Enter', steps: [{kind: 'enter'}]}
    expect(terminalShortcutApplies(unscoped, 'codex')).toBe(true)
    expect(terminalShortcutApplies(unscoped, 'powershell.exe')).toBe(true)
    expect(terminalShortcutApplies(unscoped, undefined)).toBe(true)
    expect(terminalShortcutApplies({...unscoped, includePrograms: []}, 'codex')).toBe(true)
  })

  it('matches the launch command by normalized basename, extension, and case', () => {
    expect(terminalShortcutApplies(scoped, 'codex')).toBe(true)
    expect(terminalShortcutApplies(scoped, 'codex.exe')).toBe(true)
    expect(terminalShortcutApplies(scoped, 'C:\\tools\\codex.exe')).toBe(true)
    expect(terminalShortcutApplies(scoped, '/usr/local/bin/codex')).toBe(true)
    expect(terminalShortcutApplies(scoped, 'CODEX.EXE')).toBe(true)
  })

  it('does not match unrelated programs or missing command', () => {
    expect(terminalShortcutApplies(scoped, 'pwsh.exe')).toBe(false)
    expect(terminalShortcutApplies(scoped, undefined)).toBe(false)
    expect(terminalShortcutApplies(scoped, '')).toBe(false)
  })

  it('uniqueProgramNames normalizes basenames, dedupes case-insensitively, drops empties, and keeps order', () => {
    expect(uniqueProgramNames(['/bin/zsh', 'CODEX.EXE', '', 'codex', 'C:\\tools\\codex.exe', undefined, 'powershell'])).toEqual(['zsh', 'codex', 'powershell'])
    expect(uniqueProgramNames([])).toEqual([])
    expect(uniqueProgramNames(['   ', undefined, ''])).toEqual([])
  })
})
