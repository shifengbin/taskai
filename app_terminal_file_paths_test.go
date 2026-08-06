package main

import (
	"io"
	"sync"
	"testing"

	"taskai/internal/terminal"
)

func TestAppWritesDroppedTerminalFilePaths(t *testing.T) {
	app := newAppWithoutActiveTaskTemplate(t, t.TempDir())
	backend := &filePathTerminalBackend{}
	app.terminals = terminal.NewManager(backend, app.publishTerminalEvent)
	created, err := app.terminals.Create("task-a", t.TempDir(), "/bin/sh", 80, 24)
	if err != nil {
		t.Fatalf("创建终端: %v", err)
	}

	if err := app.WriteTerminalFilePaths("task-a", created.ID, []string{"/tmp/My Project/file.txt"}); err != nil {
		t.Fatalf("写入拖放路径: %v", err)
	}
	if got := backend.session(created.ID).input(); got != "'/tmp/My Project/file.txt'" {
		t.Errorf("终端输入 = %q", got)
	}
}

type filePathTerminalBackend struct {
	mu       sync.Mutex
	sessions map[string]*filePathTerminalSession
}

func (backend *filePathTerminalBackend) Start(request terminal.StartRequest) (terminal.Session, error) {
	backend.mu.Lock()
	defer backend.mu.Unlock()
	if backend.sessions == nil {
		backend.sessions = make(map[string]*filePathTerminalSession)
	}
	session := newFilePathTerminalSession(request.ID)
	backend.sessions[request.ID] = session
	return session, nil
}

func (backend *filePathTerminalBackend) session(id string) *filePathTerminalSession {
	backend.mu.Lock()
	defer backend.mu.Unlock()
	return backend.sessions[id]
}

type filePathTerminalSession struct {
	id        string
	reader    *io.PipeReader
	writer    *io.PipeWriter
	mu        sync.Mutex
	writes    string
	closeOnce sync.Once
}

func newFilePathTerminalSession(id string) *filePathTerminalSession {
	reader, writer := io.Pipe()
	return &filePathTerminalSession{id: id, reader: reader, writer: writer}
}

func (session *filePathTerminalSession) ID() string { return session.id }
func (session *filePathTerminalSession) Read(data []byte) (int, error) {
	return session.reader.Read(data)
}
func (session *filePathTerminalSession) Write(data []byte) (int, error) {
	session.mu.Lock()
	defer session.mu.Unlock()
	session.writes += string(data)
	return len(data), nil
}
func (session *filePathTerminalSession) Resize(uint16, uint16) error { return nil }
func (session *filePathTerminalSession) Wait() error                 { return nil }
func (session *filePathTerminalSession) Close() error {
	session.closeOnce.Do(func() {
		_ = session.writer.Close()
		_ = session.reader.Close()
	})
	return nil
}
func (session *filePathTerminalSession) input() string {
	session.mu.Lock()
	defer session.mu.Unlock()
	return session.writes
}

var _ terminal.Backend = (*filePathTerminalBackend)(nil)
var _ terminal.Session = (*filePathTerminalSession)(nil)
