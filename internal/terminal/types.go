package terminal

import "io"

type State string

const (
	StateActive State = "active"
	StateExited State = "exited"
)

type StartRequest struct {
	TaskID    string
	Directory string
	Columns   uint16
	Rows      uint16
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
	TaskID     string `json:"taskId"`
	TerminalID string `json:"terminalId"`
	Type       string `json:"type"`
	Data       string `json:"data,omitempty"`
	ExitCode   *int   `json:"exitCode,omitempty"`
}

type Info struct {
	ID     string `json:"id"`
	TaskID string `json:"taskId"`
	State  State  `json:"state"`
}
