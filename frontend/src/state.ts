import type {LifecycleExecution, LifecycleExecutionWatermark, RealtimeStatusEvent, TaskRecord, TerminalEvent, TerminalRecord} from './types'
import {
  createTerminalTitleParserState,
  parseTerminalTitleOutput,
  type TerminalTitleParserState,
} from './terminal-title'

export interface PendingTerminalEvent {
  title?: string
  state?: TerminalRecord['state']
  realtimeStatus?: TerminalRecord['realtimeStatus']
}

export interface RealtimeStatusUpdate {
  tasks: TaskRecord[]
  terminals: TerminalRecord[]
}

export function mergeLifecycleTask(tasks: TaskRecord[], incoming: TaskRecord): TaskRecord[] {
  return tasks.map((current) => current.id === incoming.id ? mergeLifecycleTaskRecord(current, incoming) : current)
}

function mergeLifecycleTaskRecord(current: TaskRecord, incoming: TaskRecord): TaskRecord {
  const currentExecution = current.lifecycleExecution
  const incomingExecution = incoming.lifecycleExecution
  const currentWatermark = lifecycleExecutionWatermark(current)
  if (!incomingExecution || !incomingExecution.runId) {
    return currentWatermark ? {...incoming, lifecycleExecutionWatermark: currentWatermark} : incoming
  }
  const incomingWatermark = watermarkFromExecution(incomingExecution)
  if (!currentWatermark) {
    return {...incoming, lifecycleExecutionWatermark: incomingWatermark}
  }
  if (currentWatermark.runId === incomingWatermark.runId) {
    if (currentWatermark.revision > incomingWatermark.revision || (!currentExecution && currentWatermark.revision === incomingWatermark.revision)) {
      return current
    }
    return {...incoming, lifecycleExecutionWatermark: incomingWatermark}
  }
  if (currentExecution?.state === 'running') {
    // 仅允许生命周期的正常前置钩子到后置钩子切换，避免旧绑定返回覆盖新运行。
    if (incomingExecution.state === 'failed' || (incomingExecution.state === 'running' && !isLifecycleHookSuccessor(currentExecution, incomingExecution))) {
      return current
    }
  }
  return {...incoming, lifecycleExecutionWatermark: incomingWatermark}
}

function isLifecycleHookSuccessor(current: LifecycleExecution, incoming: LifecycleExecution): boolean {
  return (current.hook === 'beforeStart' && incoming.hook === 'postStart') ||
    (current.hook === 'beforeEnd' && incoming.hook === 'postEnd')
}

function lifecycleExecutionWatermark(task: TaskRecord): LifecycleExecutionWatermark | undefined {
  if (task.lifecycleExecution?.runId) {
    return watermarkFromExecution(task.lifecycleExecution)
  }
  return task.lifecycleExecutionWatermark
}

function watermarkFromExecution(execution: LifecycleExecution): LifecycleExecutionWatermark {
  return {runId: execution.runId ?? '', revision: execution.revision ?? 0}
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
      return title !== undefined ? {...terminal, title} : terminal
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
    if (title !== undefined) {
      pendingEvents.set(key, {...current, title})
    }
    return
  }
  if (event.type === 'exited') {
    pendingEvents.set(key, {...current, state: 'exited'})
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
