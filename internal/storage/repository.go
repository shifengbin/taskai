package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"taskai/internal/settings"
	"taskai/internal/task"
)

// Data is the complete persisted application state.
type Data struct {
	Tasks               []task.Task              `json:"tasks"`
	ExtraInfoTemplates  []task.ExtraInfoTemplate `json:"extraInfoTemplates"`
	ExtraInfos          []task.ExtraInfo         `json:"extraInfos"`
	ExtraInfoCatalogues []string                 `json:"extraInfoCatalogues,omitempty"`
	Settings            settings.Settings        `json:"settings"`
}

// Repository persists all application state in one JSON document.
type Repository struct {
	path            string
	defaultSettings settings.Settings
}

func New(path string, defaultSettings settings.Settings) *Repository {
	return &Repository{path: path, defaultSettings: defaultSettings}
}

func (repository *Repository) Load() (Data, error) {
	contents, err := os.ReadFile(repository.path)
	if os.IsNotExist(err) {
		data := defaultData(repository.defaultSettings)
		if err := repository.Save(data); err != nil {
			return Data{}, err
		}
		return data, nil
	}
	if err != nil {
		return Data{}, fmt.Errorf("read data file: %w", err)
	}

	var data Data
	if err := json.Unmarshal(contents, &data); err != nil {
		return Data{}, fmt.Errorf("decode data file: %w", err)
	}

	normalized, changed, err := normalizeData(data)
	if err != nil {
		return Data{}, err
	}
	if changed {
		if err := repository.Save(normalized); err != nil {
			return Data{}, err
		}
	}
	return normalized, nil
}

func defaultData(defaultSettings settings.Settings) Data {
	return Data{
		Tasks:               []task.Task{},
		ExtraInfoTemplates:  []task.ExtraInfoTemplate{task.BuiltInGitTemplate()},
		ExtraInfos:          []task.ExtraInfo{},
		ExtraInfoCatalogues: []string{"git"},
		Settings:            defaultSettings,
	}
}

func normalizeData(data Data) (Data, bool, error) {
	changed := false
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
		data.Tasks[index] = normalized
	}

	catalogues := collectTemplateCatalogues(data.ExtraInfoTemplates)
	if !sameJSON(data.ExtraInfoCatalogues, catalogues) {
		data.ExtraInfoCatalogues = catalogues
		changed = true
	}
	return data, changed, nil
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

func normalizeDataForSave(data Data) (Data, bool, error) {
	if data.ExtraInfos == nil {
		data.ExtraInfos = []task.ExtraInfo{}
	}
	if data.ExtraInfoTemplates == nil {
		data.ExtraInfoTemplates = []task.ExtraInfoTemplate{}
	}
	return normalizeData(data)
}

func (repository *Repository) SaveSettings(next settings.Settings) (settings.Settings, error) {
	data, err := repository.Load()
	if err != nil {
		return settings.Settings{}, err
	}
	data.Settings = next
	if err := repository.Save(data); err != nil {
		return settings.Settings{}, err
	}
	return next, nil
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
