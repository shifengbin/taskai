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
	ErrInvalidLifecycleHook   = errors.New("不支持的生命周期钩子")
	ErrInvalidLifecycleState  = errors.New("不支持的生命周期执行状态")
)

const DefaultColor = "#4f46e5"

var colorPattern = regexp.MustCompile(`^#[0-9a-f]{6}$`)

type Status string

const (
	StatusPending   Status = "pending"
	StatusRunning   Status = "running"
	StatusCompleted Status = "completed"
)

type LifecycleHook string

const (
	LifecycleHookBeforeStart LifecycleHook = "beforeStart"
	LifecycleHookPostStart   LifecycleHook = "postStart"
	LifecycleHookBeforeEnd   LifecycleHook = "beforeEnd"
	LifecycleHookPostEnd     LifecycleHook = "postEnd"
	LifecycleHookUpdateTask  LifecycleHook = "updateTask"
)

type LifecycleExecutionState string

const (
	LifecycleExecutionRunning LifecycleExecutionState = "running"
	LifecycleExecutionFailed  LifecycleExecutionState = "failed"
)

type LifecycleExecution struct {
	RunID              string                  `json:"runId,omitempty"`
	Revision           int                     `json:"revision,omitempty"`
	Hook               LifecycleHook           `json:"hook"`
	ChainID            string                  `json:"chainId"`
	CurrentCommandID   string                  `json:"currentCommandId,omitempty"`
	CurrentCommandName string                  `json:"currentCommandName,omitempty"`
	CurrentIndex       int                     `json:"currentIndex"`
	CommandCount       int                     `json:"commandCount"`
	State              LifecycleExecutionState `json:"state"`
	Error              string                  `json:"error,omitempty"`
}

type ExtraInfoParameterInputType string

const (
	ExtraInfoParameterInputText     ExtraInfoParameterInputType = "text"
	ExtraInfoParameterInputCheckbox ExtraInfoParameterInputType = "checkbox"
)

type Task struct {
	ID                 string                   `json:"id"`
	Title              string                   `json:"title"`
	Description        string                   `json:"description"`
	Color              string                   `json:"color"`
	Status             Status                   `json:"status"`
	Shelved            bool                     `json:"shelved"`
	CreatedAt          time.Time                `json:"createdAt"`
	CompletedAt        *time.Time               `json:"completedAt,omitempty"`
	WorkspaceRoot      string                   `json:"workspaceRoot,omitempty"`
	WorkspacePath      string                   `json:"workspacePath,omitempty"`
	ExtraInfo          []TaskExtraInfo          `json:"extraInfo"`
	LifecycleChains    map[LifecycleHook]string `json:"lifecycleChains"`
	LifecycleExecution *LifecycleExecution      `json:"lifecycleExecution,omitempty"`
}

type ExtraInfoParameterDefinition struct {
	Key         string                      `json:"key"`
	DisplayName string                      `json:"displayName"`
	Required    bool                        `json:"required"`
	InputType   ExtraInfoParameterInputType `json:"inputType"`
}

type ExtraInfoField struct {
	Key          string `json:"key"`
	DisplayName  string `json:"displayName"`
	Value        string `json:"value,omitempty"`
	DefaultValue string `json:"defaultValue,omitempty"`
}

type ExtraInfoTemplate struct {
	ID             string                         `json:"id"`
	Catalogue      string                         `json:"catalogue"`
	Fields         []ExtraInfoField               `json:"fields"`
	Parameters     []ExtraInfoParameterDefinition `json:"parameters"`
	BuiltIn        bool                           `json:"builtIn"`
	DisplayName    string                         `json:"displayName,omitempty"`
	Key            string                         `json:"key,omitempty"`
	KeyDisplayName string                         `json:"keyDisplayName,omitempty"`
	Value          string                         `json:"value,omitempty"`
}

type ExtraInfo struct {
	ID         string               `json:"id"`
	TemplateID string               `json:"templateId"`
	Catalogue  string               `json:"catalogue"`
	Fields     []ExtraInfoField     `json:"fields"`
	Parameters []ExtraInfoParameter `json:"parameters"`
}

type ExtraInfoParameter struct {
	Key         string                      `json:"key"`
	DisplayName string                      `json:"displayName"`
	Required    bool                        `json:"required"`
	InputType   ExtraInfoParameterInputType `json:"inputType"`
	Value       string                      `json:"value"`
}

type TaskExtraInfo struct {
	ID             string               `json:"id"`
	InformationID  string               `json:"informationId,omitempty"`
	TemplateID     string               `json:"templateId,omitempty"`
	Catalogue      string               `json:"catalogue"`
	DisplayName    string               `json:"displayName,omitempty"`
	Fields         []ExtraInfoField     `json:"fields"`
	Parameters     []ExtraInfoParameter `json:"parameters"`
	Key            string               `json:"key,omitempty"`
	KeyDisplayName string               `json:"keyDisplayName,omitempty"`
	Value          string               `json:"value,omitempty"`
}

func NewTask(title, description, color string, now time.Time) (Task, error) {
	return Task{
		ID:              newID(),
		Status:          StatusPending,
		CreatedAt:       now,
		ExtraInfo:       []TaskExtraInfo{},
		LifecycleChains: map[LifecycleHook]string{},
	}.UpdateDetails(title, description, color)
}

func IsLifecycleHook(hook LifecycleHook) bool {
	switch hook {
	case LifecycleHookBeforeStart, LifecycleHookPostStart, LifecycleHookBeforeEnd, LifecycleHookPostEnd, LifecycleHookUpdateTask:
		return true
	default:
		return false
	}
}

func NormalizeLifecycleChains(chains map[LifecycleHook]string) (map[LifecycleHook]string, error) {
	normalized := make(map[LifecycleHook]string, len(chains))
	for hook, chainID := range chains {
		if !IsLifecycleHook(hook) {
			return nil, fmt.Errorf("%w: %q", ErrInvalidLifecycleHook, hook)
		}
		if normalizedID := strings.TrimSpace(chainID); normalizedID != "" {
			normalized[hook] = normalizedID
		}
	}
	return normalized, nil
}

func NormalizeLifecycleExecution(execution *LifecycleExecution) (*LifecycleExecution, error) {
	if execution == nil {
		return nil, nil
	}
	normalized := *execution
	if !IsLifecycleHook(normalized.Hook) {
		return nil, fmt.Errorf("%w: %q", ErrInvalidLifecycleHook, normalized.Hook)
	}
	normalized.RunID = strings.TrimSpace(normalized.RunID)
	normalized.ChainID = strings.TrimSpace(normalized.ChainID)
	normalized.CurrentCommandID = strings.TrimSpace(normalized.CurrentCommandID)
	normalized.CurrentCommandName = strings.TrimSpace(normalized.CurrentCommandName)
	normalized.Error = strings.TrimSpace(normalized.Error)
	if normalized.ChainID == "" {
		return nil, fmt.Errorf("生命周期执行记录缺少命令链")
	}
	if normalized.State != LifecycleExecutionRunning && normalized.State != LifecycleExecutionFailed {
		return nil, fmt.Errorf("%w: %q", ErrInvalidLifecycleState, normalized.State)
	}
	if normalized.CommandCount <= 0 || normalized.CurrentIndex <= 0 || normalized.CurrentIndex > normalized.CommandCount {
		return nil, fmt.Errorf("生命周期执行进度无效")
	}
	if normalized.Revision < 0 || (normalized.RunID == "" && normalized.Revision != 0) || (normalized.RunID != "" && normalized.Revision == 0) {
		return nil, fmt.Errorf("生命周期执行版本无效")
	}
	return &normalized, nil
}

func NewLifecycleExecutionRunID() string {
	return newID()
}

func (current Task) IsLifecycleLocked() bool {
	return current.LifecycleExecution != nil && (current.LifecycleExecution.State == LifecycleExecutionRunning || current.LifecycleExecution.State == LifecycleExecutionFailed)
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

func (current Task) UpdateExtraInfo(extraInfo []TaskExtraInfo) (Task, error) {
	normalized, err := ValidateTaskExtraInfo(extraInfo)
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
		current.Fields = []ExtraInfoField{{Key: current.Key, DisplayName: current.KeyDisplayName, DefaultValue: current.Value}}
	}
	if !containsExtraInfoField(current.Fields, "name") {
		current.Fields = append([]ExtraInfoField{{Key: "name", DisplayName: "名称"}}, current.Fields...)
	}
	if current.ID == "" || current.Catalogue == "" || len(current.Fields) == 0 {
		return ExtraInfoTemplate{}, ErrExtraInfoFieldRequired
	}
	fields := make([]ExtraInfoField, 0, len(current.Fields))
	fieldKeys := make(map[string]bool, len(current.Fields))
	for _, field := range current.Fields {
		field.Key = strings.TrimSpace(field.Key)
		field.DisplayName = strings.TrimSpace(field.DisplayName)
		field.DefaultValue = strings.TrimSpace(field.DefaultValue)
		if field.DefaultValue == "" {
			field.DefaultValue = strings.TrimSpace(field.Value)
		}
		field.Value = ""
		if field.Key == "" || field.DisplayName == "" {
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
		parameter.InputType = NormalizeExtraInfoParameterInputType(parameter.InputType)
		if parameter.InputType == ExtraInfoParameterInputCheckbox {
			parameter.Required = false
		}
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
	if current.BuiltIn {
		if err := validateBuiltInGitTemplate(current); err != nil {
			return ExtraInfoTemplate{}, err
		}
	}
	return current, nil
}

func containsExtraInfoField(fields []ExtraInfoField, key string) bool {
	for _, field := range fields {
		if strings.TrimSpace(field.Key) == key {
			return true
		}
	}
	return false
}

func ValidateExtraInfoTemplates(templates []ExtraInfoTemplate) ([]ExtraInfoTemplate, error) {
	normalized := make([]ExtraInfoTemplate, 0, len(templates))
	catalogues := make(map[string]bool, len(templates))
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
		if catalogues[template.Catalogue] {
			return nil, fmt.Errorf("额外信息模板分类重复: %q", template.Catalogue)
		}
		catalogues[template.Catalogue] = true
		normalized = append(normalized, template)
	}
	return normalized, nil
}

func NewExtraInfo(template ExtraInfoTemplate, values map[string]string) (ExtraInfo, error) {
	return NewExtraInfoWithParameters(template, values, nil)
}

func NewExtraInfoWithParameters(template ExtraInfoTemplate, values map[string]string, parameters []ExtraInfoParameter) (ExtraInfo, error) {
	normalizedTemplate, err := NormalizeExtraInfoTemplate(template)
	if err != nil {
		return ExtraInfo{}, err
	}
	fields := make([]ExtraInfoField, 0, len(normalizedTemplate.Fields))
	fieldDefinitions := make(map[string]ExtraInfoField, len(normalizedTemplate.Fields))
	for _, definition := range normalizedTemplate.Fields {
		fieldDefinitions[definition.Key] = definition
	}
	for key := range values {
		if _, ok := fieldDefinitions[strings.TrimSpace(key)]; !ok {
			return ExtraInfo{}, fmt.Errorf("额外信息包含未定义的固定字段: %q", key)
		}
	}
	for _, definition := range normalizedTemplate.Fields {
		value, ok := values[definition.Key]
		if !ok {
			value = definition.DefaultValue
		}
		fields = append(fields, ExtraInfoField{
			Key:         definition.Key,
			DisplayName: definition.DisplayName,
			Value:       strings.TrimSpace(value),
		})
	}
	information, err := NormalizeExtraInfo(ExtraInfo{
		ID:         newID(),
		TemplateID: normalizedTemplate.ID,
		Catalogue:  normalizedTemplate.Catalogue,
		Fields:     fields,
		Parameters: parameters,
	})
	if err != nil {
		return ExtraInfo{}, err
	}
	if err := validateExtraInfoParametersForTemplate(information, normalizedTemplate); err != nil {
		return ExtraInfo{}, err
	}
	return information, nil
}

func NormalizeExtraInfo(current ExtraInfo) (ExtraInfo, error) {
	current.ID = strings.TrimSpace(current.ID)
	current.TemplateID = strings.TrimSpace(current.TemplateID)
	current.Catalogue = strings.TrimSpace(current.Catalogue)
	if current.ID == "" || current.TemplateID == "" || current.Catalogue == "" {
		return ExtraInfo{}, ErrExtraInfoFieldRequired
	}
	fields, err := normalizeExtraInfoFields(current.Fields)
	if err != nil {
		return ExtraInfo{}, err
	}
	if ExtraInfoName(ExtraInfo{Fields: fields}) == "" {
		return ExtraInfo{}, fmt.Errorf("额外信息名称不能为空")
	}
	parameters, err := normalizeExtraInfoParameters(current.Parameters, fields)
	if err != nil {
		return ExtraInfo{}, err
	}
	current.Fields = fields
	current.Parameters = parameters
	return current, nil
}

func ValidateExtraInfos(infos []ExtraInfo) ([]ExtraInfo, error) {
	normalized := make([]ExtraInfo, 0, len(infos))
	ids := make(map[string]bool, len(infos))
	for _, current := range infos {
		item, err := NormalizeExtraInfo(current)
		if err != nil {
			return nil, err
		}
		if ids[item.ID] {
			return nil, fmt.Errorf("额外信息 ID 重复: %q", item.ID)
		}
		ids[item.ID] = true
		normalized = append(normalized, item)
	}
	return normalized, nil
}

func ExtraInfoName(info ExtraInfo) string {
	for _, field := range info.Fields {
		if field.Key == "name" {
			return strings.TrimSpace(field.Value)
		}
	}
	return ""
}

func NewTaskExtraInfo(information ExtraInfo, template ExtraInfoTemplate, values map[string]string, additional []ExtraInfoParameter) (TaskExtraInfo, error) {
	normalizedInformation, err := NormalizeExtraInfo(information)
	if err != nil {
		return TaskExtraInfo{}, err
	}
	normalizedTemplate, err := NormalizeExtraInfoTemplate(template)
	if err != nil {
		return TaskExtraInfo{}, err
	}
	if normalizedInformation.TemplateID != normalizedTemplate.ID || normalizedInformation.Catalogue != normalizedTemplate.Catalogue {
		return TaskExtraInfo{}, fmt.Errorf("额外信息不属于所选模板")
	}
	if err := validateExtraInfoParametersForTemplate(normalizedInformation, normalizedTemplate); err != nil {
		return TaskExtraInfo{}, err
	}
	informationFields := make(map[string]ExtraInfoField, len(normalizedInformation.Fields))
	for _, field := range normalizedInformation.Fields {
		informationFields[field.Key] = field
	}
	fields := make([]ExtraInfoField, 0, len(normalizedTemplate.Fields))
	for _, definition := range normalizedTemplate.Fields {
		field, ok := informationFields[definition.Key]
		if !ok {
			return TaskExtraInfo{}, fmt.Errorf("额外信息缺少固定字段: %s", definition.DisplayName)
		}
		fields = append(fields, ExtraInfoField{
			Key:         definition.Key,
			DisplayName: definition.DisplayName,
			Value:       field.Value,
		})
	}

	definitions := make(map[string]ExtraInfoParameterDefinition, len(normalizedTemplate.Parameters)+len(normalizedInformation.Parameters))
	for _, definition := range normalizedTemplate.Parameters {
		definitions[definition.Key] = definition
	}
	for _, definition := range normalizedInformation.Parameters {
		definitions[definition.Key] = ExtraInfoParameterDefinition{
			Key:         definition.Key,
			DisplayName: definition.DisplayName,
			Required:    definition.Required,
			InputType:   definition.InputType,
		}
	}
	for key := range values {
		if _, ok := definitions[strings.TrimSpace(key)]; !ok {
			return TaskExtraInfo{}, fmt.Errorf("额外信息包含未定义的动态参数: %q", key)
		}
	}
	parameters := make([]ExtraInfoParameter, 0, len(normalizedTemplate.Parameters)+len(normalizedInformation.Parameters)+len(additional))
	for _, definition := range normalizedTemplate.Parameters {
		value := strings.TrimSpace(values[definition.Key])
		if definition.InputType == ExtraInfoParameterInputCheckbox && value == "" {
			value = "false"
		}
		parameters = append(parameters, ExtraInfoParameter{
			Key:         definition.Key,
			DisplayName: definition.DisplayName,
			Required:    definition.Required,
			InputType:   definition.InputType,
			Value:       value,
		})
	}
	for _, definition := range normalizedInformation.Parameters {
		value, ok := values[definition.Key]
		if !ok {
			value = definition.Value
		}
		parameters = append(parameters, ExtraInfoParameter{
			Key:         definition.Key,
			DisplayName: definition.DisplayName,
			Required:    definition.Required,
			InputType:   definition.InputType,
			Value:       strings.TrimSpace(value),
		})
	}
	parameters = append(parameters, additional...)

	return NormalizeTaskExtraInfo(TaskExtraInfo{
		ID:            newID(),
		InformationID: normalizedInformation.ID,
		TemplateID:    normalizedTemplate.ID,
		Catalogue:     normalizedTemplate.Catalogue,
		DisplayName:   ExtraInfoName(normalizedInformation),
		Fields:        fields,
		Parameters:    parameters,
	})
}

func NormalizeTaskExtraInfo(current TaskExtraInfo) (TaskExtraInfo, error) {
	current.ID = strings.TrimSpace(current.ID)
	current.InformationID = strings.TrimSpace(current.InformationID)
	current.TemplateID = strings.TrimSpace(current.TemplateID)
	current.Catalogue = strings.TrimSpace(current.Catalogue)
	current.DisplayName = strings.TrimSpace(current.DisplayName)
	if len(current.Fields) == 0 && (current.Key != "" || current.KeyDisplayName != "" || current.Value != "") {
		current.Fields = []ExtraInfoField{{Key: current.Key, DisplayName: current.KeyDisplayName, Value: current.Value}}
	}
	if current.ID == "" || current.Catalogue == "" || len(current.Fields) == 0 {
		return TaskExtraInfo{}, ErrExtraInfoFieldRequired
	}
	fields, err := normalizeExtraInfoFields(current.Fields)
	if err != nil {
		return TaskExtraInfo{}, err
	}
	fieldKeys := make(map[string]bool, len(fields))
	for _, field := range fields {
		fieldKeys[field.Key] = true
	}
	parameters := make([]ExtraInfoParameter, 0, len(current.Parameters))
	parameterKeys := make(map[string]bool, len(current.Parameters))
	for _, parameter := range current.Parameters {
		parameter, err = normalizeExtraInfoParameter(parameter)
		if err != nil {
			return TaskExtraInfo{}, err
		}
		if parameter.Key == "" || parameter.DisplayName == "" {
			return TaskExtraInfo{}, ErrExtraInfoFieldRequired
		}
		if fieldKeys[parameter.Key] {
			return TaskExtraInfo{}, fmt.Errorf("额外信息参数键不能与固定字段键相同: %q", parameter.Key)
		}
		if parameterKeys[parameter.Key] {
			return TaskExtraInfo{}, fmt.Errorf("额外信息参数键重复: %q", parameter.Key)
		}
		if parameter.Required && parameter.Value == "" {
			return TaskExtraInfo{}, fmt.Errorf("额外信息参数不能为空: %s", parameter.DisplayName)
		}
		parameterKeys[parameter.Key] = true
		parameters = append(parameters, parameter)
	}
	current.Fields = fields
	current.Parameters = parameters
	current.Key = ""
	current.KeyDisplayName = ""
	current.Value = ""
	return current, nil
}

func ValidateTaskExtraInfo(extraInfo []TaskExtraInfo) ([]TaskExtraInfo, error) {
	normalized := make([]TaskExtraInfo, 0, len(extraInfo))
	identifiers := make(map[string]bool, len(extraInfo))
	for _, current := range extraInfo {
		item, err := NormalizeTaskExtraInfo(current)
		if err != nil {
			return nil, err
		}
		identifier := item.InformationID
		if identifier == "" {
			identifier = item.ID
		}
		if identifiers[identifier] {
			return nil, fmt.Errorf("任务附加信息重复: %q", item.DisplayName)
		}
		identifiers[identifier] = true
		normalized = append(normalized, item)
	}
	return normalized, nil
}

func normalizeExtraInfoFields(fields []ExtraInfoField) ([]ExtraInfoField, error) {
	if len(fields) == 0 {
		return nil, ErrExtraInfoFieldRequired
	}
	normalized := make([]ExtraInfoField, 0, len(fields))
	keys := make(map[string]bool, len(fields))
	for _, field := range fields {
		field.Key = strings.TrimSpace(field.Key)
		field.DisplayName = strings.TrimSpace(field.DisplayName)
		field.Value = strings.TrimSpace(field.Value)
		field.DefaultValue = ""
		if field.Key == "" || field.DisplayName == "" {
			return nil, ErrExtraInfoFieldRequired
		}
		if keys[field.Key] {
			return nil, fmt.Errorf("额外信息固定字段键重复: %q", field.Key)
		}
		keys[field.Key] = true
		normalized = append(normalized, field)
	}
	return normalized, nil
}

func normalizeExtraInfoParameters(parameters []ExtraInfoParameter, fields []ExtraInfoField) ([]ExtraInfoParameter, error) {
	fieldKeys := make(map[string]bool, len(fields))
	for _, field := range fields {
		fieldKeys[field.Key] = true
	}
	normalized := make([]ExtraInfoParameter, 0, len(parameters))
	parameterKeys := make(map[string]bool, len(parameters))
	for _, parameter := range parameters {
		parameter, err := normalizeExtraInfoParameter(parameter)
		if err != nil {
			return nil, err
		}
		if parameter.Key == "" || parameter.DisplayName == "" {
			return nil, ErrExtraInfoFieldRequired
		}
		if fieldKeys[parameter.Key] {
			return nil, fmt.Errorf("额外信息参数键不能与固定字段键相同: %q", parameter.Key)
		}
		if parameterKeys[parameter.Key] {
			return nil, fmt.Errorf("额外信息参数键重复: %q", parameter.Key)
		}
		parameterKeys[parameter.Key] = true
		normalized = append(normalized, parameter)
	}
	return normalized, nil
}

func NormalizeExtraInfoParameterInputType(inputType ExtraInfoParameterInputType) ExtraInfoParameterInputType {
	if ExtraInfoParameterInputType(strings.TrimSpace(string(inputType))) == ExtraInfoParameterInputCheckbox {
		return ExtraInfoParameterInputCheckbox
	}
	return ExtraInfoParameterInputText
}

func normalizeExtraInfoParameter(parameter ExtraInfoParameter) (ExtraInfoParameter, error) {
	parameter.Key = strings.TrimSpace(parameter.Key)
	parameter.DisplayName = strings.TrimSpace(parameter.DisplayName)
	parameter.Value = strings.TrimSpace(parameter.Value)
	parameter.InputType = NormalizeExtraInfoParameterInputType(parameter.InputType)
	if parameter.InputType == ExtraInfoParameterInputCheckbox {
		parameter.Required = false
		if parameter.Value == "" {
			parameter.Value = "false"
		}
		if parameter.Value != "true" && parameter.Value != "false" {
			return ExtraInfoParameter{}, fmt.Errorf("复选框参数值必须为 true 或 false: %s", parameter.DisplayName)
		}
	}
	return parameter, nil
}

func validateExtraInfoParametersForTemplate(information ExtraInfo, template ExtraInfoTemplate) error {
	fieldKeys := make(map[string]bool, len(template.Fields))
	for _, field := range template.Fields {
		fieldKeys[field.Key] = true
	}
	parameterKeys := make(map[string]bool, len(template.Parameters))
	for _, parameter := range template.Parameters {
		parameterKeys[parameter.Key] = true
	}
	for _, parameter := range information.Parameters {
		if fieldKeys[parameter.Key] {
			return fmt.Errorf("额外信息参数键不能与模板固定字段键相同: %q", parameter.Key)
		}
		if parameterKeys[parameter.Key] {
			return fmt.Errorf("额外信息参数键不能与模板动态参数键相同: %q", parameter.Key)
		}
	}
	return nil
}

func validateBuiltInGitTemplate(template ExtraInfoTemplate) error {
	if template.Catalogue != "git" {
		return fmt.Errorf("内置额外信息模板必须使用 git 分类")
	}
	fields := make(map[string]string, len(template.Fields))
	for _, field := range template.Fields {
		fields[field.Key] = field.DisplayName
	}
	if fields["name"] != "项目名称" || fields["repository"] != "仓库地址" {
		return fmt.Errorf("Git 内置固定字段不可修改")
	}
	parameters := make(map[string]string, len(template.Parameters))
	for _, parameter := range template.Parameters {
		parameters[parameter.Key] = parameter.DisplayName
	}
	if parameters["branch"] != "仓库分支" {
		return fmt.Errorf("Git 内置动态参数不可修改")
	}
	return nil
}

func BuiltInGitTemplate() ExtraInfoTemplate {
	template, err := NormalizeExtraInfoTemplate(ExtraInfoTemplate{
		ID:        "builtin-extra-info-template-git",
		Catalogue: "git",
		BuiltIn:   true,
		Fields: []ExtraInfoField{
			{Key: "name", DisplayName: "项目名称"},
			{Key: "repository", DisplayName: "仓库地址"},
		},
		Parameters: []ExtraInfoParameterDefinition{{Key: "branch", DisplayName: "仓库分支"}},
	})
	if err != nil {
		panic(err)
	}
	return template
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
