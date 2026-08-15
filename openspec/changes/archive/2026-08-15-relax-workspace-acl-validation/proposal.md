## Why

在 Windows 上，创建任务工作目录命令会因"任务工作区根目录不安全: 目录允许其他用户写入"而失败。典型触发环境：开发工具（如 OpenAI Codex 的沙箱）会在用户 `AppData` 上添加 `CodexSandboxUsers` 等组账户的写入 ACE，这些 ACE 顺继承传递到 TaskAI 默认工作区根目录 `%APPDATA%\taskai\workspaces`；此外已删除账户的孤儿 SID 在开发机上也很常见。当前校验把这些都视为安全威胁并拒绝启动任务，但 TaskAI 面向的是单人使用的个人电脑，"其他本地账户"通常是工具的隔离账户而非攻击者。删除安全性实际由随机令牌、目录自身标记和文件系统身份比对等逻辑校验支撑，与目录 ACL 无关，可以独立成立。

## What Changes

- 移除工作区根目录的 ACL 严格校验：不再要求所有者为当前用户，不再扫描 DACL 中其他 SID 的写入权限（Windows），不再检查所有者、组写入位和其他用户写入位（Unix）。根目录仅要求是普通目录、非符号链接，且软件可访问。
- 移除私有元数据目录的 ACL 严格校验与主动硬化：创建时不再附加仅当前用户的 PROTECTED DACL（Windows）或 `0700` 权限（Unix），不再重写既有目录的安全描述符或收紧权限，不再校验 DACL 只含单一当前用户 ACE 或权限位为 `0700`。元数据目录按普通目录创建，仅要求是普通目录、非符号链接。
- 保留全部与 ACL 无关的数据安全逻辑：随机所有权令牌（Windows 备用数据流 / Unix 扩展属性）的读写探测、删除任务目录前的任务 ID + 路径边界 + 令牌 + 目录自身标记 + 文件系统身份五重校验、路径穿越和符号链接边界检查，均不改变。
- 保留文件系统不支持备用数据流 / 扩展属性时的既有拒绝行为（不降级为仅凭路径或目录身份推断归属）。
- **BREAKING**（威胁模型降级）：共享根目录下其他本地账户将可以读取和修改任务工作区内容。这是用户明确接受的决策，换取在带有沙箱账户 ACE 的环境中正常使用。

## Capabilities

### New Capabilities

（无）

### Modified Capabilities

- `task-lifecycle-command-chains`: 内置目录命令不再因工作区根目录或元数据目录的 ACL/所有者广泛而拒绝；创建和复用目录仅要求普通目录、非符号链接且软件可访问。
- `lifecycle-command-execution-progress`: 开始前执行记录的令牌侧车凭据不再要求存放在私有（仅当前用户可访问）目录中；令牌能力探测、目录归属状态和文件系统身份记录保持不变。
- `completed-task-batch-deletion`: 删除校验继续要求随机令牌匹配侧车凭据、目录自身所有权标记和文件系统身份，但不再以侧车凭据目录的私有 ACL 为前提。
- `macos-workspace-build-compatibility`: 移除"Unix 私有目录继续校验所有者和普通权限"要求中关于 `0700`、所有者和组/其他用户权限的强制校验，改为普通目录、非符号链接和软件可访问；保留不依赖扩展 ACL cgo 的构建边界和分层集成测试要求。

## Impact

- `internal/workspace/ownership_windows.go`：`validateOwnershipRoot` 删除所有者与 DACL 扫描；`createPrivateDirectory` 改为普通创建；`secureAndValidatePrivateDirectory` 与 `validatePrivateDirectory` 删除 DACL 校验与 `SetNamedSecurityInfo` 硬化。
- `internal/workspace/ownership_unix.go`：`validateOwnershipRoot` 删除 uid / 组写入 / 其他用户写入检查；`secureAndValidatePrivateDirectory` 与 `validatePrivateDirectory` 删除 `0700` 收紧与所有者校验。
- `internal/workspace/workspace.go`：相关错误信息与调用路径随之调整，令牌探测和删除校验路径不变。
- 两平台测试：删除 ACL 拒绝类用例，新增"继承宽松 ACL 的根目录可正常创建任务目录"用例。
- 文档：AGENTS.md 变更边界中"保留既有的路径校验"表述需同步为"保留路径边界、令牌与文件系统身份校验"。
