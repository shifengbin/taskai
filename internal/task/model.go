package task

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
	"time"
)

var ErrTitleRequired = errors.New("任务标题不能为空")

type Status string

const (
	StatusPending   Status = "pending"
	StatusRunning   Status = "running"
	StatusCompleted Status = "completed"
)

type Task struct {
	ID            string     `json:"id"`
	Title         string     `json:"title"`
	Description   string     `json:"description"`
	Status        Status     `json:"status"`
	CreatedAt     time.Time  `json:"createdAt"`
	CompletedAt   *time.Time `json:"completedAt,omitempty"`
	WorkspaceRoot string     `json:"workspaceRoot,omitempty"`
	WorkspacePath string     `json:"workspacePath,omitempty"`
}

func NewTask(title, description string, now time.Time) (Task, error) {
	trimmedTitle := strings.TrimSpace(title)
	if trimmedTitle == "" {
		return Task{}, ErrTitleRequired
	}

	return Task{
		ID:          newID(),
		Title:       trimmedTitle,
		Description: description,
		Status:      StatusPending,
		CreatedAt:   now,
	}, nil
}

func newID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return hex.EncodeToString([]byte(time.Now().UTC().Format("20060102150405.000000000")))
	}

	return hex.EncodeToString(bytes)
}
