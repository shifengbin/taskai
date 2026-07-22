import {describe, expect, it} from 'vitest'

import {applyTerminalEvent} from './state'
import {clampTaskTreeWidth, type TerminalRecord} from './types'

describe('终端状态路由', () => {
  it('只将输出和退出状态应用到对应终端', () => {
    const terminals: TerminalRecord[] = [
      {id: 'one', taskId: 'task-a', state: 'active'},
      {id: 'two', taskId: 'task-b', state: 'active'},
    ]
    const afterOutput = applyTerminalEvent(terminals, {
      taskId: 'task-a', terminalId: 'one', type: 'output', data: 'hello',
    })
    const afterExit = applyTerminalEvent(afterOutput, {taskId: 'task-b', terminalId: 'two', type: 'exited'})

    expect(afterExit).toEqual([
      {id: 'one', taskId: 'task-a', state: 'active', output: 'hello'},
      {id: 'two', taskId: 'task-b', state: 'exited'},
    ])
  })

  it('拖拽宽度遵循最小值和右侧最小空间', () => {
    expect(clampTaskTreeWidth(120, 1000)).toBe(280)
    expect(clampTaskTreeWidth(900, 1000)).toBe(640)
  })
})
