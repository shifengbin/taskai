package task

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"regexp"
	"strings"
	"time"
)

var (
	ErrTitleRequired = errors.New("任务标题不能为空")
	ErrInvalidColor  = errors.New("任务颜色必须是十六进制颜色值")
)

const DefaultColor = "#4f46e5"

var colorPattern = regexp.MustCompile(`^#[0-9a-f]{6}$`)

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
	Color         string     `json:"color"`
	Status        Status     `json:"status"`
	CreatedAt     time.Time  `json:"createdAt"`
	CompletedAt   *time.Time `json:"completedAt,omitempty"`
	WorkspaceRoot string     `json:"workspaceRoot,omitempty"`
	WorkspacePath string     `json:"workspacePath,omitempty"`
}

func NewTask(title, description, color string, now time.Time) (Task, error) {
	return Task{
		ID:        newID(),
		Status:    StatusPending,
		CreatedAt: now,
	}.UpdateDetails(title, description, color)
}

func (current Task) UpdateDetails(title, description, color string) (Task, error) {
	trimmedTitle := strings.TrimSpace(title)
	if trimmedTitle == "" {
		return Task{}, ErrTitleRequired
	}
	normalizedColor, err := NormalizeColor(color)
	if err != nil {
		return Task{}, err
	}
	current.Title = trimmedTitle
	current.Description = description
	current.Color = normalizedColor
	return current, nil
}

func NormalizeColor(color string) (string, error) {
	normalizedColor := strings.ToLower(strings.TrimSpace(color))
	if normalizedColor == "" {
		return DefaultColor, nil
	}
	if !colorPattern.MatchString(normalizedColor) {
		return "", ErrInvalidColor
	}
	return normalizedColor, nil
}

func newID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return hex.EncodeToString([]byte(time.Now().UTC().Format("20060102150405.000000000")))
	}

	return hex.EncodeToString(bytes)
}
