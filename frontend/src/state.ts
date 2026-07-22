import type {TerminalEvent, TerminalRecord} from './types'

export function applyTerminalEvent(terminals: TerminalRecord[], event: TerminalEvent): TerminalRecord[] {
  return terminals.map((terminal) => {
    if (terminal.id !== event.terminalId || terminal.taskId !== event.taskId) {
      return terminal
    }
    if (event.type === 'output') {
      return {...terminal, output: `${terminal.output ?? ''}${event.data ?? ''}`}
    }
    if (event.type === 'exited') {
      return {...terminal, state: 'exited'}
    }
    return terminal
  })
}
