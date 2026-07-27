import type {TerminalEvent, TerminalRecord} from './types'
import {
  createTerminalTitleParserState,
  parseTerminalTitleOutput,
  type TerminalTitleParserState,
} from './terminal-title'

export interface PendingTerminalEvent {
  output: string
  title?: string
  state?: TerminalRecord['state']
}

export function applyTerminalEvent(terminals: TerminalRecord[], event: TerminalEvent, title?: string): TerminalRecord[] {
  return terminals.map((terminal) => {
    if (terminal.id !== event.terminalId || terminal.taskId !== event.taskId) {
      return terminal
    }
    if (event.type === 'output') {
      return {
        ...terminal,
        output: `${terminal.output ?? ''}${event.data ?? ''}`,
        ...(title !== undefined ? {title} : {}),
      }
    }
    if (event.type === 'exited') {
      return {...terminal, state: 'exited'}
    }
    return terminal
  })
}

export function parseTerminalEventTitle(
  parserStates: Map<string, TerminalTitleParserState>,
  event: TerminalEvent,
): string | undefined {
  const key = terminalEventKey(event.taskId, event.terminalId)
  if (event.type === 'exited') {
    parserStates.delete(key)
    return undefined
  }
  if (event.type !== 'output') {
    return undefined
  }
  const result = parseTerminalTitleOutput(parserStates.get(key) ?? createTerminalTitleParserState(), event.data ?? '')
  parserStates.set(key, result.state)
  return result.title
}

export function terminalEventKey(taskID: string, terminalID: string): string {
  return JSON.stringify([taskID, terminalID])
}

export function registerTerminal(registeredTerminalKeys: Set<string>, terminal: TerminalRecord): void {
  registeredTerminalKeys.add(terminalEventKey(terminal.taskId, terminal.id))
}

export function bufferPendingTerminalEvent(
  pendingEvents: Map<string, PendingTerminalEvent>,
  event: TerminalEvent,
  title?: string,
): void {
  const key = terminalEventKey(event.taskId, event.terminalId)
  const current = pendingEvents.get(key)
  if (event.type === 'output') {
    pendingEvents.set(key, {
      ...current,
      output: `${current?.output ?? ''}${event.data ?? ''}`,
      ...(title !== undefined ? {title} : {}),
    })
    return
  }
  if (event.type === 'exited') {
    pendingEvents.set(key, {...current, output: current?.output ?? '', state: 'exited'})
  }
}

export function mergePendingTerminalEvents(
  pendingEvents: Map<string, PendingTerminalEvent>,
  terminal: TerminalRecord,
): TerminalRecord {
  const key = terminalEventKey(terminal.taskId, terminal.id)
  const pending = pendingEvents.get(key)
  pendingEvents.delete(key)
  return {
    ...terminal,
    output: `${terminal.output ?? ''}${pending?.output ?? ''}`,
    ...(pending?.title !== undefined ? {title: pending.title} : {}),
    ...(pending?.state !== undefined ? {state: pending.state} : {}),
  }
}

export function clearTaskTerminalTracking(
  taskID: string,
  parserStates: Map<string, TerminalTitleParserState>,
  pendingEvents: Map<string, PendingTerminalEvent>,
  registeredTerminalKeys: Set<string>,
): void {
  const prefix = `[${JSON.stringify(taskID)},`
  for (const collection of [parserStates, pendingEvents, registeredTerminalKeys]) {
    for (const key of collection.keys()) {
      if (key.startsWith(prefix)) {
        collection.delete(key)
      }
    }
  }
}
