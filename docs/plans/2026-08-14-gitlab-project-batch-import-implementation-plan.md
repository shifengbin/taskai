# GitLab 项目批量导入实施计划

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**目标：** 从私有 GitLab 完整读取当前令牌可见项目，明文记住上次成功连接，支持本地筛选和按当前结果批量选择，并把所选项目原子导入内置 Git 分类。

**架构：** `internal/gitlab` 负责 REST 通信、分页、连接默认值校验和仓库身份归一化；`internal/storage` 在单次锁和原子文件替换中保存专用明文默认连接及批量导入结果；`app.go` 把项目读取与默认连接保存拆成独立 Wails 边界；React 弹窗每次打开读取默认连接，只为仍有效的当前项目响应触发保存，并独立维护筛选和选择状态。

**技术栈：** Go 标准库、`httptest`、Wails v2、React 18、TypeScript、Vitest、Testing Library。

---

## 基线

- `npm ci`：完成，未修改依赖版本；审计报告为仓库现有依赖风险。
- `cd frontend && npm test`：16 个测试文件、336 个测试通过。
- 新 worktree 首次 `go test ./...`：因 `//go:embed all:frontend/dist` 缺少构建目录而失败。
- `cd frontend && npm run build` 后重新执行 `go test ./...`：全部 Go 包通过。

## 实施任务

### 任务一：GitLab 客户端与仓库身份

**文件：** 新增 `internal/gitlab/client.go`、`internal/gitlab/client_test.go`、`internal/gitlab/repository.go`、`internal/gitlab/repository_test.go`。

1. 先用 `httptest` 编写地址、身份、分页、重定向、状态码和脱敏失败测试并确认 RED。
2. 使用 `net/http` 实现最小客户端并确认 GREEN。
3. 先写 SSH/HTTPS/SCP 地址归一化失败测试，再实现主机和项目路径身份。
4. 运行 `go test ./internal/gitlab/...`。

### 任务二：存储事务和应用边界

**文件：** 修改 `internal/storage/repository.go`、`internal/storage/repository_test.go`、`app.go`、`app_test.go`。

1. 先写批量模板构造、跨协议去重、并发复查和保存失败回滚测试并确认 RED。
2. 在一次 `mutationMu` 临界区中加载、校验、追加并保存一次。
3. 先写应用方法的连接、脱敏和导入测试，再接入 GitLab 客户端与存储方法。
4. 运行聚焦 Go 测试并生成 Wails 绑定。

### 任务三：前端导入体验

**文件：** 修改 `frontend/src/types.ts`、`frontend/src/api.ts`、`frontend/src/api.test.ts`、`frontend/src/App.tsx`、`frontend/src/App.test.tsx`。

1. 先写 API 类型转换测试并确认 RED。
2. 先写表单、HTTP 警告、筛选、当前结果选择、归档/重复、地址模式、错误重试和成功刷新测试并确认 RED。
3. 在现有信息管理器中实现克制的工具型双阶段弹窗，不改变手动新增流程。
4. 先写完整仓库路径辅助展示测试，再实现纯派生展示。
5. 运行相关 Vitest 测试。

### 任务四：集成环境与交付验证

**文件：** 新增 `scripts/gitlab-mock-server/main.go`、`scripts/gitlab-mock-server/main_test.go` 和说明文档；同步 OpenSpec 任务记录。

1. 先写模拟服务认证、两页项目和第二页失败测试，再实现服务。
2. 运行 `wails dev`，从输出取得调试地址并使用 `mcp chrome-devtools` 完成 OpenSpec 中的浏览器步骤。
3. 关闭调试进程后运行 `go test -race ./...`、`cd frontend && npm test && npm run build`。
4. 合入当前目标分支、重复验证、执行 `./scripts/build-linux.sh amd64` 并打开应用等待确认。
5. 用户确认后合并回目标分支，完成归档、提交和 worktree 清理。

### 任务五：明文记住上次成功连接

**文件：** 修改 `internal/gitlab/client.go`、`internal/storage/repository.go`、`internal/storage/repository_test.go`、`app.go`、`gitlab_import_test.go`、`frontend/src/types.ts`、`frontend/src/api.ts`、`frontend/src/api.test.ts`、`frontend/src/components/GitLabImportDialog.tsx`、`frontend/src/App.test.tsx` 和 Wails 生成绑定。

1. 先写存储层失败测试，确认专用默认连接尚不存在；实现校验、明文读取、成功覆盖和保存失败保留旧值，再运行聚焦 Go 测试。
2. 先写应用层失败测试，确认项目读取不隐式保存默认连接、显式保存失败不覆盖旧值；实现专用读取与不回传令牌的保存方法，再运行聚焦 Go 测试。
3. 先写 API 与弹窗失败测试，覆盖打开回填、明文提示、关闭重读、失败不覆盖及 SSH 默认值；实现最小前端流程并运行聚焦 Vitest。
4. 重新生成 Wails 绑定，运行 `wails dev` 浏览器测试并通过重开弹窗、重启应用和失败连接验证持久化边界。
5. 增加晚到读取响应不保存、默认连接保存期间禁止关闭、导入期间完整锁定及失败恢复的审查回归测试。
6. 重复全量竞态测试、前端测试与构建、主线合入、发行构建和人工验收。

审查后的浏览器复核使用隔离 XDG 配置，并在浏览器控制台临时包装页面中的 Wails 绑定：分别延迟项目读取、默认连接保存和批量导入。读取延迟时确认输入禁用但允许取消，关闭后等待响应并重开，确认旧连接没有保存；保存延迟时确认取消和 Escape 无效；导入延迟并拒绝时确认所有相关控件禁用，失败后筛选、地址格式和选择完整恢复。随后恢复真实绑定完成导入，使用同一配置重启 `wails dev`，确认默认连接仍能回填。包装只增加等待或一次受控拒绝，不修改真实请求参数和持久化结果；测试完成后关闭页面、`wails dev` 和模拟服务。

## 初始无默认连接版本的浏览器集成测试记录

- 时间：2026-08-14；平台：Linux；启动命令：`go run ./scripts/gitlab-mock-server --listen 127.0.0.1:18080` 与 `XDG_CONFIG_HOME=/tmp/taskai-gitlab-import-integration wails dev -tags webkit2_41`。
- 调试地址：`http://localhost:34115`；GitLab 地址：`http://127.0.0.1:18080`；测试身份：`integration-user` / `integration-token`。
- 预置数据：手动创建 `team/existing`；模拟服务分两页返回 `team-a/api`、`team/existing`、`public/docs`、`team-b/api`、`archive/legacy`、`platform/worker`。
- 连接结果：明文 HTTP 警告在请求前出现；两页共 6 个项目完整显示；初始选择为 0；`team/existing` 禁用；`archive/legacy` 显示归档标签；两个 `api` 以完整路径区分。
- 筛选结果：`TEAM-A` 不区分大小写命中 `team-a/api`；全选后改筛选为 `team-b` 保留原选择；全选和取消当前结果只增减 `team-b/api`，总选择数依次为 1、2、1。
- 导入结果：清空筛选后显式选择 `team-b/api` 与 `archive/legacy`；HTTPS 预览覆盖全部项目并正确切回 SSH；最终以 SSH 导入 3 个项目，跳过 0 个重复项目。
- 刷新结果：Git 分类自动展开，展示 `team-a/api`、`team-b/api`、`archive/legacy`；任务选择器分别提供 `api team-a/api` 与 `api team-b/api`，主名称仍为 `api`。
- 凭据结果：该轮发生在默认连接记忆功能加入前，成功关闭后重新打开时三项输入为空；此行为已由下方“默认连接明文持久化浏览器复核”的自动回填结果替代。
- 失败模式：用 `--fail-page 2` 重启模拟服务后，第二页返回 HTTP 500；弹窗保留凭据并显示可重试错误，且不展示第一页的任何部分结果。
- 控制台：没有本功能脚本异常；唯一错误为仓库现有的 `favicon.ico` 404。测试结束后已关闭浏览器测试页、`wails dev` 和模拟服务。

### 合并前审查修复复核

- 审查后把模拟 GitLab 地址调整为 `http://127.0.0.1:18080/private/gitlab`，HTTP clone 地址保留 `/private/gitlab` 前缀，SSH clone 地址使用独立 `2424` 端口。
- 再次以隔离配置启动 `wails dev`，调试地址仍为 `http://localhost:34115`；在信息管理中先手工保存 `ssh://git@127.0.0.1:2424/team/existing.git`。
- 两页 6 项完整加载，跨协议的 `team/existing` 正确标记为已导入；切换 HTTPS 后预览地址包含相对部署路径。
- 以 HTTPS 导入 `team-a/api` 成功；重新连接时，列表同时把手工 SSH 项目和刚导入的相对路径 HTTP 项目标记为已导入，可选数从 5 降为 4。
- 浏览器控制台无错误；复核结束后已关闭测试页、`wails dev` 与模拟服务。读取关闭竞态、导入期间禁止关闭、HTTPS 降级、分页 `null`/跳页由自动化测试覆盖。

### 合入目标分支后的回归验证

- 将目标分支 `main` 快进合入功能分支，解决 API 与 Wails 生成绑定冲突后执行 `wails generate module`，生成结果同时包含 GitLab 导入与自动更新接口。
- 聚焦 Go 测试 `go test ./internal/gitlab ./internal/storage . ./scripts/gitlab-mock-server` 通过；聚焦前端测试 3 个文件、153 个用例通过。
- 全量 `go test -race ./...` 通过；前端 18 个测试文件、360 个用例通过；`npm run build` 成功完成生产构建。
- `./scripts/build-linux.sh amd64` 成功生成 Linux amd64 可执行程序；以隔离配置启动 `build/bin/taskai`，进程保持运行并等待人工界面验收。

### 默认连接明文持久化浏览器复核

- 时间：2026-08-14；隔离配置：`/tmp/taskai-gitlab-defaults-integration.tVlT8D`；启动命令：`go run ./scripts/gitlab-mock-server --listen 127.0.0.1:18080` 与 `XDG_CONFIG_HOME=/tmp/taskai-gitlab-defaults-integration.tVlT8D wails dev -tags webkit2_41`；调试地址：`http://localhost:34115`。
- 初次打开导入弹窗为空，并持续显示“GitLab 地址、用户名和访问令牌将以未加密形式保存在本机”；输入模拟服务地址、`integration-user` 和 `integration-token` 后完整加载两页 6 个项目，地址格式默认为 SSH。
- 成功连接后取消并重开，三项输入全部自动回填；随后改为错误用户名和令牌，认证失败时当前输入保留，取消并重开仍恢复先前成功值，没有被失败连接覆盖。
- 选择 `team-a/api` 以 SSH 导入成功后重开，三项默认值仍在；停止并使用同一 XDG 配置重启 `wails dev`，刷新调试页后 Git 信息及三项默认值均恢复。
- 数据文件 `taskai/tasks.json` 权限为 `0600`，确认包含明文测试令牌；重启完成后的当前浏览器页面控制台无错误。测试结束后已关闭页面、`wails dev` 和模拟服务。

### 读取与保存拆分后的浏览器复核

- 时间：2026-08-14；隔离配置：`/tmp/taskai-gitlab-review-integration.0J9Tae`；模拟 GitLab 地址：`http://127.0.0.1:18080/private/gitlab`；`wails dev` 调试地址：`http://localhost:34115`。
- 浏览器临时把一次项目读取改为手动释放。读取期间地址、用户名和令牌均禁用，取消保持可用；取消并关闭后再释放真实读取，记录到读取调用 1 次、默认连接保存调用 0 次，确认晚到结果不保存。
- 浏览器临时把一次默认连接保存改为手动释放。界面进入“正在保存连接”后取消按钮禁用，按 Escape 弹窗仍保持打开；释放真实保存后完整展示两页 6 个项目。
- 选择 `team-a/api`，筛选保留 `team-a` 并切换为 HTTPS，再把一次导入改为受控延迟失败。导入期间三项连接、获取项目、筛选、地址格式、项目复选框、全选和取消选择当前结果均禁用；失败后错误可见，筛选、HTTPS 模式和选择状态完整恢复并重新可用。
- 恢复真实绑定后以 HTTPS 成功导入 1 个项目。使用同一隔离配置重启 `wails dev` 并重新打开弹窗，地址、用户名和令牌仍自动回填；数据文件权限为 `0600`，包含约定的明文测试令牌，重启后的浏览器控制台没有警告或错误。
- 测试完成后已关闭浏览器调试页、`wails dev` 和 GitLab 模拟服务。浏览器重连时 Wails 开发终端出现两条 `runtime:ready` 未识别消息，不影响页面调用、持久化或上述功能结果。

### 可执行程序人工验收

- 2026-08-17 使用 `./scripts/build-linux.sh amd64` 成功生成 `build/bin/taskai`，以隔离配置启动并保持运行。
- 用户完成界面检查后确认功能没有问题，验收程序随后关闭，允许继续合并到 `main`。
