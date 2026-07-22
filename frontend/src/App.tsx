import {type FormEvent, useEffect, useMemo, useRef, useState} from 'react'
import {
  Alert,
  AppBar,
  Box,
  Button,
  Chip,
  CssBaseline,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  FormControlLabel,
  IconButton,
  MenuItem,
  Snackbar,
  Switch,
  TextField,
  ThemeProvider,
  Toolbar,
  Tooltip,
  Typography,
  createTheme,
} from '@mui/material'
import AddOutlinedIcon from '@mui/icons-material/AddOutlined'
import ArrowDownwardOutlinedIcon from '@mui/icons-material/ArrowDownwardOutlined'
import ArrowUpwardOutlinedIcon from '@mui/icons-material/ArrowUpwardOutlined'
import FolderOutlinedIcon from '@mui/icons-material/FolderOutlined'
import LogoutOutlinedIcon from '@mui/icons-material/LogoutOutlined'
import SettingsOutlinedIcon from '@mui/icons-material/SettingsOutlined'
import TaskAltOutlinedIcon from '@mui/icons-material/TaskAltOutlined'

import {api} from './api'
import {TaskTree} from './components/TaskTree'
import {TerminalView} from './components/TerminalView'
import {applyTerminalEvent} from './state'
import {
  clampTaskTreeWidth,
  defaultTaskColor,
	defaultTaskMenuItems,
  taskStatusLabel,
  type ColorScheme,
  type SettingsRecord,
  type TaskRecord,
	type TaskMenuItem,
  type TaskStatus,
  type TerminalRecord,
} from './types'
import './App.css'

export default function App() {
  const [tasks, setTasks] = useState<TaskRecord[]>([])
  const [terminals, setTerminals] = useState<TerminalRecord[]>([])
  const [settings, setSettings] = useState<SettingsRecord>()
  const [detectedShells, setDetectedShells] = useState<string[]>([])
  const [treeWidth, setTreeWidth] = useState(360)
  const [selectedTaskID, setSelectedTaskID] = useState<string>()
  const [selectedTerminalID, setSelectedTerminalID] = useState<string>()
  const [activeTaskStatus, setActiveTaskStatus] = useState<TaskStatus>('pending')
  const [taskDialogOpen, setTaskDialogOpen] = useState(false)
  const [editingTask, setEditingTask] = useState<TaskRecord>()
  const [settingsDialogOpen, setSettingsDialogOpen] = useState(false)
  const [finishTask, setFinishTask] = useState<TaskRecord>()
  const [quitDialogOpen, setQuitDialogOpen] = useState(false)
  const [draftTitle, setDraftTitle] = useState('')
  const [draftDescription, setDraftDescription] = useState('')
  const [draftColor, setDraftColor] = useState(defaultTaskColor)
  const [settingsDraft, setSettingsDraft] = useState<SettingsRecord>()
  const [selectedTaskMenuItemID, setSelectedTaskMenuItemID] = useState<string>()
  const [message, setMessage] = useState<string>()
  const dragging = useRef(false)
  const currentTreeWidth = useRef(treeWidth)

  useEffect(() => {
    void (async () => {
      try {
        const [loadedTasks, loadedSettings, loadedShells] = await Promise.all([api.listTasks(), api.getSettings(), api.detectShells()])
        setTasks(loadedTasks)
        setSettings(loadedSettings)
        setDetectedShells(loadedShells)
        const width = clampTaskTreeWidth(loadedSettings.taskTreeWidth)
        currentTreeWidth.current = width
        setTreeWidth(width)
      } catch (error) {
        showError(error, setMessage)
      }
    })()
    return api.onTerminalEvent((event) => {
      setTerminals((current) => applyTerminalEvent(current, event))
      if (event.type === 'error') {
        setMessage(event.data || '终端发生错误')
      }
    })
  }, [])

  useEffect(() => api.onCloseRequested(() => setQuitDialogOpen(true)), [])

  const colorScheme: ColorScheme = settings?.colorScheme === 'dark' ? 'dark' : 'light'
  const theme = useMemo(() => createAppTheme(colorScheme), [colorScheme])
  const selectedTask = tasks.find((task) => task.id === selectedTaskID)
  const selectedTerminal = terminals.find((terminal) => terminal.id === selectedTerminalID && terminal.state === 'active')
  const taskMenuItems = settings?.taskMenuItems?.length ? settings.taskMenuItems : defaultTaskMenuItems
  const selectedTaskMenuItem = settingsDraft?.taskMenuItems.find((item) => item.id === selectedTaskMenuItemID)

  const openTaskDialog = (task?: TaskRecord) => {
    setEditingTask(task)
    setDraftTitle(task?.title ?? '')
    setDraftDescription(task?.description ?? '')
    setDraftColor(task?.color || defaultTaskColor)
    setTaskDialogOpen(true)
  }

  const closeTaskDialog = () => {
    setTaskDialogOpen(false)
    setEditingTask(undefined)
  }

  const saveTask = async (event: FormEvent) => {
    event.preventDefault()
    if (!draftTitle.trim()) {
      setMessage('任务标题不能为空')
      return
    }
    try {
      if (editingTask) {
        const updated = await api.updateTask(editingTask.id, draftTitle, draftDescription, draftColor)
        setTasks((current) => replaceTask(current, updated))
      } else {
        const created = await api.createTask(draftTitle, draftDescription, draftColor)
        setTasks((current) => [...current, created])
        setActiveTaskStatus('pending')
        setSelectedTaskID(created.id)
        setSelectedTerminalID(undefined)
      }
      setDraftTitle('')
      setDraftDescription('')
      setDraftColor(defaultTaskColor)
      closeTaskDialog()
    } catch (error) {
      showError(error, setMessage)
    }
  }

  const startTask = async (taskID: string) => {
    try {
      const started = await api.startTask(taskID)
      setTasks((current) => replaceTask(current, started))
      setActiveTaskStatus('running')
      setSelectedTaskID(taskID)
      setSelectedTerminalID(undefined)
    } catch (error) {
      showError(error, setMessage)
    }
  }

  const confirmFinishTask = async () => {
    if (!finishTask) {
      return
    }
    try {
      const completed = await api.finishTask(finishTask.id)
      setTasks((current) => replaceTask(current, completed))
      setActiveTaskStatus('completed')
      setTerminals((current) => current.filter((terminal) => terminal.taskId !== finishTask.id))
      if (selectedTaskID === finishTask.id) {
        setSelectedTerminalID(undefined)
      }
      setFinishTask(undefined)
    } catch (error) {
      showError(error, setMessage)
    }
  }

  const createTerminal = async (taskID: string) => {
    try {
      const created = await api.createTerminal(taskID, 100, 32)
      setTerminals((current) => [...current, {...created, output: ''}])
      setSelectedTaskID(taskID)
      setSelectedTerminalID(created.id)
    } catch (error) {
      showError(error, setMessage)
    }
  }

  const runTaskMenuCommand = async (taskID: string, item: TaskMenuItem) => {
    try {
      if (item.showTerminal) {
        const created = await api.createCommandTerminal(taskID, item.command ?? '', item.arguments ?? [], 100, 32)
        setTerminals((current) => [...current, {...created, output: ''}])
        setSelectedTaskID(taskID)
        setSelectedTerminalID(created.id)
        return
      }
      await api.runTaskCommand(taskID, item.command ?? '', item.arguments ?? [])
    } catch (error) {
      showError(error, setMessage)
    }
  }

  const openTaskFolder = async (taskID: string) => {
    try {
      await api.openTaskFolder(taskID)
    } catch (error) {
      showError(error, setMessage)
    }
  }

  const closeTerminal = async (terminal: TerminalRecord) => {
    try {
      await api.closeTerminal(terminal.taskId, terminal.id)
    } catch (error) {
      showError(error, setMessage)
    }
  }

  const saveSettings = async () => {
    if (!settingsDraft) {
      return
    }
    try {
      const saved = await api.saveSettings(settingsDraft)
      setSettings(saved)
      setSettingsDialogOpen(false)
      setSelectedTaskMenuItemID(undefined)
    } catch (error) {
      showError(error, setMessage)
    }
  }

  const updateTaskMenuItem = (itemID: string, update: Partial<TaskMenuItem>) => {
    setSettingsDraft((current) => current ? {
      ...current,
      taskMenuItems: current.taskMenuItems.map((item) => item.id === itemID ? {...item, ...update} : item),
    } : current)
  }

  const moveTaskMenuItem = (itemID: string, offset: number) => {
    setSettingsDraft((current) => {
      if (!current) {
        return current
      }
      const index = current.taskMenuItems.findIndex((item) => item.id === itemID)
      const nextIndex = index + offset
      if (index < 0 || nextIndex < 0 || nextIndex >= current.taskMenuItems.length) {
        return current
      }
      const taskMenuItems = [...current.taskMenuItems]
      const [item] = taskMenuItems.splice(index, 1)
      taskMenuItems.splice(nextIndex, 0, item)
      return {...current, taskMenuItems}
    })
  }

  const addTaskMenuItem = () => {
    const taskMenuItem = createCustomTaskMenuItem()
    setSettingsDraft((current) => current ? {
      ...current,
      taskMenuItems: [...current.taskMenuItems, taskMenuItem],
    } : current)
    setSelectedTaskMenuItemID(taskMenuItem.id)
  }

  const removeTaskMenuItem = (itemID: string) => {
    setSettingsDraft((current) => current ? {
      ...current,
      taskMenuItems: current.taskMenuItems.filter((item) => item.id !== itemID),
    } : current)
    setSelectedTaskMenuItemID(undefined)
  }

  const setPanelWidth = (nextWidth: number) => {
    const clamped = clampTaskTreeWidth(nextWidth)
    currentTreeWidth.current = clamped
    setTreeWidth(clamped)
  }

  const persistPanelWidth = async () => {
    if (!settings) {
      return
    }
    const next = {...settings, taskTreeWidth: currentTreeWidth.current}
    setSettings(next)
    try {
      const saved = await api.saveSettings(next)
      setSettings(saved)
    } catch (error) {
      showError(error, setMessage)
    }
  }

  const requestQuit = async () => {
    try {
      if (await api.hasRunningTasks()) {
        setQuitDialogOpen(true)
        return
      }
      await api.prepareQuit()
      api.quit()
    } catch (error) {
      showError(error, setMessage)
    }
  }

  const confirmQuit = async () => {
    try {
      await api.prepareQuit()
      api.quit()
    } catch (error) {
      showError(error, setMessage)
    }
  }

  return (
    <ThemeProvider theme={theme}>
      <CssBaseline/>
      <Box sx={{height: '100vh', minWidth: 720, display: 'grid', gridTemplateRows: '52px minmax(0, 1fr)', overflow: 'hidden'}}>
        <AppBar position="static" color="transparent" elevation={0} sx={{borderBottom: 1, borderColor: 'divider', bgcolor: 'background.paper'}}>
          <Toolbar variant="dense" sx={{minHeight: '52px !important', gap: 1}}>
            <TaskAltOutlinedIcon color="primary"/>
            <Typography variant="subtitle1" sx={{fontWeight: 800, letterSpacing: 0.3}}>任务工作台</Typography>
            <Box sx={{flex: 1}}/>
            <Tooltip title="新建任务">
              <IconButton aria-label="新建任务" onClick={() => openTaskDialog()} color="primary">
                <AddOutlinedIcon/>
              </IconButton>
            </Tooltip>
            <Tooltip title="设置">
              <IconButton
                aria-label="设置"
                onClick={() => {
                  const draftMenuItems = cloneTaskMenuItems(taskMenuItems)
                  setSettingsDraft(settings ? {
                    ...settings,
                    colorScheme,
                    shellPath: settings.shellPath || detectedShells[0] || '',
                    taskMenuItems: draftMenuItems,
                  } : undefined)
                  setSelectedTaskMenuItemID(draftMenuItems[0]?.id)
                  setSettingsDialogOpen(true)
                }}
              >
                <SettingsOutlinedIcon/>
              </IconButton>
            </Tooltip>
            <Tooltip title="退出应用">
              <IconButton aria-label="退出应用" onClick={() => void requestQuit()}>
                <LogoutOutlinedIcon/>
              </IconButton>
            </Tooltip>
          </Toolbar>
        </AppBar>

        <Box sx={{display: 'grid', gridTemplateColumns: `${treeWidth}px 6px minmax(0, 1fr)`, minHeight: 0}}>
          <Box sx={{minWidth: 0, borderRight: 1, borderColor: 'divider', bgcolor: 'background.paper'}}>
            <Box sx={{height: 42, display: 'flex', alignItems: 'center', px: 1.75, borderBottom: 1, borderColor: 'divider'}}>
              <Typography variant="overline" color="text.secondary">任务与终端</Typography>
            </Box>
            <Box sx={{height: 'calc(100% - 42px)'}}>
              <TaskTree
                tasks={tasks}
                terminals={terminals}
                menuItems={taskMenuItems}
                activeStatus={activeTaskStatus}
                selectedTerminalId={selectedTerminalID}
                onChangeStatus={setActiveTaskStatus}
                onSelectTask={(task) => {
                  setSelectedTaskID(task.id)
                  setSelectedTerminalID(undefined)
                }}
                onSelectTerminal={(terminal) => {
                  setSelectedTaskID(terminal.taskId)
                  setSelectedTerminalID(terminal.id)
                }}
                onCreateTerminal={(taskID) => void createTerminal(taskID)}
                onEditTask={(taskID) => {
                  const task = tasks.find((current) => current.id === taskID)
                  if (task) {
                    openTaskDialog(task)
                  }
                }}
                onOpenTaskFolder={(taskID) => void openTaskFolder(taskID)}
                onRunMenuCommand={(taskID, item) => void runTaskMenuCommand(taskID, item)}
                onStartTask={(taskID) => void startTask(taskID)}
                onFinishTask={(taskID) => setFinishTask(tasks.find((task) => task.id === taskID))}
                onCloseTerminal={(terminal) => void closeTerminal(terminal)}
              />
            </Box>
          </Box>
          <Box
            role="separator"
            aria-label="调整任务树宽度"
            onPointerDown={(event) => {
              dragging.current = true
              event.currentTarget.setPointerCapture(event.pointerId)
            }}
            onPointerMove={(event) => {
              if (dragging.current) {
                setPanelWidth(event.clientX)
              }
            }}
            onPointerUp={(event) => {
              if (!dragging.current) {
                return
              }
              dragging.current = false
              event.currentTarget.releasePointerCapture(event.pointerId)
              void persistPanelWidth()
            }}
            sx={{cursor: 'col-resize', bgcolor: 'divider', '&:hover': {bgcolor: 'primary.main'}}}
          />
          <Box sx={{minWidth: 0, minHeight: 0, bgcolor: 'background.default'}}>
            {selectedTerminal ? (
              <TerminalView
                key={selectedTerminal.id}
                terminal={selectedTerminal}
                onWrite={(data) => void api.writeTerminal(selectedTerminal.taskId, selectedTerminal.id, data).catch((error) => showError(error, setMessage))}
                onResize={(columns, rows) => void api.resizeTerminal(selectedTerminal.taskId, selectedTerminal.id, columns, rows).catch((error) => showError(error, setMessage))}
                onClose={() => void closeTerminal(selectedTerminal)}
              />
            ) : (
              <TaskDetail task={selectedTask}/>
            )}
          </Box>
        </Box>
      </Box>

      <Dialog open={taskDialogOpen} onClose={closeTaskDialog} fullWidth maxWidth="sm">
        <Box component="form" onSubmit={saveTask}>
          <DialogTitle>{editingTask ? '编辑任务' : '新建任务'}</DialogTitle>
          <DialogContent sx={{display: 'grid', gap: 2, pt: '12px !important'}}>
            <TextField autoFocus required label="标题" value={draftTitle} onChange={(event) => setDraftTitle(event.target.value)}/>
            <TextField label="任务描述" value={draftDescription} multiline minRows={3} onChange={(event) => setDraftDescription(event.target.value)}/>
            <Box sx={{display: 'flex', alignItems: 'center', gap: 1.5}}>
              <Typography component="label" htmlFor="task-color-picker" variant="body2">任务颜色</Typography>
              <input
                id="task-color-picker"
                aria-label="任务颜色"
                type="color"
                value={draftColor}
                onChange={(event) => setDraftColor(event.target.value)}
                style={{width: 48, height: 36, padding: 2, border: 'none', background: 'transparent', cursor: 'pointer'}}
              />
            </Box>
          </DialogContent>
          <DialogActions>
            <Button onClick={closeTaskDialog}>取消</Button>
            <Button type="submit" variant="contained">{editingTask ? '保存' : '创建'}</Button>
          </DialogActions>
        </Box>
      </Dialog>

      <Dialog open={settingsDialogOpen} onClose={() => setSettingsDialogOpen(false)} fullWidth maxWidth="sm">
        <DialogTitle>设置</DialogTitle>
        <DialogContent sx={{display: 'grid', gap: 2, pt: '12px !important'}}>
          <TextField
            fullWidth
            required
            label="新任务工作区根目录"
            helperText="仅影响之后开始执行的任务，已有任务保持各自目录快照。"
            value={settingsDraft?.workspaceRoot ?? ''}
            onChange={(event) => setSettingsDraft((current) => ({
              workspaceRoot: event.target.value,
              taskTreeWidth: current?.taskTreeWidth ?? treeWidth,
              colorScheme: current?.colorScheme ?? colorScheme,
              shellPath: current?.shellPath ?? settings?.shellPath ?? detectedShells[0] ?? '',
              taskMenuItems: current?.taskMenuItems ?? cloneTaskMenuItems(taskMenuItems),
            }))}
          />
          <TextField
            fullWidth
            select
            label="颜色模式"
            value={settingsDraft?.colorScheme ?? colorScheme}
            onChange={(event) => setSettingsDraft((current) => ({
              workspaceRoot: current?.workspaceRoot ?? settings?.workspaceRoot ?? '',
              taskTreeWidth: current?.taskTreeWidth ?? treeWidth,
              colorScheme: event.target.value as ColorScheme,
              shellPath: current?.shellPath ?? settings?.shellPath ?? detectedShells[0] ?? '',
              taskMenuItems: current?.taskMenuItems ?? cloneTaskMenuItems(taskMenuItems),
            }))}
          >
            <MenuItem value="light">亮色</MenuItem>
            <MenuItem value="dark">暗色</MenuItem>
          </TextField>
          <TextField
            fullWidth
            select
            label="探测到的 Shell"
            helperText="选择后会自动填入下方的 Shell 路径。"
            value={detectedShells.includes(settingsDraft?.shellPath ?? '') ? settingsDraft?.shellPath ?? '' : ''}
            onChange={(event) => {
              if (!event.target.value) {
                return
              }
              setSettingsDraft((current) => ({
                workspaceRoot: current?.workspaceRoot ?? settings?.workspaceRoot ?? '',
                taskTreeWidth: current?.taskTreeWidth ?? treeWidth,
                colorScheme: current?.colorScheme ?? colorScheme,
                shellPath: event.target.value,
                taskMenuItems: current?.taskMenuItems ?? cloneTaskMenuItems(taskMenuItems),
              }))
            }}
          >
            <MenuItem value="">手动设置路径</MenuItem>
            {detectedShells.map((shellPath) => <MenuItem key={shellPath} value={shellPath}>{shellPath}</MenuItem>)}
          </TextField>
          <TextField
            fullWidth
            required
            label="Shell 路径"
            helperText="此路径决定从任务右键菜单新建的终端所启动的 Shell。"
            value={settingsDraft?.shellPath ?? ''}
            onChange={(event) => setSettingsDraft((current) => ({
              workspaceRoot: current?.workspaceRoot ?? settings?.workspaceRoot ?? '',
              taskTreeWidth: current?.taskTreeWidth ?? treeWidth,
              colorScheme: current?.colorScheme ?? colorScheme,
              shellPath: event.target.value,
              taskMenuItems: current?.taskMenuItems ?? cloneTaskMenuItems(taskMenuItems),
            }))}
          />
          <Box sx={{display: 'grid', gap: 1.25}}>
            <Box sx={{display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 1}}>
              <Typography variant="subtitle2">任务操作菜单</Typography>
              <Button size="small" variant="outlined" onClick={addTaskMenuItem}>新增自定义菜单项</Button>
            </Box>
            <Typography variant="caption" color="text.secondary">系统菜单项仅可调整顺序；自定义命令的启动参数每行一个。</Typography>
            <Box sx={{display: 'grid', gap: 0.75}}>
              {settingsDraft?.taskMenuItems.map((item, index) => (
                <Box key={item.id} sx={{display: 'flex', alignItems: 'center', gap: 0.5}}>
                  <Button
                    aria-label={`编辑菜单项 ${item.name}`}
                    color={selectedTaskMenuItemID === item.id ? 'primary' : 'inherit'}
                    onClick={() => setSelectedTaskMenuItemID(item.id)}
                    sx={{justifyContent: 'flex-start', flex: 1}}
                  >
                    {item.name}
                  </Button>
                  <IconButton aria-label={`上移 ${item.name}`} disabled={index === 0} onClick={() => moveTaskMenuItem(item.id, -1)} size="small"><ArrowUpwardOutlinedIcon fontSize="inherit"/></IconButton>
                  <IconButton aria-label={`下移 ${item.name}`} disabled={index === settingsDraft.taskMenuItems.length - 1} onClick={() => moveTaskMenuItem(item.id, 1)} size="small"><ArrowDownwardOutlinedIcon fontSize="inherit"/></IconButton>
                </Box>
              ))}
            </Box>
            {selectedTaskMenuItem && selectedTaskMenuItem.kind !== 'command' ? (
              <Typography variant="body2" color="text.secondary">系统固定菜单项仅可调整顺序。</Typography>
            ) : selectedTaskMenuItem ? (
              <Box sx={{display: 'grid', gap: 1.5, pt: 0.5}}>
                <TextField
                  required
                  label="菜单名称"
                  value={selectedTaskMenuItem.name}
                  onChange={(event) => updateTaskMenuItem(selectedTaskMenuItem.id, {name: event.target.value})}
                />
                <TextField
                  required
                  label="启动命令"
                  value={selectedTaskMenuItem.command ?? ''}
                  onChange={(event) => updateTaskMenuItem(selectedTaskMenuItem.id, {command: event.target.value})}
                />
                <TextField
                  label="启动参数（每行一个）"
                  helperText="每行代表一个启动参数。"
                  minRows={2}
                  multiline
                  value={(selectedTaskMenuItem.arguments ?? []).join('\n')}
                  onChange={(event) => updateTaskMenuItem(selectedTaskMenuItem.id, {arguments: event.target.value.split('\n')})}
                />
                <FormControlLabel
                  control={<Switch checked={selectedTaskMenuItem.showTerminal} onChange={(event) => updateTaskMenuItem(selectedTaskMenuItem.id, {showTerminal: event.target.checked})}/>}
                  label="显示终端"
                />
                <Box>
                  <Button color="error" size="small" onClick={() => removeTaskMenuItem(selectedTaskMenuItem.id)}>删除菜单项</Button>
                </Box>
              </Box>
            ) : null}
          </Box>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setSettingsDialogOpen(false)}>取消</Button>
          <Button variant="contained" onClick={() => void saveSettings()}>保存</Button>
        </DialogActions>
      </Dialog>

      <Dialog open={Boolean(finishTask)} onClose={() => setFinishTask(undefined)} maxWidth="xs" fullWidth>
        <DialogTitle>结束任务？</DialogTitle>
        <DialogContent>
          <Typography variant="body2" color="text.secondary">
            确认后将关闭“{finishTask?.title}”的全部终端，并删除其工作目录及所有内容。此操作无法撤销。
          </Typography>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setFinishTask(undefined)}>取消</Button>
          <Button color="error" variant="contained" onClick={() => void confirmFinishTask()}>结束并删除</Button>
        </DialogActions>
      </Dialog>

      <Dialog open={quitDialogOpen} onClose={() => setQuitDialogOpen(false)} maxWidth="xs" fullWidth>
        <DialogTitle>仍有执行中的任务</DialogTitle>
        <DialogContent>
          <Typography variant="body2" color="text.secondary">
            退出会关闭全部终端，但不会改变任务状态或删除工作目录。下次启动后这些任务仍显示为执行中。
          </Typography>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setQuitDialogOpen(false)}>取消</Button>
          <Button variant="contained" onClick={() => void confirmQuit()}>关闭终端并退出</Button>
        </DialogActions>
      </Dialog>

      <Snackbar open={Boolean(message)} autoHideDuration={5000} onClose={() => setMessage(undefined)}>
        <Alert severity="error" variant="filled" onClose={() => setMessage(undefined)}>{message}</Alert>
      </Snackbar>
    </ThemeProvider>
  )
}

function createAppTheme(colorScheme: ColorScheme) {
  return createTheme({
    palette: {
      mode: colorScheme,
      primary: {main: '#0f766e'},
      background: colorScheme === 'dark'
        ? {default: '#0f172a', paper: '#111827'}
        : {default: '#f8fafc', paper: '#ffffff'},
    },
    shape: {borderRadius: 8},
    typography: {fontFamily: 'Inter, "Noto Sans SC", system-ui, sans-serif'},
  })
}

function TaskDetail({task}: {task?: TaskRecord}) {
  if (!task) {
    return (
      <Box sx={{height: '100%', display: 'grid', placeItems: 'center', color: 'text.secondary', textAlign: 'center', p: 3}}>
        <Box>
          <FolderOutlinedIcon color="disabled" sx={{fontSize: 42, mb: 1}}/>
          <Typography>从左侧选择任务，或创建一个新任务开始。</Typography>
        </Box>
      </Box>
    )
  }
  return (
    <Box sx={{height: '100%', overflow: 'auto', p: {xs: 3, md: 5}, maxWidth: 900}}>
      <Box sx={{display: 'flex', alignItems: 'center', gap: 1, mb: 2}}>
        <Typography variant="h5" sx={{fontWeight: 750}}>{task.title}</Typography>
        <Chip label={taskStatusLabel[task.status]} size="small" variant="outlined"/>
      </Box>
      <Typography variant="body1" sx={{whiteSpace: 'pre-wrap', color: task.description ? 'text.primary' : 'text.secondary', mb: 4}}>
        {task.description || '暂无任务描述'}
      </Typography>
      {task.status === 'running' && task.workspacePath && (
        <Box sx={{display: 'grid', gap: 0.5}}>
          <Typography variant="overline" color="text.secondary">工作目录</Typography>
          <Typography variant="body2" sx={{fontFamily: 'ui-monospace, monospace', overflowWrap: 'anywhere'}}>{task.workspacePath}</Typography>
        </Box>
      )}
    </Box>
  )
}

function replaceTask(tasks: TaskRecord[], next: TaskRecord): TaskRecord[] {
  return tasks.map((task) => task.id === next.id ? next : task)
}

function cloneTaskMenuItems(items: TaskMenuItem[]): TaskMenuItem[] {
  return items.map((item) => item.arguments ? {...item, arguments: [...item.arguments]} : {...item})
}

function createCustomTaskMenuItem(): TaskMenuItem {
  const randomID = globalThis.crypto?.randomUUID?.() ?? `${Date.now()}-${Math.random().toString(36).slice(2)}`
  return {
    id: `custom-${randomID}`,
    kind: 'command',
    name: '自定义命令',
    command: '',
    arguments: [],
    showTerminal: true,
  }
}

function showError(error: unknown, setMessage: (message: string) => void) {
  setMessage(error instanceof Error ? error.message : String(error))
}
