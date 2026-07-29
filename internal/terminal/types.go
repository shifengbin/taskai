package terminal

import "io"

type State string

const (
	StateActive State = "active"
	StateExited State = "exited"
)

type StartRequest struct {
	ID          string
	TaskID      string
	Directory   string
	ShellPath   string
	Command     string
	Arguments   []string
	Environment []string
	Columns     uint16
	Rows        uint16
}

type Session interface {
	io.ReadWriteCloser
	ID() string
	Resize(columns, rows uint16) error
	Wait() error
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
	ExitReasonUnexpected          ExitReason = "unexpected"
	ExitReasonClosed              ExitReason = "closed"
	ExitReasonTaskEnded           ExitReason = "task-ended"
	ExitReasonApplicationShutdown ExitReason = "application-shutdown"
)

type Info struct {
	ID     string `json:"id"`
	TaskID string `json:"taskId"`
	State  State  `json:"state"`
}

type ActiveSession struct {
	ID      string
	Command string
}
