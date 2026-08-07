# Nebula UI 重构实施计划

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 在不改变 TaskAI 业务行为和交互语义的前提下，将前端完整换肤为 `design-preview` 的 03 Nebula 指挥中枢主题。

**Architecture:** 保留现有 React/Wails 组件树、状态流和 Radix 原语，以 CSS 变量和基础 UI 组件类承载 Nebula 亮暗主题。只有尺寸、字体、颜色、边框、圆角、动效和 xterm 视觉主题发生变化；任务树、终端和覆盖层继续调用现有回调与 API。

**Tech Stack:** React 18、TypeScript、Tailwind CSS、Radix UI、lucide-react、xterm.js、Vite、Wails。

---

### Task 1: 建立 Nebula 视觉契约测试

**Files:**
- Modify: `frontend/tailwind.config.test.ts`
- Test: `frontend/src/terminal-session.test.ts`
- Test: `frontend/src/components/TaskTree.test.tsx`

**Step 1: 写失败测试**

- 将 Tailwind 配置断言改为 Nebula 的字体族、圆角和表面令牌。
- 为 `terminalVisualTheme('light'|'dark')` 增加背景、前景、光标和关键 ANSI 色值断言。
- 保留任务卡 60px、两行文本、状态点和上下文操作的现有语义断言，并补充选中态不改变内容结构的断言。

**Step 2: 运行受影响测试确认失败**

Run: `cd frontend && npm test -- tailwind.config.test.ts src/terminal-session.test.ts src/components/TaskTree.test.tsx`

Expected: 现有 Snap 字体/色值断言失败，行为断言仍通过。

**Step 3: 提交测试契约**

```bash
git add frontend/tailwind.config.test.ts frontend/src/terminal-session.test.ts frontend/src/components/TaskTree.test.tsx
git commit -m "test(ui): define Nebula visual contracts"
```

### Task 2: 更新字体入口与主题令牌

**Files:**
- Modify: `frontend/package.json`
- Modify: `frontend/package-lock.json`
- Modify: `frontend/src/main.tsx`
- Modify: `frontend/tailwind.config.cjs`
- Modify: `frontend/src/style.css`

**Step 1: 引入本地字体资源**

- 添加 Sora 与 Chakra Petch 的本地可变字体资源；JetBrains Mono 继续复用现有资源。
- 在 `main.tsx` 只导入本地字体入口，确保运行时不访问外部 CDN。

**Step 2: 替换 CSS 令牌**

- 以 Nebula 亮暗色值替换 `--snap-*` 的实际值，保留令牌名称以避免业务 JSX 大范围改动。
- 设置 `font-sans` 为 Sora、`font-display` 为 Chakra Petch、`font-mono` 为 JetBrains Mono。
- 将焦点环、状态脉冲、背景网点和动效令牌改为青色/紫色体系。

**Step 3: 运行契约测试**

Run: `cd frontend && npm test -- tailwind.config.test.ts`

Expected: PASS。

**Step 4: 提交字体与令牌**

```bash
git add frontend/package.json frontend/package-lock.json frontend/src/main.tsx frontend/tailwind.config.cjs frontend/src/style.css
git commit -m "feat(ui): add Nebula typography and theme tokens"
```

### Task 3: 重塑应用外壳和任务树布局

**Files:**
- Modify: `frontend/src/App.css`
- Modify: `frontend/src/App.tsx`
- Modify: `frontend/src/components/TaskTree.tsx`
- Test: `frontend/src/App.test.tsx`
- Test: `frontend/src/components/TaskTree.test.tsx`

**Step 1: 写布局回归断言**

- 断言顶栏为 52px、侧栏标题为 40px，右侧终端标题由视觉类控制为 40px。
- 断言任务卡仍固定 60px，任务标题、描述、状态点和操作按钮都存在。

**Step 2: 实现应用外壳皮肤**

- 将页面画布、顶栏、侧栏、分隔条和内容区改为 Nebula 玻璃层、细边框和冷蓝背景。
- 不添加 `LIVE` 指示，不删除任何现有顶栏入口。

**Step 3: 实现任务树皮肤**

- 保留所有现有事件处理和 DOM 语义，只调整任务/终端行的圆角、边框、内嵌任务色条、悬停、选中、拖拽预览和上下文操作可见性。
- 将 Tabs 激活态改为青色底部指示线，状态点和生命周期 Chip 使用 Nebula 语义色。

**Step 4: 运行任务树与应用测试**

Run: `cd frontend && npm test -- src/App.test.tsx src/components/TaskTree.test.tsx`

Expected: PASS，所有既有交互断言保持通过。

**Step 5: 提交应用外壳**

```bash
git add frontend/src/App.css frontend/src/App.tsx frontend/src/components/TaskTree.tsx frontend/src/App.test.tsx frontend/src/components/TaskTree.test.tsx
git commit -m "feat(ui): skin workbench and task tree with Nebula"
```

### Task 4: 统一基础 UI 覆盖层

**Files:**
- Modify: `frontend/src/components/ui/button.tsx`
- Modify: `frontend/src/components/ui/icon-button.tsx`
- Modify: `frontend/src/components/ui/dialog.tsx`
- Modify: `frontend/src/components/ui/dropdown-menu.tsx`
- Modify: `frontend/src/components/ui/popover.tsx`
- Modify: `frontend/src/components/ui/tabs.tsx`
- Modify: `frontend/src/components/ui/accordion.tsx`
- Modify: `frontend/src/components/ui/alert.tsx`
- Modify: `frontend/src/components/ui/chip.tsx`
- Modify: `frontend/src/components/ui/text-field.tsx`
- Modify: `frontend/src/components/ui/checkbox.tsx`
- Modify: `frontend/src/components/ui/switch.tsx`
- Modify: `frontend/src/components/ui/tooltip.tsx`
- Modify: `frontend/src/components/ui/toast.tsx`
- Test: `frontend/src/components/ui/dialog.test.tsx`

**Step 1: 写基础控件视觉断言**

- 保留 Radix 的 role、Portal、焦点管理和键盘测试。
- 增加按钮、Dialog 和 Tabs 使用 Nebula 令牌类的断言，不断言业务文案或触发逻辑变化。

**Step 2: 更新基础控件样式**

- 统一半透明表面、细边框、8-10px 圆角、焦点环、禁用态和语义色。
- 菜单、Popover、Tooltip 和 Toast 使用同一层级的背景与边框；长内容继续在内部滚动。
- Lucide 图标描边统一为 Nebula 细线几何风格。

**Step 3: 运行基础控件测试**

Run: `cd frontend && npm test -- src/components/ui/dialog.test.tsx`

Expected: PASS。

**Step 4: 提交覆盖层样式**

```bash
git add frontend/src/components/ui frontend/src/components/ui/dialog.test.tsx
git commit -m "feat(ui): unify Nebula overlay components"
```

### Task 5: 更新终端视觉连续性

**Files:**
- Modify: `frontend/src/terminal-session.ts`
- Modify: `frontend/src/components/TerminalView.tsx`
- Modify: `frontend/src/components/TerminalStatusDot.tsx`
- Modify: `frontend/src/components/TerminalView.test.tsx`
- Modify: `frontend/src/components/TerminalStatusDot.test.tsx`
- Modify: `frontend/src/style.css`

**Step 1: 写终端主题断言**

- 断言亮暗主题的 xterm 背景、前景、光标、选区和 ANSI 关键色与 Nebula 令牌一致。
- 保留右侧快捷输入始终可见、左侧状态点和关闭入口可用的测试。

**Step 2: 更新终端皮肤**

- 只替换 xterm 主题、终端标题栏、快捷输入 Popover 和网点背景；不改变会话注册、输入输出、尺寸调整或文件拖放逻辑。
- 将状态点颜色映射为运行中工程绿、未读紫色、异常红色、空闲蓝灰。

**Step 3: 运行终端测试**

Run: `cd frontend && npm test -- src/terminal-session.test.ts src/components/TerminalView.test.tsx src/components/TerminalStatusDot.test.tsx`

Expected: PASS。

**Step 4: 提交终端样式**

```bash
git add frontend/src/terminal-session.ts frontend/src/components/TerminalView.tsx frontend/src/components/TerminalStatusDot.tsx frontend/src/style.css frontend/src/components/TerminalView.test.tsx frontend/src/components/TerminalStatusDot.test.tsx
git commit -m "feat(ui): align terminal visuals with Nebula"
```

### Task 6: 覆盖层视觉回归与无障碍检查

**Files:**
- Test: `frontend/src/App.test.tsx`
- Test: `frontend/src/components/TaskTree.test.tsx`
- Test: `frontend/src/components/TerminalView.test.tsx`
- Test: `frontend/src/components/ui/dialog.test.tsx`

**Step 1: 运行完整前端测试**

Run: `cd frontend && npm test`

Expected: PASS。

**Step 2: 运行前端生产构建**

Run: `cd frontend && npm run build`

Expected: TypeScript 和 Vite 构建成功，产物生成在 `frontend/dist`。

**Step 3: 做亮暗主题人工检查**

- 检查顶栏、任务树、详情、终端、任务编辑、设置、额外信息、模板、命令链、Popover、Dropdown、Tooltip 和 Toast。
- 检查 Portal 内容没有旧 Snap 色值、粗描边或硬阴影。
- 检查 720px 最小宽度、侧栏拖拽、内部滚动、键盘焦点和减少动态效果偏好。

**Step 4: 提交验证调整**

```bash
git add frontend
git commit -m "test(ui): verify Nebula visual regression boundaries"
```

### Task 7: 合并 worktree 并完成项目编译

**Files:**
- Modify: 无业务文件；仅合并提交和生成构建产物

**Step 1: 确认 worktree 分支状态**

Run: `git -C .worktrees/holo-ui-design status --short`

Expected: 只有本变更文件和已知的用户未跟踪 HOLO 设计文档；不得删除或覆盖该文档。

**Step 2: 将当前分支合并到 worktree 分支**

Run: `git -C .worktrees/holo-ui-design merge main`

Expected: 无需合并冲突；若 main 有新提交，先完成合并再继续。

**Step 3: 在 worktree 分支完成最终测试与编译**

Run: `go test -race ./...` 以及 `cd frontend && npm test && npm run build`

Expected: 全部通过。

**Step 4: 将 worktree 分支合并回当前分支**

Run: `git merge --no-ff holo-ui-design`

Expected: 当前分支包含 Nebula UI 提交，用户未跟踪的 HOLO 文档不被纳入合并。

**Step 5: 按项目脚本完成编译**

Run: `./scripts/build-linux.sh`

Expected: Wails Linux 构建完成并生成 `build/bin/taskai`；若当前环境缺少 GTK/WebKitGTK 或 Wails CLI，记录阻塞原因而不修改构建脚本。
