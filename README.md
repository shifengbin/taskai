# 任务工作台

一个将任务生命周期和独立嵌入式终端结合在一起的 Wails 桌面应用。

## 功能

- 任务包含必填标题、可选描述和可选颜色，并按“未执行 / 执行中 / 已完成”管理；左侧以三个状态标签页分类展示并记住上次选择，已完成任务不可再次执行。
- 新建任务可使用颜色选择器设置颜色，任务列表会以该颜色标记对应任务。
- 顶部“额外信息管理”采用“分类即模板”的方式：先定义模板的固定字段、默认值和动态参数，再按模板填写并保存可复用的信息；每个模板都自动包含固定 `name` 字段，信息列表和任务选择器均以其值展示。动态参数支持文本和复选框两种输入类型：复选框默认未选中，在应用持久化快照中保存为字符串 `true` 或 `false`，且不提供必填设置。信息还可新增自身的动态参数，任务选择信息后会带入模板参数及信息参数，只显示显示名称和可编辑值，且仍可在任务内新增动态参数。模板区域可整体折叠，信息按模板折叠分组，搜索仅过滤信息名称并保留所有分类。任务保存完整快照，因此信息或模板之后被删除仍可保留并编辑任务级动态参数。内置 Git 模板包含“项目名称”（`name`）、“仓库地址”（`repository`）和动态“仓库分支”（`branch`）；其固定键和显示名称不可修改，但可设置默认值、调整分支必填状态并添加字段或参数。
- 开始任务时创建独立工作目录并保存目录快照；结束任务经确认后关闭该任务全部终端并安全删除目录。
- 执行中的任务可从右键菜单或“任务操作”下拉菜单新增多个终端。终端作为任务树子节点显示，并在右侧使用 xterm 交互；选中的非空终端文本会自动复制到系统剪贴板，右键可直接粘贴系统剪贴板的非空文本；程序可通过 OSC 0/2 标题序列实时设置终端名称，未设置时显示“终端”；双击任务条目可展开或收起终端，任务可在同一状态标签页内拖动排序并持久化。
- 终端和任务提供会话级实时状态：绿色表示工作中、紫色表示未读、灰色表示空闲、红色表示异常。工作中和未读状态点会以扩散动画提示；任务收起时显示按“异常 > 未读 > 工作中 > 空闲”汇总的状态点；展开时仅显示各终端状态。主动关闭的终端会从界面移除，异常退出的终端保留红色状态点。
- 任务操作菜单可在设置中排序；固定操作仅可调序，自定义操作可配置名称、命令、逐行参数、是否显示为独立终端，以及可选的前置、后置脚本。自定义命令和脚本都会通过已选择 Shell 的初始化环境、在任务工作目录中启动，因此可直接使用该 Shell `PATH` 中的 `codex` 或脚本；未显示终端的命令会在任务工作目录中后台启动，例如 `code .`。
- 每个前置、后置脚本分别配置 `script`（脚本路径或 Shell `PATH` 中的可执行脚本）和 `arguments`；参数每行一个，空白行忽略。不会进行占位符替换、字符串拼接，也不会把上下文 JSON 追加到命令行。脚本会从 UTF-8 标准输入接收主命令上下文：`{"taskId":"任务 ID","directory":"任务工作目录","command":"主命令","arguments":["主命令参数"]}`。前置脚本成功后才会启动主命令；主命令实际退出后才会运行后置脚本，无论主命令是否成功退出。前置失败会显示错误并停止主命令，后置失败只显示错误；任务进入结束流程后，尚未触发的后置脚本会跳过。
- 工作区根目录、任务树宽度、亮色/暗色模式、终端 Shell 路径、任务操作菜单、实时状态方式和本机 HTTP 服务开关可持久化设置；设置使用“工作区与外观 / 终端 Shell / 菜单管理 / 实时状态”四个 Tab。新增或编辑自定义菜单项会在独立弹窗中完成，其中“前后置脚本”Tab 的“？”入口说明标准输入 JSON 字段和参数传递规则；确认主设置前不会写入磁盘。设置界面会探测当前平台可用的 Shell 供下拉选择，也支持手动填写有效路径。修改根目录只影响后续任务。
- 退出时若有执行中任务会请求确认，仅关闭 PTY，会保留任务状态和工作目录。

## 实时状态与 HTTP 接口

实时状态只保留在当前应用会话，重启后从空闲状态重新开始。设置中的“实时状态”提供两种方式：

- **根据终端标题变化（默认）**：当终端通过 OSC 0/2 更新为不同标题时，终端立即进入工作中；连续 1.5 秒没有新的标题变化后，当前选中的终端变为空闲，未选中的终端变为未读。点击终端会将它的未读状态清为空闲。
- **通过 HTTP 接口**：需要填写 1–65535 的端口，并会自动启用本机 HTTP 服务。服务只监听 `127.0.0.1`，不提供鉴权；可直接更新任务或终端状态。任务的直接更新是临时覆盖，下一次终端状态更新会重新按终端状态汇总。

也可以在标题变化模式下独立启用本机 HTTP 服务，以查询任务、终端和状态数据；当独立开关关闭且状态管理不使用 HTTP 时，服务会停止。

**新创建**的普通终端和“显示终端”的自定义命令始终会注入：

```text
TASKAI_TASK_ID=<任务 ID>
TASKAI_TERMINAL_ID=<终端 ID>
```

使用 HTTP 状态管理方式时，终端还会额外注入：

```text
TASKAI_STATUS_API=http://127.0.0.1:<端口>/api/v1
```

无终端后台自定义命令和前置、后置脚本仅注入 `TASKAI_TASK_ID`。

切换状态方式或端口不会修改已经运行的进程环境变量；请新建终端后再使用新的配置。

### 查询与更新

状态枚举为 `idle`、`working`、`unread`、`error`。查询接口：

```bash
curl "$TASKAI_STATUS_API/status"
```

示例响应：

```json
{
  "tasks": [
    {
      "taskId": "任务 ID",
      "title": "任务名称",
      "lifecycleStatus": "running",
      "status": "working",
      "terminals": [
        {"terminalId": "终端 ID", "status": "working"}
      ]
    }
  ]
}
```

`/status` 的 `status` 参数可按任务生命周期筛选，合法取值为 `pending`、`running`、`completed`。响应中的 `lifecycleStatus` 是任务生命周期，`status` 仍是实时终端状态：

```bash
curl "$TASKAI_STATUS_API/status?status=completed"
```

任务查询接口不依赖状态管理方式。`status` 参数可省略；合法取值仅为 `pending`、`running`、`completed`。省略时返回全部任务，指定时只返回对应生命周期的任务：

```bash
curl 'http://127.0.0.1:<端口>/api/v1/tasks?status=pending'
curl 'http://127.0.0.1:<端口>/api/v1/tasks?status=running'
curl 'http://127.0.0.1:<端口>/api/v1/tasks?status=completed'
curl 'http://127.0.0.1:<端口>/api/v1/tasks/<任务 ID>'
```

列表返回任务数组；详情返回单个任务，包含 `id`、`title`、`description`、`color`、`status`、创建/完成时间以及工作区根目录和任务工作目录。详情额外包含 `extraInfo`：它按 `catalogue` 聚合为对象键，每项将全部固定字段的键和值及参数键、参数值平铺为对象；复选框参数返回 JSON 布尔值 `true` 或 `false`，固定字段和文本参数返回字符串。详情还包含 `terminals`：每项返回活动终端的 `id`、执行 `command` 和实时 `status`。普通终端的命令为 Shell 路径，自定义命令终端使用其启动命令；状态为 `idle`、`working`、`unread` 或 `error`。没有活动终端时该字段为 `[]`。列表接口不返回 `extraInfo` 或 `terminals`。无效 `status` 返回 `400`，不存在的任务返回 `404`。

例如任务选择两个 Git 信息后，详情中的附加信息为：

```json
{
  "extraInfo": {
    "git": [
      {"name": "API 服务", "repository": "git@example.com:team/api.git", "branch": "main"},
      {"name": "Web 服务", "repository": "git@example.com:team/web.git", "branch": "release/1.0"}
    ]
  }
}
```

`extraInfo` 返回固定字段 `name` 作为信息名称，但不返回字段显示名称、参数显示名称或元素内独立的 `catalogue` 字段；任务未选择附加信息时返回空对象 `{}`。复选框的 JSON 布尔输出仅适用于任务详情 HTTP 接口，应用内部快照仍以字符串保存以兼容既有数据。

更新当前终端会自动重新汇总对应任务：

```bash
curl -X PUT "$TASKAI_STATUS_API/tasks/$TASKAI_TASK_ID/terminals/$TASKAI_TERMINAL_ID/status" \
  -H 'Content-Type: application/json' \
  --data '{"status":"working"}'
```

也可以直接临时更新任务状态：

```bash
curl -X PUT "$TASKAI_STATUS_API/tasks/$TASKAI_TASK_ID/status" \
  -H 'Content-Type: application/json' \
  --data '{"status":"unread"}'
```

两个更新接口的请求体都要求 `status` 字段，合法取值仅为 `idle`、`working`、`unread`、`error`，例如 `{"status":"working"}`；成功响应包含 `taskId`、`taskStatus`，终端更新还包含 `terminalId`、`terminalStatus`。无效 JSON 或状态返回 `400`，不存在的任务或终端返回 `404`，已结束任务或已关闭终端返回 `409`，错误方法返回 `405`；错误响应为 `{"error":"..."}`。

## 开发

```bash
wails dev
```

前端单独运行：

```bash
cd frontend
npm run dev
```

## 验证

```bash
go test -race ./...
cd frontend && npm test && npm run build
```

## 构建发行版

Wails 的 Linux 与 macOS 图形依赖不支持在其他操作系统上稳定交叉编译，因此请在目标系统上运行对应脚本。构建产物统一位于 `build/bin`。

Linux（默认 `amd64`，可选 `arm64` 或 `arm`）：

```bash
chmod +x scripts/build-linux.sh
./scripts/build-linux.sh
./scripts/build-linux.sh arm64
```

Windows PowerShell（默认 `amd64`，可选 `arm64` 或 `386`；传入 `-NSIS` 时生成安装程序）：

```powershell
.\scripts\build-windows.ps1
.\scripts\build-windows.ps1 -Architecture arm64
.\scripts\build-windows.ps1 -NSIS
```

macOS（默认构建 Universal 二进制；可选 `amd64` 或 `arm64`）：

```bash
chmod +x scripts/build-macos.sh
./scripts/build-macos.sh
./scripts/build-macos.sh arm64
```

## 平台依赖

- 类 Unix 平台使用 `creack/pty`。
- Windows 使用 ConPTY，需要 Windows 10 1809 / Windows Server 2019 或更高版本。
- Linux 打包或运行 Wails 需要 GTK 3 和 WebKitGTK 开发包。脚本优先使用 `libwebkit2gtk-4.1-dev`（传递 `webkit2_41` 标签），也兼容 `libwebkit2gtk-4.0-dev`。
- Windows 构建需要 Wails CLI、Go 和 C/C++ 编译工具链；使用 `-NSIS` 还需要安装 NSIS。
- macOS 构建需要 Wails CLI、Go 与 Xcode 命令行工具。
