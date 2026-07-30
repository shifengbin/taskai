package workspace

import (
	"fmt"
	"os"
	"path/filepath"
)

func Create(root, taskID string) (string, error) {
	absRoot, workspacePath, err := TaskPath(root, taskID)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(absRoot, 0o700); err != nil {
		return "", fmt.Errorf("创建任务工作区根目录失败: %w", err)
	}
	if info, err := os.Lstat(workspacePath); err == nil {
		if info.IsDir() {
			return workspacePath, nil
		}
		return "", fmt.Errorf("任务工作目录已存在且不是目录")
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("检查任务工作目录失败: %w", err)
	}
	if err := os.Mkdir(workspacePath, 0o700); err != nil {
		return "", fmt.Errorf("创建任务工作目录失败: %w", err)
	}

	return workspacePath, nil
}

func TaskPath(root, taskID string) (string, string, error) {
	if err := validateTaskID(taskID); err != nil {
		return "", "", err
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", "", fmt.Errorf("解析任务工作区根目录失败: %w", err)
	}
	absRoot = filepath.Clean(absRoot)
	workspacePath := filepath.Join(absRoot, taskID)
	if !isDirectTaskChild(absRoot, workspacePath, taskID) {
		return "", "", fmt.Errorf("任务工作目录不安全")
	}
	return absRoot, workspacePath, nil
}

func Remove(root, workspacePath, taskID string) error {
	if err := validateTaskID(taskID); err != nil {
		return err
	}

	absRoot, err := filepath.Abs(root)
	if err != nil {
		return fmt.Errorf("解析任务工作区根目录失败: %w", err)
	}
	absWorkspacePath, err := filepath.Abs(workspacePath)
	if err != nil {
		return fmt.Errorf("解析任务工作目录失败: %w", err)
	}
	absRoot = filepath.Clean(absRoot)
	absWorkspacePath = filepath.Clean(absWorkspacePath)
	if !isDirectTaskChild(absRoot, absWorkspacePath, taskID) {
		return fmt.Errorf("拒绝删除非任务工作目录")
	}

	canonicalRoot, err := filepath.EvalSymlinks(absRoot)
	if err != nil {
		return fmt.Errorf("解析任务工作区根目录失败: %w", err)
	}
	canonicalWorkspacePath, err := filepath.EvalSymlinks(absWorkspacePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("解析任务工作目录失败: %w", err)
	}
	if !isDirectTaskChild(canonicalRoot, canonicalWorkspacePath, taskID) {
		return fmt.Errorf("拒绝删除指向工作区外的目录")
	}

	if err := os.RemoveAll(absWorkspacePath); err != nil {
		return fmt.Errorf("删除任务工作目录失败: %w", err)
	}

	return nil
}

func validateTaskID(taskID string) error {
	if taskID == "" || taskID == "." || taskID == ".." || filepath.Base(taskID) != taskID {
		return fmt.Errorf("任务 ID 无效")
	}

	return nil
}

func isDirectTaskChild(root, candidate, taskID string) bool {
	relativePath, err := filepath.Rel(root, candidate)
	if err != nil {
		return false
	}

	return relativePath == taskID
}
