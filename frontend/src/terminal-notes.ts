import type {TerminalNoteTemplate} from './types'

export interface TerminalNote {
	original: string
	note: string
}

export type TerminalNotesBySession = Record<string, TerminalNote[]>

export const defaultTerminalNoteTemplate: TerminalNoteTemplate = {
	originalPrefix: '原文：',
	notePrefix: '备注：',
	listSuffix: '',
}

export function formatTerminalNotes(notes: TerminalNote[], template: TerminalNoteTemplate): string {
	return `${notes.map(({original, note}) => `${template.originalPrefix}${original}\n${template.notePrefix}${note}`).join('\n')}\n${template.listSuffix}\n`
}

export function terminalNotesForSession(notesBySession: TerminalNotesBySession, sessionKey: string): TerminalNote[] {
	return notesBySession[sessionKey] ?? []
}

export function appendTerminalNote(notesBySession: TerminalNotesBySession, sessionKey: string, note: TerminalNote): TerminalNotesBySession {
	return {...notesBySession, [sessionKey]: [...terminalNotesForSession(notesBySession, sessionKey), note]}
}

export function clearTerminalNotes(notesBySession: TerminalNotesBySession, sessionKey: string): TerminalNotesBySession {
	const {[sessionKey]: _notes, ...remaining} = notesBySession
	return remaining
}

export function clearTaskTerminalNotes(notesBySession: TerminalNotesBySession, taskID: string): TerminalNotesBySession {
	return Object.fromEntries(Object.entries(notesBySession).filter(([sessionKey]) => terminalNoteTaskID(sessionKey) !== taskID))
}

function terminalNoteTaskID(sessionKey: string): string | undefined {
	try {
		const [taskID] = JSON.parse(sessionKey) as [unknown]
		return typeof taskID === 'string' ? taskID : undefined
	} catch {
		return undefined
	}
}
