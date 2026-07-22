import {type FormEvent, useEffect, useRef, useState} from 'react'
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
  IconButton,
  Snackbar,
  TextField,
  ThemeProvider,
  Toolbar,
  Tooltip,
  Typography,
  createTheme,
} from '@mui/material'
import AddOutlinedIcon from '@mui/icons-material/AddOutlined'
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
  taskStatusLabel,
  type SettingsRecord,
  type TaskRecord,
  type TerminalRecord,
} from './types'
import './App.css'

const theme = createTheme({
  palette: {
    mode: 'light',
    primary: {main: '#0f766e'},
    background: {default: '#f8fafc', paper: '#ffffff'},
  },
  shape: {borderRadius: 8},
  typography: {fontFamily: 'Inter, "Noto Sans SC", system-ui, sans-serif'},
})

export default function App() {
  const [tasks, setTasks] = useState<TaskRecord[]>([])
  const [terminals, setTerminals] = useState<TerminalRecord[]>([])
  const [settings, setSettings] = useState<SettingsRecord>()
  const [treeWidth, setTreeWidth] = useState(360)
  const [selectedTaskID, setSelectedTaskID] = useState<string>()
  const [selectedTerminalID, setSelectedTerminalID] = useState<string>()
  const [taskDialogOpen, setTaskDialogOpen] = useState(false)
  const [settingsDialogOpen, setSettingsDialogOpen] = useState(false)
  const [finishTask, setFinishTask] = useState<TaskRecord>()
  const [quitDialogOpen, setQuitDialogOpen] = useState(false)
  const [draftTitle, setDraftTitle] = useState('')
  const [draftDescription, setDraftDescription] = useState('')
  const [settingsDraft, setSettingsDraft] = useState<SettingsRecord>()
  const [message, setMessage] = useState<string>()
  const dragging = useRef(false)
  const currentTreeWidth = useRef(treeWidth)

  useEffect(() => {
    void (async () => {
      try {
        const [loadedTasks, loadedSettings] = await Promise.all([api.listTasks(), api.getSettings()])
        setTasks(loadedTasks)
        setSettings(loadedSettings)
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

  const selectedTask = tasks.find((task) => task.id === selectedTaskID)
  const selectedTerminal = terminals.find((terminal) => terminal.id === selectedTerminalID)

  const createTask = async (event: FormEvent) => {
    event.preventDefault()
    if (!draftTitle.trim()) {
      setMessage('任务标题不能为空')
      return
    }
    try {
      const created = await api.createTask(draftTitle, draftDescription)
      setTasks((current) => [...current, created])
      setSelectedTaskID(created.id)
      setSelectedTerminalID(undefined)
      setDraftTitle('')
      setDraftDescription('')
      setTaskDialogOpen(false)
    } catch (error) {
      showError(error, setMessage)
    }
  }

  const startTask = async (taskID: string) => {
    try {
      const started = await api.startTask(taskID)
      setTasks((current) => replaceTask(current, started))
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
    } catch (error) {
      showError(error, setMessage)
    }
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
      await api.saveSettings(next)
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
              <IconButton aria-label="新建任务" onClick={() => setTaskDialogOpen(true)} color="primary">
                <AddOutlinedIcon/>
              </IconButton>
            </Tooltip>
            <Tooltip title="设置">
              <IconButton
                aria-label="设置"
                onClick={() => {
                  setSettingsDraft(settings)
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
                selectedTerminalId={selectedTerminalID}
                onSelectTask={(task) => {
                  setSelectedTaskID(task.id)
                  setSelectedTerminalID(undefined)
                }}
                onSelectTerminal={(terminal) => {
                  setSelectedTaskID(terminal.taskId)
                  setSelectedTerminalID(terminal.id)
                }}
                onCreateTerminal={(taskID) => void createTerminal(taskID)}
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

      <Dialog open={taskDialogOpen} onClose={() => setTaskDialogOpen(false)} fullWidth maxWidth="sm">
        <Box component="form" onSubmit={createTask}>
          <DialogTitle>新建任务</DialogTitle>
          <DialogContent sx={{display: 'grid', gap: 2, pt: '12px !important'}}>
            <TextField autoFocus required label="标题" value={draftTitle} onChange={(event) => setDraftTitle(event.target.value)}/>
            <TextField label="任务描述" value={draftDescription} multiline minRows={3} onChange={(event) => setDraftDescription(event.target.value)}/>
          </DialogContent>
          <DialogActions>
            <Button onClick={() => setTaskDialogOpen(false)}>取消</Button>
            <Button type="submit" variant="contained">创建</Button>
          </DialogActions>
        </Box>
      </Dialog>

      <Dialog open={settingsDialogOpen} onClose={() => setSettingsDialogOpen(false)} fullWidth maxWidth="sm">
        <DialogTitle>设置</DialogTitle>
        <DialogContent sx={{pt: '12px !important'}}>
          <TextField
            fullWidth
            required
            label="新任务工作区根目录"
            helperText="仅影响之后开始执行的任务，已有任务保持各自目录快照。"
            value={settingsDraft?.workspaceRoot ?? ''}
            onChange={(event) => setSettingsDraft((current) => ({
              workspaceRoot: event.target.value,
              taskTreeWidth: current?.taskTreeWidth ?? treeWidth,
            }))}
          />
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

function showError(error: unknown, setMessage: (message: string) => void) {
  setMessage(error instanceof Error ? error.message : String(error))
}
