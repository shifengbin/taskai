import {type Dispatch, type FormEvent, type ReactNode, type SetStateAction, useEffect, useMemo, useRef, useState} from 'react'
import {
  Accordion,
  AccordionDetails,
  AccordionSummary,
  Alert,
  AppBar,
  Box,
  Button,
	Checkbox,
  Chip,
  CssBaseline,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  FormControlLabel,
  IconButton,
  MenuItem,
  Popover,
  Snackbar,
  Switch,
  Tab,
  Tabs,
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
import DeleteOutlineOutlinedIcon from '@mui/icons-material/DeleteOutlineOutlined'
import ExpandMoreOutlinedIcon from '@mui/icons-material/ExpandMoreOutlined'
import FolderOutlinedIcon from '@mui/icons-material/FolderOutlined'
import HelpOutlinedIcon from '@mui/icons-material/HelpOutlined'
import LogoutOutlinedIcon from '@mui/icons-material/LogoutOutlined'
import SettingsOutlinedIcon from '@mui/icons-material/SettingsOutlined'
import TaskAltOutlinedIcon from '@mui/icons-material/TaskAltOutlined'
import UnfoldLessOutlinedIcon from '@mui/icons-material/UnfoldLessOutlined'
import UnfoldMoreOutlinedIcon from '@mui/icons-material/UnfoldMoreOutlined'

import {api} from './api'
import {TaskTree} from './components/TaskTree'
import {TerminalView} from './components/TerminalView'
import {
	applyRealtimeStatusToTasks,
	applyRealtimeStatusToTerminals,
  applyTerminalEvent,
	bufferPendingRealtimeStatusEvent,
  bufferPendingTerminalEvent,
  clearTaskTerminalTracking,
  mergePendingTerminalEvents,
  parseTerminalEventTitle,
  registerTerminal,
	shouldReportTerminalTitleActivity,
  terminalEventKey,
  type PendingTerminalEvent,
} from './state'
import type {TerminalTitleParserState} from './terminal-title'
import {
  clampTaskTreeWidth,
  defaultTaskColor,
	defaultTaskMenuItems,
	type ExtraInfo,
	type ExtraInfoField,
	type ExtraInfoParameterInputType,
	type LifecycleCommand,
	type LifecycleCommandChain,
	type LifecycleHook,
	lifecycleHooks,
	taskStatusLabel,
  type ColorScheme,
	type ExtraInfoTemplate,
  type TaskScript,
  type SettingsRecord,
	type TaskExtraInfo,
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
	const [extraInfoTemplates, setExtraInfoTemplates] = useState<ExtraInfoTemplate[]>([])
	const [extraInfos, setExtraInfos] = useState<ExtraInfo[]>([])
  const [detectedShells, setDetectedShells] = useState<string[]>([])
  const [initialLoadComplete, setInitialLoadComplete] = useState(false)
  const [treeWidth, setTreeWidth] = useState(360)
  const [selectedTaskID, setSelectedTaskID] = useState<string>()
  const [selectedTerminalID, setSelectedTerminalID] = useState<string>()
  const [activeTaskStatus, setActiveTaskStatus] = useState<TaskStatus>('pending')
  const [expandedTasks, setExpandedTasks] = useState<Record<string, boolean>>({})
  const [taskDialogOpen, setTaskDialogOpen] = useState(false)
	const [taskExtraInfoDraft, setTaskExtraInfoDraft] = useState<TaskExtraInfo[]>([])
	const [taskLifecycleChainsDraft, setTaskLifecycleChainsDraft] = useState<Partial<Record<LifecycleHook, string>>>({})
  const [editingTask, setEditingTask] = useState<TaskRecord>()
  const [settingsDialogOpen, setSettingsDialogOpen] = useState(false)
	const [extraInfoManagerOpen, setExtraInfoManagerOpen] = useState(false)
	const [extraInfoTemplateDraft, setExtraInfoTemplateDraft] = useState<ExtraInfoTemplate>()
	const [extraInfoDraft, setExtraInfoDraft] = useState<ExtraInfo>()
	const [extraInfoEditorOpen, setExtraInfoEditorOpen] = useState(false)
	const [newExtraInfoTemplateID, setNewExtraInfoTemplateID] = useState('')
	const [lastCreatedExtraInfoTemplateID, setLastCreatedExtraInfoTemplateID] = useState('')
	const [extraInfoSearch, setExtraInfoSearch] = useState('')
	const [templateSectionExpanded, setTemplateSectionExpanded] = useState(false)
	const [expandedExtraInfoTemplateIDs, setExpandedExtraInfoTemplateIDs] = useState<string[]>([])
  const [finishTask, setFinishTask] = useState<TaskRecord>()
  const [quitDialogOpen, setQuitDialogOpen] = useState(false)
  const [draftTitle, setDraftTitle] = useState('')
  const [draftDescription, setDraftDescription] = useState('')
  const [draftColor, setDraftColor] = useState(defaultTaskColor)
  const [settingsDraft, setSettingsDraft] = useState<SettingsRecord>()
  const [settingsTab, setSettingsTab] = useState<'workspace' | 'shell' | 'menu' | 'status' | 'lifecycle'>('workspace')
  const [statusHelpOpen, setStatusHelpOpen] = useState(false)
  const [taskMenuItemDraft, setTaskMenuItemDraft] = useState<TaskMenuItem>()
  const [taskMenuItemEditorMode, setTaskMenuItemEditorMode] = useState<'create' | 'edit'>()
  const [taskMenuItemEditorTab, setTaskMenuItemEditorTab] = useState<'basic' | 'scripts'>('basic')
  const [scriptHelpAnchor, setScriptHelpAnchor] = useState<HTMLElement>()
  const [message, setMessage] = useState<string>()
  const dragging = useRef(false)
  const currentTreeWidth = useRef(treeWidth)
  const terminalTitleParserStates = useRef(new Map<string, TerminalTitleParserState>())
  const pendingTerminalEvents = useRef(new Map<string, PendingTerminalEvent>())
  const registeredTerminalKeys = useRef(new Set<string>())
  const finishedTerminalTaskIDs = useRef(new Set<string>())
  const terminalTitleValues = useRef(new Map<string, string>())
  const latestRealtimeStatusVersion = useRef(0)
	const initializedExtraInfoGroups = useRef(false)

  useEffect(() => {
    void (async () => {
      try {
		const [loadedTasks, loadedSettings, loadedShells, loadedExtraInfoTemplates, loadedExtraInfos] = await Promise.all([api.listTasks(), api.getSettings(), api.detectShells(), api.listExtraInfoTemplates(), api.listExtraInfos()])
        setTasks(loadedTasks)
        setSettings(loadedSettings)
        setActiveTaskStatus(loadedSettings.activeTaskStatus ?? 'pending')
        setDetectedShells(loadedShells)
		setExtraInfoTemplates(loadedExtraInfoTemplates)
		setExtraInfos(loadedExtraInfos)
        const width = clampTaskTreeWidth(loadedSettings.taskTreeWidth)
        currentTreeWidth.current = width
        setTreeWidth(width)
      } catch (error) {
        showError(error, setMessage)
      } finally {
        setInitialLoadComplete(true)
      }
    })()
    const unsubscribe = api.onTerminalEvent((event) => {
      if (finishedTerminalTaskIDs.current.has(event.taskId)) {
        return
      }
      const title = parseTerminalEventTitle(terminalTitleParserStates.current, event)
      const key = terminalEventKey(event.taskId, event.terminalId)
      if (title !== undefined && shouldReportTerminalTitleActivity({title: terminalTitleValues.current.get(key)}, title)) {
        terminalTitleValues.current.set(key, title)
        void api.reportTerminalTitleActivity(event.taskId, event.terminalId).catch((error) => showError(error, setMessage))
      }
      if (!registeredTerminalKeys.current.has(key)) {
        bufferPendingTerminalEvent(pendingTerminalEvents.current, event, title)
      }
      setTerminals((current) => applyTerminalEvent(current, event, title))
      if (event.type === 'error') {
        setMessage(event.data || '终端发生错误')
      }
    })
    return () => {
      unsubscribe()
      terminalTitleParserStates.current.clear()
      pendingTerminalEvents.current.clear()
      registeredTerminalKeys.current.clear()
      finishedTerminalTaskIDs.current.clear()
      terminalTitleValues.current.clear()
    }
  }, [])

  useEffect(() => api.onCloseRequested(() => setQuitDialogOpen(true)), [])

  useEffect(() => api.onTaskScriptError((message) => setMessage(message)), [])

  useEffect(() => api.onRealtimeStatusError((message) => setMessage(message)), [])

	useEffect(() => api.onLifecycleEvent((updated) => {
		setTasks((current) => replaceTask(current, updated))
	}), [])

	useEffect(() => {
		if (initializedExtraInfoGroups.current || extraInfoTemplates.length === 0) {
			return
		}
		initializedExtraInfoGroups.current = true
		setExpandedExtraInfoTemplateIDs(extraInfoTemplates.filter((template) => template.catalogue === 'git').map((template) => template.id))
	}, [extraInfoTemplates])

	useEffect(() => {
		const search = extraInfoSearch.trim().toLocaleLowerCase()
		if (!search) {
			return
		}
		const matchedTemplateIDs = extraInfos
			.filter((info) => extraInfoName(info).toLocaleLowerCase().includes(search))
			.map((info) => info.templateId)
		setExpandedExtraInfoTemplateIDs((current) => [...new Set([...current, ...matchedTemplateIDs])])
	}, [extraInfoSearch, extraInfos])

  useEffect(() => {
    const unsubscribe = api.onRealtimeStatusEvent((event) => {
      if (event.version <= latestRealtimeStatusVersion.current) {
        return
      }
      latestRealtimeStatusVersion.current = event.version
      if (event.terminalId && !registeredTerminalKeys.current.has(terminalEventKey(event.taskId, event.terminalId))) {
        bufferPendingRealtimeStatusEvent(pendingTerminalEvents.current, event)
      }
      setTasks((current) => applyRealtimeStatusToTasks(current, event))
      setTerminals((current) => applyRealtimeStatusToTerminals(current, event))
    })
    return () => {
      unsubscribe()
      latestRealtimeStatusVersion.current = 0
    }
  }, [])

  useEffect(() => {
    const synchronizeSelection = selectedTaskID && selectedTerminalID
      ? api.selectTerminal(selectedTaskID, selectedTerminalID)
      : api.clearSelectedTerminal()
    void synchronizeSelection.catch((error) => showError(error, setMessage))
  }, [selectedTaskID, selectedTerminalID])

  const colorScheme: ColorScheme = settings?.colorScheme === 'dark' ? 'dark' : 'light'
  const theme = useMemo(() => createAppTheme(colorScheme), [colorScheme])
  const selectedTask = tasks.find((task) => task.id === selectedTaskID)
  const selectedTerminal = terminals.find((terminal) => terminal.id === selectedTerminalID && terminal.state === 'active')
  const taskMenuItems = settings?.taskMenuItems?.length ? settings.taskMenuItems : defaultTaskMenuItems
  const areAllTasksExpanded = tasks.length > 0 && tasks.every((task) => expandedTasks[task.id] ?? true)

  const toggleTaskExpanded = (taskID: string) => {
    setExpandedTasks((current) => ({...current, [taskID]: !(current[taskID] ?? true)}))
  }

  const toggleAllTasksExpanded = () => {
    const nextExpanded = !areAllTasksExpanded
    setExpandedTasks(Object.fromEntries(tasks.map((task) => [task.id, nextExpanded])))
  }

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
		setTaskExtraInfoDraft(task?.extraInfo ? cloneTaskExtraInfo(task.extraInfo) : [])
		setTaskLifecycleChainsDraft({...task?.lifecycleChains ?? settings?.lifecycleDefaultChains ?? {}})
    setTaskDialogOpen(true)
  }

  const closeTaskDialog = () => {
    setTaskDialogOpen(false)
    setEditingTask(undefined)
		setTaskExtraInfoDraft([])
		setTaskLifecycleChainsDraft({})
  }

  const saveTask = async (event: FormEvent) => {
    event.preventDefault()
    if (!draftTitle.trim()) {
      setMessage('任务标题不能为空')
      return
    }
		const missingParameter = taskExtraInfoDraft.flatMap((item) => item.parameters).find((parameter) => parameter.required && !parameter.value.trim())
		if (missingParameter) {
			setMessage(`参数“${missingParameter.displayName}”不能为空`)
			return
		}
		const invalidParameter = taskExtraInfoDraft.flatMap((item) => item.parameters).find((parameter) => !parameter.key.trim() || !parameter.displayName.trim())
		if (invalidParameter) {
			setMessage('动态参数的键和显示名称不能为空')
			return
		}
		const duplicateParameter = taskExtraInfoDraft.find((item) => new Set(item.parameters.map((parameter) => parameter.key.trim())).size !== item.parameters.length)
		if (duplicateParameter) {
			setMessage(`信息“${duplicateParameter.displayName ?? duplicateParameter.catalogue}”包含重复的动态参数键`)
			return
		}
    try {
      if (editingTask) {
			const hasExtraInfo = taskExtraInfoDraft.length > 0 || (editingTask.extraInfo?.length ?? 0) > 0
			const updated = editingTask.status === 'pending'
					? await api.updateTaskWithExtraInfoAndLifecycleChains(editingTask.id, draftTitle, draftDescription, draftColor, taskExtraInfoDraft, taskLifecycleChainsDraft)
					: hasExtraInfo
						? await api.updateTaskWithExtraInfo(editingTask.id, draftTitle, draftDescription, draftColor, taskExtraInfoDraft)
						: await api.updateTask(editingTask.id, draftTitle, draftDescription, draftColor)
        setTasks((current) => replaceTask(current, updated))
      } else {
			const hasLifecycleChainSelection = Object.keys(taskLifecycleChainsDraft).length > 0
			const created = hasLifecycleChainSelection
				? await api.createTaskWithExtraInfoAndLifecycleChains(draftTitle, draftDescription, draftColor, taskExtraInfoDraft, taskLifecycleChainsDraft)
				: taskExtraInfoDraft.length > 0
					? await api.createTaskWithExtraInfo(draftTitle, draftDescription, draftColor, taskExtraInfoDraft)
					: await api.createTask(draftTitle, draftDescription, draftColor)
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

	const openExtraInfoManager = async () => {
		setExtraInfoManagerOpen(true)
		try {
			const [templates, infos] = await Promise.all([api.listExtraInfoTemplates(), api.listExtraInfos()])
			setExtraInfoTemplates(templates)
			setExtraInfos(infos)
		} catch (error) {
			showError(error, setMessage)
		}
	}

	const openExtraInfoTemplateEditor = (template?: ExtraInfoTemplate) => {
		setExtraInfoTemplateDraft(template ? cloneExtraInfoTemplate(template) : createExtraInfoTemplateDraft(''))
	}

	const updateExtraInfoTemplateDraft = (update: Partial<ExtraInfoTemplate>) => {
		setExtraInfoTemplateDraft((current) => current ? {...current, ...update} : current)
	}

	const updateExtraInfoTemplateParameter = (index: number, update: Partial<ExtraInfoTemplate['parameters'][number]>) => {
		setExtraInfoTemplateDraft((current) => current ? {
			...current,
			parameters: current.parameters.map((parameter, parameterIndex) => parameterIndex === index ? {...parameter, ...update} : parameter),
		} : current)
	}

	const updateExtraInfoTemplateField = (index: number, update: Partial<ExtraInfoField>) => {
		setExtraInfoTemplateDraft((current) => current ? {
			...current,
			fields: current.fields.map((field, fieldIndex) => fieldIndex === index ? {...field, ...update} : field),
		} : current)
	}

	const saveExtraInfoTemplate = async () => {
		if (!extraInfoTemplateDraft) {
			return
		}
		try {
			const saved = await api.saveExtraInfoTemplate(extraInfoTemplateDraft)
			setExtraInfoTemplates((current) => {
				const exists = current.some((template) => template.id === saved.id)
				return exists ? current.map((template) => template.id === saved.id ? saved : template) : [...current, saved]
			})
			setExtraInfoTemplateDraft(undefined)
		} catch (error) {
			showError(error, setMessage)
		}
	}

	const deleteExtraInfoTemplate = async (templateID: string) => {
		try {
			await api.deleteExtraInfoTemplate(templateID)
			setExtraInfoTemplates((current) => current.filter((template) => template.id !== templateID))
		} catch (error) {
			showError(error, setMessage)
		}
	}

	const closeExtraInfoEditor = () => {
		setExtraInfoEditorOpen(false)
		setExtraInfoDraft(undefined)
		setNewExtraInfoTemplateID('')
	}

	const openExtraInfoEditor = (info?: ExtraInfo) => {
		setExtraInfoEditorOpen(true)
		if (info) {
			setExtraInfoDraft(cloneExtraInfo(info))
			return
		}
		const template = extraInfoTemplates.length === 1
			? extraInfoTemplates[0]
			: extraInfoTemplates.find((item) => item.id === lastCreatedExtraInfoTemplateID)
		setExtraInfoDraft(template ? createExtraInfoDraft(template) : undefined)
		setNewExtraInfoTemplateID(template?.id ?? '')
	}

	const selectExtraInfoTemplate = (templateID: string) => {
		setNewExtraInfoTemplateID(templateID)
		const template = extraInfoTemplates.find((item) => item.id === templateID)
		if (!template) {
			return
		}
		setExtraInfoDraft(createExtraInfoDraft(template))
	}

	const saveExtraInfo = async () => {
		if (!extraInfoDraft) {
			return
		}
		const creating = !extraInfoDraft.id
		if (!extraInfoName(extraInfoDraft).trim()) {
			setMessage('信息名称不能为空')
			return
		}
		try {
			const saved = await api.saveExtraInfo(extraInfoDraft)
			setExtraInfos((current) => current.some((item) => item.id === saved.id)
				? current.map((item) => item.id === saved.id ? saved : item)
				: [...current, saved])
			if (creating) {
				setLastCreatedExtraInfoTemplateID(saved.templateId)
			}
			closeExtraInfoEditor()
			setExpandedExtraInfoTemplateIDs((current) => [...new Set([...current, saved.templateId])])
		} catch (error) {
			showError(error, setMessage)
		}
	}

	const deleteExtraInfo = async (infoID: string) => {
		try {
			await api.deleteExtraInfo(infoID)
			setExtraInfos((current) => current.filter((item) => item.id !== infoID))
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

	const retryLifecycleChain = async (taskID: string) => {
		try {
			const updated = await api.retryTaskLifecycleCommandChain(taskID)
			setTasks((current) => replaceTask(current, updated))
		} catch (error) {
			showError(error, setMessage)
		}
	}

	const setTaskShelved = async (taskID: string, shelved: boolean) => {
		try {
			setTasks(await api.setTaskShelved(taskID, shelved))
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
      finishedTerminalTaskIDs.current.add(finishTask.id)
      clearTaskTerminalTracking(finishTask.id, terminalTitleParserStates.current, pendingTerminalEvents.current, registeredTerminalKeys.current)
      setTerminals((current) => current.filter((terminal) => terminal.taskId !== finishTask.id))
      if (selectedTaskID === finishTask.id) {
        setSelectedTerminalID(undefined)
      }
      setFinishTask(undefined)
    } catch (error) {
      showError(error, setMessage)
    }
  }

  const addTerminal = (terminal: TerminalRecord): boolean => {
    if (finishedTerminalTaskIDs.current.has(terminal.taskId)) {
      return false
    }
    const merged = mergePendingTerminalEvents(pendingTerminalEvents.current, terminal)
    registerTerminal(registeredTerminalKeys.current, merged)
    terminalTitleValues.current.set(terminalEventKey(merged.taskId, merged.id), merged.title ?? '')
    setTerminals((current) => [...current, merged])
    return true
  }

  const createTerminal = async (taskID: string) => {
    try {
      const created = await api.createTerminal(taskID, 100, 32)
      if (addTerminal(created)) {
        setSelectedTaskID(taskID)
        setSelectedTerminalID(created.id)
      }
    } catch (error) {
      showError(error, setMessage)
    }
  }

  const runTaskMenuCommand = async (taskID: string, itemID: string) => {
    try {
      const result = await api.executeTaskMenuCommand(taskID, itemID, 100, 32)
      if (result.terminal && addTerminal(result.terminal)) {
        setSelectedTaskID(taskID)
        setSelectedTerminalID(result.terminal.id)
      }
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
    setTerminals((current) => current.filter((currentTerminal) => (
      currentTerminal.id !== terminal.id || currentTerminal.taskId !== terminal.taskId
    )))
    if (selectedTerminalID === terminal.id) {
      setSelectedTerminalID(undefined)
    }
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
    setSettingsTab('workspace')
		setStatusHelpOpen(false)
    closeTaskMenuItemEditor()
  }

	const refreshLifecycleConfiguration = async () => {
		try {
			const refreshed = await api.getSettings()
			setSettings(refreshed)
			setSettingsDraft((current) => current ? {
				...current,
				lifecycleCommands: refreshed.lifecycleCommands ?? [],
				lifecycleChains: refreshed.lifecycleChains ?? [],
				lifecycleDefaultChains: refreshed.lifecycleDefaultChains ?? {},
			} : current)
		} catch (error) {
			showError(error, setMessage)
		}
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
    setTaskMenuItemEditorTab('basic')
    setScriptHelpAnchor(undefined)
  }

  const closeTaskMenuItemEditor = () => {
    setTaskMenuItemDraft(undefined)
    setTaskMenuItemEditorMode(undefined)
    setTaskMenuItemEditorTab('basic')
    setScriptHelpAnchor(undefined)
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
      beforeScript: normalizeTaskScript(taskMenuItemDraft.beforeScript),
      afterScript: normalizeTaskScript(taskMenuItemDraft.afterScript),
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
		statusManagementMode: current?.statusManagementMode ?? settings?.statusManagementMode ?? 'title-change',
		statusManagementHTTPPort: current?.statusManagementHTTPPort ?? settings?.statusManagementHTTPPort ?? 0,
		httpServiceEnabled: current?.httpServiceEnabled ?? settings?.httpServiceEnabled ?? false,
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

  const statusManagementMode = settingsDraft?.statusManagementMode ?? 'title-change'
  const httpServiceEnabled = settingsDraft?.httpServiceEnabled ?? false
  const httpServiceActive = statusManagementMode === 'http' || httpServiceEnabled
	const extraInfoDraftTemplate = extraInfoDraft ? extraInfoTemplates.find((template) => template.id === extraInfoDraft.templateId) : undefined

  return (
    <ThemeProvider theme={theme}>
      <CssBaseline/>
      <Box sx={{height: '100vh', minWidth: 720, display: 'grid', gridTemplateRows: '52px minmax(0, 1fr)', overflow: 'hidden'}}>
        <AppBar position="static" color="transparent" elevation={0} sx={{borderBottom: 1, borderColor: 'divider', bgcolor: 'background.paper'}}>
          <Toolbar variant="dense" sx={{minHeight: '52px !important', gap: 1}}>
            <TaskAltOutlinedIcon color="primary"/>
            <Typography variant="subtitle1" sx={{fontWeight: 800, letterSpacing: 0.3}}>任务工作台</Typography>
            <Box sx={{flex: 1}}/>
			<Tooltip title="额外信息管理">
				<IconButton aria-label="额外信息管理" onClick={() => void openExtraInfoManager()}>
					<FolderOutlinedIcon/>
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
						statusManagementMode: settings.statusManagementMode ?? 'title-change',
						statusManagementHTTPPort: settings.statusManagementHTTPPort ?? 0,
						httpServiceEnabled: settings.httpServiceEnabled ?? false,
                  } : undefined)
                  setTaskMenuItemDraft(undefined)
                  setTaskMenuItemEditorMode(undefined)
                  setSettingsTab('workspace')
					setStatusHelpOpen(false)
                  setTaskMenuItemEditorTab('basic')
                  setScriptHelpAnchor(undefined)
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
              <Tooltip title={areAllTasksExpanded ? '收起全部任务' : '展开全部任务'}>
                <span>
                  <IconButton
                    aria-label={areAllTasksExpanded ? '收起全部任务' : '展开全部任务'}
                    disabled={tasks.length === 0}
                    onClick={toggleAllTasksExpanded}
                    size="small"
                  >
                    {areAllTasksExpanded ? <UnfoldLessOutlinedIcon fontSize="small"/> : <UnfoldMoreOutlinedIcon fontSize="small"/>}
                  </IconButton>
                </span>
              </Tooltip>
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
                expandedTasks={expandedTasks}
                selectedTerminalId={selectedTerminalID}
                onChangeStatus={(status) => void changeActiveTaskStatus(status)}
                onToggleTaskExpanded={toggleTaskExpanded}
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
                onRunMenuCommand={(taskID, itemID) => void runTaskMenuCommand(taskID, itemID)}
                onStartTask={(taskID) => void startTask(taskID)}
				onFinishTask={(taskID) => setFinishTask(tasks.find((task) => task.id === taskID))}
				onRetryLifecycle={(taskID) => void retryLifecycleChain(taskID)}
				onSetTaskShelved={(taskID, shelved) => void setTaskShelved(taskID, shelved)}
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
			<TaskExtraInfoEditor
				templates={extraInfoTemplates}
				infos={extraInfos}
				extraInfo={taskExtraInfoDraft}
				onChange={setTaskExtraInfoDraft}
			/>
			<TaskLifecycleChainSelector
				chains={settings?.lifecycleChains ?? []}
				selected={taskLifecycleChainsDraft}
				onChange={setTaskLifecycleChainsDraft}
				disabled={editingTask?.status !== undefined && editingTask.status !== 'pending'}
			/>
          </DialogContent>
          <DialogActions>
            <Button onClick={closeTaskDialog}>取消</Button>
            <Button type="submit" variant="contained">{editingTask ? '保存' : '创建'}</Button>
          </DialogActions>
        </Box>
      </Dialog>

		<Dialog open={extraInfoManagerOpen} onClose={() => setExtraInfoManagerOpen(false)} aria-labelledby="extra-info-manager-title" fullWidth maxWidth="md">
			<DialogTitle id="extra-info-manager-title">额外信息管理</DialogTitle>
			<DialogContent data-testid="extra-info-manager-content" sx={{display: 'flex', flexDirection: 'column', gap: 2, minHeight: 0, overflowY: 'auto', pt: '12px !important'}}>
				<Box data-testid="extra-info-manager-actions" sx={{display: 'flex', justifyContent: 'flex-end', flexWrap: 'wrap', gap: 1}}>
					<Button variant="outlined" size="small" onClick={() => openExtraInfoTemplateEditor()}>新增模板</Button>
					<Button variant="contained" size="small" disabled={extraInfoTemplates.length === 0} onClick={() => openExtraInfoEditor()}>新增信息</Button>
				</Box>
				<Box sx={{minWidth: 0}}>
					<Accordion disableGutters elevation={0} expanded={templateSectionExpanded} onChange={(_, expanded) => setTemplateSectionExpanded(expanded)} sx={{minWidth: 0, border: 1, borderColor: 'divider', borderRadius: '8px !important', overflow: 'hidden', '&:before': {display: 'none'}}}>
						<AccordionSummary expandIcon={<ExpandMoreOutlinedIcon/>} aria-label="分类模板">
							<Box sx={{minWidth: 0, pr: 1}}>
								<Typography variant="subtitle2">分类模板</Typography>
								<Typography variant="caption" color="text.secondary">分类就是可填写的信息模板，定义固定字段、默认值和动态参数。</Typography>
							</Box>
						</AccordionSummary>
						<AccordionDetails sx={{display: 'grid', gap: 1, pt: 0}}>
							<Box sx={{border: 1, borderColor: 'divider', borderRadius: 1, overflow: 'hidden'}}>
								{extraInfoTemplates.map((template, index) => (
									<Box key={template.id} sx={{display: 'grid', gridTemplateColumns: {xs: 'minmax(0, 1fr)', sm: 'minmax(0, 1fr) auto'}, alignItems: 'center', gap: 1.25, px: 1.5, py: 1.25, borderBottom: index === extraInfoTemplates.length - 1 ? 0 : 1, borderColor: 'divider'}}>
										<Box sx={{display: 'grid', gap: 0.5, minWidth: 0}}>
											<Box sx={{display: 'flex', alignItems: 'center', gap: 0.75, flexWrap: 'wrap'}}>
												<Typography variant="body2" sx={{fontWeight: 700, overflowWrap: 'anywhere'}}>{template.catalogue}</Typography>
												{template.builtIn && <Chip label="内置 Git" size="small" color="primary" variant="outlined"/>}
											</Box>
											<Typography variant="caption" color="text.secondary" sx={{overflowWrap: 'anywhere'}}>{template.fields.map((field) => `${field.displayName}${field.defaultValue ? `：${field.defaultValue}` : ''}`).join(' · ')}</Typography>
										</Box>
										<Box sx={{display: 'flex', alignItems: 'center', justifyContent: {xs: 'space-between', sm: 'flex-end'}, flexWrap: 'wrap', gap: 0.75, minWidth: 0}}>
											<Chip label={`${template.fields.length} 固定 · ${template.parameters.length} 参数`} size="small" variant="outlined"/>
											<Button size="small" onClick={() => openExtraInfoTemplateEditor(template)}>编辑</Button>
											<Tooltip title={template.builtIn ? '内置 Git 模板不可删除' : '删除模板'}><span><IconButton aria-label={`删除模板 ${template.catalogue}`} size="small" color="error" disabled={template.builtIn} onClick={() => void deleteExtraInfoTemplate(template.id)}><DeleteOutlineOutlinedIcon fontSize="inherit"/></IconButton></span></Tooltip>
										</Box>
									</Box>
								))}
							</Box>
						</AccordionDetails>
					</Accordion>
				</Box>
				<Box sx={{mt: 0.5}}>
					<Typography variant="subtitle2">信息</Typography>
					<Typography variant="caption" color="text.secondary">填写固定字段后保存为可复用信息，任务选择它时会生成独立快照。</Typography>
				</Box>
				<TextField size="small" fullWidth label="搜索信息" placeholder="按名称模糊搜索" value={extraInfoSearch} onChange={(event) => setExtraInfoSearch(event.target.value)}/>
				{extraInfos.length === 0 ? <Alert severity="info" variant="outlined">暂无信息。选择一个模板后填写固定字段即可添加。</Alert> : (
					<Box sx={{display: 'grid', gap: 0.75}}>
						{extraInfoTemplates.map((template) => {
							const templateInfos = extraInfos.filter((info) => info.templateId === template.id)
							if (templateInfos.length === 0) {
								return null
							}
							const infos = templateInfos.filter((info) => extraInfoName(info).toLocaleLowerCase().includes(extraInfoSearch.trim().toLocaleLowerCase()))
							const expanded = expandedExtraInfoTemplateIDs.includes(template.id)
							return <Accordion key={template.id} disableGutters elevation={0} expanded={expanded} onChange={(_, nextExpanded) => setExpandedExtraInfoTemplateIDs((current) => nextExpanded ? [...new Set([...current, template.id])] : current.filter((id) => id !== template.id))} sx={{border: 1, borderColor: 'divider', borderRadius: '8px !important', '&:before': {display: 'none'}}}>
								<AccordionSummary expandIcon={<ExpandMoreOutlinedIcon/>} aria-label={`信息分类 ${template.displayName || template.catalogue} ${template.catalogue}`}>
									<Box sx={{display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 1, width: '100%', minWidth: 0, pr: 1}}>
										<Box sx={{minWidth: 0}}><Typography variant="body2" sx={{fontWeight: 700}} noWrap>{template.displayName || template.catalogue}</Typography><Typography variant="caption" color="text.secondary">{template.catalogue}</Typography></Box>
										<Chip label={`${infos.length}${extraInfoSearch.trim() ? ` / ${templateInfos.length}` : ''} 条信息`} size="small" variant="outlined"/>
									</Box>
								</AccordionSummary>
								<AccordionDetails sx={{display: 'grid', gap: 0, pt: 0}}>
									{infos.length === 0 ? <Typography variant="body2" color="text.secondary" sx={{px: 0.5, pb: 1}}>未找到匹配的信息。</Typography> : infos.map((info, index) => <Box key={info.id} sx={{display: 'flex', alignItems: 'center', gap: 1.25, px: 0.5, py: 1.1, borderTop: index === 0 ? 1 : 0, borderColor: 'divider'}}>
										<Box sx={{minWidth: 0, flex: 1}}><Typography variant="body2" sx={{fontWeight: 700}} noWrap>{extraInfoName(info)}</Typography></Box>
										<Button size="small" onClick={() => openExtraInfoEditor(info)}>编辑</Button>
										<IconButton aria-label={`删除信息 ${extraInfoName(info)}`} size="small" color="error" onClick={() => void deleteExtraInfo(info.id)}><DeleteOutlineOutlinedIcon fontSize="inherit"/></IconButton>
									</Box>)}
								</AccordionDetails>
							</Accordion>
						})}
					</Box>
				)}
			</DialogContent>
			<DialogActions><Button onClick={() => setExtraInfoManagerOpen(false)}>关闭</Button></DialogActions>
		</Dialog>

		<Dialog open={extraInfoEditorOpen} onClose={closeExtraInfoEditor} aria-labelledby="extra-info-value-editor-title" fullWidth maxWidth="md">
			<DialogTitle id="extra-info-value-editor-title">{extraInfoDraft?.id ? '编辑信息' : '新增信息'}</DialogTitle>
			<DialogContent sx={{display: 'grid', gap: 1.5, pt: '12px !important'}}>
				{!extraInfoDraft || !extraInfoDraft.id ? <TextField select required autoFocus size="small" label="选择模板" value={newExtraInfoTemplateID} onChange={(event) => selectExtraInfoTemplate(event.target.value)}>
					{extraInfoTemplates.map((template) => <MenuItem key={template.id} value={template.id}>{`${template.displayName || template.catalogue}（${template.catalogue}）`}</MenuItem>)}
				</TextField> : <Typography variant="caption" color="text.secondary">{extraInfoDraft.catalogue}</Typography>}
				{extraInfoDraft && <Box data-testid="extra-info-draft-fields" sx={{display: 'grid', gridTemplateColumns: '1fr', gap: 1.5, minWidth: 0}}>
					{extraInfoDraft.fields.map((field, index) => <TextField key={field.key} required={field.key === 'name'} size="small" label={field.displayName} value={field.value ?? ''} onChange={(event) => setExtraInfoDraft((current) => current ? {...current, fields: current.fields.map((item, fieldIndex) => fieldIndex === index ? {...item, value: event.target.value} : item)} : current)}/>) }
				</Box>}
				{extraInfoDraft && <Box sx={{display: 'grid', gap: 1.25, pt: 0.5}}>
					<Typography variant="subtitle2">动态参数</Typography>
					{extraInfoDraftTemplate && extraInfoDraftTemplate.parameters.length > 0 && <Box data-testid="extra-info-template-parameters" sx={{display: 'grid', gap: 1, borderTop: 1, borderColor: 'divider', pt: 1.25}}>
						<Typography variant="body2" sx={{fontWeight: 700}}>模板参数</Typography>
						{extraInfoDraftTemplate.parameters.map((parameter, index) => {
							const inputType = extraInfoParameterInputType(parameter)
							return <Box key={parameter.key || index} data-testid={`extra-info-template-parameter-preview-${index}`} sx={{display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(150px, 1fr))', gap: 1, minWidth: 0, borderTop: 1, borderColor: 'divider', py: 1.25}}>
								<Typography variant="body2" sx={{overflowWrap: 'anywhere'}}>参数键：{parameter.key || '未设置'}</Typography>
								<Typography variant="body2" sx={{overflowWrap: 'anywhere'}}>显示名称：{parameter.displayName || '未设置'}</Typography>
								<Typography variant="body2">默认值：{inputType === 'checkbox' ? 'false' : '空'}</Typography>
								{inputType !== 'checkbox' && <Chip label={parameter.required ? '必填' : '非必填'} size="small" variant="outlined" sx={{justifySelf: 'start'}}/>}
							</Box>
						})}
					</Box>}
					<Box sx={{display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 1}}>
						<Typography variant="body2" sx={{fontWeight: 700}}>信息参数</Typography>
						<Button size="small" onClick={() => setExtraInfoDraft((current) => current ? {...current, parameters: [...(current.parameters ?? []), {key: '', displayName: '', required: false, inputType: 'text', value: ''}]} : current)}>新增动态参数</Button>
					</Box>
					<Typography variant="caption" color="text.secondary">这些参数会和模板参数一起带入任务，可在任务中填写值。</Typography>
					{(extraInfoDraft.parameters ?? []).map((parameter, index) => <Box key={index} data-testid={`extra-info-draft-parameter-${index}`} sx={{display: 'grid', gap: 1.25, minWidth: 0, borderTop: 1, borderColor: 'divider', py: 1.5}}>
						<Box sx={{display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(150px, 1fr))', gap: 1.25, minWidth: 0}}>
							<TextField required size="small" label={`参数键 ${index + 1}`} value={parameter.key} onChange={(event) => setExtraInfoDraft((current) => current ? {...current, parameters: (current.parameters ?? []).map((item, parameterIndex) => parameterIndex === index ? {...item, key: event.target.value} : item)} : current)}/>
							<TextField required size="small" label={`参数显示名称 ${index + 1}`} value={parameter.displayName} onChange={(event) => setExtraInfoDraft((current) => current ? {...current, parameters: (current.parameters ?? []).map((item, parameterIndex) => parameterIndex === index ? {...item, displayName: event.target.value} : item)} : current)}/>
							<TextField select size="small" label={`参数类型 ${index + 1}`} value={extraInfoParameterInputType(parameter)} onChange={(event) => setExtraInfoDraft((current) => current ? {...current, parameters: (current.parameters ?? []).map((item, parameterIndex) => {
								if (parameterIndex !== index) {
									return item
								}
								const inputType = event.target.value as ExtraInfoParameterInputType
								return {...item, inputType, required: inputType === 'checkbox' ? false : item.required, value: inputType === 'checkbox' ? 'false' : item.value}
							})} : current)}>
								<MenuItem value="text">文本</MenuItem>
								<MenuItem value="checkbox">复选框</MenuItem>
							</TextField>
						</Box>
						<Box sx={{display: 'flex', alignItems: 'center', justifyContent: 'space-between', flexWrap: 'wrap', gap: 1.25, minWidth: 0}}>
							{extraInfoParameterInputType(parameter) === 'checkbox'
								? <FormControlLabel sx={{m: 0}} control={<Checkbox checked={parameter.value === 'true'} onChange={(event) => setExtraInfoDraft((current) => current ? {...current, parameters: (current.parameters ?? []).map((item, parameterIndex) => parameterIndex === index ? {...item, value: event.target.checked ? 'true' : 'false'} : item)} : current)}/>} label={`默认值 ${index + 1}`}/>
								: <TextField size="small" sx={{flex: '1 1 180px'}} label={`默认值 ${index + 1}`} value={parameter.value} onChange={(event) => setExtraInfoDraft((current) => current ? {...current, parameters: (current.parameters ?? []).map((item, parameterIndex) => parameterIndex === index ? {...item, value: event.target.value} : item)} : current)}/>
							}
							{extraInfoParameterInputType(parameter) !== 'checkbox' && <FormControlLabel sx={{m: 0}} control={<Checkbox checked={parameter.required} onChange={(event) => setExtraInfoDraft((current) => current ? {...current, parameters: (current.parameters ?? []).map((item, parameterIndex) => parameterIndex === index ? {...item, required: event.target.checked} : item)} : current)}/>} label={`参数 ${index + 1} 必填`}/>}
							<IconButton aria-label={`删除动态参数 ${parameter.displayName || index + 1}`} size="small" color="error" onClick={() => setExtraInfoDraft((current) => current ? {...current, parameters: (current.parameters ?? []).filter((_, parameterIndex) => parameterIndex !== index)} : current)}><DeleteOutlineOutlinedIcon fontSize="inherit"/></IconButton>
						</Box>
					</Box>)}
				</Box>}
			</DialogContent>
			<DialogActions><Button onClick={closeExtraInfoEditor}>取消</Button><Button variant="contained" disabled={!extraInfoDraft} onClick={() => void saveExtraInfo()}>保存信息</Button></DialogActions>
		</Dialog>

		<Dialog open={Boolean(extraInfoTemplateDraft)} onClose={() => setExtraInfoTemplateDraft(undefined)} aria-labelledby="extra-info-editor-title" fullWidth maxWidth="md">
			<DialogTitle id="extra-info-editor-title">{extraInfoTemplateDraft?.id ? '编辑模板' : '新增模板'}</DialogTitle>
			<DialogContent sx={{display: 'grid', gap: 1.5, pt: '12px !important'}}>
				{extraInfoTemplateDraft?.builtIn && <Alert severity="info" variant="outlined">Git 内置字段的键和显示名称不可修改；可调整默认值、分支必填状态，并添加新的字段或参数。</Alert>}
				<Box data-testid="extra-info-template-basic-fields" sx={{display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(220px, 1fr))', gap: 1.5, minWidth: 0}}>
					<TextField required size="small" label="分类" value={extraInfoTemplateDraft?.catalogue ?? ''} disabled={extraInfoTemplateDraft?.builtIn} onChange={(event) => updateExtraInfoTemplateDraft({catalogue: event.target.value})}/>
					<TextField size="small" label="模板备注" value={extraInfoTemplateDraft?.displayName ?? ''} disabled={extraInfoTemplateDraft?.builtIn} onChange={(event) => updateExtraInfoTemplateDraft({displayName: event.target.value})}/>
				</Box>
				<Box sx={{display: 'grid', gap: 1.25}}>
					<Box sx={{display: 'flex', justifyContent: 'space-between', alignItems: 'center'}}>
						<Typography variant="subtitle2">固定字段</Typography>
						<Button size="small" onClick={() => updateExtraInfoTemplateDraft({fields: [...(extraInfoTemplateDraft?.fields ?? []), {key: '', displayName: '', defaultValue: ''}]})}>新增固定字段</Button>
					</Box>
					{extraInfoTemplateDraft?.fields.map((field, index) => {
						const protectedField = Boolean(extraInfoTemplateDraft.builtIn && (field.key === 'name' || field.key === 'repository'))
						return <Box key={index} data-testid={`extra-info-template-fixed-field-${index}`} sx={{display: 'grid', gridTemplateColumns: 'minmax(0, 1fr) auto', alignItems: 'start', gap: 1.25, minWidth: 0, borderTop: 1, borderColor: 'divider', py: 1.5}}>
							<Box sx={{display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(150px, 1fr))', gap: 1.25, minWidth: 0}}>
								<TextField required size="small" label={`固定键 ${index + 1}`} disabled={protectedField} value={field.key} onChange={(event) => updateExtraInfoTemplateField(index, {key: event.target.value})}/>
								<TextField required size="small" label={`固定字段显示名称 ${index + 1}`} disabled={protectedField} value={field.displayName} onChange={(event) => updateExtraInfoTemplateField(index, {displayName: event.target.value})}/>
								<TextField size="small" label={`默认值 ${index + 1}`} value={field.defaultValue ?? ''} onChange={(event) => updateExtraInfoTemplateField(index, {defaultValue: event.target.value})}/>
							</Box>
							<IconButton aria-label={`删除固定字段 ${index + 1}`} size="small" color="error" disabled={protectedField || extraInfoTemplateDraft.fields.length === 1} onClick={() => updateExtraInfoTemplateDraft({fields: extraInfoTemplateDraft.fields.filter((_, fieldIndex) => fieldIndex !== index)})}><DeleteOutlineOutlinedIcon fontSize="inherit"/></IconButton>
						</Box>
					})}
				</Box>
				<Box sx={{display: 'grid', gap: 1.25}}>
					<Box sx={{display: 'flex', justifyContent: 'space-between', alignItems: 'center'}}>
						<Typography variant="subtitle2">动态参数</Typography>
						<Button size="small" onClick={() => updateExtraInfoTemplateDraft({parameters: [...(extraInfoTemplateDraft?.parameters ?? []), {key: '', displayName: '', required: false, inputType: 'text'}]})}>新增参数</Button>
					</Box>
					{extraInfoTemplateDraft?.parameters.map((parameter, index) => {
						const protectedParameter = Boolean(extraInfoTemplateDraft.builtIn && parameter.key === 'branch')
						return <Box key={index} data-testid={`extra-info-template-parameter-${index}`} sx={{display: 'grid', gridTemplateColumns: 'minmax(0, 1fr) auto', alignItems: 'start', gap: 1.25, minWidth: 0, borderTop: 1, borderColor: 'divider', py: 1.5}}>
							<Box sx={{display: 'grid', gap: 1.25, minWidth: 0}}>
								<Box sx={{display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(150px, 1fr))', gap: 1.25, minWidth: 0}}>
									<TextField required size="small" label={`参数键 ${index + 1}`} disabled={protectedParameter} value={parameter.key} onChange={(event) => updateExtraInfoTemplateParameter(index, {key: event.target.value})}/>
									<TextField required size="small" label={`参数显示名称 ${index + 1}`} disabled={protectedParameter} value={parameter.displayName} onChange={(event) => updateExtraInfoTemplateParameter(index, {displayName: event.target.value})}/>
									<TextField select size="small" label={`参数类型 ${index + 1}`} disabled={protectedParameter} value={extraInfoParameterInputType(parameter)} onChange={(event) => {
										const inputType = event.target.value as ExtraInfoParameterInputType
										updateExtraInfoTemplateParameter(index, {inputType, required: inputType === 'checkbox' ? false : parameter.required})
									}}>
										<MenuItem value="text">文本</MenuItem>
										<MenuItem value="checkbox">复选框</MenuItem>
									</TextField>
								</Box>
								{extraInfoParameterInputType(parameter) !== 'checkbox' && <FormControlLabel sx={{m: 0}} control={<Checkbox checked={parameter.required} onChange={(event) => updateExtraInfoTemplateParameter(index, {required: event.target.checked})}/>} label={`参数 ${index + 1} 必填`}/>}
							</Box>
							<IconButton aria-label={`删除参数 ${index + 1}`} size="small" color="error" disabled={protectedParameter} onClick={() => updateExtraInfoTemplateDraft({parameters: extraInfoTemplateDraft.parameters.filter((_, parameterIndex) => parameterIndex !== index)})}><DeleteOutlineOutlinedIcon fontSize="inherit"/></IconButton>
						</Box>
					})}
				</Box>
			</DialogContent>
			<DialogActions>
				<Button onClick={() => setExtraInfoTemplateDraft(undefined)}>取消</Button>
				<Button variant="contained" onClick={() => void saveExtraInfoTemplate()}>保存信息</Button>
			</DialogActions>
		</Dialog>

      <Dialog open={settingsDialogOpen} onClose={closeSettingsDialog} fullWidth maxWidth="md">
        <DialogTitle>设置</DialogTitle>
        <DialogContent sx={{display: 'grid', gap: 3, pt: '12px !important'}}>
          <Tabs
            value={settingsTab}
			onChange={(_, value: 'workspace' | 'shell' | 'menu' | 'status' | 'lifecycle') => setSettingsTab(value)}
            aria-label="设置分类"
            variant="scrollable"
            scrollButtons="auto"
          >
            <Tab value="workspace" label="工作区与外观"/>
            <Tab value="shell" label="终端 Shell"/>
            <Tab value="menu" label="菜单管理"/>
            <Tab value="status" label="实时状态"/>
			<Tab value="lifecycle" label="生命周期编排"/>
          </Tabs>

          {settingsTab === 'workspace' && <Box component="section" sx={{display: 'grid', gap: 1.5}}>
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
          </Box>}

          {settingsTab === 'shell' && <Box component="section" sx={{display: 'grid', gap: 1.5}}>
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
          </Box>}

          {settingsTab === 'menu' && <Box component="section" sx={{display: 'grid', gap: 1.5}}>
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
          </Box>}

          {settingsTab === 'status' && <Box component="section" sx={{display: 'grid', gap: 2}}>
            <Box sx={{display: 'grid', gap: 0.5}}>
              <Typography variant="subtitle2">状态判定</Typography>
              <Typography variant="body2" color="text.secondary">状态仅保存在本次应用会话中：终端标题变化会在 1.5 秒内显示为工作中，未选中的终端随后显示为未读。</Typography>
            </Box>
            <TextField
              fullWidth
              select
              label="状态管理方式"
              value={statusManagementMode}
              onChange={(event) => updateSettingsDraft({statusManagementMode: event.target.value as SettingsRecord['statusManagementMode']})}
            >
              <MenuItem value="title-change">根据终端标题变化</MenuItem>
              <MenuItem value="http">通过 HTTP 接口</MenuItem>
            </TextField>
            <FormControlLabel
              control={
                <Switch
                  checked={httpServiceActive}
                  disabled={statusManagementMode === 'http'}
                  onChange={(event) => updateSettingsDraft({httpServiceEnabled: event.target.checked})}
                />
              }
              label={statusManagementMode === 'http' ? '通过 HTTP 状态管理自动启用本机 HTTP 服务' : '启用本机 HTTP 服务'}
            />
            {httpServiceActive && <>
              <TextField
                fullWidth
                required
                type="number"
                label="HTTP 端口"
                helperText="仅监听 127.0.0.1；关闭独立服务且状态不使用 HTTP 时会停止服务。"
                slotProps={{htmlInput: {min: 1, max: 65535}}}
                value={settingsDraft?.statusManagementHTTPPort ?? 0}
                onChange={(event) => updateSettingsDraft({statusManagementHTTPPort: Number(event.target.value)})}
              />
              <Box sx={{display: 'flex', justifyContent: 'flex-start'}}>
                <Button variant="outlined" size="small" onClick={() => setStatusHelpOpen(true)}>查看 HTTP 接口使用说明</Button>
              </Box>
            </>}
          </Box>}

			{settingsTab === 'lifecycle' && <LifecycleManagement
				commands={settings?.lifecycleCommands ?? []}
				chains={settings?.lifecycleChains ?? []}
				defaults={settings?.lifecycleDefaultChains ?? {}}
				onSaveCommand={async (command) => {
					await api.saveLifecycleCommand(command)
					await refreshLifecycleConfiguration()
				}}
				onDeleteCommand={async (commandID) => {
					await api.deleteLifecycleCommand(commandID)
					await refreshLifecycleConfiguration()
				}}
				onSaveChain={async (chain) => {
					await api.saveLifecycleCommandChain(chain)
					await refreshLifecycleConfiguration()
				}}
				onCopyChain={async (chainID) => {
					await api.copyLifecycleCommandChain(chainID)
					await refreshLifecycleConfiguration()
				}}
				onDeleteChain={async (chainID) => {
					await api.deleteLifecycleCommandChain(chainID)
					await refreshLifecycleConfiguration()
				}}
				onSaveDefault={async (hook, chainID) => {
					const saved = await api.saveLifecycleDefaultChain(hook, chainID)
					setSettings(saved)
					setSettingsDraft((current) => current ? {...current, lifecycleDefaultChains: saved.lifecycleDefaultChains ?? {}} : current)
				}}
				onError={(error) => showError(error, setMessage)}
			/>}
        </DialogContent>
        <DialogActions>
          <Button onClick={closeSettingsDialog}>取消</Button>
          <Button variant="contained" onClick={() => void saveSettings()}>保存</Button>
        </DialogActions>
      </Dialog>

      <Dialog open={statusHelpOpen} onClose={() => setStatusHelpOpen(false)} aria-labelledby="http-status-help-title" fullWidth maxWidth="md">
        <DialogTitle id="http-status-help-title" sx={{pb: 1}}>
          HTTP 状态接口使用说明
          <Typography component="span" variant="caption" color="text.secondary" aria-hidden="true" sx={{display: 'block', mt: 0.25}}>本机接口参考 · API v1</Typography>
        </DialogTitle>
        <DialogContent dividers sx={{display: 'grid', gap: 2, py: 2}}>
          <HTTPHelpSection title="服务与设置">
            <Alert severity="info" variant="outlined" sx={{alignItems: 'center'}}>
              服务仅监听 <code>127.0.0.1:&lt;端口&gt;</code>，无需鉴权，也不会暴露到局域网。
            </Alert>
            <Box sx={{display: 'grid', gridTemplateColumns: {xs: '1fr', sm: '1fr 1fr'}, gap: 1}}>
              <HTTPHelpStep number="1" title="独立启用服务">在“实时状态”中开启“启用本机 HTTP 服务”并设置端口，可单独查询任务和状态。</HTTPHelpStep>
              <HTTPHelpStep number="2" title="使用 HTTP 管理状态">选择“通过 HTTP 接口”会自动启用服务，并向之后新建的终端注入状态变量。</HTTPHelpStep>
            </Box>
          </HTTPHelpSection>

          <HTTPHelpSection title="终端环境变量">
            <Typography variant="body2" color="text.secondary">新建的普通终端和显示终端的自定义命令始终获得任务与终端 ID；HTTP 状态管理方式下额外获得 API 地址：</Typography>
            <HTTPCodeBlock>{'TASKAI_TASK_ID=<任务 ID>\nTASKAI_TERMINAL_ID=<终端 ID>\n\n# 仅 HTTP 状态管理方式注入\nTASKAI_STATUS_API=http://127.0.0.1:<端口>/api/v1'}</HTTPCodeBlock>
            <Typography variant="body2" color="text.secondary">无终端后台命令以及前置、后置脚本仅注入 <code>TASKAI_TASK_ID</code>。</Typography>
          </HTTPHelpSection>

          <HTTPHelpSection title="查询接口">
			<HTTPEndpoint method="GET" path="/api/v1/status?status=pending|running|completed">查询任务和终端的实时状态；可按任务生命周期筛选，条目包含任务名称与生命周期状态。</HTTPEndpoint>
            <HTTPEndpoint method="GET" path="/api/v1/tasks?status=pending|running|completed">按任务生命周期筛选列表；省略 <code>status</code> 时返回全部任务。</HTTPEndpoint>
            <Typography variant="body2" color="text.secondary">任务列表查询参数：可省略；可选值为 pending、running、completed。</Typography>
            <HTTPEndpoint method="GET" path="/api/v1/tasks/:taskId">查询单个任务详情，包含标题、描述、生命周期、时间、工作目录和附加信息。</HTTPEndpoint>
            <HTTPCodeBlock>{'curl "$TASKAI_STATUS_API/status"\n\ncurl "$TASKAI_STATUS_API/tasks?status=running"\n\ncurl "$TASKAI_STATUS_API/tasks/$TASKAI_TASK_ID"'}</HTTPCodeBlock>
          </HTTPHelpSection>

			<HTTPHelpSection title="任务附加信息">
				<Typography variant="body2" color="text.secondary">先通过顶部“额外信息管理”定义模板并保存信息。任务详情的 <code>extraInfo</code> 按目录聚合，将固定字段和动态参数平铺；信息名称以固定字段 <code>name</code> 返回。</Typography>
				<HTTPCodeBlock>{'{"extraInfo":{"git":[{"name":"API 服务","repository":"git@example.com:team/api.git","branch":"main"}]}}'}</HTTPCodeBlock>
			</HTTPHelpSection>

          <HTTPHelpSection title="状态更新">
            <HTTPEndpoint method="PUT" path="/api/v1/tasks/:taskId/status">直接设置任务状态；下一次终端状态更新会重新按终端状态汇总。</HTTPEndpoint>
            <HTTPEndpoint method="PUT" path="/api/v1/tasks/:taskId/terminals/:terminalId/status">更新指定终端，并自动汇总对应任务的状态。</HTTPEndpoint>
            <Box sx={{display: 'grid', gap: 0.75}}>
              <Typography variant="body2" color="text.secondary">两个更新接口都使用以下 JSON 请求体：</Typography>
              <HTTPCodeBlock>{'{"status":"idle|working|unread|error"}'}</HTTPCodeBlock>
              <Typography variant="body2" color="text.secondary">状态更新请求体的 status：必填；合法值为 idle、working、unread、error。</Typography>
              <HTTPCodeBlock>{'curl -X PUT "$TASKAI_STATUS_API/tasks/$TASKAI_TASK_ID/terminals/$TASKAI_TERMINAL_ID/status" \\\n  -H "Content-Type: application/json" \\\n  --data \'{"status":"working"}\''}</HTTPCodeBlock>
            </Box>
          </HTTPHelpSection>

          <HTTPHelpSection title="状态与错误规则">
            <Box sx={{display: 'grid', gap: 1, p: 1.25, border: 1, borderColor: 'divider', borderRadius: 1.5, bgcolor: 'action.hover'}}>
              <Typography variant="body2"><strong>状态值：</strong><code>idle</code> 空闲、<code>working</code> 工作中、<code>unread</code> 未读、<code>error</code> 异常。</Typography>
              <Typography variant="body2"><strong>汇总优先级：</strong>异常 → 未读 → 工作中 → 空闲。</Typography>
              <Typography variant="body2"><strong>错误响应：</strong><code>{'{"error":"..."}'}</code>；无效请求为 <code>400</code>，不存在的任务或终端为 <code>404</code>，已结束任务或已关闭终端为 <code>409</code>，错误方法为 <code>405</code>。</Typography>
              <Typography variant="body2" color="text.secondary">修改端口或切换状态管理方式不会更新已运行终端的环境变量；请新建终端后再使用新配置。</Typography>
            </Box>
          </HTTPHelpSection>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setStatusHelpOpen(false)}>关闭</Button>
        </DialogActions>
      </Dialog>

      {taskMenuItemDraft && <Dialog open onClose={closeTaskMenuItemEditor} fullWidth maxWidth="sm">
        <Box component="form" onSubmit={saveTaskMenuItem}>
          <DialogTitle>{taskMenuItemEditorMode === 'create' ? '新增菜单项' : '编辑菜单项'}</DialogTitle>
          <DialogContent sx={{display: 'grid', gap: 2, pt: '12px !important'}}>
            <Tabs
              value={taskMenuItemEditorTab}
              onChange={(_, value: 'basic' | 'scripts') => setTaskMenuItemEditorTab(value)}
              aria-label="菜单项配置分类"
            >
              <Tab value="basic" label="基本配置"/>
              <Tab value="scripts" label="前后置脚本"/>
            </Tabs>

            {taskMenuItemEditorTab === 'basic' && <>
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
            </>}

            {taskMenuItemEditorTab === 'scripts' && <>
              <Box sx={{display: 'flex', alignItems: 'center', justifyContent: 'space-between'}}>
                <Typography variant="subtitle2">前置与后置脚本</Typography>
                <Tooltip title="前后置脚本使用说明">
                  <IconButton aria-label="前后置脚本使用说明" size="small" onClick={(event) => setScriptHelpAnchor(event.currentTarget)}>
                    <HelpOutlinedIcon fontSize="small"/>
                  </IconButton>
                </Tooltip>
              </Box>
              <TextField
                label="前置脚本（命令或路径）"
                helperText="填写脚本路径或 Shell PATH 中的可执行脚本。"
                value={taskMenuItemDraft.beforeScript?.script ?? ''}
                onChange={(event) => updateTaskMenuItemScript(setTaskMenuItemDraft, 'beforeScript', {script: event.target.value})}
              />
              <TextField
                label="前置脚本参数（每行一个）"
                helperText="每行代表一个前置脚本参数。"
                minRows={2}
                multiline
                value={(taskMenuItemDraft.beforeScript?.arguments ?? []).join('\n')}
                onChange={(event) => updateTaskMenuItemScript(setTaskMenuItemDraft, 'beforeScript', {arguments: event.target.value.split('\n')})}
              />
              <TextField
                label="后置脚本（命令或路径）"
                helperText="填写脚本路径或 Shell PATH 中的可执行脚本。"
                value={taskMenuItemDraft.afterScript?.script ?? ''}
                onChange={(event) => updateTaskMenuItemScript(setTaskMenuItemDraft, 'afterScript', {script: event.target.value})}
              />
              <TextField
                label="后置脚本参数（每行一个）"
                helperText="每行代表一个后置脚本参数。"
                minRows={2}
                multiline
                value={(taskMenuItemDraft.afterScript?.arguments ?? []).join('\n')}
                onChange={(event) => updateTaskMenuItemScript(setTaskMenuItemDraft, 'afterScript', {arguments: event.target.value.split('\n')})}
              />
            </>}
          </DialogContent>
          <DialogActions>
            {taskMenuItemEditorMode === 'edit' && <Button color="error" onClick={deleteTaskMenuItem}>删除菜单项</Button>}
            <Box sx={{flex: 1}}/>
            <Button onClick={closeTaskMenuItemEditor}>取消</Button>
            <Button type="submit" variant="contained">{taskMenuItemEditorMode === 'create' ? '添加菜单项' : '保存菜单项'}</Button>
          </DialogActions>
        </Box>
      </Dialog>}

      <Popover
        open={Boolean(scriptHelpAnchor)}
        anchorEl={scriptHelpAnchor}
        onClose={() => setScriptHelpAnchor(undefined)}
        anchorOrigin={{vertical: 'bottom', horizontal: 'right'}}
        transformOrigin={{vertical: 'top', horizontal: 'right'}}
      >
        <Box sx={{maxWidth: 460, p: 2, display: 'grid', gap: 1.25}}>
          <Typography variant="subtitle2">前后置脚本参数</Typography>
          <Typography variant="body2">脚本通过 UTF-8 JSON 标准输入接收主命令上下文：</Typography>
          <Box component="pre" sx={{m: 0, p: 1, overflowX: 'auto', borderRadius: 1, bgcolor: 'action.hover', fontSize: 12}}>{'{\n  "taskId": "任务 ID",\n  "directory": "任务工作目录",\n  "command": "主命令",\n  "arguments": ["主命令参数"]\n}'}</Box>
          <Box component="dl" sx={{m: 0, display: 'grid', gridTemplateColumns: 'auto 1fr', columnGap: 1, rowGap: 0.5}}>
            <Typography component="dt" variant="body2"><code>taskId</code></Typography><Typography component="dd" variant="body2" sx={{m: 0}}>任务 ID</Typography>
            <Typography component="dt" variant="body2"><code>directory</code></Typography><Typography component="dd" variant="body2" sx={{m: 0}}>任务工作目录</Typography>
            <Typography component="dt" variant="body2"><code>command</code></Typography><Typography component="dd" variant="body2" sx={{m: 0}}>主命令</Typography>
            <Typography component="dt" variant="body2"><code>arguments</code></Typography><Typography component="dd" variant="body2" sx={{m: 0}}>主命令参数数组</Typography>
          </Box>
          <Typography variant="body2">脚本填写路径或 Shell PATH 中的可执行脚本；参数每行传递为一个独立参数，空白行会忽略。</Typography>
          <Typography variant="body2">不支持占位符替换；JSON 不会追加到命令行，也不会与参数拼接。</Typography>
        </Box>
      </Popover>

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

function HTTPHelpSection({title, children}: {title: string; children: ReactNode}) {
  return (
    <Box component="section" sx={{display: 'grid', gap: 1}}>
      <Typography variant="overline" sx={{fontWeight: 800, letterSpacing: 1, lineHeight: 1.3, color: 'primary.main'}}>{title}</Typography>
      {children}
    </Box>
  )
}

function HTTPHelpStep({number, title, children}: {number: string; title: string; children: ReactNode}) {
  return (
    <Box sx={{display: 'grid', gridTemplateColumns: '24px minmax(0, 1fr)', gap: 1, p: 1.25, border: 1, borderColor: 'divider', borderRadius: 1.5}}>
      <Box sx={{width: 24, height: 24, display: 'grid', placeItems: 'center', borderRadius: '50%', bgcolor: 'primary.main', color: 'primary.contrastText', fontSize: 12, fontWeight: 800}}>{number}</Box>
      <Box sx={{display: 'grid', gap: 0.25}}>
        <Typography variant="body2" sx={{fontWeight: 700}}>{title}</Typography>
        <Typography variant="body2" color="text.secondary">{children}</Typography>
      </Box>
    </Box>
  )
}

function HTTPEndpoint({method, path, children}: {method: 'GET' | 'PUT'; path: string; children: ReactNode}) {
  return (
    <Box sx={{display: 'grid', gridTemplateColumns: {xs: '1fr', sm: 'auto minmax(0, 1fr)'}, gap: 1, alignItems: 'start', p: 1.25, border: 1, borderColor: 'divider', borderRadius: 1.5}}>
      <Chip label={method} size="small" color={method === 'GET' ? 'success' : 'primary'} sx={{fontWeight: 800, width: {xs: 'fit-content', sm: 52}}}/>
      <Box sx={{display: 'grid', gap: 0.5, minWidth: 0}}>
        <Typography component="code" variant="body2" sx={{fontFamily: 'ui-monospace, SFMono-Regular, Menlo, monospace', fontWeight: 700, overflowWrap: 'anywhere'}}>{`${method} ${path}`}</Typography>
        <Typography variant="body2" color="text.secondary">{children}</Typography>
      </Box>
    </Box>
  )
}

function HTTPCodeBlock({children}: {children: string}) {
  return <Box component="pre" sx={{m: 0, p: 1.25, overflowX: 'auto', border: 1, borderColor: 'divider', borderRadius: 1.5, bgcolor: 'action.hover', fontSize: 12, lineHeight: 1.6}}>{children}</Box>
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

function TaskLifecycleChainSelector({
	chains,
	selected,
	onChange,
	disabled = false,
}: {
	chains: LifecycleCommandChain[]
	selected: Partial<Record<LifecycleHook, string>>
	onChange: Dispatch<SetStateAction<Partial<Record<LifecycleHook, string>>>>
	disabled?: boolean
}) {
	return (
		<Box component="section" sx={{display: 'grid', gap: 1.25, borderTop: 1, borderColor: 'divider', pt: 1.75}}>
			<Box>
				<Typography variant="subtitle2">生命周期命令链</Typography>
			<Typography variant="caption" color="text.secondary">每个阶段最多选择一条链；留空表示跳过该阶段。{disabled ? ' 执行中和已完成任务不可修改。' : ''}</Typography>
			</Box>
			<Box sx={{display: 'grid', gridTemplateColumns: {xs: '1fr', sm: '1fr 1fr'}, gap: 1.25}}>
				{lifecycleHooks.map((hook) => {
					const selectedChainID = selected[hook.id] ?? ''
					const applicableChains = chains.filter((chain) => lifecycleChainAppliesTo(chain, hook.id))
					const selectedChain = chains.find((chain) => chain.id === selectedChainID)
					const displayChains = selectedChain && !applicableChains.some((chain) => chain.id === selectedChain.id)
						? [...applicableChains, selectedChain]
						: applicableChains
					return <TextField
						key={hook.id}
						select
						size="small"
						label={hook.label}
						value={selectedChainID}
						disabled={disabled}
						onChange={(event) => onChange((current) => {
							const next = {...current}
							if (event.target.value) {
								next[hook.id] = event.target.value
							} else {
								delete next[hook.id]
							}
							return next
						})}
					>
						<MenuItem value="">不执行命令链</MenuItem>
						{displayChains.map((chain) => <MenuItem key={chain.id} value={chain.id}>{chain.name}{!lifecycleChainAppliesTo(chain, hook.id) ? '（当前范围不适用）' : ''}</MenuItem>)}
					</TextField>
				})}
			</Box>
		</Box>
	)
}

function lifecycleChainAppliesTo(chain: LifecycleCommandChain, hook: LifecycleHook) {
	return chain.applicableHooks.includes(hook)
}

function lifecycleHooksCover(availableHooks: LifecycleHook[], requiredHooks: LifecycleHook[]) {
	return requiredHooks.every((hook) => availableHooks.includes(hook))
}

function lifecycleHooksLabel(hooks: LifecycleHook[]) {
	const labels = hooks.map((hook) => lifecycleHooks.find((item) => item.id === hook)?.label).filter(Boolean)
	return labels.join('、') || '未设置'
}

function LifecycleManagement({
	commands,
	chains,
	defaults,
	onSaveCommand,
	onDeleteCommand,
	onSaveChain,
	onCopyChain,
	onDeleteChain,
	onSaveDefault,
	onError,
}: {
	commands: LifecycleCommand[]
	chains: LifecycleCommandChain[]
	defaults: Partial<Record<LifecycleHook, string>>
	onSaveCommand(command: LifecycleCommand): Promise<void>
	onDeleteCommand(commandID: string): Promise<void>
	onSaveChain(chain: LifecycleCommandChain): Promise<void>
	onCopyChain(chainID: string): Promise<void>
	onDeleteChain(chainID: string): Promise<void>
	onSaveDefault(hook: LifecycleHook, chainID: string): Promise<void>
	onError(error: unknown): void
}) {
	const [commandDraft, setCommandDraft] = useState<LifecycleCommand>()
	const [chainDraft, setChainDraft] = useState<LifecycleCommandChain>()
	const commandNames = new Map(commands.map((command) => [command.id, command.name]))
	const commandsByID = new Map(commands.map((command) => [command.id, command]))

	const saveCommand = async () => {
		if (!commandDraft) {
			return
		}
		try {
			await onSaveCommand({...commandDraft, name: commandDraft.name.trim(), command: commandDraft.command?.trim(), arguments: commandDraft.arguments.map((argument) => argument.trim()).filter(Boolean), applicableHooks: [...commandDraft.applicableHooks]})
			setCommandDraft(undefined)
		} catch (error) {
			onError(error)
		}
	}

	const saveChain = async () => {
		if (!chainDraft) {
			return
		}
		try {
			await onSaveChain({...chainDraft, name: chainDraft.name.trim(), commandIds: chainDraft.commandIds.filter(Boolean), applicableHooks: [...chainDraft.applicableHooks]})
			setChainDraft(undefined)
		} catch (error) {
			onError(error)
		}
	}

	const toggleChainCommand = (commandID: string, checked: boolean) => {
		setChainDraft((current) => current ? {
			...current,
			commandIds: checked ? [...current.commandIds, commandID] : current.commandIds.filter((currentID) => currentID !== commandID),
		} : current)
	}

	const toggleApplicableHook = (hook: LifecycleHook, target: 'command' | 'chain', checked: boolean) => {
		const update = (hooks: LifecycleHook[]) => checked ? [...hooks, hook] : hooks.filter((currentHook) => currentHook !== hook)
		if (target === 'command') {
			setCommandDraft((current) => current ? {...current, applicableHooks: update(current.applicableHooks)} : current)
			return
		}
		setChainDraft((current) => current ? {...current, applicableHooks: update(current.applicableHooks)} : current)
	}

	const incompatibleChainCommandIDs = chainDraft?.commandIds.filter((commandID) => {
		const command = commandsByID.get(commandID)
		return !command || !lifecycleHooksCover(command.applicableHooks, chainDraft.applicableHooks)
	}) ?? []

	const moveChainCommand = (index: number, offset: number) => {
		setChainDraft((current) => {
			if (!current || index + offset < 0 || index + offset >= current.commandIds.length) {
				return current
			}
			const commandIds = [...current.commandIds]
			const [commandID] = commandIds.splice(index, 1)
			commandIds.splice(index + offset, 0, commandID)
			return {...current, commandIds}
		})
	}

	const moveChainCommandTo = (fromIndex: number, toIndex: number) => {
		if (fromIndex === toIndex) {
			return
		}
		setChainDraft((current) => {
			if (!current || fromIndex < 0 || toIndex < 0 || fromIndex >= current.commandIds.length || toIndex >= current.commandIds.length) {
				return current
			}
			const commandIds = [...current.commandIds]
			const [commandID] = commandIds.splice(fromIndex, 1)
			commandIds.splice(toIndex, 0, commandID)
			return {...current, commandIds}
		})
	}

	return <Box component="section" sx={{display: 'grid', gap: 2.5}}>
		<Box sx={{display: 'grid', gap: 0.5}}>
			<Typography variant="subtitle2">五个钩子的默认链</Typography>
			<Typography variant="caption" color="text.secondary">新建任务会预选这里的链；任务保存后保留自己的选择。</Typography>
		</Box>
		<Box sx={{display: 'grid', gridTemplateColumns: {xs: '1fr', sm: '1fr 1fr'}, gap: 1.25}}>
			{lifecycleHooks.map((hook) => <TextField key={hook.id} select size="small" label={hook.label} value={defaults[hook.id] ?? ''} onChange={(event) => void onSaveDefault(hook.id, event.target.value).catch(onError)}>
				<MenuItem value="">不设置默认链</MenuItem>
				{chains.filter((chain) => lifecycleChainAppliesTo(chain, hook.id)).map((chain) => <MenuItem key={chain.id} value={chain.id}>{chain.name}</MenuItem>)}
			</TextField>)}
		</Box>

		<Box sx={{display: 'grid', gap: 1.25, pt: 0.5, borderTop: 1, borderColor: 'divider'}}>
			<Box sx={{display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 1}}>
				<Box><Typography variant="subtitle2">命令库</Typography><Typography variant="caption" color="text.secondary">系统目录命令只读；自定义命令通过所选 Shell 执行。</Typography></Box>
				<Button size="small" variant="contained" onClick={() => setCommandDraft({id: '', kind: 'custom', name: '', command: '', arguments: [], applicableHooks: []})}>新增命令</Button>
			</Box>
			<Box sx={{border: 1, borderColor: 'divider', borderRadius: 1.5, overflow: 'hidden'}}>
				{commands.map((command, index) => <Box key={command.id} sx={{display: 'grid', gridTemplateColumns: 'minmax(0, 1fr) auto', gap: 1, alignItems: 'center', px: 1.25, py: 1, borderBottom: index === commands.length - 1 ? 0 : 1, borderColor: 'divider'}}>
					<Box sx={{minWidth: 0}}><Typography variant="body2" sx={{fontWeight: 700}} noWrap>{command.name}</Typography><Typography variant="caption" color="text.secondary" noWrap>{command.kind === 'custom' ? `${command.command}${command.arguments.length ? ` ${command.arguments.join(' ')}` : ''}` : '系统内置目录操作'}</Typography><Typography variant="caption" color="text.secondary" sx={{display: 'block'}}>适用：{lifecycleHooksLabel(command.applicableHooks)}</Typography></Box>
					<Box sx={{display: 'flex', alignItems: 'center', gap: 0.5}}>
						<Chip size="small" label={command.kind === 'custom' ? '自定义' : '系统'} variant="outlined"/>
						{command.kind === 'custom' && <Button size="small" onClick={() => setCommandDraft({...command, arguments: [...command.arguments], applicableHooks: [...command.applicableHooks]})}>编辑</Button>}
						{command.kind === 'custom' && <IconButton aria-label={`删除命令 ${command.name}`} color="error" size="small" onClick={() => void onDeleteCommand(command.id).catch(onError)}><DeleteOutlineOutlinedIcon fontSize="inherit"/></IconButton>}
					</Box>
				</Box>)}
			</Box>
		</Box>

		<Box sx={{display: 'grid', gap: 1.25, pt: 0.5, borderTop: 1, borderColor: 'divider'}}>
			<Box sx={{display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 1}}>
				<Box><Typography variant="subtitle2">命令链</Typography><Typography variant="caption" color="text.secondary">同一条链可被多个钩子和任务复用，修改将在下次执行或重试时生效。</Typography></Box>
				<Button size="small" variant="contained" onClick={() => setChainDraft({id: '', name: '', commandIds: [], applicableHooks: []})}>新增链</Button>
			</Box>
			<Box sx={{border: 1, borderColor: 'divider', borderRadius: 1.5, overflow: 'hidden'}}>
				{chains.map((chain, index) => <Box key={chain.id} sx={{display: 'grid', gridTemplateColumns: {xs: '1fr', sm: 'minmax(0, 1fr) auto'}, gap: 1, alignItems: 'center', px: 1.25, py: 1, borderBottom: index === chains.length - 1 ? 0 : 1, borderColor: 'divider'}}>
					<Box sx={{minWidth: 0}}><Typography variant="body2" sx={{fontWeight: 700}} noWrap>{chain.name}</Typography><Typography variant="caption" color="text.secondary" noWrap>{chain.commandIds.map((commandID) => commandNames.get(commandID) ?? '已删除命令').join(' → ')}</Typography><Typography variant="caption" color="text.secondary" sx={{display: 'block'}}>适用：{lifecycleHooksLabel(chain.applicableHooks)}</Typography></Box>
					<Box sx={{display: 'flex', justifyContent: 'flex-end', gap: 0.5, flexWrap: 'wrap'}}><Button size="small" onClick={() => setChainDraft({...chain, commandIds: [...chain.commandIds], applicableHooks: [...chain.applicableHooks]})}>编辑</Button><Button size="small" onClick={() => void onCopyChain(chain.id).catch(onError)}>复制</Button><IconButton aria-label={`删除命令链 ${chain.name}`} color="error" size="small" onClick={() => void onDeleteChain(chain.id).catch(onError)}><DeleteOutlineOutlinedIcon fontSize="inherit"/></IconButton></Box>
				</Box>)}
			</Box>
		</Box>

		<Dialog open={Boolean(commandDraft)} onClose={() => setCommandDraft(undefined)} fullWidth maxWidth="sm">
			<DialogTitle>{commandDraft?.id ? '编辑命令' : '新增命令'}</DialogTitle>
			<DialogContent sx={{display: 'grid', gap: 1.5, pt: '12px !important'}}>
				<TextField autoFocus required label="命令名称" value={commandDraft?.name ?? ''} onChange={(event) => setCommandDraft((current) => current ? {...current, name: event.target.value} : current)}/>
				<TextField required label="可执行命令" helperText="由任务设置中选定的 Shell 启动。" value={commandDraft?.command ?? ''} onChange={(event) => setCommandDraft((current) => current ? {...current, command: event.target.value} : current)}/>
				<TextField multiline minRows={3} label="参数（每行一个）" value={commandDraft?.arguments.join('\n') ?? ''} onChange={(event) => setCommandDraft((current) => current ? {...current, arguments: event.target.value.split('\n')} : current)}/>
				<Box sx={{display: 'grid', gap: 0.5}}><Typography variant="subtitle2">适用范围</Typography>{lifecycleHooks.map((hook) => <FormControlLabel key={hook.id} control={<Checkbox checked={Boolean(commandDraft?.applicableHooks.includes(hook.id))} onChange={(event) => toggleApplicableHook(hook.id, 'command', event.target.checked)}/>} label={hook.label}/>)}</Box>
			</DialogContent>
			<DialogActions><Button onClick={() => setCommandDraft(undefined)}>取消</Button><Button variant="contained" disabled={!commandDraft?.applicableHooks.length} onClick={() => void saveCommand()}>保存命令</Button></DialogActions>
		</Dialog>

		<Dialog open={Boolean(chainDraft)} onClose={() => setChainDraft(undefined)} fullWidth maxWidth="sm">
			<DialogTitle>{chainDraft?.id ? '编辑命令链' : '新增命令链'}</DialogTitle>
			<DialogContent sx={{display: 'grid', gap: 1.5, pt: '12px !important'}}>
				<TextField autoFocus required label="命令链名称" value={chainDraft?.name ?? ''} onChange={(event) => setChainDraft((current) => current ? {...current, name: event.target.value} : current)}/>
				<Box sx={{display: 'grid', gap: 0.5}}><Typography variant="subtitle2">适用范围</Typography>{lifecycleHooks.map((hook) => <FormControlLabel key={hook.id} control={<Checkbox checked={Boolean(chainDraft?.applicableHooks.includes(hook.id))} onChange={(event) => toggleApplicableHook(hook.id, 'chain', event.target.checked)}/>} label={hook.label}/>)}</Box>
				<Box sx={{display: 'grid', gap: 0.5}}><Typography variant="subtitle2">可用命令</Typography>{chainDraft?.applicableHooks.length ? commands.filter((command) => lifecycleHooksCover(command.applicableHooks, chainDraft.applicableHooks) || chainDraft.commandIds.includes(command.id)).map((command) => <FormControlLabel key={command.id} control={<Checkbox checked={Boolean(chainDraft.commandIds.includes(command.id))} onChange={(event) => toggleChainCommand(command.id, event.target.checked)}/>} label={`${command.name}${lifecycleHooksCover(command.applicableHooks, chainDraft.applicableHooks) ? '' : '（与当前范围不匹配）'}`}/>) : <Typography variant="body2" color="text.secondary">请先选择命令链适用范围。</Typography>}</Box>
				{incompatibleChainCommandIDs.length > 0 && <Typography variant="body2" color="error">请移除与当前范围不匹配的命令后再保存。</Typography>}
				<Box sx={{display: 'grid', gap: 0.5}}>
					<Typography variant="subtitle2">执行顺序</Typography>
					{chainDraft?.commandIds.length ? chainDraft.commandIds.map((commandID, index) => (
						<Box
							key={`${commandID}-${index}`}
							draggable
							onDragStart={(event) => event.dataTransfer.setData('text/plain', String(index))}
							onDragOver={(event) => event.preventDefault()}
							onDrop={(event) => {
								const fromIndex = Number(event.dataTransfer.getData('text/plain'))
								moveChainCommandTo(fromIndex, index)
							}}
							sx={{display: 'flex', alignItems: 'center', gap: 0.5, border: 1, borderColor: 'divider', borderRadius: 1, px: 1, cursor: 'grab', '&:active': {cursor: 'grabbing'}}}
						>
							<Typography variant="body2" sx={{flex: 1}}>{index + 1}. {commandNames.get(commandID) ?? '已删除命令'}</Typography>
							<IconButton aria-label={`上移链命令 ${index + 1}`} size="small" disabled={index === 0} onClick={() => moveChainCommand(index, -1)}><ArrowUpwardOutlinedIcon fontSize="inherit"/></IconButton>
							<IconButton aria-label={`下移链命令 ${index + 1}`} size="small" disabled={index === chainDraft.commandIds.length - 1} onClick={() => moveChainCommand(index, 1)}><ArrowDownwardOutlinedIcon fontSize="inherit"/></IconButton>
						</Box>
					)) : <Typography variant="body2" color="text.secondary">至少选择一个命令。</Typography>}
				</Box>
			</DialogContent>
			<DialogActions><Button onClick={() => setChainDraft(undefined)}>取消</Button><Button variant="contained" disabled={!chainDraft?.applicableHooks.length || !chainDraft.commandIds.length || incompatibleChainCommandIDs.length > 0} onClick={() => void saveChain()}>保存命令链</Button></DialogActions>
		</Dialog>
	</Box>
}

function TaskExtraInfoEditor({
	templates,
	infos,
	extraInfo,
	onChange,
}: {
	templates: ExtraInfoTemplate[]
	infos: ExtraInfo[]
	extraInfo: TaskExtraInfo[]
	onChange: Dispatch<SetStateAction<TaskExtraInfo[]>>
}) {
	const [selectedCatalogue, setSelectedCatalogue] = useState('')
	const [informationSearch, setInformationSearch] = useState('')
	useEffect(() => {
		const catalogues = [...new Set(templates.map((template) => template.catalogue))]
		setSelectedCatalogue((current) => catalogues.includes(current) ? current : catalogues[0] ?? '')
	}, [templates])

	const catalogues = [...new Set(templates.map((template) => template.catalogue))]
	const informationIDs = new Set(infos.map((info) => info.id))
	const deletedSnapshots = extraInfo.filter((item) => !item.informationId || !informationIDs.has(item.informationId))
	const selectedByInformationID = new Map(extraInfo.filter((item) => item.informationId).map((item) => [item.informationId!, item]))
	const visibleInfos = infos.filter((info) => info.catalogue === selectedCatalogue && extraInfoName(info).toLocaleLowerCase().includes(informationSearch.trim().toLocaleLowerCase()))
	const templatesByID = new Map(templates.map((template) => [template.id, template]))

	const toggleInformation = (info: ExtraInfo, selected: boolean) => {
		const template = templatesByID.get(info.templateId)
		if (!template) {
			return
		}
		onChange((current) => selected
			? [...current, createTaskExtraInfo(info, template)]
			: current.filter((item) => item.informationId !== info.id),
		)
	}

	const updateParameter = (itemKey: string, parameterIndex: number, update: Partial<TaskExtraInfo['parameters'][number]>) => {
		onChange((current) => current.map((item) => taskExtraInfoKey(item) !== itemKey ? item : {
			...item,
			parameters: item.parameters.map((parameter, index) => index === parameterIndex ? {...parameter, ...update} : parameter),
		}))
	}

	return (
		<Box component="section" sx={{display: 'grid', gap: 1.25, borderTop: 1, borderColor: 'divider', pt: 1.75}}>
			<Box>
				<Typography variant="subtitle2">附加信息</Typography>
				<Typography variant="caption" color="text.secondary">选择信息后填写其动态参数。</Typography>
			</Box>
			<Box sx={{display: 'grid', gap: 0.75}}>
				<Typography variant="caption" color="text.secondary">已选择 {extraInfo.length} 项</Typography>
				{extraInfo.length > 0 && <Box sx={{display: 'flex', flexWrap: 'wrap', gap: 0.75}}>
					{extraInfo.map((item) => (
						<Chip key={taskExtraInfoKey(item)} size="small" label={item.displayName ?? '已保存信息'} onDelete={() => onChange((current) => current.filter((currentItem) => taskExtraInfoKey(currentItem) !== taskExtraInfoKey(item)))}/>
					))}
				</Box>}
			</Box>
			{catalogues.length === 0 && deletedSnapshots.length === 0 && (
				<Typography variant="body2" color="text.secondary">暂无可选额外信息，可通过顶部“额外信息管理”添加。</Typography>
			)}
			{catalogues.length > 0 && <>
				<TextField select size="small" label="选择分类" value={selectedCatalogue} onChange={(event) => setSelectedCatalogue(event.target.value)}>
					{catalogues.map((catalogue) => <MenuItem key={catalogue} value={catalogue}>{catalogue}</MenuItem>)}
				</TextField>
				<TextField size="small" label="搜索信息" placeholder="按名称模糊搜索" value={informationSearch} onChange={(event) => setInformationSearch(event.target.value)}/>
				<Box sx={{display: 'grid', gap: 0.75, border: 1, borderColor: 'divider', borderRadius: 1, p: 1.25}}>
					{visibleInfos.length === 0 ? (
						<Typography variant="body2" color="text.secondary">该分类未找到可选信息。</Typography>
					) : visibleInfos.map((info) => {
						const selected = selectedByInformationID.get(info.id)
						return (
							<Box key={info.id} sx={{display: 'grid', gap: selected ? 1 : 0}}>
								<FormControlLabel
									control={<Checkbox checked={Boolean(selected)} onChange={(event) => toggleInformation(info, event.target.checked)}/>}
									label={extraInfoName(info)}
								/>
								{selected && <TaskExtraInfoSnapshotFields item={selected} onChange={updateParameter}/>}
							</Box>
						)
					})}
				</Box>
			</>}
			{deletedSnapshots.map((item) => (
				<Box key={taskExtraInfoKey(item)} sx={{display: 'grid', gap: 1, border: 1, borderColor: 'warning.light', borderRadius: 1, p: 1.25}}>
					<Box sx={{display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 1}}>
						<Typography variant="body2" sx={{fontWeight: 700}}>{item.displayName ?? '信息'}（已删除）</Typography>
					</Box>
					<TaskExtraInfoSnapshotFields item={item} onChange={updateParameter}/>
				</Box>
			))}
		</Box>
	)
}

function TaskExtraInfoSnapshotFields({
	item,
	onChange,
}: {
	item: TaskExtraInfo
	onChange: (itemKey: string, parameterIndex: number, update: Partial<TaskExtraInfo['parameters'][number]>) => void
}) {
	const itemKey = taskExtraInfoKey(item)
	return (
		<Box sx={{display: 'grid', gap: 1.25, pl: {xs: 0, sm: 1}, minWidth: 0}}>
			{item.parameters.map((parameter, parameterIndex) => {
				const inputType = extraInfoParameterInputType(parameter)
				return <Box key={`parameter-${parameterIndex}`} data-testid={`task-extra-info-parameter-${itemKey}-${parameterIndex}`} sx={{display: 'grid', gap: 1.25, minWidth: 0, borderTop: 1, borderColor: 'divider', py: 1.5}}>
					{inputType === 'checkbox'
						? <FormControlLabel sx={{m: 0, minWidth: 0}} control={<Checkbox checked={parameter.value === 'true'} onChange={(event) => onChange(itemKey, parameterIndex, {value: event.target.checked ? 'true' : 'false'})}/>} label={parameter.displayName || '动态参数'}/>
						: <TextField size="small" label={parameter.displayName || '动态参数'} value={parameter.value} required={parameter.required} onChange={(event) => onChange(itemKey, parameterIndex, {value: event.target.value})}/>
					}
				</Box>
			})}
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
		{task.lifecycleExecution && <Alert severity={task.lifecycleExecution.state === 'failed' ? 'error' : 'warning'} variant="outlined" sx={{mb: 3}}>
			<Typography variant="subtitle2">{lifecycleHooks.find((hook) => hook.id === task.lifecycleExecution?.hook)?.label ?? '生命周期'}：{task.lifecycleExecution.currentCommandName || '命令'}（{task.lifecycleExecution.currentIndex}/{task.lifecycleExecution.commandCount}）</Typography>
			{task.lifecycleExecution.error && <Typography variant="body2" sx={{mt: 0.5, whiteSpace: 'pre-wrap'}}>{task.lifecycleExecution.error}</Typography>}
		</Alert>}
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

function createExtraInfoTemplateDraft(catalogue: string): ExtraInfoTemplate {
	return {id: '', catalogue, displayName: '', builtIn: false, fields: [{key: 'name', displayName: '名称', defaultValue: ''}], parameters: []}
}

function cloneExtraInfoTemplate(template: ExtraInfoTemplate): ExtraInfoTemplate {
	return {...template, fields: template.fields.map((field) => ({...field})), parameters: template.parameters.map((parameter) => ({...parameter}))}
}

function createExtraInfoDraft(template: ExtraInfoTemplate): ExtraInfo {
	return {
		id: '',
		templateId: template.id,
		catalogue: template.catalogue,
		fields: template.fields.map((field) => ({key: field.key, displayName: field.displayName, value: field.defaultValue ?? ''})),
		parameters: [],
	}
}

function cloneExtraInfo(info: ExtraInfo): ExtraInfo {
	return {...info, fields: info.fields.map((field) => ({...field})), parameters: (info.parameters ?? []).map((parameter) => ({...parameter}))}
}

function extraInfoName(info: ExtraInfo): string {
	return info.fields.find((field) => field.key === 'name')?.value ?? ''
}

function extraInfoParameterInputType(parameter: {inputType?: ExtraInfoParameterInputType}): ExtraInfoParameterInputType {
	return parameter.inputType === 'checkbox' ? 'checkbox' : 'text'
}

function createTaskExtraInfo(info: ExtraInfo, template: ExtraInfoTemplate): TaskExtraInfo {
	return {
		id: '',
		informationId: info.id,
		templateId: template.id,
		catalogue: info.catalogue,
		displayName: extraInfoName(info),
		fields: info.fields.map((field) => ({...field, defaultValue: undefined})),
		parameters: [
			...template.parameters.map((parameter) => {
				const inputType = extraInfoParameterInputType(parameter)
				return {...parameter, inputType, required: inputType === 'checkbox' ? false : parameter.required, value: inputType === 'checkbox' ? 'false' : ''}
			}),
			...(info.parameters ?? []).map((parameter) => {
				const inputType = extraInfoParameterInputType(parameter)
				return {...parameter, inputType, required: inputType === 'checkbox' ? false : parameter.required, value: inputType === 'checkbox' && parameter.value !== 'true' ? 'false' : parameter.value}
			}),
		],
	}
}

function taskExtraInfoKey(item: TaskExtraInfo): string {
	return item.id || item.informationId || `${item.catalogue}:${item.displayName ?? ''}`
}

function cloneTaskExtraInfo(extraInfo: TaskExtraInfo[]): TaskExtraInfo[] {
	return extraInfo.map((item) => ({...item, fields: item.fields.map((field) => ({...field})), parameters: item.parameters.map((parameter) => ({...parameter}))}))
}

function cloneTaskMenuItems(items: TaskMenuItem[]): TaskMenuItem[] {
  return items.map(cloneTaskMenuItem)
}

function cloneTaskMenuItem(item: TaskMenuItem): TaskMenuItem {
  return {
    ...item,
    arguments: item.arguments ? [...item.arguments] : undefined,
    beforeScript: cloneTaskScript(item.beforeScript),
    afterScript: cloneTaskScript(item.afterScript),
  }
}

function cloneTaskScript(script?: TaskScript): TaskScript | undefined {
	if (!script) {
		return undefined
	}
	return {...script, arguments: script.arguments ? [...script.arguments] : undefined}
}

function normalizeTaskScript(script?: TaskScript): TaskScript | undefined {
	const path = script?.script.trim()
	if (!path) {
		return undefined
	}
	const arguments_ = script?.arguments?.map((argument) => argument.trim()).filter(Boolean)
	return arguments_?.length ? {script: path, arguments: arguments_} : {script: path}
}

function updateTaskMenuItemScript(
	setDraft: Dispatch<SetStateAction<TaskMenuItem | undefined>>,
	key: 'beforeScript' | 'afterScript',
	update: Partial<TaskScript>,
) {
  setDraft((current) => current ? {
    ...current,
    [key]: {...current[key], script: current[key]?.script ?? '', arguments: current[key]?.arguments ?? [], ...update},
  } : current)
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
