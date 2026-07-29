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
	repository *storage.Repository
	closer     TerminalCloser
	now        func() time.Time
}

func New(repository *storage.Repository, closer TerminalCloser, now func() time.Time) *Service {
	return &Service{repository: repository, closer: closer, now: now}
}

func (service *Service) CreateTask(title, description, color string) (task.Task, error) {
	return service.CreateTaskWithExtraInfo(title, description, color, nil)
}

func (service *Service) CreateTaskWithExtraInfo(title, description, color string, extraInfo []task.TaskExtraInfo) (task.Task, error) {
	data, err := service.repository.Load()
	if err != nil {
		return task.Task{}, err
	}

	created, err := task.NewTask(title, description, color, service.now())
	if err != nil {
		return task.Task{}, err
	}
	snapshots, err := buildTaskExtraInfoSnapshots(data, nil, extraInfo)
	if err != nil {
		return task.Task{}, err
	}
	created, err = created.UpdateExtraInfo(snapshots)
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

func (service *Service) UpdateTaskWithExtraInfo(taskID, title, description, color string, extraInfo []task.TaskExtraInfo) (task.Task, error) {
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
	snapshots, err := buildTaskExtraInfoSnapshots(data, data.Tasks[index].ExtraInfo, extraInfo)
	if err != nil {
		return task.Task{}, err
	}
	updated, err = updated.UpdateExtraInfo(snapshots)
	if err != nil {
		return task.Task{}, err
	}
	data.Tasks[index] = updated
	if err := service.repository.Save(data); err != nil {
		return task.Task{}, err
	}
	return updated, nil
}

func buildTaskExtraInfoSnapshots(data storage.Data, existing, requested []task.TaskExtraInfo) ([]task.TaskExtraInfo, error) {
	existingByID := make(map[string]task.TaskExtraInfo, len(existing))
	existingByInformationID := make(map[string]task.TaskExtraInfo, len(existing))
	for _, current := range existing {
		existingByID[current.ID] = current
		if current.InformationID != "" {
			existingByInformationID[current.InformationID] = current
		}
	}

	infos := make(map[string]task.ExtraInfo, len(data.ExtraInfos))
	for _, info := range data.ExtraInfos {
		infos[info.ID] = info
	}
	templates := make(map[string]task.ExtraInfoTemplate, len(data.ExtraInfoTemplates))
	for _, template := range data.ExtraInfoTemplates {
		templates[template.ID] = template
	}

	snapshots := make([]task.TaskExtraInfo, 0, len(requested))
	for _, item := range requested {
		if previous, ok := existingByID[item.ID]; ok {
			updated, err := updateExistingTaskExtraInfo(previous, item)
			if err != nil {
				return nil, err
			}
			snapshots = append(snapshots, updated)
			continue
		}
		if previous, ok := existingByInformationID[item.InformationID]; ok {
			updated, err := updateExistingTaskExtraInfo(previous, item)
			if err != nil {
				return nil, err
			}
			snapshots = append(snapshots, updated)
			continue
		}

		information, ok := infos[item.InformationID]
		if !ok {
			return nil, fmt.Errorf("额外信息 %q 不存在，无法创建任务快照", item.InformationID)
		}
		template, ok := templates[information.TemplateID]
		if !ok {
			return nil, fmt.Errorf("额外信息 %q 的模板不存在", item.InformationID)
		}
		values, additional, err := splitTaskParameterValues(information, template, item.Parameters)
		if err != nil {
			return nil, err
		}
		snapshot, err := task.NewTaskExtraInfo(information, template, values, additional)
		if err != nil {
			return nil, err
		}
		snapshots = append(snapshots, snapshot)
	}
	return task.ValidateTaskExtraInfo(snapshots)
}

func updateExistingTaskExtraInfo(previous, requested task.TaskExtraInfo) (task.TaskExtraInfo, error) {
	normalizedPrevious, err := task.NormalizeTaskExtraInfo(previous)
	if err != nil {
		return task.TaskExtraInfo{}, err
	}
	normalizedRequested, err := task.NormalizeTaskExtraInfo(requested)
	if err != nil {
		return task.TaskExtraInfo{}, err
	}
	if normalizedRequested.InformationID != "" && normalizedRequested.InformationID != normalizedPrevious.InformationID {
		return task.TaskExtraInfo{}, fmt.Errorf("任务固定信息来源不可修改")
	}
	if normalizedRequested.TemplateID != "" && normalizedRequested.TemplateID != normalizedPrevious.TemplateID {
		return task.TaskExtraInfo{}, fmt.Errorf("任务固定信息来源不可修改")
	}
	if !sameFields(normalizedPrevious.Fields, normalizedRequested.Fields) {
		return task.TaskExtraInfo{}, fmt.Errorf("任务固定字段不可修改")
	}

	values := make(map[string]string, len(normalizedRequested.Parameters))
	requestedByKey := make(map[string]task.ExtraInfoParameter, len(normalizedRequested.Parameters))
	for _, parameter := range normalizedRequested.Parameters {
		requestedByKey[parameter.Key] = parameter
	}
	parameters := make([]task.ExtraInfoParameter, 0, len(normalizedRequested.Parameters))
	for _, parameter := range normalizedPrevious.Parameters {
		requestedParameter, ok := requestedByKey[parameter.Key]
		if !ok {
			return task.TaskExtraInfo{}, fmt.Errorf("任务动态参数 %q 不可删除", parameter.DisplayName)
		}
		if requestedParameter.DisplayName != parameter.DisplayName || requestedParameter.Required != parameter.Required || task.NormalizeExtraInfoParameterInputType(requestedParameter.InputType) != task.NormalizeExtraInfoParameterInputType(parameter.InputType) {
			return task.TaskExtraInfo{}, fmt.Errorf("任务动态参数定义不可修改: %s", parameter.DisplayName)
		}
		values[parameter.Key] = requestedParameter.Value
		delete(requestedByKey, parameter.Key)
	}
	for _, parameter := range normalizedPrevious.Parameters {
		parameters = append(parameters, task.ExtraInfoParameter{
			Key:         parameter.Key,
			DisplayName: parameter.DisplayName,
			Required:    parameter.Required,
			InputType:   parameter.InputType,
			Value:       values[parameter.Key],
		})
	}
	for _, parameter := range normalizedRequested.Parameters {
		if additional, ok := requestedByKey[parameter.Key]; ok {
			parameters = append(parameters, additional)
		}
	}
	normalizedPrevious.Parameters = parameters
	return task.NormalizeTaskExtraInfo(normalizedPrevious)
}

func splitTaskParameterValues(information task.ExtraInfo, template task.ExtraInfoTemplate, parameters []task.ExtraInfoParameter) (map[string]string, []task.ExtraInfoParameter, error) {
	definitions := make(map[string]task.ExtraInfoParameterDefinition, len(template.Parameters))
	for _, definition := range template.Parameters {
		definitions[definition.Key] = definition
	}
	informationDefinitions := make(map[string]task.ExtraInfoParameter, len(information.Parameters))
	for _, definition := range information.Parameters {
		informationDefinitions[definition.Key] = definition
	}
	values := make(map[string]string, len(definitions))
	additional := make([]task.ExtraInfoParameter, 0)
	seen := make(map[string]bool, len(parameters))
	for _, parameter := range parameters {
		if seen[parameter.Key] {
			return nil, nil, fmt.Errorf("任务动态参数键重复: %q", parameter.Key)
		}
		seen[parameter.Key] = true
		if definition, ok := definitions[parameter.Key]; ok {
			if parameter.DisplayName != "" && parameter.DisplayName != definition.DisplayName {
				return nil, nil, fmt.Errorf("模板动态参数定义不可修改: %s", definition.DisplayName)
			}
			if parameter.Required != definition.Required {
				return nil, nil, fmt.Errorf("模板动态参数定义不可修改: %s", definition.DisplayName)
			}
			if task.NormalizeExtraInfoParameterInputType(parameter.InputType) != task.NormalizeExtraInfoParameterInputType(definition.InputType) {
				return nil, nil, fmt.Errorf("模板动态参数定义不可修改: %s", definition.DisplayName)
			}
			values[parameter.Key] = parameter.Value
			continue
		}
		if definition, ok := informationDefinitions[parameter.Key]; ok {
			if parameter.DisplayName != "" && parameter.DisplayName != definition.DisplayName {
				return nil, nil, fmt.Errorf("信息动态参数定义不可修改: %s", definition.DisplayName)
			}
			if parameter.Required != definition.Required {
				return nil, nil, fmt.Errorf("信息动态参数定义不可修改: %s", definition.DisplayName)
			}
			if task.NormalizeExtraInfoParameterInputType(parameter.InputType) != task.NormalizeExtraInfoParameterInputType(definition.InputType) {
				return nil, nil, fmt.Errorf("信息动态参数定义不可修改: %s", definition.DisplayName)
			}
			values[parameter.Key] = parameter.Value
			continue
		}
		additional = append(additional, parameter)
	}
	return values, additional, nil
}

func sameFields(left, right []task.ExtraInfoField) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index].Key != right[index].Key || left[index].DisplayName != right[index].DisplayName || left[index].Value != right[index].Value {
			return false
		}
	}
	return true
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
