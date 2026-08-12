## 1. 隔离开发环境

- [x] 1.1 在项目目录创建 `.worktrees`，从当前项目分支创建 `feat/add-company-framework-lifecycle-preset` worktree，并确认工作树干净且使用正确分支
- [x] 1.2 将本变更的 OpenSpec 文档带入 worktree，运行 `openspec status --change add-company-framework-lifecycle-preset` 确认实施任务可用

## 2. 新安装默认数据

- [x] 2.1 在 `internal/settings/settings_test.go` 添加失败测试，断言新安装包含“默认预设”和默认选中的“公司框架”，并验证公司框架五个钩子的完整映射
- [x] 2.2 在 `internal/settings/settings_test.go` 添加失败测试，断言新安装稳定仓库更新链显示为“更新框架仓库”且命令范围、顺序和参数保持不变
- [x] 2.3 在 `internal/settings/settings.go` 分离新安装种子与旧数据兼容构造值，增加公司框架稳定预设 ID、映射和新安装仓库更新链名称，使设置测试通过

## 3. 既有数据兼容

- [x] 3.1 在 `internal/storage/repository_test.go` 添加失败测试，断言首次持久化使用公司框架默认数据，已有当前版本设置重载后不补预设、不切换默认项且不重命名仓库更新链
- [x] 3.2 在 `internal/storage/repository_test.go` 补充历史缺字段数据测试，断言原有一次性迁移仍生成“更新仓库”和原“默认预设”语义
- [x] 3.3 调整 `internal/settings/settings.go` 的初始化构造边界，使新安装和旧数据测试全部通过且不提升预置迁移版本
- [x] 3.4 在 `internal/lifecycle/service_test.go` 或现有应用层测试中验证默认创建入口把公司框架的三个链 ID 复制到新任务，且任务不保存预设 ID

## 4. 前端回归

- [x] 4.1 在 `frontend/src/App.test.tsx` 添加失败测试，断言设置页显示两个初始预设、“公司框架”默认标记和“更新框架仓库”链名称
- [x] 4.2 在 `frontend/src/App.test.tsx` 添加失败测试，断言新建任务自动显示公司框架的三个链选择、两个空钩子并通过现有绑定提交展开后的映射
- [x] 4.3 运行前端定向测试；仅在测试揭示既有数据驱动界面无法满足规格时，对 `frontend/src/App.tsx` 作最小调整

## 5. 自动验证与文档同步

- [x] 5.1 运行 `go test -race ./internal/settings ./internal/storage ./internal/lifecycle` 和相关根包定向测试，修复本变更导致的问题后重新运行
- [x] 5.2 在 `frontend` 运行 `npm test -- --run src/App.test.tsx` 和 `npm run build`，确认界面回归与类型编译通过
- [x] 5.3 运行完整 `go test -race ./...`、`frontend` 的 `npm test` 与 `npm run build`，仅处理本变更引起的失败
- [x] 5.4 同步 `README.md` 和 `docs/plans/2026-08-12-add-company-framework-lifecycle-preset-implementation-plan.md`，运行 `openspec validate add-company-framework-lifecycle-preset --strict`

## 6. Wails 集成测试与程序确认

- [x] 6.1 使用临时 `XDG_CONFIG_HOME` 以 `wails dev` 持续运行新用户实例，从彩色终端输出取得调试地址
- [x] 6.2 使用 Chrome DevTools 验证设置页的两个预设、公司框架默认标记、更新框架仓库名称，以及新建任务表单的五个钩子选择
- [x] 6.3 创建并重新编辑一个未执行任务，确认三个链 ID 已持久化；不得启动或结束任务，验证后关闭 `wails dev`
- [x] 6.4 使用 `scripts/build-linux.sh` 编译可执行程序，打开程序且保持运行，等待用户确认功能

## 7. 合并、归档与清理

- [x] 7.1 用户确认后，将当前工作区项目对应分支合并进 worktree 分支并解决冲突，再把 worktree 分支合并回当前工作区项目对应分支
- [x] 7.2 在合并后的当前项目运行完整 Go、前端测试和 `scripts/build-linux.sh`，确认合并结果可编译
- [x] 7.3 使用 OpenSpec 归档流程同步正式规格与归档文档，检查 `README.md` 和实施记录与最终行为一致
- [x] 7.4 提交全部相关 Git 变更，确认提交和工作树状态正确后移除已合并 worktree
