# 任务工作台

一个将任务生命周期和独立嵌入式终端结合在一起的 Wails 桌面应用。

## 功能

- 任务包含必填标题、可选描述和可选颜色，并按“未执行 / 执行中 / 已完成”管理；左侧以三个状态标签页分类展示，已完成任务不可再次执行。
- 新建任务可使用颜色选择器设置颜色，任务列表会以该颜色标记对应任务。
- 开始任务时创建独立工作目录并保存目录快照；结束任务经确认后关闭该任务全部终端并安全删除目录。
- 执行中的任务可从右键菜单新增多个终端。终端作为任务树子节点显示，并在右侧使用 xterm 交互。
- 工作区根目录、任务树宽度、亮色/暗色模式和终端 Shell 路径可持久化设置；设置界面会探测当前平台可用的 Shell 供下拉选择，也支持手动填写有效路径。修改根目录只影响后续任务。
- 退出时若有执行中任务会请求确认，仅关闭 PTY，会保留任务状态和工作目录。

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
