import {useCallback, useEffect, useRef, useState} from 'react'

import {themeConcepts, type PreviewMode, type ThemeConcept} from './theme-atlas-data'
import './theme-atlas.css'

const previewModeLabels: Record<PreviewMode, string> = {
  light: '亮色',
  dark: '暗色',
  modal: '弹窗',
}

type WorkspaceMode = Exclude<PreviewMode, 'modal'>

export default function ThemeAtlas() {
  const [modes, setModes] = useState<Record<string, WorkspaceMode>>({})
  const [activeDialogThemeID, setActiveDialogThemeID] = useState<string>()
  const dialogRef = useRef<HTMLElement>(null)
  const dialogTriggerRef = useRef<HTMLElement>()
  const atlasContentRef = useRef<HTMLDivElement>(null)

  const closeDialog = useCallback(() => {
    setActiveDialogThemeID(undefined)
  }, [])

  useEffect(() => {
    atlasContentRef.current?.toggleAttribute('inert', Boolean(activeDialogThemeID))
  }, [activeDialogThemeID])

  useEffect(() => {
    if (!activeDialogThemeID) {
      dialogTriggerRef.current?.focus()
      dialogTriggerRef.current = undefined
      return
    }

    const dialog = dialogRef.current
    const titleInput = dialog?.querySelector<HTMLElement>('input')
    titleInput?.focus()
    const previousOverflow = document.body.style.overflow
    document.body.style.overflow = 'hidden'

    const trapFocus = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        event.preventDefault()
        closeDialog()
        return
      }

      if (event.key !== 'Tab' || !dialog) {
        return
      }

      const focusableElements = [...dialog.querySelectorAll<HTMLElement>('button:not([disabled]), input:not([disabled]), textarea:not([disabled])')]
      const firstFocusableElement = focusableElements[0]
      const lastFocusableElement = focusableElements.at(-1)
      if (!firstFocusableElement || !lastFocusableElement) {
        return
      }

      if (event.shiftKey && document.activeElement === firstFocusableElement) {
        event.preventDefault()
        lastFocusableElement.focus()
      } else if (!event.shiftKey && document.activeElement === lastFocusableElement) {
        event.preventDefault()
        firstFocusableElement.focus()
      } else if (!dialog.contains(document.activeElement)) {
        event.preventDefault()
        firstFocusableElement.focus()
      }
    }
    window.addEventListener('keydown', trapFocus)
    return () => {
      document.body.style.overflow = previousOverflow
      window.removeEventListener('keydown', trapFocus)
    }
  }, [activeDialogThemeID, closeDialog])

  const openDialog = (themeID: string, trigger: HTMLElement) => {
    dialogTriggerRef.current = trigger
    setActiveDialogThemeID(themeID)
  }

  const activeDialogTheme = themeConcepts.find((theme) => theme.id === activeDialogThemeID)
  const activeDialogMode = activeDialogTheme ? modes[activeDialogTheme.id] ?? 'light' : 'light'

  return <main className="theme-atlas">
    <div data-testid="atlas-content" ref={atlasContentRef}>
      <header className="atlas-intro">
        <p className="atlas-kicker">TaskAI / 视觉图鉴</p>
        <h1>六套大胆的任务工作台</h1>
        <p>每种风格均提供亮色、暗色与新建任务弹窗状态。选择一套，把忙碌的开发日常变成值得收藏的工作台。</p>
      </header>

      <nav aria-label="主题导航" className="atlas-nav">
        {themeConcepts.map((theme, index) => <a href={`#${theme.id}`} key={theme.id}><span>0{index + 1}</span>{theme.name}</a>)}
      </nav>

      <section aria-label="风格提案" className="theme-proposals">
        {themeConcepts.map((theme, index) => {
          const mode = modes[theme.id] ?? 'light'
          const dialogOpen = activeDialogThemeID === theme.id
          return <article className="theme-proposal" data-theme={theme.id} id={theme.id} key={theme.id}>
            <header className="proposal-meta">
              <p className="proposal-number">方案 / 0{index + 1}</p>
              <div>
                <p className="proposal-tagline">{theme.tagline}</p>
                <h2>{theme.name}</h2>
              </div>
              <div aria-label={`${theme.name} 配色`} className="color-swatches">
                {theme.colors.map((color) => <span key={color} style={{backgroundColor: color}}/>)}
              </div>
            </header>

            <div aria-label={`${theme.name} 预览状态`} className="preview-switcher">
              {(['light', 'dark', 'modal'] as const).map((previewMode) => <button
                aria-pressed={previewMode === 'modal' ? dialogOpen : mode === previewMode}
                key={previewMode}
                onClick={(event) => {
                  if (previewMode === 'modal') {
                    openDialog(theme.id, event.currentTarget)
                    return
                  }
                  setModes((current) => ({...current, [theme.id]: previewMode}))
                }}
                type="button"
              >
                {previewModeLabels[previewMode]}
              </button>)}
            </div>

            <WorkspacePreview mode={mode} onOpenDialog={(trigger) => openDialog(theme.id, trigger)} theme={theme}/>
          </article>
        })}
      </section>
    </div>
    {activeDialogTheme && <NewTaskDialog closeDialog={closeDialog} dialogRef={dialogRef} mode={activeDialogMode} theme={activeDialogTheme}/>}
  </main>
}

function WorkspacePreview({mode, onOpenDialog, theme}: {mode: WorkspaceMode, onOpenDialog: (trigger: HTMLElement) => void, theme: ThemeConcept}) {

  return <div className="workspace-preview" data-mode={mode} data-testid="workspace-preview" data-theme={theme.id}>
    <div className="preview-appbar">
      <div className="brand-mark"><span>✦</span>task<span>ai</span></div>
      <div className="appbar-actions"><button aria-label="打开额外信息" type="button">◎</button><button aria-label="打开设置" type="button">⚙</button></div>
    </div>

    <div className="preview-workspace">
      <aside aria-label="任务树" className="preview-sidebar">
        <div className="sidebar-heading"><span>任务与终端</span><button aria-label="新建任务" onClick={(event) => onOpenDialog(event.currentTarget)} type="button">＋</button></div>
        <div className="status-tabs"><span className="is-active">待办 <b>04</b></span><span>进行中 <b>02</b></span><span>完成 <b>18</b></span></div>
        <div className="task-list">
          <button className="task-row selected" type="button"><i/><span>{theme.task}</span><b>···</b></button>
          <button className="task-row" type="button"><i/><span>整理交付备注</span><b>···</b></button>
          <button className="task-row" type="button"><i/><span>同步接口契约</span><b>···</b></button>
          <p className="shelved-label">已搁置 / 02</p>
          <button className="task-row is-shelved" type="button"><i/><span>留给明天的灵感</span></button>
        </div>
        <div className="sidebar-bottom"><span className="status-dot is-working"/>所有状态已同步</div>
      </aside>

      <section aria-label="任务详情" className="preview-content">
        <header className="task-toolbar"><span className="crumb">任务 / 待办</span><button type="button">任务操作 <b>⌄</b></button></header>
        <div className="task-content">
          <div className="task-title-row"><span className="task-token">✦</span><div><p>今天的焦点</p><h3>{theme.task}</h3></div><span className="task-status">准备就绪</span></div>
          <div className="task-summary"><p>把待处理内容切成清晰步骤，先完成最有能量的一件。</p><div><span>发布</span><span>设计系统</span><span>本周</span></div></div>
          <div className="detail-grid"><div><span>工作目录</span><code>~/workspaces/release-spark</code></div><div><span>任务模板</span><strong>产品发布</strong></div><div><span>开始时间</span><strong>今天 10:30</strong></div></div>
          <div className="terminal-card" aria-label="终端区域"><div className="terminal-title"><span><i className="status-dot is-working"/> {theme.terminal}</span><b>＋</b></div><pre><code><span>$</span> {theme.terminal}{'\n'}<em>✓ workspace is ready</em>{'\n'}<span>$</span> _</code></pre></div>
        </div>
      </section>
    </div>

  </div>
}

function NewTaskDialog({closeDialog, dialogRef, mode, theme}: {closeDialog: () => void, dialogRef: React.RefObject<HTMLElement>, mode: WorkspaceMode, theme: ThemeConcept}) {
  return <div className="atlas-dialog-backdrop" data-mode={mode} data-theme={theme.id} onClick={closeDialog}>
    <section aria-labelledby={`${theme.id}-new-task-title`} aria-modal="true" className="preview-dialog" data-theme={theme.id} onClick={(event) => event.stopPropagation()} ref={dialogRef} role="dialog">
      <div className="dialog-heading"><div><p>新的小冒险</p><h3 id={`${theme.id}-new-task-title`}>新建任务</h3></div><button aria-label="关闭新建任务" onClick={closeDialog} type="button">×</button></div>
      <label>任务标题<input autoFocus defaultValue={theme.task}/></label>
      <label>任务描述<textarea defaultValue="把这件事做得清晰、轻快，并且漂亮。" rows={2}/></label>
      <div className="dialog-color-row"><span>任务颜色</span><div><button aria-label="选择珊瑚色" className="color-choice is-selected" type="button"/><button aria-label="选择蓝色" className="color-choice" type="button"/><button aria-label="选择黄色" className="color-choice" type="button"/></div></div>
      <div className="dialog-actions"><button onClick={closeDialog} type="button">取消</button><button type="button">保存任务 <span>↗</span></button></div>
    </section>
  </div>
}
