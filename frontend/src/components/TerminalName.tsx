import {useEffect, useRef, useState} from 'react'

import {terminalAliasDetails, terminalDisplayName, type TerminalRecord} from '../types'
import {Input, Tooltip, TooltipContent, TooltipTrigger} from './ui'

interface TerminalNameProps {
  terminal: TerminalRecord
  onAliasChange(alias: string | undefined): void
  className?: string
  testId?: string
  detailsDisplay?: 'tooltip' | 'inline-session-details'
  tooltipPlacement?: {
    side: 'top' | 'right' | 'bottom' | 'left'
    align: 'start' | 'center' | 'end'
    sideOffset: number
    avoidCollisions?: boolean
  }
}

export function TerminalName({terminal, onAliasChange, className, testId, detailsDisplay = 'tooltip', tooltipPlacement}: TerminalNameProps) {
  const inputRef = useRef<HTMLInputElement>(null)
  const [editing, setEditing] = useState(false)
  const [draft, setDraft] = useState('')
  const hasAlias = Boolean(terminal.alias?.trim())
  const details = terminalAliasDetails(terminal)

  useEffect(() => {
    if (editing) {
      inputRef.current?.focus()
    }
  }, [editing])

  const save = () => {
    onAliasChange(draft.trim() || undefined)
    setEditing(false)
  }

  if (editing) {
    return (
      <Input
        ref={inputRef}
        aria-label="终端别名"
        value={draft}
        className={className}
        onClick={(event) => event.stopPropagation()}
        onChange={(event) => setDraft(event.target.value)}
        onBlur={save}
        onKeyDown={(event) => {
          if (event.key === 'Enter') {
            event.preventDefault()
            save()
          } else if (event.key === 'Escape') {
            event.preventDefault()
            setEditing(false)
          }
        }}
      />
    )
  }

  const name = (
    <span
      data-testid={testId}
      className={className}
      style={{whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'clip'}}
      onDoubleClick={(event) => {
        event.stopPropagation()
        setDraft(terminal.alias ?? '')
        setEditing(true)
      }}
    >
      {detailsDisplay === 'inline-session-details' && hasAlias
        ? `${terminalDisplayName(terminal)}(${details.actualName}:${details.command})`
        : terminalDisplayName(terminal)}
    </span>
  )

  if (!hasAlias || detailsDisplay === 'inline-session-details') {
    return name
  }

  return (
    <Tooltip delayDuration={0}>
      <TooltipTrigger asChild>{name}</TooltipTrigger>
      <TooltipContent data-testid="terminal-alias-details" className={tooltipPlacement ? 'pointer-events-none' : undefined} {...tooltipPlacement}>
        <span data-testid="terminal-alias-actual-name" className="block">标题: {details.actualName}</span>
        <span data-testid="terminal-alias-command" className="block">命令: {details.command}</span>
      </TooltipContent>
    </Tooltip>
  )
}
