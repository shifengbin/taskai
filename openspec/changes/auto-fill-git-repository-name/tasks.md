## 1. Git 信息草稿回填

- [x] 1.1 在 `frontend/src/App.tsx` 增加可测试的仓库名称提取逻辑：仅从去除首尾空白后、以 `.git` 结尾的地址中提取最后一个 `/` 后的非空名称。
- [x] 1.2 调整额外信息固定字段的更新逻辑：编辑 `git` 分类的 `repository` 字段时，始终保存地址输入值；仅在 `name` 为空白且提取成功时同步回填项目名称。
- [x] 1.3 保持非 `git` 分类、非仓库地址字段、模板默认值初始化和已有非空项目名称的现有行为不变。

## 2. 前端交互测试

- [x] 2.1 在 `frontend/src/App.test.tsx` 补充新增 Git 信息的测试，验证填写 `git@gitlab.jiandan100.cn:webdev/interact-study.git` 后项目名称自动为 `interact-study`。
- [x] 2.2 补充回归测试，验证已填写项目名称时编辑仓库地址不会覆盖名称，且未以 `.git` 结尾或无法提取的地址不会回填空名称。
- [x] 2.3 补充非 Git 分类的回归测试，验证同样的 `repository` 字段编辑不会自动修改名称。

## 3. 验证

- [x] 3.1 在 `frontend` 目录运行 `npm test`，确认额外信息管理及全部前端交互测试通过。
- [x] 3.2 在 `frontend` 目录运行 `npm run build`，确认 TypeScript 类型检查与 Vite 构建通过。
- [x] 3.3 运行 `openspec validate auto-fill-git-repository-name --strict` 和 `git diff --check`，确认变更产物及工作区差异有效。
