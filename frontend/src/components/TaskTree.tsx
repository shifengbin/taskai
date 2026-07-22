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
  Tooltip,
  Typography,
} from '@mui/material'
import AddOutlinedIcon from '@mui/icons-material/AddOutlined'
import ChevronRightIcon from '@mui/icons-material/ChevronRight'
import CloseOutlinedIcon from '@mui/icons-material/CloseOutlined'
import ExpandMoreIcon from '@mui/icons-material/ExpandMore'
import PlayArrowOutlinedIcon from '@mui/icons-material/PlayArrowOutlined'
import TerminalOutlinedIcon from '@mui/icons-material/TerminalOutlined'
import TaskAltOutlinedIcon from '@mui/icons-material/TaskAltOutlined'

import {taskStatusLabel, terminalStatusLabel, type TaskRecord, type TerminalRecord} from '../types'

interface ContextMenuPosition {
  taskId: string
  mouseX: number
  mouseY: number
}

interface TaskTreeProps {
  tasks: TaskRecord[]
  terminals: TerminalRecord[]
  selectedTerminalId?: string
  onSelectTask(task: TaskRecord): void
  onSelectTerminal(terminal: TerminalRecord): void
  onCreateTerminal(taskID: string): void
  onStartTask(taskID: string): void
  onFinishTask(taskID: string): void
  onCloseTerminal?(terminal: TerminalRecord): void
}

export function TaskTree({
  tasks,
  terminals,
  selectedTerminalId,
  onSelectTask,
  onSelectTerminal,
  onCreateTerminal,
  onStartTask,
  onFinishTask,
  onCloseTerminal,
}: TaskTreeProps) {
  const [expanded, setExpanded] = useState<Record<string, boolean>>({})
  const [contextMenu, setContextMenu] = useState<ContextMenuPosition | null>(null)
  const terminalsByTask = useMemo(() => {
    return terminals.reduce<Record<string, TerminalRecord[]>>((byTask, terminal) => {
      byTask[terminal.taskId] = [...(byTask[terminal.taskId] ?? []), terminal]
      return byTask
    }, {})
  }, [terminals])

  const toggleExpanded = (taskID: string) => {
    setExpanded((current) => ({...current, [taskID]: !current[taskID]}))
  }

  const requestContextMenu = (event: React.MouseEvent, task: TaskRecord) => {
    if (task.status !== 'running') {
      return
    }
    event.preventDefault()
    setContextMenu({taskId: task.id, mouseX: event.clientX + 2, mouseY: event.clientY - 6})
  }

  return (
    <Box component="nav" aria-label="任务和终端" sx={{height: '100%', overflow: 'auto'}}>
      <List disablePadding dense>
        {tasks.map((task) => {
          const childTerminals = terminalsByTask[task.id] ?? []
          const isExpanded = expanded[task.id] ?? true
          return (
            <Box key={task.id}>
              <ListItemButton
                onClick={() => onSelectTask(task)}
                onContextMenu={(event) => requestContextMenu(event, task)}
                sx={{minHeight: 48, gap: 0.5}}
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
                  secondary={task.description || taskStatusLabel[task.status]}
                  slotProps={{
                    primary: {noWrap: true, sx: {fontWeight: 600}},
                    secondary: {noWrap: true},
                  }}
                />
                <Chip label={taskStatusLabel[task.status]} size="small" color={statusColor(task.status)} variant="outlined"/>
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
              </ListItemButton>
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
        open={contextMenu !== null}
        onClose={() => setContextMenu(null)}
        anchorReference="anchorPosition"
        anchorPosition={contextMenu ? {top: contextMenu.mouseY, left: contextMenu.mouseX} : undefined}
      >
        <MenuItem
          onClick={() => {
            if (contextMenu) {
              onCreateTerminal(contextMenu.taskId)
            }
            setContextMenu(null)
          }}
        >
          <TerminalOutlinedIcon fontSize="small"/>
          <Typography component="span" sx={{ml: 1}}>新增终端</Typography>
        </MenuItem>
      </Menu>
    </Box>
  )
}

function statusColor(status: TaskRecord['status']): 'default' | 'primary' | 'success' {
  if (status === 'running') {
    return 'primary'
  }
  if (status === 'completed') {
    return 'success'
  }
  return 'default'
}
