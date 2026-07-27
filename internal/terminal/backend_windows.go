//go:build windows

package terminal

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"syscall"

	"github.com/charmbracelet/x/conpty"
	"golang.org/x/sys/windows"
)

type windowsBackend struct {
	shell string
}

type windowsSession struct {
	id      string
	console *conpty.ConPty
	process *os.Process
	once    sync.Once
}

func NewBackend() Backend {
	return &windowsBackend{shell: os.Getenv("ComSpec")}
}

func (backend *windowsBackend) Start(request StartRequest) (Session, error) {
	if err := validateDirectory(request.Directory); err != nil {
		return nil, err
	}

	commandPath := request.Command
	if commandPath == "" {
		commandPath = request.ShellPath
	}
	if commandPath == "" {
		commandPath = backend.shell
	}
	if commandPath == "" {
		commandPath = os.Getenv("ComSpec")
	}
	if commandPath == "" {
		commandPath = "cmd.exe"
	}
	resolvedCommand, err := exec.LookPath(commandPath)
	if err != nil {
		return nil, fmt.Errorf("找不到 Windows 终端命令: %w", err)
	}
	console, err := conpty.New(int(normalizedDimension(request.Columns, 80)), int(normalizedDimension(request.Rows, 24)), 0)
	if err != nil {
		return nil, fmt.Errorf("创建 Windows ConPTY 失败: %w", err)
	}
	arguments := append([]string{resolvedCommand}, request.Arguments...)
	pid, processHandle, err := console.Spawn(resolvedCommand, arguments, &syscall.ProcAttr{Dir: request.Directory, Env: embeddedTerminalEnvironment(request.Environment)})
	if err != nil {
		_ = console.Close()
		return nil, fmt.Errorf("启动 Windows 终端失败: %w", err)
	}
	process, err := os.FindProcess(pid)
	if err != nil {
		_ = windows.CloseHandle(windows.Handle(processHandle))
		_ = console.Close()
		return nil, fmt.Errorf("获取 Windows 终端进程失败: %w", err)
	}
	_ = windows.CloseHandle(windows.Handle(processHandle))

	return &windowsSession{id: sessionID(request.ID), console: console, process: process}, nil
}

func (session *windowsSession) ID() string { return session.id }

func (session *windowsSession) Read(data []byte) (int, error) { return session.console.Read(data) }

func (session *windowsSession) Write(data []byte) (int, error) { return session.console.Write(data) }

func (session *windowsSession) Resize(columns, rows uint16) error {
	return session.console.Resize(int(columns), int(rows))
}

func (session *windowsSession) Wait() error {
	_, err := session.process.Wait()
	return err
}

func (session *windowsSession) Close() error {
	var closeError error
	session.once.Do(func() {
		if err := session.process.Kill(); err != nil && !errors.Is(err, os.ErrProcessDone) {
			closeError = err
		}
		if err := session.console.Close(); err != nil {
			closeError = errors.Join(closeError, err)
		}
	})
	return closeError
}
