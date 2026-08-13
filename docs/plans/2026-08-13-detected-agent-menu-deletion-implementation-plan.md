# 自动代理菜单删除记忆实施计划

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 让删除过的自动代理菜单不再补充，并在已有精确命令配置时跳过自动项。

**Architecture:** 删除选择保存在仓储根数据，不加入公开设置类型。设置保存负责记录自动稳定 ID 的删除，启动仓储合并同时依据删除记录、稳定 ID 和精确命令字段决定是否追加。

**Tech Stack:** Go、JSON 仓储、Wails、React、Vitest、OpenSpec。

---

### Task 1: 合并规则

**Files:**
- Modify: `internal/settings/agent_task_menu.go`
- Test: `internal/settings/agent_task_menu_test.go`

1. 先添加失败测试：删除记录抑制自动项；已有命令型菜单的 `command` 精确为 `codex` 或 `claude` 时不追加；相似名称、不同命令和非命令类型不误判。
2. 运行 `go test ./internal/settings -run 'TestMergeDetectedAgentTaskMenuItems'`，确认测试因缺少新规则失败。
3. 扩展合并输入并实现稳定 ID、删除记录与精确命令判断。
4. 重跑定向测试确认通过。

### Task 2: 仓储删除记忆

**Files:**
- Modify: `internal/storage/repository.go`
- Test: `internal/storage/repository_test.go`

1. 先添加失败测试：删除自动项后记录稳定 ID并跨仓储重载保留；删除自定义项不记录；启动合并尊重删除记录。
2. 运行对应仓储测试，确认因缺少字段和记录逻辑失败。
3. 增加根数据字段、规范化函数和 `SaveSettings` 删除差异记录，并把记录传给启动合并。
4. 重跑仓储与应用定向测试确认通过。

### Task 3: 前端保存回归

**Files:**
- Modify: `frontend/src/App.test.tsx`

1. 添加测试：在菜单设置中编辑并删除自动 `codex` 项，保存请求只移除该项并保留 `claude` 与其他菜单。
2. 运行 `cd frontend && npm test -- --run src/App.test.tsx`，确认用例通过现有通用删除链路。
3. 若测试暴露问题，仅修正删除保存链路，不新增界面字段。

### Task 4: 规格与完整验证

**Files:**
- Modify: `openspec/changes/add-detected-agent-task-menus/design.md`
- Modify: `openspec/changes/add-detected-agent-task-menus/specs/detected-agent-task-menus/spec.md`
- Modify: `openspec/changes/add-detected-agent-task-menus/tasks.md`

1. 同步删除记忆、精确命令去重和浏览器集成测试场景。
2. 运行 `go test -race ./...`、`cd frontend && npm test -- --run && npm run build`、Windows 目标交叉编译和 `openspec validate add-detected-agent-task-menus --strict`。
3. 使用测试 PATH 和 `wails dev -tags webkit2_41`，在 Chrome DevTools 删除自动 `codex`、保存并重启，确认不恢复且 `claude` 与用户菜单保留。
4. 使用 `scripts/build-linux.sh` 重新编译并打开程序，等待用户确认。

提交、合并、归档与 worktree 清理由项目总流程在用户最终确认后统一执行。
