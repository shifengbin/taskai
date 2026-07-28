//go:build windows

package terminal

import (
	"syscall"
	"testing"
)

func TestManagerTreatsWindowsOperationAbortedAsTerminalExit(t *testing.T) {
	events := make(chan Event, 2)
	manager := NewManager(&windowsReadErrorBackend{err: syscall.Errno(995)}, func(event Event) { events <- event })

	created, err := manager.Create("task-a", t.TempDir(), "", 80, 24)
	if err != nil {
		t.Fatalf("创建终端: %v", err)
	}

	event := receiveEvent(t, events)
	if event.TaskID != created.TaskID || event.TerminalID != created.ID || event.Type != "exited" {
		t.Fatalf("ERROR_OPERATION_ABORTED 事件 = %#v，期望仅产生退出事件", event)
	}
}

func TestManagerReportsOtherWindowsTerminalReadErrors(t *testing.T) {
	events := make(chan Event, 2)
	manager := NewManager(&windowsReadErrorBackend{err: syscall.Errno(996)}, func(event Event) { events <- event })

	created, err := manager.Create("task-a", t.TempDir(), "", 80, 24)
	if err != nil {
		t.Fatalf("创建终端: %v", err)
	}

	if event := receiveEvent(t, events); event.TaskID != created.TaskID || event.TerminalID != created.ID || event.Type != "error" {
		t.Fatalf("普通 Windows 读取错误事件 = %#v，期望错误事件", event)
	}
	if event := receiveEvent(t, events); event.Type != "exited" {
		t.Fatalf("普通 Windows 读取错误后事件 = %#v，期望退出事件", event)
	}
}

type windowsReadErrorBackend struct {
	err error
}

func (backend *windowsReadErrorBackend) Start(StartRequest) (Session, error) {
	return &windowsReadErrorSession{id: "terminal-windows-read-error", err: backend.err}, nil
}

type windowsReadErrorSession struct {
	id  string
	err error
}

func (session *windowsReadErrorSession) ID() string                     { return session.id }
func (session *windowsReadErrorSession) Read([]byte) (int, error)       { return 0, session.err }
func (session *windowsReadErrorSession) Write(data []byte) (int, error) { return len(data), nil }
func (session *windowsReadErrorSession) Resize(uint16, uint16) error    { return nil }
func (session *windowsReadErrorSession) Wait() error                    { return nil }
func (session *windowsReadErrorSession) Close() error                   { return nil }
