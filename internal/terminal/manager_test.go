package terminal

import (
	"errors"
	"fmt"
	"io"
	"sync"
	"testing"
	"time"
)

func TestManagerRoutesOutputAndKeepsTasksIsolated(t *testing.T) {
	t.Parallel()

	backend := &fakeBackend{}
	events := make(chan Event, 8)
	manager := NewManager(backend, func(event Event) { events <- event })

	first, err := manager.Create("task-a", t.TempDir(), "", 80, 24)
	if err != nil {
		t.Fatalf("创建第一个终端: %v", err)
	}
	second, err := manager.Create("task-b", t.TempDir(), "", 80, 24)
	if err != nil {
		t.Fatalf("创建第二个终端: %v", err)
	}
	if first.TaskID != "task-a" || second.TaskID != "task-b" {
		t.Fatalf("终端任务归属错误: %#v %#v", first, second)
	}

	backend.session(first.ID).emit("first output")
	backend.session(second.ID).emit("second output")

	seen := map[string]string{}
	for len(seen) < 2 {
		event := receiveEvent(t, events)
		if event.Type == "output" {
			seen[event.TerminalID] = event.Data
		}
	}
	if seen[first.ID] != "first output" || seen[second.ID] != "second output" {
		t.Fatalf("输出路由错误: %#v", seen)
	}

	if err := manager.Write("task-a", first.ID, "input"); err != nil {
		t.Fatalf("写入终端: %v", err)
	}
	if got := backend.session(first.ID).input(); got != "input" {
		t.Fatalf("输入未写入所属终端: %q", got)
	}
	if err := manager.Write("task-a", second.ID, "wrong"); err == nil {
		t.Fatal("不应允许跨任务写入终端")
	}
}

func TestManagerBlocksNewTerminalWhileTaskIsClosing(t *testing.T) {
	t.Parallel()

	backend := &fakeBackend{}
	manager := NewManager(backend, func(Event) {})
	created, err := manager.Create("task-a", t.TempDir(), "", 80, 24)
	if err != nil {
		t.Fatalf("创建终端: %v", err)
	}

	closed := make(chan error, 1)
	go func() { closed <- manager.CloseTask("task-a") }()
	select {
	case err := <-closed:
		if err != nil {
			t.Fatalf("关闭任务终端: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("关闭任务终端超时")
	}

	if _, err := manager.Create("task-a", t.TempDir(), "", 80, 24); err == nil {
		t.Fatal("结束任务期间不应允许新增终端")
	}
	if backend.session(created.ID).wasClosed() == false {
		t.Fatal("任务终端未关闭")
	}
}

func TestManagerClosesAllWithoutChangingTaskInformation(t *testing.T) {
	t.Parallel()

	backend := &fakeBackend{}
	manager := NewManager(backend, func(Event) {})
	first, err := manager.Create("task-a", t.TempDir(), "", 80, 24)
	if err != nil {
		t.Fatalf("创建终端: %v", err)
	}
	second, err := manager.Create("task-b", t.TempDir(), "", 80, 24)
	if err != nil {
		t.Fatalf("创建终端: %v", err)
	}

	if err := manager.CloseAll(); err != nil {
		t.Fatalf("关闭全部终端: %v", err)
	}
	if !backend.session(first.ID).wasClosed() || !backend.session(second.ID).wasClosed() {
		t.Fatal("未关闭全部终端")
	}
}

func TestManagerAllowsConcurrentCloseOfOneTerminal(t *testing.T) {
	t.Parallel()

	manager := NewManager(&fakeBackend{}, func(Event) {})
	created, err := manager.Create("task-a", t.TempDir(), "", 80, 24)
	if err != nil {
		t.Fatalf("创建终端: %v", err)
	}

	start := make(chan struct{})
	results := make(chan error, 2)
	for range 2 {
		go func() {
			<-start
			results <- manager.Close("task-a", created.ID)
		}()
	}
	close(start)
	for range 2 {
		if err := <-results; err != nil {
			t.Fatalf("并发关闭终端: %v", err)
		}
	}
}

func TestManagerRunsExitCallbackRegisteredAfterTerminalExit(t *testing.T) {
	backend := &fakeBackend{}
	events := make(chan Event, 2)
	manager := NewManager(backend, func(event Event) { events <- event })
	created, err := manager.Create("task-a", t.TempDir(), "", 80, 24)
	if err != nil {
		t.Fatalf("创建终端: %v", err)
	}
	if err := backend.session(created.ID).Close(); err != nil {
		t.Fatalf("关闭终端: %v", err)
	}
	for {
		if event := receiveEvent(t, events); event.Type == "exited" {
			break
		}
	}

	called := make(chan struct{}, 1)
	manager.OnExit(created.TaskID, created.ID, func() { called <- struct{}{} })
	select {
	case <-called:
	case <-time.After(time.Second):
		t.Fatal("退出后注册的回调未执行")
	}
}

func receiveEvent(t *testing.T, events <-chan Event) Event {
	t.Helper()
	select {
	case event := <-events:
		return event
	case <-time.After(time.Second):
		t.Fatal("等待终端事件超时")
		return Event{}
	}
}

type fakeBackend struct {
	mu       sync.Mutex
	sessions map[string]*fakeSession
}

func (backend *fakeBackend) Start(request StartRequest) (Session, error) {
	backend.mu.Lock()
	defer backend.mu.Unlock()
	if backend.sessions == nil {
		backend.sessions = make(map[string]*fakeSession)
	}
	id := fmt.Sprintf("terminal-%d", len(backend.sessions)+1)
	session := newFakeSession(id)
	backend.sessions[id] = session
	return session, nil
}

func (backend *fakeBackend) session(id string) *fakeSession {
	backend.mu.Lock()
	defer backend.mu.Unlock()
	return backend.sessions[id]
}

type fakeSession struct {
	id        string
	reader    *io.PipeReader
	writer    *io.PipeWriter
	mu        sync.Mutex
	writes    string
	closed    bool
	closeOnce sync.Once
}

func newFakeSession(id string) *fakeSession {
	reader, writer := io.Pipe()
	return &fakeSession{id: id, reader: reader, writer: writer}
}

func (session *fakeSession) ID() string { return session.id }

func (session *fakeSession) Read(buffer []byte) (int, error) { return session.reader.Read(buffer) }

func (session *fakeSession) Write(data []byte) (int, error) {
	session.mu.Lock()
	defer session.mu.Unlock()
	if session.closed {
		return 0, errors.New("已关闭")
	}
	session.writes += string(data)
	return len(data), nil
}

func (session *fakeSession) Resize(uint16, uint16) error { return nil }

func (session *fakeSession) Wait() error { return nil }

func (session *fakeSession) Close() error {
	session.closeOnce.Do(func() {
		session.mu.Lock()
		session.closed = true
		session.mu.Unlock()
		_ = session.writer.Close()
		_ = session.reader.Close()
	})
	return nil
}

func (session *fakeSession) emit(data string) { _, _ = session.writer.Write([]byte(data)) }

func (session *fakeSession) input() string {
	session.mu.Lock()
	defer session.mu.Unlock()
	return session.writes
}

func (session *fakeSession) wasClosed() bool {
	session.mu.Lock()
	defer session.mu.Unlock()
	return session.closed
}
