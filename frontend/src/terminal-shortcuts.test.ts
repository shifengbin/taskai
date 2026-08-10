import {describe, expect, it} from 'vitest'

import {findTerminalShortcut, keyboardEventShortcut, keyboardEventStep, terminalShortcutInput, terminalShortcutStepInput} from './terminal-shortcuts'
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
