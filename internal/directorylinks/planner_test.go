package directorylinks

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"taskai/internal/task"
)

func TestPlanUsesBasenameAndParentComponentsDeterministically(t *testing.T) {
	root := t.TempDir()
	api := makeDirectory(t, root, "api")
	web := makeDirectory(t, root, "web")
	projectASource := makeDirectory(t, root, "project-a", "src")
	projectBSource := makeDirectory(t, root, "project-b", "src")
	projectACommonSource := makeDirectory(t, root, "project-a", "common", "source")
	projectBCommonSource := makeDirectory(t, root, "project-b", "common", "source")

	for _, test := range []struct {
		name  string
		paths []string
		want  []string
	}{
		{name: "basenames", paths: []string{api, web}, want: []string{"api", "web"}},
		{name: "direct parents", paths: []string{projectASource, projectBSource}, want: []string{"project-a-src", "project-b-src"}},
		{name: "multiple parents", paths: []string{projectACommonSource, projectBCommonSource}, want: []string{"project-a-common-source", "project-b-common-source"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			links, err := Plan(directoryTemplate(), map[string]any{"sources": test.paths})
			if err != nil {
				t.Fatalf("Plan() error = %v", err)
			}
			if got := linkNames(links); !reflect.DeepEqual(got, test.want) {
				t.Fatalf("Plan() names = %#v，期望 %#v", got, test.want)
			}
		})
	}
}

func TestPlanRecomputesAllNamesWhenConflictingSourceIsAdded(t *testing.T) {
	root := t.TempDir()
	projectASource := makeDirectory(t, root, "project-a", "src")
	projectBSource := makeDirectory(t, root, "project-b", "src")

	initial, err := Plan(directoryTemplate(), map[string]any{"sources": []string{projectASource}})
	if err != nil {
		t.Fatalf("Plan(initial) error = %v", err)
	}
	if got := linkNames(initial); !reflect.DeepEqual(got, []string{"src"}) {
		t.Fatalf("Plan(initial) names = %#v", got)
	}
	updated, err := Plan(directoryTemplate(), map[string]any{"sources": []string{projectASource, projectBSource}})
	if err != nil {
		t.Fatalf("Plan(updated) error = %v", err)
	}
	if got := linkNames(updated); !reflect.DeepEqual(got, []string{"project-a-src", "project-b-src"}) {
		t.Fatalf("Plan(updated) names = %#v", got)
	}
}

func TestPlanTreatsNamesCaseInsensitively(t *testing.T) {
	root := t.TempDir()
	upper := makeDirectory(t, root, "project-a", "Src")
	lower := makeDirectory(t, root, "project-b", "src")

	links, err := Plan(directoryTemplate(), map[string]any{"sources": []string{upper, lower}})
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	if got := linkNames(links); !reflect.DeepEqual(got, []string{"project-a-Src", "project-b-src"}) {
		t.Fatalf("Plan() names = %#v", got)
	}
}

func TestPlanRejectsDuplicateSourcesAcrossFields(t *testing.T) {
	root := t.TempDir()
	source := makeDirectory(t, root, "src")
	template := task.TaskTemplate{ID: "directories", Name: "目录", Fields: []task.TaskTemplateField{
		{Key: "primary", DisplayName: "主目录", InputType: task.TaskTemplateFieldInputDirectories},
		{Key: "secondary", DisplayName: "附加目录", InputType: task.TaskTemplateFieldInputDirectories},
	}}
	_, err := Plan(template, map[string]any{
		"primary":   []string{source},
		"secondary": []string{filepath.Join(source, ".")},
	})
	if err == nil || !strings.Contains(err.Error(), "主目录") || !strings.Contains(err.Error(), "附加目录") || !strings.Contains(err.Error(), source) {
		t.Fatalf("Plan() error = %v，期望包含两个字段和来源路径", err)
	}
}

func TestPlanRejectsSourceWithoutUsableName(t *testing.T) {
	root := filepath.VolumeName(t.TempDir()) + string(filepath.Separator)
	_, err := Plan(directoryTemplate(), map[string]any{"sources": []string{root}})
	if err == nil {
		t.Fatal("Plan() error = nil，期望拒绝文件系统根目录")
	}
}

func directoryTemplate() task.TaskTemplate {
	return task.TaskTemplate{ID: "directories", Name: "目录", Fields: []task.TaskTemplateField{{
		Key: "sources", DisplayName: "来源目录", InputType: task.TaskTemplateFieldInputDirectories, Multiple: true,
	}}}
}

func makeDirectory(t *testing.T, root string, components ...string) string {
	t.Helper()
	directory := filepath.Join(append([]string{root}, components...)...)
	if err := os.MkdirAll(directory, 0o755); err != nil {
		t.Fatal(err)
	}
	return directory
}

func linkNames(links []Link) []string {
	names := make([]string, 0, len(links))
	for _, link := range links {
		names = append(names, link.Name)
	}
	return names
}
