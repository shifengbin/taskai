import {render, screen} from '@testing-library/react'
import {describe, expect, it} from 'vitest'

import {Dialog, DialogContent, DialogTitle} from './dialog'

const dialogAnimationClasses = [
  'data-[state=open]:animate-in',
  'data-[state=closed]:animate-out',
  'data-[state=open]:fade-in-0',
  'data-[state=closed]:fade-out-0',
  'data-[state=open]:zoom-in-95',
  'data-[state=closed]:zoom-out-95',
]

describe('Dialog', () => {
  it('不为遮罩和内容添加打开或关闭动画', () => {
    render(
      <Dialog open>
        <DialogContent>
          <DialogTitle>新建任务</DialogTitle>
        </DialogContent>
      </Dialog>,
    )

    const content = screen.getByRole('dialog', {name: '新建任务'})
    const overlay = document.querySelector<HTMLElement>('[data-state="open"].fixed.inset-0')

    expect(overlay).not.toBeNull()
    for (const animationClass of dialogAnimationClasses) {
      expect(content).not.toHaveClass(animationClass)
      expect(overlay).not.toHaveClass(animationClass)
    }
  })

  it('使用不透明 Nebula 弹窗表面、细边框和柔和阴影', () => {
    render(
      <Dialog open>
        <DialogContent>
          <DialogTitle>新建任务</DialogTitle>
        </DialogContent>
      </Dialog>,
    )

    const content = screen.getByRole('dialog', {name: '新建任务'})

    expect(content).toHaveClass('border', 'rounded-snap', 'bg-snap-overlay', 'shadow-snap-lg')
    expect(content).not.toHaveClass('bg-snap-surface')
    expect(content).not.toHaveClass('border-2')
  })
})
