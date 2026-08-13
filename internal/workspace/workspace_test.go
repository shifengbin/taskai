package workspace

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
)

func TestCreateCreatesTaskIDChildDirectory(t *testing.T) {
	root := filepath.Join(t.TempDir(), "workspaces")

	result, err := Create(root, "task-1")

	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	path := result.Path
	if !result.Created {
		t.Fatal("Create() Created = false，期望报告本次新建")
	}
	if path != filepath.Join(root, "task-1") {
		t.Errorf("Create() path = %q, want %q", path, filepath.Join(root, "task-1"))
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	if !info.IsDir() {
		t.Errorf("Create() created non-directory %q", path)
	}
}

func TestCreateReusesExistingSafeTaskWorkspace(t *testing.T) {
	root := filepath.Join(t.TempDir(), "workspaces")
	first, err := Create(root, "task-1")
	if err != nil {
		t.Fatalf("first Create() error = %v", err)
	}
	second, err := Create(root, "task-1")
	if err != nil {
		t.Fatalf("second Create() error = %v", err)
	}
	if first.Path != second.Path || !first.Created || second.Created {
		t.Errorf("Create() 首次结果 = %#v，复用结果 = %#v", first, second)
	}
}

func TestRemoveDeletesMatchingTaskWorkspace(t *testing.T) {
	root := filepath.Join(t.TempDir(), "workspaces")
	result, err := Create(root, "task-1")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	path := result.Path
	if err := os.WriteFile(filepath.Join(path, "output.txt"), []byte("temporary"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	err = Remove(root, path, "task-1")

	if err != nil {
		t.Fatalf("Remove() error = %v", err)
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("Remove() workspace still exists, Stat() error = %v", err)
	}
	if _, err := os.Stat(root); err != nil {
		t.Errorf("Remove() removed workspace root: %v", err)
	}
}

func TestRemoveRejectsWorkspaceRoot(t *testing.T) {
	root := filepath.Join(t.TempDir(), "workspaces")
	if err := os.MkdirAll(root, 0o700); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	err := Remove(root, root, "task-1")

	if err == nil {
		t.Fatal("Remove() error = nil, want root deletion rejection")
	}
	if _, err := os.Stat(root); err != nil {
		t.Errorf("Remove() deleted workspace root: %v", err)
	}
}

func TestRemoveRejectsWorkspaceForDifferentTask(t *testing.T) {
	root := filepath.Join(t.TempDir(), "workspaces")
	result, err := Create(root, "task-2")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	path := result.Path

	err = Remove(root, path, "task-1")

	if err == nil {
		t.Fatal("Remove() error = nil, want mismatched task ID rejection")
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("Remove() deleted mismatched workspace: %v", err)
	}
}

func TestRemoveRejectsWorkspaceOutsideRoot(t *testing.T) {
	root := filepath.Join(t.TempDir(), "workspaces")
	outside := filepath.Join(t.TempDir(), "outside-task")
	if err := os.MkdirAll(outside, 0o700); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	err := Remove(root, outside, "outside-task")

	if err == nil {
		t.Fatal("Remove() error = nil, want outside path rejection")
	}
	if _, err := os.Stat(outside); err != nil {
		t.Errorf("Remove() deleted outside directory: %v", err)
	}
}

func TestRemoveRejectsSymlinkToOutsideRoot(t *testing.T) {
	root := filepath.Join(t.TempDir(), "workspaces")
	if err := os.MkdirAll(root, 0o700); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	outside := filepath.Join(t.TempDir(), "outside-task")
	if err := os.MkdirAll(outside, 0o700); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	path := filepath.Join(root, "task-1")
	if err := os.Symlink(outside, path); err != nil {
		t.Skipf("Symlink() unavailable: %v", err)
	}

	err := Remove(root, path, "task-1")

	if err == nil {
		t.Fatal("Remove() error = nil, want symlink rejection")
	}
	if _, err := os.Stat(outside); err != nil {
		t.Errorf("Remove() deleted symlink target: %v", err)
	}
}

func TestCreateOwnedCanRecoverAfterDirectoryCreationBeforeStateUpdate(t *testing.T) {
	root := filepath.Join(t.TempDir(), "workspaces")
	token, err := NewOwnershipToken()
	if err != nil {
		t.Fatalf("NewOwnershipToken() error = %v", err)
	}
	first, err := CreateOwned(root, "task-1", token)
	if err != nil {
		t.Fatalf("first CreateOwned() error = %v", err)
	}
	second, err := CreateOwned(root, "task-1", token)
	if err != nil {
		t.Fatalf("second CreateOwned() error = %v", err)
	}
	if !first.Created || !second.Created || first.Path != second.Path {
		t.Fatalf("CreateOwned() results = first %#v second %#v", first, second)
	}

	removed, err := RemoveOwned(root, first.Path, "task-1", token)
	if err != nil {
		t.Fatalf("RemoveOwned() error = %v", err)
	}
	if !removed {
		t.Fatal("RemoveOwned() removed = false")
	}
	if _, err := os.Stat(first.Path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("RemoveOwned() 后目录仍存在: %v", err)
	}
}

func TestCreateOwnedCleansClaimedStagingWhenTargetWasOccupiedAfterCrash(t *testing.T) {
	root := filepath.Join(t.TempDir(), "workspaces")
	token, err := NewOwnershipToken()
	if err != nil {
		t.Fatalf("NewOwnershipToken() error = %v", err)
	}
	workspacePath, stagingPath, claimPath := stageOwnedWorkspaceForTest(t, root, "task-1", token)
	if err := os.Mkdir(workspacePath, 0o700); err != nil {
		t.Fatalf("Mkdir() replacement error = %v", err)
	}
	markerPath := filepath.Join(workspacePath, "keep.txt")
	if err := os.WriteFile(markerPath, []byte("keep"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	result, err := CreateOwned(root, "task-1", token)

	if err != nil {
		t.Fatalf("CreateOwned() error = %v", err)
	}
	if result.Path != workspacePath || result.Created {
		t.Fatalf("CreateOwned() result = %#v", result)
	}
	if contents, readErr := os.ReadFile(markerPath); readErr != nil || string(contents) != "keep" {
		t.Fatalf("占用目录被修改: contents=%q err=%v", contents, readErr)
	}
	assertPathMissing(t, stagingPath)
	assertPathMissing(t, claimPath)
}

func TestRemoveOwnedCleansClaimedStagingWithoutTouchingOccupiedTarget(t *testing.T) {
	root := filepath.Join(t.TempDir(), "workspaces")
	token, err := NewOwnershipToken()
	if err != nil {
		t.Fatalf("NewOwnershipToken() error = %v", err)
	}
	workspacePath, stagingPath, claimPath := stageOwnedWorkspaceForTest(t, root, "task-1", token)
	if err := os.Mkdir(workspacePath, 0o700); err != nil {
		t.Fatalf("Mkdir() replacement error = %v", err)
	}
	markerPath := filepath.Join(workspacePath, "keep.txt")
	if err := os.WriteFile(markerPath, []byte("keep"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	removed, err := RemoveOwned(root, workspacePath, "task-1", token)

	if err != nil {
		t.Fatalf("RemoveOwned() error = %v", err)
	}
	if !removed {
		t.Fatal("RemoveOwned() removed = false")
	}
	if contents, readErr := os.ReadFile(markerPath); readErr != nil || string(contents) != "keep" {
		t.Fatalf("占用目录被修改: contents=%q err=%v", contents, readErr)
	}
	assertPathMissing(t, stagingPath)
	assertPathMissing(t, claimPath)
}

func TestRemoveOwnedRejectsReplacedSameNameDirectory(t *testing.T) {
	root := filepath.Join(t.TempDir(), "workspaces")
	token, err := NewOwnershipToken()
	if err != nil {
		t.Fatalf("NewOwnershipToken() error = %v", err)
	}
	created, err := CreateOwned(root, "task-1", token)
	if err != nil {
		t.Fatalf("CreateOwned() error = %v", err)
	}
	originalPath := filepath.Join(t.TempDir(), "original")
	if err := os.Rename(created.Path, originalPath); err != nil {
		t.Fatalf("Rename() error = %v", err)
	}
	if err := os.Mkdir(created.Path, 0o700); err != nil {
		t.Fatalf("Mkdir() replacement error = %v", err)
	}
	markerPath := filepath.Join(created.Path, "keep.txt")
	if err := os.WriteFile(markerPath, []byte("keep"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	removed, err := RemoveOwned(root, created.Path, "task-1", token)
	if err == nil {
		t.Fatal("RemoveOwned() error = nil")
	}
	if removed {
		t.Fatal("RemoveOwned() removed = true")
	}
	contents, readErr := os.ReadFile(markerPath)
	if readErr != nil || string(contents) != "keep" {
		t.Fatalf("同名替换目录被修改: contents=%q err=%v", contents, readErr)
	}
	if _, statErr := os.Stat(originalPath); statErr != nil {
		t.Fatalf("原归属目录被修改: %v", statErr)
	}
}

func TestRemoveOwnedRejectsRecreatedSameNameDirectoryAfterOriginalDeletion(t *testing.T) {
	root := filepath.Join(t.TempDir(), "workspaces")
	token, err := NewOwnershipToken()
	if err != nil {
		t.Fatalf("NewOwnershipToken() error = %v", err)
	}
	created, err := CreateOwned(root, "task-1", token)
	if err != nil {
		t.Fatalf("CreateOwned() error = %v", err)
	}
	if err := os.RemoveAll(created.Path); err != nil {
		t.Fatalf("RemoveAll() error = %v", err)
	}
	if err := os.Mkdir(created.Path, 0o700); err != nil {
		t.Fatalf("Mkdir() replacement error = %v", err)
	}
	markerPath := filepath.Join(created.Path, "keep.txt")
	if err := os.WriteFile(markerPath, []byte("keep"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	removed, err := RemoveOwned(root, created.Path, "task-1", token)
	if err == nil {
		t.Fatal("RemoveOwned() error = nil")
	}
	if removed {
		t.Fatal("RemoveOwned() removed = true")
	}
	contents, readErr := os.ReadFile(markerPath)
	if readErr != nil || string(contents) != "keep" {
		t.Fatalf("同名重建目录被修改: contents=%q err=%v", contents, readErr)
	}
}

func TestCreateOwnedConcurrentFirstCreationRecoversSameWorkspace(t *testing.T) {
	root := filepath.Join(t.TempDir(), "workspaces")
	token, err := NewOwnershipToken()
	if err != nil {
		t.Fatalf("NewOwnershipToken() error = %v", err)
	}

	const workers = 16
	start := make(chan struct{})
	results := make(chan CreateResult, workers)
	errors := make(chan error, workers)
	var waitGroup sync.WaitGroup
	for range workers {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			<-start
			result, createErr := CreateOwned(root, "task-1", token)
			results <- result
			errors <- createErr
		}()
	}
	close(start)
	waitGroup.Wait()
	close(results)
	close(errors)

	for createErr := range errors {
		if createErr != nil {
			t.Fatalf("CreateOwned() concurrent error = %v", createErr)
		}
	}
	for result := range results {
		if result.Path != filepath.Join(root, "task-1") || !result.Created {
			t.Fatalf("CreateOwned() concurrent result = %#v", result)
		}
	}
}

func TestCreateOwnedRejectsPreexistingInsecureOwnershipMetadataDirectory(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows 通过 ACL 验证私有目录")
	}
	root := filepath.Join(t.TempDir(), "workspaces")
	metadataPath := filepath.Join(root, ownershipMetadataDirectory)
	if err := os.MkdirAll(metadataPath, 0o777); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.Chmod(metadataPath, 0o777); err != nil {
		t.Fatalf("Chmod() error = %v", err)
	}
	token, err := NewOwnershipToken()
	if err != nil {
		t.Fatalf("NewOwnershipToken() error = %v", err)
	}

	result, err := CreateOwned(root, "task-1", token)

	if err == nil {
		t.Fatalf("CreateOwned() result = %#v, error = nil", result)
	}
	info, statErr := os.Stat(metadataPath)
	if statErr != nil {
		t.Fatalf("Stat() error = %v", statErr)
	}
	if info.Mode().Perm() != 0o777 {
		t.Fatalf("不安全元数据目录被修改: mode=%v", info.Mode().Perm())
	}
	if _, statErr := os.Stat(filepath.Join(root, "task-1")); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("拒绝后仍创建任务目录: %v", statErr)
	}
}

func TestCreateOwnedAllowsStickyWritableWorkspaceRoot(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows 通过 ACL 验证工作区根目录")
	}
	root := filepath.Join(t.TempDir(), "workspaces")
	if err := os.MkdirAll(root, 0o777); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.Chmod(root, os.ModeSticky|0o777); err != nil {
		t.Fatalf("Chmod() error = %v", err)
	}
	token, err := NewOwnershipToken()
	if err != nil {
		t.Fatalf("NewOwnershipToken() error = %v", err)
	}

	result, err := CreateOwned(root, "task-1", token)

	if err != nil {
		t.Fatalf("CreateOwned() error = %v", err)
	}
	if !result.Created {
		t.Fatalf("CreateOwned() result = %#v", result)
	}
}

func TestDirectoryOwnershipCapabilityProbeLeavesNoArtifacts(t *testing.T) {
	root := filepath.Join(t.TempDir(), "workspaces")
	if err := os.MkdirAll(root, 0o700); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	metadataPath, err := ensureOwnershipMetadataDirectory(root)
	if err != nil {
		t.Fatalf("ensureOwnershipMetadataDirectory() error = %v", err)
	}

	if err := ensureDirectoryOwnershipCapability(metadataPath); err != nil {
		t.Fatalf("ensureDirectoryOwnershipCapability() error = %v", err)
	}

	entries, err := os.ReadDir(metadataPath)
	if err != nil {
		t.Fatalf("ReadDir() error = %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("能力探测留下文件: %v", entries)
	}
}

func TestCreateOwnedStopsBeforeTaskDirectoryWhenOwnershipMarkUnsupported(t *testing.T) {
	root := filepath.Join(t.TempDir(), "workspaces")
	token, err := NewOwnershipToken()
	if err != nil {
		t.Fatalf("NewOwnershipToken() error = %v", err)
	}
	originalWriter := writeDirectoryOwnershipToken
	writeDirectoryOwnershipToken = func(string, string) error {
		return errors.New("unsupported ownership mark")
	}
	t.Cleanup(func() { writeDirectoryOwnershipToken = originalWriter })

	result, err := CreateOwned(root, "task-1", token)

	if err == nil {
		t.Fatalf("CreateOwned() result = %#v, error = nil", result)
	}
	if _, statErr := os.Stat(filepath.Join(root, "task-1")); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("能力不足后仍创建任务目录: %v", statErr)
	}
	entries, readErr := os.ReadDir(filepath.Join(root, ownershipMetadataDirectory))
	if readErr != nil {
		t.Fatalf("ReadDir() error = %v", readErr)
	}
	if len(entries) != 0 {
		t.Fatalf("能力失败留下所有权文件: %v", entries)
	}
}

func TestRemoveOwnedDoesNotTouchPreexistingDirectoryWithoutClaim(t *testing.T) {
	root := filepath.Join(t.TempDir(), "workspaces")
	path := filepath.Join(root, "task-1")
	if err := os.MkdirAll(path, 0o700); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	token, err := NewOwnershipToken()
	if err != nil {
		t.Fatalf("NewOwnershipToken() error = %v", err)
	}

	removed, err := RemoveOwned(root, path, "task-1", token)
	if err != nil {
		t.Fatalf("RemoveOwned() error = %v", err)
	}
	if removed {
		t.Fatal("RemoveOwned() removed = true")
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("未归属目录被删除: %v", err)
	}
}

func TestRemoveOwnedTreatsMissingWorkspaceRootAsAlreadyRemoved(t *testing.T) {
	root := filepath.Join(t.TempDir(), "missing-workspaces")
	token, err := NewOwnershipToken()
	if err != nil {
		t.Fatalf("NewOwnershipToken() error = %v", err)
	}
	path := filepath.Join(root, "task-1")

	removed, err := RemoveOwned(root, path, "task-1", token)

	if err != nil {
		t.Fatalf("RemoveOwned() error = %v", err)
	}
	if removed {
		t.Fatal("RemoveOwned() removed = true")
	}
}

func TestRemoveOwnedRestoresWorkspaceWhenFilesystemCleanupFails(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows 权限模型不使用目录写权限触发清理失败")
	}
	root := filepath.Join(t.TempDir(), "workspaces")
	token, err := NewOwnershipToken()
	if err != nil {
		t.Fatalf("NewOwnershipToken() error = %v", err)
	}
	created, err := CreateOwned(root, "task-1", token)
	if err != nil {
		t.Fatalf("CreateOwned() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(created.Path, "keep.txt"), []byte("keep"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if err := os.Chmod(created.Path, 0o500); err != nil {
		t.Fatalf("Chmod() error = %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(created.Path, 0o700) })

	removed, err := RemoveOwned(root, created.Path, "task-1", token)
	if err == nil {
		t.Fatal("RemoveOwned() error = nil")
	}
	if removed {
		t.Fatal("RemoveOwned() removed = true")
	}
	if err := os.Chmod(created.Path, 0o700); err != nil {
		t.Fatalf("restore Chmod() error = %v", err)
	}
	if contents, readErr := os.ReadFile(filepath.Join(created.Path, "keep.txt")); readErr != nil || string(contents) != "keep" {
		t.Fatalf("清理失败后目录未恢复: contents=%q err=%v", contents, readErr)
	}
}

func TestRemoveRefusesFallbackWhenOwnershipClaimIsCorrupted(t *testing.T) {
	root := filepath.Join(t.TempDir(), "workspaces")
	token, err := NewOwnershipToken()
	if err != nil {
		t.Fatalf("NewOwnershipToken() error = %v", err)
	}
	created, err := CreateOwned(root, "task-1", token)
	if err != nil {
		t.Fatalf("CreateOwned() error = %v", err)
	}
	claimPath := filepath.Join(root, ownershipMetadataDirectory, token+".json")
	if err := os.WriteFile(claimPath, []byte("not-json"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if err := Remove(root, created.Path, "task-1"); err == nil {
		t.Fatal("Remove() error = nil")
	}
	if _, err := os.Stat(created.Path); err != nil {
		t.Fatalf("凭据损坏后目录被删除: %v", err)
	}
}

func TestPathExistsCheckedDoesNotTreatFilesystemErrorsAsMissing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "parent", "task-1")
	if err := os.MkdirAll(path, 0o700); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	parent := filepath.Dir(path)
	if runtime.GOOS == "windows" {
		t.Skip("Windows 权限模型不使用目录执行权限触发 Lstat 失败")
	}
	if err := os.Chmod(parent, 0o000); err != nil {
		t.Fatalf("Chmod() error = %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(parent, 0o700) })

	exists, err := pathExistsChecked(path)
	if err == nil {
		t.Fatal("pathExistsChecked() error = nil")
	}
	if exists {
		t.Fatal("pathExistsChecked() exists = true")
	}
}

func stageOwnedWorkspaceForTest(t *testing.T, root, taskID, token string) (string, string, string) {
	t.Helper()
	if err := os.MkdirAll(root, 0o700); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	metadataPath, err := ensureOwnershipMetadataDirectory(root)
	if err != nil {
		t.Fatalf("ensureOwnershipMetadataDirectory() error = %v", err)
	}
	claimPath, stagingPath, _, err := ownershipArtifactPaths(metadataPath, token)
	if err != nil {
		t.Fatalf("ownershipArtifactPaths() error = %v", err)
	}
	if err := os.Mkdir(stagingPath, 0o700); err != nil {
		t.Fatalf("Mkdir() staging error = %v", err)
	}
	if err := setDirectoryOwnershipToken(stagingPath, token); err != nil {
		t.Fatalf("setDirectoryOwnershipToken() error = %v", err)
	}
	identity, err := directoryIdentity(stagingPath)
	if err != nil {
		t.Fatalf("directoryIdentity() error = %v", err)
	}
	if err := writeOwnershipClaim(claimPath, ownershipClaim{TaskID: taskID, Token: token, Identity: identity}); err != nil {
		t.Fatalf("writeOwnershipClaim() error = %v", err)
	}
	return filepath.Join(root, taskID), stagingPath, claimPath
}

func assertPathMissing(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("路径仍存在 %q: %v", path, err)
	}
}
