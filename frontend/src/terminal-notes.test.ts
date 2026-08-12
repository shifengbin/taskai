import {describe, expect, it} from 'vitest'

import {appendTerminalNote, clearTerminalNotes, clearTaskTerminalNotes, formatTerminalNotes, terminalNotesForSession} from './terminal-notes'
import {terminalEventKey} from './state'

const template = {originalPrefix: '原文：', notePrefix: '备注：', listSuffix: '请处理'}

describe('terminal notes', () => {
	it('按创建顺序格式化记录并自动添加换行和统一后缀', () => {
		expect(formatTerminalNotes([
			{original: '一', note: '甲'},
			{original: '二', note: '乙'},
		], template)).toBe('原文：一\n备注：甲\n原文：二\n备注：乙\n请处理\n')
	})

	it('统一后缀为空时仍保留结尾换行', () => {
		expect(formatTerminalNotes([{original: '一', note: '甲'}], {...template, listSuffix: ''})).toBe('原文：一\n备注：甲\n\n')
	})

	it('仅清除指定终端会话的备注', () => {
		const firstKey = terminalEventKey('task-1', 'terminal-1')
		const secondKey = terminalEventKey('task-1', 'terminal-2')
		const notes = appendTerminalNote(appendTerminalNote({}, firstKey, {original: '一', note: '甲'}), secondKey, {original: '二', note: '乙'})

		const cleared = clearTerminalNotes(notes, firstKey)

		expect(terminalNotesForSession(cleared, firstKey)).toEqual([])
		expect(terminalNotesForSession(cleared, secondKey)).toEqual([{original: '二', note: '乙'}])
	})

	it('清除任务时保留其他任务终端的备注', () => {
		const firstKey = terminalEventKey('task-1', 'terminal-1')
		const secondKey = terminalEventKey('task-1', 'terminal-2')
		const otherTaskKey = terminalEventKey('task-2', 'terminal-1')
		const notes = appendTerminalNote(
			appendTerminalNote(
				appendTerminalNote({}, firstKey, {original: '一', note: '甲'}),
				secondKey,
				{original: '二', note: '乙'},
			),
			otherTaskKey,
			{original: '三', note: '丙'},
		)

		const cleared = clearTaskTerminalNotes(notes, 'task-1')

		expect(terminalNotesForSession(cleared, firstKey)).toEqual([])
		expect(terminalNotesForSession(cleared, secondKey)).toEqual([])
		expect(terminalNotesForSession(cleared, otherTaskKey)).toEqual([{original: '三', note: '丙'}])
	})
})
