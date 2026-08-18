package directorylinks

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"taskai/internal/task"
)

type Link struct {
	Name          string `json:"name"`
	SourcePath    string `json:"sourcePath"`
	CanonicalPath string `json:"canonicalPath"`
	FieldKey      string `json:"fieldKey"`
	FieldName     string `json:"fieldName"`
}

type linkCandidate struct {
	link      Link
	baseName  string
	ancestors []string
	depth     int
}

func Plan(template task.TaskTemplate, values map[string]any) ([]Link, error) {
	normalizedTemplate, err := task.NormalizeTaskTemplate(template)
	if err != nil {
		return nil, err
	}
	resolved, err := task.ResolveTaskTemplateFields(normalizedTemplate, values)
	if err != nil {
		return nil, err
	}
	candidates := make([]linkCandidate, 0)
	canonicalSources := make(map[string]Link)
	for _, field := range normalizedTemplate.Fields {
		if field.InputType != task.TaskTemplateFieldInputDirectories {
			continue
		}
		directories := resolved[field.Key].([]string)
		for _, source := range directories {
			link, baseName, ancestors, err := planSource(field, source)
			if err != nil {
				return nil, err
			}
			canonicalKey := canonicalSourceKey(link.CanonicalPath)
			if previous, found := canonicalSources[canonicalKey]; found {
				return nil, fmt.Errorf("任务模板目录重复: 字段 %q 的 %q 与字段 %q 的 %q 指向同一目录", field.DisplayName, link.SourcePath, previous.FieldName, previous.SourcePath)
			}
			canonicalSources[canonicalKey] = link
			candidates = append(candidates, linkCandidate{link: link, baseName: baseName, ancestors: ancestors})
		}
	}
	return assignLinkNames(candidates)
}

func planSource(field task.TaskTemplateField, source string) (Link, string, []string, error) {
	source = strings.TrimSpace(source)
	if source == "" || !filepath.IsAbs(source) {
		return Link{}, "", nil, fmt.Errorf("任务模板字段 %q 的目录必须是绝对路径: %q", field.DisplayName, source)
	}
	source = filepath.Clean(source)
	info, err := os.Stat(source)
	if err != nil {
		return Link{}, "", nil, fmt.Errorf("任务模板字段 %q 的目录不可访问 %q: %w", field.DisplayName, source, err)
	}
	if !info.IsDir() {
		return Link{}, "", nil, fmt.Errorf("任务模板字段 %q 的路径不是目录: %q", field.DisplayName, source)
	}
	handle, err := os.Open(source)
	if err != nil {
		return Link{}, "", nil, fmt.Errorf("任务模板字段 %q 的目录不可访问 %q: %w", field.DisplayName, source, err)
	}
	if err := handle.Close(); err != nil {
		return Link{}, "", nil, fmt.Errorf("任务模板字段 %q 的目录不可访问 %q: %w", field.DisplayName, source, err)
	}
	canonical, err := filepath.EvalSymlinks(source)
	if err != nil {
		return Link{}, "", nil, fmt.Errorf("任务模板字段 %q 的目录不可规范化 %q: %w", field.DisplayName, source, err)
	}
	canonical = filepath.Clean(canonical)
	baseName := filepath.Base(source)
	if err := validateLinkName(baseName); err != nil {
		return Link{}, "", nil, fmt.Errorf("任务模板字段 %q 的目录无法生成链接名称 %q: %w", field.DisplayName, source, err)
	}
	return Link{
		SourcePath:    source,
		CanonicalPath: canonical,
		FieldKey:      field.Key,
		FieldName:     field.DisplayName,
	}, baseName, parentComponents(source), nil
}

func assignLinkNames(candidates []linkCandidate) ([]Link, error) {
	for {
		groups := make(map[string][]int, len(candidates))
		for index := range candidates {
			name := candidateName(candidates[index])
			if err := validateLinkName(name); err != nil {
				return nil, fmt.Errorf("任务模板字段 %q 的目录无法生成安全链接名称 %q: %w", candidates[index].link.FieldName, candidates[index].link.SourcePath, err)
			}
			candidates[index].link.Name = name
			groups[strings.ToLower(name)] = append(groups[strings.ToLower(name)], index)
		}
		conflicts := make(map[int]bool)
		for _, indexes := range groups {
			if len(indexes) < 2 {
				continue
			}
			for _, index := range indexes {
				conflicts[index] = true
			}
		}
		if len(conflicts) == 0 {
			links := make([]Link, 0, len(candidates))
			for _, candidate := range candidates {
				links = append(links, candidate.link)
			}
			sort.Slice(links, func(left, right int) bool {
				leftName := strings.ToLower(links[left].Name)
				rightName := strings.ToLower(links[right].Name)
				if leftName == rightName {
					return links[left].SourcePath < links[right].SourcePath
				}
				return leftName < rightName
			})
			return links, nil
		}
		for index := range conflicts {
			candidates[index].depth++
			if candidates[index].depth > len(candidates[index].ancestors) {
				return nil, fmt.Errorf("任务模板字段 %q 的目录无法生成唯一链接名称: %q", candidates[index].link.FieldName, candidates[index].link.SourcePath)
			}
		}
	}
}

func candidateName(candidate linkCandidate) string {
	if candidate.depth == 0 {
		return candidate.baseName
	}
	parts := make([]string, 0, candidate.depth+1)
	for index := candidate.depth - 1; index >= 0; index-- {
		parts = append(parts, candidate.ancestors[index])
	}
	parts = append(parts, candidate.baseName)
	return strings.Join(parts, "-")
}

func parentComponents(path string) []string {
	components := make([]string, 0)
	for parent := filepath.Dir(path); ; parent = filepath.Dir(parent) {
		if parent == filepath.Dir(parent) {
			break
		}
		component := filepath.Base(parent)
		if component == "" || component == "." || component == string(filepath.Separator) {
			break
		}
		components = append(components, component)
	}
	return components
}

func validateLinkName(name string) error {
	if name == "" || name == "." || name == ".." || filepath.Base(name) != name || strings.ContainsAny(name, `/\`) || strings.ContainsRune(name, 0) {
		return fmt.Errorf("名称无效")
	}
	if len([]byte(name)) > 255 {
		return fmt.Errorf("名称超过文件系统限制")
	}
	if runtime.GOOS != "windows" {
		return nil
	}
	if strings.ContainsAny(name, `<>:"/\|?*`) || strings.HasSuffix(name, " ") || strings.HasSuffix(name, ".") {
		return fmt.Errorf("名称包含 Windows 不支持的字符")
	}
	stem := strings.ToUpper(strings.TrimSuffix(name, filepath.Ext(name)))
	switch stem {
	case "CON", "PRN", "AUX", "NUL", "COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9", "LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9":
		return fmt.Errorf("名称是 Windows 保留名称")
	}
	return nil
}

func canonicalSourceKey(path string) string {
	if runtime.GOOS == "windows" {
		return strings.ToLower(path)
	}
	return path
}
