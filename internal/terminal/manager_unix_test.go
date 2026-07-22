//go:build !windows

package terminal

import (
	"syscall"
	"testing"
)

func TestManagerTreatsUnixPTYEIOAsTerminalExit(t *testing.T) {
	events := make(chan Event, 2)
	manager := NewManager(&readErrorBackend{err: syscall.EIO}, func(event Event) { events <- event })

	created, err := manager.Create("task-a", t.TempDir(), "", 80, 24)
	if err != nil {
		t.Fatalf("创建终端: %v", err)
	}

	event := receiveEvent(t, events)
	if event.TaskID != created.TaskID || event.TerminalID != created.ID || event.Type != "exited" {
		t.Fatalf("PTY EIO 事件 = %#v，期望退出事件", event)
	}
}

type readErrorBackend struct {
	err error
}

func (backend *readErrorBackend) Start(StartRequest) (Session, error) {
	return &readErrorSession{id: "terminal-eio", err: backend.err}, nil
}

type readErrorSession struct {
	id  string
	err error
}

func (session *readErrorSession) ID() string                     { return session.id }
func (session *readErrorSession) Read([]byte) (int, error)       { return 0, session.err }
func (session *readErrorSession) Write(data []byte) (int, error) { return len(data), nil }
func (session *readErrorSession) Resize(uint16, uint16) error    { return nil }
func (session *readErrorSession) Wait() error                    { return nil }
func (session *readErrorSession) Close() error                   { return nil }
