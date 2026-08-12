package storage

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"taskai/internal/quickinput"
	"taskai/internal/settings"
	"taskai/internal/task"
)

// Data is the complete persisted application state.
type Data struct {
	Tasks               []task.Task              `json:"tasks"`
	ExtraInfoTemplates  []task.ExtraInfoTemplate `json:"extraInfoTemplates"`
	ExtraInfos          []task.ExtraInfo         `json:"extraInfos"`
	ExtraInfoCatalogues []string                 `json:"extraInfoCatalogues,omitempty"`
	QuickInputs         []quickinput.QuickInput  `json:"quickInputs"`
	Settings            settings.Settings        `json:"settings"`
}

// Repository persists all application state in one JSON document.
type Repository struct {
	path             string
	defaultSettings  settings.Settings
	loadMu           sync.Mutex
	mutationMu       sync.Mutex
	startupRecovered bool
}

func New(path string, defaultSettings settings.Settings) *Repository {
	return &Repository{path: path, defaultSettings: defaultSettings}
}

func (repository *Repository) Load() (Data, error) {
	repository.loadMu.Lock()
	defer repository.loadMu.Unlock()

	contents, err := os.ReadFile(repository.path)
	if os.IsNotExist(err) {
		data, _, normalizeErr := normalizeData(defaultData(repository.defaultSettings), false)
		if normalizeErr != nil {
			return Data{}, normalizeErr
		}
		if err := repository.Save(data); err != nil {
			return Data{}, err
		}
		repository.startupRecovered = true
		return data, nil
	}
	if err != nil {
		return Data{}, fmt.Errorf("read data file: %w", err)
	}

	var data Data
	if err := json.Unmarshal(contents, &data); err != nil {
		return Data{}, fmt.Errorf("decode data file: %w", err)
	}
	var rawData map[string]json.RawMessage
	if err := json.Unmarshal(contents, &rawData); err == nil {
		var rawSettings map[string]json.RawMessage
		if err := json.Unmarshal(rawData["settings"], &rawSettings); err == nil {
			if _, present := rawSettings["terminalShortcuts"]; !present {
				data.Settings.TerminalShortcuts = []settings.TerminalShortcut{}
			}
			if _, present := rawSettings["terminalNoteTemplate"]; !present {
				data.Settings.TerminalNoteTemplate = settings.DefaultTerminalNoteTemplate()
			}
		}
	}

	normalized, changed, err := normalizeData(data, !repository.startupRecovered)
	if err != nil {
		return Data{}, err
	}
	if changed {
		if err := repository.Save(normalized); err != nil {
			return Data{}, err
		}
	}
	repository.startupRecovered = true
	return normalized, nil
}

func defaultData(defaultSettings settings.Settings) Data {
	return Data{
		Tasks:               []task.Task{},
		ExtraInfoTemplates:  []task.ExtraInfoTemplate{task.BuiltInGitTemplate()},
		ExtraInfos:          []task.ExtraInfo{},
		ExtraInfoCatalogues: []string{"git"},
		QuickInputs:         []quickinput.QuickInput{},
		Settings:            defaultSettings,
	}
}

func normalizeQuickInputs(inputs []quickinput.QuickInput) ([]quickinput.QuickInput, error) {
	normalized := make([]quickinput.QuickInput, len(inputs))
	seenIDs := make(map[string]struct{}, len(inputs))
	for index, input := range inputs {
		current, err := quickinput.Normalize(input)
		if err != nil {
			return nil, fmt.Errorf("normalize quick input: %w", err)
		}
		if _, exists := seenIDs[current.ID]; exists {
			return nil, fmt.Errorf("快捷输入 ID %q 重复", current.ID)
		}
		seenIDs[current.ID] = struct{}{}
		normalized[index] = current
	}
	return normalized, nil
}

func normalizeData(data Data, recoverInterruptedLifecycle bool) (Data, bool, error) {
	changed := false
	normalizedTerminalFontSize := settings.NormalizeTerminalFontSize(data.Settings.TerminalFontSize)
	if data.Settings.TerminalFontSize != normalizedTerminalFontSize {
		data.Settings.TerminalFontSize = normalizedTerminalFontSize
		changed = true
	}
	normalizedTerminalTheme := settings.NormalizeTerminalTheme(data.Settings.TerminalTheme)
	if data.Settings.TerminalTheme != normalizedTerminalTheme {
		data.Settings.TerminalTheme = normalizedTerminalTheme
		changed = true
	}
	normalizedTerminalShortcuts, err := settings.NormalizeTerminalShortcuts(data.Settings.TerminalShortcuts)
	if err != nil {
		return Data{}, false, fmt.Errorf("normalize terminal shortcut settings: %w", err)
	}
	if !sameJSON(data.Settings.TerminalShortcuts, normalizedTerminalShortcuts) {
		changed = true
	}
	data.Settings.TerminalShortcuts = normalizedTerminalShortcuts
	menuItems, err := settings.NormalizeTaskMenuItems(data.Settings.TaskMenuItems)
	if err != nil {
		return Data{}, false, fmt.Errorf("normalize task menu settings: %w", err)
	}
	if !sameJSON(data.Settings.TaskMenuItems, menuItems) {
		changed = true
	}
	data.Settings.TaskMenuItems = menuItems
	templateSettings, err := settings.NormalizeTaskTemplates(data.Settings)
	if err != nil {
		return Data{}, false, fmt.Errorf("normalize task template settings: %w", err)
	}
	if !sameJSON(data.Settings, templateSettings) {
		changed = true
	}
	data.Settings = templateSettings
	lifecycleSettings, err := settings.NormalizeLifecycle(data.Settings)
	if err != nil {
		return Data{}, false, fmt.Errorf("normalize lifecycle settings: %w", err)
	}
	if !sameJSON(data.Settings, lifecycleSettings) {
		changed = true
	}
	data.Settings = lifecycleSettings
	presetSettings, presetChanged := settings.ApplyPresetMigration(data.Settings)
	if presetChanged {
		data.Settings = presetSettings
		changed = true
	}
	validatedLifecycleSettings, err := settings.NormalizeLifecycle(data.Settings)
	if err != nil {
		return Data{}, false, fmt.Errorf("validate lifecycle presets: %w", err)
	}
	if !sameJSON(data.Settings, validatedLifecycleSettings) {
		changed = true
	}
	data.Settings = validatedLifecycleSettings
	if data.Tasks == nil {
		data.Tasks = []task.Task{}
		changed = true
	}
	if data.ExtraInfoTemplates == nil {
		data.ExtraInfoTemplates = []task.ExtraInfoTemplate{}
		changed = true
	}
	if data.ExtraInfos == nil {
		migratedTemplates, migratedInfos, err := migrateLegacyTemplates(data.ExtraInfoTemplates)
		if err != nil {
			return Data{}, false, err
		}
		data.ExtraInfoTemplates = migratedTemplates
		data.ExtraInfos = migratedInfos
		changed = true
	} else {
		for index := range data.ExtraInfoTemplates {
			normalized, err := task.NormalizeExtraInfoTemplate(data.ExtraInfoTemplates[index])
			if err != nil {
				return Data{}, false, fmt.Errorf("normalize extra info template: %w", err)
			}
			if !sameJSON(data.ExtraInfoTemplates[index], normalized) {
				changed = true
			}
			data.ExtraInfoTemplates[index] = normalized
		}
		for index := range data.ExtraInfos {
			normalized, err := task.NormalizeExtraInfo(data.ExtraInfos[index])
			if err != nil {
				return Data{}, false, fmt.Errorf("normalize extra info: %w", err)
			}
			if !sameJSON(data.ExtraInfos[index], normalized) {
				changed = true
			}
			data.ExtraInfos[index] = normalized
		}
	}
	if data.QuickInputs == nil {
		data.QuickInputs = []quickinput.QuickInput{}
		changed = true
	}
	quickInputs, err := normalizeQuickInputs(data.QuickInputs)
	if err != nil {
		return Data{}, false, err
	}
	if !sameJSON(data.QuickInputs, quickInputs) {
		changed = true
	}
	data.QuickInputs = quickInputs

	if ensureBuiltInGitTemplate(&data.ExtraInfoTemplates) {
		changed = true
	}
	templates, err := task.ValidateExtraInfoTemplates(data.ExtraInfoTemplates)
	if err != nil {
		return Data{}, false, fmt.Errorf("validate extra info templates: %w", err)
	}
	if !sameJSON(data.ExtraInfoTemplates, templates) {
		changed = true
	}
	data.ExtraInfoTemplates = templates
	infos, err := task.ValidateExtraInfos(data.ExtraInfos)
	if err != nil {
		return Data{}, false, fmt.Errorf("validate extra infos: %w", err)
	}
	if !sameJSON(data.ExtraInfos, infos) {
		changed = true
	}
	data.ExtraInfos = infos
	for index := range data.Tasks {
		if data.Tasks[index].TemplateFields == nil {
			data.Tasks[index].TemplateFields = map[string]any{}
			changed = true
		}
		templateFields, err := task.NormalizeTaskTemplateValues(data.Tasks[index].TemplateFields)
		if err != nil {
			return Data{}, false, fmt.Errorf("normalize task template fields: %w", err)
		}
		if !sameJSON(data.Tasks[index].TemplateFields, templateFields) {
			changed = true
		}
		data.Tasks[index].TemplateFields = templateFields
		if data.Tasks[index].ExtraInfo == nil {
			data.Tasks[index].ExtraInfo = []task.TaskExtraInfo{}
			changed = true
		}
		for extraInfoIndex := range data.Tasks[index].ExtraInfo {
			normalized, err := task.NormalizeTaskExtraInfo(data.Tasks[index].ExtraInfo[extraInfoIndex])
			if err != nil {
				return Data{}, false, fmt.Errorf("normalize task extra info: %w", err)
			}
			if !sameJSON(data.Tasks[index].ExtraInfo[extraInfoIndex], normalized) {
				changed = true
			}
			data.Tasks[index].ExtraInfo[extraInfoIndex] = normalized
		}
		normalized, err := data.Tasks[index].UpdateExtraInfo(data.Tasks[index].ExtraInfo)
		if err != nil {
			return Data{}, false, fmt.Errorf("validate task extra info: %w", err)
		}
		lifecycleNormalized, lifecycleChanged, err := normalizeTaskLifecycle(normalized, data.Settings.DefaultLifecyclePresetChains(), recoverInterruptedLifecycle)
		if err != nil {
			return Data{}, false, fmt.Errorf("normalize task lifecycle: %w", err)
		}
		if lifecycleChanged {
			changed = true
		}
		data.Tasks[index] = lifecycleNormalized
	}

	catalogues := collectTemplateCatalogues(data.ExtraInfoTemplates)
	if !sameJSON(data.ExtraInfoCatalogues, catalogues) {
		data.ExtraInfoCatalogues = catalogues
		changed = true
	}
	return data, changed, nil
}

func normalizeTaskLifecycle(current task.Task, defaults map[task.LifecycleHook]string, recoverInterruptedLifecycle bool) (task.Task, bool, error) {
	changed := false
	if current.LifecycleChains == nil {
		current.LifecycleChains = defaultLifecycleChainsForTask(current.Status, defaults)
		changed = true
	} else {
		normalizedChains, err := task.NormalizeLifecycleChains(current.LifecycleChains)
		if err != nil {
			return task.Task{}, false, err
		}
		if !sameJSON(current.LifecycleChains, normalizedChains) {
			changed = true
		}
		current.LifecycleChains = normalizedChains
	}

	normalizedExecution, err := task.NormalizeLifecycleExecution(current.LifecycleExecution)
	if err != nil {
		return task.Task{}, false, err
	}
	if recoverInterruptedLifecycle && normalizedExecution != nil && normalizedExecution.State == task.LifecycleExecutionRunning {
		normalizedExecution.State = task.LifecycleExecutionFailed
		if normalizedExecution.Error == "" {
			normalizedExecution.Error = "应用重启中断命令链执行"
		}
	}
	if !sameJSON(current.LifecycleExecution, normalizedExecution) {
		changed = true
	}
	current.LifecycleExecution = normalizedExecution
	return current, changed, nil
}

func defaultLifecycleChainsForTask(status task.Status, defaults map[task.LifecycleHook]string) map[task.LifecycleHook]string {
	chains := make(map[task.LifecycleHook]string)
	if status == task.StatusPending || status == "" {
		if chainID := defaults[task.LifecycleHookBeforeStart]; chainID != "" {
			chains[task.LifecycleHookBeforeStart] = chainID
		}
	}
	if status == task.StatusPending || status == "" || status == task.StatusRunning {
		if chainID := defaults[task.LifecycleHookPostEnd]; chainID != "" {
			chains[task.LifecycleHookPostEnd] = chainID
		}
	}
	return chains
}

func ensureBuiltInGitTemplate(templates *[]task.ExtraInfoTemplate) bool {
	for index := range *templates {
		if (*templates)[index].BuiltIn && (*templates)[index].Catalogue == "git" {
			return false
		}
	}
	for index := range *templates {
		if (*templates)[index].Catalogue == "git" {
			(*templates)[index].Catalogue = nextCatalogue("git-legacy", *templates)
			return true
		}
	}
	*templates = append(*templates, task.BuiltInGitTemplate())
	return true
}

func migrateLegacyTemplates(legacy []task.ExtraInfoTemplate) ([]task.ExtraInfoTemplate, []task.ExtraInfo, error) {
	templates := []task.ExtraInfoTemplate{task.BuiltInGitTemplate()}
	infos := make([]task.ExtraInfo, 0, len(legacy))
	templateBySchema := map[string]task.ExtraInfoTemplate{}

	for index, legacyTemplate := range legacy {
		catalogue := legacyTemplate.Catalogue
		if catalogue == "" {
			catalogue = "legacy"
		}
		if catalogue == "git" {
			catalogue = "git-legacy"
		}
		candidate := legacyTemplate
		candidate.ID = migratedID("template", legacyTemplate.ID, index)
		candidate.Catalogue = catalogue
		candidate.BuiltIn = false
		normalized, err := task.NormalizeExtraInfoTemplate(candidate)
		if err != nil {
			return nil, nil, fmt.Errorf("migrate legacy template %d: %w", index, err)
		}
		signature := templateSignature(normalized)
		template, exists := templateBySchema[signature]
		if !exists {
			normalized.ID = migratedID("template", legacyTemplate.ID, index)
			normalized.Catalogue = nextCatalogue(normalized.Catalogue, templates)
			template = normalized
			templates = append(templates, template)
			templateBySchema[signature] = template
		}

		values := make(map[string]string, len(legacyTemplate.Fields)+1)
		for _, field := range legacyTemplate.Fields {
			values[field.Key] = field.Value
			if values[field.Key] == "" {
				values[field.Key] = field.DefaultValue
			}
		}
		if legacyTemplate.Key != "" {
			values[legacyTemplate.Key] = legacyTemplate.Value
		}
		if displayName := strings.TrimSpace(legacyTemplate.DisplayName); displayName != "" {
			values["name"] = displayName
		}
		if values["name"] == "" {
			values["name"] = fmt.Sprintf("迁移信息 %d", index+1)
		}
		info, err := task.NewExtraInfo(template, values)
		if err != nil {
			return nil, nil, fmt.Errorf("migrate legacy information %d: %w", index, err)
		}
		info.ID = migratedID("information", legacyTemplate.ID, index)
		infos = append(infos, info)
	}
	return templates, infos, nil
}

func migratedID(kind, legacyID string, index int) string {
	if strings.TrimSpace(legacyID) != "" {
		return "migrated-" + kind + "-" + legacyID
	}
	return fmt.Sprintf("migrated-%s-%d", kind, index+1)
}

func templateSignature(template task.ExtraInfoTemplate) string {
	copy := template
	copy.ID = ""
	copy.Catalogue = ""
	copy.BuiltIn = false
	copy.DisplayName = ""
	for index := range copy.Fields {
		copy.Fields[index].DefaultValue = ""
		copy.Fields[index].Value = ""
	}
	contents, _ := json.Marshal(copy)
	return string(contents)
}

func nextCatalogue(base string, templates []task.ExtraInfoTemplate) string {
	base = strings.TrimSpace(base)
	if base == "" {
		base = "custom"
	}
	for suffix := 1; ; suffix++ {
		candidate := base
		if suffix > 1 {
			candidate = fmt.Sprintf("%s-%d", base, suffix)
		}
		found := false
		for _, template := range templates {
			if template.Catalogue == candidate {
				found = true
				break
			}
		}
		if !found {
			return candidate
		}
	}
}

func collectTemplateCatalogues(templates []task.ExtraInfoTemplate) []string {
	catalogues := make([]string, 0, len(templates))
	for _, template := range templates {
		catalogues = append(catalogues, template.Catalogue)
	}
	sort.Strings(catalogues)
	return catalogues
}

func sameJSON(left, right any) bool {
	leftContents, _ := json.Marshal(left)
	rightContents, _ := json.Marshal(right)
	return string(leftContents) == string(rightContents)
}

func (repository *Repository) Save(data Data) error {
	normalized, _, err := normalizeDataForSave(data)
	if err != nil {
		return err
	}
	contents, err := json.MarshalIndent(normalized, "", "  ")
	if err != nil {
		return fmt.Errorf("encode data file: %w", err)
	}
	contents = append(contents, '\n')

	directory := filepath.Dir(repository.path)
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return fmt.Errorf("create data directory: %w", err)
	}
	temporary, err := os.CreateTemp(directory, ".taskai-*.json")
	if err != nil {
		return fmt.Errorf("create temporary data file: %w", err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if _, err := temporary.Write(contents); err != nil {
		temporary.Close()
		return fmt.Errorf("write temporary data file: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close temporary data file: %w", err)
	}
	if err := os.Rename(temporaryPath, repository.path); err != nil {
		return fmt.Errorf("replace data file: %w", err)
	}
	return nil
}

// SaveTaskSnapshot updates tasks without restoring settings from a stale task snapshot.
func (repository *Repository) SaveTaskSnapshot(tasks []task.Task) error {
	repository.mutationMu.Lock()
	defer repository.mutationMu.Unlock()

	data, err := repository.Load()
	if err != nil {
		return err
	}
	data.Tasks = tasks
	return repository.Save(data)
}

func normalizeDataForSave(data Data) (Data, bool, error) {
	if data.ExtraInfos == nil {
		data.ExtraInfos = []task.ExtraInfo{}
	}
	if data.ExtraInfoTemplates == nil {
		data.ExtraInfoTemplates = []task.ExtraInfoTemplate{}
	}
	if data.QuickInputs == nil {
		data.QuickInputs = []quickinput.QuickInput{}
	}
	return normalizeData(data, false)
}

func (repository *Repository) SaveSettings(next settings.Settings) (settings.Settings, error) {
	repository.mutationMu.Lock()
	defer repository.mutationMu.Unlock()

	data, err := repository.Load()
	if err != nil {
		return settings.Settings{}, err
	}
	templatesProvided := next.TaskTemplates != nil
	// 生命周期配置只能通过专用接口更新，普通设置快照不能覆盖最新链定义。
	next.LifecycleCommands = data.Settings.LifecycleCommands
	next.LifecycleChains = data.Settings.LifecycleChains
	next.LifecyclePresets = data.Settings.LifecyclePresets
	next.DefaultLifecyclePresetID = data.Settings.DefaultLifecyclePresetID
	next.LegacyLifecycleDefaultChains = data.Settings.LegacyLifecycleDefaultChains
	validated, err := settings.NormalizeTaskTemplates(next)
	if err != nil {
		return settings.Settings{}, err
	}
	if !templatesProvided {
		validated.TaskTemplates = append([]task.TaskTemplate(nil), data.Settings.TaskTemplates...)
		validated.ActiveTaskTemplateID = data.Settings.ActiveTaskTemplateID
	} else if err := validateTaskTemplateUpdates(data.Settings, validated, data.Tasks); err != nil {
		return settings.Settings{}, err
	}
	if validated.PresetVersion < data.Settings.PresetVersion {
		validated.PresetVersion = data.Settings.PresetVersion
	}
	validated.TerminalShortcuts, err = settings.NormalizeTerminalShortcuts(validated.TerminalShortcuts)
	if err != nil {
		return settings.Settings{}, err
	}
	data.Settings = validated
	if err := repository.Save(data); err != nil {
		return settings.Settings{}, err
	}
	return data.Settings, nil
}

func (repository *Repository) SaveWindowMaximized(maximized bool) error {
	repository.mutationMu.Lock()
	defer repository.mutationMu.Unlock()

	data, err := repository.Load()
	if err != nil {
		return err
	}
	data.Settings.WindowMaximized = maximized
	return repository.Save(data)
}

func validateTaskTemplateUpdates(previous, next settings.Settings, tasks []task.Task) error {
	previousByID := make(map[string]task.TaskTemplate, len(previous.TaskTemplates))
	for _, template := range previous.TaskTemplates {
		previousByID[template.ID] = template
	}
	nextByID := make(map[string]struct{}, len(next.TaskTemplates))
	for _, template := range next.TaskTemplates {
		nextByID[template.ID] = struct{}{}
	}

	for _, current := range tasks {
		if !isActiveTask(current) || current.TaskTemplateID != "" || len(current.TemplateFields) == 0 {
			continue
		}
		for _, template := range previous.TaskTemplates {
			if _, found := nextByID[template.ID]; !found {
				return fmt.Errorf("未执行或执行中的历史任务包含无法确定来源的模板字段，不能删除任务模板")
			}
		}
	}
	for _, template := range previous.TaskTemplates {
		if _, found := nextByID[template.ID]; found {
			continue
		}
		for _, current := range tasks {
			if isActiveTask(current) && current.TaskTemplateID == template.ID {
				return fmt.Errorf("任务模板 %q 仍被未执行或执行中的任务 %q 引用", template.Name, current.Title)
			}
		}
	}

	values := make([]map[string]any, 0, len(tasks))
	for _, current := range tasks {
		values = append(values, current.TemplateFields)
	}
	for _, template := range next.TaskTemplates {
		previousTemplate, found := previousByID[template.ID]
		if !found {
			continue
		}
		if err := task.ValidateTaskTemplateUpdate(previousTemplate, template, values); err != nil {
			return err
		}
	}
	return nil
}

func isActiveTask(current task.Task) bool {
	return current.Status == task.StatusPending || current.Status == task.StatusRunning
}

func (repository *Repository) ListLifecycleCommands() ([]settings.LifecycleCommand, error) {
	data, err := repository.Load()
	if err != nil {
		return nil, err
	}
	return append([]settings.LifecycleCommand(nil), data.Settings.LifecycleCommands...), nil
}

func (repository *Repository) SaveLifecycleCommand(next settings.LifecycleCommand) (settings.LifecycleCommand, error) {
	repository.mutationMu.Lock()
	defer repository.mutationMu.Unlock()

	data, err := repository.Load()
	if err != nil {
		return settings.LifecycleCommand{}, err
	}
	if fixed := fixedLifecycleCommand(next.ID); fixed.ID != "" {
		return settings.LifecycleCommand{}, fmt.Errorf("系统内置生命周期命令不可编辑")
	}
	if len(next.ApplicableHooks) == 0 {
		return settings.LifecycleCommand{}, fmt.Errorf("生命周期命令至少选择一个适用范围")
	}
	if next.ID == "" {
		next.ID, err = newLifecycleID("command")
		if err != nil {
			return settings.LifecycleCommand{}, err
		}
		next.Kind = settings.LifecycleCommandKindCustom
	} else if next.Kind == "" {
		next.Kind = settings.LifecycleCommandKindCustom
	}

	index := lifecycleCommandIndex(data.Settings.LifecycleCommands, next.ID)
	if index >= 0 {
		data.Settings.LifecycleCommands[index] = next
	} else {
		data.Settings.LifecycleCommands = append(data.Settings.LifecycleCommands, next)
	}
	data.Settings, err = settings.NormalizeLifecycle(data.Settings)
	if err != nil {
		return settings.LifecycleCommand{}, err
	}
	if err := repository.Save(data); err != nil {
		return settings.LifecycleCommand{}, err
	}
	return data.Settings.LifecycleCommands[lifecycleCommandIndex(data.Settings.LifecycleCommands, next.ID)], nil
}

func (repository *Repository) DeleteLifecycleCommand(id string) error {
	repository.mutationMu.Lock()
	defer repository.mutationMu.Unlock()

	data, err := repository.Load()
	if err != nil {
		return err
	}
	index := lifecycleCommandIndex(data.Settings.LifecycleCommands, id)
	if index < 0 {
		return fmt.Errorf("生命周期命令 %q 不存在", id)
	}
	if fixed := fixedLifecycleCommand(id); fixed.ID != "" {
		return fmt.Errorf("系统内置生命周期命令不可删除")
	}
	for _, chain := range data.Settings.LifecycleChains {
		for _, reference := range chain.Commands {
			if reference.CommandID == id {
				return fmt.Errorf("生命周期命令 %q 仍被命令链 %q 引用", id, chain.Name)
			}
		}
	}
	data.Settings.LifecycleCommands = append(data.Settings.LifecycleCommands[:index], data.Settings.LifecycleCommands[index+1:]...)
	data.Settings, err = settings.NormalizeLifecycle(data.Settings)
	if err != nil {
		return err
	}
	return repository.Save(data)
}

func (repository *Repository) ListLifecycleCommandChains() ([]settings.LifecycleCommandChain, error) {
	data, err := repository.Load()
	if err != nil {
		return nil, err
	}
	return append([]settings.LifecycleCommandChain(nil), data.Settings.LifecycleChains...), nil
}

func (repository *Repository) ListLifecyclePresets() ([]settings.LifecyclePreset, error) {
	data, err := repository.Load()
	if err != nil {
		return nil, err
	}
	return copyLifecyclePresets(data.Settings.LifecyclePresets), nil
}

func (repository *Repository) SaveLifecyclePreset(next settings.LifecyclePreset) (settings.LifecyclePreset, error) {
	repository.mutationMu.Lock()
	defer repository.mutationMu.Unlock()

	data, err := repository.Load()
	if err != nil {
		return settings.LifecyclePreset{}, err
	}
	next.ID = strings.TrimSpace(next.ID)
	if next.ID == "" {
		next.ID, err = newLifecycleID("preset")
		if err != nil {
			return settings.LifecyclePreset{}, err
		}
	}
	index := lifecyclePresetIndex(data.Settings.LifecyclePresets, next.ID)
	if index >= 0 {
		data.Settings.LifecyclePresets[index] = next
	} else {
		data.Settings.LifecyclePresets = append(data.Settings.LifecyclePresets, next)
	}
	data.Settings, err = settings.NormalizeLifecycle(data.Settings)
	if err != nil {
		return settings.LifecyclePreset{}, err
	}
	if err := repository.Save(data); err != nil {
		return settings.LifecyclePreset{}, err
	}
	return copyLifecyclePreset(data.Settings.LifecyclePresets[lifecyclePresetIndex(data.Settings.LifecyclePresets, next.ID)]), nil
}

func (repository *Repository) CopyLifecyclePreset(id string) (settings.LifecyclePreset, error) {
	data, err := repository.Load()
	if err != nil {
		return settings.LifecyclePreset{}, err
	}
	index := lifecyclePresetIndex(data.Settings.LifecyclePresets, strings.TrimSpace(id))
	if index < 0 {
		return settings.LifecyclePreset{}, fmt.Errorf("生命周期预设 %q 不存在", id)
	}
	copy := copyLifecyclePreset(data.Settings.LifecyclePresets[index])
	copy.ID, err = newLifecycleID("preset")
	if err != nil {
		return settings.LifecyclePreset{}, err
	}
	copy.Name += " 副本"
	return repository.SaveLifecyclePreset(copy)
}

func (repository *Repository) DeleteLifecyclePreset(id string) error {
	repository.mutationMu.Lock()
	defer repository.mutationMu.Unlock()

	data, err := repository.Load()
	if err != nil {
		return err
	}
	id = strings.TrimSpace(id)
	index := lifecyclePresetIndex(data.Settings.LifecyclePresets, id)
	if index < 0 {
		return fmt.Errorf("生命周期预设 %q 不存在", id)
	}
	data.Settings.LifecyclePresets = append(data.Settings.LifecyclePresets[:index], data.Settings.LifecyclePresets[index+1:]...)
	if data.Settings.DefaultLifecyclePresetID == id {
		data.Settings.DefaultLifecyclePresetID = ""
	}
	data.Settings, err = settings.NormalizeLifecycle(data.Settings)
	if err != nil {
		return err
	}
	return repository.Save(data)
}

func (repository *Repository) SaveDefaultLifecyclePreset(id string) (settings.Settings, error) {
	repository.mutationMu.Lock()
	defer repository.mutationMu.Unlock()

	data, err := repository.Load()
	if err != nil {
		return settings.Settings{}, err
	}
	id = strings.TrimSpace(id)
	if id != "" && lifecyclePresetIndex(data.Settings.LifecyclePresets, id) < 0 {
		return settings.Settings{}, fmt.Errorf("生命周期预设 %q 不存在", id)
	}
	data.Settings.DefaultLifecyclePresetID = id
	data.Settings, err = settings.NormalizeLifecycle(data.Settings)
	if err != nil {
		return settings.Settings{}, err
	}
	if err := repository.Save(data); err != nil {
		return settings.Settings{}, err
	}
	return data.Settings, nil
}

func (repository *Repository) SaveLifecycleCommandChain(next settings.LifecycleCommandChain) (settings.LifecycleCommandChain, error) {
	repository.mutationMu.Lock()
	defer repository.mutationMu.Unlock()

	data, err := repository.Load()
	if err != nil {
		return settings.LifecycleCommandChain{}, err
	}
	if len(next.ApplicableHooks) == 0 {
		return settings.LifecycleCommandChain{}, fmt.Errorf("生命周期命令链至少选择一个适用范围")
	}
	if next.ID == "" {
		next.ID, err = newLifecycleID("chain")
		if err != nil {
			return settings.LifecycleCommandChain{}, err
		}
	}
	for _, current := range data.Tasks {
		if current.Status != task.StatusPending && current.Status != task.StatusRunning {
			continue
		}
		for hook, selectedChainID := range current.LifecycleChains {
			if selectedChainID == next.ID && !lifecycleChainSupportsHook(next, hook) {
				return settings.LifecycleCommandChain{}, fmt.Errorf("生命周期命令链 %q 仍被任务 %q 用于 %s，不能移除该适用范围", next.Name, current.Title, hook)
			}
		}
	}
	for _, preset := range data.Settings.LifecyclePresets {
		for hook, selectedChainID := range preset.Chains {
			if selectedChainID == next.ID && !lifecycleChainSupportsHook(next, hook) {
				return settings.LifecycleCommandChain{}, fmt.Errorf("生命周期命令链 %q 仍被预设 %q 用于 %s，不能移除该适用范围", next.Name, preset.Name, hook)
			}
		}
	}
	index := lifecycleCommandChainIndex(data.Settings.LifecycleChains, next.ID)
	if index >= 0 {
		data.Settings.LifecycleChains[index] = next
	} else {
		data.Settings.LifecycleChains = append(data.Settings.LifecycleChains, next)
	}
	data.Settings, err = settings.NormalizeLifecycle(data.Settings)
	if err != nil {
		return settings.LifecycleCommandChain{}, err
	}
	if err := repository.Save(data); err != nil {
		return settings.LifecycleCommandChain{}, err
	}
	return data.Settings.LifecycleChains[lifecycleCommandChainIndex(data.Settings.LifecycleChains, next.ID)], nil
}

func (repository *Repository) CopyLifecycleCommandChain(id string) (settings.LifecycleCommandChain, error) {
	data, err := repository.Load()
	if err != nil {
		return settings.LifecycleCommandChain{}, err
	}
	index := lifecycleCommandChainIndex(data.Settings.LifecycleChains, id)
	if index < 0 {
		return settings.LifecycleCommandChain{}, fmt.Errorf("生命周期命令链 %q 不存在", id)
	}
	copy := data.Settings.LifecycleChains[index]
	copy.ID, err = newLifecycleID("chain")
	if err != nil {
		return settings.LifecycleCommandChain{}, err
	}
	copy.Name = copy.Name + " 副本"
	copy.Commands = append([]settings.LifecycleCommandReference(nil), copy.Commands...)
	for index := range copy.Commands {
		copy.Commands[index].Arguments = append([]string(nil), copy.Commands[index].Arguments...)
	}
	return repository.SaveLifecycleCommandChain(copy)
}

func (repository *Repository) DeleteLifecycleCommandChain(id string) error {
	repository.mutationMu.Lock()
	defer repository.mutationMu.Unlock()

	data, err := repository.Load()
	if err != nil {
		return err
	}
	index := lifecycleCommandChainIndex(data.Settings.LifecycleChains, id)
	if index < 0 {
		return fmt.Errorf("生命周期命令链 %q 不存在", id)
	}
	for _, current := range data.Tasks {
		if current.Status != task.StatusPending && current.Status != task.StatusRunning {
			continue
		}
		for _, selectedID := range current.LifecycleChains {
			if selectedID == id {
				return fmt.Errorf("生命周期命令链 %q 仍被任务 %q 引用", id, current.Title)
			}
		}
	}
	for _, preset := range data.Settings.LifecyclePresets {
		for _, selectedID := range preset.Chains {
			if selectedID == id {
				return fmt.Errorf("生命周期命令链 %q 仍被预设 %q 引用", id, preset.Name)
			}
		}
	}
	data.Settings.LifecycleChains = append(data.Settings.LifecycleChains[:index], data.Settings.LifecycleChains[index+1:]...)
	data.Settings, err = settings.NormalizeLifecycle(data.Settings)
	if err != nil {
		return err
	}
	return repository.Save(data)
}

func lifecycleCommandIndex(commands []settings.LifecycleCommand, id string) int {
	for index, command := range commands {
		if command.ID == id {
			return index
		}
	}
	return -1
}

func lifecycleCommandChainIndex(chains []settings.LifecycleCommandChain, id string) int {
	for index, chain := range chains {
		if chain.ID == id {
			return index
		}
	}
	return -1
}

func lifecyclePresetIndex(presets []settings.LifecyclePreset, id string) int {
	for index, preset := range presets {
		if preset.ID == id {
			return index
		}
	}
	return -1
}

func copyLifecyclePreset(current settings.LifecyclePreset) settings.LifecyclePreset {
	copy := current
	copy.Chains = make(map[task.LifecycleHook]string, len(current.Chains))
	for hook, chainID := range current.Chains {
		copy.Chains[hook] = chainID
	}
	return copy
}

func copyLifecyclePresets(presets []settings.LifecyclePreset) []settings.LifecyclePreset {
	copy := make([]settings.LifecyclePreset, len(presets))
	for index, preset := range presets {
		copy[index] = copyLifecyclePreset(preset)
	}
	return copy
}

func lifecycleChainSupportsHook(chain settings.LifecycleCommandChain, hook task.LifecycleHook) bool {
	for _, applicableHook := range chain.ApplicableHooks {
		if applicableHook == hook {
			return true
		}
	}
	return false
}

func fixedLifecycleCommand(id string) settings.LifecycleCommand {
	for _, command := range settings.DefaultLifecycleCommands() {
		if command.ID == id {
			return command
		}
	}
	return settings.LifecycleCommand{}
}

func newLifecycleID(kind string) (string, error) {
	bytes := make([]byte, 12)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("生成生命周期%s ID: %w", kind, err)
	}
	return fmt.Sprintf("lifecycle-%s-%x", kind, bytes), nil
}

func (repository *Repository) ListExtraInfoTemplates() ([]task.ExtraInfoTemplate, error) {
	data, err := repository.Load()
	if err != nil {
		return nil, err
	}
	return data.ExtraInfoTemplates, nil
}

func (repository *Repository) ListExtraInfos() ([]task.ExtraInfo, error) {
	data, err := repository.Load()
	if err != nil {
		return nil, err
	}
	return data.ExtraInfos, nil
}

func (repository *Repository) SaveExtraInfoTemplate(next task.ExtraInfoTemplate) (task.ExtraInfoTemplate, error) {
	repository.mutationMu.Lock()
	defer repository.mutationMu.Unlock()

	data, err := repository.Load()
	if err != nil {
		return task.ExtraInfoTemplate{}, err
	}

	index := -1
	for candidateIndex, candidate := range data.ExtraInfoTemplates {
		if candidate.ID == next.ID && next.ID != "" {
			index = candidateIndex
			if candidate.BuiltIn {
				next.BuiltIn = true
			}
			break
		}
	}
	if next.ID == "" {
		created, err := task.NewExtraInfoTemplate(next.Catalogue, next.DisplayName, next.Fields, next.Parameters)
		if err != nil {
			return task.ExtraInfoTemplate{}, err
		}
		next = created
	} else {
		normalized, err := task.NormalizeExtraInfoTemplate(next)
		if err != nil {
			return task.ExtraInfoTemplate{}, err
		}
		next = normalized
	}

	if index >= 0 {
		for _, info := range data.ExtraInfos {
			if info.TemplateID == next.ID {
				if info.Catalogue != next.Catalogue {
					return task.ExtraInfoTemplate{}, fmt.Errorf("template category cannot change while information %q still uses it", info.ID)
				}
				if err := ensureInformationFieldsRemain(info, next); err != nil {
					return task.ExtraInfoTemplate{}, err
				}
			}
		}
		data.ExtraInfoTemplates[index] = next
	} else {
		data.ExtraInfoTemplates = append(data.ExtraInfoTemplates, next)
	}
	if _, err := task.ValidateExtraInfoTemplates(data.ExtraInfoTemplates); err != nil {
		return task.ExtraInfoTemplate{}, err
	}
	data.ExtraInfoCatalogues = collectTemplateCatalogues(data.ExtraInfoTemplates)
	if err := repository.Save(data); err != nil {
		return task.ExtraInfoTemplate{}, err
	}
	return next, nil
}

func ensureInformationFieldsRemain(info task.ExtraInfo, template task.ExtraInfoTemplate) error {
	keys := make(map[string]struct{}, len(template.Fields))
	for _, field := range template.Fields {
		keys[field.Key] = struct{}{}
	}
	for _, field := range info.Fields {
		if _, exists := keys[field.Key]; !exists {
			return fmt.Errorf("template field %q is used by information %q and cannot be removed", field.Key, info.ID)
		}
	}
	return nil
}

func (repository *Repository) DeleteExtraInfoTemplate(id string) error {
	repository.mutationMu.Lock()
	defer repository.mutationMu.Unlock()

	data, err := repository.Load()
	if err != nil {
		return err
	}
	index := -1
	for candidateIndex, candidate := range data.ExtraInfoTemplates {
		if candidate.ID == id {
			if candidate.BuiltIn {
				return fmt.Errorf("built-in extra info template cannot be deleted")
			}
			index = candidateIndex
			break
		}
	}
	if index < 0 {
		return fmt.Errorf("extra info template %q does not exist", id)
	}
	for _, info := range data.ExtraInfos {
		if info.TemplateID == id {
			return fmt.Errorf("extra info template %q is still used by information %q", id, info.ID)
		}
	}
	data.ExtraInfoTemplates = append(data.ExtraInfoTemplates[:index], data.ExtraInfoTemplates[index+1:]...)
	data.ExtraInfoCatalogues = collectTemplateCatalogues(data.ExtraInfoTemplates)
	return repository.Save(data)
}

func (repository *Repository) SaveExtraInfo(next task.ExtraInfo) (task.ExtraInfo, error) {
	repository.mutationMu.Lock()
	defer repository.mutationMu.Unlock()

	data, err := repository.Load()
	if err != nil {
		return task.ExtraInfo{}, err
	}
	var template task.ExtraInfoTemplate
	for _, candidate := range data.ExtraInfoTemplates {
		if candidate.ID == next.TemplateID {
			template = candidate
			break
		}
	}
	if template.ID == "" {
		return task.ExtraInfo{}, fmt.Errorf("extra info template %q does not exist", next.TemplateID)
	}
	values := make(map[string]string, len(next.Fields))
	for _, field := range next.Fields {
		values[field.Key] = field.Value
	}
	normalized, err := task.NewExtraInfoWithParameters(template, values, next.Parameters)
	if err != nil {
		return task.ExtraInfo{}, err
	}
	if next.ID != "" {
		normalized.ID = next.ID
	}
	for index, candidate := range data.ExtraInfos {
		if candidate.ID == normalized.ID {
			data.ExtraInfos[index] = normalized
			if err := repository.Save(data); err != nil {
				return task.ExtraInfo{}, err
			}
			return normalized, nil
		}
	}
	data.ExtraInfos = append(data.ExtraInfos, normalized)
	if err := repository.Save(data); err != nil {
		return task.ExtraInfo{}, err
	}
	return normalized, nil
}

func (repository *Repository) DeleteExtraInfo(id string) error {
	repository.mutationMu.Lock()
	defer repository.mutationMu.Unlock()

	data, err := repository.Load()
	if err != nil {
		return err
	}
	for index, candidate := range data.ExtraInfos {
		if candidate.ID == id {
			data.ExtraInfos = append(data.ExtraInfos[:index], data.ExtraInfos[index+1:]...)
			return repository.Save(data)
		}
	}
	return fmt.Errorf("extra info %q does not exist", id)
}

func (repository *Repository) ListQuickInputs() ([]quickinput.QuickInput, error) {
	data, err := repository.Load()
	if err != nil {
		return nil, err
	}
	return append([]quickinput.QuickInput(nil), data.QuickInputs...), nil
}

func (repository *Repository) SaveQuickInput(next quickinput.QuickInput) (quickinput.QuickInput, error) {
	repository.mutationMu.Lock()
	defer repository.mutationMu.Unlock()

	data, err := repository.Load()
	if err != nil {
		return quickinput.QuickInput{}, err
	}
	creating := strings.TrimSpace(next.ID) == ""
	if creating {
		next, err = quickinput.New(next.Name, next.Content)
	} else {
		next, err = quickinput.Normalize(next)
	}
	if err != nil {
		return quickinput.QuickInput{}, err
	}

	for index, current := range data.QuickInputs {
		if current.ID == next.ID {
			data.QuickInputs[index] = next
			if err := repository.Save(data); err != nil {
				return quickinput.QuickInput{}, err
			}
			return next, nil
		}
	}
	if !creating {
		return quickinput.QuickInput{}, fmt.Errorf("快捷输入 %q 不存在", next.ID)
	}
	data.QuickInputs = append(data.QuickInputs, next)
	if err := repository.Save(data); err != nil {
		return quickinput.QuickInput{}, err
	}
	return next, nil
}

func (repository *Repository) DeleteQuickInput(id string) error {
	repository.mutationMu.Lock()
	defer repository.mutationMu.Unlock()

	data, err := repository.Load()
	if err != nil {
		return err
	}
	id = strings.TrimSpace(id)
	for index, current := range data.QuickInputs {
		if current.ID == id {
			data.QuickInputs = append(data.QuickInputs[:index], data.QuickInputs[index+1:]...)
			return repository.Save(data)
		}
	}
	return fmt.Errorf("快捷输入 %q 不存在", id)
}

func (repository *Repository) ReorderQuickInputs(ids []string) ([]quickinput.QuickInput, error) {
	repository.mutationMu.Lock()
	defer repository.mutationMu.Unlock()

	data, err := repository.Load()
	if err != nil {
		return nil, err
	}
	if len(ids) != len(data.QuickInputs) {
		return nil, fmt.Errorf("快捷输入排序必须包含全部项目")
	}
	byID := make(map[string]quickinput.QuickInput, len(data.QuickInputs))
	for _, current := range data.QuickInputs {
		byID[current.ID] = current
	}
	reordered := make([]quickinput.QuickInput, 0, len(ids))
	seenIDs := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if _, exists := seenIDs[id]; exists {
			return nil, fmt.Errorf("快捷输入排序包含重复 ID %q", id)
		}
		current, exists := byID[id]
		if !exists {
			return nil, fmt.Errorf("快捷输入 %q 不存在", id)
		}
		seenIDs[id] = struct{}{}
		reordered = append(reordered, current)
	}
	data.QuickInputs = reordered
	if err := repository.Save(data); err != nil {
		return nil, err
	}
	return append([]quickinput.QuickInput(nil), reordered...), nil
}

// ListExtraInfoCatalogues is retained for callers that still consume catalogue names.
func (repository *Repository) ListExtraInfoCatalogues() ([]string, error) {
	data, err := repository.Load()
	if err != nil {
		return nil, err
	}
	return data.ExtraInfoCatalogues, nil
}

// SaveExtraInfoCatalogue creates a minimal custom template for legacy callers.
func (repository *Repository) SaveExtraInfoCatalogue(catalogue string) (string, error) {
	created, err := task.NewExtraInfoTemplate(catalogue, "", nil, nil)
	if err != nil {
		return "", err
	}
	saved, err := repository.SaveExtraInfoTemplate(created)
	if err != nil {
		return "", err
	}
	return saved.Catalogue, nil
}

func (repository *Repository) DeleteExtraInfoCatalogue(catalogue string) error {
	data, err := repository.Load()
	if err != nil {
		return err
	}
	for _, template := range data.ExtraInfoTemplates {
		if template.Catalogue == catalogue {
			return repository.DeleteExtraInfoTemplate(template.ID)
		}
	}
	return fmt.Errorf("extra info catalogue %q does not exist", catalogue)
}
