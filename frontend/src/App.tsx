import {type Dispatch, type FormEvent, type ReactNode, type SetStateAction, useEffect, useMemo, useRef, useState} from 'react'

import {ArrowDown, ArrowUp, CheckCheck, CheckCircle2, FolderOpen, HelpCircle, Maximize, Minimize, Plus, RotateCcw, Settings as SettingsIcon, Trash2} from 'lucide-react'

import {api} from './api'
import taskAiMark from './assets/task-ai-mark.svg'
import {
  cn,
  Button as SnapButton,
  IconButton as SnapIconButton,
  Chip as SnapChip,
  Alert as SnapAlert,
  Input,
  Textarea,
  Label,
  Field,
  Dialog as SnapDialog,
  DialogContent as SnapDialogContent,
  DialogHeader as SnapDialogHeader,
  DialogFooter as SnapDialogFooter,
  DialogTitle as SnapDialogTitle,
  DialogDescription as SnapDialogDescription,
  Tabs as SnapTabs,
  TabsList as SnapTabsList,
  TabsTrigger as SnapTabsTrigger,
  TabsContent as SnapTabsContent,
  Accordion as SnapAccordion,
  AccordionItem as SnapAccordionItem,
  AccordionTrigger as SnapAccordionTrigger,
  AccordionContent as SnapAccordionContent,
  Tooltip as SnapTooltip,
  TooltipContent as SnapTooltipContent,
  TooltipProvider as SnapTooltipProvider,
  TooltipTrigger as SnapTooltipTrigger,
  Popover as SnapPopover,
  PopoverTrigger as SnapPopoverTrigger,
  PopoverContent as SnapPopoverContent,
  PopoverAnchor as SnapPopoverAnchor,
  Switch as SnapSwitch,
  Checkbox as SnapCheckbox,
  ScrollArea as SnapScrollArea,
  ScrollBar as SnapScrollBar,
  ToastProvider as SnapToastProvider,
  ToastViewport as SnapToastViewport,
  Toast as SnapToast,
  ToastTitle as SnapToastTitle,
  ToastDescription as SnapToastDescription,
  ToastClose as SnapToastClose,
} from './components/ui'
import {TaskTree, type TaskStartFeedback} from './components/TaskTree'
import {TerminalShortcutSettings} from './components/TerminalShortcutSettings'
import {TerminalView} from './components/TerminalView'
import {TerminalSessionRegistry} from './terminal-session'
import {uniqueProgramNames} from './terminal-shortcuts'
import {resolveTerminalFontFamily} from './terminal-font'
import {defaultTerminalFontSize, maximumTerminalFontSize, minimumTerminalFontSize, normalizeTerminalFontSize, terminalFontSizePercent} from './terminal-font-size'
import {defaultTerminalTheme, normalizeTerminalTheme, type TerminalTheme} from './terminal-theme'
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
	type LifecyclePreset,
	lifecycleHooks,
	taskStatusLabel,
  type ColorScheme,
	type ExtraInfoTemplate,
	type QuickInput,
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
	type TerminalFontCandidate,
} from './types'
import {ClipboardSetText} from '../wailsjs/runtime/runtime'
import './App.css'

type Notification = {
	text: string
	severity: 'success' | 'error'
}

const customLifecyclePresetID = '__custom__'

const selectClass = 'w-full border-2 border-snap-outline rounded-snap bg-snap-surface px-3 py-2 text-sm text-snap-ink shadow-snap-sm outline-none transition-[box-shadow,border-color] focus-visible:border-snap-cobalt focus-visible:ring-[3px] focus-visible:ring-snap-cobalt disabled:cursor-not-allowed disabled:opacity-60'
const defaultTerminalFontCandidate: TerminalFontCandidate = {family: '', spacing: 'mono'}
const terminalBasicColorFields: Array<{key: Exclude<keyof TerminalTheme, 'selectionBackground'>, label: string}> = [
	{key: 'background', label: '背景'},
	{key: 'foreground', label: '前景'},
	{key: 'cursor', label: '光标'},
	{key: 'cursorAccent', label: '光标反差'},
	{key: 'selectionForeground', label: '选区前景'},
]
const terminalAnsiColorFields: Array<{key: keyof TerminalTheme, label: string}> = [
	{key: 'black', label: '黑'},
	{key: 'red', label: '红'},
	{key: 'green', label: '绿'},
	{key: 'yellow', label: '黄'},
	{key: 'blue', label: '蓝'},
	{key: 'magenta', label: '洋红'},
	{key: 'cyan', label: '青'},
	{key: 'white', label: '白'},
]
const terminalBrightAnsiColorFields: Array<{key: keyof TerminalTheme, label: string}> = [
	{key: 'brightBlack', label: '亮黑'},
	{key: 'brightRed', label: '亮红'},
	{key: 'brightGreen', label: '亮绿'},
	{key: 'brightYellow', label: '亮黄'},
	{key: 'brightBlue', label: '亮蓝'},
	{key: 'brightMagenta', label: '亮洋红'},
	{key: 'brightCyan', label: '亮青'},
	{key: 'brightWhite', label: '亮白'},
]

export default function App() {
  const [tasks, setTasks] = useState<TaskRecord[]>([])
  const [terminals, setTerminals] = useState<TerminalRecord[]>([])
	const [settings, setSettings] = useState<SettingsRecord>()
	const [extraInfoTemplates, setExtraInfoTemplates] = useState<ExtraInfoTemplate[]>([])
	const [extraInfos, setExtraInfos] = useState<ExtraInfo[]>([])
	const [quickInputs, setQuickInputs] = useState<QuickInput[]>([])
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
	const [taskLifecyclePresetID, setTaskLifecyclePresetID] = useState('')
	const [taskLifecycleSelectionExplicit, setTaskLifecycleSelectionExplicit] = useState(false)
  const [editingTask, setEditingTask] = useState<TaskRecord>()
  const [settingsDialogOpen, setSettingsDialogOpen] = useState(false)
	const [extraInfoManagerOpen, setExtraInfoManagerOpen] = useState(false)
	const [informationManagerTab, setInformationManagerTab] = useState<'extra-info' | 'quick-input'>('extra-info')
	const [extraInfoTemplateDraft, setExtraInfoTemplateDraft] = useState<ExtraInfoTemplate>()
	const [extraInfoDraft, setExtraInfoDraft] = useState<ExtraInfo>()
	const [extraInfoEditorOpen, setExtraInfoEditorOpen] = useState(false)
	const [newExtraInfoTemplateID, setNewExtraInfoTemplateID] = useState('')
	const [lastCreatedExtraInfoTemplateID, setLastCreatedExtraInfoTemplateID] = useState('')
	const [extraInfoSearch, setExtraInfoSearch] = useState('')
	const [templateSectionExpanded, setTemplateSectionExpanded] = useState(false)
	const [expandedExtraInfoTemplateIDs, setExpandedExtraInfoTemplateIDs] = useState<string[]>([])
	const [quickInputSearch, setQuickInputSearch] = useState('')
	const [quickInputDraft, setQuickInputDraft] = useState<QuickInput>()
	const [quickInputDeletion, setQuickInputDeletion] = useState<QuickInput>()
  const [finishTask, setFinishTask] = useState<TaskRecord>()
	const [taskDeletionSelectionMode, setTaskDeletionSelectionMode] = useState(false)
	const [selectedTaskIDs, setSelectedTaskIDs] = useState<string[]>([])
	const [taskDeletionOpen, setTaskDeletionOpen] = useState(false)
  const [quitDialogOpen, setQuitDialogOpen] = useState(false)
  const [draftTitle, setDraftTitle] = useState('')
  const [draftDescription, setDraftDescription] = useState('')
  const [draftColor, setDraftColor] = useState(defaultTaskColor)
  const [settingsDraft, setSettingsDraft] = useState<SettingsRecord>()
	const [terminalFontCandidates, setTerminalFontCandidates] = useState<TerminalFontCandidate[]>([defaultTerminalFontCandidate])
	const [terminalFontsLoading, setTerminalFontsLoading] = useState(false)
	const [terminalFontsError, setTerminalFontsError] = useState(false)
	const [terminalFontPickerExpanded, setTerminalFontPickerExpanded] = useState(false)
	const [terminalThemePickerExpanded, setTerminalThemePickerExpanded] = useState(false)
	const fontOptions = useMemo(() => {
		const selectedFamily = settingsDraft?.terminalFontFamily?.trim() ?? ''
		if (!selectedFamily || terminalFontCandidates.some((candidate) => candidate.family === selectedFamily)) {
			return terminalFontCandidates
		}
		return [...terminalFontCandidates, {family: selectedFamily, spacing: 'unavailable' as const}]
	}, [settingsDraft?.terminalFontFamily, terminalFontCandidates])
	const selectedTerminalFont = fontOptions.find((candidate) => candidate.family === (settingsDraft?.terminalFontFamily?.trim() ?? '')) ?? defaultTerminalFontCandidate
	const terminalFontSizeDraft = normalizeTerminalFontSize(settingsDraft?.terminalFontSize)
	const terminalThemeDraft = normalizeTerminalTheme(settingsDraft?.terminalTheme)
	const terminalSelectionOpacity = terminalThemeSelectionOpacity(terminalThemeDraft)
	const [settingsTab, setSettingsTab] = useState<'workspace' | 'shell' | 'shortcuts' | 'menu' | 'status' | 'lifecycle' | 'templates'>('workspace')
  const [statusHelpOpen, setStatusHelpOpen] = useState(false)
  const [taskMenuItemDraft, setTaskMenuItemDraft] = useState<TaskMenuItem>()
  const [taskMenuItemEditorMode, setTaskMenuItemEditorMode] = useState<'create' | 'edit'>()
  const [taskMenuItemEditorTab, setTaskMenuItemEditorTab] = useState<'basic' | 'scripts'>('basic')
	const taskMenuItemIsCommand = taskMenuItemDraft?.kind === 'command'
	const taskMenuItemIsShelvingToggle = taskMenuItemDraft?.kind === 'toggle-shelved'
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
	const terminalFontFamily = useRef('')
	const terminalFontSize = useRef(defaultTerminalFontSize)
	const terminalTheme = useRef<TerminalTheme>(defaultTerminalTheme)
	const terminalSessions = useRef<TerminalSessionRegistry>()
	if (!terminalSessions.current) {
		terminalSessions.current = new TerminalSessionRegistry((taskID, terminalID, data) => {
			void api.writeTerminal(taskID, terminalID, data).catch((error) => showError(error, setMessage))
		}, () => terminalFontFamily.current, () => terminalFontSize.current, () => terminalTheme.current)
	}
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
		const [loadedTasks, loadedSettings, loadedShells, loadedExtraInfoTemplates, loadedExtraInfos, loadedQuickInputs] = await Promise.all([api.listTasks(), api.getSettings(), api.detectShells(), api.listExtraInfoTemplates(), api.listExtraInfos(), api.listQuickInputs()])
        setTasks(loadedTasks)
		const loadedTerminalTheme = normalizeTerminalTheme(loadedSettings.terminalTheme)
		setSettings({...loadedSettings, terminalTheme: loadedTerminalTheme})
		terminalFontFamily.current = loadedSettings.terminalFontFamily ?? ''
		terminalFontSize.current = normalizeTerminalFontSize(loadedSettings.terminalFontSize)
		terminalTheme.current = loadedTerminalTheme
        setActiveTaskStatus(loadedSettings.activeTaskStatus ?? 'pending')
        setDetectedShells(loadedShells)
		setExtraInfoTemplates(loadedExtraInfoTemplates)
		setExtraInfos(loadedExtraInfos)
		setQuickInputs(loadedQuickInputs)
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
      terminalSessions.current?.handleTerminalEvent(event)
      const title = parseTerminalEventTitle(terminalTitleParserStates.current, event)
      const key = terminalEventKey(event.taskId, event.terminalId)
      if (title !== undefined && shouldReportTerminalTitleActivity({title: terminalTitleValues.current.get(key)}, title)) {
        terminalTitleValues.current.set(key, title)
        void api.reportTerminalTitleActivity(event.taskId, event.terminalId).catch((error) => showError(error, setMessage))
      }
      if (!registeredTerminalKeys.current.has(key)) {
        bufferPendingTerminalEvent(pendingTerminalEvents.current, event, title)
      }
      if (event.type !== 'output' || title !== undefined) {
        setTerminals((current) => applyTerminalEvent(current, event, title))
      }
      if (event.type === 'error') {
        showErrorMessage(event.data || '终端发生错误')
      }
    })
    return () => {
      unsubscribe()
      terminalSessions.current?.disposeAll()
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

  useEffect(() => {
    const preventFileDropNavigation = (event: DragEvent) => {
      if (!Array.from(event.dataTransfer?.types ?? []).includes('Files')) {
        return
      }
      event.preventDefault()
    }
    document.addEventListener('dragover', preventFileDropNavigation)
    document.addEventListener('drop', preventFileDropNavigation)
    return () => {
      document.removeEventListener('dragover', preventFileDropNavigation)
      document.removeEventListener('drop', preventFileDropNavigation)
    }
  }, [])

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

  // 把 dark class 挂在 <html> 上（而非仅 .taskai-app）：Radix Dialog/Toast/Tooltip 与
  // 自绘 createPortal 菜单都挂载到 document.body，在 .taskai-app 之外；只有把暗色令牌
  // 提升到文档根，这些浮层与原生 <select> 下拉（依赖 color-scheme）才会跟随暗色模式。
  useEffect(() => {
    document.documentElement.classList.toggle('dark', colorScheme === 'dark')
  }, [colorScheme])

  const selectedTask = tasks.find((task) => task.id === selectedTaskID)
	const selectedTerminal = terminals.find((terminal) => terminal.id === selectedTerminalID && terminal.state === 'active')
	const taskMenuItems = settings?.taskMenuItems?.length ? settings.taskMenuItems : defaultTaskMenuItems
	const activeTaskTemplate = settings?.taskTemplates?.find((template) => template.id === settings.activeTaskTemplateId)
	const lifecyclePresets = settings?.lifecyclePresets ?? []
	const activeTaskTemplateIDs = useMemo(() => new Set(tasks
		.filter((task) => (task.status === 'pending' || task.status === 'running') && Boolean(task.taskTemplateId))
		.map((task) => task.taskTemplateId!)), [tasks])
	const hasLegacyActiveTaskTemplateFields = useMemo(() => tasks.some((task) =>
		(task.status === 'pending' || task.status === 'running')
		&& !task.taskTemplateId
		&& Object.keys(task.templateFields ?? {}).length > 0,
	), [tasks])
	const areAllTasksExpanded = tasks.length > 0 && tasks.every((task) => expandedTasks[task.id] ?? true)
	const canSelectTasksForDeletion = activeTaskStatus === 'pending' || activeTaskStatus === 'completed'
	const deletionStatusLabel = activeTaskStatus === 'pending' ? '未执行' : '已完成'
	const deletableTaskIDs = useMemo(() => tasks
		.filter((task) => canSelectTasksForDeletion && task.status === activeTaskStatus && !task.lifecycleExecution)
		.map((task) => task.id), [activeTaskStatus, canSelectTasksForDeletion, tasks])
	const deletableTaskIDSet = useMemo(() => new Set(deletableTaskIDs), [deletableTaskIDs])
	const selectedDeletableTaskIDs = useMemo(() => selectedTaskIDs
		.filter((taskID) => deletableTaskIDSet.has(taskID)), [deletableTaskIDSet, selectedTaskIDs])

	useEffect(() => {
		setSelectedTaskIDs((current) => {
			const next = current.filter((taskID) => deletableTaskIDSet.has(taskID))
			return next.length === current.length ? current : next
		})
	}, [deletableTaskIDSet])

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
      <StartupScreen/>
    )
  }

  const openTaskDialog = (task?: TaskRecord) => {
    setEditingTask(task)
    setDraftTitle(task?.title ?? '')
    setDraftDescription(task?.description ?? '')
    setDraftColor(task ? task.color || defaultTaskColor : randomTaskColor())
		setTaskExtraInfoDraft(task?.extraInfo ? cloneTaskExtraInfo(task.extraInfo) : [])
		setTaskTemplateFieldsDraft(resolveTaskTemplateValues(activeTaskTemplate, task?.templateFields))
		const defaultPresetID = settings?.defaultLifecyclePresetId ?? ''
		const defaultPreset = lifecyclePresets.find((preset) => preset.id === defaultPresetID)
		const lifecycleChains = {...task?.lifecycleChains ?? defaultPreset?.chains ?? {}}
		setTaskLifecycleChainsDraft(lifecycleChains)
		setTaskLifecyclePresetID(task ? lifecyclePresetIDForChains(lifecyclePresets, lifecycleChains) ?? customLifecyclePresetID : defaultPreset?.id ?? '')
		setTaskLifecycleSelectionExplicit(Boolean(task) || Boolean(defaultPreset))
    setTaskDialogOpen(true)
  }

  const closeTaskDialog = () => {
    setTaskDialogOpen(false)
    setEditingTask(undefined)
		setTaskExtraInfoDraft([])
		setTaskTemplateFieldsDraft({})
		setTaskLifecycleChainsDraft({})
		setTaskLifecyclePresetID('')
		setTaskLifecycleSelectionExplicit(false)
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
			const created = taskLifecycleSelectionExplicit
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
		setInformationManagerTab('extra-info')
		try {
			const [templates, infos, inputs] = await Promise.all([api.listExtraInfoTemplates(), api.listExtraInfos(), api.listQuickInputs()])
			setExtraInfoTemplates(templates)
			setExtraInfos(infos)
			setQuickInputs(inputs)
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

	const updateExtraInfoDraftField = (key: string, value: string) => {
		setExtraInfoDraft((current) => {
			if (!current) {
				return current
			}
			const name = extraInfoName(current)
			const inferredName = current.catalogue === 'git' && key === 'repository' && !name.trim()
				? gitRepositoryName(value)
				: ''
			return {
				...current,
				fields: current.fields.map((field) => {
					if (field.key === key) {
						return {...field, value}
					}
					if (inferredName && field.key === 'name') {
						return {...field, value: inferredName}
					}
					return field
				}),
			}
		})
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

	const openQuickInputEditor = (input?: QuickInput) => {
		setQuickInputDraft(input ? {...input} : {id: '', name: '', content: ''})
	}

	const closeQuickInputEditor = () => setQuickInputDraft(undefined)

	const saveQuickInput = async () => {
		if (!quickInputDraft) {
			return
		}
		const name = quickInputDraft.name.trim()
		if (!name) {
			showErrorMessage('快捷输入名称不能为空')
			return
		}
		if ([...name].length > 100) {
			showErrorMessage('快捷输入名称不能超过 100 个字符')
			return
		}
		if (!quickInputDraft.content.trim()) {
			showErrorMessage('快捷输入内容必须包含非空白字符')
			return
		}
		try {
			const saved = await api.saveQuickInput({...quickInputDraft, name})
			setQuickInputs((current) => current.some((input) => input.id === saved.id)
				? current.map((input) => input.id === saved.id ? saved : input)
				: [...current, saved])
			closeQuickInputEditor()
		} catch (error) {
			showError(error, setMessage)
		}
	}

	const confirmDeleteQuickInput = async () => {
		if (!quickInputDeletion) {
			return
		}
		try {
			await api.deleteQuickInput(quickInputDeletion.id)
			setQuickInputs((current) => current.filter((input) => input.id !== quickInputDeletion.id))
			setQuickInputDeletion(undefined)
		} catch (error) {
			showError(error, setMessage)
		}
	}

	const moveQuickInput = async (inputID: string, direction: -1 | 1) => {
		const currentIndex = quickInputs.findIndex((input) => input.id === inputID)
		const targetIndex = currentIndex + direction
		if (currentIndex < 0 || targetIndex < 0 || targetIndex >= quickInputs.length) {
			return
		}
		const orderedInputIDs = quickInputs.map((input) => input.id)
		const [movedID] = orderedInputIDs.splice(currentIndex, 1)
		orderedInputIDs.splice(targetIndex, 0, movedID)
		try {
			setQuickInputs(await api.reorderQuickInputs(orderedInputIDs))
		} catch (error) {
			showError(error, setMessage)
			try {
				setQuickInputs(await api.listQuickInputs())
			} catch (reloadError) {
				showError(reloadError, setMessage)
			}
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
      terminalSessions.current?.disposeTask(finishTask.id)
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
    terminalSessions.current?.dispose(terminal.taskId, terminal.id)
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

	const exitTaskDeletionSelectionMode = () => {
		setTaskDeletionSelectionMode(false)
		setSelectedTaskIDs([])
		setTaskDeletionOpen(false)
	}

	const toggleTaskDeletion = (taskID: string) => {
		if (!deletableTaskIDSet.has(taskID)) {
			return
		}
		setSelectedTaskIDs((current) => current.includes(taskID)
			? current.filter((currentTaskID) => currentTaskID !== taskID)
			: [...current, taskID])
	}

	const confirmDeleteTasks = async () => {
		if (selectedDeletableTaskIDs.length === 0) {
			setTaskDeletionOpen(false)
			return
		}
		try {
			const remainingTasks = await api.deleteTasks(selectedDeletableTaskIDs)
			const deletedTaskIDSet = new Set(selectedDeletableTaskIDs)
			setTasks(remainingTasks)
				setTerminals((current) => current.filter((terminal) => !deletedTaskIDSet.has(terminal.taskId)))
				setExpandedTasks((current) => Object.fromEntries(Object.entries(current).filter(([taskID]) => !deletedTaskIDSet.has(taskID))))
				for (const taskID of deletedTaskIDSet) {
					terminalSessions.current?.disposeTask(taskID)
					finishedTerminalTaskIDs.current.delete(taskID)
				lifecycleStatusTargets.current.delete(taskID)
				clearTaskTerminalTracking(taskID, terminalTitleParserStates.current, pendingTerminalEvents.current, registeredTerminalKeys.current)
			}
			for (const terminal of terminals) {
				if (deletedTaskIDSet.has(terminal.taskId)) {
					terminalTitleValues.current.delete(terminalEventKey(terminal.taskId, terminal.id))
				}
			}
			if (selectedTaskID && deletedTaskIDSet.has(selectedTaskID)) {
				setSelectedTaskID(undefined)
				setSelectedTerminalID(undefined)
			} else if (selectedTerminalID && terminals.some((terminal) => terminal.id === selectedTerminalID && deletedTaskIDSet.has(terminal.taskId))) {
				setSelectedTerminalID(undefined)
			}
			exitTaskDeletionSelectionMode()
		} catch (error) {
			setTaskDeletionOpen(false)
			showError(error, setMessage)
		}
	}

  const saveSettings = async () => {
    if (!settingsDraft) {
      return
    }
    try {
		const saved = await api.saveSettings(settingsDraft)
		const savedTerminalTheme = normalizeTerminalTheme(saved.terminalTheme)
		setSettings({...saved, terminalTheme: savedTerminalTheme})
		terminalFontFamily.current = saved.terminalFontFamily ?? ''
		terminalFontSize.current = normalizeTerminalFontSize(saved.terminalFontSize)
		terminalTheme.current = savedTerminalTheme
		terminalSessions.current?.setAppearance(terminalFontFamily.current, terminalFontSize.current, terminalTheme.current)
      setSettingsDialogOpen(false)
      closeTaskMenuItemEditor()
    } catch (error) {
      showError(error, setMessage)
    }
  }

  const changeActiveTaskStatus = async (status: TaskStatus) => {
    const previousStatus = activeTaskStatus
		if (taskDeletionSelectionMode && status !== activeTaskStatus) {
			exitTaskDeletionSelectionMode()
		}
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
		setTerminalFontPickerExpanded(false)
		setTerminalThemePickerExpanded(false)
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
				lifecyclePresets: refreshed.lifecyclePresets ?? [],
				defaultLifecyclePresetId: refreshed.defaultLifecyclePresetId ?? '',
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
    const item = taskMenuItemDraft.kind === 'command'
      ? {
        ...taskMenuItemDraft,
        name: taskMenuItemDraft.name.trim(),
        command: taskMenuItemDraft.command?.trim(),
        arguments: taskMenuItemDraft.arguments?.filter((argument) => argument.trim()),
        disableTaskAIMouseClipboard: taskMenuItemDraft.showTerminal && taskMenuItemDraft.disableTaskAIMouseClipboard === true,
        beforeScript: normalizeTaskScript(taskMenuItemDraft.beforeScript),
        afterScript: normalizeTaskScript(taskMenuItemDraft.afterScript),
      }
      : {
        id: taskMenuItemDraft.id,
        kind: taskMenuItemDraft.kind,
        name: taskMenuItemDraft.name.trim(),
        unshelveName: taskMenuItemDraft.kind === 'toggle-shelved' ? taskMenuItemDraft.unshelveName?.trim() : undefined,
        showTerminal: false,
      }
    if (!item.name || (item.kind === 'command' && !item.command) || (item.kind === 'toggle-shelved' && !item.unshelveName)) {
      showErrorMessage(item.kind === 'command' ? '菜单名称和启动命令不能为空' : '菜单名称不能为空')
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
		terminalFontFamily: current?.terminalFontFamily ?? settings?.terminalFontFamily ?? '',
		terminalFontSize: normalizeTerminalFontSize(current?.terminalFontSize ?? settings?.terminalFontSize),
		terminalTheme: normalizeTerminalTheme(current?.terminalTheme ?? settings?.terminalTheme),
		terminalShortcuts: current?.terminalShortcuts ?? settings?.terminalShortcuts ?? [],
      shellPath: current?.shellPath ?? settings?.shellPath ?? detectedShells[0] ?? '',
      taskMenuItems: current?.taskMenuItems ?? cloneTaskMenuItems(taskMenuItems),
		activeTaskStatus: current?.activeTaskStatus ?? settings?.activeTaskStatus ?? activeTaskStatus,
		statusManagementMode: current?.statusManagementMode ?? settings?.statusManagementMode ?? 'title-change',
		statusManagementHTTPPort: current?.statusManagementHTTPPort ?? settings?.statusManagementHTTPPort ?? 0,
		httpServiceEnabled: current?.httpServiceEnabled ?? settings?.httpServiceEnabled ?? false,
		taskTemplates: current?.taskTemplates ?? settings?.taskTemplates,
		activeTaskTemplateId: current?.activeTaskTemplateId ?? settings?.activeTaskTemplateId,
      ...update,
    }))
  }

	const loadTerminalFonts = async () => {
		setTerminalFontsLoading(true)
		setTerminalFontsError(false)
		try {
			setTerminalFontCandidates(normalizeTerminalFontCandidates(await api.listTerminalFonts()))
		} catch {
			setTerminalFontCandidates([defaultTerminalFontCandidate])
			setTerminalFontsError(true)
		} finally {
			setTerminalFontsLoading(false)
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
	const statusManagementDescription = statusManagementMode === 'output-change'
		? '任意非空终端输出会在 1.5 秒内显示为工作中，未选中的终端静默后显示为未读。'
		: statusManagementMode === 'http'
			? '状态由 HTTP 接口更新；终端输出不会自动改变状态。'
			: '终端标题变化会在 1.5 秒内显示为工作中，未选中的终端随后显示为未读。'
	const extraInfoDraftTemplate = extraInfoDraft ? extraInfoTemplates.find((template) => template.id === extraInfoDraft.templateId) : undefined
	const normalizedQuickInputSearch = quickInputSearch.trim().toLocaleLowerCase()
	const visibleQuickInputs = quickInputs.filter((input) => !normalizedQuickInputSearch
		|| input.name.toLocaleLowerCase().includes(normalizedQuickInputSearch)
		|| input.content.toLocaleLowerCase().includes(normalizedQuickInputSearch))

  return (
    <SnapToastProvider duration={5000}>
      <SnapTooltipProvider delayDuration={300}>
      <div className="taskai-app" data-color-scheme={colorScheme} style={{height: '100vh', minWidth: 720, display: 'grid', gridTemplateRows: '52px minmax(0, 1fr)', overflow: 'hidden'}}>
        <header className="flex h-[52px] items-center gap-2 border-b-2 border-snap-outline bg-snap-surface px-3">
          <img className="taskai-brand-mark" src={taskAiMark} alt="任务 AI 图标"/>
          <span className="font-display text-base font-extrabold tracking-[0.3px] text-snap-ink">任务工作台</span>
          <span className="flex-1"/>
            <SnapTooltip>
              <SnapTooltipTrigger asChild>
				<SnapIconButton aria-label="信息管理" onClick={() => void openExtraInfoManager()}>
					<FolderOpen className="h-[18px] w-[18px]" strokeWidth={2.25}/>
                </SnapIconButton>
              </SnapTooltipTrigger>
				<SnapTooltipContent>信息管理</SnapTooltipContent>
            </SnapTooltip>
            <SnapTooltip>
              <SnapTooltipTrigger asChild>
              <SnapIconButton
                aria-label="设置"
                onClick={() => {
                  const draftMenuItems = cloneTaskMenuItems(taskMenuItems)
				  setSettingsDraft(settings ? {
				    ...settings,
                    colorScheme,
					terminalFontFamily: settings.terminalFontFamily ?? '',
						terminalFontSize: normalizeTerminalFontSize(settings.terminalFontSize),
						terminalTheme: normalizeTerminalTheme(settings.terminalTheme),
						terminalShortcuts: settings.terminalShortcuts ?? [],
                    shellPath: settings.shellPath || detectedShells[0] || '',
						taskMenuItems: draftMenuItems,
						statusManagementMode: settings.statusManagementMode ?? 'title-change',
						statusManagementHTTPPort: settings.statusManagementHTTPPort ?? 0,
						httpServiceEnabled: settings.httpServiceEnabled ?? false,
	                  } : undefined)
                  setTaskMenuItemDraft(undefined)
                  setTaskMenuItemEditorMode(undefined)
                  setSettingsTab('workspace')
				  setTerminalFontPickerExpanded(false)
					setStatusHelpOpen(false)
				  setTaskMenuItemEditorTab('basic')
				  setScriptHelpAnchor(undefined)
                  setSettingsDialogOpen(true)
					void loadTerminalFonts()
                }}
              >
                <SettingsIcon className="h-[18px] w-[18px]" strokeWidth={2.25}/>
              </SnapIconButton>
              </SnapTooltipTrigger>
              <SnapTooltipContent>设置</SnapTooltipContent>
            </SnapTooltip>
          </header>

        <div style={{display: 'grid', gridTemplateColumns: `${treeWidth}px 6px minmax(0, 1fr)`, minHeight: 0}}>
          <div className="taskai-sidebar-shell" style={{minWidth: 0, minHeight: 0, display: 'grid', gridTemplateRows: '40px minmax(0, 1fr)'}}>
            <div className="taskai-sidebar-header flex h-[40px] items-center gap-2 border-b-2 border-snap-outline px-[10px]">
              <span className="font-display text-xs font-extrabold uppercase tracking-wide text-snap-muted">任务与终端</span>
              <span className="flex-1"/>
								{canSelectTasksForDeletion && taskDeletionSelectionMode ? <>
						<span className="text-xs text-snap-muted tabular-nums whitespace-nowrap">已选 {selectedDeletableTaskIDs.length} 项</span>
						<SnapIconButton title="全选可删除任务" aria-label={`全选${deletionStatusLabel}任务`} size="sm" disabled={deletableTaskIDs.length === 0} onClick={() => setSelectedTaskIDs(deletableTaskIDs)}>
							<CheckCheck className="h-4 w-4" strokeWidth={2.25}/>
						</SnapIconButton>
						<SnapIconButton title="删除已选任务记录" aria-label="删除已选任务记录" size="sm" disabled={selectedDeletableTaskIDs.length === 0} onClick={() => setTaskDeletionOpen(true)} className="text-snap-error hover:text-snap-error">
							<Trash2 className="h-4 w-4" strokeWidth={2.25}/>
						</SnapIconButton>
						<SnapButton variant="ghost" size="sm" onClick={exitTaskDeletionSelectionMode}>取消</SnapButton>
					</> : <>
						{canSelectTasksForDeletion && <SnapIconButton title={`选择${deletionStatusLabel}任务`} aria-label={`选择${deletionStatusLabel}任务`} size="sm" onClick={() => setTaskDeletionSelectionMode(true)}>
							<CheckCheck className="h-4 w-4" strokeWidth={2.25}/>
						</SnapIconButton>}
						<SnapIconButton title={areAllTasksExpanded ? '收起全部任务' : '展开全部任务'} aria-label={areAllTasksExpanded ? '收起全部任务' : '展开全部任务'} disabled={tasks.length === 0} onClick={toggleAllTasksExpanded} size="sm">
							{areAllTasksExpanded ? <Minimize className="h-4 w-4" strokeWidth={2.25}/> : <Maximize className="h-4 w-4" strokeWidth={2.25}/>}
						</SnapIconButton>
						<SnapIconButton title="新建任务" aria-label="新建任务" onClick={() => openTaskDialog()} size="sm" className="text-snap-cobalt hover:text-snap-cobalt">
							<Plus className="h-4 w-4" strokeWidth={2.25}/>
						</SnapIconButton>
					</>}
            </div>
            <div style={{minHeight: 0}}>
              <TaskTree
                tasks={tasks}
                terminals={terminals}
                menuItems={taskMenuItems}
                activeStatus={activeTaskStatus}
					taskDeletionSelectionMode={taskDeletionSelectionMode}
					selectedTaskIDs={selectedDeletableTaskIDs}
                expandedTasks={expandedTasks}
                selectedTaskID={selectedTaskID}
                selectedTerminalId={selectedTerminalID}
                startedTaskFeedback={startedTaskFeedback}
                onChangeStatus={(status) => void changeActiveTaskStatus(status)}
					onToggleTaskDeletion={toggleTaskDeletion}
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
            </div>
          </div>
          <div
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
            className="taskai-panel-resizer w-[6px] cursor-col-resize bg-snap-outline/25 transition-colors hover:bg-snap-cobalt"
          />
          <div className="taskai-content-pane" style={{minWidth: 0, minHeight: 0}}>
            {selectedTerminal ? (
              <TerminalView
                key={selectedTerminal.id}
                terminal={selectedTerminal}
                sessionRegistry={terminalSessions.current!}
                quickInputs={quickInputs}
                fontSize={normalizeTerminalFontSize(settings?.terminalFontSize)}
				terminalTheme={settings?.terminalTheme}
				terminalShortcuts={settings?.terminalShortcuts}
                onResize={(columns, rows) => void api.resizeTerminal(selectedTerminal.taskId, selectedTerminal.id, columns, rows).catch((error) => showError(error, setMessage))}
                onError={(error) => showError(error, setMessage)}
                onClose={() => void closeTerminal(selectedTerminal)}
              />
            ) : (
              <TaskDetail
				task={selectedTask}
				template={activeTaskTemplate}
				onCopyLifecycleCommandInput={(taskID) => void copyLifecycleCommandInput(taskID).catch((error) => showError(error, setMessage))}
			/>
            )}
          </div>
        </div>
      </div>

      <SnapDialog open={taskDialogOpen} onOpenChange={(open) => { if (!open) closeTaskDialog() }}>
        <SnapDialogContent className="max-w-xl snap-no-scrollbar">
          <form onSubmit={saveTask} style={{display: 'flex', flexDirection: 'column', flex: '1 1 auto', minHeight: 0}}>
            <SnapDialogHeader>
              <SnapDialogTitle>{editingTask ? '编辑任务' : '新建任务'}</SnapDialogTitle>
            </SnapDialogHeader>
            <div data-testid="task-dialog-content" className="grid gap-4 px-1" style={{overflowY: 'auto', minHeight: 0}}>
                <Field label="标题"><Input autoFocus required value={draftTitle} onChange={(event) => setDraftTitle(event.target.value)}/></Field>
                <Field label="任务描述"><Textarea rows={3} value={draftDescription} onChange={(event) => setDraftDescription(event.target.value)}/></Field>
			<TaskTemplateFieldsEditor template={activeTaskTemplate} values={taskTemplateFieldsDraft} onChange={setTaskTemplateFieldsDraft}/>
                <div className="flex items-center gap-3">
                  <label htmlFor="task-color-picker" className="text-sm text-snap-ink">任务颜色</label>
                  <input
                    id="task-color-picker"
                    aria-label="任务颜色"
                    type="color"
                    value={draftColor}
                    onChange={(event) => setDraftColor(event.target.value)}
                    style={{width: 48, height: 36, padding: 2, border: 'none', background: 'transparent', cursor: 'pointer'}}
                  />
                </div>
			<TaskExtraInfoEditor
				templates={extraInfoTemplates}
				infos={extraInfos}
				extraInfo={taskExtraInfoDraft}
				onChange={setTaskExtraInfoDraft}
			/>
			<TaskLifecycleChainSelector
				chains={settings?.lifecycleChains ?? []}
				presets={lifecyclePresets}
				selected={taskLifecycleChainsDraft}
				presetID={taskLifecyclePresetID}
				onPresetChange={(presetID, chains) => {
					setTaskLifecyclePresetID(presetID)
					setTaskLifecycleChainsDraft(chains)
					setTaskLifecycleSelectionExplicit(true)
				}}
				onChange={(chains) => {
					setTaskLifecycleChainsDraft(chains)
					setTaskLifecyclePresetID(lifecyclePresetIDForChains(lifecyclePresets, chains) ?? customLifecyclePresetID)
					setTaskLifecycleSelectionExplicit(true)
				}}
				disabled={editingTask?.status !== undefined && editingTask.status !== 'pending'}
			/>
              </div>
            <SnapDialogFooter>
              <SnapButton variant="ghost" onClick={closeTaskDialog}>取消</SnapButton>
              <SnapButton type="submit" variant="primary">{editingTask ? '保存' : '创建'}</SnapButton>
            </SnapDialogFooter>
          </form>
        </SnapDialogContent>
      </SnapDialog>

		<SnapDialog open={extraInfoManagerOpen} onOpenChange={(open) => { if (!open) setExtraInfoManagerOpen(false) }}>
			<SnapDialogContent className="max-w-2xl" showClose={false} data-testid="extra-info-manager-content" style={{display: 'flex', flexDirection: 'column'}}>
				<SnapDialogHeader>
					<SnapDialogTitle>信息管理</SnapDialogTitle>
				</SnapDialogHeader>
				<SnapTabs value={informationManagerTab} onValueChange={(value) => setInformationManagerTab(value as 'extra-info' | 'quick-input')}>
					<SnapTabsList aria-label="信息管理标签">
						<SnapTabsTrigger value="extra-info">额外信息管理</SnapTabsTrigger>
						<SnapTabsTrigger value="quick-input">快捷输入管理</SnapTabsTrigger>
					</SnapTabsList>
					<SnapTabsContent value="extra-info">
				<div data-testid="extra-info-manager-scroll" className="max-h-[55vh] overflow-y-auto snap-no-scrollbar">
					<div className="grid gap-4 px-1">
						<div data-testid="extra-info-manager-actions" className="flex flex-wrap justify-end gap-2">
							<SnapButton variant="secondary" size="sm" onClick={() => openExtraInfoTemplateEditor()}>新增模板</SnapButton>
							<SnapButton variant="primary" size="sm" disabled={extraInfoTemplates.length === 0} onClick={() => openExtraInfoEditor()}>新增信息</SnapButton>
						</div>
						<div className="min-w-0">
							<SnapAccordion type="single" collapsible value={templateSectionExpanded ? 'templates' : ''} onValueChange={(value) => setTemplateSectionExpanded(value === 'templates')}>
								<SnapAccordionItem value="templates">
									<SnapAccordionTrigger aria-label="分类模板">
										<span className="min-w-0 pr-2">
											<span className="block font-display text-sm font-bold text-snap-ink">分类模板</span>
											<span className="block text-xs text-snap-muted">分类就是可填写的信息模板，定义固定字段、默认值和动态参数。</span>
										</span>
									</SnapAccordionTrigger>
									<SnapAccordionContent>
										<div className="overflow-hidden rounded-snap border-2 border-snap-outline divide-y-2 divide-snap-outline">
											{extraInfoTemplates.map((template) => (
												<div key={template.id} className="grid grid-cols-[minmax(0,1fr)] sm:grid-cols-[minmax(0,1fr)_auto] items-center gap-2 px-3 py-3">
													<div className="grid gap-1 min-w-0">
														<div className="flex flex-wrap items-center gap-1.5">
															<p className="text-sm font-bold text-snap-ink break-words m-0">{template.catalogue}</p>
															{template.builtIn && <SnapChip variant="info">内置 Git</SnapChip>}
														</div>
														<span className="text-xs text-snap-muted break-words">{template.fields.map((field) => `${field.displayName}${field.defaultValue ? `：${field.defaultValue}` : ''}`).join(' · ')}</span>
													</div>
													<div className="flex flex-wrap justify-start sm:justify-end items-center gap-1.5 min-w-0">
														<SnapChip>{`${template.fields.length} 固定 · ${template.parameters.length} 参数`}</SnapChip>
														<SnapButton variant="secondary" size="sm" onClick={() => openExtraInfoTemplateEditor(template)}>编辑</SnapButton>
														<SnapIconButton title={template.builtIn ? '内置 Git 模板不可删除' : '删除模板'} aria-label={`删除模板 ${template.catalogue}`} size="sm" disabled={template.builtIn} onClick={() => void deleteExtraInfoTemplate(template.id)} className="text-snap-error hover:text-snap-error">
															<Trash2 className="h-4 w-4" strokeWidth={2.25}/>
														</SnapIconButton>
													</div>
												</div>
											))}
										</div>
									</SnapAccordionContent>
								</SnapAccordionItem>
							</SnapAccordion>
						</div>
						<div className="mt-1">
							<span className="block font-display text-sm font-bold text-snap-ink">信息</span>
							<span className="block text-xs text-snap-muted">填写固定字段后保存为可复用信息，任务选择它时会生成独立快照。</span>
						</div>
						<Field label="搜索信息"><Input placeholder="按名称模糊搜索" value={extraInfoSearch} onChange={(event) => setExtraInfoSearch(event.target.value)}/></Field>
						{extraInfos.length === 0 ? <SnapAlert severity="info">暂无信息。选择一个模板后填写固定字段即可添加。</SnapAlert> : (
							<div className="grid gap-3">
								{extraInfoTemplates.map((template) => {
									const templateInfos = extraInfos.filter((info) => info.templateId === template.id)
									if (templateInfos.length === 0) {
										return null
									}
									const infos = templateInfos.filter((info) => extraInfoName(info).toLocaleLowerCase().includes(extraInfoSearch.trim().toLocaleLowerCase()))
									return <SnapAccordion key={template.id} type="single" collapsible value={expandedExtraInfoTemplateIDs.includes(template.id) ? template.id : undefined} onValueChange={(value) => setExpandedExtraInfoTemplateIDs((current) => value === template.id ? [...new Set([...current, template.id])] : current.filter((id) => id !== template.id))}>
										<SnapAccordionItem value={template.id}>
											<SnapAccordionTrigger aria-label={`信息分类 ${template.displayName || template.catalogue} ${template.catalogue}`}>
												<div className="flex items-center justify-between gap-2 w-full min-w-0 pr-2">
													<div className="min-w-0"><span className="block text-sm font-bold text-snap-ink truncate">{template.displayName || template.catalogue}</span><span className="block text-xs text-snap-muted">{template.catalogue}</span></div>
													<SnapChip>{`${infos.length}${extraInfoSearch.trim() ? ` / ${templateInfos.length}` : ''} 条信息`}</SnapChip>
												</div>
											</SnapAccordionTrigger>
											<SnapAccordionContent>
												<div className="grid gap-0">
													{infos.length === 0 ? <span className="block text-sm text-snap-muted px-1 pb-2">未找到匹配的信息。</span> : infos.map((info, index) => <div key={info.id} className={`flex items-center gap-2 px-1 py-2 ${index === 0 ? 'border-t-2' : ''} border-snap-outline`}>
														<div className="min-w-0 flex-1"><span className="block text-sm font-bold text-snap-ink truncate">{extraInfoName(info)}</span></div>
														<SnapButton variant="secondary" size="sm" onClick={() => openExtraInfoEditor(info)}>编辑</SnapButton>
														<SnapIconButton title={`删除信息 ${extraInfoName(info)}`} aria-label={`删除信息 ${extraInfoName(info)}`} size="sm" onClick={() => void deleteExtraInfo(info.id)} className="text-snap-error hover:text-snap-error"><Trash2 className="h-4 w-4" strokeWidth={2.25}/></SnapIconButton>
													</div>)}
												</div>
											</SnapAccordionContent>
										</SnapAccordionItem>
									</SnapAccordion>
								})}
							</div>
						)}
					</div>
				</div>
					</SnapTabsContent>
					<SnapTabsContent value="quick-input">
						<div data-testid="quick-input-manager-scroll" className="max-h-[55vh] overflow-y-auto snap-no-scrollbar">
							<div className="grid gap-4 px-1 py-1">
								<div className="flex flex-wrap justify-end gap-2">
									<SnapButton variant="primary" size="sm" onClick={() => openQuickInputEditor()}>新增快捷输入</SnapButton>
								</div>
								<Field label="搜索快捷输入"><Input placeholder="按名称或内容搜索" value={quickInputSearch} onChange={(event) => setQuickInputSearch(event.target.value)}/></Field>
								{quickInputs.length === 0 ? <SnapAlert severity="info">暂无快捷输入。</SnapAlert> : visibleQuickInputs.length === 0 ? <SnapAlert severity="info">未找到匹配的快捷输入。</SnapAlert> : (
									<div className="grid overflow-hidden rounded-snap border-2 border-snap-outline divide-y-2 divide-snap-outline">
										{visibleQuickInputs.map((input) => {
											const index = quickInputs.findIndex((current) => current.id === input.id)
											return <div key={input.id} className="grid gap-2 px-3 py-3 sm:grid-cols-[minmax(0,1fr)_auto] sm:items-center">
												<div className="grid min-w-0 gap-1">
													<span className="text-sm font-bold text-snap-ink break-words">{input.name}</span>
													<span className="font-mono text-xs text-snap-muted break-all" title={input.content}>{quickInputContentPreview(input.content)}</span>
												</div>
												<div className="flex flex-wrap items-center gap-1.5">
													{!normalizedQuickInputSearch && <>
														<SnapIconButton title="上移快捷输入" aria-label={`上移快捷输入 ${input.name}`} size="sm" disabled={index === 0} onClick={() => void moveQuickInput(input.id, -1)}><ArrowUp className="h-4 w-4" strokeWidth={2.25}/></SnapIconButton>
														<SnapIconButton title="下移快捷输入" aria-label={`下移快捷输入 ${input.name}`} size="sm" disabled={index === quickInputs.length - 1} onClick={() => void moveQuickInput(input.id, 1)}><ArrowDown className="h-4 w-4" strokeWidth={2.25}/></SnapIconButton>
													</>}
													<SnapButton variant="secondary" size="sm" onClick={() => openQuickInputEditor(input)}>编辑</SnapButton>
													<SnapIconButton title={`删除快捷输入 ${input.name}`} aria-label={`删除快捷输入 ${input.name}`} size="sm" onClick={() => setQuickInputDeletion(input)} className="text-snap-error hover:text-snap-error"><Trash2 className="h-4 w-4" strokeWidth={2.25}/></SnapIconButton>
												</div>
											</div>
										})}
									</div>
								)}
							</div>
						</div>
					</SnapTabsContent>
				</SnapTabs>
				<SnapDialogFooter>
					<SnapButton variant="secondary" onClick={() => setExtraInfoManagerOpen(false)}>关闭</SnapButton>
				</SnapDialogFooter>
			</SnapDialogContent>
		</SnapDialog>

		<SnapDialog open={Boolean(quickInputDraft)} onOpenChange={(open) => { if (!open) closeQuickInputEditor() }}>
			<SnapDialogContent className="max-w-xl">
				<SnapDialogHeader>
					<SnapDialogTitle>{quickInputDraft?.id ? '编辑快捷输入' : '新增快捷输入'}</SnapDialogTitle>
					<SnapDialogDescription>内容会按原样插入终端，不会由 TaskAI 自动执行。</SnapDialogDescription>
				</SnapDialogHeader>
				<div className="grid gap-3 px-1">
					<Field label="名称" hint={`${[...(quickInputDraft?.name ?? '')].length} / 100 个字符`}><Input autoFocus maxLength={100} value={quickInputDraft?.name ?? ''} onChange={(event) => setQuickInputDraft((current) => current ? {...current, name: event.target.value} : current)}/></Field>
					<Field label="内容" hint={`已用 ${[...(quickInputDraft?.content ?? '')].length} 个字符`}><Textarea rows={8} className="font-mono" value={quickInputDraft?.content ?? ''} onChange={(event) => setQuickInputDraft((current) => current ? {...current, content: event.target.value} : current)}/></Field>
				</div>
				<SnapDialogFooter>
					<SnapButton variant="secondary" onClick={closeQuickInputEditor}>取消</SnapButton>
					<SnapButton variant="primary" onClick={() => void saveQuickInput()}>保存快捷输入</SnapButton>
				</SnapDialogFooter>
			</SnapDialogContent>
		</SnapDialog>

		<SnapDialog open={Boolean(quickInputDeletion)} onOpenChange={(open) => { if (!open) setQuickInputDeletion(undefined) }}>
			<SnapDialogContent className="max-w-md">
				<SnapDialogHeader>
					<SnapDialogTitle>删除快捷输入？</SnapDialogTitle>
					<SnapDialogDescription>确认后将删除“{quickInputDeletion?.name}”。此操作无法撤销。</SnapDialogDescription>
				</SnapDialogHeader>
				<SnapDialogFooter>
					<SnapButton variant="secondary" onClick={() => setQuickInputDeletion(undefined)}>取消</SnapButton>
					<SnapButton variant="danger" onClick={() => void confirmDeleteQuickInput()}>删除快捷输入</SnapButton>
				</SnapDialogFooter>
			</SnapDialogContent>
		</SnapDialog>

		<SnapDialog open={extraInfoEditorOpen} onOpenChange={(open) => { if (!open) closeExtraInfoEditor() }}>
			<SnapDialogContent className="max-w-2xl">
				<SnapDialogHeader>
					<SnapDialogTitle>{extraInfoDraft?.id ? '编辑信息' : '新增信息'}</SnapDialogTitle>
				</SnapDialogHeader>
				<SnapScrollArea className="max-h-[55vh]">
					<div className="grid gap-3 px-1">
						{!extraInfoDraft || !extraInfoDraft.id ? (
							<Field label="选择模板">
								<select required autoFocus className={selectClass} value={newExtraInfoTemplateID} onChange={(event) => selectExtraInfoTemplate(event.target.value)}>
									{extraInfoTemplates.map((template) => <option key={template.id} value={template.id}>{`${template.displayName || template.catalogue}（${template.catalogue}）`}</option>)}
								</select>
							</Field>
						) : <span className="text-xs text-snap-muted">{extraInfoDraft.catalogue}</span>}
						{extraInfoDraft && <div data-testid="extra-info-draft-fields" className="grid gap-3 px-0" style={{display: 'grid', gridTemplateColumns: '1fr', minWidth: 0}}>
							{extraInfoDraft.fields.map((field) => <Field key={field.key} label={field.displayName}><Input required={field.key === 'name'} value={field.value ?? ''} onChange={(event) => updateExtraInfoDraftField(field.key, event.target.value)}/></Field>) }
						</div>}
						{extraInfoDraft && <div className="grid gap-2 pt-1">
							<span className="font-display text-sm font-bold text-snap-ink">动态参数</span>
							{extraInfoDraftTemplate && extraInfoDraftTemplate.parameters.length > 0 && <div data-testid="extra-info-template-parameters" className="grid gap-2 border-t-2 border-snap-outline pt-2">
								<span className="text-sm font-bold text-snap-ink">模板参数</span>
								{extraInfoDraftTemplate.parameters.map((parameter, index) => {
									const inputType = extraInfoParameterInputType(parameter)
									return <div key={parameter.key || index} data-testid={`extra-info-template-parameter-preview-${index}`} className="grid gap-1 border-t-2 border-snap-outline py-2" style={{gridTemplateColumns: 'repeat(auto-fit, minmax(150px, 1fr))', minWidth: 0}}>
										<span className="text-sm break-words">参数键：{parameter.key || '未设置'}</span>
										<span className="text-sm break-words">显示名称：{parameter.displayName || '未设置'}</span>
										<span className="text-sm">默认值：{inputType === 'checkbox' ? 'false' : '空'}</span>
										{inputType !== 'checkbox' && <SnapChip className="justify-self-start">{parameter.required ? '必填' : '非必填'}</SnapChip>}
									</div>
								})}
							</div>}
							<div className="flex items-center justify-between gap-2">
								<span className="text-sm font-bold text-snap-ink">信息参数</span>
								<SnapButton variant="secondary" size="sm" onClick={() => setExtraInfoDraft((current) => current ? {...current, parameters: [...(current.parameters ?? []), {key: '', displayName: '', required: false, inputType: 'text', value: ''}]} : current)}>新增动态参数</SnapButton>
							</div>
							<span className="text-xs text-snap-muted">这些参数会和模板参数一起带入任务，可在任务中填写值。</span>
							{(extraInfoDraft.parameters ?? []).map((parameter, index) => <div key={index} data-testid={`extra-info-draft-parameter-${index}`} className="grid gap-2 border-t-2 border-snap-outline py-2" style={{borderTopWidth: '2px', borderTopStyle: 'solid', borderTopColor: 'var(--snap-outline)', minWidth: 0}}>
								<div className="grid gap-2" style={{gridTemplateColumns: 'repeat(auto-fit, minmax(150px, 1fr))', minWidth: 0}}>
									<Field label={`参数键 ${index + 1}`}><Input required value={parameter.key} onChange={(event) => setExtraInfoDraft((current) => current ? {...current, parameters: (current.parameters ?? []).map((item, parameterIndex) => parameterIndex === index ? {...item, key: event.target.value} : item)} : current)}/></Field>
									<Field label={`参数显示名称 ${index + 1}`}><Input required value={parameter.displayName} onChange={(event) => setExtraInfoDraft((current) => current ? {...current, parameters: (current.parameters ?? []).map((item, parameterIndex) => parameterIndex === index ? {...item, displayName: event.target.value} : item)} : current)}/></Field>
									<Field label={`参数类型 ${index + 1}`}>
										<select className={selectClass} value={extraInfoParameterInputType(parameter)} onChange={(event) => setExtraInfoDraft((current) => current ? {...current, parameters: (current.parameters ?? []).map((item, parameterIndex) => {
											if (parameterIndex !== index) {
												return item
											}
											const inputType = event.target.value as ExtraInfoParameterInputType
											return {...item, inputType, required: inputType === 'checkbox' ? false : item.required, value: inputType === 'checkbox' ? 'false' : item.value}
										})} : current)}>
											<option value="text">文本</option>
											<option value="checkbox">复选框</option>
										</select>
									</Field>
								</div>
								<div className="flex items-center justify-between flex-wrap gap-2" style={{minWidth: 0}}>
									{extraInfoParameterInputType(parameter) === 'checkbox'
										? <label className="flex items-center gap-2 text-sm text-snap-ink"><SnapCheckbox checked={parameter.value === 'true'} onCheckedChange={(checked) => setExtraInfoDraft((current) => current ? {...current, parameters: (current.parameters ?? []).map((item, parameterIndex) => parameterIndex === index ? {...item, value: checked ? 'true' : 'false'} : item)} : current)}/><span>{`默认值 ${index + 1}`}</span></label>
										: <Field label={`默认值 ${index + 1}`} className="min-w-[180px] flex-1"><Input value={parameter.value} onChange={(event) => setExtraInfoDraft((current) => current ? {...current, parameters: (current.parameters ?? []).map((item, parameterIndex) => parameterIndex === index ? {...item, value: event.target.value} : item)} : current)}/></Field>
									}
									{extraInfoParameterInputType(parameter) !== 'checkbox' && <label className="flex items-center gap-2 text-sm text-snap-ink"><SnapCheckbox checked={parameter.required} onCheckedChange={(checked) => setExtraInfoDraft((current) => current ? {...current, parameters: (current.parameters ?? []).map((item, parameterIndex) => parameterIndex === index ? {...item, required: checked === true} : item)} : current)}/><span>{`参数 ${index + 1} 必填`}</span></label>}
									<SnapIconButton title="删除动态参数" aria-label={`删除动态参数 ${parameter.displayName || index + 1}`} size="sm" onClick={() => setExtraInfoDraft((current) => current ? {...current, parameters: (current.parameters ?? []).filter((_, parameterIndex) => parameterIndex !== index)} : current)} className="text-snap-error hover:text-snap-error"><Trash2 className="h-4 w-4" strokeWidth={2.25}/></SnapIconButton>
								</div>
							</div>)}
						</div>}
					</div>
				</SnapScrollArea>
				<SnapDialogFooter>
					<SnapButton variant="secondary" onClick={closeExtraInfoEditor}>取消</SnapButton>
					<SnapButton variant="primary" disabled={!extraInfoDraft} onClick={() => void saveExtraInfo()}>保存信息</SnapButton>
				</SnapDialogFooter>
			</SnapDialogContent>
		</SnapDialog>

		<SnapDialog open={Boolean(extraInfoTemplateDraft)} onOpenChange={(open) => { if (!open) setExtraInfoTemplateDraft(undefined) }}>
			<SnapDialogContent className="max-w-2xl">
				<SnapDialogHeader>
					<SnapDialogTitle>{extraInfoTemplateDraft?.id ? '编辑模板' : '新增模板'}</SnapDialogTitle>
				</SnapDialogHeader>
				<SnapScrollArea className="max-h-[55vh]">
					<div className="grid gap-3 px-1">
						{extraInfoTemplateDraft?.builtIn && <SnapAlert severity="info">Git 内置字段的键和显示名称不可修改；可调整默认值、分支必填状态，并添加新的字段或参数。</SnapAlert>}
						<div data-testid="extra-info-template-basic-fields" className="grid gap-3" style={{display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(220px, 1fr))', minWidth: 0}}>
							<Field label="分类"><Input required disabled={extraInfoTemplateDraft?.builtIn} value={extraInfoTemplateDraft?.catalogue ?? ''} onChange={(event) => updateExtraInfoTemplateDraft({catalogue: event.target.value})}/></Field>
							<Field label="模板备注"><Input disabled={extraInfoTemplateDraft?.builtIn} value={extraInfoTemplateDraft?.displayName ?? ''} onChange={(event) => updateExtraInfoTemplateDraft({displayName: event.target.value})}/></Field>
						</div>
						<div className="grid gap-2">
							<div className="flex justify-between items-center">
								<span className="font-display text-sm font-bold text-snap-ink">固定字段</span>
								<SnapButton variant="secondary" size="sm" onClick={() => updateExtraInfoTemplateDraft({fields: [...(extraInfoTemplateDraft?.fields ?? []), {key: '', displayName: '', defaultValue: ''}]})}>新增固定字段</SnapButton>
							</div>
							{extraInfoTemplateDraft?.fields.map((field, index) => {
								const protectedField = Boolean(extraInfoTemplateDraft.builtIn && (field.key === 'name' || field.key === 'repository'))
								return <div key={index} data-testid={`extra-info-template-fixed-field-${index}`} className="grid items-start gap-2 border-t-2 border-snap-outline py-2" style={{borderTopWidth: '2px', borderTopStyle: 'solid', borderTopColor: 'var(--snap-outline)', gridTemplateColumns: 'minmax(0, 1fr) auto', minWidth: 0}}>
									<div className="grid gap-2" style={{gridTemplateColumns: 'repeat(auto-fit, minmax(150px, 1fr))', minWidth: 0}}>
										<Field label={`固定键 ${index + 1}`}><Input required disabled={protectedField} value={field.key} onChange={(event) => updateExtraInfoTemplateField(index, {key: event.target.value})}/></Field>
										<Field label={`固定字段显示名称 ${index + 1}`}><Input required disabled={protectedField} value={field.displayName} onChange={(event) => updateExtraInfoTemplateField(index, {displayName: event.target.value})}/></Field>
										<Field label={`默认值 ${index + 1}`}><Input value={field.defaultValue ?? ''} onChange={(event) => updateExtraInfoTemplateField(index, {defaultValue: event.target.value})}/></Field>
									</div>
									<SnapIconButton title="删除固定字段" aria-label={`删除固定字段 ${index + 1}`} size="sm" disabled={protectedField || extraInfoTemplateDraft.fields.length === 1} onClick={() => updateExtraInfoTemplateDraft({fields: extraInfoTemplateDraft.fields.filter((_, fieldIndex) => fieldIndex !== index)})} className="text-snap-error hover:text-snap-error"><Trash2 className="h-4 w-4" strokeWidth={2.25}/></SnapIconButton>
								</div>
							})}
						</div>
						<div className="grid gap-2">
							<div className="flex justify-between items-center">
								<span className="font-display text-sm font-bold text-snap-ink">动态参数</span>
								<SnapButton variant="secondary" size="sm" onClick={() => updateExtraInfoTemplateDraft({parameters: [...(extraInfoTemplateDraft?.parameters ?? []), {key: '', displayName: '', required: false, inputType: 'text'}]})}>新增参数</SnapButton>
							</div>
							{extraInfoTemplateDraft?.parameters.map((parameter, index) => {
								const protectedParameter = Boolean(extraInfoTemplateDraft.builtIn && parameter.key === 'branch')
								return <div key={index} data-testid={`extra-info-template-parameter-${index}`} className="grid items-start gap-2 border-t-2 border-snap-outline py-2" style={{borderTopWidth: '2px', borderTopStyle: 'solid', borderTopColor: 'var(--snap-outline)', gridTemplateColumns: 'minmax(0, 1fr) auto', minWidth: 0}}>
									<div className="grid gap-2" style={{minWidth: 0}}>
										<div className="grid gap-2" style={{gridTemplateColumns: 'repeat(auto-fit, minmax(150px, 1fr))', minWidth: 0}}>
											<Field label={`参数键 ${index + 1}`}><Input required disabled={protectedParameter} value={parameter.key} onChange={(event) => updateExtraInfoTemplateParameter(index, {key: event.target.value})}/></Field>
											<Field label={`参数显示名称 ${index + 1}`}><Input required disabled={protectedParameter} value={parameter.displayName} onChange={(event) => updateExtraInfoTemplateParameter(index, {displayName: event.target.value})}/></Field>
											<Field label={`参数类型 ${index + 1}`}>
												<select className={selectClass} disabled={protectedParameter} value={extraInfoParameterInputType(parameter)} onChange={(event) => {
													const inputType = event.target.value as ExtraInfoParameterInputType
													updateExtraInfoTemplateParameter(index, {inputType, required: inputType === 'checkbox' ? false : parameter.required})
												}}>
													<option value="text">文本</option>
													<option value="checkbox">复选框</option>
												</select>
											</Field>
										</div>
										{extraInfoParameterInputType(parameter) !== 'checkbox' && <label className="flex items-center gap-2 text-sm text-snap-ink"><SnapCheckbox checked={parameter.required} onCheckedChange={(checked) => updateExtraInfoTemplateParameter(index, {required: checked === true})}/><span>{`参数 ${index + 1} 必填`}</span></label>}
									</div>
									<SnapIconButton title="删除参数" aria-label={`删除参数 ${index + 1}`} size="sm" disabled={protectedParameter} onClick={() => updateExtraInfoTemplateDraft({parameters: extraInfoTemplateDraft.parameters.filter((_, parameterIndex) => parameterIndex !== index)})} className="text-snap-error hover:text-snap-error"><Trash2 className="h-4 w-4" strokeWidth={2.25}/></SnapIconButton>
								</div>
							})}
						</div>
					</div>
				</SnapScrollArea>
				<SnapDialogFooter>
					<SnapButton variant="secondary" onClick={() => setExtraInfoTemplateDraft(undefined)}>取消</SnapButton>
					<SnapButton variant="primary" onClick={() => void saveExtraInfoTemplate()}>保存信息</SnapButton>
				</SnapDialogFooter>
			</SnapDialogContent>
		</SnapDialog>

      <SnapDialog open={settingsDialogOpen} onOpenChange={(open) => { if (!open) closeSettingsDialog() }}>
        <SnapDialogContent className="max-w-6xl">
          <SnapDialogHeader>
            <SnapDialogTitle>设置</SnapDialogTitle>
          </SnapDialogHeader>
          <SnapScrollArea className="max-h-[62vh] min-w-0">
            <div className="grid min-w-0 gap-4 px-1">
              <SnapTabs value={settingsTab} onValueChange={(value) => setSettingsTab(value as 'workspace' | 'shell' | 'shortcuts' | 'menu' | 'status' | 'lifecycle' | 'templates')}>
                <SnapTabsList className="overflow-x-auto snap-no-scrollbar" aria-label="设置分类">
                  <SnapTabsTrigger className="flex-none min-w-max" value="workspace">工作区与外观</SnapTabsTrigger>
                  <SnapTabsTrigger className="flex-none min-w-max" value="shell">终端 Shell</SnapTabsTrigger>
                  <SnapTabsTrigger className="flex-none min-w-max" value="shortcuts">终端快捷键</SnapTabsTrigger>
                  <SnapTabsTrigger className="flex-none min-w-max" value="menu">菜单管理</SnapTabsTrigger>
                  <SnapTabsTrigger className="flex-none min-w-max" value="status">实时状态</SnapTabsTrigger>
                  <SnapTabsTrigger className="flex-none min-w-max" value="lifecycle">生命周期编排</SnapTabsTrigger>
                  <SnapTabsTrigger className="flex-none min-w-max" value="templates">任务模板</SnapTabsTrigger>
                </SnapTabsList>
              </SnapTabs>

              {settingsTab === 'workspace' && <div className="grid gap-3">
                <Field label="新任务工作区根目录" hint="仅影响之后开始执行的任务，已有任务保持各自目录快照。"><Input required value={settingsDraft?.workspaceRoot ?? ''} onChange={(event) => updateSettingsDraft({workspaceRoot: event.target.value})}/></Field>
                <Field label="颜色模式">
                  <select className={selectClass} value={settingsDraft?.colorScheme ?? colorScheme} onChange={(event) => updateSettingsDraft({colorScheme: event.target.value as ColorScheme})}>
                    <option value="light">亮色</option>
                    <option value="dark">暗色</option>
                  </select>
                </Field>
				<Field label="终端字体">
					<div className="grid gap-2">
						{terminalFontsLoading && <span role="status" className="text-sm text-snap-muted">正在读取系统字体…</span>}
						{terminalFontsError && <span role="status" className="text-sm text-snap-muted">系统字体不可用，已保留默认终端字体。</span>}
						<SnapAccordion type="single" collapsible value={terminalFontPickerExpanded ? 'terminal-fonts' : ''} onValueChange={(value) => setTerminalFontPickerExpanded(value === 'terminal-fonts')}>
							<SnapAccordionItem value="terminal-fonts">
								<SnapAccordionTrigger aria-label={`终端字体选择，当前为 ${terminalFontCandidateName(selectedTerminalFont)}`}>
									<div className="grid w-full gap-2 pr-2">
										<div className="flex items-center justify-between gap-3">
											<span className="min-w-0 truncate text-sm font-bold text-snap-ink">{terminalFontCandidateName(selectedTerminalFont)}</span>
											<SnapChip variant={selectedTerminalFont.spacing === 'unavailable' ? 'amber' : 'muted'}>{terminalFontSpacingLabel(selectedTerminalFont.spacing)}</SnapChip>
										</div>
										<div className="grid gap-0.5 text-sm leading-5 text-snap-ink" style={{fontFamily: resolveTerminalFontFamily(selectedTerminalFont.family), fontSize: `${terminalFontSizeDraft}px`}}>
											<span>中文：终端字体预览</span>
											<span>English: TaskAI $ git status</span>
											<span>┌─ ~/taskai $ █</span>
										</div>
									</div>
								</SnapAccordionTrigger>
								<SnapAccordionContent>
									<div role="radiogroup" aria-label="终端字体" className="grid max-h-72 gap-2 overflow-y-auto pr-1">
										{fontOptions.map((candidate) => {
											const selected = (settingsDraft?.terminalFontFamily?.trim() ?? '') === candidate.family
											const name = terminalFontCandidateName(candidate)
											return <button
												key={`${candidate.family}-${candidate.spacing}`}
												type="button"
												role="radio"
												aria-label={`选择终端字体 ${name}`}
												aria-checked={selected}
												onClick={() => {
													updateSettingsDraft({terminalFontFamily: candidate.family})
													setTerminalFontPickerExpanded(false)
												}}
												className={cn('grid w-full gap-2 rounded-snap border-2 p-3 text-left shadow-snap-sm outline-none transition-[border-color,box-shadow,transform] focus-visible:border-snap-cobalt focus-visible:ring-[3px] focus-visible:ring-snap-cobalt', selected ? 'border-snap-cobalt bg-snap-cobalt/10' : 'border-snap-outline bg-snap-surface hover:-translate-x-px hover:-translate-y-px hover:shadow-snap')}
											>
												<div className="flex items-center justify-between gap-3">
													<span className="min-w-0 truncate text-sm font-bold text-snap-ink">{name}</span>
													<SnapChip variant={candidate.spacing === 'unavailable' ? 'amber' : 'muted'}>{terminalFontSpacingLabel(candidate.spacing)}</SnapChip>
												</div>
												<div className="grid gap-0.5 text-sm leading-5 text-snap-ink" style={{fontFamily: resolveTerminalFontFamily(candidate.family), fontSize: `${terminalFontSizeDraft}px`}}>
													<span>中文：终端字体预览</span>
													<span>English: TaskAI $ git status</span>
													<span>┌─ ~/taskai $ █</span>
												</div>
											</button>
										})}
									</div>
								</SnapAccordionContent>
							</SnapAccordionItem>
						</SnapAccordion>
					</div>
				</Field>
				<Field label="终端字号">
					<div className="grid gap-2">
						<div className="flex items-center justify-between gap-3">
							<span className="text-sm font-bold tabular-nums text-snap-ink" aria-live="polite">{`${terminalFontSizeDraft} px（${terminalFontSizePercent(terminalFontSizeDraft)}%）`}</span>
							<SnapTooltip>
								<SnapTooltipTrigger asChild>
									<SnapIconButton title="恢复默认字号（13 px）" aria-label="恢复默认字号（13 px）" size="sm" onClick={() => updateSettingsDraft({terminalFontSize: defaultTerminalFontSize})}>
										<RotateCcw className="h-4 w-4" strokeWidth={2.25}/>
									</SnapIconButton>
								</SnapTooltipTrigger>
								<SnapTooltipContent>恢复默认字号（13 px）</SnapTooltipContent>
							</SnapTooltip>
						</div>
						<input
							type="range"
							aria-label="终端字号"
							min={minimumTerminalFontSize}
							max={maximumTerminalFontSize}
							step={1}
							value={terminalFontSizeDraft}
							onChange={(event) => updateSettingsDraft({terminalFontSize: Number(event.target.value)})}
							className="w-full accent-snap-cobalt"
						/>
						<div className="flex justify-between text-xs tabular-nums text-snap-muted"><span>{minimumTerminalFontSize} px</span><span>{maximumTerminalFontSize} px</span></div>
					</div>
				</Field>
				<Field label="终端配色" hint="配色独立于应用亮暗模式；编辑仅会更新此处预览，保存后才应用到终端。">
					<SnapAccordion type="single" collapsible value={terminalThemePickerExpanded ? 'terminal-theme' : ''} onValueChange={(value) => setTerminalThemePickerExpanded(value === 'terminal-theme')}>
						<SnapAccordionItem value="terminal-theme">
							<SnapAccordionTrigger aria-label="终端配色设置">
								<span className="text-sm font-bold text-snap-ink">自定义终端颜色</span>
							</SnapAccordionTrigger>
							<SnapAccordionContent>
								<div className="grid gap-4 pt-2">
									<div
										data-testid="terminal-theme-preview"
										className="rounded-snap border-2 p-3 font-mono text-sm shadow-snap-sm"
										style={{backgroundColor: terminalThemeDraft.background, color: terminalThemeDraft.foreground, fontFamily: resolveTerminalFontFamily(selectedTerminalFont.family), fontSize: `${terminalFontSizeDraft}px`}}
									>
										<div><span style={{color: terminalThemeDraft.green}}>taskai</span><span style={{color: terminalThemeDraft.foreground}}> $ git status</span></div>
										<div style={{color: terminalThemeDraft.brightWhite}}>On branch main</div>
										<div style={{backgroundColor: terminalThemeDraft.selectionBackground, color: terminalThemeDraft.selectionForeground, display: 'inline'}}>已选择的终端输出</div>
									</div>
									<div className="grid gap-2">
										<span className="text-sm font-bold text-snap-ink">基础颜色</span>
										<div className="grid grid-cols-2 gap-2 sm:grid-cols-3">
											{terminalBasicColorFields.map((field) => <TerminalThemeColorPicker
												key={field.key}
												label={field.label}
												value={terminalThemeDraft[field.key]}
												onChange={(value) => updateSettingsDraft({terminalTheme: {...terminalThemeDraft, [field.key]: value}})}
											/>)}
										</div>
									</div>
									<div className="grid gap-2">
										<div className="flex items-center justify-between gap-3">
											<span className="text-sm font-bold text-snap-ink">选区背景</span>
											<TerminalThemeColorPicker label="选区背景" value={terminalThemeDraft.selectionBackground} onChange={(value) => updateSettingsDraft({terminalTheme: {...terminalThemeDraft, selectionBackground: `${value}${terminalThemeDraft.selectionBackground.slice(7, 9)}`}})}/>
										</div>
										<div className="grid gap-1">
											<div className="flex items-center justify-between text-sm text-snap-ink"><label htmlFor="terminal-selection-opacity">选区透明度</label><span className="tabular-nums">{terminalSelectionOpacity}%</span></div>
											<input id="terminal-selection-opacity" aria-label="终端选区透明度" type="range" min={0} max={100} step={1} value={terminalSelectionOpacity} onChange={(event) => updateSettingsDraft({terminalTheme: terminalThemeWithSelectionOpacity(terminalThemeDraft, Number(event.target.value))})} className="w-full accent-snap-cobalt"/>
										</div>
									</div>
									<TerminalThemeColorGroup title="常规 ANSI" fields={terminalAnsiColorFields} theme={terminalThemeDraft} onChange={(key, value) => updateSettingsDraft({terminalTheme: {...terminalThemeDraft, [key]: value}})}/>
									<TerminalThemeColorGroup title="高亮 ANSI" fields={terminalBrightAnsiColorFields} theme={terminalThemeDraft} onChange={(key, value) => updateSettingsDraft({terminalTheme: {...terminalThemeDraft, [key]: value}})}/>
									<div className="flex justify-end">
										<SnapButton variant="secondary" size="sm" onClick={() => updateSettingsDraft({terminalTheme: defaultTerminalTheme})}><RotateCcw className="mr-1 h-4 w-4" strokeWidth={2.25}/>恢复默认终端配色</SnapButton>
									</div>
								</div>
							</SnapAccordionContent>
						</SnapAccordionItem>
					</SnapAccordion>
				</Field>
              </div>}

              {settingsTab === 'shell' && <div className="grid gap-3">
                <Field label="探测到的 Shell" hint="选择后会自动填入下方的 Shell 路径。">
                  <select className={selectClass} value={detectedShells.includes(settingsDraft?.shellPath ?? '') ? settingsDraft?.shellPath ?? '' : ''} onChange={(event) => { if (event.target.value) { updateSettingsDraft({shellPath: event.target.value}) } }}>
                    <option value="">手动设置路径</option>
                    {detectedShells.map((shellPath) => <option key={shellPath} value={shellPath}>{shellPath}</option>)}
                  </select>
                </Field>
                <Field label="Shell 路径" hint="此 Shell 会启动任务终端，并提供自定义命令所需的初始化环境。"><Input required value={settingsDraft?.shellPath ?? ''} onChange={(event) => updateSettingsDraft({shellPath: event.target.value})}/></Field>
              </div>}

              {settingsTab === 'shortcuts' && <TerminalShortcutSettings
                shortcuts={settingsDraft?.terminalShortcuts ?? []}
                candidatePrograms={uniqueProgramNames([
                  settingsDraft?.shellPath,
                  ...(settingsDraft?.taskMenuItems ?? [])
                    .filter((item) => item.kind === 'command' && item.showTerminal && item.command)
                    .map((item) => item.command),
                ])}
                onChange={(terminalShortcuts) => updateSettingsDraft({terminalShortcuts})}
              />}

              {settingsTab === 'menu' && <div className="grid gap-3">
                <div className="flex items-center justify-between gap-3">
                  <span className="text-sm text-snap-muted">右键菜单与“任务操作”下拉菜单共用此顺序。系统项可改名与调序，固定操作不可修改。</span>
                  <SnapButton variant="primary" size="sm" onClick={() => openTaskMenuItemEditor()}>新增菜单项</SnapButton>
                </div>
                <div className="overflow-hidden rounded-snap border-2 border-snap-outline divide-y divide-snap-outline">
                  {settingsDraft?.taskMenuItems.map((item, index) => (
                    <div key={item.id} className="flex items-center gap-2 px-3 py-2">
                      <div className="min-w-0 flex-1">
                        <span className="block text-sm font-bold text-snap-ink truncate">{item.name}</span>
                        <span className="block text-xs text-snap-muted truncate">{item.kind === 'command' ? item.command : '系统固定操作'}</span>
                      </div>
                      <SnapChip>{item.kind === 'command' ? item.showTerminal ? '显示终端' : '后台启动' : '系统固定'}</SnapChip>
                      <SnapButton variant="secondary" size="sm" aria-label={`编辑菜单项 ${item.name}`} onClick={() => openTaskMenuItemEditor(item)}>编辑</SnapButton>
                      <SnapIconButton title={`上移 ${item.name}`} aria-label={`上移 ${item.name}`} size="sm" disabled={index === 0} onClick={() => moveTaskMenuItem(item.id, -1)}><ArrowUp className="h-4 w-4" strokeWidth={2.25}/></SnapIconButton>
                      <SnapIconButton title={`下移 ${item.name}`} aria-label={`下移 ${item.name}`} size="sm" disabled={index === settingsDraft.taskMenuItems.length - 1} onClick={() => moveTaskMenuItem(item.id, 1)}><ArrowDown className="h-4 w-4" strokeWidth={2.25}/></SnapIconButton>
                    </div>
                  ))}
                </div>
              </div>}

              {settingsTab === 'status' && <div className="grid gap-4">
                <div className="grid gap-1">
                  <span className="font-display text-sm font-bold text-snap-ink">状态判定</span>
                  <span className="text-sm text-snap-muted">状态仅保存在本次应用会话中：<span>{statusManagementDescription}</span></span>
                </div>
                <Field label="状态管理方式">
                  <select className={selectClass} value={statusManagementMode} onChange={(event) => updateSettingsDraft({statusManagementMode: event.target.value as SettingsRecord['statusManagementMode']})}>
                    <option value="title-change">根据终端标题变化</option>
                    <option value="output-change">根据终端输出变化</option>
                    <option value="http">通过 HTTP 接口</option>
                  </select>
                </Field>
                <label className="flex items-center gap-2 text-sm text-snap-ink">
                  <SnapSwitch checked={httpServiceActive} disabled={statusManagementMode === 'http'} onCheckedChange={(checked) => updateSettingsDraft({httpServiceEnabled: checked === true})}/>
                  <span>{statusManagementMode === 'http' ? '通过 HTTP 状态管理自动启用本机 HTTP 服务' : '启用本机 HTTP 服务'}</span>
                </label>
                {httpServiceActive && <>
                  <Field label="HTTP 端口" hint="仅监听 127.0.0.1；关闭独立服务且状态不使用 HTTP 时会停止服务。"><Input required type="number" min={1} max={65535} value={settingsDraft?.statusManagementHTTPPort ?? 0} onChange={(event) => updateSettingsDraft({statusManagementHTTPPort: Number(event.target.value)})}/></Field>
                  <div className="flex justify-start">
                    <SnapButton variant="secondary" size="sm" onClick={() => setStatusHelpOpen(true)}>查看 HTTP 接口使用说明</SnapButton>
                  </div>
                </>}
              </div>}

			{settingsTab === 'lifecycle' && <LifecycleManagement
				commands={settings?.lifecycleCommands ?? []}
				chains={settings?.lifecycleChains ?? []}
				presets={lifecyclePresets}
				defaultPresetID={settings?.defaultLifecyclePresetId ?? ''}
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
				onSavePreset={async (preset) => {
					await api.saveLifecyclePreset(preset)
					await refreshLifecycleConfiguration()
				}}
				onCopyPreset={async (presetID) => {
					await api.copyLifecyclePreset(presetID)
					await refreshLifecycleConfiguration()
				}}
				onDeletePreset={async (presetID) => {
					await api.deleteLifecyclePreset(presetID)
					await refreshLifecycleConfiguration()
				}}
				onSaveDefaultPreset={async (presetID) => {
					const saved = await api.saveDefaultLifecyclePreset(presetID)
					setSettings(saved)
					setSettingsDraft((current) => current ? {
						...current,
						lifecyclePresets: saved.lifecyclePresets ?? [],
						defaultLifecyclePresetId: saved.defaultLifecyclePresetId ?? '',
					} : current)
				}}
				onError={(error) => showError(error, setMessage)}
			/>}

			{settingsTab === 'templates' && <TaskTemplateManagement
				templates={settingsDraft?.taskTemplates ?? []}
				activeTemplateID={settingsDraft?.activeTaskTemplateId ?? ''}
				activeTaskTemplateIDs={activeTaskTemplateIDs}
				hasLegacyActiveTaskTemplateFields={hasLegacyActiveTaskTemplateFields}
				onChange={(taskTemplates, activeTaskTemplateId) => updateSettingsDraft({taskTemplates, activeTaskTemplateId})}
			/>}
            </div>
          </SnapScrollArea>
          <SnapDialogFooter>
            <SnapButton variant="secondary" onClick={closeSettingsDialog}>取消</SnapButton>
            <SnapButton variant="primary" onClick={() => void saveSettings()}>保存</SnapButton>
          </SnapDialogFooter>
        </SnapDialogContent>
      </SnapDialog>

      <SnapDialog open={statusHelpOpen} onOpenChange={(open) => { if (!open) setStatusHelpOpen(false) }}>
        <SnapDialogContent className="max-w-2xl" showClose={false}>
          <SnapDialogHeader>
            <SnapDialogTitle>HTTP 状态接口使用说明</SnapDialogTitle>
            <span className="mt-0.5 block text-xs text-snap-muted" aria-hidden="true">本机接口参考 · API v1</span>
          </SnapDialogHeader>
          <SnapScrollArea className="max-h-[62vh]">
            <div className="grid gap-4 px-1 py-1">
              <HTTPHelpSection title="服务与设置">
                <SnapAlert severity="info">服务仅监听 <code>127.0.0.1:&lt;端口&gt;</code>，无需鉴权，也不会暴露到局域网。</SnapAlert>
                <div className="grid gap-2 sm:grid-cols-2">
                  <HTTPHelpStep number="1" title="独立启用服务">在“实时状态”中开启“启用本机 HTTP 服务”并设置端口，可单独查询任务和状态；本机 HTTP 服务正在监听时，之后新建的终端会获得 API 地址。</HTTPHelpStep>
                  <HTTPHelpStep number="2" title="使用 HTTP 管理状态">选择“通过 HTTP 接口”会自动启用服务，因此之后新建的终端也会获得 API 地址。</HTTPHelpStep>
                </div>
              </HTTPHelpSection>

              <HTTPHelpSection title="终端环境变量">
                <span className="text-sm text-snap-muted">新建的普通终端和显示终端的自定义命令始终获得任务与终端 ID；本机 HTTP 服务正在监听时额外获得 API 地址：</span>
                <HTTPCodeBlock>{'TASKAI_TASK_ID=<任务 ID>\nTASKAI_TERMINAL_ID=<终端 ID>\n\n# 仅本机 HTTP 服务正在监听时注入\nTASKAI_STATUS_API=http://127.0.0.1:<端口>/api/v1'}</HTTPCodeBlock>
                <span className="text-sm text-snap-muted">无终端后台命令以及前置、后置脚本仅注入 <code>TASKAI_TASK_ID</code>。</span>
              </HTTPHelpSection>

              <HTTPHelpSection title="查询接口">
				<HTTPEndpoint method="GET" path="/api/v1/status?status=pending|running|completed">查询任务和终端的实时状态；可按任务生命周期筛选，条目包含任务名称与生命周期状态。</HTTPEndpoint>
                <HTTPEndpoint method="GET" path="/api/v1/tasks?status=pending|running|completed">按任务生命周期筛选列表；省略 <code>status</code> 时返回全部任务。</HTTPEndpoint>
                <span className="text-sm text-snap-muted">任务列表查询参数：可省略；可选值为 pending、running、completed。</span>
                <HTTPEndpoint method="GET" path="/api/v1/tasks/:taskId">查询单个任务详情，包含标题、描述、生命周期、时间、工作目录和附加信息。</HTTPEndpoint>
                <HTTPCodeBlock>{'curl "$TASKAI_STATUS_API/status"\n\ncurl "$TASKAI_STATUS_API/tasks?status=running"\n\ncurl "$TASKAI_STATUS_API/tasks/$TASKAI_TASK_ID"'}</HTTPCodeBlock>
              </HTTPHelpSection>

				<HTTPHelpSection title="任务附加信息">
					<span className="text-sm text-snap-muted">先通过顶部“额外信息管理”定义模板并保存信息。任务详情的 <code>extraInfo</code> 按目录聚合，将固定字段和动态参数平铺；信息名称以固定字段 <code>name</code> 返回。</span>
					<HTTPCodeBlock>{'{"extraInfo":{"git":[{"name":"API 服务","repository":"git@example.com:team/api.git","branch":"main"}]}}'}</HTTPCodeBlock>
				</HTTPHelpSection>

              <HTTPHelpSection title="状态更新">
                <HTTPEndpoint method="PUT" path="/api/v1/tasks/:taskId/status">直接设置任务状态；下一次终端状态更新会重新按终端状态汇总。</HTTPEndpoint>
                <HTTPEndpoint method="PUT" path="/api/v1/tasks/:taskId/terminals/:terminalId/status">更新指定终端，并自动汇总对应任务的状态。</HTTPEndpoint>
                <div className="grid gap-2">
                  <span className="text-sm text-snap-muted">两个更新接口都使用以下 JSON 请求体：</span>
                  <HTTPCodeBlock>{'{"status":"idle|working|unread|error"}'}</HTTPCodeBlock>
                  <span className="text-sm text-snap-muted">状态更新请求体的 status：必填；合法值为 idle、working、unread、error。</span>
                  <HTTPCodeBlock>{'curl -X PUT "$TASKAI_STATUS_API/tasks/$TASKAI_TASK_ID/terminals/$TASKAI_TERMINAL_ID/status" \\\n  -H "Content-Type: application/json" \\\n  --data \'{"status":"working"}\''}</HTTPCodeBlock>
                </div>
              </HTTPHelpSection>

              <HTTPHelpSection title="状态与错误规则">
                <div className="grid gap-2 rounded-snap border-2 border-snap-outline bg-snap-elevated p-2">
                  <span className="text-sm text-snap-ink"><strong>状态值：</strong><code>idle</code> 空闲、<code>working</code> 工作中、<code>unread</code> 未读、<code>error</code> 异常。</span>
                  <span className="text-sm text-snap-ink"><strong>汇总优先级：</strong>异常 → 未读 → 工作中 → 空闲。</span>
                  <span className="text-sm text-snap-ink"><strong>错误响应：</strong><code>{'{"error":"..."}'}</code>；无效请求为 <code>400</code>，不存在的任务或终端为 <code>404</code>，已结束任务或已关闭终端为 <code>409</code>，错误方法为 <code>405</code>。</span>
                  <span className="text-sm text-snap-muted">修改端口、服务开关或切换状态管理方式不会更新已运行终端的环境变量；请新建终端后再使用新配置。</span>
                </div>
              </HTTPHelpSection>
            </div>
          </SnapScrollArea>
          <SnapDialogFooter>
            <SnapButton variant="secondary" onClick={() => setStatusHelpOpen(false)}>关闭</SnapButton>
          </SnapDialogFooter>
        </SnapDialogContent>
      </SnapDialog>

      {taskMenuItemDraft && <SnapDialog open onOpenChange={(open) => { if (!open) closeTaskMenuItemEditor() }}>
        <SnapDialogContent className="max-w-md">
          <form onSubmit={saveTaskMenuItem} style={{display: 'flex', flexDirection: 'column', flex: '1 1 auto', minHeight: 0}}>
            <SnapDialogHeader>
              <SnapDialogTitle>{taskMenuItemEditorMode === 'create' ? '新增菜单项' : '编辑菜单项'}</SnapDialogTitle>
            </SnapDialogHeader>
            {taskMenuItemIsCommand && <SnapTabs value={taskMenuItemEditorTab} onValueChange={(value) => setTaskMenuItemEditorTab(value as 'basic' | 'scripts')}>
              <SnapTabsList aria-label="菜单项配置分类">
                <SnapTabsTrigger value="basic">基本配置</SnapTabsTrigger>
                <SnapTabsTrigger value="scripts">前后置脚本</SnapTabsTrigger>
              </SnapTabsList>
            </SnapTabs>}

            <div className="grid gap-3 px-1 py-1" style={{overflowY: 'auto', minHeight: 0}}>
              {taskMenuItemEditorTab === 'basic' && taskMenuItemIsCommand && <>
                <Field label="菜单名称"><Input autoFocus required value={taskMenuItemDraft.name} onChange={(event) => setTaskMenuItemDraft((current) => current ? {...current, name: event.target.value} : current)}/></Field>
                <Field label="启动命令"><Input required value={taskMenuItemDraft.command ?? ''} onChange={(event) => setTaskMenuItemDraft((current) => current ? {...current, command: event.target.value} : current)}/></Field>
                <Field label="启动参数（每行一个）" hint="每行代表一个启动参数。"><Textarea rows={2} value={(taskMenuItemDraft.arguments ?? []).join('\n')} onChange={(event) => setTaskMenuItemDraft((current) => current ? {...current, arguments: event.target.value.split('\n')} : current)}/></Field>
                <label className="flex items-center gap-2 text-sm text-snap-ink">
                  <SnapSwitch checked={taskMenuItemDraft.showTerminal} onCheckedChange={(checked) => setTaskMenuItemDraft((current) => {
                    if (!current) {
                      return current
                    }
                    const showTerminal = checked === true
                    return {
                      ...current,
                      showTerminal,
                      disableTaskAIMouseClipboard: showTerminal ? current.disableTaskAIMouseClipboard === true : false,
                    }
                  })}/>
                  <span>显示终端</span>
                </label>
                {taskMenuItemDraft.showTerminal && <label className="flex items-center gap-2 text-sm text-snap-ink">
                  <SnapSwitch checked={taskMenuItemDraft.disableTaskAIMouseClipboard === true} onCheckedChange={(checked) => setTaskMenuItemDraft((current) => current ? {...current, disableTaskAIMouseClipboard: checked === true} : current)}/>
                  <span>禁用 TaskAI 鼠标复制与右键粘贴</span>
                </label>}
              </>}

              {taskMenuItemEditorTab === 'basic' && !taskMenuItemIsCommand && <>
                <span className="text-sm text-snap-muted">系统固定操作不可修改。</span>
                {taskMenuItemIsShelvingToggle ? <>
                  <Field label="搁置任务名称"><Input autoFocus required value={taskMenuItemDraft.name} onChange={(event) => setTaskMenuItemDraft((current) => current ? {...current, name: event.target.value} : current)}/></Field>
                  <Field label="取消搁置名称"><Input required value={taskMenuItemDraft.unshelveName ?? ''} onChange={(event) => setTaskMenuItemDraft((current) => current ? {...current, unshelveName: event.target.value} : current)}/></Field>
                </> : <Field label="菜单名称"><Input autoFocus required value={taskMenuItemDraft.name} onChange={(event) => setTaskMenuItemDraft((current) => current ? {...current, name: event.target.value} : current)}/></Field>}
              </>}

              {taskMenuItemEditorTab === 'scripts' && taskMenuItemIsCommand && <>
                <div className="flex items-center justify-between">
                  <span className="font-display text-sm font-bold text-snap-ink">前置与后置脚本</span>
                  <SnapPopover open={Boolean(scriptHelpAnchor)} onOpenChange={(open) => { if (!open) setScriptHelpAnchor(undefined) }}>
                    <SnapPopoverAnchor asChild>
                      <SnapIconButton title="前后置脚本使用说明" aria-label="前后置脚本使用说明" size="sm" onClick={(event) => setScriptHelpAnchor(event.currentTarget as HTMLElement)}>
                        <HelpCircle className="h-4 w-4" strokeWidth={2.25}/>
                      </SnapIconButton>
                    </SnapPopoverAnchor>
                    <SnapPopoverContent align="end" className="w-[460px] p-4">
                      <div className="grid gap-2">
                        <span className="font-display text-sm font-bold text-snap-ink">前后置脚本参数</span>
                        <span className="text-sm text-snap-ink">脚本通过 UTF-8 JSON 标准输入接收主命令上下文：</span>
                        <pre className="m-0 overflow-x-auto rounded-snap-sm bg-snap-elevated p-2 text-xs">{'{\n  "taskId": "任务 ID",\n  "directory": "任务工作目录",\n  "command": "主命令",\n  "arguments": ["主命令参数"]\n}'}</pre>
                        <dl className="m-0 grid" style={{gridTemplateColumns: 'auto 1fr', columnGap: '0.25rem', rowGap: '0.125rem'}}>
                          <dt className="text-sm text-snap-ink"><code>taskId</code></dt><dd className="m-0 text-sm text-snap-ink">任务 ID</dd>
                          <dt className="text-sm text-snap-ink"><code>directory</code></dt><dd className="m-0 text-sm text-snap-ink">任务工作目录</dd>
                          <dt className="text-sm text-snap-ink"><code>command</code></dt><dd className="m-0 text-sm text-snap-ink">主命令</dd>
                          <dt className="text-sm text-snap-ink"><code>arguments</code></dt><dd className="m-0 text-sm text-snap-ink">主命令参数数组</dd>
                        </dl>
                        <span className="text-sm text-snap-ink">脚本填写路径或 Shell PATH 中的可执行脚本；参数每行传递为一个独立参数，空白行会忽略。</span>
                        <span className="text-sm text-snap-ink">不支持占位符替换；JSON 不会追加到命令行，也不会与参数拼接。</span>
                      </div>
                    </SnapPopoverContent>
                  </SnapPopover>
                </div>
                <Field label="前置脚本（命令或路径）" hint="填写脚本路径或 Shell PATH 中的可执行脚本。"><Input value={taskMenuItemDraft.beforeScript?.script ?? ''} onChange={(event) => updateTaskMenuItemScript(setTaskMenuItemDraft, 'beforeScript', {script: event.target.value})}/></Field>
                <Field label="前置脚本参数（每行一个）" hint="每行代表一个前置脚本参数。"><Textarea rows={2} value={(taskMenuItemDraft.beforeScript?.arguments ?? []).join('\n')} onChange={(event) => updateTaskMenuItemScript(setTaskMenuItemDraft, 'beforeScript', {arguments: event.target.value.split('\n')})}/></Field>
                <Field label="后置脚本（命令或路径）" hint="填写脚本路径或 Shell PATH 中的可执行脚本。"><Input value={taskMenuItemDraft.afterScript?.script ?? ''} onChange={(event) => updateTaskMenuItemScript(setTaskMenuItemDraft, 'afterScript', {script: event.target.value})}/></Field>
                <Field label="后置脚本参数（每行一个）" hint="每行代表一个后置脚本参数。"><Textarea rows={2} value={(taskMenuItemDraft.afterScript?.arguments ?? []).join('\n')} onChange={(event) => updateTaskMenuItemScript(setTaskMenuItemDraft, 'afterScript', {arguments: event.target.value.split('\n')})}/></Field>
              </>}
            </div>

            <SnapDialogFooter>
              {taskMenuItemEditorMode === 'edit' && taskMenuItemIsCommand && <SnapButton variant="danger" onClick={deleteTaskMenuItem}>删除菜单项</SnapButton>}
              <div className="flex-1"/>
              <SnapButton variant="secondary" onClick={closeTaskMenuItemEditor}>取消</SnapButton>
              <SnapButton type="submit" variant="primary">{taskMenuItemEditorMode === 'create' ? '添加菜单项' : '保存菜单项'}</SnapButton>
            </SnapDialogFooter>
          </form>
        </SnapDialogContent>
      </SnapDialog>}

			<SnapDialog open={taskDeletionOpen} onOpenChange={(open) => { if (!open) setTaskDeletionOpen(false) }}>
				<SnapDialogContent className="max-w-sm" showClose={false}>
					<SnapDialogHeader><SnapDialogTitle>删除 {selectedDeletableTaskIDs.length} 个任务记录？</SnapDialogTitle></SnapDialogHeader>
					<SnapDialogDescription>
						此操作只会移除任务记录，不会删除工作目录或运行生命周期命令。此操作无法撤销。
					</SnapDialogDescription>
					<SnapDialogFooter>
						<SnapButton variant="ghost" onClick={() => setTaskDeletionOpen(false)}>取消</SnapButton>
						<SnapButton variant="danger" onClick={() => void confirmDeleteTasks()}>删除记录</SnapButton>
					</SnapDialogFooter>
				</SnapDialogContent>
			</SnapDialog>

      <SnapDialog open={Boolean(finishTask)} onOpenChange={(open) => { if (!open) setFinishTask(undefined) }}>
        <SnapDialogContent className="max-w-sm" showClose={false}>
          <SnapDialogHeader><SnapDialogTitle>结束任务？</SnapDialogTitle></SnapDialogHeader>
          <SnapDialogDescription>
            确认后将结束“{finishTask?.title}”并关闭其全部终端。结束后命令链会按该任务的配置执行。此操作无法撤销。
          </SnapDialogDescription>
          <SnapDialogFooter>
            <SnapButton variant="ghost" onClick={() => setFinishTask(undefined)}>取消</SnapButton>
            <SnapButton variant="danger" onClick={() => void confirmFinishTask()}>结束任务</SnapButton>
          </SnapDialogFooter>
        </SnapDialogContent>
      </SnapDialog>

      <SnapDialog open={quitDialogOpen} onOpenChange={(open) => { if (!open) setQuitDialogOpen(false) }}>
        <SnapDialogContent className="max-w-sm" showClose={false}>
          <SnapDialogHeader><SnapDialogTitle>仍有执行中的任务</SnapDialogTitle></SnapDialogHeader>
          <SnapDialogDescription>
            退出会关闭全部终端，但不会改变任务状态或删除工作目录。下次启动后这些任务仍显示为执行中。
          </SnapDialogDescription>
          <SnapDialogFooter>
            <SnapButton variant="ghost" onClick={() => setQuitDialogOpen(false)}>取消</SnapButton>
            <SnapButton variant="primary" onClick={() => void confirmQuit()}>关闭终端并退出</SnapButton>
          </SnapDialogFooter>
        </SnapDialogContent>
      </SnapDialog>

      {message && (
        <SnapToast
          open
          onOpenChange={(open) => { if (!open) setMessage(undefined) }}
          duration={5000}
          variant={message.severity === 'error' ? 'error' : 'success'}
          role="alert"
          data-severity={message.severity}
        >
          <SnapToastTitle>{message.text}</SnapToastTitle>
          <SnapToastClose aria-label="关闭"/>
        </SnapToast>
      )}
      <SnapToastViewport/>
    </SnapTooltipProvider>
    </SnapToastProvider>
  )
}

function HTTPHelpSection({title, children}: {title: string; children: ReactNode}) {
  return (
    <section className="grid gap-2">
      <span className="font-display text-xs font-extrabold uppercase leading-tight tracking-wider text-snap-cobalt">{title}</span>
      {children}
    </section>
  )
}

function HTTPHelpStep({number, title, children}: {number: string; title: string; children: ReactNode}) {
  return (
    <div className="grid gap-2 rounded-snap border-2 border-snap-outline p-2" style={{gridTemplateColumns: '24px minmax(0, 1fr)'}}>
      <div className="grid h-6 w-6 place-items-center rounded-full bg-snap-cobalt text-xs font-extrabold text-white">{number}</div>
      <div className="grid gap-0.5">
        <span className="text-sm font-bold text-snap-ink">{title}</span>
        <span className="text-sm text-snap-muted">{children}</span>
      </div>
    </div>
  )
}

function HTTPEndpoint({method, path, children}: {method: 'GET' | 'PUT'; path: string; children: ReactNode}) {
  return (
    <div className="grid items-start gap-2 rounded-snap border-2 border-snap-outline p-2 sm:grid-cols-[auto_minmax(0,1fr)]">
      <SnapChip variant={method === 'GET' ? 'default' : 'info'} className="font-extrabold sm:w-[52px]">{method}</SnapChip>
      <div className="grid gap-1 min-w-0">
        <code className="break-words font-mono text-sm font-bold text-snap-ink">{`${method} ${path}`}</code>
        <span className="text-sm text-snap-muted">{children}</span>
      </div>
    </div>
  )
}

function HTTPCodeBlock({children}: {children: string}) {
  return <pre className="m-0 overflow-x-auto rounded-snap border-2 border-snap-outline bg-snap-elevated p-2 text-xs leading-relaxed">{children}</pre>
}

function StartupScreen() {
  return (
    <div
      role="status"
      aria-label="正在加载任务工作台"
      className="grid h-screen min-w-[720px] place-items-center bg-snap-canvas"
    >
      <div className="grid place-items-center gap-1.5 text-snap-muted">
        <CheckCircle2 className="h-10 w-10 text-snap-cobalt" strokeWidth={2.25}/>
        <p className="text-sm">正在加载任务工作台</p>
      </div>
    </div>
  )
}

function TaskLifecycleChainSelector({
	chains,
	presets,
	selected,
	presetID,
	onPresetChange,
	onChange,
	disabled = false,
}: {
	chains: LifecycleCommandChain[]
	presets: LifecyclePreset[]
	selected: Partial<Record<LifecycleHook, string>>
	presetID: string
	onPresetChange(presetID: string, chains: Partial<Record<LifecycleHook, string>>): void
	onChange(chains: Partial<Record<LifecycleHook, string>>): void
	disabled?: boolean
}) {
	return (
		<section className="grid gap-2 border-t-2 border-snap-outline pt-3">
			<div>
				<span className="font-display text-sm font-bold text-snap-ink">生命周期命令链</span>
				<span className="block text-xs text-snap-muted">预设会替换全部阶段；逐项调整后显示为自定义。{disabled ? ' 执行中和已完成任务不可修改。' : ''}</span>
			</div>
			<Field label="命令链预设">
				<select className={selectClass} value={presetID} disabled={disabled} aria-disabled={disabled ? 'true' : undefined} onChange={(event) => {
					const nextPresetID = event.target.value
					const preset = presets.find((current) => current.id === nextPresetID)
					onPresetChange(nextPresetID, {...preset?.chains ?? {}})
				}}>
					<option value="">不使用预设</option>
					{presets.map((preset) => <option key={preset.id} value={preset.id}>{preset.name}</option>)}
					{presetID === customLifecyclePresetID && <option value={customLifecyclePresetID} disabled>自定义</option>}
				</select>
			</Field>
			<div className="grid gap-2 sm:grid-cols-2">
				{lifecycleHooks.map((hook) => {
					const selectedChainID = selected[hook.id] ?? ''
					const applicableChains = chains.filter((chain) => lifecycleChainAppliesTo(chain, hook.id))
					const selectedChain = chains.find((chain) => chain.id === selectedChainID)
					const displayChains = selectedChain && !applicableChains.some((chain) => chain.id === selectedChain.id)
						? [...applicableChains, selectedChain]
						: applicableChains
					return <Field key={hook.id} label={hook.label}>
						<select className={selectClass} value={selectedChainID} disabled={disabled} aria-disabled={disabled ? 'true' : undefined} onChange={(event) => {
							const next = {...selected}
							if (event.target.value) {
								next[hook.id] = event.target.value
							} else {
								delete next[hook.id]
							}
							onChange(next)
						}}>
							<option value="">不执行命令链</option>
							{displayChains.map((chain) => <option key={chain.id} value={chain.id}>{chain.name}{!lifecycleChainAppliesTo(chain, hook.id) ? '（当前范围不适用）' : ''}</option>)}
						</select>
					</Field>
				})}
			</div>
		</section>
	)
}

function lifecyclePresetIDForChains(presets: LifecyclePreset[], chains: Partial<Record<LifecycleHook, string>>): string | undefined {
	return presets.find((preset) => lifecycleChainsEqual(preset.chains, chains))?.id
}

function lifecycleChainsEqual(left: Partial<Record<LifecycleHook, string>>, right: Partial<Record<LifecycleHook, string>>): boolean {
	return lifecycleHooks.every((hook) => (left[hook.id] ?? '') === (right[hook.id] ?? ''))
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
	presets,
	defaultPresetID,
	onSaveCommand,
	onDeleteCommand,
	onSaveChain,
	onCopyChain,
	onDeleteChain,
	onSavePreset,
	onCopyPreset,
	onDeletePreset,
	onSaveDefaultPreset,
	onError,
}: {
	commands: LifecycleCommand[]
	chains: LifecycleCommandChain[]
	presets: LifecyclePreset[]
	defaultPresetID: string
	onSaveCommand(command: LifecycleCommand): Promise<void>
	onDeleteCommand(commandID: string): Promise<void>
	onSaveChain(chain: LifecycleCommandChain): Promise<void>
	onCopyChain(chainID: string): Promise<void>
	onDeleteChain(chainID: string): Promise<void>
	onSavePreset(preset: LifecyclePreset): Promise<void>
	onCopyPreset(presetID: string): Promise<void>
	onDeletePreset(presetID: string): Promise<void>
	onSaveDefaultPreset(presetID: string): Promise<void>
	onError(error: unknown): void
}) {
	const [commandDraft, setCommandDraft] = useState<LifecycleCommand>()
	const [chainDraft, setChainDraft] = useState<LifecycleCommandChain>()
	const [presetDraft, setPresetDraft] = useState<LifecyclePreset>()
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

	const savePreset = async () => {
		if (!presetDraft) {
			return
		}
		try {
			await onSavePreset({
				...presetDraft,
				name: presetDraft.name.trim(),
				chains: Object.fromEntries(Object.entries(presetDraft.chains).filter(([, chainID]) => Boolean(chainID))) as Partial<Record<LifecycleHook, string>>,
			})
			setPresetDraft(undefined)
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

	return <section className="grid gap-5">
		<div className="grid gap-2">
			<div className="flex items-center justify-between gap-2 flex-wrap">
				<div>
					<span className="font-display text-sm font-bold text-snap-ink">命令链预设</span>
					<span className="block text-xs text-snap-muted">新建任务可一键套用预设；任务创建后保留独立的链选择。</span>
				</div>
				<SnapButton variant="primary" size="sm" onClick={() => setPresetDraft({id: '', name: '', chains: {}})}>新增预设</SnapButton>
			</div>
			<Field label="默认预设">
				<select className={selectClass} value={defaultPresetID} onChange={(event) => void onSaveDefaultPreset(event.target.value).catch(onError)}>
					<option value="">不设置默认预设</option>
					{presets.map((preset) => <option key={preset.id} value={preset.id}>{preset.name}</option>)}
				</select>
			</Field>
			<div className="overflow-hidden rounded-snap border-2 border-snap-outline divide-y divide-snap-outline">
				{presets.length === 0 && <span className="block px-3 py-2 text-sm text-snap-muted">尚无命令链预设。</span>}
				{presets.map((preset) => <div key={preset.id} className="grid gap-2 items-center px-3 py-2 sm:grid-cols-[minmax(0,1fr)_auto]">
					<div className="min-w-0">
						<div className="flex items-center gap-1.5 min-w-0">
							<span className="text-sm font-bold text-snap-ink truncate">{preset.name}</span>
							{preset.id === defaultPresetID && <SnapChip variant="info">默认</SnapChip>}
						</div>
						<span className="block text-xs text-snap-muted truncate">{lifecycleHooks.filter((hook) => preset.chains[hook.id]).map((hook) => `${hook.label}: ${chains.find((chain) => chain.id === preset.chains[hook.id])?.name ?? '已删除命令链'}`).join('；') || '不执行命令链'}</span>
					</div>
					<div className="flex justify-end gap-1 flex-wrap">
						<SnapButton variant="secondary" size="sm" aria-label={`编辑预设 ${preset.name}`} onClick={() => setPresetDraft({...preset, chains: {...preset.chains}})}>编辑</SnapButton>
						<SnapButton variant="secondary" size="sm" onClick={() => void onCopyPreset(preset.id).catch(onError)}>复制</SnapButton>
						<SnapIconButton title={`删除预设 ${preset.name}`} aria-label={`删除预设 ${preset.name}`} size="sm" onClick={() => void onDeletePreset(preset.id).catch(onError)} className="text-snap-error hover:text-snap-error"><Trash2 className="h-4 w-4" strokeWidth={2.25}/></SnapIconButton>
					</div>
				</div>)}
			</div>
		</div>

		<div className="grid gap-2 pt-1 border-t-2 border-snap-outline">
			<div className="flex items-center justify-between gap-2">
				<div><span className="font-display text-sm font-bold text-snap-ink">命令库</span><span className="block text-xs text-snap-muted">系统目录命令只读；自定义命令通过所选 Shell 执行。</span></div>
				<SnapButton variant="primary" size="sm" onClick={() => setCommandDraft({id: '', kind: 'custom', name: '', command: '', arguments: [], chainArgumentMode: 'disabled', applicableHooks: []})}>新增命令</SnapButton>
			</div>
			<div className="overflow-hidden rounded-snap border-2 border-snap-outline divide-y divide-snap-outline">
				{commands.map((command) => <div key={command.id} className="grid gap-2 items-center px-3 py-2" style={{gridTemplateColumns: 'minmax(0, 1fr) auto'}}>
					<div className="min-w-0"><span className="block text-sm font-bold text-snap-ink truncate">{command.name}</span><span className="block text-xs text-snap-muted truncate">{command.kind === 'custom' ? `${command.command}${command.arguments.length ? ` ${command.arguments.join(' ')}` : ''}` : '系统内置命令'}</span>{command.documentation && <span className="block text-xs text-snap-muted" style={{whiteSpace: 'pre-wrap'}}>{command.documentation}</span>}<span className="block text-xs text-snap-muted">适用：{lifecycleHooksLabel(command.applicableHooks)}</span></div>
					<div className="flex items-center gap-1">
						<SnapChip>{command.kind === 'custom' ? '自定义' : '系统'}</SnapChip>
						{command.kind === 'custom' && <SnapButton variant="secondary" size="sm" aria-label={`编辑命令 ${command.name}`} onClick={() => setCommandDraft({...command, arguments: [...command.arguments], chainArgumentMode: command.chainArgumentMode === 'disabled' ? 'disabled' : 'enabled', applicableHooks: [...command.applicableHooks]})}>编辑</SnapButton>}
						{command.kind === 'custom' && <SnapIconButton title={`删除命令 ${command.name}`} aria-label={`删除命令 ${command.name}`} size="sm" onClick={() => void onDeleteCommand(command.id).catch(onError)} className="text-snap-error hover:text-snap-error"><Trash2 className="h-4 w-4" strokeWidth={2.25}/></SnapIconButton>}
					</div>
				</div>)}
			</div>
		</div>

		<div className="grid gap-2 pt-1 border-t-2 border-snap-outline">
			<div className="flex items-center justify-between gap-2">
				<div><span className="font-display text-sm font-bold text-snap-ink">命令链</span><span className="block text-xs text-snap-muted">同一条链可被多个钩子和任务复用，修改将在下次执行或重试时生效。</span></div>
				<div className="flex items-center gap-1.5 flex-wrap justify-end">
					<SnapButton variant="secondary" size="sm" onClick={() => setChainHelpOpen(true)}>查看命令链扩展规则</SnapButton>
					<SnapButton variant="primary" size="sm" onClick={() => setChainDraft({id: '', name: '', commands: [], applicableHooks: []})}>新增链</SnapButton>
				</div>
			</div>
			<div className="overflow-hidden rounded-snap border-2 border-snap-outline divide-y divide-snap-outline">
				{chains.map((chain) => <div key={chain.id} className="grid gap-2 items-center px-3 py-2 sm:grid-cols-[minmax(0,1fr)_auto]">
					<div className="min-w-0"><span className="block text-sm font-bold text-snap-ink truncate">{chain.name}</span><span className="block text-xs text-snap-muted truncate">{chain.commands.map((reference) => commandNames.get(reference.commandId) ?? '已删除命令').join(' → ')}</span><span className="block text-xs text-snap-muted">适用：{lifecycleHooksLabel(chain.applicableHooks)}</span></div>
					<div className="flex justify-end gap-1 flex-wrap"><SnapButton variant="secondary" size="sm" aria-label={`编辑命令链 ${chain.name}`} onClick={() => setChainDraft({...chain, commands: chain.commands.map((reference) => ({...reference, arguments: [...reference.arguments]})), applicableHooks: [...chain.applicableHooks]})}>编辑</SnapButton><SnapButton variant="secondary" size="sm" onClick={() => void onCopyChain(chain.id).catch(onError)}>复制</SnapButton><SnapIconButton title={`删除命令链 ${chain.name}`} aria-label={`删除命令链 ${chain.name}`} size="sm" onClick={() => void onDeleteChain(chain.id).catch(onError)} className="text-snap-error hover:text-snap-error"><Trash2 className="h-4 w-4" strokeWidth={2.25}/></SnapIconButton></div>
				</div>)}
			</div>
		</div>

		<SnapDialog open={Boolean(commandDraft)} onOpenChange={(open) => { if (!open) setCommandDraft(undefined) }}>
			<SnapDialogContent className="max-w-md">
				<SnapDialogHeader><SnapDialogTitle>{commandDraft?.id ? '编辑命令' : '新增命令'}</SnapDialogTitle></SnapDialogHeader>
				<SnapScrollArea className="max-h-[55vh]">
					<div className="grid gap-3 px-1">
						<Field label="命令名称"><Input autoFocus required value={commandDraft?.name ?? ''} onChange={(event) => setCommandDraft((current) => current ? {...current, name: event.target.value} : current)}/></Field>
						<Field label="可执行命令" hint="由任务设置中选定的 Shell 启动。"><Input required value={commandDraft?.command ?? ''} onChange={(event) => setCommandDraft((current) => current ? {...current, command: event.target.value} : current)}/></Field>
						<Field label="固定参数（每行一个）"><Textarea rows={3} value={commandDraft?.arguments.join('\n') ?? ''} onChange={(event) => setCommandDraft((current) => current ? {...current, arguments: event.target.value.split('\n')} : current)}/></Field>
						<label className="flex items-center gap-2 text-sm text-snap-ink"><SnapCheckbox checked={commandDraft?.chainArgumentMode === 'enabled'} onCheckedChange={(checked) => setCommandDraft((current) => current ? {...current, chainArgumentMode: checked ? 'enabled' : 'disabled'} : current)}/><span>允许在命令链中追加参数</span></label>
						<div className="grid gap-1"><span className="font-display text-sm font-bold text-snap-ink">适用范围</span>{lifecycleHooks.map((hook) => <label key={hook.id} className="flex items-center gap-2 text-sm text-snap-ink"><SnapCheckbox checked={Boolean(commandDraft?.applicableHooks.includes(hook.id))} onCheckedChange={(checked) => toggleApplicableHook(hook.id, 'command', checked === true)}/><span>{hook.label}</span></label>)}</div>
					</div>
				</SnapScrollArea>
				<SnapDialogFooter><SnapButton variant="secondary" onClick={() => setCommandDraft(undefined)}>取消</SnapButton><SnapButton variant="primary" disabled={!commandDraft?.applicableHooks.length} onClick={() => void saveCommand()}>保存命令</SnapButton></SnapDialogFooter>
			</SnapDialogContent>
		</SnapDialog>

		<SnapDialog open={Boolean(chainDraft)} onOpenChange={(open) => { if (!open) setChainDraft(undefined) }}>
			<SnapDialogContent className="max-w-md">
				<SnapDialogHeader><SnapDialogTitle>{chainDraft?.id ? '编辑命令链' : '新增命令链'}</SnapDialogTitle></SnapDialogHeader>
				<SnapScrollArea className="max-h-[55vh]">
					<div className="grid gap-3 px-1">
						<Field label="命令链名称"><Input autoFocus required value={chainDraft?.name ?? ''} onChange={(event) => setChainDraft((current) => current ? {...current, name: event.target.value} : current)}/></Field>
						<div className="grid gap-1"><span className="font-display text-sm font-bold text-snap-ink">适用范围</span>{lifecycleHooks.map((hook) => <label key={hook.id} className="flex items-center gap-2 text-sm text-snap-ink"><SnapCheckbox checked={Boolean(chainDraft?.applicableHooks.includes(hook.id))} onCheckedChange={(checked) => toggleApplicableHook(hook.id, 'chain', checked === true)}/><span>{hook.label}</span></label>)}</div>
						<div className="grid gap-1"><span className="font-display text-sm font-bold text-snap-ink">可用命令</span>{chainDraft?.applicableHooks.length ? commands.filter((command) => lifecycleHooksCover(command.applicableHooks, chainDraft.applicableHooks) || chainDraft.commands.some((reference) => reference.commandId === command.id)).map((command) => <label key={command.id} className="flex items-center gap-2 text-sm text-snap-ink"><SnapCheckbox checked={Boolean(chainDraft.commands.some((reference) => reference.commandId === command.id))} onCheckedChange={(checked) => toggleChainCommand(command.id, checked === true)}/><span>{`${command.name}${lifecycleHooksCover(command.applicableHooks, chainDraft.applicableHooks) ? '' : '（与当前范围不匹配）'}`}</span></label>) : <span className="text-sm text-snap-muted">请先选择命令链适用范围。</span>}</div>
						{incompatibleChainCommands.length > 0 && <SnapAlert severity="error">请移除与当前范围不匹配的命令后再保存。</SnapAlert>}
						<div className="grid gap-1">
							<span className="font-display text-sm font-bold text-snap-ink">执行顺序</span>
							{chainDraft?.commands.length ? chainDraft.commands.map((reference, index) => {
								const command = commandsByID.get(reference.commandId)
								return (
								<div
									key={`${reference.commandId}-${index}`}
									draggable
									onDragStart={(event) => event.dataTransfer.setData('text/plain', String(index))}
									onDragOver={(event) => event.preventDefault()}
									onDrop={(event) => {
										const fromIndex = Number(event.dataTransfer.getData('text/plain'))
										moveChainCommandTo(fromIndex, index)
									}}
									className="grid gap-1 items-center rounded-snap border-2 border-snap-outline p-2 cursor-grab active:cursor-grabbing"
									style={{gridTemplateColumns: 'minmax(0, 1fr) auto auto'}}
								>
									<span className="text-sm text-snap-ink">{index + 1}. {commandNames.get(reference.commandId) ?? '已删除命令'}</span>
									<SnapIconButton title={`上移链命令 ${index + 1}`} aria-label={`上移链命令 ${index + 1}`} size="sm" disabled={index === 0} onClick={() => moveChainCommand(index, -1)}><ArrowUp className="h-4 w-4" strokeWidth={2.25}/></SnapIconButton>
									<SnapIconButton title={`下移链命令 ${index + 1}`} aria-label={`下移链命令 ${index + 1}`} size="sm" disabled={index === chainDraft.commands.length - 1} onClick={() => moveChainCommand(index, 1)}><ArrowDown className="h-4 w-4" strokeWidth={2.25}/></SnapIconButton>
									{lifecycleCommandAllowsChainArguments(command) && <div style={{gridColumn: '1 / -1'}}><Field label={`${command?.name ?? '命令'} 追加参数（每行一个）`}><Textarea rows={2} value={reference.arguments.join('\n')} onChange={(event) => updateChainCommandArguments(index, event.target.value.split('\n'))}/></Field></div>}
									{command?.documentation && <span className="text-xs text-snap-muted" style={{gridColumn: '1 / -1', whiteSpace: 'pre-wrap'}}>{command.documentation}</span>}
								</div>
								)
							}) : <span className="text-sm text-snap-muted">至少选择一个命令。</span>}
						</div>
					</div>
				</SnapScrollArea>
				<SnapDialogFooter><SnapButton variant="secondary" onClick={() => setChainDraft(undefined)}>取消</SnapButton><SnapButton variant="primary" disabled={!chainDraft?.applicableHooks.length || !chainDraft.commands.length || incompatibleChainCommands.length > 0} onClick={() => void saveChain()}>保存命令链</SnapButton></SnapDialogFooter>
			</SnapDialogContent>
		</SnapDialog>

		<SnapDialog open={Boolean(presetDraft)} onOpenChange={(open) => { if (!open) setPresetDraft(undefined) }}>
			<SnapDialogContent className="max-w-md">
				<SnapDialogHeader><SnapDialogTitle>{presetDraft?.id ? '编辑命令链预设' : '新增命令链预设'}</SnapDialogTitle></SnapDialogHeader>
				<SnapScrollArea className="max-h-[55vh]">
					<div className="grid gap-3 px-1">
						<Field label="预设名称"><Input autoFocus required value={presetDraft?.name ?? ''} onChange={(event) => setPresetDraft((current) => current ? {...current, name: event.target.value} : current)}/></Field>
						<div className="grid gap-2 sm:grid-cols-2">
							{lifecycleHooks.map((hook) => {
								const chainID = presetDraft?.chains[hook.id] ?? ''
								const applicableChains = chains.filter((chain) => lifecycleChainAppliesTo(chain, hook.id))
								const selectedChain = chains.find((chain) => chain.id === chainID)
								const displayChains = selectedChain && !applicableChains.some((chain) => chain.id === selectedChain.id) ? [...applicableChains, selectedChain] : applicableChains
								return <Field key={hook.id} label={hook.label}>
									<select className={selectClass} value={chainID} onChange={(event) => setPresetDraft((current) => {
										if (!current) {
											return current
										}
										const nextChains = {...current.chains}
										if (event.target.value) {
											nextChains[hook.id] = event.target.value
										} else {
											delete nextChains[hook.id]
										}
										return {...current, chains: nextChains}
									})}>
										<option value="">不执行命令链</option>
										{displayChains.map((chain) => <option key={chain.id} value={chain.id}>{chain.name}{!lifecycleChainAppliesTo(chain, hook.id) ? '（当前范围不适用）' : ''}</option>)}
									</select>
								</Field>
							})}
						</div>
					</div>
				</SnapScrollArea>
				<SnapDialogFooter><SnapButton variant="secondary" onClick={() => setPresetDraft(undefined)}>取消</SnapButton><SnapButton variant="primary" disabled={!presetDraft?.name.trim()} onClick={() => void savePreset()}>保存预设</SnapButton></SnapDialogFooter>
			</SnapDialogContent>
		</SnapDialog>

		<SnapDialog open={chainHelpOpen} onOpenChange={(open) => { if (!open) setChainHelpOpen(false) }}>
			<SnapDialogContent className="max-w-2xl" showClose={false}>
				<SnapDialogHeader><SnapDialogTitle>命令链扩展规则</SnapDialogTitle></SnapDialogHeader>
				<SnapScrollArea className="max-h-[62vh]">
					<div className="grid gap-4 px-1 py-1">
						<section className="grid gap-1">
							<span className="font-display text-sm font-bold text-snap-ink">适用范围与顺序</span>
							<span className="text-sm text-snap-muted">命令链按配置顺序执行，可用于开始前、开始后、结束前、结束后和更新任务后五个钩子。链中的每条命令都必须覆盖该链选择的全部适用范围。</span>
						</section>
						<section className="grid gap-1">
							<span className="font-display text-sm font-bold text-snap-ink">参数传递</span>
							<span className="text-sm text-snap-muted">每行配置一个独立参数。固定参数会先于命令链追加参数传入。关闭“允许在命令链中追加参数”只会隐藏编辑框，不会删除已经保存的追加参数。</span>
						</section>
						<section className="grid gap-1">
							<span className="font-display text-sm font-bold text-snap-ink">标准输入与输出</span>
							<span className="text-sm text-snap-muted">首条自定义命令接收任务详情 JSON；本机 HTTP 服务正在监听时，顶层 <code>baseURL</code> 是当前 API 基址，只通过标准输入提供，不是终端环境变量。当前任务详情中的“复制当前命令链输入 JSON”可用于取得此刻的首段快照。后续自定义命令接收前一条自定义命令的原始标准输出，系统不会自动解析、合并或补回 JSON。</span>
						</section>
						<section className="grid gap-1">
							<span className="font-display text-sm font-bold text-snap-ink">环境变量</span>
							<span className="text-sm text-snap-muted">应用向任务关联的命令与脚本注入 <code>TASKAI_TASK_ID</code>；新建终端还会获得 <code>TASKAI_TERMINAL_ID</code>，本机 HTTP 服务正在监听时额外获得 <code>TASKAI_STATUS_API</code>。当前任务模板中勾选环境变量注入的字段仅传给自定义生命周期 Shell 命令。</span>
						</section>
						<section className="grid gap-1">
							<span className="font-display text-sm font-bold text-snap-ink">内置命令</span>
							<span className="text-sm text-snap-muted">创建和删除任务工作目录由应用直接执行。Git 仓库克隆的追加参数可留空，或仅填写一行 <code>dir=&lt;相对目录&gt;</code>；目录必须保持在任务工作目录内。</span>
						</section>
						<section className="grid gap-1">
							<span className="font-display text-sm font-bold text-snap-ink">失败与重试</span>
							<span className="text-sm text-snap-muted">链执行失败会保留错误和进度，重试始终从链首重新执行，并读取最新的链定义。</span>
						</section>
					</div>
				</SnapScrollArea>
				<SnapDialogFooter><SnapButton variant="secondary" onClick={() => setChainHelpOpen(false)}>关闭</SnapButton></SnapDialogFooter>
			</SnapDialogContent>
		</SnapDialog>
	</section>
}

function TaskTemplateManagement({
	templates,
	activeTemplateID,
	activeTaskTemplateIDs,
	hasLegacyActiveTaskTemplateFields,
	onChange,
}: {
	templates: TaskTemplate[]
	activeTemplateID: string
	activeTaskTemplateIDs: ReadonlySet<string>
	hasLegacyActiveTaskTemplateFields: boolean
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

	const deletionBlockReason = (templateID: string) => {
		if (hasLegacyActiveTaskTemplateFields) {
			return '未执行或执行中的旧任务包含无法归属的模板字段，完成这些任务后才能删除模板'
		}
		if (activeTaskTemplateIDs.has(templateID)) {
			return '未执行或执行中的任务正在使用此模板，暂不能删除'
		}
		return undefined
	}

	return <section className="grid gap-3">
		<div className="flex items-center justify-between gap-3">
			<div>
				<span className="font-display text-sm font-bold text-snap-ink">任务模板</span>
				<span className="block text-xs text-snap-muted">当前模板决定新建、编辑、HTTP 和生命周期命令链可见的字段。</span>
			</div>
			<SnapButton variant="primary" size="sm" onClick={() => {
				setDraft(createTaskTemplateDraft())
				setDraftError(undefined)
			}}><Plus className="h-4 w-4" strokeWidth={2.25}/>新增模板</SnapButton>
		</div>
		<Field label="当前任务模板">
			<select className={selectClass} value={activeTemplateID} onChange={(event) => onChange(templates, event.target.value)}>
				<option value="">不使用任务模板</option>
				{templates.map((template) => <option key={template.id} value={template.id}>{template.name}</option>)}
			</select>
		</Field>
		{templates.length === 0 ? <SnapAlert severity="info">暂无任务模板。新建模板后可选择为当前模板。</SnapAlert> : <div className="overflow-hidden rounded-snap border-2 border-snap-outline divide-y divide-snap-outline">
			{templates.map((template) => {
				const blockedReason = deletionBlockReason(template.id)
				return <div key={template.id} className="flex items-center gap-2 px-3 py-2">
					<div className="min-w-0 flex-1">
						<span className="block text-sm font-bold text-snap-ink truncate">{template.name}</span>
						<span className="block text-xs text-snap-muted">{template.fields.length} 个字段{template.id === activeTemplateID ? ' · 当前使用' : ''}</span>
					</div>
					<SnapButton variant="secondary" size="sm" onClick={() => {
						setDraft(cloneTaskTemplate(template))
						setDraftError(undefined)
					}}>编辑</SnapButton>
					<SnapIconButton title={blockedReason ?? '删除模板'} aria-label={`删除任务模板 ${template.name}`} size="sm" disabled={Boolean(blockedReason)} onClick={() => onChange(templates.filter((current) => current.id !== template.id), activeTemplateID === template.id ? '' : activeTemplateID)} className="text-snap-error hover:text-snap-error"><Trash2 className="h-4 w-4" strokeWidth={2.25}/></SnapIconButton>
				</div>
			})}
		</div>}

		<SnapDialog open={Boolean(draft)} onOpenChange={(open) => { if (!open) setDraft(undefined) }}>
			<SnapDialogContent className="max-w-2xl">
				<SnapDialogHeader><SnapDialogTitle>{templates.some((template) => template.id === draft?.id) ? '编辑任务模板' : '新增任务模板'}</SnapDialogTitle></SnapDialogHeader>
				<SnapScrollArea className="max-h-[62vh]">
					<div className="grid gap-4 px-1">
						{draftError && <SnapAlert severity="error">{draftError}</SnapAlert>}
						<Field label="模板名称"><Input required autoFocus value={draft?.name ?? ''} onChange={(event) => setDraft((current) => current ? {...current, name: event.target.value} : current)}/></Field>
						<div className="flex items-center justify-between gap-3">
							<div>
								<span className="font-display text-sm font-bold text-snap-ink">字段</span>
								<span className="block text-xs text-snap-muted">字段顺序决定任务表单、HTTP 字段和命令环境变量的显示顺序。</span>
							</div>
							<SnapButton variant="secondary" size="sm" onClick={() => setDraft((current) => current ? {...current, fields: [...current.fields, createTaskTemplateField()]} : current)}><Plus className="h-4 w-4" strokeWidth={2.25}/>新增字段</SnapButton>
						</div>
						<div className="grid gap-2">
							{draft?.fields.map((field, index) => <div key={index} className="grid gap-2 items-center rounded-snap border-2 border-snap-outline p-2" style={{gridTemplateColumns: 'minmax(110px, 1fr) minmax(110px, 1fr) 126px auto'}}>
								<Field label={`字段 ${index + 1} 键`}><Input required value={field.key} onChange={(event) => updateField(index, {key: event.target.value})}/></Field>
								<Field label={`字段 ${index + 1} 显示名称`}><Input required value={field.displayName} onChange={(event) => updateField(index, {displayName: event.target.value})}/></Field>
								<Field label={`字段 ${index + 1} 类型`}>
									<select className={selectClass} value={field.inputType} onChange={(event) => {
										const inputType: TaskTemplateField['inputType'] = event.target.value === 'bool' ? 'bool' : 'string'
										updateField(index, {inputType, defaultValue: inputType === 'bool' ? false : ''})
									}}>
										<option value="string">字符串</option>
										<option value="bool">布尔值</option>
									</select>
								</Field>
								<div className="flex items-center gap-0.5">
									<SnapIconButton title="上移字段" aria-label={`上移模板字段 ${index + 1}`} size="sm" disabled={index === 0} onClick={() => moveField(index, -1)}><ArrowUp className="h-4 w-4" strokeWidth={2.25}/></SnapIconButton>
									<SnapIconButton title="下移字段" aria-label={`下移模板字段 ${index + 1}`} size="sm" disabled={index === (draft?.fields.length ?? 0) - 1} onClick={() => moveField(index, 1)}><ArrowDown className="h-4 w-4" strokeWidth={2.25}/></SnapIconButton>
									<SnapIconButton title="删除字段" aria-label={`删除模板字段 ${index + 1}`} size="sm" disabled={(draft?.fields.length ?? 0) === 1} onClick={() => setDraft((current) => current ? {...current, fields: current.fields.filter((_, fieldIndex) => fieldIndex !== index)} : current)} className="text-snap-error hover:text-snap-error"><Trash2 className="h-4 w-4" strokeWidth={2.25}/></SnapIconButton>
								</div>
								<div className="flex flex-wrap items-center gap-2" style={{gridColumn: '1 / -1'}}>
									{field.inputType === 'bool'
										? <label className="flex items-center gap-2 text-sm text-snap-ink"><SnapCheckbox checked={field.defaultValue === true} onCheckedChange={(checked) => updateField(index, {defaultValue: checked})}/><span>默认选中</span></label>
										: <Field label={`字段 ${index + 1} 默认值`} className="min-w-[220px] flex-1"><Input value={typeof field.defaultValue === 'string' ? field.defaultValue : ''} onChange={(event) => updateField(index, {defaultValue: event.target.value})}/></Field>}
									<label className="flex items-center gap-2 text-sm text-snap-ink"><SnapCheckbox checked={field.required} onCheckedChange={(checked) => updateField(index, {required: checked === true})}/><span>{field.inputType === 'bool' ? '必填（必须勾选）' : '必填'}</span></label>
									<label className="flex items-center gap-2 text-sm text-snap-ink"><SnapCheckbox checked={field.injectEnvironment} onCheckedChange={(checked) => updateField(index, {injectEnvironment: checked === true})}/><span>注入生命周期环境变量</span></label>
								</div>
							</div>)}
						</div>
					</div>
				</SnapScrollArea>
				<SnapDialogFooter><SnapButton variant="secondary" onClick={() => setDraft(undefined)}>取消</SnapButton><SnapButton variant="primary" onClick={saveDraft}>保存模板</SnapButton></SnapDialogFooter>
			</SnapDialogContent>
		</SnapDialog>
	</section>
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
	return <section data-testid="task-template-fields" className="grid gap-2 border-t-2 border-snap-outline pt-3">
		<span className="font-display text-sm font-bold text-snap-ink">{template.name}</span>
		<div className="grid gap-2 sm:grid-cols-2">
			{template.fields.map((field) => field.inputType === 'bool'
				? <label key={field.key} className="flex items-center gap-2 text-sm text-snap-ink min-w-0"><SnapCheckbox checked={values[field.key] === true} onCheckedChange={(checked) => onChange((current) => ({...current, [field.key]: checked}))}/><span>{field.displayName}</span></label>
				: <Field key={field.key} label={field.displayName}><Input required={field.required} value={typeof values[field.key] === 'string' ? (values[field.key] as string) : ''} onChange={(event) => onChange((current) => ({...current, [field.key]: event.target.value}))}/></Field>,
			)}
		</div>
	</section>
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
		<section className="grid gap-2 border-t-2 border-snap-outline pt-3">
			<div>
				<span className="font-display text-sm font-bold text-snap-ink">附加信息</span>
				<span className="block text-xs text-snap-muted">选择信息后填写其动态参数。</span>
			</div>
			<div className="grid gap-2">
				<span className="text-xs text-snap-muted">已选择 {extraInfo.length} 项</span>
				{extraInfo.length > 0 && <div className="flex flex-wrap gap-2">
					{extraInfo.map((item) => {
						const name = item.displayName ?? '已保存信息'
						return (
							<SnapChip key={taskExtraInfoKey(item)}>{name}
								<button type="button" aria-label={`移除 ${name}`} onClick={() => onChange((current) => current.filter((currentItem) => taskExtraInfoKey(currentItem) !== taskExtraInfoKey(item)))} className="ml-1 inline-grid h-4 w-4 place-items-center rounded-snap-sm hover:bg-snap-surface-2">×</button>
							</SnapChip>
						)
					})}
				</div>}
			</div>
			{catalogues.length === 0 && deletedSnapshots.length === 0 && (
				<span className="text-sm text-snap-muted">暂无可选额外信息，可通过顶部“额外信息管理”添加。</span>
			)}
			{catalogues.length > 0 && <>
				<Field label="选择分类">
					<select className={selectClass} value={selectedCatalogue} onChange={(event) => setSelectedCatalogue(event.target.value)}>
						{catalogues.map((catalogue) => <option key={catalogue} value={catalogue}>{catalogue}</option>)}
					</select>
				</Field>
				<Field label="搜索信息"><Input placeholder="按名称模糊搜索" value={informationSearch} onChange={(event) => setInformationSearch(event.target.value)}/></Field>
				<div className="grid gap-2 rounded-snap border-2 border-snap-outline p-2 max-h-[300px] overflow-y-auto snap-no-scrollbar">
					{visibleInfos.length === 0 ? (
						<span className="text-sm text-snap-muted">该分类未找到可选信息。</span>
					) : visibleInfos.map((info) => {
						const selected = selectedByInformationID.get(info.id)
						return (
							<div key={info.id} className="grid gap-2">
								<label className="flex items-center gap-2 text-sm text-snap-ink">
									<SnapCheckbox checked={Boolean(selected)} onCheckedChange={(checked) => toggleInformation(info, checked === true)}/>
									<span>{extraInfoName(info)}</span>
								</label>
								{selected && <TaskExtraInfoSnapshotFields item={selected} onChange={updateParameter}/>}
							</div>
						)
					})}
				</div>
			</>}
			{deletedSnapshots.map((item) => (
				<div key={taskExtraInfoKey(item)} className="grid gap-2 rounded-snap border-2 border-snap-amber p-2">
					<div className="flex items-center justify-between gap-2">
						<span className="text-sm font-bold text-snap-ink">{item.displayName ?? '信息'}（已删除）</span>
					</div>
					<TaskExtraInfoSnapshotFields item={item} onChange={updateParameter}/>
				</div>
			))}
		</section>
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
		<div className="grid gap-2 pl-0 sm:pl-1 min-w-0">
			{item.parameters.map((parameter, parameterIndex) => {
				const inputType = extraInfoParameterInputType(parameter)
				return <div key={`parameter-${parameterIndex}`} data-testid={`task-extra-info-parameter-${itemKey}-${parameterIndex}`} className="grid gap-2 min-w-0 border-t-2 border-snap-outline py-2" style={{borderTopWidth: '2px', borderTopStyle: 'solid', borderTopColor: 'var(--snap-outline)'}}>
					{inputType === 'checkbox'
						? <label className="flex items-center gap-2 text-sm text-snap-ink min-w-0"><SnapCheckbox checked={parameter.value === 'true'} onCheckedChange={(checked) => onChange(itemKey, parameterIndex, {value: checked ? 'true' : 'false'})}/><span>{parameter.displayName || '动态参数'}</span></label>
						: <Field label={parameter.displayName || '动态参数'}><Input value={parameter.value} required={parameter.required} onChange={(event) => onChange(itemKey, parameterIndex, {value: event.target.value})}/></Field>
					}
				</div>
			})}
		</div>
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
      <div className="grid h-full place-items-center p-6 text-center text-snap-muted">
        <div className="grid max-w-[360px] gap-2 rounded-snap border-2 border-snap-outline/25 bg-snap-surface px-8 py-7 shadow-snap-sm">
          <FolderOpen className="h-[42px] w-[42px] justify-self-center text-snap-muted" strokeWidth={2.25}/>
          <p className="text-snap-ink">从左侧选择任务，或创建一个新任务开始。</p>
        </div>
      </div>
    )
  }
	const templateValues = resolveTaskTemplateValues(template, task.templateFields)
  return (
    <div className="taskai-task-detail p-6 md:p-10" style={{height: '100%', width: '100%', maxWidth: 'none', overflow: 'auto'}}>
      <div className="taskai-task-detail__heading mb-4 flex items-center gap-2">
        <h2 className="font-display text-2xl font-extrabold text-snap-ink">{task.title}</h2>
        <SnapChip>{taskStatusLabel[task.status]}</SnapChip>
      </div>
      <p className={`mb-6 whitespace-pre-wrap ${task.description ? 'text-snap-ink' : 'text-snap-muted'}`}>
        {task.description || '暂无任务描述'}
      </p>
		<section className="taskai-detail-section mb-5 grid gap-2">
			<span className="taskai-detail-section__title font-display text-xs font-extrabold uppercase tracking-wide text-snap-muted">任务模板</span>
			{template ? <div className="taskai-detail-card grid gap-2 rounded-snap border-2 border-snap-outline/25 p-3">
				<span className="font-display text-sm font-bold text-snap-ink">{template.name}</span>
				{template.fields.map((field) => {
					const value = taskTemplateFieldDisplayValue(templateValues[field.key])
					const environment = field.injectEnvironment ? `TASKAI_${field.key.toUpperCase()}=${value}` : undefined
					return <div key={field.key} className="grid gap-1 border-t-2 border-snap-outline/15 pt-2">
						<TaskDetailValue label={field.displayName || field.key} value={value}/>
						<span className="text-xs text-snap-muted">字段键：<code>{field.key}</code></span>
						<span className="text-xs text-snap-muted">{environment ? <>生成环境变量 <code>{environment}</code></> : '不生成环境变量'}</span>
						{environment && <span className="text-xs text-snap-muted">仅自定义生命周期 Shell 命令</span>}
					</div>
				})}
			</div> : <SnapAlert severity="info">未启用任务模板</SnapAlert>}
		</section>
		<section className="taskai-detail-section mb-5 grid gap-2">
			<span className="taskai-detail-section__title font-display text-xs font-extrabold uppercase tracking-wide text-snap-muted">额外信息</span>
			{task.extraInfo?.length ? Object.entries(groupTaskExtraInfoByCatalogue(task.extraInfo)).map(([catalogue, items]) => (
				<div className="taskai-detail-card grid gap-2 rounded-snap border-2 border-snap-outline/25 p-3" key={catalogue}>
					<span className="font-display text-sm font-bold text-snap-ink">{catalogue}</span>
					{items.map((item, itemIndex) => (
						<div key={item.id || `${catalogue}-${itemIndex}`} className={`grid gap-2 pt-3${itemIndex === 0 ? '' : ' border-t-2 border-snap-outline/15'}`}>
							<span className="text-sm font-bold text-snap-ink">{item.displayName || item.fields.find((field) => field.key === 'name')?.value || catalogue}</span>
							<div className="grid grid-cols-1 gap-2 sm:grid-cols-2">
								{item.fields.map((field) => <TaskDetailValue key={`field-${field.key}`} label={field.displayName || field.key} value={field.value || '—'}/>) }
								{item.parameters.map((parameter) => <TaskDetailValue key={`parameter-${parameter.key}`} label={parameter.displayName || parameter.key} value={extraInfoParameterInputType(parameter) === 'checkbox' ? parameter.value === 'true' ? '是' : '否' : parameter.value || '—'}/>) }
							</div>
						</div>
					))}
				</div>
			)) : <SnapAlert severity="info">未添加额外信息</SnapAlert>}
		</section>
		<section className="taskai-detail-section mb-5 grid gap-2">
			<span className="taskai-detail-section__title font-display text-xs font-extrabold uppercase tracking-wide text-snap-muted">系统环境变量</span>
			<div className="grid gap-1.5">
				<TaskDetailValue label="TASKAI_TASK_ID" value="任务关联的自定义命令和前后置脚本"/>
				<TaskDetailValue label="TASKAI_TERMINAL_ID" value="仅新建的普通终端和显示终端的自定义命令"/>
				<TaskDetailValue label="TASKAI_STATUS_API" value="本机 HTTP 服务正在监听时，注入到之后新建的终端"/>
			</div>
		</section>
		<SnapButton variant="secondary" size="sm" className="mb-3 self-start" onClick={() => onCopyLifecycleCommandInput(task.id)}>复制当前命令链输入 JSON</SnapButton>
		{task.lifecycleExecution && <SnapAlert severity={task.lifecycleExecution.state === 'failed' ? 'error' : 'warning'} className="mb-3">
			<span className="font-display text-sm font-bold text-snap-ink">{lifecycleHooks.find((hook) => hook.id === task.lifecycleExecution?.hook)?.label ?? '生命周期'}：{task.lifecycleExecution.currentCommandName || '命令'}（{task.lifecycleExecution.currentIndex}/{task.lifecycleExecution.commandCount}）</span>
			{task.lifecycleExecution.error && <span className="mt-0.5 block text-sm text-snap-ink" style={{whiteSpace: 'pre-wrap'}}>{task.lifecycleExecution.error}</span>}
		</SnapAlert>}
      {task.status === 'running' && task.workspacePath && (
        <div className="grid gap-1">
          <span className="font-display text-xs font-extrabold uppercase tracking-wide text-snap-muted">工作目录</span>
          <span className="text-sm text-snap-ink" style={{fontFamily: 'ui-monospace, monospace', overflowWrap: 'anywhere'}}>{task.workspacePath}</span>
        </div>
      )}
    </div>
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
	return <div className="taskai-detail-value grid min-w-0 gap-0.5 border-l-2 border-snap-cobalt/40 pl-2">
		<span className="break-words font-display text-xs font-extrabold uppercase tracking-wide text-snap-muted">{label}</span>
		<span className="break-words whitespace-pre-wrap text-sm text-snap-ink">{value}</span>
	</div>
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

function quickInputContentPreview(content: string): string {
	const compact = content.replace(/\r?\n/g, ' ↵ ').replace(/\t/g, ' ⇥ ')
	return [...compact].length > 120 ? `${[...compact].slice(0, 120).join('')}…` : compact
}

function gitRepositoryName(repository: string): string {
	const normalized = repository.trim()
	if (!normalized.endsWith('.git')) {
		return ''
	}
	const path = normalized.slice(0, -'.git'.length)
	const slashIndex = path.lastIndexOf('/')
	return slashIndex < 0 ? '' : path.slice(slashIndex + 1)
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
    disableTaskAIMouseClipboard: false,
  }
}

function showError(error: unknown, setMessage: Dispatch<SetStateAction<Notification | undefined>>) {
	setMessage({text: error instanceof Error ? error.message : String(error), severity: 'error'})
}

function normalizeTerminalFontCandidates(candidates: TerminalFontCandidate[]): TerminalFontCandidate[] {
	const normalized = new Map<string, TerminalFontCandidate>()
	for (const candidate of candidates) {
		const family = candidate.family?.trim() ?? ''
		if (family && candidate.spacing !== 'mono' && candidate.spacing !== 'dual') {
			continue
		}
		const key = family.toLocaleLowerCase()
		if (!normalized.has(key)) {
			normalized.set(key, {family, spacing: family ? candidate.spacing : 'mono'})
		}
	}
	const listed = [...normalized.values()].filter((candidate) => candidate.family)
	listed.sort((left, right) => left.family.localeCompare(right.family))
	return [defaultTerminalFontCandidate, ...listed]
}

function terminalFontCandidateName(candidate: TerminalFontCandidate): string {
	return candidate.family || '默认终端字体'
}

function terminalFontSpacingLabel(spacing: TerminalFontCandidate['spacing']): string {
	if (spacing === 'dual') {
		return '双宽'
	}
	if (spacing === 'unavailable') {
		return '当前字体不可用'
	}
	return '等宽'
}

function terminalThemeSelectionOpacity(theme: TerminalTheme): number {
	const opacity = theme.selectionBackground.length === 9 ? Number.parseInt(theme.selectionBackground.slice(7), 16) : 255
	return Math.round(opacity / 255 * 100)
}

function terminalThemeWithSelectionOpacity(theme: TerminalTheme, opacity: number): TerminalTheme {
	const normalizedOpacity = Math.min(100, Math.max(0, Math.round(opacity)))
	const alpha = Math.round(normalizedOpacity / 100 * 255).toString(16).toUpperCase().padStart(2, '0')
	return {...theme, selectionBackground: `${theme.selectionBackground.slice(0, 7)}${alpha}`}
}

function terminalColorInputValue(value: string): string {
	return value.slice(0, 7)
}

function TerminalThemeColorPicker({label, value, onChange}: {label: string, value: string, onChange(value: string): void}) {
	return <label className="flex items-center justify-between gap-2 rounded-snap border-2 border-snap-outline bg-snap-surface px-2 py-1.5 text-sm text-snap-ink">
		<span>{label}</span>
		<input aria-label={`终端${label}色`} type="color" value={terminalColorInputValue(value)} onChange={(event) => onChange(event.target.value.toUpperCase())} className="h-8 w-10 cursor-pointer rounded border-0 bg-transparent p-0"/>
	</label>
}

function TerminalThemeColorGroup({title, fields, theme, onChange}: {
	title: string
	fields: Array<{key: keyof TerminalTheme, label: string}>
	theme: TerminalTheme
	onChange(key: keyof TerminalTheme, value: string): void
}) {
	return <div className="grid gap-2">
		<span className="text-sm font-bold text-snap-ink">{title}</span>
		<div className="grid grid-cols-2 gap-2 sm:grid-cols-4">
			{fields.map((field) => <TerminalThemeColorPicker key={field.key} label={field.label} value={theme[field.key]} onChange={(value) => onChange(field.key, value)}/>)}
		</div>
	</div>
}
