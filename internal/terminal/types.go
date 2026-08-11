package terminal

import "io"

type State string

const (
	StateActive State = "active"
	StateExited State = "exited"
)

type StartRequest struct {
	ID                          string
	TaskID                      string
	Directory                   string
	ShellPath                   string
	Command                     string
	Arguments                   []string
	Environment                 []string
	DisableTaskAIMouseClipboard bool
	Columns                     uint16
	Rows                        uint16
}

type CommandOptions struct {
	DisableTaskAIMouseClipboard bool
}

type Session interface {
	io.ReadWriteCloser
	ID() string
	Resize(columns, rows uint16) error
	Wait() (ExitResult, error)
}

type ExitResult struct {
	ExitCode *int
}

func exitResultFromCode(exitCode int) ExitResult {
	if exitCode < 0 {
		return ExitResult{}
	}
	return ExitResult{ExitCode: &exitCode}
}

type Backend interface {
	Start(StartRequest) (Session, error)
}

type Event struct {
	TaskID     string     `json:"taskId"`
	TerminalID string     `json:"terminalId"`
	Type       string     `json:"type"`
	Data       string     `json:"data,omitempty"`
	ExitCode   *int       `json:"exitCode,omitempty"`
	ExitReason ExitReason `json:"exitReason,omitempty"`
}

type ExitReason string

const (
	ExitReasonNormal              ExitReason = "normal"
	ExitReasonUnexpected          ExitReason = "unexpected"
	ExitReasonClosed              ExitReason = "closed"
	ExitReasonTaskEnded           ExitReason = "task-ended"
	ExitReasonApplicationShutdown ExitReason = "application-shutdown"
)

type Info struct {
	ID                          string `json:"id"`
	TaskID                      string `json:"taskId"`
	State                       State  `json:"state"`
	DisableTaskAIMouseClipboard bool   `json:"disableTaskAIMouseClipboard"`
	Command                     string `json:"command,omitempty"`
}

type ActiveSession struct {
	ID      string
	Command string
}
