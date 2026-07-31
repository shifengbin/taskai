package lifecycle

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"taskai/internal/realtime"
)

func TestBuildCommandInputUsesTaskDetailAndOptionalBaseURL(t *testing.T) {
	resource := realtime.TaskResource{
		ID: "task-1", Title: "发布", Description: "发布 API", Color: "#4f46e5", Status: "running",
		CreatedAt: time.Date(2026, time.July, 29, 8, 0, 0, 0, time.UTC), WorkspaceRoot: "/workspaces", WorkspacePath: "/workspaces/task-1",
		TemplateFields: map[string]any{"environment": "production", "dryRun": false},
	}
	input, err := BuildCommandInput(resource, "http://127.0.0.1:18765/api/v1")
	if err != nil {
		t.Fatalf("BuildCommandInput() error = %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(input, &decoded); err != nil {
		t.Fatalf("Unmarshal input error = %v", err)
	}
	if decoded["id"] != "task-1" || decoded["workspacePath"] != "/workspaces/task-1" || decoded["baseURL"] != "http://127.0.0.1:18765/api/v1" {
		t.Fatalf("命令输入 = %#v", decoded)
	}
	if got, want := decoded["templateFields"], map[string]any{"environment": "production", "dryRun": false}; !reflect.DeepEqual(got, want) {
		t.Fatalf("命令输入模板字段 = %#v，期望 %#v", got, want)
	}

	withoutHTTP, err := BuildCommandInput(resource, "")
	if err != nil {
		t.Fatalf("BuildCommandInput() without HTTP error = %v", err)
	}
	decoded = map[string]any{}
	if err := json.Unmarshal(withoutHTTP, &decoded); err != nil {
		t.Fatalf("Unmarshal input without HTTP error = %v", err)
	}
	if _, found := decoded["baseURL"]; found {
		t.Fatalf("未运行 HTTP 服务时不应传入 baseURL: %#v", decoded)
	}
}
