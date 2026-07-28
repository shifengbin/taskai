package task

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

var (
	ErrTitleRequired          = errors.New("任务标题不能为空")
	ErrInvalidColor           = errors.New("任务颜色必须是十六进制颜色值")
	ErrExtraInfoFieldRequired = errors.New("额外信息字段不能为空")
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
	ID            string      `json:"id"`
	Title         string      `json:"title"`
	Description   string      `json:"description"`
	Color         string      `json:"color"`
	Status        Status      `json:"status"`
	CreatedAt     time.Time   `json:"createdAt"`
	CompletedAt   *time.Time  `json:"completedAt,omitempty"`
	WorkspaceRoot string      `json:"workspaceRoot,omitempty"`
	WorkspacePath string      `json:"workspacePath,omitempty"`
	ExtraInfo     []ExtraInfo `json:"extraInfo"`
}

type ExtraInfoParameterDefinition struct {
	Key         string `json:"key"`
	DisplayName string `json:"displayName"`
	Required    bool   `json:"required"`
}

type ExtraInfoField struct {
	Key         string `json:"key"`
	DisplayName string `json:"displayName"`
	Value       string `json:"value"`
}

type ExtraInfoTemplate struct {
	ID             string                         `json:"id"`
	Catalogue      string                         `json:"catalogue"`
	DisplayName    string                         `json:"displayName"`
	Fields         []ExtraInfoField               `json:"fields"`
	Parameters     []ExtraInfoParameterDefinition `json:"parameters"`
	Key            string                         `json:"key,omitempty"`
	KeyDisplayName string                         `json:"keyDisplayName,omitempty"`
	Value          string                         `json:"value,omitempty"`
}

type ExtraInfoParameter struct {
	Key         string `json:"key"`
	DisplayName string `json:"displayName"`
	Required    bool   `json:"required"`
	Value       string `json:"value"`
}

type ExtraInfo struct {
	ID             string               `json:"id"`
	Catalogue      string               `json:"catalogue"`
	DisplayName    string               `json:"displayName"`
	Fields         []ExtraInfoField     `json:"fields"`
	Parameters     []ExtraInfoParameter `json:"parameters"`
	Key            string               `json:"key,omitempty"`
	KeyDisplayName string               `json:"keyDisplayName,omitempty"`
	Value          string               `json:"value,omitempty"`
}

func NewTask(title, description, color string, now time.Time) (Task, error) {
	return Task{
		ID:        newID(),
		Status:    StatusPending,
		CreatedAt: now,
		ExtraInfo: []ExtraInfo{},
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

func (current Task) UpdateExtraInfo(extraInfo []ExtraInfo) (Task, error) {
	normalized, err := NormalizeExtraInfo(extraInfo)
	if err != nil {
		return Task{}, err
	}
	current.ExtraInfo = normalized
	return current, nil
}

func NewExtraInfoTemplate(catalogue, displayName string, fields []ExtraInfoField, parameters []ExtraInfoParameterDefinition) (ExtraInfoTemplate, error) {
	return NormalizeExtraInfoTemplate(ExtraInfoTemplate{
		ID:          newID(),
		Catalogue:   catalogue,
		DisplayName: displayName,
		Fields:      fields,
		Parameters:  parameters,
	})
}

func NormalizeExtraInfoTemplate(current ExtraInfoTemplate) (ExtraInfoTemplate, error) {
	current.ID = strings.TrimSpace(current.ID)
	current.Catalogue = strings.TrimSpace(current.Catalogue)
	current.DisplayName = strings.TrimSpace(current.DisplayName)
	if len(current.Fields) == 0 && (current.Key != "" || current.KeyDisplayName != "" || current.Value != "") {
		current.Fields = []ExtraInfoField{{Key: current.Key, DisplayName: current.KeyDisplayName, Value: current.Value}}
	}
	if current.ID == "" || current.Catalogue == "" || current.DisplayName == "" || len(current.Fields) == 0 {
		return ExtraInfoTemplate{}, ErrExtraInfoFieldRequired
	}
	fields := make([]ExtraInfoField, 0, len(current.Fields))
	fieldKeys := make(map[string]bool, len(current.Fields))
	for _, field := range current.Fields {
		field.Key = strings.TrimSpace(field.Key)
		field.DisplayName = strings.TrimSpace(field.DisplayName)
		field.Value = strings.TrimSpace(field.Value)
		if field.Key == "" || field.DisplayName == "" || field.Value == "" {
			return ExtraInfoTemplate{}, ErrExtraInfoFieldRequired
		}
		if fieldKeys[field.Key] {
			return ExtraInfoTemplate{}, fmt.Errorf("额外信息固定字段键重复: %q", field.Key)
		}
		fieldKeys[field.Key] = true
		fields = append(fields, field)
	}
	current.Fields = fields
	current.Key = ""
	current.KeyDisplayName = ""
	current.Value = ""

	parameters := make([]ExtraInfoParameterDefinition, 0, len(current.Parameters))
	parameterKeys := make(map[string]bool, len(current.Parameters))
	for _, parameter := range current.Parameters {
		parameter.Key = strings.TrimSpace(parameter.Key)
		parameter.DisplayName = strings.TrimSpace(parameter.DisplayName)
		if parameter.Key == "" || parameter.DisplayName == "" {
			return ExtraInfoTemplate{}, ErrExtraInfoFieldRequired
		}
		if fieldKeys[parameter.Key] {
			return ExtraInfoTemplate{}, fmt.Errorf("额外信息参数键不能与固定字段键相同: %q", parameter.Key)
		}
		if parameterKeys[parameter.Key] {
			return ExtraInfoTemplate{}, fmt.Errorf("额外信息参数键重复: %q", parameter.Key)
		}
		parameterKeys[parameter.Key] = true
		parameters = append(parameters, parameter)
	}
	current.Parameters = parameters
	return current, nil
}

func ValidateExtraInfoTemplates(templates []ExtraInfoTemplate) ([]ExtraInfoTemplate, error) {
	normalized := make([]ExtraInfoTemplate, 0, len(templates))
	displayNames := make(map[string]bool, len(templates))
	ids := make(map[string]bool, len(templates))
	for _, template := range templates {
		template, err := NormalizeExtraInfoTemplate(template)
		if err != nil {
			return nil, err
		}
		if ids[template.ID] {
			return nil, fmt.Errorf("额外信息模板 ID 重复: %q", template.ID)
		}
		ids[template.ID] = true
		nameKey := template.Catalogue + "\x00" + template.DisplayName
		if displayNames[nameKey] {
			return nil, fmt.Errorf("同一目录下的信息展示名称重复: %q", template.DisplayName)
		}
		displayNames[nameKey] = true
		normalized = append(normalized, template)
	}
	return normalized, nil
}

func NewExtraInfo(template ExtraInfoTemplate, values map[string]string) (ExtraInfo, error) {
	normalizedTemplate, err := NormalizeExtraInfoTemplate(template)
	if err != nil {
		return ExtraInfo{}, err
	}
	parameters := make([]ExtraInfoParameter, 0, len(normalizedTemplate.Parameters))
	for _, definition := range normalizedTemplate.Parameters {
		value := strings.TrimSpace(values[definition.Key])
		if definition.Required && value == "" {
			return ExtraInfo{}, fmt.Errorf("额外信息参数不能为空: %s", definition.DisplayName)
		}
		parameters = append(parameters, ExtraInfoParameter{
			Key:         definition.Key,
			DisplayName: definition.DisplayName,
			Required:    definition.Required,
			Value:       value,
		})
	}
	return ExtraInfo{
		ID:          normalizedTemplate.ID,
		Catalogue:   normalizedTemplate.Catalogue,
		DisplayName: normalizedTemplate.DisplayName,
		Fields:      append([]ExtraInfoField{}, normalizedTemplate.Fields...),
		Parameters:  parameters,
	}, nil
}

func NormalizeExtraInfo(extraInfo []ExtraInfo) ([]ExtraInfo, error) {
	normalized := make([]ExtraInfo, 0, len(extraInfo))
	ids := make(map[string]bool, len(extraInfo))
	for _, current := range extraInfo {
		definitions := make([]ExtraInfoParameterDefinition, 0, len(current.Parameters))
		values := make(map[string]string, len(current.Parameters))
		for _, parameter := range current.Parameters {
			definitions = append(definitions, ExtraInfoParameterDefinition{
				Key: parameter.Key, DisplayName: parameter.DisplayName, Required: parameter.Required,
			})
			values[parameter.Key] = parameter.Value
		}
		template, err := NormalizeExtraInfoTemplate(ExtraInfoTemplate{
			ID: current.ID, Catalogue: current.Catalogue, DisplayName: current.DisplayName,
			Fields: current.Fields, Key: current.Key, KeyDisplayName: current.KeyDisplayName, Value: current.Value, Parameters: definitions,
		})
		if err != nil {
			return nil, err
		}
		if ids[template.ID] {
			return nil, fmt.Errorf("任务附加信息重复: %q", template.DisplayName)
		}
		ids[template.ID] = true
		item, err := NewExtraInfo(template, values)
		if err != nil {
			return nil, err
		}
		normalized = append(normalized, item)
	}
	return normalized, nil
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
