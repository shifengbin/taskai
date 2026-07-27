package realtime

import (
	"bytes"
	"encoding/json"
	"net"
	"net/http"
	"testing"
	"time"
)

func TestHTTPServerQueriesAndUpdatesRealtimeStatuses(t *testing.T) {
	service := New(Options{})
	service.RegisterTask("task-1")
	service.RegisterTerminal("task-1", "terminal-1")
	server := startHTTPServer(t, service, func(taskID string) TaskState {
		if taskID == "task-1" {
			return TaskStateRunning
		}
		return TaskStateMissing
	})

	response, err := http.Get(server.APIURL() + "/status")
	if err != nil {
		t.Fatalf("查询状态: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("查询状态码 = %d，期望 %d", response.StatusCode, http.StatusOK)
	}
	var snapshot StatusSnapshot
	if err := json.NewDecoder(response.Body).Decode(&snapshot); err != nil {
		t.Fatalf("解析状态查询: %v", err)
	}
	if len(snapshot.Tasks) != 1 || snapshot.Tasks[0].TaskID != "task-1" || snapshot.Tasks[0].Status != StatusIdle {
		t.Fatalf("查询状态 = %#v", snapshot)
	}

	updated := putStatus(t, server, "/tasks/task-1/status", StatusError)
	if updated.TaskStatus != StatusError {
		t.Fatalf("直接更新任务状态 = %q，期望 %q", updated.TaskStatus, StatusError)
	}
	updated = putStatus(t, server, "/tasks/task-1/terminals/terminal-1/status", StatusWorking)
	if updated.TaskStatus != StatusWorking || updated.TerminalStatus != StatusWorking {
		t.Fatalf("更新终端状态 = %#v", updated)
	}
}

func TestHTTPServerReturnsJSONErrorsForInvalidRequests(t *testing.T) {
	service := New(Options{})
	service.RegisterTask("task-running")
	service.RegisterTerminal("task-running", "terminal-1")
	server := startHTTPServer(t, service, func(taskID string) TaskState {
		switch taskID {
		case "task-running":
			return TaskStateRunning
		case "task-ended":
			return TaskStateEnded
		default:
			return TaskStateMissing
		}
	})

	assertHTTPStatus(t, http.MethodPut, server.APIURL()+"/tasks/task-running/status", []byte(`{"status":"unknown"}`), http.StatusBadRequest)
	assertHTTPStatus(t, http.MethodPut, server.APIURL()+"/tasks/missing/status", []byte(`{"status":"idle"}`), http.StatusNotFound)
	assertHTTPStatus(t, http.MethodPut, server.APIURL()+"/tasks/task-ended/status", []byte(`{"status":"idle"}`), http.StatusConflict)
	assertHTTPStatus(t, http.MethodPut, server.APIURL()+"/tasks/task-running/terminals/missing/status", []byte(`{"status":"idle"}`), http.StatusNotFound)
	assertHTTPStatus(t, http.MethodPost, server.APIURL()+"/status", nil, http.StatusMethodNotAllowed)
}

func TestHTTPServerKeepsExistingListenerWhenReconfigureFails(t *testing.T) {
	service := New(Options{})
	server := startHTTPServer(t, service, func(string) TaskState { return TaskStateRunning })
	previousURL := server.APIURL()
	occupied, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("占用测试端口: %v", err)
	}
	defer occupied.Close()

	port := occupied.Addr().(*net.TCPAddr).Port
	if err := server.Configure(port); err == nil {
		t.Fatal("重配被占用端口 error = nil，期望错误")
	}
	if server.APIURL() != previousURL {
		t.Errorf("重配失败后 API 地址 = %q，期望保留 %q", server.APIURL(), previousURL)
	}
	response, err := http.Get(previousURL + "/status")
	if err != nil {
		t.Fatalf("重配失败后查询旧服务: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Errorf("重配失败后旧服务状态码 = %d，期望 %d", response.StatusCode, http.StatusOK)
	}
}

func TestHTTPServerKeepsExistingListenerWhenConfiguringSamePort(t *testing.T) {
	server := startHTTPServer(t, New(Options{}), nil)
	previousURL := server.APIURL()
	port := server.listener.Addr().(*net.TCPAddr).Port

	if err := server.Configure(port); err != nil {
		t.Fatalf("同端口重配 HTTP 服务: %v", err)
	}
	if server.APIURL() != previousURL {
		t.Errorf("同端口重配后的 API 地址 = %q，期望保持 %q", server.APIURL(), previousURL)
	}
}

func TestHTTPServerListsTasksByLifecycleStatusAndReturnsDetails(t *testing.T) {
	completedAt := time.Date(2026, time.July, 27, 12, 0, 0, 0, time.UTC)
	tasks := []TaskResource{
		{ID: "task-pending", Title: "待执行", Description: "等待开始", Color: "#4f46e5", Status: "pending", CreatedAt: completedAt.Add(-time.Hour)},
		{ID: "task-running", Title: "执行中", Description: "正在工作", Color: "#22c55e", Status: "running", CreatedAt: completedAt.Add(-30 * time.Minute), WorkspaceRoot: "/tmp/workspaces", WorkspacePath: "/tmp/workspaces/task-running"},
		{ID: "task-completed", Title: "已完成", Description: "已经结束", Color: "#f97316", Status: "completed", CreatedAt: completedAt.Add(-2 * time.Hour), CompletedAt: &completedAt},
	}
	server := startHTTPServerWithCatalog(t, TaskCatalog{
		List: func() ([]TaskResource, error) { return tasks, nil },
		Get: func(taskID string) (TaskResource, bool, error) {
			for _, item := range tasks {
				if item.ID == taskID {
					return item, true, nil
				}
			}
			return TaskResource{}, false, nil
		},
	})

	for _, expected := range []struct {
		status string
		id     string
	}{
		{status: "pending", id: "task-pending"},
		{status: "running", id: "task-running"},
		{status: "completed", id: "task-completed"},
	} {
		response, err := http.Get(server.APIURL() + "/tasks?status=" + expected.status)
		if err != nil {
			t.Fatalf("查询 %s 任务列表: %v", expected.status, err)
		}
		var listed []TaskResource
		if err := json.NewDecoder(response.Body).Decode(&listed); err != nil {
			response.Body.Close()
			t.Fatalf("解析 %s 任务列表: %v", expected.status, err)
		}
		response.Body.Close()
		if len(listed) != 1 || listed[0].ID != expected.id || listed[0].Status != expected.status {
			t.Fatalf("%s 任务列表 = %#v", expected.status, listed)
		}
	}

	response, err := http.Get(server.APIURL() + "/tasks/task-running")
	if err != nil {
		t.Fatalf("查询任务详情: %v", err)
	}
	defer response.Body.Close()
	var detail TaskResource
	if err := json.NewDecoder(response.Body).Decode(&detail); err != nil {
		t.Fatalf("解析任务详情: %v", err)
	}
	if detail.ID != "task-running" || detail.Description != "正在工作" || detail.WorkspacePath != "/tmp/workspaces/task-running" {
		t.Fatalf("任务详情 = %#v", detail)
	}

	assertHTTPStatus(t, http.MethodGet, server.APIURL()+"/tasks?status=archived", nil, http.StatusBadRequest)
	assertHTTPStatus(t, http.MethodGet, server.APIURL()+"/tasks/missing", nil, http.StatusNotFound)
	assertHTTPStatus(t, http.MethodPost, server.APIURL()+"/tasks", nil, http.StatusMethodNotAllowed)
}

func startHTTPServer(t *testing.T, service *Service, resolveTask func(string) TaskState) *HTTPServer {
	t.Helper()
	server := NewHTTPServer(HTTPServerOptions{Service: service, ResolveTask: resolveTask})
	if err := server.Configure(0); err != nil {
		t.Fatalf("启动 HTTP 状态服务: %v", err)
	}
	t.Cleanup(func() {
		if err := server.Close(); err != nil {
			t.Errorf("关闭 HTTP 状态服务: %v", err)
		}
	})
	return server
}

func startHTTPServerWithCatalog(t *testing.T, catalog TaskCatalog) *HTTPServer {
	t.Helper()
	server := NewHTTPServer(HTTPServerOptions{TaskCatalog: catalog})
	if err := server.Configure(0); err != nil {
		t.Fatalf("启动任务 HTTP 服务: %v", err)
	}
	t.Cleanup(func() {
		if err := server.Close(); err != nil {
			t.Errorf("关闭任务 HTTP 服务: %v", err)
		}
	})
	return server
}

func putStatus(t *testing.T, server *HTTPServer, path string, status Status) StatusUpdateResponse {
	t.Helper()
	contents, err := json.Marshal(map[string]Status{"status": status})
	if err != nil {
		t.Fatalf("编码状态请求: %v", err)
	}
	request, err := http.NewRequest(http.MethodPut, server.APIURL()+path, bytes.NewReader(contents))
	if err != nil {
		t.Fatalf("创建更新请求: %v", err)
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("更新状态: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("更新状态码 = %d，期望 %d", response.StatusCode, http.StatusOK)
	}
	var updated StatusUpdateResponse
	if err := json.NewDecoder(response.Body).Decode(&updated); err != nil {
		t.Fatalf("解析更新响应: %v", err)
	}
	return updated
}

func assertHTTPStatus(t *testing.T, method, url string, body []byte, want int) {
	t.Helper()
	request, err := http.NewRequest(method, url, bytes.NewReader(body))
	if err != nil {
		t.Fatalf("创建请求: %v", err)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("发送请求: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != want {
		t.Errorf("%s %s 状态码 = %d，期望 %d", method, url, response.StatusCode, want)
	}
	var payload map[string]string
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Errorf("解析错误响应: %v", err)
	}
	if payload["error"] == "" {
		t.Errorf("错误响应缺少 error 字段: %#v", payload)
	}
}
