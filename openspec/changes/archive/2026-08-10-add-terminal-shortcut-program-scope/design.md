## Context

终端快捷键（capability `terminal-shortcut-inputs`）在当前聚焦终端拦截已配置的组合键，把预设的文本/按键步骤写入 PTY。拦截只发生在唯一一处闸门：`TerminalView.tsx` 中通过 `sessionRegistry.setCustomKeyEventHandler` 注册的 xterm `attachCustomKeyEventHandler` 回调——返回 `false` 表示由 TaskAI 消费（写入步骤），返回 `true` 表示透传（xterm 按原行为把按键发给 PTY）。

问题：codex 这类交互式 TUI 接管终端输入，原始组合键（如 `Shift+Enter` 换行）与其在 TaskAI 里的映射动作冲突，且 TUI 程序是无界集合（codex、vim、less、htop、lazygit……）。

关键事实：后端 `managedSession`（`internal/terminal/manager.go`）在创建终端时已经把启动命令记到了 `managedSession.command`（无命令时退化为 shell 路径），并经 `ListActive → ActiveSession{ID, Command}` 暴露过一次。但创建终端的返回值 `Info`（`internal/terminal/types.go`）与前端 `TerminalRecord`（`frontend/src/types.ts`）都**没有**这个字段，所以键盘闸门目前拿不到它。

## Goals / Non-Goals

**Goals:**

- 让每条终端快捷键可按"生效程序"收窄作用范围，使 TUI 程序自动获得原始按键、无需枚举无界黑名单。
- 复用后端已有的启动命令信息，零运行时检测成本。
- 零数据迁移、完全向后兼容。

**Non-Goals:**

- 不检测终端运行时的前台进程（不做 OS 级前台进程查询、不解析标题判断当前程序）。
- 不跟踪用户在 shell 内手动运行的子程序——按明确的产品取舍，这类终端按其启动命令（shell）判定。
- 不引入全局"透传模式"开关（可作为后续独立增强）。
- 不改变 `Ctrl+Shift+P` 快捷输入选择器的优先级与行为。

## Decisions

### 决策 1：以"启动命令"作为终端所属程序，而非运行时检测

终端的"所属程序"在其被 TaskAI 创建的那一刻静态确定，取自创建时使用的启动命令（任务菜单命令项、生命周期命令链创建的命令终端）。后端 `managedSession.command` 已记录该值，本变更只需把它经 `Info` 透传到前端 `TerminalRecord`。

**备选与取舍：**

- *运行时前台进程查询*：最符合"现在跑的是谁"的直觉，但本应用以 Windows 为主，ConPTY 把 PTY 托管在 sidecar 进程中，映射回逻辑前台进程代价高、跨平台差异大；macOS 还有 SIP/权限限制。**否决**。
- *终端标题（OSC 0/2）匹配*：`terminal-title.ts` 已有完整解析器，成本低，但依赖程序确实设置了标题、且 shell prompt 不会覆写，脆弱。**否决，留作后续增强**。
- *手动透传开关*：100% 确定但不自动。**否决为本次目标，留作后续**。

启动命令静态打标彻底绕开了"运行时检测"这一最贵的子问题。

### 决策 2：include（包含/白名单）语义，空列表 = 全部终端

每条快捷键携带可选的 `includePrograms: string[]`。空/缺失表示对所有终端生效（等价于现状）；非空时仅在该终端的启动命令命中列表时才拦截执行，否则透传。

**备选与取舍：**

- *exclude（黑名单）语义*：要"在所有 TUI 里让位"需要枚举无界 TUI 集合；而 shell 是有限且固定的几个（powershell/pwsh/bash/zsh/fish/cmd）。白名单把"无界黑名单"换成"有限白名单"，一次填写即可让任意 TUI 自动让位，包括将来新出现的程序。**采用 include**。
- *空列表 = 哪都不生效（严格模式）*：会破坏向后兼容（老快捷键突然失效），且新建快捷键必须先填列表才能用。**否决**；采用"空 = 全部"的标准可选过滤器语义。

### 决策 3：归一化后精确匹配，而非子串/正则

匹配规则：取启动命令与列表每一项的 basename，去除 Windows 可执行扩展名（`.exe`、`.com`），忽略大小写，然后**精确相等**比较；列表中任一项相等即命中。

**备选与取舍：**

- *子串匹配*：曾作为宽容默认考虑，但有真实误伤——例如列表里的 `sh` 会命中 `powershell`、`vim` 会命中 `gvim`，反而让快捷键在意外的终端里触发，违背本特性初衷。**否决**。
- *glob/正则*：表达力强但增加配置心智负担与校验复杂度，当前需求不需要。**留作后续可选项**。

精确匹配可预测、可测试，且 basename+去扩展名归一化已能覆盖 `codex` / `codex.exe` / `C:\tools\codex.exe` / `/usr/local/bin/codex` 这些常见形态。

### 决策 4：沿用既有"终端属性经 Info 透传"模式扩展 command

`Info` 增加 `Command` 字段，在 `createWithEnvironmentBuilder` 构造 `Info` 时从 `managedSession.command` 带出（与现有 `DisableTaskAIMouseClipboard` 走 `StartRequest → Info → TerminalRecord` 完全同构，是该路径的现成先例）。前端 `TerminalRecord` 增加 `command?`，在终端记录构造处复制该字段。`TerminalView` 的键盘闸门读取 `terminal.command` 做生效范围判定。

因 `command` 在终端生命周期内不变，把它加入 `customKeyEventHandler` 所在 `useEffect` 的依赖数组即可（或用 ref 读取最新值），无需额外的实时同步机制。

## Risks / Trade-offs

- **[手动运行子程序不被识别]** → 明确的产品取舍：用户在 shell 内手动运行 codex 时，快捷键仍会触发。用户已确认接受（"自己在 shell 里执行的不算"）。要享受排除，用菜单命令直接创建该程序终端。
- **[包裹式启动命令不命中]** → 若菜单命令写作 `bash -c "codex ..."`，后端记录的启动命令是 `bash` 而非 `codex`，不会命中 `codex`。缓解：建议菜单命令直接调用目标程序；文档说明此限制。
- **[旧版本创建的终端缺 command]** → 由旧版本创建、`Info` 未带 `command` 的终端，前端 `TerminalRecord.command` 为空，按"不匹配任何非空生效列表"处理（空列表=全部的情形不受影响）。属可接受的降级，重启后新建终端即带该字段。
- **[include 列表误填导致快捷键"消失"]** → 用户若把生效程序填错（如拼错 shell 名），快捷键会在本应生效的终端里不触发。缓解：编辑器标注"留空=全部终端生效"，并可在终端头部展示其所属程序以便核对（见 Open Questions）。

## Migration Plan

- 所有新增字段（`TerminalShortcut.IncludePrograms`、`Info.Command`、`TerminalRecord.command`）均为可选；Go 端 JSON 标签用 `omitempty`。
- 既有设置缺少 `includePrograms` → 加载为空 → 等价"全部终端生效"，行为不变。
- 既有终端记录缺少 `command` → 视为不匹配非空列表；空列表场景不受影响。
- 无需数据迁移脚本；需重新生成 `frontend/wailsjs/` 绑定以反映 `Info` 新字段。
- 回滚：移除字段即恢复原状，无持久化副作用。

## Open Questions

- 是否在终端头部（`TerminalView` 标题区）显示该终端的所属程序（归一化后的 command），让用户直观核对生效范围配置？非必需，可后续补。
- 未来是否抽离"可复用程序组"供多条快捷键共享引用，避免重复填写？当前每条快捷键独立列表已足够，留作后续优化。
