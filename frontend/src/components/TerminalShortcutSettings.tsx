import {Plus, Trash2} from 'lucide-react'

import {keyboardEventShortcut, keyboardEventStep} from '../terminal-shortcuts'
import {type TerminalShortcut, type TerminalShortcutStep} from '../types'
import {Button, Field, Input} from './ui'

interface TerminalShortcutSettingsProps {
  shortcuts: TerminalShortcut[]
  onChange(shortcuts: TerminalShortcut[]): void
}

export function TerminalShortcutSettings({shortcuts, onChange}: TerminalShortcutSettingsProps) {
  const updateShortcut = (id: string, update: Partial<TerminalShortcut>) => {
    onChange(shortcuts.map((shortcut) => shortcut.id === id ? {...shortcut, ...update} : shortcut))
  }

  const addShortcut = () => {
    onChange([...shortcuts, {id: newShortcutID(), shortcut: '', steps: [{kind: 'text', text: '\\'}, {kind: 'key', key: 'Enter', modifiers: []}]}])
  }

  return <div className="grid gap-3">
    <div className="flex items-start justify-between gap-3">
      <div className="grid gap-1">
        <span className="font-display text-sm font-bold text-snap-ink">终端快捷键</span>
        <span className="text-sm text-snap-muted">只在当前聚焦的活动终端生效。文本动作按原样写入，按键动作发送对应的终端按键输入。</span>
      </div>
      <Button variant="primary" size="sm" onClick={addShortcut}><Plus className="mr-1 h-4 w-4"/>新增快捷键</Button>
    </div>
    {shortcuts.length === 0 && <div className="rounded-snap border-2 border-dashed border-snap-outline px-3 py-5 text-center text-sm text-snap-muted">尚未配置快捷键</div>}
    <div className="grid gap-3">
      {shortcuts.map((shortcut, index) => <ShortcutCard
        key={shortcut.id}
        shortcut={shortcut}
        index={index}
        onChange={(update) => updateShortcut(shortcut.id, update)}
        onDelete={() => onChange(shortcuts.filter((candidate) => candidate.id !== shortcut.id))}
      />)}
    </div>
  </div>
}

function ShortcutCard({shortcut, index, onChange, onDelete}: {shortcut: TerminalShortcut, index: number, onChange(update: Partial<TerminalShortcut>): void, onDelete(): void}) {
  const updateStep = (stepIndex: number, step: TerminalShortcutStep) => {
    onChange({steps: shortcut.steps.map((current, currentIndex) => currentIndex === stepIndex ? step : current)})
  }
  const removeStep = (stepIndex: number) => {
    onChange({steps: shortcut.steps.filter((_, currentIndex) => currentIndex !== stepIndex)})
  }

  return <div className="grid gap-3 rounded-snap border-2 border-snap-outline bg-snap-surface p-3 shadow-snap-sm" data-testid={`terminal-shortcut-${index}`}>
    <div className="flex items-center justify-between gap-3">
      <span className="text-sm font-bold text-snap-ink">快捷键 {index + 1}</span>
      <Button variant="ghost" size="sm" onClick={onDelete}><Trash2 className="mr-1 h-4 w-4"/>删除</Button>
    </div>
    <Field label="按键组合" hint="点击输入框后按下组合键，例如 Shift+Enter。">
      <ShortcutCapture value={shortcut.shortcut} onChange={(value) => onChange({shortcut: value})}/>
    </Field>
    <Field label="输入动作" hint="按从上到下的顺序执行。">
      <div className="grid gap-2">
        {shortcut.steps.map((step, stepIndex) => {
          const normalizedStep = step.kind === 'enter'
            ? {kind: 'key' as const, key: 'Enter', modifiers: []}
            : step.kind === 'key'
              ? {...step, modifiers: step.modifiers ?? []}
              : step
          return <div key={`${shortcut.id}-${stepIndex}`} className="flex items-center gap-2">
            <select
              className="w-28 shrink-0 rounded-snap border-2 border-snap-outline bg-snap-surface px-2 py-2 text-sm text-snap-ink outline-none focus-visible:border-snap-cobalt"
              aria-label={`快捷键 ${index + 1} 动作 ${stepIndex + 1} 类型`}
              value={normalizedStep.kind}
              onChange={(event) => updateStep(stepIndex, event.target.value === 'key' ? {kind: 'key', key: normalizedStep.kind === 'key' ? normalizedStep.key : 'Enter', modifiers: normalizedStep.kind === 'key' ? normalizedStep.modifiers : []} : {kind: 'text', text: normalizedStep.kind === 'text' ? normalizedStep.text : ''})}
            >
              <option value="text">文本</option>
              <option value="key">按键</option>
            </select>
            {normalizedStep.kind === 'text' && <Input aria-label={`快捷键 ${index + 1} 动作 ${stepIndex + 1} 文本`} value={normalizedStep.text} onChange={(event) => updateStep(stepIndex, {kind: 'text', text: event.target.value})}/>}
            {normalizedStep.kind === 'key' && <ShortcutKeyCapture value={formatShortcutKey(normalizedStep)} onChange={(keyStep) => updateStep(stepIndex, keyStep)}/>}
            <Button variant="ghost" size="sm" aria-label={`删除快捷键 ${index + 1} 动作 ${stepIndex + 1}`} onClick={() => removeStep(stepIndex)} disabled={shortcut.steps.length <= 1}><Trash2 className="h-4 w-4"/></Button>
          </div>
        })}
        <div className="flex flex-wrap gap-2">
          <Button variant="secondary" size="sm" onClick={() => onChange({steps: [...shortcut.steps, {kind: 'text', text: ''}]})}><Plus className="mr-1 h-4 w-4"/>添加文本动作</Button>
          <Button variant="secondary" size="sm" onClick={() => onChange({steps: [...shortcut.steps, {kind: 'key', key: 'Enter', modifiers: []}]})}><Plus className="mr-1 h-4 w-4"/>添加按键动作</Button>
        </div>
      </div>
    </Field>
  </div>
}

function ShortcutCapture({value, onChange}: {value: string, onChange(value: string): void}) {
  return <Input
    aria-label="按键组合"
    placeholder="按下快捷键"
    value={value}
    onKeyDown={(event) => {
      event.preventDefault()
      const shortcut = keyboardEventShortcut(event.nativeEvent)
      if (shortcut) onChange(shortcut)
    }}
    onChange={() => {}}
  />
}

function ShortcutKeyCapture({value, onChange}: {value: string, onChange(value: {kind: 'key', key: string, modifiers: string[]}): void}) {
  return <Input
    aria-label="按键动作录入"
    placeholder="点击后按下按键"
    value={value}
    onKeyDown={(event) => {
      event.preventDefault()
      const step = keyboardEventStep(event.nativeEvent)
      if (step?.kind === 'key') onChange(step)
    }}
    onChange={() => {}}
  />
}

function formatShortcutKey(step: {kind: 'key', key: string, modifiers?: string[]}): string {
  return [...(step.modifiers ?? []), step.key].join('+')
}

function newShortcutID(): string {
  if (typeof crypto !== 'undefined' && 'randomUUID' in crypto) return crypto.randomUUID()
  return `terminal-shortcut-${Date.now()}-${Math.random().toString(16).slice(2)}`
}
