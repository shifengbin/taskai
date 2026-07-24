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
  Divider,
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
  const [initialLoadComplete, setInitialLoadComplete] = useState(false)
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
  const [taskMenuItemDraft, setTaskMenuItemDraft] = useState<TaskMenuItem>()
  const [taskMenuItemEditorMode, setTaskMenuItemEditorMode] = useState<'create' | 'edit'>()
  const [message, setMessage] = useState<string>()
  const dragging = useRef(false)
  const currentTreeWidth = useRef(treeWidth)

  useEffect(() => {
    void (async () => {
      try {
        const [loadedTasks, loadedSettings, loadedShells] = await Promise.all([api.listTasks(), api.getSettings(), api.detectShells()])
        setTasks(loadedTasks)
        setSettings(loadedSettings)
        setActiveTaskStatus(loadedSettings.activeTaskStatus ?? 'pending')
        setDetectedShells(loadedShells)
        const width = clampTaskTreeWidth(loadedSettings.taskTreeWidth)
        currentTreeWidth.current = width
        setTreeWidth(width)
      } catch (error) {
        showError(error, setMessage)
      } finally {
        setInitialLoadComplete(true)
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

  if (!initialLoadComplete) {
    return (
      <ThemeProvider theme={theme}>
        <CssBaseline/>
        <StartupScreen/>
      </ThemeProvider>
    )
  }

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
        void changeActiveTaskStatus('pending')
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
      void changeActiveTaskStatus('running')
      setSelectedTaskID(taskID)
      setSelectedTerminalID(undefined)
    } catch (error) {
      showError(error, setMessage)
    }
  }

  const reorderTasks = async (taskID: string, targetTaskID: string, position: 'before' | 'after') => {
    const sourceTask = tasks.find((task) => task.id === taskID)
    if (!sourceTask || sourceTask.status !== activeTaskStatus) {
      return
    }
    const taskIDs = tasks.filter((task) => task.status === sourceTask.status).map((task) => task.id)
    if (!taskIDs.includes(targetTaskID) || taskID === targetTaskID) {
      return
    }
    const orderedTaskIDs = taskIDs.filter((currentTaskID) => currentTaskID !== taskID)
    const targetIndex = orderedTaskIDs.indexOf(targetTaskID)
    if (targetIndex < 0) {
      return
    }
    orderedTaskIDs.splice(position === 'before' ? targetIndex : targetIndex + 1, 0, taskID)
    try {
      setTasks(await api.reorderTasks(sourceTask.status, orderedTaskIDs))
    } catch (error) {
      showError(error, setMessage)
      try {
        setTasks(await api.listTasks())
      } catch (reloadError) {
        showError(reloadError, setMessage)
      }
    }
  }

  const confirmFinishTask = async () => {
    if (!finishTask) {
      return
    }
    try {
      const completed = await api.finishTask(finishTask.id)
      setTasks((current) => replaceTask(current, completed))
      void changeActiveTaskStatus('completed')
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
      closeTaskMenuItemEditor()
    } catch (error) {
      showError(error, setMessage)
    }
  }

  const changeActiveTaskStatus = async (status: TaskStatus) => {
    const previousStatus = activeTaskStatus
    setActiveTaskStatus(status)
    if (!settings) {
      return
    }
    const next = {...settings, activeTaskStatus: status}
    setSettings(next)
    try {
      const saved = await api.saveSettings(next)
      setSettings(saved)
    } catch (error) {
      setActiveTaskStatus(previousStatus)
      setSettings(settings)
      showError(error, setMessage)
    }
  }

  const closeSettingsDialog = () => {
    setSettingsDialogOpen(false)
    closeTaskMenuItemEditor()
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

  const openTaskMenuItemEditor = (item?: TaskMenuItem) => {
    setTaskMenuItemEditorMode(item ? 'edit' : 'create')
    setTaskMenuItemDraft(item ? cloneTaskMenuItem(item) : createCustomTaskMenuItem())
  }

  const closeTaskMenuItemEditor = () => {
    setTaskMenuItemDraft(undefined)
    setTaskMenuItemEditorMode(undefined)
  }

  const saveTaskMenuItem = (event: FormEvent) => {
    event.preventDefault()
    if (!taskMenuItemDraft || !taskMenuItemEditorMode) {
      return
    }
    const item = {
      ...taskMenuItemDraft,
      name: taskMenuItemDraft.name.trim(),
      command: taskMenuItemDraft.command?.trim(),
      arguments: taskMenuItemDraft.arguments?.filter((argument) => argument.trim()),
    }
    if (!item.name || !item.command) {
      setMessage('菜单名称和启动命令不能为空')
      return
    }
    setSettingsDraft((current) => current ? {
      ...current,
      taskMenuItems: taskMenuItemEditorMode === 'create'
        ? [...current.taskMenuItems, item]
        : current.taskMenuItems.map((currentItem) => currentItem.id === item.id ? item : currentItem),
    } : current)
    closeTaskMenuItemEditor()
  }

  const deleteTaskMenuItem = () => {
    if (!taskMenuItemDraft || taskMenuItemEditorMode !== 'edit') {
      return
    }
    setSettingsDraft((current) => current ? {
      ...current,
      taskMenuItems: current.taskMenuItems.filter((item) => item.id !== taskMenuItemDraft.id),
    } : current)
    closeTaskMenuItemEditor()
  }

  const updateSettingsDraft = (update: Partial<SettingsRecord>) => {
    setSettingsDraft((current) => ({
      workspaceRoot: current?.workspaceRoot ?? settings?.workspaceRoot ?? '',
      taskTreeWidth: current?.taskTreeWidth ?? treeWidth,
      colorScheme: current?.colorScheme ?? colorScheme,
      shellPath: current?.shellPath ?? settings?.shellPath ?? detectedShells[0] ?? '',
      taskMenuItems: current?.taskMenuItems ?? cloneTaskMenuItems(taskMenuItems),
      activeTaskStatus: current?.activeTaskStatus ?? settings?.activeTaskStatus ?? activeTaskStatus,
      ...update,
    }))
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
                  setTaskMenuItemDraft(undefined)
                  setTaskMenuItemEditorMode(undefined)
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
            <Box sx={{height: 42, display: 'flex', alignItems: 'center', px: 1.25, borderBottom: 1, borderColor: 'divider'}}>
              <Typography variant="overline" color="text.secondary">任务与终端</Typography>
              <Box sx={{flex: 1}}/>
              <Tooltip title="新建任务">
                <IconButton aria-label="新建任务" onClick={() => openTaskDialog()} color="primary" size="small">
                  <AddOutlinedIcon fontSize="small"/>
                </IconButton>
              </Tooltip>
            </Box>
            <Box sx={{height: 'calc(100% - 42px)'}}>
              <TaskTree
                tasks={tasks}
                terminals={terminals}
                menuItems={taskMenuItems}
                activeStatus={activeTaskStatus}
                selectedTerminalId={selectedTerminalID}
                onChangeStatus={(status) => void changeActiveTaskStatus(status)}
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
                onReorderTasks={(taskID, targetTaskID, position) => void reorderTasks(taskID, targetTaskID, position)}
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

      <Dialog open={settingsDialogOpen} onClose={closeSettingsDialog} fullWidth maxWidth="md">
        <DialogTitle>设置</DialogTitle>
        <DialogContent sx={{display: 'grid', gap: 3, pt: '12px !important'}}>
          <Box component="section" sx={{display: 'grid', gap: 1.5}}>
            <Divider textAlign="left"><Typography variant="subtitle2">工作区与外观</Typography></Divider>
            <TextField
              fullWidth
              required
              label="新任务工作区根目录"
              helperText="仅影响之后开始执行的任务，已有任务保持各自目录快照。"
              value={settingsDraft?.workspaceRoot ?? ''}
              onChange={(event) => updateSettingsDraft({workspaceRoot: event.target.value})}
            />
            <TextField
              fullWidth
              select
              label="颜色模式"
              value={settingsDraft?.colorScheme ?? colorScheme}
              onChange={(event) => updateSettingsDraft({colorScheme: event.target.value as ColorScheme})}
            >
              <MenuItem value="light">亮色</MenuItem>
              <MenuItem value="dark">暗色</MenuItem>
            </TextField>
          </Box>

          <Box component="section" sx={{display: 'grid', gap: 1.5}}>
            <Divider textAlign="left"><Typography variant="subtitle2">终端 Shell</Typography></Divider>
            <TextField
              fullWidth
              select
              label="探测到的 Shell"
              helperText="选择后会自动填入下方的 Shell 路径。"
              value={detectedShells.includes(settingsDraft?.shellPath ?? '') ? settingsDraft?.shellPath ?? '' : ''}
              onChange={(event) => {
                if (event.target.value) {
                  updateSettingsDraft({shellPath: event.target.value})
                }
              }}
            >
              <MenuItem value="">手动设置路径</MenuItem>
              {detectedShells.map((shellPath) => <MenuItem key={shellPath} value={shellPath}>{shellPath}</MenuItem>)}
            </TextField>
            <TextField
              fullWidth
              required
              label="Shell 路径"
              helperText="此 Shell 会启动任务终端，并提供自定义命令所需的初始化环境。"
              value={settingsDraft?.shellPath ?? ''}
              onChange={(event) => updateSettingsDraft({shellPath: event.target.value})}
            />
          </Box>

          <Box component="section" sx={{display: 'grid', gap: 1.5}}>
            <Divider textAlign="left"><Typography variant="subtitle2">任务操作菜单</Typography></Divider>
            <Box sx={{display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 2}}>
              <Typography variant="body2" color="text.secondary">右键菜单与“任务操作”下拉菜单共用此顺序。系统项仅可调序。</Typography>
              <Button size="small" variant="contained" onClick={() => openTaskMenuItemEditor()}>新增菜单项</Button>
            </Box>
            <Box sx={{border: 1, borderColor: 'divider', borderRadius: 1, overflow: 'hidden'}}>
              {settingsDraft?.taskMenuItems.map((item, index) => (
                <Box key={item.id} sx={{display: 'flex', alignItems: 'center', gap: 1, px: 1.5, py: 1, borderBottom: index === settingsDraft.taskMenuItems.length - 1 ? 0 : 1, borderColor: 'divider'}}>
                  <Box sx={{minWidth: 0, flex: 1}}>
                    <Typography variant="body2" sx={{fontWeight: 650}} noWrap>{item.name}</Typography>
                    <Typography variant="caption" color="text.secondary" noWrap>{item.kind === 'command' ? item.command : '系统固定操作'}</Typography>
                  </Box>
                  <Chip label={item.kind === 'command' ? item.showTerminal ? '显示终端' : '后台启动' : '系统固定'} size="small" variant="outlined"/>
                  {item.kind === 'command' && <Button aria-label={`编辑菜单项 ${item.name}`} size="small" onClick={() => openTaskMenuItemEditor(item)}>编辑</Button>}
                  <IconButton aria-label={`上移 ${item.name}`} disabled={index === 0} onClick={() => moveTaskMenuItem(item.id, -1)} size="small"><ArrowUpwardOutlinedIcon fontSize="inherit"/></IconButton>
                  <IconButton aria-label={`下移 ${item.name}`} disabled={index === settingsDraft.taskMenuItems.length - 1} onClick={() => moveTaskMenuItem(item.id, 1)} size="small"><ArrowDownwardOutlinedIcon fontSize="inherit"/></IconButton>
                </Box>
              ))}
            </Box>
          </Box>
        </DialogContent>
        <DialogActions>
          <Button onClick={closeSettingsDialog}>取消</Button>
          <Button variant="contained" onClick={() => void saveSettings()}>保存</Button>
        </DialogActions>
      </Dialog>

      {taskMenuItemDraft && <Dialog open onClose={closeTaskMenuItemEditor} fullWidth maxWidth="sm">
        <Box component="form" onSubmit={saveTaskMenuItem}>
          <DialogTitle>{taskMenuItemEditorMode === 'create' ? '新增菜单项' : '编辑菜单项'}</DialogTitle>
          <DialogContent sx={{display: 'grid', gap: 2, pt: '12px !important'}}>
            <Typography variant="body2" color="text.secondary">填写完成后确认菜单项；主设置保存前，变更不会持久化。</Typography>
            <TextField
              autoFocus
              required
              label="菜单名称"
              value={taskMenuItemDraft.name}
              onChange={(event) => setTaskMenuItemDraft((current) => current ? {...current, name: event.target.value} : current)}
            />
            <TextField
              required
              label="启动命令"
              value={taskMenuItemDraft.command ?? ''}
              onChange={(event) => setTaskMenuItemDraft((current) => current ? {...current, command: event.target.value} : current)}
            />
            <TextField
              label="启动参数（每行一个）"
              helperText="每行代表一个启动参数。"
              minRows={2}
              multiline
              value={(taskMenuItemDraft.arguments ?? []).join('\n')}
              onChange={(event) => setTaskMenuItemDraft((current) => current ? {...current, arguments: event.target.value.split('\n')} : current)}
            />
            <FormControlLabel
              control={<Switch checked={taskMenuItemDraft.showTerminal} onChange={(event) => setTaskMenuItemDraft((current) => current ? {...current, showTerminal: event.target.checked} : current)}/>}
              label="显示终端"
            />
          </DialogContent>
          <DialogActions>
            {taskMenuItemEditorMode === 'edit' && <Button color="error" onClick={deleteTaskMenuItem}>删除菜单项</Button>}
            <Box sx={{flex: 1}}/>
            <Button onClick={closeTaskMenuItemEditor}>取消</Button>
            <Button type="submit" variant="contained">{taskMenuItemEditorMode === 'create' ? '添加菜单项' : '保存菜单项'}</Button>
          </DialogActions>
        </Box>
      </Dialog>}

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
      divider: colorScheme === 'dark' ? '#1e293b' : '#cbd5e1',
    },
    shape: {borderRadius: 8},
    typography: {fontFamily: 'Inter, "Noto Sans SC", system-ui, sans-serif'},
  })
}

function StartupScreen() {
  return (
    <Box
      role="status"
      aria-label="正在加载任务工作台"
      sx={{height: '100vh', minWidth: 720, display: 'grid', placeItems: 'center', bgcolor: 'background.default'}}
    >
      <Box sx={{display: 'grid', placeItems: 'center', gap: 1.5, color: 'text.secondary'}}>
        <TaskAltOutlinedIcon color="primary" sx={{fontSize: 40}}/>
        <Typography variant="body2">正在加载任务工作台</Typography>
      </Box>
    </Box>
  )
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
  return items.map(cloneTaskMenuItem)
}

function cloneTaskMenuItem(item: TaskMenuItem): TaskMenuItem {
  return item.arguments ? {...item, arguments: [...item.arguments]} : {...item}
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
