package directorylinks

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"taskai/internal/task"
	"taskai/internal/workspace"
)

const (
	manifestSchemaVersion = 1
	manifestStateStable   = "stable"
	manifestStatePending  = "pending"
	manifestSuffix        = ".directory-links.json"
	manifestMaxSize       = 1024 * 1024
)

type manifestLink struct {
	Name       string `json:"name"`
	SourcePath string `json:"sourcePath"`
}

type linkManifest struct {
	SchemaVersion int            `json:"schemaVersion"`
	TaskID        string         `json:"taskId"`
	Token         string         `json:"token"`
	State         string         `json:"state"`
	Links         []manifestLink `json:"links,omitempty"`
	Previous      []manifestLink `json:"previous,omitempty"`
	Desired       []manifestLink `json:"desired,omitempty"`
}

type Synchronizer struct {
	links DirectoryLinkFS
}

func NewSynchronizer(links DirectoryLinkFS) *Synchronizer {
	return &Synchronizer{links: links}
}

func (synchronizer *Synchronizer) Sync(root, workspacePath, taskID, token string, desired []Link) error {
	if synchronizer == nil || synchronizer.links == nil {
		return fmt.Errorf("目录链接文件系统不可用")
	}
	normalizedDesired, err := normalizeDesiredLinks(desired)
	if err != nil {
		return err
	}
	return workspace.WithOwnedWorkspace(root, workspacePath, taskID, token, func(current workspace.OwnedWorkspaceContext) error {
		path := filepath.Join(current.MetadataPath, token+manifestSuffix)
		manifest, found, err := readManifest(path)
		if err != nil {
			return err
		}
		if found {
			if err := validateManifest(manifest, taskID, token); err != nil {
				return err
			}
		} else {
			manifest = linkManifest{
				SchemaVersion: manifestSchemaVersion,
				TaskID:        taskID,
				Token:         token,
				State:         manifestStateStable,
				Links:         []manifestLink{},
			}
		}

		registered := registeredManifestLinks(manifest)
		if err := synchronizer.preflight(current.Path, registered, normalizedDesired); err != nil {
			return err
		}
		pending := linkManifest{
			SchemaVersion: manifestSchemaVersion,
			TaskID:        taskID,
			Token:         token,
			State:         manifestStatePending,
			Previous:      flattenRegisteredLinks(registered),
			Desired:       append([]manifestLink(nil), normalizedDesired...),
		}
		if err := writeManifest(path, pending); err != nil {
			return err
		}
		if err := synchronizer.converge(current.Path, registered, normalizedDesired); err != nil {
			return err
		}
		stable := linkManifest{
			SchemaVersion: manifestSchemaVersion,
			TaskID:        taskID,
			Token:         token,
			State:         manifestStateStable,
			Links:         append([]manifestLink(nil), normalizedDesired...),
		}
		return writeManifest(path, stable)
	})
}

func normalizeDesiredLinks(desired []Link) ([]manifestLink, error) {
	normalized := make([]manifestLink, 0, len(desired))
	names := make(map[string]bool, len(desired))
	sources := make(map[string]manifestLink, len(desired))
	for _, link := range desired {
		if err := validateLinkName(link.Name); err != nil {
			return nil, fmt.Errorf("目录链接名称无效 %q: %w", link.Name, err)
		}
		nameKey := strings.ToLower(link.Name)
		if names[nameKey] {
			return nil, fmt.Errorf("目录链接名称重复: %q", link.Name)
		}
		names[nameKey] = true

		fieldName := strings.TrimSpace(link.FieldName)
		if fieldName == "" {
			fieldName = strings.TrimSpace(link.FieldKey)
		}
		if fieldName == "" {
			fieldName = "目录"
		}
		planned, _, _, err := planSource(task.TaskTemplateField{Key: link.FieldKey, DisplayName: fieldName, InputType: task.TaskTemplateFieldInputDirectories}, link.SourcePath)
		if err != nil {
			return nil, err
		}
		canonicalKey := canonicalSourceKey(planned.CanonicalPath)
		if previous, found := sources[canonicalKey]; found {
			return nil, fmt.Errorf("任务模板目录重复: %q 与 %q 指向同一目录", planned.SourcePath, previous.SourcePath)
		}
		record := manifestLink{Name: link.Name, SourcePath: planned.SourcePath}
		sources[canonicalKey] = record
		normalized = append(normalized, record)
	}
	sort.Slice(normalized, func(left, right int) bool {
		return strings.ToLower(normalized[left].Name) < strings.ToLower(normalized[right].Name)
	})
	return normalized, nil
}

func (synchronizer *Synchronizer) preflight(workspacePath string, registered map[string][]manifestLink, desired []manifestLink) error {
	actual := make(map[string]string, len(registered))
	for nameKey, records := range registered {
		path := filepath.Join(workspacePath, records[0].Name)
		target, exists, err := synchronizer.links.Read(path)
		if err != nil {
			return fmt.Errorf("管理目录链接身份不匹配 %q: %w", path, err)
		}
		if !exists {
			continue
		}
		if !targetMatchesAny(target, records) {
			return fmt.Errorf("管理目录链接身份不匹配 %q: 当前目标 %q 未登记", path, target)
		}
		actual[nameKey] = target
	}
	for _, record := range desired {
		nameKey := strings.ToLower(record.Name)
		if _, found := actual[nameKey]; found {
			continue
		}
		path := filepath.Join(workspacePath, record.Name)
		target, exists, err := synchronizer.links.Read(path)
		if err != nil {
			return fmt.Errorf("期望目录链接名称已被未知条目占用 %q: %w", path, err)
		}
		if !exists {
			continue
		}
		if records, managed := registered[nameKey]; !managed || !targetMatchesAny(target, records) {
			return fmt.Errorf("期望目录链接名称已被未登记链接占用 %q -> %q", path, target)
		}
	}
	return nil
}

func (synchronizer *Synchronizer) converge(workspacePath string, registered map[string][]manifestLink, desired []manifestLink) error {
	desiredByName := make(map[string]manifestLink, len(desired))
	for _, record := range desired {
		desiredByName[strings.ToLower(record.Name)] = record
	}
	for nameKey, records := range registered {
		path := filepath.Join(workspacePath, records[0].Name)
		target, exists, err := synchronizer.links.Read(path)
		if err != nil {
			return fmt.Errorf("删除管理目录链接前验证失败 %q: %w", path, err)
		}
		if !exists {
			continue
		}
		if !targetMatchesAny(target, records) {
			return fmt.Errorf("删除管理目录链接前身份不匹配 %q -> %q", path, target)
		}
		if wanted, keep := desiredByName[nameKey]; keep && sameLinkTarget(target, wanted.SourcePath) {
			continue
		}
		if err := synchronizer.links.Remove(path); err != nil {
			return err
		}
	}
	for _, record := range desired {
		path := filepath.Join(workspacePath, record.Name)
		target, exists, err := synchronizer.links.Read(path)
		if err != nil {
			return fmt.Errorf("创建目录链接前检查失败 %q: %w", path, err)
		}
		if exists {
			if !sameLinkTarget(target, record.SourcePath) {
				return fmt.Errorf("目录链接目标冲突 %q: 当前 %q，期望 %q", path, target, record.SourcePath)
			}
			continue
		}
		if err := synchronizer.links.Create(path, record.SourcePath); err != nil {
			return err
		}
	}
	return nil
}

func registeredManifestLinks(manifest linkManifest) map[string][]manifestLink {
	registered := make(map[string][]manifestLink)
	add := func(records []manifestLink) {
		for _, record := range records {
			key := strings.ToLower(record.Name)
			duplicate := false
			for _, existing := range registered[key] {
				if sameLinkTarget(existing.SourcePath, record.SourcePath) {
					duplicate = true
					break
				}
			}
			if !duplicate {
				registered[key] = append(registered[key], record)
			}
		}
	}
	if manifest.State == manifestStateStable {
		add(manifest.Links)
	} else {
		add(manifest.Previous)
		add(manifest.Desired)
	}
	return registered
}

func flattenRegisteredLinks(registered map[string][]manifestLink) []manifestLink {
	flattened := make([]manifestLink, 0)
	for _, records := range registered {
		flattened = append(flattened, records...)
	}
	sort.Slice(flattened, func(left, right int) bool {
		leftName := strings.ToLower(flattened[left].Name)
		rightName := strings.ToLower(flattened[right].Name)
		if leftName == rightName {
			return flattened[left].SourcePath < flattened[right].SourcePath
		}
		return leftName < rightName
	})
	return flattened
}

func targetMatchesAny(target string, records []manifestLink) bool {
	for _, record := range records {
		if sameLinkTarget(target, record.SourcePath) {
			return true
		}
	}
	return false
}

func sameLinkTarget(left, right string) bool {
	left = filepath.Clean(left)
	right = filepath.Clean(right)
	if runtime.GOOS == "windows" {
		return strings.EqualFold(left, right)
	}
	return left == right
}

func manifestPath(root, token string) string {
	return filepath.Join(filepath.Clean(root), ".taskai-ownership", token+manifestSuffix)
}

func readManifest(path string) (linkManifest, bool, error) {
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return linkManifest{}, false, nil
	}
	if err != nil {
		return linkManifest{}, false, fmt.Errorf("检查目录链接清单失败: %w", err)
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Size() > manifestMaxSize {
		return linkManifest{}, false, fmt.Errorf("目录链接清单不安全或过大")
	}
	file, err := os.Open(path)
	if err != nil {
		return linkManifest{}, false, fmt.Errorf("打开目录链接清单失败: %w", err)
	}
	defer file.Close()
	decoder := json.NewDecoder(io.LimitReader(file, manifestMaxSize+1))
	decoder.DisallowUnknownFields()
	var manifest linkManifest
	if err := decoder.Decode(&manifest); err != nil {
		return linkManifest{}, false, fmt.Errorf("目录链接清单损坏: %w", err)
	}
	if decoder.Decode(&struct{}{}) != io.EOF {
		return linkManifest{}, false, fmt.Errorf("目录链接清单包含多余内容")
	}
	return manifest, true, nil
}

func validateManifest(manifest linkManifest, taskID, token string) error {
	if manifest.SchemaVersion != manifestSchemaVersion || manifest.TaskID != taskID || manifest.Token != token {
		return fmt.Errorf("目录链接清单身份无效")
	}
	if manifest.State != manifestStateStable && manifest.State != manifestStatePending {
		return fmt.Errorf("目录链接清单状态无效: %q", manifest.State)
	}
	if manifest.State == manifestStateStable && (len(manifest.Previous) > 0 || len(manifest.Desired) > 0) {
		return fmt.Errorf("稳定目录链接清单包含事务数据")
	}
	if manifest.State == manifestStatePending && len(manifest.Links) > 0 {
		return fmt.Errorf("待提交目录链接清单包含稳定数据")
	}
	for _, record := range append(append(append([]manifestLink(nil), manifest.Links...), manifest.Previous...), manifest.Desired...) {
		if err := validateLinkName(record.Name); err != nil || !filepath.IsAbs(record.SourcePath) {
			return fmt.Errorf("目录链接清单条目无效: %q -> %q", record.Name, record.SourcePath)
		}
	}
	return nil
}

func writeManifest(path string, manifest linkManifest) error {
	contents, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("编码目录链接清单失败: %w", err)
	}
	contents = append(contents, '\n')
	temporary, err := os.CreateTemp(filepath.Dir(path), ".directory-links-*.tmp")
	if err != nil {
		return fmt.Errorf("创建目录链接临时清单失败: %w", err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(0o600); err != nil {
		temporary.Close()
		return fmt.Errorf("设置目录链接清单权限失败: %w", err)
	}
	if _, err := temporary.Write(contents); err != nil {
		temporary.Close()
		return fmt.Errorf("写入目录链接清单失败: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return fmt.Errorf("同步目录链接清单失败: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("关闭目录链接清单失败: %w", err)
	}
	if err := replaceManifestFile(temporaryPath, path); err != nil {
		return fmt.Errorf("提交目录链接清单失败: %w", err)
	}
	return nil
}
