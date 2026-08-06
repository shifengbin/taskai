package realtime

import (
	"sort"
	"sync"
	"time"
)

type Status string

const (
	StatusIdle    Status = "idle"
	StatusWorking Status = "working"
	StatusUnread  Status = "unread"
	StatusError   Status = "error"
)

const TitleActivityTimeout = 1500 * time.Millisecond

type Mode string

const (
	ModeTitleChange  Mode = "title-change"
	ModeOutputChange Mode = "output-change"
	ModeHTTP         Mode = "http"
)

type Timer interface {
	Stop() bool
}

type Clock interface {
	Now() time.Time
	AfterFunc(time.Duration, func()) Timer
}

type Options struct {
	Clock   Clock
	Mode    Mode
	Publish func(Event)
}

type Event struct {
	Version         uint64 `json:"version"`
	TaskID          string `json:"taskId"`
	TaskStatus      Status `json:"taskStatus"`
	TerminalID      string `json:"terminalId,omitempty"`
	TerminalStatus  Status `json:"terminalStatus,omitempty"`
	TerminalRemoved bool   `json:"terminalRemoved,omitempty"`
}

type StatusSnapshot struct {
	Tasks []TaskSnapshot `json:"tasks"`
}

type TaskSnapshot struct {
	TaskID          string             `json:"taskId"`
	Title           string             `json:"title,omitempty"`
	LifecycleStatus string             `json:"lifecycleStatus,omitempty"`
	Status          Status             `json:"status"`
	Terminals       []TerminalSnapshot `json:"terminals"`
}

type TerminalSnapshot struct {
	TerminalID string `json:"terminalId"`
	Status     Status `json:"status"`
}

type TerminalPresence string

const (
	TerminalMissing TerminalPresence = "missing"
	TerminalActive  TerminalPresence = "active"
	TerminalRemoved TerminalPresence = "removed"
)

type Service struct {
	mu       sync.Mutex
	clock    Clock
	mode     Mode
	publish  func(Event)
	version  uint64
	tasks    map[string]*taskState
	selected terminalKey
}

type taskState struct {
	terminals map[string]*terminalState
	removed   map[string]bool
	override  *Status
}

type terminalState struct {
	status       Status
	generation   uint64
	lastActivity time.Time
	timer        Timer
}

type terminalKey struct {
	taskID     string
	terminalID string
}

type systemClock struct{}

func (systemClock) Now() time.Time {
	return time.Now()
}

func (systemClock) AfterFunc(delay time.Duration, callback func()) Timer {
	return time.AfterFunc(delay, callback)
}

func New(options Options) *Service {
	clock := options.Clock
	if clock == nil {
		clock = systemClock{}
	}
	mode := options.Mode
	if mode == "" {
		mode = ModeTitleChange
	}
	return &Service{
		clock:   clock,
		mode:    mode,
		publish: options.Publish,
		tasks:   make(map[string]*taskState),
	}
}

func (service *Service) RegisterTask(taskID string) {
	service.mu.Lock()
	service.taskLocked(taskID)
	event := service.eventLocked(taskID, "", false)
	service.mu.Unlock()
	service.emit(event)
}

func (service *Service) RegisterTerminal(taskID, terminalID string) {
	service.mu.Lock()
	task := service.taskLocked(taskID)
	delete(task.removed, terminalID)
	if _, exists := task.terminals[terminalID]; !exists {
		task.terminals[terminalID] = &terminalState{status: StatusIdle}
	}
	event := service.eventLocked(taskID, terminalID, false)
	service.mu.Unlock()
	service.emit(event)
}

func (service *Service) SetMode(mode Mode) {
	if mode == "" {
		mode = ModeTitleChange
	}
	service.mu.Lock()
	if service.mode == mode {
		service.mu.Unlock()
		return
	}
	service.mode = mode
	events := make([]Event, 0, len(service.tasks))
	for taskID, task := range service.tasks {
		task.override = nil
		for _, terminal := range task.terminals {
			terminal.generation++
			if terminal.timer != nil {
				terminal.timer.Stop()
				terminal.timer = nil
			}
			terminal.lastActivity = time.Time{}
			terminal.status = StatusIdle
		}
		events = append(events, service.eventLocked(taskID, "", false))
	}
	service.mu.Unlock()
	for _, event := range events {
		service.emit(event)
	}
}

func (service *Service) SetTaskStatus(taskID string, status Status) bool {
	if !status.Valid() {
		return false
	}
	service.mu.Lock()
	task := service.taskLocked(taskID)
	task.override = statusPointer(status)
	event := service.eventLocked(taskID, "", false)
	service.mu.Unlock()
	service.emit(event)
	return true
}

func (service *Service) SetTerminalStatus(taskID, terminalID string, status Status) bool {
	if !status.Valid() {
		return false
	}
	service.mu.Lock()
	task := service.tasks[taskID]
	if task == nil || task.terminals[terminalID] == nil {
		service.mu.Unlock()
		return false
	}
	terminal := task.terminals[terminalID]
	terminal.generation++
	if terminal.timer != nil {
		terminal.timer.Stop()
		terminal.timer = nil
	}
	terminal.lastActivity = time.Time{}
	terminal.status = status
	task.override = nil
	event := service.eventLocked(taskID, terminalID, false)
	service.mu.Unlock()
	service.emit(event)
	return true
}

func (service *Service) ReportTitleActivity(taskID, terminalID string) bool {
	return service.reportActivity(taskID, terminalID, ModeTitleChange)
}

func (service *Service) ReportOutputActivity(taskID, terminalID string) bool {
	return service.reportActivity(taskID, terminalID, ModeOutputChange)
}

func (service *Service) reportActivity(taskID, terminalID string, source Mode) bool {
	service.mu.Lock()
	if service.mode != source {
		service.mu.Unlock()
		return false
	}
	task := service.tasks[taskID]
	if task == nil || task.terminals[terminalID] == nil {
		service.mu.Unlock()
		return false
	}
	terminal := task.terminals[terminalID]
	changed := terminal.status != StatusWorking || task.override != nil
	terminal.lastActivity = service.clock.Now()
	task.override = nil
	if terminal.status != StatusWorking {
		terminal.status = StatusWorking
	}
	if terminal.timer == nil {
		terminal.generation++
		generation := terminal.generation
		terminal.timer = service.clock.AfterFunc(TitleActivityTimeout, func() {
			service.handleActivityTimeout(taskID, terminalID, source, generation)
		})
	}
	var event Event
	if changed {
		event = service.eventLocked(taskID, terminalID, false)
	}
	service.mu.Unlock()
	if changed {
		service.emit(event)
	}
	return true
}

func (service *Service) SelectTerminal(taskID, terminalID string) {
	service.mu.Lock()
	service.selected = terminalKey{taskID: taskID, terminalID: terminalID}
	task := service.tasks[taskID]
	if task == nil || task.terminals[terminalID] == nil || task.terminals[terminalID].status != StatusUnread {
		service.mu.Unlock()
		return
	}
	task.terminals[terminalID].status = StatusIdle
	task.override = nil
	event := service.eventLocked(taskID, terminalID, false)
	service.mu.Unlock()
	service.emit(event)
}

func (service *Service) ClearSelection() {
	service.mu.Lock()
	service.selected = terminalKey{}
	service.mu.Unlock()
}

func (service *Service) RemoveTerminal(taskID, terminalID string) bool {
	service.mu.Lock()
	task := service.tasks[taskID]
	if task == nil || task.terminals[terminalID] == nil {
		service.mu.Unlock()
		return false
	}
	terminal := task.terminals[terminalID]
	terminal.generation++
	if terminal.timer != nil {
		terminal.timer.Stop()
		terminal.timer = nil
	}
	delete(task.terminals, terminalID)
	task.removed[terminalID] = true
	if service.selected == (terminalKey{taskID: taskID, terminalID: terminalID}) {
		service.selected = terminalKey{}
	}
	event := service.eventLocked(taskID, terminalID, true)
	service.mu.Unlock()
	service.emit(event)
	return true
}

func (service *Service) RemoveTask(taskID string) {
	service.mu.Lock()
	task := service.tasks[taskID]
	if task == nil {
		service.mu.Unlock()
		return
	}
	for _, terminal := range task.terminals {
		terminal.generation++
		if terminal.timer != nil {
			terminal.timer.Stop()
			terminal.timer = nil
		}
	}
	delete(service.tasks, taskID)
	if service.selected.taskID == taskID {
		service.selected = terminalKey{}
	}
	service.mu.Unlock()
}

func (service *Service) TaskStatus(taskID string) Status {
	service.mu.Lock()
	defer service.mu.Unlock()
	return service.taskStatusLocked(service.tasks[taskID])
}

func (service *Service) TerminalStatus(taskID, terminalID string) Status {
	service.mu.Lock()
	defer service.mu.Unlock()
	if task := service.tasks[taskID]; task != nil {
		if terminal := task.terminals[terminalID]; terminal != nil {
			return terminal.status
		}
	}
	return StatusIdle
}

func (service *Service) TerminalPresence(taskID, terminalID string) TerminalPresence {
	service.mu.Lock()
	defer service.mu.Unlock()
	task := service.tasks[taskID]
	if task == nil {
		return TerminalMissing
	}
	if task.terminals[terminalID] != nil {
		return TerminalActive
	}
	if task.removed[terminalID] {
		return TerminalRemoved
	}
	return TerminalMissing
}

func (service *Service) Snapshot() StatusSnapshot {
	service.mu.Lock()
	defer service.mu.Unlock()
	taskIDs := make([]string, 0, len(service.tasks))
	for taskID := range service.tasks {
		taskIDs = append(taskIDs, taskID)
	}
	sort.Strings(taskIDs)
	snapshot := StatusSnapshot{Tasks: make([]TaskSnapshot, 0, len(taskIDs))}
	for _, taskID := range taskIDs {
		task := service.tasks[taskID]
		terminalIDs := make([]string, 0, len(task.terminals))
		for terminalID := range task.terminals {
			terminalIDs = append(terminalIDs, terminalID)
		}
		sort.Strings(terminalIDs)
		item := TaskSnapshot{TaskID: taskID, Status: service.taskStatusLocked(task), Terminals: make([]TerminalSnapshot, 0, len(terminalIDs))}
		for _, terminalID := range terminalIDs {
			item.Terminals = append(item.Terminals, TerminalSnapshot{TerminalID: terminalID, Status: task.terminals[terminalID].status})
		}
		snapshot.Tasks = append(snapshot.Tasks, item)
	}
	return snapshot
}

func (service *Service) handleActivityTimeout(taskID, terminalID string, source Mode, generation uint64) {
	service.mu.Lock()
	if service.mode != source {
		service.mu.Unlock()
		return
	}
	task := service.tasks[taskID]
	if task == nil || task.terminals[terminalID] == nil {
		service.mu.Unlock()
		return
	}
	terminal := task.terminals[terminalID]
	if terminal.generation != generation {
		service.mu.Unlock()
		return
	}
	remaining := TitleActivityTimeout - service.clock.Now().Sub(terminal.lastActivity)
	if remaining > 0 {
		terminal.timer = service.clock.AfterFunc(remaining, func() {
			service.handleActivityTimeout(taskID, terminalID, source, generation)
		})
		service.mu.Unlock()
		return
	}
	terminal.timer = nil
	if service.selected == (terminalKey{taskID: taskID, terminalID: terminalID}) {
		terminal.status = StatusIdle
	} else {
		terminal.status = StatusUnread
	}
	task.override = nil
	event := service.eventLocked(taskID, terminalID, false)
	service.mu.Unlock()
	service.emit(event)
}

func (service *Service) taskLocked(taskID string) *taskState {
	task := service.tasks[taskID]
	if task != nil {
		return task
	}
	task = &taskState{terminals: make(map[string]*terminalState), removed: make(map[string]bool)}
	service.tasks[taskID] = task
	return task
}

func (service *Service) eventLocked(taskID, terminalID string, terminalRemoved bool) Event {
	service.version++
	event := Event{
		Version:         service.version,
		TaskID:          taskID,
		TaskStatus:      service.taskStatusLocked(service.tasks[taskID]),
		TerminalID:      terminalID,
		TerminalRemoved: terminalRemoved,
	}
	if task := service.tasks[taskID]; task != nil && !terminalRemoved {
		if terminal := task.terminals[terminalID]; terminal != nil {
			event.TerminalStatus = terminal.status
		}
	}
	return event
}

func (service *Service) taskStatusLocked(task *taskState) Status {
	if task == nil {
		return StatusIdle
	}
	if task.override != nil {
		return *task.override
	}
	status := StatusIdle
	for _, terminal := range task.terminals {
		if terminal.status.priority() > status.priority() {
			status = terminal.status
		}
	}
	return status
}

func (service *Service) emit(event Event) {
	if service.publish != nil {
		service.publish(event)
	}
}

func (status Status) Valid() bool {
	return status == StatusIdle || status == StatusWorking || status == StatusUnread || status == StatusError
}

func (status Status) priority() int {
	switch status {
	case StatusError:
		return 4
	case StatusUnread:
		return 3
	case StatusWorking:
		return 2
	default:
		return 1
	}
}

func statusPointer(status Status) *Status {
	return &status
}
