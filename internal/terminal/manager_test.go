package terminal

import (
	"errors"
	"fmt"
	"io"
	"sync"
	"testing"
	"time"
	"unicode/utf8"
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

func TestManagerPublishesOnlyCompleteUTF8OutputChunks(t *testing.T) {
	backend := &fakeBackend{}
	events := make(chan Event, 4)
	manager := NewManager(backend, func(event Event) { events <- event })
	created, err := manager.Create("task-a", t.TempDir(), "", 80, 24)
	if err != nil {
		t.Fatalf("创建终端: %v", err)
	}

	output := []byte("before 中 after")
	splitAt := len("before ") + 1
	backend.session(created.ID).emitBytes(output[:splitAt])
	backend.session(created.ID).emitBytes(output[splitAt:])

	combined := ""
	for range 2 {
		event := receiveEvent(t, events)
		if event.Type != "output" {
			t.Fatalf("事件类型 = %q，期望 output", event.Type)
		}
		if !utf8.ValidString(event.Data) {
			t.Fatalf("输出事件包含不完整 UTF-8: %q", event.Data)
		}
		combined += event.Data
	}
	if combined != string(output) {
		t.Fatalf("合并输出 = %q，期望 %q", combined, string(output))
	}
}

func TestManagerPreservesFragmentedEmojiAndANSIOutput(t *testing.T) {
	backend := &fakeBackend{}
	events := make(chan Event, 8)
	manager := NewManager(backend, func(event Event) { events <- event })
	created, err := manager.Create("task-a", t.TempDir(), "", 80, 24)
	if err != nil {
		t.Fatalf("创建终端: %v", err)
	}

	output := []byte("\x1b[36m🙂\x1b[0m")
	splitAt := len("\x1b[36m") + 1
	backend.session(created.ID).emitBytes(output[:splitAt])
	backend.session(created.ID).emitBytes(output[splitAt : splitAt+2])
	backend.session(created.ID).emitBytes(output[splitAt+2:])
	if err := manager.Close(created.TaskID, created.ID); err != nil {
		t.Fatalf("关闭终端: %v", err)
	}

	received := receiveEventsThroughExit(t, events)
	combined := ""
	for index, event := range received {
		if event.Type != "output" {
			continue
		}
		if !utf8.ValidString(event.Data) {
			t.Fatalf("第 %d 个输出事件包含不完整 UTF-8: %q", index, event.Data)
		}
		combined += event.Data
	}
	if combined != string(output) {
		t.Fatalf("合并输出 = %q，期望 %q", combined, string(output))
	}
	if received[len(received)-1].Type != "exited" {
		t.Fatalf("最后一个事件 = %q，期望 exited", received[len(received)-1].Type)
	}
}

func TestManagerFlushesIncompleteUTF8BeforeExit(t *testing.T) {
	backend := &fakeBackend{}
	events := make(chan Event, 8)
	manager := NewManager(backend, func(event Event) { events <- event })
	created, err := manager.Create("task-a", t.TempDir(), "", 80, 24)
	if err != nil {
		t.Fatalf("创建终端: %v", err)
	}

	backend.session(created.ID).emitBytes([]byte{0xe4, 0xb8})
	if err := manager.Close(created.TaskID, created.ID); err != nil {
		t.Fatalf("关闭终端: %v", err)
	}

	received := receiveEventsThroughExit(t, events)
	outputIndex := -1
	exitIndex := -1
	for index, event := range received {
		switch event.Type {
		case "output":
			if event.Data != string(utf8.RuneError) {
				t.Fatalf("残留字节输出 = %q，期望替换字符", event.Data)
			}
			outputIndex = index
		case "exited":
			exitIndex = index
		}
	}
	if outputIndex == -1 || exitIndex == -1 || outputIndex > exitIndex {
		t.Fatalf("输出和退出事件顺序错误: %#v", received)
	}
}

func TestManagerFlushesIncompleteUTF8BeforeUnexpectedReadError(t *testing.T) {
	readErr := errors.New("读取失败")
	events := make(chan Event, 4)
	manager := NewManager(nil, func(event Event) { events <- event })
	managed := &managedSession{
		info:    Info{ID: "terminal-1", TaskID: "task-a"},
		session: &singleReadSession{id: "terminal-1", data: []byte{0xe4, 0xb8}, err: readErr},
		done:    make(chan struct{}),
	}

	go manager.watch(managed)

	received := receiveEventsThroughExit(t, events)
	if len(received) != 3 {
		t.Fatalf("事件数量 = %d，期望 3: %#v", len(received), received)
	}
	if received[0].Type != "output" || received[0].Data != string(utf8.RuneError) {
		t.Fatalf("第一个事件 = %#v，期望替换字符输出", received[0])
	}
	if received[1].Type != "error" || received[1].Data != readErr.Error() {
		t.Fatalf("第二个事件 = %#v，期望读取错误", received[1])
	}
	if received[2].Type != "exited" {
		t.Fatalf("第三个事件 = %#v，期望 exited", received[2])
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

func TestManagerAssignsTerminalIDAndEnvironmentBeforeStartingProcess(t *testing.T) {
	backend := &fakeBackend{}
	manager := NewManager(backend, func(Event) {})

	created, err := manager.CreateWithEnvironment(
		"task-a",
		t.TempDir(),
		"/bin/sh",
		[]string{"TASKAI_STATUS_API=http://127.0.0.1:18765/api/v1", "TASKAI_TASK_ID=task-a"},
		80,
		24,
	)
	if err != nil {
		t.Fatalf("创建终端: %v", err)
	}

	request := backend.request(created.ID)
	if request.ID != created.ID {
		t.Errorf("启动请求终端 ID = %q，期望 %q", request.ID, created.ID)
	}
	if !containsEnvironment(request.Environment, "TASKAI_STATUS_API=http://127.0.0.1:18765/api/v1") || !containsEnvironment(request.Environment, "TASKAI_TASK_ID=task-a") {
		t.Errorf("启动请求环境 = %#v", request.Environment)
	}
}

func TestManagerBuildsTerminalEnvironmentAfterAssigningID(t *testing.T) {
	backend := &fakeBackend{}
	manager := NewManager(backend, func(Event) {})

	created, err := manager.CreateWithEnvironmentBuilder("task-a", t.TempDir(), "/bin/sh", func(terminalID string) []string {
		return []string{"TASKAI_TERMINAL_ID=" + terminalID}
	}, 80, 24)
	if err != nil {
		t.Fatalf("创建终端: %v", err)
	}
	if request := backend.request(created.ID); !containsEnvironment(request.Environment, "TASKAI_TERMINAL_ID="+created.ID) {
		t.Errorf("按终端 ID 构造的环境 = %#v，期望包含 %q", request.Environment, "TASKAI_TERMINAL_ID="+created.ID)
	}
}

func TestManagerSnapshotsMouseClipboardPolicyForCommand(t *testing.T) {
	backend := &fakeBackend{}
	manager := NewManager(backend, func(Event) {})

	created, err := manager.CreateCommandWithOptionsAndEnvironmentBuilder(
		"task-a",
		t.TempDir(),
		"/bin/sh",
		"claude",
		nil,
		CommandOptions{DisableTaskAIMouseClipboard: true},
		nil,
		80,
		24,
	)
	if err != nil {
		t.Fatalf("创建命令终端: %v", err)
	}
	if !created.DisableTaskAIMouseClipboard {
		t.Fatal("终端信息未快照鼠标剪贴板策略")
	}
	if !backend.request(created.ID).DisableTaskAIMouseClipboard {
		t.Fatal("启动请求未传递鼠标剪贴板策略")
	}

	defaultCreated, err := manager.CreateCommand("task-a", t.TempDir(), "/bin/sh", "codex", nil, 80, 24)
	if err != nil {
		t.Fatalf("创建默认命令终端: %v", err)
	}
	if defaultCreated.DisableTaskAIMouseClipboard {
		t.Fatal("默认命令终端不应禁用 TaskAI 鼠标剪贴板")
	}
}

func TestManagerSnapshotsLaunchCommandOnInfo(t *testing.T) {
	backend := &fakeBackend{}
	manager := NewManager(backend, func(Event) {})

	commandCreated, err := manager.CreateCommand("task-a", t.TempDir(), "/bin/sh", "codex", nil, 80, 24)
	if err != nil {
		t.Fatalf("创建命令终端: %v", err)
	}
	if commandCreated.Command != "codex" {
		t.Fatalf("命令终端 Info.Command = %q，期望 codex", commandCreated.Command)
	}

	shellCreated, err := manager.Create("task-a", t.TempDir(), "/bin/zsh", 80, 24)
	if err != nil {
		t.Fatalf("创建纯 shell 终端: %v", err)
	}
	if shellCreated.Command != "/bin/zsh" {
		t.Fatalf("纯 shell 终端 Info.Command = %q，期望 /bin/zsh", shellCreated.Command)
	}
}

func TestManagerPublishesExpectedExitReasonForExplicitClose(t *testing.T) {
	backend := &fakeBackend{}
	events := make(chan Event, 2)
	manager := NewManager(backend, func(event Event) { events <- event })
	created, err := manager.Create("task-a", t.TempDir(), "", 80, 24)
	if err != nil {
		t.Fatalf("创建终端: %v", err)
	}

	if err := manager.Close(created.TaskID, created.ID); err != nil {
		t.Fatalf("主动关闭终端: %v", err)
	}
	for {
		event := receiveEvent(t, events)
		if event.Type == "exited" {
			if event.ExitReason != ExitReasonClosed {
				t.Errorf("主动关闭退出原因 = %q，期望 %q", event.ExitReason, ExitReasonClosed)
			}
			return
		}
	}
}

func TestManagerPublishesNormalExitReasonForNaturalZeroExit(t *testing.T) {
	backend := &fakeBackend{}
	events := make(chan Event, 2)
	manager := NewManager(backend, func(event Event) { events <- event })
	created, err := manager.Create("task-a", t.TempDir(), "", 80, 24)
	if err != nil {
		t.Fatalf("创建终端: %v", err)
	}

	if err := backend.session(created.ID).Close(); err != nil {
		t.Fatalf("模拟终端自然退出: %v", err)
	}
	for {
		event := receiveEvent(t, events)
		if event.Type == "exited" {
			if event.ExitReason != ExitReason("normal") {
				t.Errorf("自然零退出原因 = %q，期望 normal", event.ExitReason)
			}
			if event.ExitCode == nil || *event.ExitCode != 0 {
				t.Errorf("自然零退出码 = %v，期望 0", event.ExitCode)
			}
			return
		}
	}
}

func TestManagerPublishesUnexpectedExitReasonForNaturalNonZeroExit(t *testing.T) {
	backend := &fakeBackend{}
	events := make(chan Event, 2)
	manager := NewManager(backend, func(event Event) { events <- event })
	created, err := manager.Create("task-a", t.TempDir(), "", 80, 24)
	if err != nil {
		t.Fatalf("创建终端: %v", err)
	}
	backend.session(created.ID).setWaitResult(exitResultFromCode(1), nil)

	if err := backend.session(created.ID).Close(); err != nil {
		t.Fatalf("模拟终端非零退出: %v", err)
	}
	for {
		event := receiveEvent(t, events)
		if event.Type == "exited" {
			if event.ExitReason != ExitReasonUnexpected {
				t.Errorf("自然非零退出原因 = %q，期望 %q", event.ExitReason, ExitReasonUnexpected)
			}
			if event.ExitCode == nil || *event.ExitCode != 1 {
				t.Errorf("自然非零退出码 = %v，期望 1", event.ExitCode)
			}
			return
		}
	}
}

func TestManagerPublishesUnexpectedExitWithoutCodeWhenWaitFails(t *testing.T) {
	backend := &fakeBackend{}
	events := make(chan Event, 2)
	manager := NewManager(backend, func(event Event) { events <- event })
	created, err := manager.Create("task-a", t.TempDir(), "", 80, 24)
	if err != nil {
		t.Fatalf("创建终端: %v", err)
	}
	backend.session(created.ID).setWaitResult(ExitResult{}, errors.New("等待终端进程失败"))

	if err := backend.session(created.ID).Close(); err != nil {
		t.Fatalf("模拟终端等待失败: %v", err)
	}
	for {
		event := receiveEvent(t, events)
		if event.Type == "exited" {
			if event.ExitReason != ExitReasonUnexpected {
				t.Errorf("等待失败退出原因 = %q，期望 %q", event.ExitReason, ExitReasonUnexpected)
			}
			if event.ExitCode != nil {
				t.Errorf("等待失败退出码 = %v，期望缺失", event.ExitCode)
			}
			return
		}
	}
}

func TestManagerPreservesExplicitCloseReasonForNonZeroExit(t *testing.T) {
	backend := &fakeBackend{}
	events := make(chan Event, 2)
	manager := NewManager(backend, func(event Event) { events <- event })
	created, err := manager.Create("task-a", t.TempDir(), "", 80, 24)
	if err != nil {
		t.Fatalf("创建终端: %v", err)
	}
	backend.session(created.ID).setWaitResult(exitResultFromCode(1), nil)

	if err := manager.Close(created.TaskID, created.ID); err != nil {
		t.Fatalf("主动关闭终端: %v", err)
	}
	for {
		event := receiveEvent(t, events)
		if event.Type == "exited" {
			if event.ExitReason != ExitReasonClosed {
				t.Errorf("主动关闭退出原因 = %q，期望 %q", event.ExitReason, ExitReasonClosed)
			}
			if event.ExitCode == nil || *event.ExitCode != 1 {
				t.Errorf("主动关闭退出码 = %v，期望 1", event.ExitCode)
			}
			return
		}
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

func receiveEventsThroughExit(t *testing.T, events <-chan Event) []Event {
	t.Helper()
	received := make([]Event, 0, 4)
	for {
		event := receiveEvent(t, events)
		received = append(received, event)
		if event.Type == "exited" {
			return received
		}
	}
}

type fakeBackend struct {
	mu       sync.Mutex
	sessions map[string]*fakeSession
	requests map[string]StartRequest
}

func (backend *fakeBackend) Start(request StartRequest) (Session, error) {
	backend.mu.Lock()
	defer backend.mu.Unlock()
	if backend.sessions == nil {
		backend.sessions = make(map[string]*fakeSession)
	}
	if backend.requests == nil {
		backend.requests = make(map[string]StartRequest)
	}
	id := request.ID
	if id == "" {
		id = fmt.Sprintf("terminal-%d", len(backend.sessions)+1)
	}
	session := newFakeSession(id)
	backend.sessions[id] = session
	backend.requests[id] = request
	return session, nil
}

func (backend *fakeBackend) session(id string) *fakeSession {
	backend.mu.Lock()
	defer backend.mu.Unlock()
	return backend.sessions[id]
}

func (backend *fakeBackend) request(id string) StartRequest {
	backend.mu.Lock()
	defer backend.mu.Unlock()
	return backend.requests[id]
}

type fakeSession struct {
	id         string
	reader     *io.PipeReader
	writer     *io.PipeWriter
	mu         sync.Mutex
	writes     string
	writeCount int
	closed     bool
	waitResult ExitResult
	waitErr    error
	closeOnce  sync.Once
}

type singleReadSession struct {
	id   string
	data []byte
	err  error
	read bool
}

func (session *singleReadSession) ID() string { return session.id }

func (session *singleReadSession) Read(buffer []byte) (int, error) {
	if session.read {
		return 0, io.EOF
	}
	session.read = true
	return copy(buffer, session.data), session.err
}

func (session *singleReadSession) Write(data []byte) (int, error) { return len(data), nil }

func (session *singleReadSession) Resize(uint16, uint16) error { return nil }

func (session *singleReadSession) Wait() (ExitResult, error) { return exitResultFromCode(0), nil }

func (session *singleReadSession) Close() error { return nil }

func newFakeSession(id string) *fakeSession {
	reader, writer := io.Pipe()
	return &fakeSession{id: id, reader: reader, writer: writer, waitResult: exitResultFromCode(0)}
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
	session.writeCount++
	return len(data), nil
}

func (session *fakeSession) Resize(uint16, uint16) error { return nil }

func (session *fakeSession) Wait() (ExitResult, error) {
	session.mu.Lock()
	defer session.mu.Unlock()
	return session.waitResult, session.waitErr
}

func (session *fakeSession) setWaitResult(result ExitResult, waitErr error) {
	session.mu.Lock()
	defer session.mu.Unlock()
	session.waitResult = result
	session.waitErr = waitErr
}

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

func (session *fakeSession) emitBytes(data []byte) { _, _ = session.writer.Write(data) }

func (session *fakeSession) input() string {
	session.mu.Lock()
	defer session.mu.Unlock()
	return session.writes
}

func (session *fakeSession) inputWriteCount() int {
	session.mu.Lock()
	defer session.mu.Unlock()
	return session.writeCount
}

func (session *fakeSession) wasClosed() bool {
	session.mu.Lock()
	defer session.mu.Unlock()
	return session.closed
}

func containsEnvironment(environment []string, value string) bool {
	for _, entry := range environment {
		if entry == value {
			return true
		}
	}
	return false
}
