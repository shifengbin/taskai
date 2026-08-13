package workspace

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const ownershipMetadataDirectory = ".taskai-ownership"

var ownershipMutationMu sync.Mutex

var writeDirectoryOwnershipToken = setDirectoryOwnershipToken

type ownershipClaim struct {
	TaskID   string `json:"taskId"`
	Token    string `json:"token"`
	Identity string `json:"identity"`
}

type CreateResult struct {
	Path    string
	Created bool
}

func Create(root, taskID string) (CreateResult, error) {
	absRoot, workspacePath, err := TaskPath(root, taskID)
	if err != nil {
		return CreateResult{}, err
	}
	if err := os.MkdirAll(absRoot, 0o700); err != nil {
		return CreateResult{}, fmt.Errorf("创建任务工作区根目录失败: %w", err)
	}
	if info, err := os.Lstat(workspacePath); err == nil {
		if info.IsDir() {
			return CreateResult{Path: workspacePath}, nil
		}
		return CreateResult{}, fmt.Errorf("任务工作目录已存在且不是目录")
	} else if !os.IsNotExist(err) {
		return CreateResult{}, fmt.Errorf("检查任务工作目录失败: %w", err)
	}
	if err := os.Mkdir(workspacePath, 0o700); err != nil {
		return CreateResult{}, fmt.Errorf("创建任务工作目录失败: %w", err)
	}

	return CreateResult{Path: workspacePath, Created: true}, nil
}

func NewOwnershipToken() (string, error) {
	contents := make([]byte, 32)
	if _, err := rand.Read(contents); err != nil {
		return "", fmt.Errorf("生成工作目录所有权令牌失败: %w", err)
	}
	return hex.EncodeToString(contents), nil
}

func CreateOwned(root, taskID, token string) (CreateResult, error) {
	ownershipMutationMu.Lock()
	defer ownershipMutationMu.Unlock()

	absRoot, workspacePath, err := TaskPath(root, taskID)
	if err != nil {
		return CreateResult{}, err
	}
	if err := validateOwnershipToken(token); err != nil {
		return CreateResult{}, err
	}
	if err := os.MkdirAll(absRoot, 0o700); err != nil {
		return CreateResult{}, fmt.Errorf("创建任务工作区根目录失败: %w", err)
	}
	metadataPath, err := ensureOwnershipMetadataDirectory(absRoot)
	if err != nil {
		return CreateResult{}, err
	}
	claimPath, stagingPath, quarantinePath, err := ownershipArtifactPaths(metadataPath, token)
	if err != nil {
		return CreateResult{}, err
	}

	claim, found, err := readOwnershipClaim(claimPath, taskID, token)
	if err != nil {
		return CreateResult{}, err
	}
	if found {
		ownedPath, err := claimedDirectoryPath(workspacePath, stagingPath, claim)
		if err != nil {
			return CreateResult{}, err
		}
		switch ownedPath {
		case workspacePath:
			return CreateResult{Path: workspacePath, Created: true}, nil
		case stagingPath:
			if err := renameNoReplace(stagingPath, workspacePath); err != nil {
				workspaceExists, checkErr := pathExistsChecked(workspacePath)
				if checkErr != nil {
					return CreateResult{}, fmt.Errorf("检查任务工作目录失败: %w", checkErr)
				}
				if workspaceExists {
					if cleanupErr := cleanupClaimedStaging(claimPath, stagingPath, quarantinePath, claim); cleanupErr != nil {
						return CreateResult{}, fmt.Errorf("任务工作目录在创建过程中被占用，清理待创建目录失败: %w", cleanupErr)
					}
					info, statErr := os.Lstat(workspacePath)
					if statErr == nil && info.IsDir() {
						return CreateResult{Path: workspacePath}, nil
					}
					return CreateResult{}, fmt.Errorf("任务工作目录已存在且不是目录")
				}
				return CreateResult{}, fmt.Errorf("完成任务工作目录创建失败: %w", err)
			}
			return CreateResult{Path: workspacePath, Created: true}, nil
		default:
			return CreateResult{}, fmt.Errorf("工作目录所有权凭据与现有目录不匹配")
		}
	}

	if info, err := os.Lstat(workspacePath); err == nil {
		if info.IsDir() {
			return CreateResult{Path: workspacePath}, nil
		}
		return CreateResult{}, fmt.Errorf("任务工作目录已存在且不是目录")
	} else if !os.IsNotExist(err) {
		return CreateResult{}, fmt.Errorf("检查任务工作目录失败: %w", err)
	}
	if err := removeUnclaimedStaging(stagingPath); err != nil {
		return CreateResult{}, err
	}
	if err := ensureDirectoryOwnershipCapability(metadataPath); err != nil {
		return CreateResult{}, fmt.Errorf("当前文件系统无法安全记录工作目录所有权: %w", err)
	}
	if err := os.Mkdir(stagingPath, 0o700); err != nil {
		return CreateResult{}, fmt.Errorf("准备任务工作目录失败: %w", err)
	}
	if err := writeDirectoryOwnershipToken(stagingPath, token); err != nil {
		_ = os.Remove(stagingPath)
		return CreateResult{}, fmt.Errorf("写入新建工作目录所有权标记失败: %w", err)
	}
	identity, err := directoryIdentity(stagingPath)
	if err != nil {
		_ = os.Remove(stagingPath)
		return CreateResult{}, fmt.Errorf("读取新建工作目录身份失败: %w", err)
	}
	claim = ownershipClaim{TaskID: taskID, Token: token, Identity: identity}
	if err := writeOwnershipClaim(claimPath, claim); err != nil {
		return CreateResult{}, err
	}
	if err := renameNoReplace(stagingPath, workspacePath); err != nil {
		workspaceExists, checkErr := pathExistsChecked(workspacePath)
		if checkErr != nil {
			return CreateResult{}, fmt.Errorf("检查任务工作目录失败: %w", checkErr)
		}
		if workspaceExists {
			if cleanupErr := cleanupOwnershipArtifacts(claimPath, stagingPath); cleanupErr != nil {
				return CreateResult{}, fmt.Errorf("任务工作目录已被占用，清理创建凭据失败: %w", cleanupErr)
			}
			info, statErr := os.Lstat(workspacePath)
			if statErr == nil && info.IsDir() {
				return CreateResult{Path: workspacePath}, nil
			}
			return CreateResult{}, fmt.Errorf("任务工作目录已存在且不是目录")
		}
		return CreateResult{}, fmt.Errorf("完成任务工作目录创建失败: %w", err)
	}

	return CreateResult{Path: workspacePath, Created: true}, nil
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
	ownershipToken, found, err := findOwnershipClaimToken(absRoot, taskID)
	if err != nil {
		return err
	}
	if found {
		removed, err := RemoveOwned(absRoot, absWorkspacePath, taskID, ownershipToken)
		if err != nil {
			return err
		}
		if removed {
			return nil
		}
	}
	if _, err := os.Lstat(absWorkspacePath); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("检查任务工作目录失败: %w", err)
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

func RemoveOwned(root, workspacePath, taskID, token string) (bool, error) {
	ownershipMutationMu.Lock()
	defer ownershipMutationMu.Unlock()

	absRoot, expectedWorkspacePath, err := TaskPath(root, taskID)
	if err != nil {
		return false, err
	}
	if err := validateOwnershipToken(token); err != nil {
		return false, err
	}
	rootExists, err := pathExistsChecked(absRoot)
	if err != nil {
		return false, fmt.Errorf("检查任务工作区根目录失败: %w", err)
	}
	if !rootExists {
		return false, nil
	}
	metadataPath, exists, err := validatedOwnershipMetadataDirectory(absRoot)
	if err != nil {
		return false, err
	}
	if !exists {
		return false, nil
	}
	claimPath, stagingPath, quarantinePath, err := ownershipArtifactPaths(metadataPath, token)
	if err != nil {
		return false, err
	}
	claim, found, err := readOwnershipClaim(claimPath, taskID, token)
	if err != nil {
		return false, err
	}
	if !found {
		if err := removeUnclaimedStaging(stagingPath); err != nil {
			return false, err
		}
		return false, nil
	}
	absWorkspacePath, err := filepath.Abs(workspacePath)
	if err != nil {
		return false, fmt.Errorf("解析任务工作目录失败: %w", err)
	}
	absWorkspacePath = filepath.Clean(absWorkspacePath)
	if absWorkspacePath != expectedWorkspacePath {
		return false, fmt.Errorf("拒绝删除非任务工作目录")
	}

	quarantineMatches, err := directoryMatchesClaim(quarantinePath, claim)
	if err != nil {
		return false, fmt.Errorf("验证隔离工作目录所有权失败: %w", err)
	}
	if quarantineMatches {
		if err := os.RemoveAll(quarantinePath); err != nil {
			return false, fmt.Errorf("删除隔离工作目录失败: %w", err)
		}
		if err := os.Remove(claimPath); err != nil && !os.IsNotExist(err) {
			return false, fmt.Errorf("删除工作目录所有权凭据失败: %w", err)
		}
		return true, nil
	}
	quarantineExists, err := pathExistsChecked(quarantinePath)
	if err != nil {
		return false, fmt.Errorf("检查工作目录隔离路径失败: %w", err)
	}
	if quarantineExists {
		return false, fmt.Errorf("工作目录隔离路径已被占用")
	}
	stagingMatches, err := directoryMatchesClaim(stagingPath, claim)
	if err != nil {
		return false, fmt.Errorf("验证待创建工作目录所有权失败: %w", err)
	}
	if stagingMatches {
		if err := cleanupClaimedStaging(claimPath, stagingPath, quarantinePath, claim); err != nil {
			return false, err
		}
		return true, nil
	}
	stagingExists, err := pathExistsChecked(stagingPath)
	if err != nil {
		return false, fmt.Errorf("检查待创建工作目录失败: %w", err)
	}
	if stagingExists {
		return false, fmt.Errorf("工作目录所有权凭据与待创建目录不匹配")
	}

	workspaceExists, err := pathExistsChecked(absWorkspacePath)
	if err != nil {
		return false, fmt.Errorf("检查任务工作目录失败: %w", err)
	}
	if !workspaceExists {
		if err := os.Remove(claimPath); err != nil && !os.IsNotExist(err) {
			return false, fmt.Errorf("删除工作目录所有权凭据失败: %w", err)
		}
		return true, nil
	}
	workspaceMatches, err := directoryMatchesClaim(absWorkspacePath, claim)
	if err != nil {
		return false, fmt.Errorf("验证任务工作目录所有权失败: %w", err)
	}
	if !workspaceMatches {
		return false, fmt.Errorf("任务工作目录已被替换，拒绝删除")
	}
	if err := renameNoReplace(absWorkspacePath, quarantinePath); err != nil {
		return false, fmt.Errorf("隔离任务工作目录失败: %w", err)
	}
	quarantineMatches, err = directoryMatchesClaim(quarantinePath, claim)
	if err != nil || !quarantineMatches {
		restoreErr := renameNoReplace(quarantinePath, absWorkspacePath)
		if restoreErr != nil {
			return false, fmt.Errorf("任务工作目录在删除时被替换，恢复目录失败: %v", restoreErr)
		}
		if err != nil {
			return false, fmt.Errorf("验证隔离工作目录所有权失败: %w", err)
		}
		return false, fmt.Errorf("任务工作目录在删除时被替换，拒绝删除")
	}
	if err := os.RemoveAll(quarantinePath); err != nil {
		if restoreErr := renameNoReplace(quarantinePath, absWorkspacePath); restoreErr != nil {
			return false, fmt.Errorf("删除任务工作目录失败: %v；恢复目录失败: %v", err, restoreErr)
		}
		return false, fmt.Errorf("删除任务工作目录失败: %w", err)
	}
	if err := os.Remove(claimPath); err != nil && !os.IsNotExist(err) {
		return false, fmt.Errorf("删除工作目录所有权凭据失败: %w", err)
	}
	return true, nil
}

func cleanupClaimedStaging(claimPath, stagingPath, quarantinePath string, claim ownershipClaim) error {
	if err := renameNoReplace(stagingPath, quarantinePath); err != nil {
		return fmt.Errorf("隔离待创建工作目录失败: %w", err)
	}
	matches, err := directoryMatchesClaim(quarantinePath, claim)
	if err != nil || !matches {
		restoreErr := renameNoReplace(quarantinePath, stagingPath)
		if restoreErr != nil {
			return fmt.Errorf("待创建工作目录在清理时被替换，恢复失败: %v", restoreErr)
		}
		if err != nil {
			return fmt.Errorf("验证隔离待创建工作目录所有权失败: %w", err)
		}
		return fmt.Errorf("待创建工作目录在清理时被替换")
	}
	if err := os.RemoveAll(quarantinePath); err != nil {
		if restoreErr := renameNoReplace(quarantinePath, stagingPath); restoreErr != nil {
			return fmt.Errorf("清理待创建工作目录失败: %v；恢复失败: %v", err, restoreErr)
		}
		return fmt.Errorf("清理待创建工作目录失败: %w", err)
	}
	if err := os.Remove(claimPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("删除工作目录所有权凭据失败: %w", err)
	}
	return nil
}

func ensureOwnershipMetadataDirectory(root string) (string, error) {
	if err := validateOwnershipRoot(root); err != nil {
		return "", fmt.Errorf("任务工作区根目录不安全: %w", err)
	}
	metadataPath, err := ownershipMetadataPath(root)
	if err != nil {
		return "", err
	}
	created := false
	info, err := os.Lstat(metadataPath)
	if os.IsNotExist(err) {
		mkdirErr := createPrivateDirectory(metadataPath)
		if mkdirErr != nil && !os.IsExist(mkdirErr) {
			return "", fmt.Errorf("创建工作目录所有权数据目录失败: %w", mkdirErr)
		}
		created = mkdirErr == nil
		info, err = os.Lstat(metadataPath)
	}
	if err != nil {
		return "", fmt.Errorf("检查工作目录所有权数据目录失败: %w", err)
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("工作目录所有权数据目录不安全")
	}
	if created {
		if err := secureAndValidatePrivateDirectory(metadataPath, info); err != nil {
			_ = os.Remove(metadataPath)
			return "", fmt.Errorf("工作目录所有权数据目录不安全: %w", err)
		}
	} else if err := validatePrivateDirectory(metadataPath, info); err != nil {
		return "", fmt.Errorf("工作目录所有权数据目录不安全: %w", err)
	}
	return metadataPath, nil
}

func ensureDirectoryOwnershipCapability(metadataPath string) error {
	probePath, err := os.MkdirTemp(metadataPath, ".capability-*")
	if err != nil {
		return fmt.Errorf("创建所有权能力探测目录失败: %w", err)
	}
	defer os.RemoveAll(probePath)

	const probeToken = "0000000000000000000000000000000000000000000000000000000000000000"
	if err := writeDirectoryOwnershipToken(probePath, probeToken); err != nil {
		return fmt.Errorf("写入所有权标记失败: %w", err)
	}
	actualToken, found, err := directoryOwnershipToken(probePath)
	if err != nil {
		return fmt.Errorf("读取所有权标记失败: %w", err)
	}
	if !found || actualToken != probeToken {
		return fmt.Errorf("所有权标记读写校验失败")
	}
	return nil
}

func ownershipMetadataPath(root string) (string, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("解析任务工作区根目录失败: %w", err)
	}
	return filepath.Join(filepath.Clean(absRoot), ownershipMetadataDirectory), nil
}

func validatedOwnershipMetadataDirectory(root string) (string, bool, error) {
	if err := validateOwnershipRoot(root); err != nil {
		return "", false, fmt.Errorf("任务工作区根目录不安全: %w", err)
	}
	metadataPath, err := ownershipMetadataPath(root)
	if err != nil {
		return "", false, err
	}
	info, err := os.Lstat(metadataPath)
	if os.IsNotExist(err) {
		return metadataPath, false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("检查工作目录所有权数据目录失败: %w", err)
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return "", false, fmt.Errorf("工作目录所有权数据目录不安全")
	}
	if err := validatePrivateDirectory(metadataPath, info); err != nil {
		return "", false, fmt.Errorf("工作目录所有权数据目录不安全: %w", err)
	}
	return metadataPath, true, nil
}

func ownershipArtifactPaths(metadataPath, token string) (string, string, string, error) {
	if err := validateOwnershipToken(token); err != nil {
		return "", "", "", err
	}
	return filepath.Join(metadataPath, token+".json"), filepath.Join(metadataPath, token+".staging"), filepath.Join(metadataPath, token+".deleting"), nil
}

func validateOwnershipToken(token string) error {
	if len(token) != 64 {
		return fmt.Errorf("工作目录所有权令牌无效")
	}
	if _, err := hex.DecodeString(token); err != nil {
		return fmt.Errorf("工作目录所有权令牌无效")
	}
	return nil
}

func writeOwnershipClaim(path string, claim ownershipClaim) error {
	contents, err := json.Marshal(claim)
	if err != nil {
		return fmt.Errorf("编码工作目录所有权凭据失败: %w", err)
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), ".claim-*.tmp")
	if err != nil {
		return fmt.Errorf("创建工作目录所有权临时凭据失败: %w", err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(0o600); err != nil {
		temporary.Close()
		return fmt.Errorf("设置工作目录所有权凭据权限失败: %w", err)
	}
	if _, err := temporary.Write(contents); err != nil {
		temporary.Close()
		return fmt.Errorf("写入工作目录所有权凭据失败: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return fmt.Errorf("同步工作目录所有权凭据失败: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("关闭工作目录所有权凭据失败: %w", err)
	}
	if err := renameNoReplace(temporaryPath, path); err != nil {
		return fmt.Errorf("保存工作目录所有权凭据失败: %w", err)
	}
	return nil
}

func readOwnershipClaim(path, taskID, token string) (ownershipClaim, bool, error) {
	claim, found, err := readOwnershipClaimFile(path)
	if err != nil || !found {
		return ownershipClaim{}, found, err
	}
	if claim.TaskID != taskID || claim.Token != token || strings.TrimSpace(claim.Identity) == "" {
		return ownershipClaim{}, false, fmt.Errorf("工作目录所有权凭据无效")
	}
	return claim, true, nil
}

func readOwnershipClaimFile(path string) (ownershipClaim, bool, error) {
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return ownershipClaim{}, false, nil
	}
	if err != nil {
		return ownershipClaim{}, false, fmt.Errorf("检查工作目录所有权凭据失败: %w", err)
	}
	if !info.Mode().IsRegular() {
		return ownershipClaim{}, false, fmt.Errorf("工作目录所有权凭据不安全")
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		return ownershipClaim{}, false, fmt.Errorf("读取工作目录所有权凭据失败: %w", err)
	}
	var claim ownershipClaim
	if err := json.Unmarshal(contents, &claim); err != nil {
		return ownershipClaim{}, false, fmt.Errorf("解析工作目录所有权凭据失败: %w", err)
	}
	return claim, true, nil
}

func claimedDirectoryPath(workspacePath, stagingPath string, claim ownershipClaim) (string, error) {
	workspaceMatches, err := directoryMatchesClaim(workspacePath, claim)
	if err != nil {
		return "", fmt.Errorf("验证任务工作目录所有权失败: %w", err)
	}
	if workspaceMatches {
		return workspacePath, nil
	}
	stagingMatches, err := directoryMatchesClaim(stagingPath, claim)
	if err != nil {
		return "", fmt.Errorf("验证待创建工作目录所有权失败: %w", err)
	}
	if stagingMatches {
		return stagingPath, nil
	}
	return "", nil
}

func directoryMatchesClaim(path string, claim ownershipClaim) (bool, error) {
	identity, err := directoryIdentity(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	if identity != claim.Identity {
		return false, nil
	}
	token, found, err := directoryOwnershipToken(path)
	if err != nil {
		return false, err
	}
	return found && token == claim.Token, nil
}

func findOwnershipClaimToken(root, taskID string) (string, bool, error) {
	metadataPath, exists, err := validatedOwnershipMetadataDirectory(root)
	if err != nil {
		return "", false, err
	}
	if !exists {
		return "", false, nil
	}
	entries, err := os.ReadDir(metadataPath)
	if err != nil {
		return "", false, fmt.Errorf("读取工作目录所有权数据失败: %w", err)
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		token := strings.TrimSuffix(entry.Name(), ".json")
		if validateOwnershipToken(token) != nil {
			continue
		}
		claim, found, readErr := readOwnershipClaimFile(filepath.Join(metadataPath, entry.Name()))
		if readErr != nil {
			return "", false, readErr
		}
		if !found {
			continue
		}
		if claim.Token != token || strings.TrimSpace(claim.Identity) == "" {
			return "", false, fmt.Errorf("工作目录所有权凭据无效")
		}
		if claim.TaskID == taskID {
			return token, true, nil
		}
	}
	return "", false, nil
}

func removeUnclaimedStaging(path string) error {
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("检查残留待创建工作目录失败: %w", err)
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("残留待创建工作目录不安全")
	}
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("清理残留待创建工作目录失败: %w", err)
	}
	return nil
}

func cleanupOwnershipArtifacts(claimPath, stagingPath string) error {
	if err := os.RemoveAll(stagingPath); err != nil {
		return err
	}
	if err := os.Remove(claimPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func pathExistsChecked(path string) (bool, error) {
	_, err := os.Lstat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
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
