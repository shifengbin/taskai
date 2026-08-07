package quickinput

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"unicode/utf8"
)

const MaxNameLength = 100

// QuickInput is a reusable text fragment inserted into an active terminal.
type QuickInput struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Content string `json:"content"`
}

func New(name, content string) (QuickInput, error) {
	id, err := newID()
	if err != nil {
		return QuickInput{}, err
	}
	return Normalize(QuickInput{ID: id, Name: name, Content: content})
}

func Normalize(next QuickInput) (QuickInput, error) {
	next.ID = strings.TrimSpace(next.ID)
	next.Name = strings.TrimSpace(next.Name)
	if next.ID == "" {
		return QuickInput{}, fmt.Errorf("快捷输入 ID 不能为空")
	}
	if next.Name == "" {
		return QuickInput{}, fmt.Errorf("快捷输入名称不能为空")
	}
	if utf8.RuneCountInString(next.Name) > MaxNameLength {
		return QuickInput{}, fmt.Errorf("快捷输入名称不能超过 %d 个字符", MaxNameLength)
	}
	if strings.TrimSpace(next.Content) == "" {
		return QuickInput{}, fmt.Errorf("快捷输入内容必须包含非空白字符")
	}
	return next, nil
}

func newID() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("生成快捷输入 ID: %w", err)
	}
	return "quick-input-" + hex.EncodeToString(bytes), nil
}
