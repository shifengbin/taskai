package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

type applicationInstanceLock struct {
	file     *os.File
	close    sync.Once
	closeErr error
}

func acquireApplicationInstanceLock(dataDirectory string) (*applicationInstanceLock, error) {
	if err := os.MkdirAll(dataDirectory, 0o700); err != nil {
		return nil, fmt.Errorf("创建应用数据目录失败: %w", err)
	}
	lockPath := filepath.Join(dataDirectory, ".instance.lock")
	file, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("打开应用实例锁失败: %w", err)
	}
	if err := lockApplicationInstanceFile(file); err != nil {
		_ = file.Close()
		return nil, fmt.Errorf("TaskAI 已在运行或无法获取应用实例锁: %w", err)
	}
	return &applicationInstanceLock{file: file}, nil
}

func (lock *applicationInstanceLock) Close() error {
	if lock == nil || lock.file == nil {
		return nil
	}
	lock.close.Do(func() {
		if err := unlockApplicationInstanceFile(lock.file); err != nil {
			lock.closeErr = err
		}
		if err := lock.file.Close(); lock.closeErr == nil {
			lock.closeErr = err
		}
	})
	return lock.closeErr
}
