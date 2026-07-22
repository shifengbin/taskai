//go:build !windows

package terminal

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/creack/pty"
)

type unixBackend struct {
	shell string
}

type unixSession struct {
	id   string
	file *os.File
	cmd  *exec.Cmd
	once sync.Once
}

func NewBackend() Backend {
	return &unixBackend{shell: os.Getenv("SHELL")}
}

func (backend *unixBackend) Start(request StartRequest) (Session, error) {
	if err := validateDirectory(request.Directory); err != nil {
		return nil, err
	}

	commandPath := request.Command
	if commandPath != "" && request.ShellPath != "" {
		arguments := append([]string{"-ic", `exec "$@"`, request.ShellPath, commandPath}, request.Arguments...)
		command := exec.Command(request.ShellPath, arguments...)
		command.Dir = request.Directory
		command.Env = embeddedTerminalEnvironment()
		return startUnixCommand(command, request)
	}
	if commandPath == "" {
		commandPath = request.ShellPath
	}
	if commandPath == "" {
		commandPath = backend.shell
	}
	if commandPath == "" {
		commandPath = "/bin/sh"
	}
	command := exec.Command(commandPath, request.Arguments...)
	command.Dir = request.Directory
	command.Env = embeddedTerminalEnvironment()
	return startUnixCommand(command, request)
}

func startUnixCommand(command *exec.Cmd, request StartRequest) (Session, error) {
	file, err := pty.StartWithSize(command, &pty.Winsize{
		Cols: normalizedDimension(request.Columns, 80),
		Rows: normalizedDimension(request.Rows, 24),
	})
	if err != nil {
		return nil, fmt.Errorf("启动终端失败: %w", err)
	}

	return &unixSession{id: newSessionID(), file: file, cmd: command}, nil
}

func embeddedTerminalEnvironment() []string {
	environment := os.Environ()
	for index, entry := range environment {
		if strings.HasPrefix(entry, "TERM=") {
			environment[index] = "TERM=xterm-256color"
			return environment
		}
	}
	return append(environment, "TERM=xterm-256color")
}

func (session *unixSession) ID() string { return session.id }

func (session *unixSession) Read(data []byte) (int, error) { return session.file.Read(data) }

func (session *unixSession) Write(data []byte) (int, error) { return session.file.Write(data) }

func (session *unixSession) Resize(columns, rows uint16) error {
	return pty.Setsize(session.file, &pty.Winsize{Cols: columns, Rows: rows})
}

func (session *unixSession) Wait() error { return session.cmd.Wait() }

func (session *unixSession) Close() error {
	var closeError error
	session.once.Do(func() {
		closeError = session.file.Close()
		if session.cmd.Process != nil {
			if err := session.cmd.Process.Kill(); err != nil && !errors.Is(err, os.ErrProcessDone) {
				closeError = errors.Join(closeError, err)
			}
		}
	})
	return closeError
}
