import {useMemo, useRef, useState} from 'react'
import {
  Box,
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
import TerminalOutlinedIcon from '@mui/icons-material/TerminalOutlined'
import TaskAltOutlinedIcon from '@mui/icons-material/TaskAltOutlined'

import {defaultTaskColor, defaultTaskMenuItems, terminalDisplayName, type TaskMenuItem, type TaskRecord, type TaskStatus, type TerminalRecord} from '../types'
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

interface TaskTreeProps {
  tasks: TaskRecord[]
  terminals: TerminalRecord[]
  menuItems?: TaskMenuItem[]
  activeStatus: TaskStatus
  expandedTasks?: Record<string, boolean>
  selectedTerminalId?: string
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
  onCloseTerminal?(terminal: TerminalRecord): void
  onReorderTasks?(taskID: string, targetTaskID: string, position: TaskDropPosition): void
}

export function TaskTree({
  tasks,
  terminals,
  menuItems = defaultTaskMenuItems,
  activeStatus,
  expandedTasks,
  selectedTerminalId,
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
  onCloseTerminal,
  onReorderTasks,
}: TaskTreeProps) {
  const [localExpandedTasks, setLocalExpandedTasks] = useState<Record<string, boolean>>({})
  const [taskMenu, setTaskMenu] = useState<TaskMenuState | null>(null)
  const [draggedTaskID, setDraggedTaskID] = useState<string>()
  const [dragPreviewPosition, setDragPreviewPosition] = useState<TaskDragPreviewPosition>()
  const [dropTarget, setDropTarget] = useState<TaskDropTarget>()
  const pointerDragRef = useRef<TaskPointerDrag>()
  const suppressTaskClickRef = useRef(false)
  const terminalsByTask = useMemo(() => {
    return terminals.reduce<Record<string, TerminalRecord[]>>((byTask, terminal) => {
      byTask[terminal.taskId] = [...(byTask[terminal.taskId] ?? []), terminal]
      return byTask
    }, {})
  }, [terminals])
  const visibleTasks = useMemo(() => tasks.filter((task) => task.status === activeStatus), [activeStatus, tasks])
  const taskCounts = useMemo(() => tasks.reduce<Record<TaskStatus, number>>((counts, task) => {
    counts[task.status] += 1
    return counts
  }, {pending: 0, running: 0, completed: 0}), [tasks])
  const taskMenuTask = taskMenu ? tasks.find((task) => task.id === taskMenu.taskID) : undefined
  const draggedTask = draggedTaskID ? tasks.find((task) => task.id === draggedTaskID) : undefined
  const expanded = expandedTasks ?? localExpandedTasks
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

  const activeMenuItems = taskMenuTask ? menuItems.filter((item) => taskMenuTask.status === 'running' || item.kind === 'edit-task') : []

  return (
    <Box component="nav" aria-label="任务和终端" sx={{height: '100%', display: 'grid', gridTemplateRows: 'auto minmax(0, 1fr)'}}>
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
      <List disablePadding dense sx={{overflow: 'auto'}}>
        {visibleTasks.map((task) => {
          const childTerminals = terminalsByTask[task.id] ?? []
          const isExpanded = expanded[task.id] ?? true
          const taskColor = task.color || defaultTaskColor
          const dropPosition = dropTarget?.taskID === task.id ? dropTarget.position : undefined
          return (
            <Box
              key={task.id}
              data-task-container={task.id}
            >
              {dropPosition === 'before' && <TaskDropIndicator taskTitle={task.title} position={dropPosition}/>}
              <Tooltip
                title={task.description || '暂无任务描述'}
                placement="right"
                slotProps={{tooltip: {sx: {maxWidth: 480, whiteSpace: 'pre-wrap', overflowWrap: 'anywhere'}}}}
              >
                <ListItemButton
                  data-task-id={task.id}
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
                    bgcolor: `${taskColor}14`,
                    opacity: draggedTaskID === task.id ? 0.5 : 1,
                    outline: draggedTaskID === task.id ? '2px solid' : '2px solid transparent',
                    outlineColor: draggedTaskID === task.id ? 'primary.main' : 'transparent',
                    outlineOffset: -2,
                    cursor: 'grab',
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
                {task.status === 'pending' && (
                  <Tooltip title="执行">
                    <IconButton
                      aria-label="执行"
                      size="small"
                      onClick={(event) => {
                        event.stopPropagation()
                        onStartTask(task.id)
                      }}
                    >
                      <PlayArrowOutlinedIcon fontSize="small"/>
                    </IconButton>
                  </Tooltip>
                )}
                {task.status === 'running' && (
                  <Tooltip title="结束">
                    <IconButton
                      aria-label="结束"
                      size="small"
                      onClick={(event) => {
                        event.stopPropagation()
                        onFinishTask(task.id)
                      }}
                    >
                      <TaskAltOutlinedIcon fontSize="small"/>
                    </IconButton>
                  </Tooltip>
                )}
                <Tooltip title="任务操作">
                  <IconButton
                    aria-label="任务操作"
                    size="small"
                    onClick={(event) => {
                      event.stopPropagation()
                      setTaskMenu({taskID: task.id, anchorEl: event.currentTarget})
                    }}
                  >
                    <MoreVertIcon fontSize="small"/>
                  </IconButton>
                </Tooltip>
                </ListItemButton>
              </Tooltip>
              <Collapse in={isExpanded} timeout="auto" unmountOnExit>
                <List disablePadding dense sx={{pl: 3.25}}>
                  {childTerminals.map((terminal) => (
                    <ListItemButton
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
                      <TerminalStatusDot state={terminal.state}/>
                      {terminal.state === 'active' && onCloseTerminal && (
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
          )
        })}
      </List>
      {draggedTask && dragPreviewPosition && <TaskDragPreview task={draggedTask} position={dragPreviewPosition}/>}
      <Menu
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
      </Menu>
    </Box>
  )
}

function TaskDropIndicator({taskTitle, position}: {taskTitle: string; position: TaskDropPosition}) {
  const positionLabel = position === 'before' ? '之前' : '之后'
  return (
    <Box
      role="status"
      aria-label={`将任务插入“${taskTitle}”${positionLabel}`}
      sx={{
        height: 12,
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
          height: 3,
          borderRadius: 99,
          bgcolor: 'primary.main',
          transform: 'translateY(-50%)',
        },
        '&::after': {
          content: '""',
          position: 'absolute',
          top: '50%',
          left: -4,
          width: 10,
          height: 10,
          borderRadius: '50%',
          bgcolor: 'primary.main',
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
        borderRadius: 1,
        bgcolor: 'background.paper',
        boxShadow: 3,
        pointerEvents: 'none',
      }}
    >
      <DragIndicatorIcon fontSize="small" color="action"/>
      <Typography variant="body2" sx={{fontWeight: 600, overflowWrap: 'anywhere'}}>{task.title}</Typography>
    </Box>
  )
}
