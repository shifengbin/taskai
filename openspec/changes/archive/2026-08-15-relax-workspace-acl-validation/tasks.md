## 1. Windows 工作区校验调整

- [x] 1.1 修改 `internal/workspace/ownership_windows.go` 的 `validateOwnershipRoot`：保留"存在、普通目录、非符号链接"检查，删除所有者 SID 比对和 DACL 中其他 SID 写入权限扫描，移除不再使用的 `windows.OpenCurrentProcessToken` 等相关代码
- [x] 1.2 修改 `createPrivateDirectory`：改为不带安全描述符的普通目录创建（`os.Mkdir`），删除 `SecurityDescriptorFromString` 与 `SecurityAttributes` 相关代码
- [x] 1.3 修改 `secureAndValidatePrivateDirectory` 与 `validatePrivateDirectory`：删除 `SetNamedSecurityInfo` 硬化、DACL AceCount 与单一当前用户 ACE 校验、所有者比对，仅保留"是普通目录、非符号链接"的形状校验
- [x] 1.4 清理 `ownership_windows.go` 中因此不再使用的导入（`unsafe`、`golang.org/x/sys/windows` 中仅剩令牌备用数据流所需部分）

## 2. Unix 工作区校验调整

- [x] 2.1 修改 `internal/workspace/ownership_unix.go` 的 `validateOwnershipRoot`：保留目录形状检查，删除 uid 比对、组写入位和其他用户写入位检查
- [x] 2.2 修改 `secureAndValidatePrivateDirectory` 与 `validatePrivateDirectory`：删除 `0700` 收紧（chmod）、uid 比对和 `0o077` 权限位校验，仅保留目录形状校验；`createPrivateDirectory` 改为不带强制权限的普通创建
- [x] 2.3 确认两平台的令牌读写、探测和删除校验路径未被改动：`setDirectoryOwnershipToken`、`directoryOwnershipToken` 及 `workspace.go` 中五重删除校验保持原样

## 3. 测试更新

- [x] 3.1 删除或改写 `internal/workspace` 中所有 ACL 拒绝类测试用例（所有者不是当前用户、目录访问控制列表不安全、允许其他用户写入/访问、权限收紧为 0700 等），改为断言宽松 ACL/权限目录被接受
- [x] 3.2 新增 Windows 回归测试：向临时工作区根目录添加 `icacls <root> /grant "Users:(OI)(CI)M"` 式的宽松 ACE 后，`Create` 任务目录成功且令牌探测通过
- [x] 3.3 新增 Unix 回归测试（`GOOS=linux` 或本机执行）：工作区根目录权限为 `0777` 时任务目录创建成功；元数据目录权限为 `0755` 时直接复用
- [x] 3.4 运行 `go test -race ./...` 与 `cd frontend && npm test && npm run build` 全量验证
- [x] 3.5 验证既有严格 ACL 元数据目录（旧版本创建）在新逻辑下直接通过校验

## 4. 集成测试（wails dev + MCP Chrome DevTools）

- [x] 4.1 准备复现环境：将应用设置的工作区根目录指向 `%APPDATA%\taskai\workspaces`（本机已带 `CodexSandboxUsers` 与孤儿 SID 的继承 ACE），用 `Get-Acl` 记录修改前的拒绝事实；若无继承 ACE，用 `icacls <root> /grant "Users:(OI)(CI)M"` 手工添加
- [x] 4.2 以 `wails dev` 启动应用（不禁用终端颜色、不自动退出），从输出获取调试地址，用 MCP Chrome DevTools 打开
- [x] 4.3 创建一个 `beforeStart` 链为"创建任务工作目录"的未执行任务并启动：确认任务进入执行中、目录创建成功、无"任务工作区根目录不安全"错误
- [x] 4.4 完成该任务（`postEnd` 链为"删除任务工作目录"），确认受控清理仍成功：目录被删除、任务进入已完成、无权限相关失败
- [x] 4.5 删除一个开始前失败任务的用例（可选，复用现有失败链配置）：确认令牌与文件系统身份校验仍按既有行为拒绝被替换的目录（由既有单元测试 TestRemoveOwnedRejectsReplacedSameNameDirectory 与 TestRemoveOwnedRejectsRecreatedSameNameDirectoryAfterOriginalDeletion 覆盖并通过）
- [x] 4.6 测试完成后关闭调试进程

## 5. 文档同步

- [x] 5.1 更新 AGENTS.md 变更边界：将"保留既有的路径校验"表述同步为"保留路径边界、所有权令牌与文件系统身份校验"，并记录单人使用个人电脑的威胁模型前提
- [x] 5.2 在 `docs/plans/` 归档本次设计记录（含威胁模型降级决策），确认与最终实现一致
