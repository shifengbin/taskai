package lifecycle

import (
	"fmt"
	"time"

	"taskai/internal/storage"
	"taskai/internal/task"
	"taskai/internal/workspace"
)

type TerminalCloser interface {
	CloseTask(taskID string) error
}

type TerminalReopener interface {
	ReopenTask(taskID string)
}

type Service struct {
	repository storage.Repository
	closer     TerminalCloser
	now        func() time.Time
}

func New(repository storage.Repository, closer TerminalCloser, now func() time.Time) *Service {
	return &Service{repository: repository, closer: closer, now: now}
}

func (service *Service) CreateTask(title, description, color string) (task.Task, error) {
	return service.CreateTaskWithExtraInfo(title, description, color, nil)
}

func (service *Service) CreateTaskWithExtraInfo(title, description, color string, extraInfo []task.ExtraInfo) (task.Task, error) {
	data, err := service.repository.Load()
	if err != nil {
		return task.Task{}, err
	}

	created, err := task.NewTask(title, description, color, service.now())
	if err != nil {
		return task.Task{}, err
	}
	created, err = created.UpdateExtraInfo(extraInfo)
	if err != nil {
		return task.Task{}, err
	}
	data.Tasks = append(data.Tasks, created)
	if err := service.repository.Save(data); err != nil {
		return task.Task{}, err
	}

	return created, nil
}

func (service *Service) ListTasks() ([]task.Task, error) {
	data, err := service.repository.Load()
	if err != nil {
		return nil, err
	}

	return data.Tasks, nil
}

func (service *Service) ReorderTasks(status task.Status, taskIDs []string) ([]task.Task, error) {
	if !isTaskStatus(status) {
		return nil, fmt.Errorf("不支持的任务状态: %q", status)
	}
	data, err := service.repository.Load()
	if err != nil {
		return nil, err
	}

	positions := make([]int, 0)
	tasksByID := make(map[string]task.Task)
	for index, current := range data.Tasks {
		if current.Status != status {
			continue
		}
		positions = append(positions, index)
		tasksByID[current.ID] = current
	}
	if len(taskIDs) != len(positions) {
		return nil, fmt.Errorf("任务排序数量不匹配")
	}

	seen := make(map[string]bool, len(taskIDs))
	for _, taskID := range taskIDs {
		if seen[taskID] {
			return nil, fmt.Errorf("任务排序包含重复任务: %q", taskID)
		}
		if _, ok := tasksByID[taskID]; !ok {
			return nil, fmt.Errorf("任务排序包含无效任务: %q", taskID)
		}
		seen[taskID] = true
	}
	for index, position := range positions {
		data.Tasks[position] = tasksByID[taskIDs[index]]
	}
	if err := service.repository.Save(data); err != nil {
		return nil, err
	}

	return data.Tasks, nil
}

func (service *Service) UpdateTask(taskID, title, description, color string) (task.Task, error) {
	data, err := service.repository.Load()
	if err != nil {
		return task.Task{}, err
	}
	index, err := taskIndex(data.Tasks, taskID)
	if err != nil {
		return task.Task{}, err
	}
	updated, err := data.Tasks[index].UpdateDetails(title, description, color)
	if err != nil {
		return task.Task{}, err
	}
	data.Tasks[index] = updated
	if err := service.repository.Save(data); err != nil {
		return task.Task{}, err
	}
	return updated, nil
}

func (service *Service) UpdateTaskWithExtraInfo(taskID, title, description, color string, extraInfo []task.ExtraInfo) (task.Task, error) {
	data, err := service.repository.Load()
	if err != nil {
		return task.Task{}, err
	}
	index, err := taskIndex(data.Tasks, taskID)
	if err != nil {
		return task.Task{}, err
	}
	updated, err := data.Tasks[index].UpdateDetails(title, description, color)
	if err != nil {
		return task.Task{}, err
	}
	updated, err = updated.UpdateExtraInfo(extraInfo)
	if err != nil {
		return task.Task{}, err
	}
	data.Tasks[index] = updated
	if err := service.repository.Save(data); err != nil {
		return task.Task{}, err
	}
	return updated, nil
}

func (service *Service) StartTask(taskID string) (task.Task, error) {
	data, err := service.repository.Load()
	if err != nil {
		return task.Task{}, err
	}
	index, err := taskIndex(data.Tasks, taskID)
	if err != nil {
		return task.Task{}, err
	}
	if data.Tasks[index].Status != task.StatusPending {
		return task.Task{}, fmt.Errorf("任务当前状态不能开始执行")
	}

	workspacePath, err := workspace.Create(data.Settings.WorkspaceRoot, taskID)
	if err != nil {
		return task.Task{}, err
	}
	data.Tasks[index].Status = task.StatusRunning
	data.Tasks[index].WorkspaceRoot = data.Settings.WorkspaceRoot
	data.Tasks[index].WorkspacePath = workspacePath
	if err := service.repository.Save(data); err != nil {
		workspace.Remove(data.Settings.WorkspaceRoot, workspacePath, taskID)
		return task.Task{}, err
	}

	return data.Tasks[index], nil
}

func (service *Service) FinishTask(taskID string) (task.Task, error) {
	data, err := service.repository.Load()
	if err != nil {
		return task.Task{}, err
	}
	index, err := taskIndex(data.Tasks, taskID)
	if err != nil {
		return task.Task{}, err
	}
	current := data.Tasks[index]
	if current.Status != task.StatusRunning {
		return task.Task{}, fmt.Errorf("任务当前状态不能结束执行")
	}
	if err := service.closer.CloseTask(taskID); err != nil {
		return task.Task{}, fmt.Errorf("关闭任务终端失败: %w", err)
	}
	if err := workspace.Remove(current.WorkspaceRoot, current.WorkspacePath, taskID); err != nil {
		if reopener, ok := service.closer.(TerminalReopener); ok {
			reopener.ReopenTask(taskID)
		}
		return task.Task{}, err
	}

	completedAt := service.now()
	data.Tasks[index].Status = task.StatusCompleted
	data.Tasks[index].CompletedAt = &completedAt
	if err := service.repository.Save(data); err != nil {
		return task.Task{}, err
	}

	return data.Tasks[index], nil
}

func taskIndex(tasks []task.Task, taskID string) (int, error) {
	for index, current := range tasks {
		if current.ID == taskID {
			return index, nil
		}
	}

	return 0, fmt.Errorf("任务不存在")
}

func isTaskStatus(status task.Status) bool {
	return status == task.StatusPending || status == task.StatusRunning || status == task.StatusCompleted
}
