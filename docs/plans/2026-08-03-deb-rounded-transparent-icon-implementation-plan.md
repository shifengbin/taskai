# DEB 圆角透明图标实施计划

> **For Codex:** REQUIRED SUB-SKILL: Use `superpowers:executing-plans` to implement this plan task-by-task.

**目标：** 让 Linux DEB 中安装的 TaskAI 图标保留圆角外侧的透明区域，而不改变既有图案、发布接口或 Windows/Linux 共用图标来源。

**架构：** `frontend/src/assets/task-ai-mark.svg` 是当前视觉稿；将其以 1024px RGBA PNG 导出到 `build/appicon.png`。现有 Linux 打包脚本继续原样复制该 PNG，Windows 则继续由同一 PNG 生成 ICO。测试从 DEB 解包出的 PNG 读取左上角 alpha，覆盖发布产物而非只检查源文件。

**技术栈：** SVG、PNG RGBA、FFmpeg、Bash、`dpkg-deb`、`tar`。

---

### 任务 1：为发布图标加入透明通道回归测试

**文件：**
- 修改：`scripts/build-linux.test.sh`
- 验证：`bash scripts/build-linux.test.sh`

**步骤 1：编写失败测试**

在现有 DEB 内容断言后，读取 `build/appicon.png` 及 DEB 中的 `taskai.png`。使用 `ffprobe` 验证像素格式含 alpha，并以 FFmpeg 裁切左上角一个像素，断言 RGBA 的 alpha 字节为 `0`。

**步骤 2：确认测试失败**

运行 `bash scripts/build-linux.test.sh`。预期在源 PNG 为 `rgb24` 或左上角 alpha 不为零的断言失败。

### 任务 2：从现有 SVG 重新导出 RGBA 发布图标

**文件：**
- 修改：`build/appicon.png`
- 来源：`frontend/src/assets/task-ai-mark.svg`

**步骤 1：导出图标**

将 SVG 以 1024px RGBA PNG 渲染到 `build/appicon.png`，不添加背景层；SVG 圆角矩形外侧保持透明。

**步骤 2：确认测试通过**

运行 `bash scripts/build-linux.test.sh`。预期源图和 DEB 中图标均为带 alpha 的 PNG，且左上角透明；现有二进制、桌面文件、符号链接和控制信息断言继续通过。

### 任务 3：验证真实发布产物并同步变更记录

**文件：**
- 修改：`openspec/changes/sync-windows-icon-and-add-deb-package/tasks.md`
- 验证：`build/bin/taskai_<版本>_amd64.deb`

**步骤 1：构建并检查**

运行 `bash scripts/build-linux.sh --deb --version 0.0.0+icon-alpha-test`。从生成的 DEB 中读取 `taskai.png`，确认像素格式带 alpha、左上角 alpha 为零，并以 `dpkg-deb --contents` 保留安装布局检查。

**步骤 2：全量验证**

运行 `go test -race ./...`、`cd frontend && npm test -- --run`、`cd frontend && npm run build`、`openspec validate sync-windows-icon-and-add-deb-package --strict --no-interactive` 与 `git diff --check`。

**步骤 3：更新任务状态**

将完成的 OpenSpec 任务标为 `[x]`；Windows 主机上的 EXE/NSIS 运行验证继续保留为独立待办。
