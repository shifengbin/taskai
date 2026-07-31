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
import taskAiMark from './assets/task-ai-mark.svg'
import {TaskTree, type TaskStartFeedback} from './components/TaskTree'
import {TerminalView} from './components/TerminalView'
import {
	applyRealtimeStatusToTasks,
	applyRealtimeStatusToTerminals,
  applyTerminalEvent,
	bufferPendingRealtimeStatusEvent,
  bufferPendingTerminalEvent,
  clearTaskTerminalTracking,
	mergeLifecycleTask,
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
	randomTaskColor,
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
	type TaskTemplate,
	type TaskTemplateField,
	type TaskTemplateValues,
	type TaskMenuItem,
  type TaskStatus,
  type TerminalRecord,
} from './types'
import {ClipboardSetText} from '../wailsjs/runtime/runtime'
import './App.css'

type Notification = {
  text: string
  severity: 'success' | 'error'
}

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
	const [taskTemplateFieldsDraft, setTaskTemplateFieldsDraft] = useState<TaskTemplateValues>({})
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
	const [settingsTab, setSettingsTab] = useState<'workspace' | 'shell' | 'menu' | 'status' | 'lifecycle' | 'templates'>('workspace')
  const [statusHelpOpen, setStatusHelpOpen] = useState(false)
  const [taskMenuItemDraft, setTaskMenuItemDraft] = useState<TaskMenuItem>()
  const [taskMenuItemEditorMode, setTaskMenuItemEditorMode] = useState<'create' | 'edit'>()
  const [taskMenuItemEditorTab, setTaskMenuItemEditorTab] = useState<'basic' | 'scripts'>('basic')
  const [scriptHelpAnchor, setScriptHelpAnchor] = useState<HTMLElement>()
  const [message, setMessage] = useState<Notification>()
  const showErrorMessage = (text: string) => setMessage({text, severity: 'error'})
  const [startedTaskFeedback, setStartedTaskFeedback] = useState<TaskStartFeedback>()
  const dragging = useRef(false)
  const currentTreeWidth = useRef(treeWidth)
  const terminalTitleParserStates = useRef(new Map<string, TerminalTitleParserState>())
  const pendingTerminalEvents = useRef(new Map<string, PendingTerminalEvent>())
  const registeredTerminalKeys = useRef(new Set<string>())
  const finishedTerminalTaskIDs = useRef(new Set<string>())
  const terminalTitleValues = useRef(new Map<string, string>())
	const latestRealtimeStatusVersion = useRef(0)
	const lifecycleStatusTargets = useRef(new Map<string, TaskStatus>())
	const startFeedbackSequence = useRef(0)
	const startFeedbackTimeout = useRef<ReturnType<typeof setTimeout>>()
	const initializedExtraInfoGroups = useRef(false)

	const showStartedTaskFeedback = (taskID: string) => {
		if (startFeedbackTimeout.current) {
			clearTimeout(startFeedbackTimeout.current)
		}
		const feedback = {taskID, sequence: ++startFeedbackSequence.current}
		setStartedTaskFeedback(feedback)
		startFeedbackTimeout.current = setTimeout(() => {
			setStartedTaskFeedback((current) => current?.sequence === feedback.sequence ? undefined : current)
			startFeedbackTimeout.current = undefined
		}, 700)
	}

	const activateStartedTask = (taskID: string) => {
		lifecycleStatusTargets.current.delete(taskID)
		showStartedTaskFeedback(taskID)
		if (activeTaskStatus !== 'running') {
			void changeActiveTaskStatus('running')
		}
	}

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
        showErrorMessage(event.data || '终端发生错误')
      }
    })
    return () => {
      unsubscribe()
      terminalTitleParserStates.current.clear()
      pendingTerminalEvents.current.clear()
      registeredTerminalKeys.current.clear()
      finishedTerminalTaskIDs.current.clear()
      terminalTitleValues.current.clear()
			if (startFeedbackTimeout.current) {
				clearTimeout(startFeedbackTimeout.current)
			}
    }
  }, [])

  useEffect(() => api.onCloseRequested(() => setQuitDialogOpen(true)), [])

  useEffect(() => api.onTaskScriptError((message) => showErrorMessage(message)), [])

  useEffect(() => api.onRealtimeStatusError((message) => showErrorMessage(message)), [])

	useEffect(() => api.onLifecycleEvent((updated) => {
		setTasks((current) => mergeLifecycleTask(current, updated))
		const targetStatus = lifecycleStatusTargets.current.get(updated.id)
		if (targetStatus === 'running' && updated.status === 'running') {
			activateStartedTask(updated.id)
		}
	}), [activeTaskStatus, settings])

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
	const activeTaskTemplate = settings?.taskTemplates?.find((template) => template.id === settings.activeTaskTemplateId)
	const areAllTasksExpanded = tasks.length > 0 && tasks.every((task) => expandedTasks[task.id] ?? true)

	const copyLifecycleCommandInput = async (taskID: string) => {
		const input = await api.getLifecycleCommandInput(taskID)
		if (!await ClipboardSetText(input)) {
			throw new Error('无法写入系统剪贴板')
		}
		setMessage({text: '已复制当前命令链输入 JSON', severity: 'success'})
	}

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
    setDraftColor(task ? task.color || defaultTaskColor : randomTaskColor())
		setTaskExtraInfoDraft(task?.extraInfo ? cloneTaskExtraInfo(task.extraInfo) : [])
		setTaskTemplateFieldsDraft(resolveTaskTemplateValues(activeTaskTemplate, task?.templateFields))
		setTaskLifecycleChainsDraft({...task?.lifecycleChains ?? settings?.lifecycleDefaultChains ?? {}})
    setTaskDialogOpen(true)
  }

  const closeTaskDialog = () => {
    setTaskDialogOpen(false)
    setEditingTask(undefined)
		setTaskExtraInfoDraft([])
		setTaskTemplateFieldsDraft({})
		setTaskLifecycleChainsDraft({})
  }

  const saveTask = async (event: FormEvent) => {
    event.preventDefault()
    if (!draftTitle.trim()) {
      showErrorMessage('任务标题不能为空')
      return
    }
		const missingParameter = taskExtraInfoDraft.flatMap((item) => item.parameters).find((parameter) => parameter.required && !parameter.value.trim())
		if (missingParameter) {
			showErrorMessage(`参数“${missingParameter.displayName}”不能为空`)
			return
		}
		const invalidParameter = taskExtraInfoDraft.flatMap((item) => item.parameters).find((parameter) => !parameter.key.trim() || !parameter.displayName.trim())
		if (invalidParameter) {
			showErrorMessage('动态参数的键和显示名称不能为空')
			return
		}
		const duplicateParameter = taskExtraInfoDraft.find((item) => new Set(item.parameters.map((parameter) => parameter.key.trim())).size !== item.parameters.length)
		if (duplicateParameter) {
			showErrorMessage(`信息“${duplicateParameter.displayName ?? duplicateParameter.catalogue}”包含重复的动态参数键`)
			return
		}
		const missingTemplateField = activeTaskTemplate?.fields.find((field) => {
			if (!field.required) {
				return false
			}
			const value = taskTemplateFieldsDraft[field.key]
			return field.inputType === 'bool' ? value !== true : !String(value ?? '').trim()
		})
		if (missingTemplateField) {
			showErrorMessage(missingTemplateField.inputType === 'bool'
				? `字段“${missingTemplateField.displayName}”必须勾选`
				: `字段“${missingTemplateField.displayName}”不能为空`)
			return
		}
    try {
      if (editingTask) {
			const hasExtraInfo = taskExtraInfoDraft.length > 0 || (editingTask.extraInfo?.length ?? 0) > 0
			const updated = editingTask.status === 'pending'
					? activeTaskTemplate
						? await api.updateTaskWithExtraInfoTemplateFieldsAndLifecycleChains(editingTask.id, draftTitle, draftDescription, draftColor, taskExtraInfoDraft, taskTemplateFieldsDraft, taskLifecycleChainsDraft)
						: await api.updateTaskWithExtraInfoAndLifecycleChains(editingTask.id, draftTitle, draftDescription, draftColor, taskExtraInfoDraft, taskLifecycleChainsDraft)
					: activeTaskTemplate
						? await api.updateTaskWithExtraInfoAndTemplateFields(editingTask.id, draftTitle, draftDescription, draftColor, taskExtraInfoDraft, taskTemplateFieldsDraft)
					: hasExtraInfo
						? await api.updateTaskWithExtraInfo(editingTask.id, draftTitle, draftDescription, draftColor, taskExtraInfoDraft)
						: await api.updateTask(editingTask.id, draftTitle, draftDescription, draftColor)
        setTasks((current) => mergeLifecycleTask(current, updated))
      } else {
			const hasLifecycleChainSelection = Object.keys(taskLifecycleChainsDraft).length > 0
			const created = hasLifecycleChainSelection
				? activeTaskTemplate
					? await api.createTaskWithExtraInfoTemplateFieldsAndLifecycleChains(draftTitle, draftDescription, draftColor, taskExtraInfoDraft, taskTemplateFieldsDraft, taskLifecycleChainsDraft)
					: await api.createTaskWithExtraInfoAndLifecycleChains(draftTitle, draftDescription, draftColor, taskExtraInfoDraft, taskLifecycleChainsDraft)
				: activeTaskTemplate
					? await api.createTaskWithExtraInfoAndTemplateFields(draftTitle, draftDescription, draftColor, taskExtraInfoDraft, taskTemplateFieldsDraft)
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
			showErrorMessage('信息名称不能为空')
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
		lifecycleStatusTargets.current.set(taskID, 'running')
		setSelectedTaskID(taskID)
		setSelectedTerminalID(undefined)
    try {
      const started = await api.startTask(taskID)
		setTasks((current) => {
			const existing = current.find((task) => task.id === taskID)
			if (existing?.status === 'running' && started.status === 'pending') {
				return current
			}
			return mergeLifecycleTask(current, started)
		})
      if (started.status === 'running') {
			activateStartedTask(taskID)
      }
    } catch (error) {
		lifecycleStatusTargets.current.delete(taskID)
      showError(error, setMessage)
    }
  }

	const retryLifecycleChain = async (taskID: string) => {
		try {
			const updated = await api.retryTaskLifecycleCommandChain(taskID)
			setTasks((current) => mergeLifecycleTask(current, updated))
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
      setTasks((current) => mergeLifecycleTask(current, completed))
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
      showErrorMessage('菜单名称和启动命令不能为空')
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
      <Box className={`taskai-app taskai-app--${colorScheme}`} data-color-scheme={colorScheme} sx={{height: '100vh', minWidth: 720, display: 'grid', gridTemplateRows: '52px minmax(0, 1fr)', overflow: 'hidden'}}>
        <AppBar className="taskai-topbar" position="static" color="transparent" elevation={0} sx={{borderBottom: 1, borderColor: 'divider', bgcolor: 'background.paper'}}>
          <Toolbar className="taskai-topbar__toolbar" variant="dense" sx={{minHeight: '52px !important', gap: 1}}>
            <Box component="img" className="taskai-brand-mark" src={taskAiMark} alt="任务 AI 图标"/>
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
          <Box className="taskai-sidebar-shell" sx={{minWidth: 0, minHeight: 0, display: 'grid', gridTemplateRows: '42px minmax(0, 1fr)', borderRight: 1, borderColor: 'divider', bgcolor: 'background.paper'}}>
            <Box className="taskai-sidebar-header" sx={{height: 42, display: 'flex', alignItems: 'center', px: 1.25, borderBottom: 1, borderColor: 'divider'}}>
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
            <Box sx={{minHeight: 0}}>
              <TaskTree
                tasks={tasks}
                terminals={terminals}
                menuItems={taskMenuItems}
                activeStatus={activeTaskStatus}
                expandedTasks={expandedTasks}
                selectedTaskID={selectedTaskID}
                selectedTerminalId={selectedTerminalID}
                startedTaskFeedback={startedTaskFeedback}
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
            className="taskai-panel-resizer"
            sx={{cursor: 'col-resize', bgcolor: 'divider', '&:hover': {bgcolor: 'primary.main'}}}
          />
          <Box className="taskai-content-pane" sx={{minWidth: 0, minHeight: 0, bgcolor: 'background.default'}}>
            {selectedTerminal ? (
              <TerminalView
                key={selectedTerminal.id}
                terminal={selectedTerminal}
                onWrite={(data) => void api.writeTerminal(selectedTerminal.taskId, selectedTerminal.id, data).catch((error) => showError(error, setMessage))}
                onResize={(columns, rows) => void api.resizeTerminal(selectedTerminal.taskId, selectedTerminal.id, columns, rows).catch((error) => showError(error, setMessage))}
                onClose={() => void closeTerminal(selectedTerminal)}
              />
            ) : (
              <TaskDetail
				task={selectedTask}
				template={activeTaskTemplate}
				onCopyLifecycleCommandInput={(taskID) => void copyLifecycleCommandInput(taskID).catch((error) => showError(error, setMessage))}
			/>
            )}
          </Box>
        </Box>
      </Box>

      <Dialog
        open={taskDialogOpen}
        onClose={closeTaskDialog}
        scroll="paper"
        fullWidth
        maxWidth="sm"
        slotProps={{paper: {sx: {maxHeight: 'calc(100dvh - 32px)'}}}}
      >
        <Box component="form" onSubmit={saveTask} sx={{display: 'flex', flexDirection: 'column', flex: '1 1 auto', minHeight: 0, overflow: 'hidden'}}>
          <DialogTitle sx={{flexShrink: 0}}>{editingTask ? '编辑任务' : '新建任务'}</DialogTitle>
          <DialogContent data-testid="task-dialog-content" sx={{display: 'grid', gap: 2, flex: '1 1 auto', minHeight: 0, overflowY: 'auto', pt: '12px !important'}}>
            <TextField autoFocus required label="标题" value={draftTitle} onChange={(event) => setDraftTitle(event.target.value)}/>
            <TextField label="任务描述" value={draftDescription} multiline minRows={3} onChange={(event) => setDraftDescription(event.target.value)}/>
			<TaskTemplateFieldsEditor template={activeTaskTemplate} values={taskTemplateFieldsDraft} onChange={setTaskTemplateFieldsDraft}/>
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
          <DialogActions sx={{flexShrink: 0}}>
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
			onChange={(_, value: 'workspace' | 'shell' | 'menu' | 'status' | 'lifecycle' | 'templates') => setSettingsTab(value)}
            aria-label="设置分类"
            variant="scrollable"
            scrollButtons="auto"
          >
            <Tab value="workspace" label="工作区与外观"/>
            <Tab value="shell" label="终端 Shell"/>
            <Tab value="menu" label="菜单管理"/>
			<Tab value="status" label="实时状态"/>
			<Tab value="lifecycle" label="生命周期编排"/>
			<Tab value="templates" label="任务模板"/>
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

			{settingsTab === 'templates' && <TaskTemplateManagement
				templates={settingsDraft?.taskTemplates ?? []}
				activeTemplateID={settingsDraft?.activeTaskTemplateId ?? ''}
				onChange={(taskTemplates, activeTaskTemplateId) => updateSettingsDraft({taskTemplates, activeTaskTemplateId})}
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
              <HTTPHelpStep number="1" title="独立启用服务">在“实时状态”中开启“启用本机 HTTP 服务”并设置端口，可单独查询任务和状态；本机 HTTP 服务正在监听时，之后新建的终端会获得 API 地址。</HTTPHelpStep>
              <HTTPHelpStep number="2" title="使用 HTTP 管理状态">选择“通过 HTTP 接口”会自动启用服务，因此之后新建的终端也会获得 API 地址。</HTTPHelpStep>
            </Box>
          </HTTPHelpSection>

          <HTTPHelpSection title="终端环境变量">
            <Typography variant="body2" color="text.secondary">新建的普通终端和显示终端的自定义命令始终获得任务与终端 ID；本机 HTTP 服务正在监听时额外获得 API 地址：</Typography>
            <HTTPCodeBlock>{'TASKAI_TASK_ID=<任务 ID>\nTASKAI_TERMINAL_ID=<终端 ID>\n\n# 仅本机 HTTP 服务正在监听时注入\nTASKAI_STATUS_API=http://127.0.0.1:<端口>/api/v1'}</HTTPCodeBlock>
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
              <Typography variant="body2" color="text.secondary">修改端口、服务开关或切换状态管理方式不会更新已运行终端的环境变量；请新建终端后再使用新配置。</Typography>
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

      {message && <Snackbar open autoHideDuration={5000} onClose={() => setMessage(undefined)}>
        <Alert severity={message.severity} variant="filled" onClose={() => setMessage(undefined)}>{message.text}</Alert>
      </Snackbar>}
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
  const isDark = colorScheme === 'dark'
  const colors = isDark
    ? {
        canvas: '#19221d', paper: '#222d26', sidebar: '#29362c', detail: '#1d2821', ink: '#e7eee7', muted: '#a6b5a7', divider: '#3c4b41',
        primary: '#9cc3ab', accent: '#db9a7f', track: '#dce9df', error: '#ef9a8a', warning: '#e3b66f', info: '#9dc9dc', contrast: '#16271d',
      }
    : {
        canvas: '#edf0ea', paper: '#fafbf7', sidebar: '#e4ebe3', detail: '#ffffff', ink: '#29332c', muted: '#657268', divider: '#b8c5b9',
        primary: '#547565', accent: '#b16b50', track: '#dbe8de', error: '#ae4c40', warning: '#916521', info: '#376d80', contrast: '#ffffff',
      }
  const secondaryContrast = isDark ? '#2b241e' : '#ffffff'

  return createTheme({
    palette: {
      mode: colorScheme,
      primary: {main: colors.primary, contrastText: colors.contrast},
      secondary: {main: colors.accent, contrastText: secondaryContrast},
      success: {main: colors.primary, contrastText: colors.contrast},
      warning: {main: colors.warning, contrastText: colors.contrast},
      error: {main: colors.error, contrastText: colors.contrast},
      info: {main: colors.info, contrastText: colors.contrast},
      background: {default: colors.canvas, paper: colors.paper},
      text: {primary: colors.ink, secondary: colors.muted},
      divider: colors.divider,
      action: {
        active: colors.primary,
        hover: isDark ? 'rgba(156, 195, 171, 0.13)' : 'rgba(84, 117, 101, 0.09)',
        selected: isDark ? 'rgba(156, 195, 171, 0.20)' : 'rgba(84, 117, 101, 0.14)',
        disabled: isDark ? 'rgba(231, 238, 231, 0.34)' : 'rgba(41, 51, 44, 0.35)',
        disabledBackground: isDark ? 'rgba(231, 238, 231, 0.10)' : 'rgba(41, 51, 44, 0.08)',
      },
    },
    shape: {borderRadius: 8},
    typography: {
      fontFamily: 'Nunito, "Noto Sans SC", sans-serif',
      button: {fontWeight: 750, letterSpacing: 0},
      overline: {fontWeight: 800, letterSpacing: '0.1em'},
      subtitle1: {fontFamily: '"Songti SC", "Noto Serif CJK SC", Nunito, serif', fontWeight: 700, letterSpacing: 0},
      h5: {fontFamily: '"Songti SC", "Noto Serif CJK SC", Nunito, serif', fontWeight: 700, letterSpacing: 0},
    },
    components: {
      MuiCssBaseline: {
        styleOverrides: {
          '*': {boxSizing: 'border-box'},
          '::selection': {backgroundColor: colors.track, color: colors.ink},
        },
      },
      MuiPaper: {styleOverrides: {root: {backgroundImage: 'none'}}},
      MuiAppBar: {styleOverrides: {root: {color: colors.ink, backgroundImage: 'none'}}},
      MuiButtonBase: {styleOverrides: {root: {'&:focus-visible': {outline: `3px solid ${colors.primary}`, outlineOffset: 2}}}},
      MuiIconButton: {styleOverrides: {root: {
        border: '1px solid transparent', borderRadius: 6,
        '&:hover': {borderColor: colors.divider, backgroundColor: isDark ? 'rgba(156, 195, 171, 0.16)' : 'rgba(84, 117, 101, 0.10)'},
      }}},
      MuiButton: {styleOverrides: {root: {borderRadius: 6, boxShadow: 'none'}, contained: {boxShadow: 'none'}}},
      MuiDialog: {styleOverrides: {paper: {
        border: `1px solid ${colors.divider}`, borderRadius: 10, boxShadow: isDark ? '0 18px 40px rgba(0, 0, 0, 0.38)' : '0 18px 40px rgba(41, 51, 44, 0.14)', overflow: 'hidden',
      }}},
      MuiDialogTitle: {styleOverrides: {root: {
        padding: '16px 20px', color: colors.ink, fontWeight: 900,
        backgroundColor: colors.paper,
      }}},
      MuiDialogActions: {styleOverrides: {root: {padding: '14px 20px', borderTop: `1px solid ${colors.divider}`, backgroundColor: colors.sidebar}}},
      MuiOutlinedInput: {styleOverrides: {root: {borderRadius: 6, backgroundColor: colors.detail, '&:hover .MuiOutlinedInput-notchedOutline': {borderColor: colors.primary}, '&.Mui-focused .MuiOutlinedInput-notchedOutline': {borderColor: colors.primary, borderWidth: 2}}}},
      MuiInputLabel: {styleOverrides: {root: {fontWeight: 800}}},
      MuiTabs: {styleOverrides: {indicator: {height: 3, borderRadius: 99, backgroundColor: colors.primary}}},
      MuiTab: {styleOverrides: {root: {fontWeight: 800, '&.Mui-selected': {color: colors.ink, backgroundColor: isDark ? 'rgba(156, 195, 171, 0.14)' : 'rgba(84, 117, 101, 0.10)'}}}},
      MuiChip: {styleOverrides: {root: {borderRadius: 5, fontWeight: 750}}},
      MuiAccordion: {styleOverrides: {root: {backgroundColor: colors.detail, '&.Mui-expanded': {boxShadow: `inset 3px 0 0 ${colors.primary}`}}}},
      MuiMenu: {styleOverrides: {paper: {border: `1px solid ${colors.divider}`, borderRadius: 8, boxShadow: isDark ? '0 12px 28px rgba(0, 0, 0, 0.30)' : '0 12px 28px rgba(41, 51, 44, 0.12)'}}},
      MuiPopover: {styleOverrides: {paper: {border: `1px solid ${colors.divider}`, borderRadius: 8, boxShadow: isDark ? '0 12px 28px rgba(0, 0, 0, 0.30)' : '0 12px 28px rgba(41, 51, 44, 0.12)'}}},
      MuiMenuItem: {styleOverrides: {root: {gap: 0.5, fontWeight: 750, '&:hover': {backgroundColor: isDark ? 'rgba(156, 195, 171, 0.14)' : 'rgba(84, 117, 101, 0.10)'}}}},
      MuiAlert: {styleOverrides: {root: {borderRadius: 6, fontWeight: 700}}},
      MuiTooltip: {styleOverrides: {tooltip: {border: `1px solid ${colors.divider}`, borderRadius: 5, color: colors.ink, backgroundColor: colors.detail, fontWeight: 700}}},
      MuiSwitch: {styleOverrides: {track: {opacity: 1, backgroundColor: colors.muted}, switchBase: {'&.Mui-checked + .MuiSwitch-track': {opacity: 1, backgroundColor: colors.primary}}}},
    },
  })
}

function StartupScreen() {
  return (
    <Box
      role="status"
      aria-label="正在加载任务工作台"
      className="taskai-startup"
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

function lifecycleCommandAllowsChainArguments(command?: LifecycleCommand) {
	return command?.chainArgumentMode !== 'disabled'
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
	const [chainHelpOpen, setChainHelpOpen] = useState(false)
	const commandNames = new Map(commands.map((command) => [command.id, command.name]))
	const commandsByID = new Map(commands.map((command) => [command.id, command]))

	const saveCommand = async () => {
		if (!commandDraft) {
			return
		}
		try {
			await onSaveCommand({...commandDraft, name: commandDraft.name.trim(), command: commandDraft.command?.trim(), arguments: commandDraft.arguments.map((argument) => argument.trim()).filter(Boolean), chainArgumentMode: commandDraft.chainArgumentMode === 'enabled' ? 'enabled' : 'disabled', applicableHooks: [...commandDraft.applicableHooks]})
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
			await onSaveChain({
				...chainDraft,
				name: chainDraft.name.trim(),
				commands: chainDraft.commands.filter((reference) => reference.commandId).map((reference) => ({
					commandId: reference.commandId,
					arguments: reference.arguments.map((argument) => argument.trim()).filter(Boolean),
				})),
				applicableHooks: [...chainDraft.applicableHooks],
			})
			setChainDraft(undefined)
		} catch (error) {
			onError(error)
		}
	}

	const toggleChainCommand = (commandID: string, checked: boolean) => {
		setChainDraft((current) => current ? {
			...current,
			commands: checked
				? [...current.commands, {commandId: commandID, arguments: []}]
				: current.commands.filter((reference) => reference.commandId !== commandID),
		} : current)
	}

	const updateChainCommandArguments = (index: number, referenceArguments: string[]) => {
		setChainDraft((current) => {
			if (!current || index < 0 || index >= current.commands.length) {
				return current
			}
			const commands = [...current.commands]
			commands[index] = {...commands[index], arguments: referenceArguments}
			return {...current, commands}
		})
	}

	const toggleApplicableHook = (hook: LifecycleHook, target: 'command' | 'chain', checked: boolean) => {
		const update = (hooks: LifecycleHook[]) => checked ? [...hooks, hook] : hooks.filter((currentHook) => currentHook !== hook)
		if (target === 'command') {
			setCommandDraft((current) => current ? {...current, applicableHooks: update(current.applicableHooks)} : current)
			return
		}
		setChainDraft((current) => current ? {...current, applicableHooks: update(current.applicableHooks)} : current)
	}

	const incompatibleChainCommands = chainDraft?.commands.filter((reference) => {
		const command = commandsByID.get(reference.commandId)
		return !command || !lifecycleHooksCover(command.applicableHooks, chainDraft.applicableHooks)
	}) ?? []

	const moveChainCommand = (index: number, offset: number) => {
		setChainDraft((current) => {
			if (!current || index + offset < 0 || index + offset >= current.commands.length) {
				return current
			}
			const commands = [...current.commands]
			const [command] = commands.splice(index, 1)
			commands.splice(index + offset, 0, command)
			return {...current, commands}
		})
	}

	const moveChainCommandTo = (fromIndex: number, toIndex: number) => {
		if (fromIndex === toIndex) {
			return
		}
		setChainDraft((current) => {
			if (!current || fromIndex < 0 || toIndex < 0 || fromIndex >= current.commands.length || toIndex >= current.commands.length) {
				return current
			}
			const commands = [...current.commands]
			const [command] = commands.splice(fromIndex, 1)
			commands.splice(toIndex, 0, command)
			return {...current, commands}
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
				<Button size="small" variant="contained" onClick={() => setCommandDraft({id: '', kind: 'custom', name: '', command: '', arguments: [], chainArgumentMode: 'disabled', applicableHooks: []})}>新增命令</Button>
			</Box>
			<Box sx={{border: 1, borderColor: 'divider', borderRadius: 1.5, overflow: 'hidden'}}>
				{commands.map((command, index) => <Box key={command.id} sx={{display: 'grid', gridTemplateColumns: 'minmax(0, 1fr) auto', gap: 1, alignItems: 'center', px: 1.25, py: 1, borderBottom: index === commands.length - 1 ? 0 : 1, borderColor: 'divider'}}>
					<Box sx={{minWidth: 0}}><Typography variant="body2" sx={{fontWeight: 700}} noWrap>{command.name}</Typography><Typography variant="caption" color="text.secondary" noWrap>{command.kind === 'custom' ? `${command.command}${command.arguments.length ? ` ${command.arguments.join(' ')}` : ''}` : '系统内置命令'}</Typography>{command.documentation && <Typography variant="caption" color="text.secondary" sx={{display: 'block', whiteSpace: 'pre-wrap'}}>{command.documentation}</Typography>}<Typography variant="caption" color="text.secondary" sx={{display: 'block'}}>适用：{lifecycleHooksLabel(command.applicableHooks)}</Typography></Box>
					<Box sx={{display: 'flex', alignItems: 'center', gap: 0.5}}>
						<Chip size="small" label={command.kind === 'custom' ? '自定义' : '系统'} variant="outlined"/>
						{command.kind === 'custom' && <Button size="small" aria-label={`编辑命令 ${command.name}`} onClick={() => setCommandDraft({...command, arguments: [...command.arguments], chainArgumentMode: command.chainArgumentMode === 'disabled' ? 'disabled' : 'enabled', applicableHooks: [...command.applicableHooks]})}>编辑</Button>}
						{command.kind === 'custom' && <IconButton aria-label={`删除命令 ${command.name}`} color="error" size="small" onClick={() => void onDeleteCommand(command.id).catch(onError)}><DeleteOutlineOutlinedIcon fontSize="inherit"/></IconButton>}
					</Box>
				</Box>)}
			</Box>
		</Box>

		<Box sx={{display: 'grid', gap: 1.25, pt: 0.5, borderTop: 1, borderColor: 'divider'}}>
			<Box sx={{display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 1}}>
				<Box><Typography variant="subtitle2">命令链</Typography><Typography variant="caption" color="text.secondary">同一条链可被多个钩子和任务复用，修改将在下次执行或重试时生效。</Typography></Box>
				<Box sx={{display: 'flex', alignItems: 'center', gap: 0.75, flexWrap: 'wrap', justifyContent: 'flex-end'}}>
					<Button size="small" variant="outlined" onClick={() => setChainHelpOpen(true)}>查看命令链扩展规则</Button>
					<Button size="small" variant="contained" onClick={() => setChainDraft({id: '', name: '', commands: [], applicableHooks: []})}>新增链</Button>
				</Box>
			</Box>
			<Box sx={{border: 1, borderColor: 'divider', borderRadius: 1.5, overflow: 'hidden'}}>
				{chains.map((chain, index) => <Box key={chain.id} sx={{display: 'grid', gridTemplateColumns: {xs: '1fr', sm: 'minmax(0, 1fr) auto'}, gap: 1, alignItems: 'center', px: 1.25, py: 1, borderBottom: index === chains.length - 1 ? 0 : 1, borderColor: 'divider'}}>
					<Box sx={{minWidth: 0}}><Typography variant="body2" sx={{fontWeight: 700}} noWrap>{chain.name}</Typography><Typography variant="caption" color="text.secondary" noWrap>{chain.commands.map((reference) => commandNames.get(reference.commandId) ?? '已删除命令').join(' → ')}</Typography><Typography variant="caption" color="text.secondary" sx={{display: 'block'}}>适用：{lifecycleHooksLabel(chain.applicableHooks)}</Typography></Box>
					<Box sx={{display: 'flex', justifyContent: 'flex-end', gap: 0.5, flexWrap: 'wrap'}}><Button size="small" aria-label={`编辑命令链 ${chain.name}`} onClick={() => setChainDraft({...chain, commands: chain.commands.map((reference) => ({...reference, arguments: [...reference.arguments]})), applicableHooks: [...chain.applicableHooks]})}>编辑</Button><Button size="small" onClick={() => void onCopyChain(chain.id).catch(onError)}>复制</Button><IconButton aria-label={`删除命令链 ${chain.name}`} color="error" size="small" onClick={() => void onDeleteChain(chain.id).catch(onError)}><DeleteOutlineOutlinedIcon fontSize="inherit"/></IconButton></Box>
				</Box>)}
			</Box>
		</Box>

		<Dialog open={Boolean(commandDraft)} onClose={() => setCommandDraft(undefined)} fullWidth maxWidth="sm">
			<DialogTitle>{commandDraft?.id ? '编辑命令' : '新增命令'}</DialogTitle>
			<DialogContent sx={{display: 'grid', gap: 1.5, pt: '12px !important'}}>
				<TextField autoFocus required label="命令名称" value={commandDraft?.name ?? ''} onChange={(event) => setCommandDraft((current) => current ? {...current, name: event.target.value} : current)}/>
				<TextField required label="可执行命令" helperText="由任务设置中选定的 Shell 启动。" value={commandDraft?.command ?? ''} onChange={(event) => setCommandDraft((current) => current ? {...current, command: event.target.value} : current)}/>
				<TextField multiline minRows={3} label="固定参数（每行一个）" value={commandDraft?.arguments.join('\n') ?? ''} onChange={(event) => setCommandDraft((current) => current ? {...current, arguments: event.target.value.split('\n')} : current)}/>
				<FormControlLabel control={<Checkbox checked={commandDraft?.chainArgumentMode === 'enabled'} onChange={(event) => setCommandDraft((current) => current ? {...current, chainArgumentMode: event.target.checked ? 'enabled' : 'disabled'} : current)}/>} label="允许在命令链中追加参数"/>
				<Box sx={{display: 'grid', gap: 0.5}}><Typography variant="subtitle2">适用范围</Typography>{lifecycleHooks.map((hook) => <FormControlLabel key={hook.id} control={<Checkbox checked={Boolean(commandDraft?.applicableHooks.includes(hook.id))} onChange={(event) => toggleApplicableHook(hook.id, 'command', event.target.checked)}/>} label={hook.label}/>)}</Box>
			</DialogContent>
			<DialogActions><Button onClick={() => setCommandDraft(undefined)}>取消</Button><Button variant="contained" disabled={!commandDraft?.applicableHooks.length} onClick={() => void saveCommand()}>保存命令</Button></DialogActions>
		</Dialog>

		<Dialog open={Boolean(chainDraft)} onClose={() => setChainDraft(undefined)} fullWidth maxWidth="sm">
			<DialogTitle>{chainDraft?.id ? '编辑命令链' : '新增命令链'}</DialogTitle>
			<DialogContent sx={{display: 'grid', gap: 1.5, pt: '12px !important'}}>
				<TextField autoFocus required label="命令链名称" value={chainDraft?.name ?? ''} onChange={(event) => setChainDraft((current) => current ? {...current, name: event.target.value} : current)}/>
				<Box sx={{display: 'grid', gap: 0.5}}><Typography variant="subtitle2">适用范围</Typography>{lifecycleHooks.map((hook) => <FormControlLabel key={hook.id} control={<Checkbox checked={Boolean(chainDraft?.applicableHooks.includes(hook.id))} onChange={(event) => toggleApplicableHook(hook.id, 'chain', event.target.checked)}/>} label={hook.label}/>)}</Box>
				<Box sx={{display: 'grid', gap: 0.5}}><Typography variant="subtitle2">可用命令</Typography>{chainDraft?.applicableHooks.length ? commands.filter((command) => lifecycleHooksCover(command.applicableHooks, chainDraft.applicableHooks) || chainDraft.commands.some((reference) => reference.commandId === command.id)).map((command) => <FormControlLabel key={command.id} control={<Checkbox checked={Boolean(chainDraft.commands.some((reference) => reference.commandId === command.id))} onChange={(event) => toggleChainCommand(command.id, event.target.checked)}/>} label={`${command.name}${lifecycleHooksCover(command.applicableHooks, chainDraft.applicableHooks) ? '' : '（与当前范围不匹配）'}`}/>) : <Typography variant="body2" color="text.secondary">请先选择命令链适用范围。</Typography>}</Box>
				{incompatibleChainCommands.length > 0 && <Typography variant="body2" color="error">请移除与当前范围不匹配的命令后再保存。</Typography>}
				<Box sx={{display: 'grid', gap: 0.5}}>
					<Typography variant="subtitle2">执行顺序</Typography>
					{chainDraft?.commands.length ? chainDraft.commands.map((reference, index) => {
						const command = commandsByID.get(reference.commandId)
						return (
						<Box
							key={`${reference.commandId}-${index}`}
							draggable
							onDragStart={(event) => event.dataTransfer.setData('text/plain', String(index))}
							onDragOver={(event) => event.preventDefault()}
							onDrop={(event) => {
								const fromIndex = Number(event.dataTransfer.getData('text/plain'))
								moveChainCommandTo(fromIndex, index)
							}}
							sx={{display: 'grid', gridTemplateColumns: 'minmax(0, 1fr) auto auto', gap: 0.5, alignItems: 'center', border: 1, borderColor: 'divider', borderRadius: 1, p: 1, cursor: 'grab', '&:active': {cursor: 'grabbing'}}}
						>
							<Typography variant="body2">{index + 1}. {commandNames.get(reference.commandId) ?? '已删除命令'}</Typography>
							<IconButton aria-label={`上移链命令 ${index + 1}`} size="small" disabled={index === 0} onClick={() => moveChainCommand(index, -1)}><ArrowUpwardOutlinedIcon fontSize="inherit"/></IconButton>
							<IconButton aria-label={`下移链命令 ${index + 1}`} size="small" disabled={index === chainDraft.commands.length - 1} onClick={() => moveChainCommand(index, 1)}><ArrowDownwardOutlinedIcon fontSize="inherit"/></IconButton>
							{lifecycleCommandAllowsChainArguments(command) && <TextField fullWidth multiline minRows={2} label={`${command?.name ?? '命令'} 追加参数（每行一个）`} value={reference.arguments.join('\n')} onChange={(event) => updateChainCommandArguments(index, event.target.value.split('\n'))} sx={{gridColumn: '1 / -1', cursor: 'text'}}/>}
							{command?.documentation && <Typography variant="caption" color="text.secondary" sx={{gridColumn: '1 / -1', whiteSpace: 'pre-wrap'}}>{command.documentation}</Typography>}
						</Box>
						)
					}) : <Typography variant="body2" color="text.secondary">至少选择一个命令。</Typography>}
				</Box>
			</DialogContent>
			<DialogActions><Button onClick={() => setChainDraft(undefined)}>取消</Button><Button variant="contained" disabled={!chainDraft?.applicableHooks.length || !chainDraft.commands.length || incompatibleChainCommands.length > 0} onClick={() => void saveChain()}>保存命令链</Button></DialogActions>
		</Dialog>

		<Dialog open={chainHelpOpen} onClose={() => setChainHelpOpen(false)} aria-labelledby="lifecycle-chain-help-title" fullWidth maxWidth="md">
			<DialogTitle id="lifecycle-chain-help-title">命令链扩展规则</DialogTitle>
			<DialogContent dividers sx={{display: 'grid', gap: 2}}>
				<Box component="section" sx={{display: 'grid', gap: 0.5}}>
					<Typography variant="subtitle2">适用范围与顺序</Typography>
					<Typography variant="body2" color="text.secondary">命令链按配置顺序执行，可用于开始前、开始后、结束前、结束后和更新任务后五个钩子。链中的每条命令都必须覆盖该链选择的全部适用范围。</Typography>
				</Box>
				<Box component="section" sx={{display: 'grid', gap: 0.5}}>
					<Typography variant="subtitle2">参数传递</Typography>
					<Typography variant="body2" color="text.secondary">每行配置一个独立参数。固定参数会先于命令链追加参数传入。关闭“允许在命令链中追加参数”只会隐藏编辑框，不会删除已经保存的追加参数。</Typography>
				</Box>
				<Box component="section" sx={{display: 'grid', gap: 0.5}}>
					<Typography variant="subtitle2">标准输入与输出</Typography>
					<Typography variant="body2" color="text.secondary">首条自定义命令接收任务详情 JSON；本机 HTTP 服务正在监听时，顶层 <code>baseURL</code> 是当前 API 基址，只通过标准输入提供，不是终端环境变量。当前任务详情中的“复制当前命令链输入 JSON”可用于取得此刻的首段快照。后续自定义命令接收前一条自定义命令的原始标准输出，系统不会自动解析、合并或补回 JSON。</Typography>
				</Box>
				<Box component="section" sx={{display: 'grid', gap: 0.5}}>
					<Typography variant="subtitle2">环境变量</Typography>
					<Typography variant="body2" color="text.secondary">应用向任务关联的命令与脚本注入 <code>TASKAI_TASK_ID</code>；新建终端还会获得 <code>TASKAI_TERMINAL_ID</code>，本机 HTTP 服务正在监听时额外获得 <code>TASKAI_STATUS_API</code>。当前任务模板中勾选环境变量注入的字段仅传给自定义生命周期 Shell 命令。</Typography>
				</Box>
				<Box component="section" sx={{display: 'grid', gap: 0.5}}>
					<Typography variant="subtitle2">内置命令</Typography>
					<Typography variant="body2" color="text.secondary">创建和删除任务工作目录由应用直接执行。Git 仓库克隆的追加参数可留空，或仅填写一行 <code>dir=&lt;相对目录&gt;</code>；目录必须保持在任务工作目录内。</Typography>
				</Box>
				<Box component="section" sx={{display: 'grid', gap: 0.5}}>
					<Typography variant="subtitle2">失败与重试</Typography>
					<Typography variant="body2" color="text.secondary">链执行失败会保留错误和进度，重试始终从链首重新执行，并读取最新的链定义。</Typography>
				</Box>
			</DialogContent>
			<DialogActions><Button onClick={() => setChainHelpOpen(false)}>关闭</Button></DialogActions>
		</Dialog>
	</Box>
}

function TaskTemplateManagement({
	templates,
	activeTemplateID,
	onChange,
}: {
	templates: TaskTemplate[]
	activeTemplateID: string
	onChange(templates: TaskTemplate[], activeTemplateID: string): void
}) {
	const [draft, setDraft] = useState<TaskTemplate>()
	const [draftError, setDraftError] = useState<string>()

	const updateField = (index: number, update: Partial<TaskTemplateField>) => {
		setDraft((current) => current ? {
			...current,
			fields: current.fields.map((field, fieldIndex) => fieldIndex === index ? {...field, ...update} : field),
		} : current)
	}

	const moveField = (index: number, offset: number) => {
		setDraft((current) => {
			if (!current || index + offset < 0 || index + offset >= current.fields.length) {
				return current
			}
			const fields = [...current.fields]
			const [field] = fields.splice(index, 1)
			fields.splice(index + offset, 0, field)
			return {...current, fields}
		})
	}

	const saveDraft = () => {
		if (!draft) {
			return
		}
		const normalized: TaskTemplate = {
			...draft,
			name: draft.name.trim(),
			fields: draft.fields.map((field) => ({
				...field,
				key: field.key.trim(),
				displayName: field.displayName.trim(),
				defaultValue: field.inputType === 'bool' ? field.defaultValue === true : typeof field.defaultValue === 'string' ? field.defaultValue : '',
			})),
		}
		if (!normalized.name) {
			setDraftError('模板名称不能为空')
			return
		}
		if (templates.some((template) => template.id !== normalized.id && template.name.trim().toLocaleLowerCase() === normalized.name.toLocaleLowerCase())) {
			setDraftError('模板名称不能重复')
			return
		}
		if (!normalized.fields.length || normalized.fields.some((field) => !field.key || !field.displayName)) {
			setDraftError('每个字段都需要键和显示名称')
			return
		}
		if (normalized.fields.some((field) => !/^[A-Za-z][A-Za-z0-9_]*$/.test(field.key))) {
			setDraftError('字段键必须以 ASCII 字母开头，且只能包含字母、数字和下划线')
			return
		}
		const keys = normalized.fields.map((field) => field.key.toLocaleLowerCase())
		if (new Set(keys).size !== keys.length) {
			setDraftError('字段键不能重复')
			return
		}
		onChange(templates.some((template) => template.id === normalized.id)
			? templates.map((template) => template.id === normalized.id ? normalized : template)
			: [...templates, normalized], activeTemplateID)
		setDraft(undefined)
		setDraftError(undefined)
	}

	return <Box component="section" sx={{display: 'grid', gap: 1.5}}>
		<Box sx={{display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 2}}>
			<Box>
				<Typography variant="subtitle2">任务模板</Typography>
				<Typography variant="caption" color="text.secondary">当前模板决定新建、编辑、HTTP 和生命周期命令链可见的字段。</Typography>
			</Box>
			<Button variant="contained" size="small" startIcon={<AddOutlinedIcon/>} onClick={() => {
				setDraft(createTaskTemplateDraft())
				setDraftError(undefined)
			}}>新增模板</Button>
		</Box>
		<TextField select size="small" label="当前任务模板" value={activeTemplateID} onChange={(event) => onChange(templates, event.target.value)}>
			<MenuItem value="">不使用任务模板</MenuItem>
			{templates.map((template) => <MenuItem key={template.id} value={template.id}>{template.name}</MenuItem>)}
		</TextField>
		{templates.length === 0 ? <Alert severity="info" variant="outlined">暂无任务模板。新建模板后可选择为当前模板。</Alert> : <Box sx={{border: 1, borderColor: 'divider', borderRadius: 1, overflow: 'hidden'}}>
			{templates.map((template, index) => <Box key={template.id} sx={{display: 'flex', alignItems: 'center', gap: 1, px: 1.5, py: 1.25, borderBottom: index === templates.length - 1 ? 0 : 1, borderColor: 'divider'}}>
				<Box sx={{minWidth: 0, flex: 1}}>
					<Typography variant="body2" sx={{fontWeight: 650}} noWrap>{template.name}</Typography>
					<Typography variant="caption" color="text.secondary">{template.fields.length} 个字段{template.id === activeTemplateID ? ' · 当前使用' : ''}</Typography>
				</Box>
				<Button size="small" onClick={() => {
					setDraft(cloneTaskTemplate(template))
					setDraftError(undefined)
				}}>编辑</Button>
				<Tooltip title="删除模板"><IconButton aria-label={`删除任务模板 ${template.name}`} size="small" color="error" onClick={() => onChange(templates.filter((current) => current.id !== template.id), activeTemplateID === template.id ? '' : activeTemplateID)}><DeleteOutlineOutlinedIcon fontSize="inherit"/></IconButton></Tooltip>
			</Box>)}
		</Box>}

		<Dialog open={Boolean(draft)} onClose={() => setDraft(undefined)} fullWidth maxWidth="md" aria-labelledby="task-template-editor-title">
			<DialogTitle id="task-template-editor-title">{templates.some((template) => template.id === draft?.id) ? '编辑任务模板' : '新增任务模板'}</DialogTitle>
			<DialogContent sx={{display: 'grid', gap: 2, pt: '12px !important'}}>
				{draftError && <Alert severity="error">{draftError}</Alert>}
				<TextField required autoFocus size="small" label="模板名称" value={draft?.name ?? ''} onChange={(event) => setDraft((current) => current ? {...current, name: event.target.value} : current)}/>
				<Box sx={{display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 2}}>
					<Box>
						<Typography variant="subtitle2">字段</Typography>
						<Typography variant="caption" color="text.secondary">字段顺序决定任务表单、HTTP 字段和命令环境变量的显示顺序。</Typography>
					</Box>
					<Button size="small" startIcon={<AddOutlinedIcon/>} onClick={() => setDraft((current) => current ? {...current, fields: [...current.fields, createTaskTemplateField()]} : current)}>新增字段</Button>
				</Box>
				<Box sx={{display: 'grid', gap: 1}}>
					{draft?.fields.map((field, index) => <Box key={index} sx={{display: 'grid', gridTemplateColumns: {xs: '1fr auto', sm: 'minmax(110px, 1fr) minmax(110px, 1fr) 126px auto'}, gap: 1, alignItems: 'center', border: 1, borderColor: 'divider', borderRadius: 1, p: 1}}>
						<TextField required size="small" label={`字段 ${index + 1} 键`} value={field.key} onChange={(event) => updateField(index, {key: event.target.value})}/>
						<TextField required size="small" label={`字段 ${index + 1} 显示名称`} value={field.displayName} onChange={(event) => updateField(index, {displayName: event.target.value})}/>
						<TextField select size="small" label={`字段 ${index + 1} 类型`} value={field.inputType} onChange={(event) => {
							const inputType: TaskTemplateField['inputType'] = event.target.value === 'bool' ? 'bool' : 'string'
							updateField(index, {inputType, defaultValue: inputType === 'bool' ? false : ''})
						}}>
							<MenuItem value="string">字符串</MenuItem>
							<MenuItem value="bool">布尔值</MenuItem>
						</TextField>
						<Box sx={{display: 'flex', alignItems: 'center', gap: 0.25}}>
							<Tooltip title="上移字段"><span><IconButton aria-label={`上移模板字段 ${index + 1}`} size="small" disabled={index === 0} onClick={() => moveField(index, -1)}><ArrowUpwardOutlinedIcon fontSize="inherit"/></IconButton></span></Tooltip>
							<Tooltip title="下移字段"><span><IconButton aria-label={`下移模板字段 ${index + 1}`} size="small" disabled={index === (draft?.fields.length ?? 0) - 1} onClick={() => moveField(index, 1)}><ArrowDownwardOutlinedIcon fontSize="inherit"/></IconButton></span></Tooltip>
							<Tooltip title="删除字段"><span><IconButton aria-label={`删除模板字段 ${index + 1}`} size="small" color="error" disabled={(draft?.fields.length ?? 0) === 1} onClick={() => setDraft((current) => current ? {...current, fields: current.fields.filter((_, fieldIndex) => fieldIndex !== index)} : current)}><DeleteOutlineOutlinedIcon fontSize="inherit"/></IconButton></span></Tooltip>
						</Box>
						<Box sx={{gridColumn: '1 / -1', display: 'flex', flexWrap: 'wrap', alignItems: 'center', gap: 1.5}}>
							{field.inputType === 'bool'
								? <FormControlLabel sx={{m: 0}} control={<Checkbox checked={field.defaultValue === true} onChange={(event) => updateField(index, {defaultValue: event.target.checked})}/>} label="默认选中"/>
								: <TextField size="small" label={`字段 ${index + 1} 默认值`} value={typeof field.defaultValue === 'string' ? field.defaultValue : ''} onChange={(event) => updateField(index, {defaultValue: event.target.value})} sx={{minWidth: 220, flex: 1}}/>}
							<FormControlLabel sx={{m: 0}} control={<Checkbox checked={field.required} onChange={(event) => updateField(index, {required: event.target.checked})}/>} label={field.inputType === 'bool' ? '必填（必须勾选）' : '必填'}/>
							<FormControlLabel sx={{m: 0}} control={<Checkbox checked={field.injectEnvironment} onChange={(event) => updateField(index, {injectEnvironment: event.target.checked})}/>} label="注入生命周期环境变量"/>
						</Box>
					</Box>)}
				</Box>
			</DialogContent>
			<DialogActions><Button onClick={() => setDraft(undefined)}>取消</Button><Button variant="contained" onClick={saveDraft}>保存模板</Button></DialogActions>
		</Dialog>
	</Box>
}

function TaskTemplateFieldsEditor({
	template,
	values,
	onChange,
}: {
	template?: TaskTemplate
	values: TaskTemplateValues
	onChange: Dispatch<SetStateAction<TaskTemplateValues>>
}) {
	if (!template) {
		return null
	}
	return <Box component="section" data-testid="task-template-fields" sx={{display: 'grid', gap: 1.25, borderTop: 1, borderColor: 'divider', pt: 1.75}}>
		<Typography variant="subtitle2">{template.name}</Typography>
		<Box sx={{display: 'grid', gridTemplateColumns: {xs: '1fr', sm: '1fr 1fr'}, gap: 1.25}}>
			{template.fields.map((field) => field.inputType === 'bool'
				? <FormControlLabel key={field.key} sx={{m: 0, minWidth: 0}} control={<Checkbox checked={values[field.key] === true} onChange={(event) => onChange((current) => ({...current, [field.key]: event.target.checked}))}/>} label={field.displayName}/>
				: <TextField key={field.key} size="small" required={field.required} label={field.displayName} value={typeof values[field.key] === 'string' ? values[field.key] : ''} onChange={(event) => onChange((current) => ({...current, [field.key]: event.target.value}))}/>,
			)}
		</Box>
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

function TaskDetail({
	task,
	template,
	onCopyLifecycleCommandInput,
}: {
	task?: TaskRecord
	template?: TaskTemplate
	onCopyLifecycleCommandInput(taskID: string): void
}) {
  if (!task) {
    return (
      <Box className="taskai-detail-empty" sx={{height: '100%', display: 'grid', placeItems: 'center', color: 'text.secondary', textAlign: 'center', p: 3}}>
        <Box className="taskai-detail-empty__card">
          <FolderOutlinedIcon color="disabled" sx={{fontSize: 42, mb: 1}}/>
          <Typography>从左侧选择任务，或创建一个新任务开始。</Typography>
        </Box>
      </Box>
    )
  }
	const templateValues = resolveTaskTemplateValues(template, task.templateFields)
  return (
    <Box className="taskai-task-detail" sx={{height: '100%', width: '100%', overflow: 'auto', p: {xs: 2.5, md: 4}, maxWidth: 'none'}}>
      <Box className="taskai-task-detail__heading" sx={{display: 'flex', alignItems: 'center', gap: 1, mb: 2}}>
        <Typography variant="h5" sx={{fontWeight: 800}}>{task.title}</Typography>
        <Chip label={taskStatusLabel[task.status]} size="small" variant="outlined"/>
      </Box>
      <Typography variant="body1" sx={{whiteSpace: 'pre-wrap', color: task.description ? 'text.primary' : 'text.secondary', mb: 4}}>
        {task.description || '暂无任务描述'}
      </Typography>
		<Box className="taskai-detail-section" component="section" sx={{display: 'grid', gap: 1, mb: 3}}>
			<Typography className="taskai-detail-section__title" variant="overline" color="text.secondary">任务模板</Typography>
			{template ? <Box className="taskai-detail-card" sx={{display: 'grid', gap: 1, border: 1, borderColor: 'divider', borderRadius: 1, p: 1.5}}>
				<Typography variant="subtitle2">{template.name}</Typography>
				{template.fields.map((field) => {
					const value = taskTemplateFieldDisplayValue(templateValues[field.key])
					const environment = field.injectEnvironment ? `TASKAI_${field.key.toUpperCase()}=${value}` : undefined
					return <Box key={field.key} sx={{display: 'grid', gap: 0.5, borderTop: 1, borderColor: 'divider', pt: 1}}>
						<TaskDetailValue label={field.displayName || field.key} value={value}/>
						<Typography variant="caption" color="text.secondary">字段键：<code>{field.key}</code></Typography>
						<Typography variant="caption" color="text.secondary">{environment ? <>生成环境变量 <code>{environment}</code></> : '不生成环境变量'}</Typography>
						{environment && <Typography variant="caption" color="text.secondary">仅自定义生命周期 Shell 命令</Typography>}
					</Box>
				})}
			</Box> : <Alert severity="info" variant="outlined">未启用任务模板</Alert>}
		</Box>
		<Box className="taskai-detail-section" component="section" sx={{display: 'grid', gap: 1, mb: 3}}>
			<Typography className="taskai-detail-section__title" variant="overline" color="text.secondary">额外信息</Typography>
			{task.extraInfo?.length ? Object.entries(groupTaskExtraInfoByCatalogue(task.extraInfo)).map(([catalogue, items]) => (
				<Box className="taskai-detail-card" key={catalogue} sx={{display: 'grid', gap: 1, border: 1, borderColor: 'divider', borderRadius: 1, p: 1.5}}>
					<Typography variant="subtitle2">{catalogue}</Typography>
					{items.map((item, itemIndex) => (
						<Box key={item.id || `${catalogue}-${itemIndex}`} sx={{display: 'grid', gap: 1, borderTop: itemIndex === 0 ? 0 : 1, borderColor: 'divider', pt: itemIndex === 0 ? 0 : 1.25}}>
							<Typography variant="body2" sx={{fontWeight: 700}}>{item.displayName || item.fields.find((field) => field.key === 'name')?.value || catalogue}</Typography>
							<Box sx={{display: 'grid', gridTemplateColumns: {xs: '1fr', sm: 'repeat(2, minmax(0, 1fr))'}, gap: 1}}>
								{item.fields.map((field) => <TaskDetailValue key={`field-${field.key}`} label={field.displayName || field.key} value={field.value || '—'}/>) }
								{item.parameters.map((parameter) => <TaskDetailValue key={`parameter-${parameter.key}`} label={parameter.displayName || parameter.key} value={extraInfoParameterInputType(parameter) === 'checkbox' ? parameter.value === 'true' ? '是' : '否' : parameter.value || '—'}/>) }
							</Box>
						</Box>
					))}
				</Box>
			)) : <Alert severity="info" variant="outlined">未添加额外信息</Alert>}
		</Box>
		<Box className="taskai-detail-section" component="section" sx={{display: 'grid', gap: 1, mb: 3}}>
			<Typography className="taskai-detail-section__title" variant="overline" color="text.secondary">系统环境变量</Typography>
			<Box sx={{display: 'grid', gap: 0.75}}>
				<TaskDetailValue label="TASKAI_TASK_ID" value="任务关联的自定义命令和前后置脚本"/>
				<TaskDetailValue label="TASKAI_TERMINAL_ID" value="仅新建的普通终端和显示终端的自定义命令"/>
				<TaskDetailValue label="TASKAI_STATUS_API" value="本机 HTTP 服务正在监听时，注入到之后新建的终端"/>
			</Box>
		</Box>
		<Button variant="outlined" size="small" sx={{mb: 3, alignSelf: 'start'}} onClick={() => onCopyLifecycleCommandInput(task.id)}>复制当前命令链输入 JSON</Button>
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

function groupTaskExtraInfoByCatalogue(items: TaskExtraInfo[]): Record<string, TaskExtraInfo[]> {
	return items.reduce<Record<string, TaskExtraInfo[]>>((grouped, item) => {
		const catalogue = item.catalogue || '未分类'
		grouped[catalogue] = [...(grouped[catalogue] ?? []), item]
		return grouped
	}, {})
}

function TaskDetailValue({label, value}: {label: string, value: string}) {
	return <Box className="taskai-detail-value" sx={{display: 'grid', gap: 0.25, minWidth: 0}}>
		<Typography variant="caption" color="text.secondary" sx={{overflowWrap: 'anywhere'}}>{label}</Typography>
		<Typography variant="body2" sx={{whiteSpace: 'pre-wrap', overflowWrap: 'anywhere'}}>{value}</Typography>
	</Box>
}

function taskTemplateFieldDisplayValue(value: string | boolean | undefined): string {
	if (typeof value === 'boolean') {
		return `${value}`
	}
	return value || '—'
}

function createExtraInfoTemplateDraft(catalogue: string): ExtraInfoTemplate {
	return {id: '', catalogue, displayName: '', builtIn: false, fields: [{key: 'name', displayName: '名称', defaultValue: ''}], parameters: []}
}

function createTaskTemplateDraft(): TaskTemplate {
	return {id: `task-template-${Date.now()}-${Math.random().toString(36).slice(2, 10)}`, name: '', fields: [createTaskTemplateField()]}
}

function createTaskTemplateField(): TaskTemplateField {
	return {key: '', displayName: '', inputType: 'string', required: false, defaultValue: '', injectEnvironment: false}
}

function cloneTaskTemplate(template: TaskTemplate): TaskTemplate {
	return {...template, fields: template.fields.map((field) => ({...field}))}
}

function resolveTaskTemplateValues(template?: TaskTemplate, existing?: TaskTemplateValues): TaskTemplateValues {
	if (!template) {
		return {}
	}
	return Object.fromEntries(template.fields.map((field) => {
		const saved = existing?.[field.key]
		if (field.inputType === 'bool') {
			return [field.key, typeof saved === 'boolean' ? saved : field.defaultValue === true]
		}
		return [field.key, typeof saved === 'string' ? saved : typeof field.defaultValue === 'string' ? field.defaultValue : '']
	}))
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

function showError(error: unknown, setMessage: Dispatch<SetStateAction<Notification | undefined>>) {
	setMessage({text: error instanceof Error ? error.message : String(error), severity: 'error'})
}
