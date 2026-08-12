# 终端备注功能 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 让用户为活动终端选区添加运行期备注，在标题栏汇总查看，并按可配置模板模拟粘贴回同一终端。

**Architecture:** Go 设置模型持久化备注模板并迁移旧数据；前端用纯函数格式化记录，`App` 以终端会话键拥有仅运行期的备注状态，`TerminalView` 负责选区入口、输入框、面板和同会话发送。终端关闭、退出、任务结束或应用卸载时，`App` 清除相应内存记录。

**Tech Stack:** Go、React 18、TypeScript、xterm.js、Radix UI、Vitest、Wails。

---

### Task 1: 持久化备注模板设置

**Files:**

- Modify: `internal/settings/settings.go`
- Test: `internal/settings/settings_test.go`
- Modify: `internal/storage/repository.go`
- Test: `internal/storage/repository_test.go`
- Modify: `frontend/src/types.ts`
- Modify: `frontend/src/api.ts`
- Modify: `frontend/wailsjs/go/models.ts`

**Step 1: 为默认值、空字段和旧设置迁移写失败测试**

在 `internal/settings/settings_test.go` 添加测试，断言默认模板为：

```go
TerminalNoteTemplate{
    OriginalPrefix: "原文：",
    NotePrefix:     "备注：",
    ListSuffix:     "",
}
```

在 `internal/storage/repository_test.go` 添加一个缺少 `terminalNoteTemplate` 的 JSON 数据文件夹具，断言 `Load()` 返回默认模板并将模板回写；再添加保存三个空字符串后重新加载的测试，断言三个空字符串不被替换为默认值。

**Step 2: 运行测试确认失败**

Run: `go test ./internal/settings ./internal/storage -run 'TerminalNoteTemplate|TerminalNotes'`

Expected: FAIL，提示缺少模板类型、字段或迁移行为。

**Step 3: 添加模型和迁移逻辑**

在 `internal/settings/settings.go` 定义模板类型和默认构造函数，并把字段加入 `Settings` 与 `Default()`：

```go
type TerminalNoteTemplate struct {
    OriginalPrefix string `json:"originalPrefix"`
    NotePrefix     string `json:"notePrefix"`
    ListSuffix     string `json:"listSuffix"`
}

func DefaultTerminalNoteTemplate() TerminalNoteTemplate {
    return TerminalNoteTemplate{OriginalPrefix: "原文：", NotePrefix: "备注："}
}
```

不要在 `Validate()` 中把零值模板重置为默认值，否则用户显式保存的三个空字符串会丢失。`repository.Load()` 解析原始设置 JSON 后，仅当整个 `terminalNoteTemplate` 键缺失时，填充默认模板并让现有归一化回写路径保存它。

在 `frontend/src/types.ts` 添加同名的 `TerminalNoteTemplate` 接口和 `SettingsRecord.terminalNoteTemplate`；在 `frontend/src/api.ts` 保证 `saveSettings` 的 Wails payload 携带该字段。执行 `wails generate module` 后接受生成的 `frontend/wailsjs/go/models.ts` 字段，不手工伪造生成绑定。

**Step 4: 运行目标测试**

Run: `go test ./internal/settings ./internal/storage -run 'TerminalNoteTemplate|TerminalNotes'`

Expected: PASS。

**Step 5: 提交设置模型变更**

```bash
git add internal/settings/settings.go internal/settings/settings_test.go internal/storage/repository.go internal/storage/repository_test.go frontend/src/types.ts frontend/src/api.ts frontend/wailsjs/go/models.ts
git commit -m "feat: persist terminal note templates"
```

### Task 2: 建立运行期备注模型和确定的汇总格式

**Files:**

- Create: `frontend/src/terminal-notes.ts`
- Test: `frontend/src/terminal-notes.test.ts`
- Modify: `frontend/src/types.ts`

**Step 1: 编写格式化和会话隔离失败测试**

在 `frontend/src/terminal-notes.test.ts` 测试两个记录、空列表统一后缀和不同会话键。核心输出断言为：

```ts
expect(formatTerminalNotes([
  {original: '一', note: '甲'},
  {original: '二', note: '乙'},
], {originalPrefix: '原文：', notePrefix: '备注：', listSuffix: '请处理'}))
  .toBe('原文：一\n备注：甲\n原文：二\n备注：乙\n请处理\n')
```

另测空统一后缀得到记录文本后的两个换行位置，且辅助函数只清除指定 `taskId + terminalId` 的记录。

**Step 2: 运行测试确认失败**

Run: `cd frontend && npm test -- terminal-notes.test.ts`

Expected: FAIL，提示模块或导出不存在。

**Step 3: 实现纯模型和格式化函数**

在新文件中定义运行期类型和会话状态辅助函数：

```ts
export interface TerminalNote {
  original: string
  note: string
}

export function formatTerminalNotes(notes: TerminalNote[], template: TerminalNoteTemplate): string {
  return `${notes.map(({original, note}) => `${template.originalPrefix}${original}\n${template.notePrefix}${note}`).join('\n')}\n${template.listSuffix}\n`
}
```

只对非空列表调用格式化函数；模板和记录文本按原样保留。使用现有 `terminalEventKey(taskId, terminalId)` 作为状态映射的键，不新增第二种键格式。

**Step 4: 运行测试确认通过**

Run: `cd frontend && npm test -- terminal-notes.test.ts`

Expected: PASS。

**Step 5: 提交运行期模型**

```bash
git add frontend/src/terminal-notes.ts frontend/src/terminal-notes.test.ts frontend/src/types.ts
git commit -m "feat: add terminal note formatting"
```

### Task 3: 扩展终端会话读取选区并实现备注交互

**Files:**

- Modify: `frontend/src/terminal-session.ts`
- Test: `frontend/src/terminal-session.test.ts`
- Modify: `frontend/src/components/TerminalView.tsx`
- Test: `frontend/src/components/TerminalView.test.tsx`

**Step 1: 为会话选区读取写失败测试**

在 `frontend/src/terminal-session.test.ts` 让 xterm mock 返回选区文本，断言注册表的新公开方法只返回指定活动会话的选择；不存在或已经关闭的会话返回空字符串。

**Step 2: 实现受限选区读取方法并验证**

在 `TerminalSessionRegistry` 添加方法：

```ts
selectionText(taskID: string, terminalID: string): string {
  const session = this.sessions.get(terminalSessionKey(taskID, terminalID))
  if (!session || this.closedTerminalKeys.has(terminalSessionKey(taskID, terminalID))) return ''
  return session.terminal.getSelection()
}
```

Run: `cd frontend && npm test -- terminal-session.test.ts`

Expected: PASS。

**Step 3: 为浮动入口、输入和发送写失败组件测试**

扩展 xterm mock，使其能设置选择文本并触发终端内容区的 `pointerup`。新增用例覆盖：

- 非空选择在下一动画帧显示“添加备注”按钮，空选择不显示。
- 点击按钮显示带原文和多行备注框的对话框；空白备注不可保存，保存后只调用当前终端的追加回调并使焦点回到 xterm。
- `disableTaskAIMouseClipboard: true` 时，即使 mock 有选择文本也不显示入口。
- 标题栏入口显示数量、面板按顺序显示记录；发送使用 `registry.pasteInput` 的等价通道，传入格式化文本，并在成功与关闭失败两种情况下都调用清空回调。

**Step 4: 实现 `TerminalView` 备注状态和 UI**

扩展 props，以只读记录和显式状态回调连接 `App`：

```ts
notes?: TerminalNote[]
noteTemplate?: TerminalNoteTemplate
onAddNote?(note: TerminalNote): void
onClearNotes?(): void
```

在内容容器的 `onPointerUp` 中保存相对坐标，并用 `requestAnimationFrame` 调用 `selectionText`。坐标用 `getBoundingClientRect()` 和按钮尺寸限制在容器内。选择为空、终端失活、模板输入框关闭以及点击其他位置时重置浮动状态。

复用 `Dialog`、`Popover`、`Textarea` 和 `IconButton`：浮动按钮用消息图标和 `opacity-70` 样式；标题栏按钮的 `aria-label` 形如 `终端备注（2 条）`。发送动作按以下顺序执行，避免异常分支遗漏清空：

```ts
const content = formatTerminalNotes(notes, noteTemplate)
const wrote = sessionRegistry.pasteInput(terminal.taskId, terminal.id, content)
onClearNotes?.()
if (!wrote) onErrorRef.current?.(new Error('终端已关闭，无法发送备注'))
```

发送动作不得追加 Enter；成功与失败后都关闭面板，并只聚焦当前终端。

**Step 5: 运行组件和会话测试**

Run: `cd frontend && npm test -- terminal-session.test.ts src/components/TerminalView.test.tsx`

Expected: PASS。

**Step 6: 提交终端交互**

```bash
git add frontend/src/terminal-session.ts frontend/src/terminal-session.test.ts frontend/src/components/TerminalView.tsx frontend/src/components/TerminalView.test.tsx
git commit -m "feat: add terminal selection notes"
```

### Task 4: 在 App 中维护备注生命周期并传递给终端

**Files:**

- Modify: `frontend/src/App.tsx`
- Test: `frontend/src/App.test.tsx`
- Modify: `frontend/src/types.ts`

**Step 1: 为终端生命周期写失败 App 测试**

扩展 `TerminalView` mock，暴露其接收的备注数量、追加回调和清空回调。测试以下路径：

- 为 A 添加备注后，切换到 B 时 B 收到空列表，切回 A 时仍收到 A 的记录。
- 终端 `exited` 事件、手动关闭、结束任务、删除任务和组件卸载都会清除对应会话记录。
- 备注模板从加载设置传给当前终端；缺失旧前端设置时使用前端默认模板。

**Step 2: 运行 App 测试确认失败**

Run: `cd frontend && npm test -- src/App.test.tsx`

Expected: FAIL，提示备注 props、状态或清理行为缺失。

**Step 3: 添加状态所有权和释放路径**

在 `App` 中新增按 `terminalEventKey` 索引的状态。新增记录时用不可变更新追加；清除单个会话、任务下全部会话和全部记录分别用专用辅助函数，避免在各个回调中复制筛选逻辑。

在终端事件订阅收到任意 `exited` 事件时清除该键；同时在 `closeTerminal` 成功后、`confirmFinishTask`、`confirmDeleteTasks` 和主 effect 的卸载清理中释放记录。将当前终端的列表、当前设置的模板、新增和清空回调传入 `TerminalView`。不得把这些记录并入 `SettingsRecord`、任务或 API 调用。

**Step 4: 运行 App 测试确认通过**

Run: `cd frontend && npm test -- src/App.test.tsx`

Expected: PASS。

**Step 5: 提交生命周期管理**

```bash
git add frontend/src/App.tsx frontend/src/App.test.tsx frontend/src/types.ts
git commit -m "feat: scope terminal notes to terminal sessions"
```

### Task 5: 在设置草稿中编辑备注模板

**Files:**

- Modify: `frontend/src/App.tsx`
- Test: `frontend/src/App.test.tsx`
- Modify: `frontend/src/types.ts`
- Modify: `frontend/wailsjs/go/models.ts`

**Step 1: 为设置编辑和取消写失败测试**

在 `frontend/src/App.test.tsx` 打开设置中的“终端备注”分类，修改三个字段后保存并断言 `SaveSettings` 收到完整 `terminalNoteTemplate`；再修改后取消，重新打开设置并断言显示的是原有模板。

**Step 2: 实现设置分类和草稿控件**

将 `settingsTab` 联合类型与标签列表添加 `notes` 值和“终端备注”标签。在其内容中使用三个 `Field` 与 `Textarea`，并通过既有 `updateSettingsDraft` 更新嵌套模板：

```tsx
onChange={(event) => updateSettingsDraft({
  terminalNoteTemplate: {...terminalNoteTemplateDraft, originalPrefix: event.target.value},
})}
```

提供前端 `defaultTerminalNoteTemplate`，只在服务器返回旧的缺失字段时用于展示和发送；用户正在编辑的空字符串不触发默认值回填。沿用现有保存和取消路径，不新增独立保存按钮。

**Step 3: 运行设置测试确认通过**

Run: `cd frontend && npm test -- src/App.test.tsx`

Expected: PASS。

**Step 4: 重新生成并检查 Wails 绑定**

Run: `wails generate module && git diff -- frontend/wailsjs/`

Expected: `settings.TerminalNoteTemplate` 及其 TypeScript 创建逻辑出现，且没有无关绑定改动。

**Step 5: 提交设置界面**

```bash
git add frontend/src/App.tsx frontend/src/App.test.tsx frontend/src/types.ts frontend/wailsjs/go/models.ts
git commit -m "feat: configure terminal note templates"
```

### Task 6: 同步规格、端到端验证和交付

**Files:**

- Modify: `openspec/changes/terminal-notes/tasks.md`
- Modify: `openspec/changes/terminal-notes/specs/terminal-notes/spec.md`
- Modify: `openspec/changes/terminal-notes/specs/terminal-mouse-clipboard-policy/spec.md`
- Modify: `docs/plans/2026-08-12-terminal-notes-design.md`

**Step 1: 运行受影响测试和构建**

Run:

```bash
go test ./internal/settings ./internal/storage
cd frontend && npm test -- terminal-notes.test.ts terminal-session.test.ts src/components/TerminalView.test.tsx src/App.test.tsx && npm run build
cd .. && go test -race ./...
```

Expected: 全部通过；前端构建在 Go 全量测试前生成 `frontend/dist`。

**Step 2: 使用 Wails 开发模式做集成验证**

Run: `wails dev`

Expected: 应用保持运行并输出开发地址。用浏览器工具验证：修改并保存模板；在普通终端中选择文本、添加两条备注、查看数量、发送且确认汇总进入同一终端并清空；关闭一个有备注的终端后确认记录消失；在禁用 TaskAI 鼠标剪贴板的菜单终端中确认不出现备注入口。

发现问题时先补充相应的失败测试，再回到相关任务最小修复并重新运行该测试与集成验证。

**Step 3: 编译并打开可执行程序**

Run: 使用 `scripts/` 中适用于当前平台的编译脚本，然后启动生成的应用程序。

Expected: 应用启动且终端备注功能可用。完成此步后等待用户确认，未确认前不把 worktree 分支合并回当前工作区分支。

**Step 4: 确认后整合、归档和清理**

用户确认后，按项目流程将当前工作区变更合并到本 worktree 分支，解决冲突并再次构建；再将 `feat/terminal-notes` 合并到当前工作区目标分支。运行完整验证，更新本任务清单和最终文档，归档 OpenSpec 变更，提交 Git 变更，并移除已合并的 worktree。
