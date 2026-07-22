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

func (service *Service) CreateTask(title, description string) (task.Task, error) {
	data, err := service.repository.Load()
	if err != nil {
		return task.Task{}, err
	}

	created, err := task.NewTask(title, description, service.now())
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
