import {render, screen} from '@testing-library/react'
import {describe, expect, it} from 'vitest'

import {TerminalStatusDot} from './TerminalStatusDot'

describe('TerminalStatusDot', () => {
  it('为工作中和未读状态标识扩散动效', () => {
    const {rerender} = render(<TerminalStatusDot status="working"/>)
    expect(screen.getByRole('status', {name: '终端状态：工作中'})).toHaveAttribute('data-pulse', 'active')

    rerender(<TerminalStatusDot status="unread"/>)
    expect(screen.getByRole('status', {name: '终端状态：未读'})).toHaveAttribute('data-pulse', 'active')
  })

  it('不为静态状态标识扩散动效', () => {
    const {rerender} = render(<TerminalStatusDot status="idle"/>)
    expect(screen.getByRole('status', {name: '终端状态：空闲'})).not.toHaveAttribute('data-pulse')

    rerender(<TerminalStatusDot status="error"/>)
    expect(screen.getByRole('status', {name: '终端状态：异常'})).not.toHaveAttribute('data-pulse')
  })
})
