package terminal

import (
	"errors"
	"fmt"
	"io"
	"sort"
	"sync"
	"unicode/utf8"
)

var ErrTaskClosing = errors.New("任务正在结束，不能新增终端")

type Manager struct {
	backend Backend
	publish func(Event)

	mu            sync.Mutex
	sessions      map[string]map[string]*managedSession
	closed        map[string]map[string]bool
	exitReasons   map[string]map[string]ExitReason
	exitCallbacks map[string]map[string][]func()
	closingTasks  map[string]bool
}

type managedSession struct {
	info      Info
	command   string
	shellPath string
	session   Session
	done      chan struct{}
}

type TerminalEnvironmentBuilder func(terminalID string) []string

func NewManager(backend Backend, publish func(Event)) *Manager {
	if publish == nil {
		publish = func(Event) {}
	}
	return &Manager{
		backend:       backend,
		publish:       publish,
		sessions:      make(map[string]map[string]*managedSession),
		closed:        make(map[string]map[string]bool),
		exitReasons:   make(map[string]map[string]ExitReason),
		exitCallbacks: make(map[string]map[string][]func()),
		closingTasks:  make(map[string]bool),
	}
}

func (manager *Manager) Create(taskID, directory, shellPath string, columns, rows uint16) (Info, error) {
	return manager.CreateWithEnvironment(taskID, directory, shellPath, nil, columns, rows)
}

func (manager *Manager) CreateWithEnvironment(taskID, directory, shellPath string, environment []string, columns, rows uint16) (Info, error) {
	return manager.create(StartRequest{
		TaskID:      taskID,
		Directory:   directory,
		ShellPath:   shellPath,
		Environment: append([]string(nil), environment...),
		Columns:     columns,
		Rows:        rows,
	})
}

func (manager *Manager) CreateWithEnvironmentBuilder(taskID, directory, shellPath string, environment TerminalEnvironmentBuilder, columns, rows uint16) (Info, error) {
	return manager.createWithEnvironmentBuilder(StartRequest{
		TaskID:    taskID,
		Directory: directory,
		ShellPath: shellPath,
		Columns:   columns,
		Rows:      rows,
	}, environment)
}

func (manager *Manager) CreateCommand(taskID, directory, shellPath, command string, arguments []string, columns, rows uint16) (Info, error) {
	return manager.CreateCommandWithEnvironment(taskID, directory, shellPath, command, arguments, nil, columns, rows)
}

func (manager *Manager) CreateCommandWithEnvironment(taskID, directory, shellPath, command string, arguments, environment []string, columns, rows uint16) (Info, error) {
	return manager.create(StartRequest{
		TaskID:      taskID,
		Directory:   directory,
		ShellPath:   shellPath,
		Command:     command,
		Arguments:   append([]string(nil), arguments...),
		Environment: append([]string(nil), environment...),
		Columns:     columns,
		Rows:        rows,
	})
}

func (manager *Manager) CreateCommandWithEnvironmentBuilder(taskID, directory, shellPath, command string, arguments []string, environment TerminalEnvironmentBuilder, columns, rows uint16) (Info, error) {
	return manager.CreateCommandWithOptionsAndEnvironmentBuilder(taskID, directory, shellPath, command, arguments, CommandOptions{}, environment, columns, rows)
}

func (manager *Manager) CreateCommandWithOptionsAndEnvironmentBuilder(taskID, directory, shellPath, command string, arguments []string, options CommandOptions, environment TerminalEnvironmentBuilder, columns, rows uint16) (Info, error) {
	return manager.createWithEnvironmentBuilder(StartRequest{
		TaskID:                      taskID,
		Directory:                   directory,
		ShellPath:                   shellPath,
		Command:                     command,
		Arguments:                   append([]string(nil), arguments...),
		DisableTaskAIMouseClipboard: options.DisableTaskAIMouseClipboard,
		Columns:                     columns,
		Rows:                        rows,
	}, environment)
}

func (manager *Manager) create(request StartRequest) (Info, error) {
	return manager.createWithEnvironmentBuilder(request, nil)
}

func (manager *Manager) createWithEnvironmentBuilder(request StartRequest, environment TerminalEnvironmentBuilder) (Info, error) {
	if request.TaskID == "" {
		return Info{}, fmt.Errorf("任务 ID 不能为空")
	}
	if request.Directory == "" {
		return Info{}, fmt.Errorf("终端工作目录不能为空")
	}

	manager.mu.Lock()
	defer manager.mu.Unlock()
	if manager.closingTasks[request.TaskID] {
		return Info{}, ErrTaskClosing
	}

	request.Columns = normalizedDimension(request.Columns, 80)
	request.Rows = normalizedDimension(request.Rows, 24)
	request.ID = sessionID(request.ID)
	if environment != nil {
		request.Environment = append([]string(nil), environment(request.ID)...)
	}
	session, err := manager.backend.Start(request)
	if err != nil {
		return Info{}, err
	}

	command := request.Command
	if command == "" {
		command = request.ShellPath
	}
	info := Info{
		ID:                          request.ID,
		TaskID:                      request.TaskID,
		State:                       StateActive,
		DisableTaskAIMouseClipboard: request.DisableTaskAIMouseClipboard,
		Command:                     command,
	}
	managed := &managedSession{info: info, command: command, shellPath: request.ShellPath, session: session, done: make(chan struct{})}
	if manager.sessions[request.TaskID] == nil {
		manager.sessions[request.TaskID] = make(map[string]*managedSession)
	}
	if _, exists := manager.sessions[request.TaskID][info.ID]; exists {
		_ = session.Close()
		return Info{}, fmt.Errorf("终端 ID 重复")
	}
	manager.sessions[request.TaskID][info.ID] = managed
	go manager.watch(managed)

	return info, nil
}

func (manager *Manager) ListActive(taskID string) []ActiveSession {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	byID := manager.sessions[taskID]
	items := make([]ActiveSession, 0, len(byID))
	for _, managed := range byID {
		items = append(items, ActiveSession{ID: managed.info.ID, Command: managed.command})
	}
	sort.Slice(items, func(left, right int) bool { return items[left].ID < items[right].ID })
	return items
}

func (manager *Manager) Write(taskID, terminalID, data string) error {
	managed, err := manager.session(taskID, terminalID)
	if err != nil {
		return err
	}
	_, err = managed.session.Write([]byte(data))
	return err
}

func (manager *Manager) WriteFilePaths(taskID, terminalID string, paths []string) error {
	managed, err := manager.session(taskID, terminalID)
	if err != nil {
		return err
	}
	data, err := formatDroppedFilePaths(managed.shellPath, paths)
	if err != nil {
		return err
	}
	_, err = managed.session.Write([]byte(data))
	return err
}

func (manager *Manager) Resize(taskID, terminalID string, columns, rows uint16) error {
	managed, err := manager.session(taskID, terminalID)
	if err != nil {
		return err
	}
	return managed.session.Resize(normalizedDimension(columns, 80), normalizedDimension(rows, 24))
}

func (manager *Manager) Close(taskID, terminalID string) error {
	manager.mu.Lock()
	managed := manager.sessions[taskID][terminalID]
	alreadyClosed := manager.closed[taskID][terminalID]
	if managed != nil {
		manager.setExitReasonLocked(taskID, terminalID, ExitReasonClosed)
	}
	manager.mu.Unlock()
	if managed == nil {
		if alreadyClosed {
			return nil
		}
		return fmt.Errorf("终端不存在或不属于当前任务")
	}
	if err := managed.session.Close(); err != nil {
		manager.clearExitReason(taskID, terminalID, ExitReasonClosed)
		return err
	}
	<-managed.done
	return nil
}

func (manager *Manager) OnExit(taskID, terminalID string, callback func()) {
	if callback == nil {
		return
	}
	manager.mu.Lock()
	if manager.closed[taskID][terminalID] {
		manager.mu.Unlock()
		go callback()
		return
	}
	if manager.sessions[taskID][terminalID] == nil {
		manager.mu.Unlock()
		return
	}
	if manager.exitCallbacks[taskID] == nil {
		manager.exitCallbacks[taskID] = make(map[string][]func())
	}
	manager.exitCallbacks[taskID][terminalID] = append(manager.exitCallbacks[taskID][terminalID], callback)
	manager.mu.Unlock()
}

// CloseTask prevents new sessions, closes every existing session and waits for output pumps to stop.
// The task remains blocked after a successful call because its owner is about to remove the workspace.
func (manager *Manager) CloseTask(taskID string) error {
	manager.mu.Lock()
	if manager.closingTasks[taskID] {
		manager.mu.Unlock()
		return ErrTaskClosing
	}
	manager.closingTasks[taskID] = true
	sessions := manager.taskSessionsLocked(taskID)
	for _, managed := range sessions {
		manager.setExitReasonLocked(taskID, managed.info.ID, ExitReasonTaskEnded)
	}
	manager.mu.Unlock()

	if err := closeSessions(sessions); err != nil {
		manager.mu.Lock()
		delete(manager.closingTasks, taskID)
		for _, managed := range sessions {
			manager.clearExitReasonLocked(taskID, managed.info.ID, ExitReasonTaskEnded)
		}
		manager.mu.Unlock()
		return err
	}
	return nil
}

// ReopenTask is used only when the task lifecycle cannot finish cleanup and must remain runnable.
func (manager *Manager) ReopenTask(taskID string) {
	manager.mu.Lock()
	defer manager.mu.Unlock()
	delete(manager.closingTasks, taskID)
}

func (manager *Manager) CloseAll() error {
	manager.mu.Lock()
	all := make([]*managedSession, 0)
	for taskID := range manager.sessions {
		manager.closingTasks[taskID] = true
		sessions := manager.taskSessionsLocked(taskID)
		for _, managed := range sessions {
			manager.setExitReasonLocked(taskID, managed.info.ID, ExitReasonApplicationShutdown)
		}
		all = append(all, sessions...)
	}
	manager.mu.Unlock()
	return closeSessions(all)
}

func (manager *Manager) session(taskID, terminalID string) (*managedSession, error) {
	manager.mu.Lock()
	defer manager.mu.Unlock()
	managed := manager.sessions[taskID][terminalID]
	if managed == nil {
		return nil, fmt.Errorf("终端不存在或不属于当前任务")
	}
	return managed, nil
}

func (manager *Manager) taskSessionsLocked(taskID string) []*managedSession {
	byID := manager.sessions[taskID]
	sessions := make([]*managedSession, 0, len(byID))
	for _, managed := range byID {
		sessions = append(sessions, managed)
	}
	return sessions
}

func (manager *Manager) watch(managed *managedSession) {
	defer func() {
		_ = managed.session.Wait()
		manager.mu.Lock()
		if byID := manager.sessions[managed.info.TaskID]; byID != nil {
			delete(byID, managed.info.ID)
			if len(byID) == 0 {
				delete(manager.sessions, managed.info.TaskID)
			}
		}
		if manager.closed[managed.info.TaskID] == nil {
			manager.closed[managed.info.TaskID] = make(map[string]bool)
		}
		manager.closed[managed.info.TaskID][managed.info.ID] = true
		exitReason := manager.exitReasonLocked(managed.info.TaskID, managed.info.ID)
		manager.clearExitReasonLocked(managed.info.TaskID, managed.info.ID, exitReason)
		callbacks := manager.exitCallbacks[managed.info.TaskID][managed.info.ID]
		delete(manager.exitCallbacks[managed.info.TaskID], managed.info.ID)
		if len(manager.exitCallbacks[managed.info.TaskID]) == 0 {
			delete(manager.exitCallbacks, managed.info.TaskID)
		}
		manager.mu.Unlock()
		manager.publish(Event{TaskID: managed.info.TaskID, TerminalID: managed.info.ID, Type: "exited", ExitReason: exitReason})
		for _, callback := range callbacks {
			go callback()
		}
		close(managed.done)
	}()

	buffer := make([]byte, 4096)
	output := terminalOutputFramer{}
	for {
		count, err := managed.session.Read(buffer)
		if count > 0 {
			manager.publishOutput(managed, output.write(buffer[:count]))
		}
		if err != nil {
			manager.publishOutput(managed, output.flush())
			if !errors.Is(err, io.EOF) && !isExpectedTerminalReadError(err) {
				manager.publish(Event{TaskID: managed.info.TaskID, TerminalID: managed.info.ID, Type: "error", Data: err.Error()})
			}
			return
		}
	}
}

func (manager *Manager) publishOutput(managed *managedSession, data string) {
	if data == "" {
		return
	}
	manager.publish(Event{TaskID: managed.info.TaskID, TerminalID: managed.info.ID, Type: "output", Data: data})
}

type terminalOutputFramer struct {
	pending []byte
}

func (framer *terminalOutputFramer) write(data []byte) string {
	combined := append(append([]byte(nil), framer.pending...), data...)
	framer.pending = framer.pending[:0]
	output := make([]byte, 0, len(combined))
	for len(combined) > 0 {
		if !utf8.FullRune(combined) {
			framer.pending = append(framer.pending, combined...)
			break
		}
		runeValue, size := utf8.DecodeRune(combined)
		if runeValue == utf8.RuneError && size == 1 {
			output = append(output, string(utf8.RuneError)...)
		} else {
			output = append(output, combined[:size]...)
		}
		combined = combined[size:]
	}
	return string(output)
}

func (framer *terminalOutputFramer) flush() string {
	if len(framer.pending) == 0 {
		return ""
	}
	framer.pending = framer.pending[:0]
	return string(utf8.RuneError)
}

func (manager *Manager) setExitReasonLocked(taskID, terminalID string, reason ExitReason) {
	if manager.exitReasons[taskID] == nil {
		manager.exitReasons[taskID] = make(map[string]ExitReason)
	}
	manager.exitReasons[taskID][terminalID] = reason
}

func (manager *Manager) exitReasonLocked(taskID, terminalID string) ExitReason {
	reason := manager.exitReasons[taskID][terminalID]
	if reason == "" {
		return ExitReasonUnexpected
	}
	return reason
}

func (manager *Manager) clearExitReason(taskID, terminalID string, reason ExitReason) {
	manager.mu.Lock()
	defer manager.mu.Unlock()
	manager.clearExitReasonLocked(taskID, terminalID, reason)
}

func (manager *Manager) clearExitReasonLocked(taskID, terminalID string, reason ExitReason) {
	byID := manager.exitReasons[taskID]
	if byID == nil || byID[terminalID] != reason {
		return
	}
	delete(byID, terminalID)
	if len(byID) == 0 {
		delete(manager.exitReasons, taskID)
	}
}

func closeSessions(sessions []*managedSession) error {
	var firstError error
	for _, managed := range sessions {
		if err := managed.session.Close(); err != nil && firstError == nil {
			firstError = err
		}
	}
	if firstError != nil {
		return firstError
	}
	for _, managed := range sessions {
		<-managed.done
	}
	return nil
}

func normalizedDimension(value, fallback uint16) uint16 {
	if value == 0 {
		return fallback
	}
	return value
}
