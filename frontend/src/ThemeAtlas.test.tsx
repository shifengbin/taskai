import {cleanup, render, screen, within} from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import {afterEach, describe, expect, it} from 'vitest'

import ThemeAtlas from './ThemeAtlas'

describe('ThemeAtlas', () => {
  afterEach(cleanup)

  it('renders six named style proposals with all preview states', () => {
    render(<ThemeAtlas/>)

    expect(screen.getAllByRole('article')).toHaveLength(6)
    expect(screen.getByRole('heading', {name: '果汁俱乐部'})).toBeInTheDocument()
    expect(screen.getByRole('heading', {name: '夜市电台'})).toBeInTheDocument()
    expect(screen.getByRole('heading', {name: '拼图总部'})).toBeInTheDocument()
    expect(screen.getByRole('heading', {name: '熔岩赛道'})).toBeInTheDocument()
    expect(screen.getByRole('heading', {name: '泳池派对'})).toBeInTheDocument()
    expect(screen.getByRole('heading', {name: '糖纸档案'})).toBeInTheDocument()
    expect(screen.getAllByRole('button', {name: '亮色'})).toHaveLength(6)
    expect(screen.getAllByRole('button', {name: '暗色'})).toHaveLength(6)
    expect(screen.getAllByRole('button', {name: '弹窗'})).toHaveLength(6)
  })

  it('switches one proposal independently and closes its new-task dialog', async () => {
    const user = userEvent.setup()
    render(<ThemeAtlas/>)

    const juiceHeading = screen.getByRole('heading', {name: '果汁俱乐部'})
    const juiceCard = juiceHeading.closest('article')
    const nightMarketHeading = screen.getByRole('heading', {name: '夜市电台'})
    const nightMarketCard = nightMarketHeading.closest('article')

    if (!juiceCard || !nightMarketCard) {
      throw new Error('未找到主题预览卡片')
    }

    await user.click(within(juiceCard).getByRole('button', {name: '暗色'}))
    expect(within(juiceCard).getByTestId('workspace-preview')).toHaveAttribute('data-mode', 'dark')
    expect(within(nightMarketCard).getByTestId('workspace-preview')).toHaveAttribute('data-mode', 'light')

    await user.click(within(juiceCard).getByRole('button', {name: '弹窗'}))
    expect(screen.getByRole('dialog', {name: '新建任务'})).toBeInTheDocument()

    await user.click(screen.getByRole('button', {name: '取消'}))
    expect(screen.queryByRole('dialog', {name: '新建任务'})).not.toBeInTheDocument()
    expect(within(juiceCard).getByTestId('workspace-preview')).toHaveAttribute('data-mode', 'dark')

    await user.click(within(juiceCard).getByRole('button', {name: '弹窗'}))
    await user.keyboard('{Escape}')
    expect(screen.queryByRole('dialog', {name: '新建任务'})).not.toBeInTheDocument()
  })

  it('opens the matching new-task dialog from a workspace action', async () => {
    const user = userEvent.setup()
    render(<ThemeAtlas/>)

    const poolCard = screen.getByRole('heading', {name: '泳池派对'}).closest('article')
    if (!poolCard) {
      throw new Error('未找到泳池派对卡片')
    }

    await user.click(within(poolCard).getByRole('button', {name: '新建任务'}))
    expect(screen.getByRole('dialog', {name: '新建任务'})).toBeInTheDocument()
  })

  it('places the active dialog above inert proposal content', async () => {
    const user = userEvent.setup()
    render(<ThemeAtlas/>)

    const nightMarketCard = screen.getByRole('heading', {name: '夜市电台'}).closest('article')
    if (!nightMarketCard) {
      throw new Error('未找到夜市电台卡片')
    }

    await user.click(within(nightMarketCard).getByRole('button', {name: '弹窗'}))

    expect(within(nightMarketCard).queryByRole('dialog', {name: '新建任务'})).not.toBeInTheDocument()
    expect(screen.getByRole('dialog', {name: '新建任务'})).toHaveAttribute('data-theme', 'night-market')
    expect(screen.getByTestId('atlas-content')).toHaveAttribute('inert')
  })

  it('keeps the selected color scheme visible behind its dialog', async () => {
    const user = userEvent.setup()
    render(<ThemeAtlas/>)

    const lavaCard = screen.getByRole('heading', {name: '熔岩赛道'}).closest('article')
    if (!lavaCard) {
      throw new Error('未找到熔岩赛道卡片')
    }

    await user.click(within(lavaCard).getByRole('button', {name: '暗色'}))
    await user.click(within(lavaCard).getByRole('button', {name: '弹窗'}))

    expect(within(lavaCard).getByTestId('workspace-preview')).toHaveAttribute('data-mode', 'dark')
    expect(screen.getByRole('dialog', {name: '新建任务'})).toBeInTheDocument()
  })

  it('restores the triggering control focus after Escape', async () => {
    const user = userEvent.setup()
    render(<ThemeAtlas/>)

    const juiceCard = screen.getByRole('heading', {name: '果汁俱乐部'}).closest('article')
    if (!juiceCard) {
      throw new Error('未找到果汁俱乐部卡片')
    }

    const juiceModalButton = within(juiceCard).getByRole('button', {name: '弹窗'})
    await user.click(juiceModalButton)

    expect(screen.getByRole('dialog', {name: '新建任务'})).toBeInTheDocument()

    await user.keyboard('{Escape}')
    expect(screen.queryByRole('dialog', {name: '新建任务'})).not.toBeInTheDocument()
    expect(juiceModalButton).toHaveFocus()
  })

  it('cycles keyboard focus within the active dialog', async () => {
    const user = userEvent.setup()
    render(<ThemeAtlas/>)

    const puzzleCard = screen.getByRole('heading', {name: '拼图总部'}).closest('article')
    if (!puzzleCard) {
      throw new Error('未找到拼图总部卡片')
    }

    await user.click(within(puzzleCard).getByRole('button', {name: '弹窗'}))
    const dialog = screen.getByRole('dialog', {name: '新建任务'})
    const titleField = within(dialog).getByRole('textbox', {name: '任务标题'})
    const closeButton = within(dialog).getByRole('button', {name: '关闭新建任务'})
    const saveButton = within(dialog).getByRole('button', {name: /保存任务/})

    expect(titleField).toHaveFocus()
    await user.tab({shift: true})
    expect(closeButton).toHaveFocus()
    await user.tab({shift: true})
    expect(saveButton).toHaveFocus()
  })
})
