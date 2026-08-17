# 放宽工作区 ACL 校验设计记录

- 日期：2026-08-15
- 变更：`openspec/changes/relax-workspace-acl-validation/`

## 背景与问题

Windows 上创建任务工作目录命令报错"任务工作区根目录不安全: 目录允许其他用户写入"，`beforeStart` 命令链失败导致任务无法启动。根因是 OpenAI Codex 沙箱在用户 `AppData` 上添加的 `CodexSandboxUsers` 组与已删除账户孤儿 SID 的 Modify ACE，随继承传递到 TaskAI 默认工作区根目录 `%APPDATA%\taskai\workspaces`；旧版 `validateOwnershipRoot` 扫描 DACL 时把这两类 ACE 判定为安全威胁。

## 威胁模型降级决策

用户确认：TaskAI 支持场景固定为单人使用的个人电脑。工作区根目录和所有权元数据目录不再要求"仅当前用户私有"，只要求是普通目录、非符号链接且软件进程可访问；同时移除软件主动改写用户目录安全描述符或权限位的硬化行为。

该决策的安全性依据：删除任务目录前的五重校验（任务 ID、路径边界、随机令牌、目录自身所有权标记、文件系统身份）全部是应用层逻辑，不依赖目录 ACL。即使其他本地账户读到侧车凭据中的令牌，也无法伪造文件系统身份，替换目录仍会被拒绝删除。已知接受的风险：共享环境下其他本地账户可读写任务工作区内容。

## 实现要点

- `internal/workspace/ownership_windows.go`：`validateOwnershipRoot` 收敛为目录形状检查；`createPrivateDirectory` 改为普通 `os.Mkdir`；删除 `secureAndValidatePrivateDirectory`、`validatePrivateDirectory` 及所有 DACL 读写代码。
- `internal/workspace/ownership_unix.go`：对称删除 uid、组写入位、其他用户写入位、`0700` 收紧逻辑；保留 xattr 令牌函数与形状检查。
- `internal/workspace/workspace.go`：`ensureOwnershipMetadataDirectory` 与 `validatedOwnershipMetadataDirectory` 不再调用平台私有目录校验，仅保留"是普通目录、非符号链接"检查。
- 令牌读写（Windows 备用数据流 `:taskai.workspace-token` / Unix xattr `user.taskai.workspace-token`）、能力探测和删除校验路径未做任何修改。

## 验证

- 单元测试：宽松权限元数据目录被接受且不被改写；Unix `0777` 根目录创建成功；Windows 通过 `icacls /grant "Users:(OI)(CI)M"` 添加宽松 ACE 后创建成功。
- 集成测试：`wails dev` + MCP Chrome DevTools，在带 `CodexSandboxUsers` 继承 ACE 的默认根目录上启动任务，确认目录创建、任务进入执行中、结束删除流程正常。
- 回归：`go test -race ./...`、前端测试与构建、Darwin/Linux 交叉编译。
