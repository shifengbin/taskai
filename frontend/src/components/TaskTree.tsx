import {useEffect, useLayoutEffect, useMemo, useRef, useState} from 'react'
import {
  Box,
  Chip,
  Collapse,
  IconButton,
  List,
  ListItemButton,
  ListItemText,
  Menu,
  MenuItem,
  Tab,
  Tabs,
  Tooltip,
  Typography,
} from '@mui/material'
import AddOutlinedIcon from '@mui/icons-material/AddOutlined'
import ChevronRightIcon from '@mui/icons-material/ChevronRight'
import CloseOutlinedIcon from '@mui/icons-material/CloseOutlined'
import DragIndicatorIcon from '@mui/icons-material/DragIndicator'
import EditOutlinedIcon from '@mui/icons-material/EditOutlined'
import ExpandMoreIcon from '@mui/icons-material/ExpandMore'
import FolderOpenOutlinedIcon from '@mui/icons-material/FolderOpenOutlined'
import MoreVertIcon from '@mui/icons-material/MoreVert'
import OpenInNewOutlinedIcon from '@mui/icons-material/OpenInNewOutlined'
import PlayArrowOutlinedIcon from '@mui/icons-material/PlayArrowOutlined'
import ReplayOutlinedIcon from '@mui/icons-material/ReplayOutlined'
import TerminalOutlinedIcon from '@mui/icons-material/TerminalOutlined'
import TaskAltOutlinedIcon from '@mui/icons-material/TaskAltOutlined'

import {defaultTaskColor, defaultTaskMenuItems, terminalDisplayName, terminalRealtimeStatus, type TaskMenuItem, type TaskRecord, type TaskStatus, type TerminalRecord} from '../types'
import {TerminalStatusDot} from './TerminalStatusDot'

interface TaskMenuState {
  taskID: string
  anchorEl?: HTMLElement
  position?: {
    top: number
    left: number
  }
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
  expandedTasks?: Record<string, boolean>
  selectedTaskID?: string
  selectedTerminalId?: string
  startedTaskFeedback?: TaskStartFeedback
  onChangeStatus(status: TaskStatus): void
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
  onReorderTasks?(taskID: string, targetTaskID: string, position: TaskDropPosition): void
}

export function TaskTree({
  tasks,
  terminals,
  menuItems = defaultTaskMenuItems,
  activeStatus,
  expandedTasks,
  selectedTaskID,
  selectedTerminalId,
  startedTaskFeedback,
  onChangeStatus,
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
  onReorderTasks,
}: TaskTreeProps) {
  const [localExpandedTasks, setLocalExpandedTasks] = useState<Record<string, boolean>>({})
  const [shelvedExpanded, setShelvedExpanded] = useState(false)
  const [taskMenu, setTaskMenu] = useState<TaskMenuState | null>(null)
  const [draggedTaskID, setDraggedTaskID] = useState<string>()
  const [dragPreviewPosition, setDragPreviewPosition] = useState<TaskDragPreviewPosition>()
  const [dropTarget, setDropTarget] = useState<TaskDropTarget>()
  const pointerDragRef = useRef<TaskPointerDrag>()
  const suppressTaskClickRef = useRef(false)
  const taskListRef = useRef<HTMLUListElement | null>(null)
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
		if (task.lifecycleExecution) {
			return
		}
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
    } else if (item.kind === 'command') {
      onRunMenuCommand?.(taskMenu.taskID, item.id)
    }
    setTaskMenu(null)
  }

  const activeMenuItems = taskMenuTask && !taskMenuTask.lifecycleExecution ? menuItems.filter((item) => taskMenuTask.status === 'running' || item.kind === 'edit-task') : []

  return (
    <Box className="taskai-task-tree" component="nav" aria-label="任务和终端" sx={{height: '100%', minHeight: 0, display: 'grid', gridTemplateRows: 'auto minmax(0, 1fr)'}}>
      <Tabs
        value={activeStatus}
        onChange={(_event, status: TaskStatus) => onChangeStatus(status)}
        aria-label="任务状态筛选"
        variant="fullWidth"
        sx={{borderBottom: 1, borderColor: 'divider', minHeight: 42}}
      >
        <Tab value="pending" label={`未执行 (${taskCounts.pending})`} sx={{minHeight: 42, minWidth: 0, px: 0.5}}/>
        <Tab value="running" label={`执行中 (${taskCounts.running})`} sx={{minHeight: 42, minWidth: 0, px: 0.5}}/>
        <Tab value="completed" label={`已完成 (${taskCounts.completed})`} sx={{minHeight: 42, minWidth: 0, px: 0.5}}/>
      </Tabs>
      <List className="taskai-task-tree__list" ref={taskListRef} data-testid="task-tree-list" disablePadding dense sx={{minHeight: 0, overflowX: 'hidden', overflowY: 'auto', scrollbarWidth: 'none', '&::-webkit-scrollbar': {display: 'none'}}}>
        {visibleTasks.map((task) => {
          const childTerminals = terminalsByTask[task.id] ?? []
          const isExpanded = expanded[task.id] ?? true
          const isSelectedTask = task.id === selectedTaskID
          const taskColor = task.color || defaultTaskColor
          const dropPosition = dropTarget?.taskID === task.id ? dropTarget.position : undefined
			const execution = task.lifecycleExecution
				const locked = Boolean(execution)
				const executionLabel = execution ? `${lifecycleHookLabel(execution.hook)} · ${execution.currentCommandName || '命令'} ${execution.currentIndex}/${execution.commandCount}` : ''
				const isShelvedTask = activeStatus === 'running' && Boolean(task.shelved)
				const isFirstShelvedTask = task.id === firstShelvedTaskID
          const startFeedbackMode = task.status === 'running' && startedTaskFeedback?.taskID === task.id
            ? reducedMotion ? 'static' : 'flash'
            : undefined
          return (
            <Box
              key={task.id}
            >
              {isFirstShelvedTask && (
                <ListItemButton
                  className="taskai-task-row taskai-task-row--shelved"
                  aria-label={shelvedExpanded ? '收起已搁置任务' : '展开已搁置任务'}
                  onClick={() => setShelvedExpanded((current) => !current)}
                  sx={{minHeight: 44, gap: 0.75, borderTop: 1, borderColor: 'divider'}}
                >
                  {shelvedExpanded ? <ExpandMoreIcon fontSize="small"/> : <ChevronRightIcon fontSize="small"/>}
                  <ListItemText primary={`已搁置 (${shelvedTasks.length})`} slotProps={{primary: {noWrap: true, sx: {fontWeight: 600}}}}/>
                </ListItemButton>
              )}
              {(!isShelvedTask || shelvedExpanded) && (
                <Box
              data-task-container={task.id}
            >
              {dropPosition === 'before' && <TaskDropIndicator taskTitle={task.title} position={dropPosition}/>}
              <Tooltip
                title={task.description || '暂无任务描述'}
                placement="right"
                slotProps={{tooltip: {sx: {maxWidth: 480, whiteSpace: 'pre-wrap', overflowWrap: 'anywhere'}}}}
              >
                <ListItemButton
                  className="taskai-task-row"
                  data-task-id={task.id}
						data-task-selected={isSelectedTask || undefined}
						data-task-start-feedback={startFeedbackMode}
                  selected={isSelectedTask}
                  onClick={(event) => {
                    if (suppressTaskClickRef.current) {
                      suppressTaskClickRef.current = false
                      event.preventDefault()
                      return
                    }
                    onSelectTask(task)
                  }}
                  onContextMenu={(event) => requestContextMenu(event, task)}
                  onDoubleClick={(event) => {
                    if ((event.target as HTMLElement).closest('button')) {
                      return
                    }
                    toggleExpanded(task.id)
                  }}
                  onPointerDown={(event) => beginTaskPointerDrag(event, task.id)}
                  onPointerMove={moveTaskPointerDrag}
                  onPointerUp={completeTaskPointerDrag}
                  onPointerCancel={cancelTaskPointerDrag}
                  style={{borderLeftColor: taskColor}}
                  sx={{
                    minHeight: 48,
                    gap: 0.5,
                    borderLeft: 4,
                    borderLeftStyle: 'solid',
                    position: 'relative',
                    borderRadius: '3px 15px 3px 15px',
                    bgcolor: isSelectedTask ? `${taskColor}2e` : `${taskColor}14`,
                    opacity: draggedTaskID === task.id ? 0.5 : 1,
                    outline: draggedTaskID === task.id ? '2px solid' : '2px solid transparent',
						outlineColor: draggedTaskID === task.id ? 'primary.main' : 'transparent',
						outlineOffset: -2,
						'&.Mui-selected': {backgroundColor: `${taskColor}2e`, boxShadow: `inset -3px 0 0 ${taskColor}, inset 0 0 0 1px ${taskColor}`},
						'&.Mui-selected:hover': {backgroundColor: `${taskColor}3b`},
						...(startFeedbackMode === 'static' && {boxShadow: `inset 0 0 0 2px ${taskColor}`}),
						...(startFeedbackMode === 'flash' && {
							animation: 'taskai-task-start-feedback 350ms ease-in-out 2',
							'@keyframes taskai-task-start-feedback': {
								'0%, 100%': {boxShadow: `inset 0 0 0 0 ${taskColor}`, backgroundColor: `${taskColor}14`},
								'50%': {boxShadow: `inset 0 0 0 2px ${taskColor}`, backgroundColor: `${taskColor}2e`},
							},
						}),
						cursor: locked ? 'not-allowed' : 'grab',
                    touchAction: 'none',
                    userSelect: 'none',
                    '&:active': {cursor: 'grabbing'},
                  }}
                >
                {task.status === 'running' && (
                  <IconButton
                    aria-label={isExpanded ? '收起终端' : '展开终端'}
                    size="small"
                    onClick={(event) => {
                      event.stopPropagation()
                      toggleExpanded(task.id)
                    }}
                  >
                    {isExpanded ? <ExpandMoreIcon fontSize="small"/> : <ChevronRightIcon fontSize="small"/>}
                  </IconButton>
                )}
                <ListItemText
                  primary={task.title}
                  secondary={task.description || undefined}
                  slotProps={{
                    primary: {noWrap: true, sx: {fontWeight: 600}},
                    secondary: {noWrap: true},
                  }}
                />
						{execution && <Tooltip title={execution.error || '命令链正在执行'}>
							<Chip label={executionLabel} size="small" color={execution.state === 'failed' ? 'error' : 'warning'} variant="outlined" sx={{maxWidth: 180, '& .MuiChip-label': {overflow: 'hidden', textOverflow: 'ellipsis'}}}/>
						</Tooltip>}
                {task.status === 'running' && !isExpanded && <TerminalStatusDot status={task.realtimeStatus ?? 'idle'}/>}
                {task.status === 'pending' && (
                  <Tooltip title="执行">
						<span>
							<IconButton
                      aria-label="执行"
                      size="small"
						disabled={locked}
                      onClick={(event) => {
                        event.stopPropagation()
                        onStartTask(task.id)
                      }}
							>
								<PlayArrowOutlinedIcon fontSize="small"/>
							</IconButton>
						</span>
                  </Tooltip>
                )}
                {task.status === 'running' && (
                  <Tooltip title="结束">
						<span>
							<IconButton
                      aria-label="结束"
                      size="small"
						disabled={locked}
                      onClick={(event) => {
                        event.stopPropagation()
                        onFinishTask(task.id)
                      }}
							>
								<TaskAltOutlinedIcon fontSize="small"/>
							</IconButton>
						</span>
                  </Tooltip>
                )}
						{execution?.state === 'failed' && onRetryLifecycle && (
							<Tooltip title="从命令链首条命令重新执行">
								<IconButton aria-label="重试命令链" size="small" color="error" onClick={(event) => {
									event.stopPropagation()
									onRetryLifecycle(task.id)
								}}>
									<ReplayOutlinedIcon fontSize="small"/>
								</IconButton>
							</Tooltip>
						)}
                <Tooltip title="任务操作">
						<span>
							<IconButton
                    aria-label="任务操作"
                    size="small"
						disabled={locked}
                    onClick={(event) => {
                      event.stopPropagation()
                      setTaskMenu({taskID: task.id, anchorEl: event.currentTarget})
                    }}
							>
								<MoreVertIcon fontSize="small"/>
							</IconButton>
						</span>
                </Tooltip>
                </ListItemButton>
              </Tooltip>
              <Collapse in={isExpanded} timeout="auto" unmountOnExit>
                <List disablePadding dense sx={{pl: 3.25}}>
                  {childTerminals.map((terminal) => (
                    <ListItemButton
                      className="taskai-terminal-row"
                      key={terminal.id}
                      selected={terminal.id === selectedTerminalId}
                      onClick={() => onSelectTerminal(terminal)}
                      sx={{minHeight: 38, gap: 0.75}}
                    >
                      <TerminalOutlinedIcon fontSize="small" color={terminal.state === 'active' ? 'primary' : 'disabled'}/>
                      <ListItemText
                        sx={{flex: 1, minWidth: 0}}
                        primary={
                          <Typography
                            data-testid={`task-tree-terminal-title-${terminal.id}`}
                            variant="body2"
                            sx={{whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'clip'}}
                          >
                            {terminalDisplayName(terminal)}
                          </Typography>
                        }
                      />
                      <TerminalStatusDot status={terminalRealtimeStatus(terminal)}/>
                      {onCloseTerminal && (
                        <Tooltip title="关闭终端">
                          <IconButton
                            aria-label="关闭终端"
                            size="small"
                            onClick={(event) => {
                              event.stopPropagation()
                              onCloseTerminal(terminal)
                            }}
                          >
                            <CloseOutlinedIcon fontSize="small"/>
                          </IconButton>
                        </Tooltip>
                      )}
                    </ListItemButton>
                  ))}
                  {task.status === 'running' && childTerminals.length === 0 && (
                    <Box sx={{display: 'flex', alignItems: 'center', gap: 0.75, px: 2, py: 1, color: 'text.secondary'}}>
                      <TerminalOutlinedIcon fontSize="small"/>
                      <Typography variant="caption">右键任务后可新增终端</Typography>
                    </Box>
                  )}
                </List>
              </Collapse>
              {dropPosition === 'after' && <TaskDropIndicator taskTitle={task.title} position={dropPosition}/>}
            </Box>
              )}
            </Box>
          )
        })}
      </List>
      {draggedTask && dragPreviewPosition && <TaskDragPreview task={draggedTask} position={dragPreviewPosition}/>}
      <Menu
        className="taskai-task-menu"
        open={taskMenu !== null}
        onClose={() => setTaskMenu(null)}
        anchorEl={taskMenu?.anchorEl}
        anchorReference={taskMenu?.anchorEl ? 'anchorEl' : 'anchorPosition'}
        anchorPosition={taskMenu?.position}
      >
        {activeMenuItems.map((item) => (
          <MenuItem key={item.id} onClick={() => runTaskMenuItem(item)}>
            {item.kind === 'edit-task' && <EditOutlinedIcon fontSize="small"/>}
            {item.kind === 'create-terminal' && <TerminalOutlinedIcon fontSize="small"/>}
            {item.kind === 'open-folder' && <FolderOpenOutlinedIcon fontSize="small"/>}
            {item.kind === 'command' && (item.showTerminal ? <TerminalOutlinedIcon fontSize="small"/> : <OpenInNewOutlinedIcon fontSize="small"/>)}
            <Typography component="span" sx={{ml: 1}}>{item.name}</Typography>
          </MenuItem>
        ))}
        {taskMenuTask?.status === 'running' && !taskMenuTask.lifecycleExecution && (
          <MenuItem onClick={() => {
            onSetTaskShelved?.(taskMenuTask.id, !taskMenuTask.shelved)
            setTaskMenu(null)
          }}>
            <Typography component="span">{taskMenuTask.shelved ? '取消搁置' : '搁置任务'}</Typography>
          </MenuItem>
        )}
      </Menu>
    </Box>
  )
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
    <Box
      role="status"
      aria-label={`将任务插入“${taskTitle}”${positionLabel}`}
      sx={{
        height: 14,
        mx: 1,
        position: 'relative',
        pointerEvents: 'none',
        zIndex: 1,
        '&::before': {
          content: '""',
          position: 'absolute',
          top: '50%',
          right: 0,
          left: 0,
          height: 4,
          borderRadius: 2,
          backgroundImage: (theme) => `repeating-linear-gradient(-33deg, ${theme.palette.secondary.main} 0 8px, ${theme.palette.primary.main} 9px 15px)`,
          transform: 'translateY(-50%)',
        },
        '&::after': {
          content: '""',
          position: 'absolute',
          top: '50%',
          left: -4,
          width: 10,
          height: 10,
          borderRadius: '2px 8px 2px 8px',
          bgcolor: 'secondary.main',
          transform: 'translateY(-50%)',
          boxShadow: (theme) => `0 0 0 2px ${theme.palette.background.paper}`,
        },
      }}
    />
  )
}

function TaskDragPreview({task, position}: {task: TaskRecord; position: TaskDragPreviewPosition}) {
  const taskColor = task.color || defaultTaskColor
  return (
    <Box
      role="status"
      aria-label={`正在调整任务“${task.title}”`}
      className="taskai-task-drag-preview"
      sx={{
        position: 'fixed',
        zIndex: (theme) => theme.zIndex.tooltip,
        top: `min(${position.y + 14}px, calc(100vh - 56px))`,
        left: `min(${position.x + 14}px, calc(100vw - 32px))`,
        display: 'flex',
        alignItems: 'center',
        gap: 0.75,
        maxWidth: 'min(300px, calc(100vw - 32px))',
        minHeight: 40,
        px: 1,
        py: 0.5,
        border: 1,
        borderColor: 'divider',
        borderLeft: 4,
        borderLeftStyle: 'solid',
        borderLeftColor: taskColor,
        borderRadius: '5px 16px 5px 16px',
        bgcolor: 'background.paper',
        boxShadow: (theme) => `6px 6px 0 ${theme.palette.secondary.main}`,
        pointerEvents: 'none',
      }}
    >
      <DragIndicatorIcon fontSize="small" color="action"/>
      <Typography variant="body2" sx={{fontWeight: 600, overflowWrap: 'anywhere'}}>{task.title}</Typography>
    </Box>
  )
}
