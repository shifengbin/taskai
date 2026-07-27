import { describe, expect, it } from 'vitest'
import { createTerminalTitleParserState, parseTerminalTitleOutput } from './terminal-title'

describe('parseTerminalTitleOutput', () => {
  it('解析单个输出片段中的 OSC 0 和 BEL 标题', () => {
    const result = parseTerminalTitleOutput(createTerminalTitleParserState(), '构建开始\x1b]0;正在编译\x07构建继续')

    expect(result.title).toBe('正在编译')
  })

  it('解析单个输出片段中的 OSC 2 和 ST 标题', () => {
    const result = parseTerminalTitleOutput(createTerminalTitleParserState(), '\x1b]2;任务运行中\x1b\\')

    expect(result.title).toBe('任务运行中')
  })

  it('解析由 8 位 C1 ST 终止的标题', () => {
    const result = parseTerminalTitleOutput(createTerminalTitleParserState(), '\x1b]0;真实标题\x9c')

    expect(result.title).toBe('真实标题')
    expect(result.state).toEqual({phase: 'text'})
  })

  it('让 C1 ST 终止忽略和超长丢弃状态', () => {
    const ignored = parseTerminalTitleOutput(createTerminalTitleParserState(), '\x1b]1;窗口图标\x9c')
    const discarded = parseTerminalTitleOutput(createTerminalTitleParserState(), `\x1b]0;${'x'.repeat(10_000)}\x9c`)

    expect(ignored.state).toEqual({phase: 'text'})
    expect(discarded.state).toEqual({phase: 'text'})
  })

  it('返回同一输出片段中最后一个完整标题', () => {
    const result = parseTerminalTitleOutput(createTerminalTitleParserState(), '\x1b]0;准备\x07日志\x1b]2;完成\x1b\\')

    expect(result.title).toBe('完成')
  })

  it('在标题序列跨多个输出片段时恢复解析', () => {
    const first = parseTerminalTitleOutput(createTerminalTitleParserState(), '日志\x1b]0;编')
    const second = parseTerminalTitleOutput(first.state, '译中\x1b')
    const third = parseTerminalTitleOutput(second.state, '\\后续日志')

    expect(first.title).toBeUndefined()
    expect(second.title).toBeUndefined()
    expect(third.title).toBe('编译中')
  })

  it('在 ESC 与 ] 分别落在输出片段时恢复解析', () => {
    const first = parseTerminalTitleOutput(createTerminalTitleParserState(), '日志\x1b')
    const second = parseTerminalTitleOutput(first.state, ']2;分片标题\x07')

    expect(first.title).toBeUndefined()
    expect(second.title).toBe('分片标题')
  })

  it('限制超长标题并继续解析后续标题', () => {
    const discarded = parseTerminalTitleOutput(createTerminalTitleParserState(), `\x1b]0;${'x'.repeat(10_000)}`)
    const recovered = parseTerminalTitleOutput(discarded.state, '\x1b]2;恢复标题\x07')

    expect(discarded.title).toBeUndefined()
    expect(recovered.title).toBe('恢复标题')
  })

  it('在同一输出片段中丢弃超长未终止 OSC 并解析后续标题', () => {
    const result = parseTerminalTitleOutput(createTerminalTitleParserState(), `\x1b]0;${'x'.repeat(10_000)}\x1b]2;真实标题\x07`)

    expect(result.title).toBe('真实标题')
  })

  it('在下一输出片段中丢弃超长未终止 OSC 并解析后续标题', () => {
    const first = parseTerminalTitleOutput(createTerminalTitleParserState(), `\x1b]0;${'x'.repeat(4_000)}`)
    const result = parseTerminalTitleOutput(first.state, `${'x'.repeat(100)}\x1b]2;真实标题\x07`)

    expect(result.title).toBe('真实标题')
  })

  it('在超长旧 OSC 后跨事件完成新标题', () => {
    const first = parseTerminalTitleOutput(createTerminalTitleParserState(), `\x1b]0;${'x'.repeat(4_000)}`)
    const second = parseTerminalTitleOutput(first.state, `${'x'.repeat(100)}\x1b]2;真实标题`)
    const third = parseTerminalTitleOutput(second.state, '\x07')

    expect(third.title).toBe('真实标题')
  })

  it('在超长旧 OSC 后使用最后一个中断前序列的完整标题', () => {
    const discarded = parseTerminalTitleOutput(createTerminalTitleParserState(), `\x1b]0;${'x'.repeat(10_000)}`)
    const result = parseTerminalTitleOutput(discarded.state, '\x1b]0;中间\x1b]2;真实标题\x07')

    expect(result.title).toBe('真实标题')
  })

  it('不将其他 OSC 序列识别为终端标题', () => {
    const result = parseTerminalTitleOutput(createTerminalTitleParserState(), '\x1b]1;窗口图标\x07\x1b]3;其他数据\x1b\\')

    expect(result.title).toBeUndefined()
  })
})
