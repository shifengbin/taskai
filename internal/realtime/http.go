package realtime

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"taskai/internal/task"
)

type TaskState string

const (
	TaskStateMissing TaskState = "missing"
	TaskStateRunning TaskState = "running"
	TaskStateEnded   TaskState = "ended"
)

type HTTPServerOptions struct {
	Service     *Service
	ResolveTask func(string) TaskState
	TaskCatalog TaskCatalog
}

type TaskCatalog struct {
	List func() ([]TaskResource, error)
	Get  func(string) (TaskResource, bool, error)
}

type TaskResource struct {
	ID                 string                        `json:"id"`
	Title              string                        `json:"title"`
	Description        string                        `json:"description"`
	Color              string                        `json:"color"`
	Status             string                        `json:"status"`
	CreatedAt          time.Time                     `json:"createdAt"`
	CompletedAt        *time.Time                    `json:"completedAt,omitempty"`
	WorkspaceRoot      string                        `json:"workspaceRoot,omitempty"`
	WorkspacePath      string                        `json:"workspacePath,omitempty"`
	LifecycleChains    map[task.LifecycleHook]string `json:"lifecycleChains"`
	LifecycleExecution *task.LifecycleExecution      `json:"lifecycleExecution,omitempty"`
	TemplateFields     map[string]any                `json:"templateFields"`
	ExtraInfo          *map[string][]map[string]any  `json:"extraInfo,omitempty"`
	Terminals          *[]TerminalResource           `json:"terminals,omitempty"`
}

type TerminalResource struct {
	ID      string `json:"id"`
	Command string `json:"command"`
	Status  Status `json:"status"`
}

type HTTPServer struct {
	service     *Service
	resolveTask func(string) TaskState
	taskCatalog TaskCatalog

	mu       sync.Mutex
	listener net.Listener
	server   *http.Server
}

type StatusUpdateResponse struct {
	TaskID         string `json:"taskId"`
	TaskStatus     Status `json:"taskStatus"`
	TerminalID     string `json:"terminalId,omitempty"`
	TerminalStatus Status `json:"terminalStatus,omitempty"`
}

func NewHTTPServer(options HTTPServerOptions) *HTTPServer {
	service := options.Service
	if service == nil {
		service = New(Options{})
	}
	resolveTask := options.ResolveTask
	if resolveTask == nil {
		resolveTask = func(string) TaskState { return TaskStateRunning }
	}
	return &HTTPServer{service: service, resolveTask: resolveTask, taskCatalog: options.TaskCatalog}
}

func (server *HTTPServer) Configure(port int) error {
	if port < 0 || port > 65535 {
		return fmt.Errorf("HTTP 状态服务端口必须在 0 到 65535 之间")
	}
	server.mu.Lock()
	if port != 0 && server.listener != nil {
		if address, ok := server.listener.Addr().(*net.TCPAddr); ok && address.Port == port {
			server.mu.Unlock()
			return nil
		}
	}
	listener, err := net.Listen("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)))
	if err != nil {
		server.mu.Unlock()
		return fmt.Errorf("监听 HTTP 状态服务端口: %w", err)
	}
	next := &http.Server{Handler: http.HandlerFunc(server.ServeHTTP)}
	go func() {
		_ = next.Serve(listener)
	}()

	previous := server.server
	server.listener = listener
	server.server = next
	server.mu.Unlock()
	if previous != nil {
		_ = previous.Close()
	}
	return nil
}

func (server *HTTPServer) Close() error {
	server.mu.Lock()
	current := server.server
	server.server = nil
	server.listener = nil
	server.mu.Unlock()
	if current == nil {
		return nil
	}
	return current.Close()
}

func (server *HTTPServer) APIURL() string {
	server.mu.Lock()
	defer server.mu.Unlock()
	if server.listener == nil {
		return ""
	}
	return "http://" + server.listener.Addr().String() + "/api/v1"
}

func (server *HTTPServer) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	parts := strings.Split(strings.Trim(request.URL.Path, "/"), "/")
	if len(parts) == 3 && parts[0] == "api" && parts[1] == "v1" && parts[2] == "tasks" {
		if request.Method != http.MethodGet {
			server.writeMethodNotAllowed(writer)
			return
		}
		server.listTasks(writer, request)
		return
	}
	if len(parts) == 4 && parts[0] == "api" && parts[1] == "v1" && parts[2] == "tasks" {
		if request.Method != http.MethodGet {
			server.writeMethodNotAllowed(writer)
			return
		}
		server.getTask(writer, parts[3])
		return
	}
	if len(parts) == 3 && parts[0] == "api" && parts[1] == "v1" && parts[2] == "status" {
		if request.Method != http.MethodGet {
			server.writeMethodNotAllowed(writer)
			return
		}
		server.listStatus(writer, request)
		return
	}
	if len(parts) == 5 && parts[0] == "api" && parts[1] == "v1" && parts[2] == "tasks" && parts[4] == "status" {
		if request.Method != http.MethodPut {
			server.writeMethodNotAllowed(writer)
			return
		}
		server.updateTaskStatus(writer, request, parts[3])
		return
	}
	if len(parts) == 7 && parts[0] == "api" && parts[1] == "v1" && parts[2] == "tasks" && parts[4] == "terminals" && parts[6] == "status" {
		if request.Method != http.MethodPut {
			server.writeMethodNotAllowed(writer)
			return
		}
		server.updateTerminalStatus(writer, request, parts[3], parts[5])
		return
	}
	server.writeError(writer, http.StatusNotFound, "接口不存在")
}

func (server *HTTPServer) listStatus(writer http.ResponseWriter, request *http.Request) {
	status := request.URL.Query().Get("status")
	if status != "" && !validTaskLifecycleStatus(status) {
		server.writeError(writer, http.StatusBadRequest, "任务状态筛选无效")
		return
	}

	snapshot := server.service.Snapshot()
	if server.taskCatalog.List == nil {
		if status != "" {
			server.writeError(writer, http.StatusInternalServerError, "任务查询不可用")
			return
		}
		server.writeJSON(writer, http.StatusOK, snapshot)
		return
	}

	tasks, err := server.taskCatalog.List()
	if err != nil {
		server.writeError(writer, http.StatusInternalServerError, "读取任务失败")
		return
	}
	realtimeByTaskID := make(map[string]TaskSnapshot, len(snapshot.Tasks))
	for _, item := range snapshot.Tasks {
		realtimeByTaskID[item.TaskID] = item
	}
	combined := StatusSnapshot{Tasks: make([]TaskSnapshot, 0, len(tasks))}
	for _, item := range tasks {
		if status != "" && item.Status != status {
			continue
		}
		current, found := realtimeByTaskID[item.ID]
		if !found {
			current = TaskSnapshot{TaskID: item.ID, Status: StatusIdle, Terminals: []TerminalSnapshot{}}
		}
		current.Title = item.Title
		current.LifecycleStatus = item.Status
		combined.Tasks = append(combined.Tasks, current)
	}
	server.writeJSON(writer, http.StatusOK, combined)
}

func (server *HTTPServer) listTasks(writer http.ResponseWriter, request *http.Request) {
	if server.taskCatalog.List == nil {
		server.writeError(writer, http.StatusInternalServerError, "任务查询不可用")
		return
	}
	status := request.URL.Query().Get("status")
	if status != "" && !validTaskLifecycleStatus(status) {
		server.writeError(writer, http.StatusBadRequest, "任务状态筛选无效")
		return
	}
	tasks, err := server.taskCatalog.List()
	if err != nil {
		server.writeError(writer, http.StatusInternalServerError, "读取任务失败")
		return
	}
	if status == "" {
		server.writeJSON(writer, http.StatusOK, tasks)
		return
	}
	filtered := make([]TaskResource, 0, len(tasks))
	for _, item := range tasks {
		if item.Status == status {
			filtered = append(filtered, item)
		}
	}
	server.writeJSON(writer, http.StatusOK, filtered)
}

func validTaskLifecycleStatus(status string) bool {
	return status == "pending" || status == "running" || status == "completed"
}

func (server *HTTPServer) getTask(writer http.ResponseWriter, taskID string) {
	if server.taskCatalog.Get == nil {
		server.writeError(writer, http.StatusInternalServerError, "任务查询不可用")
		return
	}
	item, found, err := server.taskCatalog.Get(taskID)
	if err != nil {
		server.writeError(writer, http.StatusInternalServerError, "读取任务失败")
		return
	}
	if !found {
		server.writeError(writer, http.StatusNotFound, "任务不存在")
		return
	}
	server.writeJSON(writer, http.StatusOK, item)
}

func (server *HTTPServer) updateTaskStatus(writer http.ResponseWriter, request *http.Request, taskID string) {
	if !server.requireRunningTask(writer, taskID) {
		return
	}
	status, ok := server.decodeStatus(writer, request)
	if !ok {
		return
	}
	server.service.RegisterTask(taskID)
	server.service.SetTaskStatus(taskID, status)
	server.writeJSON(writer, http.StatusOK, StatusUpdateResponse{TaskID: taskID, TaskStatus: server.service.TaskStatus(taskID)})
}

func (server *HTTPServer) updateTerminalStatus(writer http.ResponseWriter, request *http.Request, taskID, terminalID string) {
	if !server.requireRunningTask(writer, taskID) {
		return
	}
	status, ok := server.decodeStatus(writer, request)
	if !ok {
		return
	}
	switch server.service.TerminalPresence(taskID, terminalID) {
	case TerminalRemoved:
		server.writeError(writer, http.StatusConflict, "终端已关闭")
		return
	case TerminalMissing:
		server.writeError(writer, http.StatusNotFound, "终端不存在")
		return
	}
	server.service.SetTerminalStatus(taskID, terminalID, status)
	server.writeJSON(writer, http.StatusOK, StatusUpdateResponse{
		TaskID:         taskID,
		TaskStatus:     server.service.TaskStatus(taskID),
		TerminalID:     terminalID,
		TerminalStatus: server.service.TerminalStatus(taskID, terminalID),
	})
}

func (server *HTTPServer) requireRunningTask(writer http.ResponseWriter, taskID string) bool {
	switch server.resolveTask(taskID) {
	case TaskStateRunning:
		return true
	case TaskStateEnded:
		server.writeError(writer, http.StatusConflict, "任务已结束")
	default:
		server.writeError(writer, http.StatusNotFound, "任务不存在")
	}
	return false
}

func (server *HTTPServer) decodeStatus(writer http.ResponseWriter, request *http.Request) (Status, bool) {
	var payload struct {
		Status Status `json:"status"`
	}
	decoder := json.NewDecoder(request.Body)
	if err := decoder.Decode(&payload); err != nil || !payload.Status.Valid() {
		server.writeError(writer, http.StatusBadRequest, "状态请求无效")
		return "", false
	}
	return payload.Status, true
}

func (server *HTTPServer) writeMethodNotAllowed(writer http.ResponseWriter) {
	server.writeError(writer, http.StatusMethodNotAllowed, "不支持的请求方法")
}

func (server *HTTPServer) writeError(writer http.ResponseWriter, status int, message string) {
	server.writeJSON(writer, status, map[string]string{"error": message})
}

func (server *HTTPServer) writeJSON(writer http.ResponseWriter, status int, payload any) {
	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(payload)
}
