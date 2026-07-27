import type {RealtimeStatusEvent, TaskRecord, TerminalEvent, TerminalRecord} from './types'
import {
  createTerminalTitleParserState,
  parseTerminalTitleOutput,
  type TerminalTitleParserState,
} from './terminal-title'

export interface PendingTerminalEvent {
  output: string
  title?: string
  state?: TerminalRecord['state']
  realtimeStatus?: TerminalRecord['realtimeStatus']
}

export interface RealtimeStatusUpdate {
  tasks: TaskRecord[]
  terminals: TerminalRecord[]
}

export function applyRealtimeStatusEvent(
  tasks: TaskRecord[],
  terminals: TerminalRecord[],
  event: RealtimeStatusEvent,
): RealtimeStatusUpdate {
  return {
    tasks: applyRealtimeStatusToTasks(tasks, event),
    terminals: applyRealtimeStatusToTerminals(terminals, event),
  }
}

export function applyRealtimeStatusToTasks(tasks: TaskRecord[], event: RealtimeStatusEvent): TaskRecord[] {
  return tasks.map((task) => task.id === event.taskId ? {...task, realtimeStatus: event.taskStatus} : task)
}

export function applyRealtimeStatusToTerminals(terminals: TerminalRecord[], event: RealtimeStatusEvent): TerminalRecord[] {
  if (!event.terminalId) {
    return terminals
  }
  if (event.terminalRemoved) {
    return terminals.filter((terminal) => terminal.taskId !== event.taskId || terminal.id !== event.terminalId)
  }
  if (!event.terminalStatus) {
    return terminals
  }
  return terminals.map((terminal) => (
    terminal.taskId === event.taskId && terminal.id === event.terminalId
      ? {...terminal, realtimeStatus: event.terminalStatus}
      : terminal
  ))
}

export function shouldReportTerminalTitleActivity(terminal: Pick<TerminalRecord, 'title'>, title: string | undefined): boolean {
  return title !== undefined && terminal.title !== title
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

export function bufferPendingRealtimeStatusEvent(
  pendingEvents: Map<string, PendingTerminalEvent>,
  event: RealtimeStatusEvent,
): void {
  if (!event.terminalId || !event.terminalStatus || event.terminalRemoved) {
    return
  }
  const key = terminalEventKey(event.taskId, event.terminalId)
  const current = pendingEvents.get(key)
  pendingEvents.set(key, {
    ...current,
    output: current?.output ?? '',
    realtimeStatus: event.terminalStatus,
  })
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
    ...(pending?.realtimeStatus !== undefined ? {realtimeStatus: pending.realtimeStatus} : {}),
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
