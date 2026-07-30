import {describe, expect, it} from 'vitest'

import {settings as wailsSettings} from '../wailsjs/go/models'
import {clampTaskTreeWidth} from './types'

describe('clampTaskTreeWidth', () => {
  it('将拖拽产生的小数宽度归一化为整数', () => {
    expect(clampTaskTreeWidth(311.25390625, 1280)).toBe(311)
  })

  it('保留生命周期命令引用参数和系统使用说明', () => {
    const command = wailsSettings.LifecycleCommand.createFrom({
      id: 'system.lifecycle.git-clone',
      kind: 'git-clone',
      name: 'Git 仓库克隆',
      arguments: [],
      documentation: '参数：dir=<相对目录>（必填）',
      applicableHooks: ['beforeStart', 'beforeEnd', 'updateTask'],
    })
    const chain = wailsSettings.LifecycleCommandChain.createFrom({
      id: 'clone-repositories',
      name: '克隆仓库',
      commands: [{commandId: command.id, arguments: ['dir=repositories']}],
      applicableHooks: ['beforeStart'],
    })

    expect(command.documentation).toContain('dir=<相对目录>')
    expect(chain.commands).toEqual([{commandId: 'system.lifecycle.git-clone', arguments: ['dir=repositories']}])
  })
})
