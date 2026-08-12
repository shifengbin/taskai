import {useEffect, useLayoutEffect, useMemo, useRef, useState, type CSSProperties} from 'react'
import {createPortal} from 'react-dom'
import {
	Archive,
	ArchiveRestore,
	ChevronDown,
	ChevronRight,
	ExternalLink,
	FolderOpen,
	GripVertical,
	MoreVertical,
	Pencil,
	Play,
	RotateCcw,
	CircleStop,
	Terminal,
	X,
} from 'lucide-react'

import {defaultTaskColor, defaultTaskMenuItems, terminalRealtimeStatus, type TaskMenuItem, type TaskRecord, type TaskStatus, type TerminalRecord} from '../types'
import {TerminalName} from './TerminalName'
import {TerminalStatusDot} from './TerminalStatusDot'
import {Checkbox, IconButton, Tabs, TabsList, TabsTrigger, cn} from './ui'

interface TaskMenuState {
  taskID: string
  anchorEl?: HTMLElement
  position?: {
    top: number
    left: number
  }
}

interface TaskMenuPlacement {
  top: number
  left: number
  maxHeight: number
  maxWidth: number
}

type TaskDropPosition = 'before' | 'after'

interface TaskDropTarget {
  taskID: string
  position: TaskDropPosition
}

interface TaskPointerDrag {
  taskID: string
  pointerID: number
  startX: number
  startY: number
  active: boolean
  dropTarget?: TaskDropTarget
}

interface TaskDragPreviewPosition {
  x: number
  y: number
}

export interface TaskStartFeedback {
  taskID: string
  sequence: number
}

interface TaskTreeProps {
  tasks: TaskRecord[]
  terminals: TerminalRecord[]
  menuItems?: TaskMenuItem[]
	activeStatus: TaskStatus
	taskDeletionSelectionMode?: boolean
	selectedTaskIDs?: string[]
	expandedTasks?: Record<string, boolean>
  selectedTaskID?: string
  selectedTerminalId?: string
  startedTaskFeedback?: TaskStartFeedback
	onChangeStatus(status: TaskStatus): void
	onToggleTaskDeletion?(taskID: string): void
  onToggleTaskExpanded?(taskID: string): void
  onSelectTask(task: TaskRecord): void
  onSelectTerminal(terminal: TerminalRecord): void
  onCreateTerminal(taskID: string): void
  onEditTask(taskID: string): void
  onOpenTaskFolder(taskID: string): void
  onRunMenuCommand?(taskID: string, itemID: string): void
  onStartTask(taskID: string): void
  onFinishTask(taskID: string): void
  onRetryLifecycle?(taskID: string): void
  onSetTaskShelved?(taskID: string, shelved: boolean): void
  onCloseTerminal?(terminal: TerminalRecord): void
  onAliasChange?(terminal: TerminalRecord, alias: string | undefined): void
  onReorderTasks?(taskID: string, targetTaskID: string, position: TaskDropPosition): void
}

export function TaskTree({
  tasks,
  terminals,
	menuItems = defaultTaskMenuItems,
	activeStatus,
	taskDeletionSelectionMode = false,
	selectedTaskIDs = [],
  expandedTasks,
  selectedTaskID,
  selectedTerminalId,
  startedTaskFeedback,
	onChangeStatus,
	onToggleTaskDeletion,
	onToggleTaskExpanded,
  onSelectTask,
  onSelectTerminal,
  onCreateTerminal,
  onEditTask,
  onOpenTaskFolder,
  onRunMenuCommand,
  onStartTask,
  onFinishTask,
  onRetryLifecycle,
  onSetTaskShelved,
  onCloseTerminal,
  onAliasChange,
  onReorderTasks,
}: TaskTreeProps) {
  const [localExpandedTasks, setLocalExpandedTasks] = useState<Record<string, boolean>>({})
  const [shelvedExpanded, setShelvedExpanded] = useState(false)
  const [taskMenu, setTaskMenu] = useState<TaskMenuState | null>(null)
  const [taskMenuPlacement, setTaskMenuPlacement] = useState<TaskMenuPlacement | null>(null)
  const [draggedTaskID, setDraggedTaskID] = useState<string>()
  const [dragPreviewPosition, setDragPreviewPosition] = useState<TaskDragPreviewPosition>()
  const [dropTarget, setDropTarget] = useState<TaskDropTarget>()
  const [descTooltip, setDescTooltip] = useState<{top: number; left: number; text: string} | null>(null)
  const pointerDragRef = useRef<TaskPointerDrag>()
  const suppressTaskClickRef = useRef(false)
  const taskListRef = useRef<HTMLUListElement | null>(null)
  const taskMenuRef = useRef<HTMLDivElement | null>(null)
  const reducedMotion = useReducedMotion()
  const terminalsByTask = useMemo(() => {
    return terminals.reduce<Record<string, TerminalRecord[]>>((byTask, terminal) => {
      byTask[terminal.taskId] = [...(byTask[terminal.taskId] ?? []), terminal]
      return byTask
    }, {})
  }, [terminals])
  const visibleTasks = useMemo(() => {
    const matching = tasks.filter((task) => task.status === activeStatus)
    if (activeStatus !== 'running') {
      return matching
    }
    return [...matching.filter((task) => !task.shelved), ...matching.filter((task) => task.shelved)]
  }, [activeStatus, tasks])
  const shelvedTasks = useMemo(() => activeStatus === 'running' ? visibleTasks.filter((task) => task.shelved) : [], [activeStatus, visibleTasks])
  const firstShelvedTaskID = shelvedTasks[0]?.id
  const taskCounts = useMemo(() => tasks.reduce<Record<TaskStatus, number>>((counts, task) => {
    counts[task.status] += 1
    return counts
  }, {pending: 0, running: 0, completed: 0}), [tasks])
  const taskMenuTask = taskMenu ? tasks.find((task) => task.id === taskMenu.taskID) : undefined
  const draggedTask = draggedTaskID ? tasks.find((task) => task.id === draggedTaskID) : undefined
	const expanded = expandedTasks ?? localExpandedTasks
	const selectingTasksForDeletion = (activeStatus === 'pending' || activeStatus === 'completed') && taskDeletionSelectionMode

  useLayoutEffect(() => {
    if (activeStatus !== 'running' || !startedTaskFeedback) {
      return
    }
    const taskList = taskListRef.current
    const taskItem = Array.from(taskList?.querySelectorAll<HTMLElement>('[data-task-id]') ?? [])
      .find((item) => item.dataset.taskId === startedTaskFeedback.taskID)
    if (!taskList || !taskItem) {
      return
    }
    const listBounds = taskList.getBoundingClientRect()
    const taskBounds = taskItem.getBoundingClientRect()
    const offset = taskBounds.top < listBounds.top
      ? taskBounds.top - listBounds.top
      : taskBounds.bottom > listBounds.bottom
        ? taskBounds.bottom - listBounds.bottom
        : 0
    if (offset !== 0) {
      taskList.scrollTop = Math.max(0, taskList.scrollTop + offset)
    }
  }, [activeStatus, startedTaskFeedback?.sequence])

  // 任务操作菜单：点击外部或按下 Escape 时关闭。
  useEffect(() => {
    if (!taskMenu) {
      return
    }
    const handlePointerDown = (event: PointerEvent) => {
      if (taskMenuRef.current && event.target instanceof Node && taskMenuRef.current.contains(event.target)) {
        return
      }
      setTaskMenu(null)
    }
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        setTaskMenu(null)
      }
    }
    document.addEventListener('pointerdown', handlePointerDown)
    document.addEventListener('keydown', handleKeyDown)
    return () => {
      document.removeEventListener('pointerdown', handlePointerDown)
      document.removeEventListener('keydown', handleKeyDown)
    }
  }, [taskMenu])

  const toggleExpanded = (taskID: string) => {
    if (onToggleTaskExpanded) {
      onToggleTaskExpanded(taskID)
      return
    }
    setLocalExpandedTasks((current) => ({...current, [taskID]: !(current[taskID] ?? true)}))
  }

  const getDropPosition = (taskItem: HTMLElement, clientY: number): TaskDropPosition => {
    const bounds = taskItem.getBoundingClientRect()
    return clientY < bounds.top + bounds.height / 2 ? 'before' : 'after'
  }

  const getPointerDropTarget = (clientX: number, clientY: number): TaskDropTarget | undefined => {
    const pointerTarget = document.elementFromPoint(clientX, clientY)
    if (!pointerTarget) {
      return undefined
    }
    const taskContainer = pointerTarget?.closest<HTMLElement>('[data-task-container]')
    const targetTaskID = taskContainer?.dataset.taskContainer
    if (!targetTaskID) {
      return undefined
    }
    if (activeStatus === 'running' && tasks.find((task) => task.id === targetTaskID)?.shelved) {
      return undefined
    }
    const taskItem = pointerTarget.closest<HTMLElement>('[data-task-id]')
    if (!taskItem || taskItem.dataset.taskId !== targetTaskID) {
      return {taskID: targetTaskID, position: 'after'}
    }
    return {taskID: targetTaskID, position: getDropPosition(taskItem, clientY)}
  }

  const setTaskDropTarget = (nextDropTarget?: TaskDropTarget) => {
    setDropTarget((current) => current?.taskID === nextDropTarget?.taskID && current?.position === nextDropTarget?.position ? current : nextDropTarget)
  }

  const clearTaskDrag = () => {
    pointerDragRef.current = undefined
    setDraggedTaskID(undefined)
    setDragPreviewPosition(undefined)
    setTaskDropTarget()
  }

	const beginTaskPointerDrag = (event: React.PointerEvent<HTMLElement>, taskID: string) => {
		if (selectingTasksForDeletion) {
			return
		}
		const current = tasks.find((task) => task.id === taskID)
		if (current?.lifecycleExecution || (activeStatus === 'running' && current?.shelved)) {
			return
		}
    suppressTaskClickRef.current = false
    if (event.button > 0 || (event.target as HTMLElement).closest('button')) {
      return
    }
    setDragPreviewPosition(undefined)
    pointerDragRef.current = {taskID, pointerID: event.pointerId, startX: event.clientX, startY: event.clientY, active: false}
    event.currentTarget.setPointerCapture?.(event.pointerId)
  }

  const moveTaskPointerDrag = (event: React.PointerEvent<HTMLElement>) => {
    const pointerDrag = pointerDragRef.current
    if (!pointerDrag || pointerDrag.pointerID !== event.pointerId) {
      return
    }
    if (!pointerDrag.active) {
      const distance = Math.hypot(event.clientX - pointerDrag.startX, event.clientY - pointerDrag.startY)
      if (distance < 6) {
        return
      }
      pointerDrag.active = true
      setDraggedTaskID(pointerDrag.taskID)
    }
    event.preventDefault()
    setDragPreviewPosition({x: event.clientX, y: event.clientY})
    const nextDropTarget = getPointerDropTarget(event.clientX, event.clientY)
    pointerDrag.dropTarget = nextDropTarget?.taskID === pointerDrag.taskID ? undefined : nextDropTarget
    setTaskDropTarget(pointerDrag.dropTarget)
  }

  const completeTaskPointerDrag = (event: React.PointerEvent<HTMLElement>) => {
    const pointerDrag = pointerDragRef.current
    if (!pointerDrag || pointerDrag.pointerID !== event.pointerId) {
      return
    }
    if (event.currentTarget.hasPointerCapture?.(event.pointerId)) {
      event.currentTarget.releasePointerCapture(event.pointerId)
    }
    pointerDragRef.current = undefined
    if (pointerDrag.active) {
      suppressTaskClickRef.current = true
      if (pointerDrag.dropTarget) {
        onReorderTasks?.(pointerDrag.taskID, pointerDrag.dropTarget.taskID, pointerDrag.dropTarget.position)
      }
    }
    setDraggedTaskID(undefined)
    setDragPreviewPosition(undefined)
    setTaskDropTarget()
  }

  const cancelTaskPointerDrag = (event: React.PointerEvent<HTMLElement>) => {
    const pointerDrag = pointerDragRef.current
    if (!pointerDrag || pointerDrag.pointerID !== event.pointerId) {
      return
    }
    if (event.currentTarget.hasPointerCapture?.(event.pointerId)) {
      event.currentTarget.releasePointerCapture(event.pointerId)
    }
    clearTaskDrag()
  }

	const requestContextMenu = (event: React.MouseEvent, task: TaskRecord) => {
		event.preventDefault()
		if (selectingTasksForDeletion) {
			return
		}
		if (task.lifecycleExecution) {
			return
		}
		setTaskMenuPlacement(null)
		setTaskMenu({taskID: task.id, position: {top: event.clientY - 6, left: event.clientX + 2}})
  }

  const runTaskMenuItem = (item: TaskMenuItem) => {
    if (!taskMenu) {
      return
    }
    if (item.kind === 'edit-task') {
      onEditTask(taskMenu.taskID)
    } else if (item.kind === 'create-terminal') {
      onCreateTerminal(taskMenu.taskID)
    } else if (item.kind === 'open-folder') {
      onOpenTaskFolder(taskMenu.taskID)
    } else if (item.kind === 'toggle-shelved' && taskMenuTask) {
      onSetTaskShelved?.(taskMenuTask.id, !taskMenuTask.shelved)
    } else if (item.kind === 'command') {
      onRunMenuCommand?.(taskMenu.taskID, item.id)
    }
    setTaskMenu(null)
  }

  const activeMenuItems = taskMenuTask && !taskMenuTask.lifecycleExecution ? menuItems.filter((item) => taskMenuTask.status === 'running' || item.kind === 'edit-task') : []

  useLayoutEffect(() => {
    if (!taskMenu || !taskMenuRef.current) {
      return
    }
    const updatePlacement = () => {
      const menu = taskMenuRef.current
      if (!menu) {
        return
      }
      const origin = taskMenu.anchorEl
        ? (() => {
            const bounds = taskMenu.anchorEl?.getBoundingClientRect()
            return bounds ? {top: bounds.top, bottom: bounds.bottom, left: bounds.left} : undefined
          })()
        : taskMenu.position && {top: taskMenu.position.top, bottom: taskMenu.position.top + 6, left: taskMenu.position.left}
      if (!origin) {
        return
      }
      const bounds = menu.getBoundingClientRect()
      setTaskMenuPlacement(calculateTaskMenuPlacement(origin, bounds, window.innerWidth, window.innerHeight))
    }
    updatePlacement()
    window.addEventListener('resize', updatePlacement)
    window.addEventListener('scroll', updatePlacement, true)
    return () => {
      window.removeEventListener('resize', updatePlacement)
      window.removeEventListener('scroll', updatePlacement, true)
    }
  }, [taskMenu, activeMenuItems.length])

  const taskMenuStyle = useMemo<CSSProperties>(() => {
    if (!taskMenu) {
      return {display: 'none'}
    }
    if (!taskMenuPlacement) {
      return {position: 'fixed', top: 0, left: 0, visibility: 'hidden'}
    }
    return {
      position: 'fixed',
      top: taskMenuPlacement.top,
      left: taskMenuPlacement.left,
      maxHeight: taskMenuPlacement.maxHeight,
      maxWidth: taskMenuPlacement.maxWidth,
      overflowY: 'auto',
    }
  }, [taskMenu, taskMenuPlacement])

  return (
    <nav aria-label="任务和终端" className="grid h-full min-h-0" style={{gridTemplateRows: 'auto minmax(0, 1fr)'}}>
      <Tabs value={activeStatus} onValueChange={(value) => onChangeStatus(value as TaskStatus)}>
        <TabsList aria-label="任务状态筛选">
          <TabsTrigger value="pending">{`未执行 (${taskCounts.pending})`}</TabsTrigger>
          <TabsTrigger value="running">{`执行中 (${taskCounts.running})`}</TabsTrigger>
          <TabsTrigger value="completed">{`已完成 (${taskCounts.completed})`}</TabsTrigger>
        </TabsList>
      </Tabs>
      <ul
        ref={taskListRef}
        data-testid="task-tree-list"
        className="snap-no-scrollbar m-0 list-none p-0 pt-3"
        style={{minHeight: 0, overflowX: 'hidden', overflowY: 'auto', scrollbarWidth: 'none'}}
      >
			{visibleTasks.map((task) => {
          const childTerminals = terminalsByTask[task.id] ?? []
          const isExpanded = expanded[task.id] ?? true
				const isSelectedTask = task.id === selectedTaskID
          const taskColor = task.color || defaultTaskColor
          const dropPosition = dropTarget?.taskID === task.id ? dropTarget.position : undefined
				const execution = task.lifecycleExecution
					const locked = Boolean(execution)
					const canSelectTaskForDeletion = selectingTasksForDeletion && !locked && Boolean(onToggleTaskDeletion)
					const isTaskSelectedForDeletion = canSelectTaskForDeletion && selectedTaskIDs.includes(task.id)
					const isTaskRowSelected = selectingTasksForDeletion ? isTaskSelectedForDeletion : isSelectedTask
				const executionLabel = execution ? `${lifecycleHookLabel(execution.hook)} · ${execution.currentCommandName || '命令'} ${execution.currentIndex}/${execution.commandCount}` : ''
				const isShelvedTask = activeStatus === 'running' && Boolean(task.shelved)
				const isFirstShelvedTask = task.id === firstShelvedTaskID
          const startFeedbackMode = task.status === 'running' && startedTaskFeedback?.taskID === task.id
            ? reducedMotion ? 'static' : 'flash'
            : undefined
          return (
            <li key={task.id}>
              {isFirstShelvedTask && (
                <button
                  type="button"
                  aria-label={shelvedExpanded ? '收起已搁置任务' : '展开已搁置任务'}
                  onClick={() => setShelvedExpanded((current) => !current)}
                  className="taskai-shelved-toggle flex w-full items-center gap-2 border-t-2 border-snap-outline/25 px-2 py-2 text-left text-sm font-semibold text-snap-ink"
                >
                  {shelvedExpanded ? <ChevronDown className="h-4 w-4"/> : <ChevronRight className="h-4 w-4"/>}
                  <span>{`已搁置 (${shelvedTasks.length})`}</span>
                </button>
              )}
              {(!isShelvedTask || shelvedExpanded) && (
                <div data-task-container={task.id}>
              {dropPosition === 'before' && <TaskDropIndicator taskTitle={task.title} position={dropPosition}/>}
              <div
                className={cn(
                  'taskai-task-row taskai-contextual-container relative flex h-[60px] items-center gap-1.5 py-2 pl-3 pr-1.5',
                  'cursor-default touch-none select-none',
                  isTaskRowSelected && 'taskai-task-row--selected',
                )}
                data-task-id={task.id}
						data-task-selected={isTaskRowSelected || undefined}
						data-task-start-feedback={startFeedbackMode}
						data-task-actions-active={taskMenu?.taskID === task.id || undefined}
                onContextMenu={(event) => requestContextMenu(event, task)}
                onMouseEnter={(event) => {
                  const bounds = event.currentTarget.getBoundingClientRect()
                  setDescTooltip({top: bounds.top, left: bounds.right + 8, text: task.description || '暂无描述'})
                }}
                onMouseLeave={() => setDescTooltip(null)}
						onClick={(event) => {
                  if (suppressTaskClickRef.current) {
                    suppressTaskClickRef.current = false
                    event.preventDefault()
                    return
                  }
							if (selectingTasksForDeletion) {
								if (canSelectTaskForDeletion) {
									onToggleTaskDeletion?.(task.id)
								}
								return
							}
							onSelectTask(task)
                  }}
                  onDoubleClick={(event) => {
					if (selectingTasksForDeletion) {
                      return
                    }
                    if ((event.target as HTMLElement).closest('button')) {
                      return
                    }
                    toggleExpanded(task.id)
                  }}
                  onPointerDown={(event) => beginTaskPointerDrag(event, task.id)}
                  onPointerMove={moveTaskPointerDrag}
                  onPointerUp={completeTaskPointerDrag}
                  onPointerCancel={cancelTaskPointerDrag}
                  style={{
                    ['--task-color' as string]: taskColor,
                    opacity: draggedTaskID === task.id ? 0.5 : 1,
                    outline: draggedTaskID === task.id ? '2px solid var(--snap-cobalt)' : '2px solid transparent',
                    outlineOffset: -2,
					cursor: selectingTasksForDeletion ? (canSelectTaskForDeletion ? 'pointer' : 'not-allowed') : locked ? 'not-allowed' : 'grab',
                  } as CSSProperties}
                >
						{selectingTasksForDeletion && (
							<Checkbox
								checked={isTaskSelectedForDeletion}
								disabled={!canSelectTaskForDeletion}
								aria-label={`选择任务 ${task.title}`}
								className="mr-1"
								onClick={(event) => event.stopPropagation()}
								onCheckedChange={() => onToggleTaskDeletion?.(task.id)}
							/>
						)}
						{task.status === 'running' && (
						  <div className="taskai-contextual-actions flex shrink-0 items-center" data-testid={`task-row-leading-actions-${task.id}`}>
						    <IconButton
						      aria-label={isExpanded ? '收起终端' : '展开终端'}
						      title={isExpanded ? '收起终端' : '展开终端'}
						      className="h-7 w-7"
						      onClick={(event) => {
						        event.stopPropagation()
						        toggleExpanded(task.id)
						      }}
						    >
						      {isExpanded ? <ChevronDown className="h-4 w-4"/> : <ChevronRight className="h-4 w-4"/>}
						    </IconButton>
						  </div>
						)}
                <div className="flex min-w-0 flex-1 flex-col justify-center">
                  <span className="truncate font-display text-[13.5px] font-extrabold text-snap-ink">{task.title}</span>
                  <span className="mt-px truncate font-sans text-[11.5px] font-medium text-snap-muted">{task.description || '暂无描述'}</span>
                </div>
						{execution && (
							<span
                  title={execution.error || '命令链正在执行'}
                  className={cn(
                    'taskai-lifecycle-chip ml-1 inline-flex max-w-[180px] shrink-0 items-center rounded-snap-sm border-2 px-2 py-0.5 text-xs font-bold',
                    execution.state === 'failed' ? 'taskai-lifecycle-chip--error' : 'taskai-lifecycle-chip--warning',
                  )}
                >
                  <span className="truncate">{executionLabel}</span>
                </span>
						)}
                {task.status === 'running' && !isExpanded && <TerminalStatusDot status={task.realtimeStatus ?? 'idle'}/>}
						{!selectingTasksForDeletion && (
						  <div className="taskai-contextual-actions flex shrink-0 items-center gap-1.5" data-testid={`task-row-trailing-actions-${task.id}`}>
						    {task.status === 'pending' && (
						      <IconButton
						        aria-label="执行"
						        title="执行"
						        disabled={locked}
						        className="h-7 w-7"
						        onClick={(event) => {
						          event.stopPropagation()
						          onStartTask(task.id)
						        }}
						      >
						        <Play className="h-4 w-4"/>
						      </IconButton>
						    )}
						    {task.status === 'running' && (
						      <IconButton
						        aria-label="结束"
						        title="结束"
						        disabled={locked}
						        className="h-7 w-7"
						        onClick={(event) => {
						          event.stopPropagation()
						          onFinishTask(task.id)
						        }}
						      >
						        <CircleStop className="h-4 w-4"/>
						      </IconButton>
						    )}
						    {execution?.state === 'failed' && onRetryLifecycle && (
						      <IconButton
						        aria-label="重试命令链"
						        title="从命令链首条命令重新执行"
						        className="h-7 w-7 text-snap-error"
						        onClick={(event) => {
						          event.stopPropagation()
						          onRetryLifecycle(task.id)
						        }}
						      >
						        <RotateCcw className="h-4 w-4"/>
						      </IconButton>
						    )}
							    {!selectingTasksForDeletion && (
						      <IconButton
						        aria-label="任务操作"
						        title="任务操作"
						        disabled={locked}
						        className="h-7 w-7"
							        onClick={(event) => {
							          event.stopPropagation()
							          setTaskMenuPlacement(null)
							          setTaskMenu({taskID: task.id, anchorEl: event.currentTarget})
							        }}
						      >
						        <MoreVertical className="h-4 w-4"/>
						      </IconButton>
						    )}
						  </div>
						)}
              </div>
              {isExpanded && (childTerminals.length > 0 || task.status === 'running') && (
                <ul className="snap-no-scrollbar m-0 mb-3 flex list-none flex-col p-0 pl-3 pr-2">
                  {childTerminals.map((terminal) => (
                    <li key={terminal.id}>
                      <div
                        role="button"
                        tabIndex={0}
                        aria-pressed={terminal.id === selectedTerminalId}
                        onClick={() => onSelectTerminal(terminal)}
                        onKeyDown={(event) => {
                          if (event.key === 'Enter' || event.key === ' ') {
                            event.preventDefault()
                            onSelectTerminal(terminal)
                          }
                        }}
						className={cn(
						  'taskai-terminal-row taskai-contextual-container flex min-h-[38px] cursor-default items-center gap-2 px-2 py-1 text-sm text-snap-ink',
                        )}
                      >
                        <Terminal className={cn('h-4 w-4 shrink-0', terminal.state === 'active' ? 'text-snap-cobalt' : 'text-snap-muted')}/>
                        <div style={{flex: 1, minWidth: 0}}>
                          <TerminalName
                            terminal={terminal}
                            onAliasChange={(alias) => onAliasChange?.(terminal, alias)}
                            testId={`task-tree-terminal-title-${terminal.id}`}
                            className="block"
                            tooltipPlacement={{side: 'right', align: 'start', sideOffset: 8, avoidCollisions: false}}
                          />
                        </div>
                        <TerminalStatusDot status={terminalRealtimeStatus(terminal)}/>
						{onCloseTerminal && (
						  <div className="taskai-contextual-actions flex shrink-0 items-center" data-testid={`task-terminal-actions-${terminal.id}`}>
						    <IconButton
						      aria-label="关闭终端"
						      title="关闭终端"
						      className="h-7 w-7"
						      onClick={(event) => {
						        event.stopPropagation()
						        onCloseTerminal(terminal)
						      }}
						    >
						      <X className="h-4 w-4"/>
						    </IconButton>
						  </div>
						)}
                      </div>
                    </li>
                  ))}
                  {task.status === 'running' && childTerminals.length === 0 && (
                    <li className="flex items-center gap-2 px-3 py-2 text-xs text-snap-muted">
                      <Terminal className="h-4 w-4"/>
                      <span>右键任务后可新增终端</span>
                    </li>
                  )}
                </ul>
              )}
              {dropPosition === 'after' && <TaskDropIndicator taskTitle={task.title} position={dropPosition}/>}
            </div>
              )}
            </li>
          )
        })}
      </ul>
      {draggedTask && dragPreviewPosition && <TaskDragPreview task={draggedTask} position={dragPreviewPosition}/>}
      {descTooltip && createPortal(
        <div
          role="tooltip"
          className="pointer-events-none fixed z-50 max-w-[480px] break-words whitespace-pre-wrap rounded-snap-sm border border-snap-outline bg-snap-overlay px-2 py-1 text-xs font-bold text-snap-ink shadow-snap"
          style={{top: descTooltip.top, left: descTooltip.left}}
        >
          {descTooltip.text}
        </div>,
        document.body,
      )}
      {taskMenu && createPortal(
        <div
          ref={taskMenuRef}
          role="menu"
          className="z-50 min-w-[12rem] border border-snap-outline rounded-snap bg-snap-overlay py-1 text-snap-ink shadow-snap-lg"
          style={taskMenuStyle}
        >
          {activeMenuItems.map((item) => (
            <div
              key={item.id}
              role="menuitem"
              className="flex cursor-default items-center gap-2 px-3 py-1.5 text-sm font-semibold text-snap-ink transition-colors hover:bg-snap-cobalt hover:text-white"
              onClick={() => runTaskMenuItem(item)}
            >
              {item.kind === 'edit-task' && <Pencil className="h-4 w-4"/>}
              {item.kind === 'create-terminal' && <Terminal className="h-4 w-4"/>}
              {item.kind === 'open-folder' && <FolderOpen className="h-4 w-4"/>}
              {item.kind === 'toggle-shelved' && (taskMenuTask?.shelved ? <ArchiveRestore className="h-4 w-4"/> : <Archive className="h-4 w-4"/>)}
              {item.kind === 'command' && (item.showTerminal ? <Terminal className="h-4 w-4"/> : <ExternalLink className="h-4 w-4"/>)}
              <span>{item.kind === 'toggle-shelved' && taskMenuTask?.shelved ? item.unshelveName ?? '取消搁置' : item.name}</span>
            </div>
          ))}
        </div>,
        document.body,
      )}
    </nav>
  )
}

function calculateTaskMenuPlacement(
  origin: {top: number; bottom: number; left: number},
  menu: Pick<DOMRect, 'width' | 'height'>,
  viewportWidth: number,
  viewportHeight: number,
): TaskMenuPlacement {
  const viewportPadding = 8
  const menuOffset = 4
  const availableBelow = Math.max(0, viewportHeight - viewportPadding - origin.bottom - menuOffset)
  const availableAbove = Math.max(0, origin.top - viewportPadding - menuOffset)
  const placeBelow = menu.height <= availableBelow || (menu.height > availableAbove && availableBelow >= availableAbove)
  const maxLeft = Math.max(viewportPadding, viewportWidth - viewportPadding - menu.width)
  const maxHeight = placeBelow ? availableBelow : availableAbove
  const displayedHeight = Math.min(menu.height, maxHeight)
  return {
    top: placeBelow ? origin.bottom + menuOffset : origin.top - menuOffset - displayedHeight,
    left: Math.min(Math.max(origin.left, viewportPadding), maxLeft),
    maxHeight,
    maxWidth: Math.max(0, viewportWidth - viewportPadding * 2),
  }
}

function lifecycleHookLabel(hook: NonNullable<TaskRecord['lifecycleExecution']>['hook']): string {
	switch (hook) {
	case 'beforeStart': return '开始前'
	case 'postStart': return '开始后'
	case 'beforeEnd': return '结束前'
	case 'postEnd': return '结束后'
	case 'updateTask': return '更新后'
	}
}

function useReducedMotion(): boolean {
  const [reducedMotion, setReducedMotion] = useState(() => readReducedMotionPreference())

  useEffect(() => {
    const mediaQuery = window.matchMedia?.('(prefers-reduced-motion: reduce)')
    if (!mediaQuery) {
      return
    }
    const update = () => setReducedMotion(mediaQuery.matches)
    mediaQuery.addEventListener('change', update)
    return () => mediaQuery.removeEventListener('change', update)
  }, [])

  return reducedMotion
}

function readReducedMotionPreference(): boolean {
  return typeof window !== 'undefined' && window.matchMedia?.('(prefers-reduced-motion: reduce)').matches === true
}

function TaskDropIndicator({taskTitle, position}: {taskTitle: string; position: TaskDropPosition}) {
  const positionLabel = position === 'before' ? '之前' : '之后'
  return (
    <div
      role="status"
      aria-label={`将任务插入“${taskTitle}”${positionLabel}`}
      className="relative mx-1 my-0 h-3.5"
      style={{pointerEvents: 'none', zIndex: 1}}
    >
      <span
        aria-hidden
        className="absolute left-0 right-0 top-1/2 h-1 -translate-y-1/2 rounded-full bg-snap-cobalt"
      />
      <span
        aria-hidden
        className="absolute left-[-4px] top-1/2 h-2.5 w-2.5 -translate-y-1/2 rounded-full border-2 border-snap-surface bg-snap-cobalt"
      />
    </div>
  )
}

function TaskDragPreview({task, position}: {task: TaskRecord; position: TaskDragPreviewPosition}) {
  const taskColor = task.color || defaultTaskColor
  return (
    <div
      role="status"
      aria-label={`正在调整任务“${task.title}”`}
      className="taskai-task-drag-preview pointer-events-none flex max-w-[min(300px,calc(100vw-32px))] items-center gap-2 border border-snap-outline/25 py-1 pl-1 pr-2"
      style={{
        position: 'fixed',
        zIndex: 60,
        top: `min(${position.y + 14}px, calc(100vh - 56px))`,
        left: `min(${position.x + 14}px, calc(100vw - 32px))`,
        minHeight: 40,
        borderLeft: `4px solid ${taskColor}`,
        backgroundColor: 'var(--snap-surface)',
      }}
    >
      <GripVertical className="h-4 w-4 text-snap-muted"/>
      <span className="font-display text-sm font-bold text-snap-ink" style={{overflowWrap: 'anywhere'}}>{task.title}</span>
    </div>
  )
}
