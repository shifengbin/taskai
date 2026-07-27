package terminal

import (
	"errors"
	"fmt"
	"io"
	"sync"
)

var ErrTaskClosing = errors.New("任务正在结束，不能新增终端")

type Manager struct {
	backend Backend
	publish func(Event)

	mu            sync.Mutex
	sessions      map[string]map[string]*managedSession
	closed        map[string]map[string]bool
	exitCallbacks map[string]map[string][]func()
	closingTasks  map[string]bool
}

type managedSession struct {
	info    Info
	session Session
	done    chan struct{}
}

func NewManager(backend Backend, publish func(Event)) *Manager {
	if publish == nil {
		publish = func(Event) {}
	}
	return &Manager{
		backend:       backend,
		publish:       publish,
		sessions:      make(map[string]map[string]*managedSession),
		closed:        make(map[string]map[string]bool),
		exitCallbacks: make(map[string]map[string][]func()),
		closingTasks:  make(map[string]bool),
	}
}

func (manager *Manager) Create(taskID, directory, shellPath string, columns, rows uint16) (Info, error) {
	return manager.create(StartRequest{
		TaskID:    taskID,
		Directory: directory,
		ShellPath: shellPath,
		Columns:   columns,
		Rows:      rows,
	})
}

func (manager *Manager) CreateCommand(taskID, directory, shellPath, command string, arguments []string, columns, rows uint16) (Info, error) {
	return manager.create(StartRequest{
		TaskID:    taskID,
		Directory: directory,
		ShellPath: shellPath,
		Command:   command,
		Arguments: append([]string(nil), arguments...),
		Columns:   columns,
		Rows:      rows,
	})
}

func (manager *Manager) create(request StartRequest) (Info, error) {
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
	session, err := manager.backend.Start(request)
	if err != nil {
		return Info{}, err
	}

	info := Info{ID: session.ID(), TaskID: request.TaskID, State: StateActive}
	if info.ID == "" {
		_ = session.Close()
		return Info{}, fmt.Errorf("终端会话未提供 ID")
	}
	managed := &managedSession{info: info, session: session, done: make(chan struct{})}
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

func (manager *Manager) Write(taskID, terminalID, data string) error {
	managed, err := manager.session(taskID, terminalID)
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
	manager.mu.Unlock()
	if managed == nil {
		if alreadyClosed {
			return nil
		}
		return fmt.Errorf("终端不存在或不属于当前任务")
	}
	if err := managed.session.Close(); err != nil {
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
	manager.mu.Unlock()

	if err := closeSessions(sessions); err != nil {
		manager.mu.Lock()
		delete(manager.closingTasks, taskID)
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
		all = append(all, manager.taskSessionsLocked(taskID)...)
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
		callbacks := manager.exitCallbacks[managed.info.TaskID][managed.info.ID]
		delete(manager.exitCallbacks[managed.info.TaskID], managed.info.ID)
		if len(manager.exitCallbacks[managed.info.TaskID]) == 0 {
			delete(manager.exitCallbacks, managed.info.TaskID)
		}
		manager.mu.Unlock()
		manager.publish(Event{TaskID: managed.info.TaskID, TerminalID: managed.info.ID, Type: "exited"})
		for _, callback := range callbacks {
			go callback()
		}
		close(managed.done)
	}()

	buffer := make([]byte, 4096)
	for {
		count, err := managed.session.Read(buffer)
		if count > 0 {
			manager.publish(Event{
				TaskID:     managed.info.TaskID,
				TerminalID: managed.info.ID,
				Type:       "output",
				Data:       string(buffer[:count]),
			})
		}
		if err != nil {
			if !errors.Is(err, io.EOF) && !isExpectedTerminalReadError(err) {
				manager.publish(Event{TaskID: managed.info.TaskID, TerminalID: managed.info.ID, Type: "error", Data: err.Error()})
			}
			return
		}
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
