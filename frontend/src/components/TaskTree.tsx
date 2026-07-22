import {useMemo, useState} from 'react'
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
import EditOutlinedIcon from '@mui/icons-material/EditOutlined'
import ExpandMoreIcon from '@mui/icons-material/ExpandMore'
import FolderOpenOutlinedIcon from '@mui/icons-material/FolderOpenOutlined'
import MoreVertIcon from '@mui/icons-material/MoreVert'
import OpenInNewOutlinedIcon from '@mui/icons-material/OpenInNewOutlined'
import PlayArrowOutlinedIcon from '@mui/icons-material/PlayArrowOutlined'
import TerminalOutlinedIcon from '@mui/icons-material/TerminalOutlined'
import TaskAltOutlinedIcon from '@mui/icons-material/TaskAltOutlined'

import {defaultTaskColor, defaultTaskMenuItems, terminalStatusLabel, type TaskMenuItem, type TaskRecord, type TaskStatus, type TerminalRecord} from '../types'

interface TaskMenuState {
  taskID: string
  anchorEl?: HTMLElement
  position?: {
    top: number
    left: number
  }
}

interface TaskTreeProps {
  tasks: TaskRecord[]
  terminals: TerminalRecord[]
  menuItems?: TaskMenuItem[]
  activeStatus: TaskStatus
  selectedTerminalId?: string
  onChangeStatus(status: TaskStatus): void
  onSelectTask(task: TaskRecord): void
  onSelectTerminal(terminal: TerminalRecord): void
  onCreateTerminal(taskID: string): void
  onEditTask(taskID: string): void
  onOpenTaskFolder(taskID: string): void
  onRunMenuCommand?(taskID: string, item: TaskMenuItem): void
  onStartTask(taskID: string): void
  onFinishTask(taskID: string): void
  onCloseTerminal?(terminal: TerminalRecord): void
}

export function TaskTree({
  tasks,
  terminals,
  menuItems = defaultTaskMenuItems,
  activeStatus,
  selectedTerminalId,
  onChangeStatus,
  onSelectTask,
  onSelectTerminal,
  onCreateTerminal,
  onEditTask,
  onOpenTaskFolder,
  onRunMenuCommand,
  onStartTask,
  onFinishTask,
  onCloseTerminal,
}: TaskTreeProps) {
  const [expanded, setExpanded] = useState<Record<string, boolean>>({})
  const [taskMenu, setTaskMenu] = useState<TaskMenuState | null>(null)
  const terminalsByTask = useMemo(() => {
    return terminals.reduce<Record<string, TerminalRecord[]>>((byTask, terminal) => {
      if (terminal.state === 'exited') {
        return byTask
      }
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

  const toggleExpanded = (taskID: string) => {
    setExpanded((current) => ({...current, [taskID]: !current[taskID]}))
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
      onRunMenuCommand?.(taskMenu.taskID, item)
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
          return (
            <Box key={task.id}>
              <Tooltip
                title={task.description || '暂无任务描述'}
                placement="right"
                slotProps={{tooltip: {sx: {maxWidth: 480, whiteSpace: 'pre-wrap', overflowWrap: 'anywhere'}}}}
              >
                <ListItemButton
                  data-task-id={task.id}
                  onClick={() => onSelectTask(task)}
                  onContextMenu={(event) => requestContextMenu(event, task)}
                  style={{borderLeftColor: taskColor}}
                  sx={{minHeight: 48, gap: 0.5, borderLeft: 4, borderLeftStyle: 'solid', bgcolor: `${taskColor}14`}}
                >
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
                  {childTerminals.map((terminal, index) => (
                    <ListItemButton
                      key={terminal.id}
                      selected={terminal.id === selectedTerminalId}
                      onClick={() => onSelectTerminal(terminal)}
                      sx={{minHeight: 38, gap: 0.75}}
                    >
                      <TerminalOutlinedIcon fontSize="small" color={terminal.state === 'active' ? 'primary' : 'disabled'}/>
                      <ListItemText
                        primary={`终端 ${index + 1}`}
                        slotProps={{primary: {noWrap: true, variant: 'body2'}}}
                      />
                      <Chip label={terminalStatusLabel[terminal.state]} size="small" variant="outlined"/>
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
            </Box>
          )
        })}
      </List>
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
