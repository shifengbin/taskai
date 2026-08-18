# 任务目录链接实施记录

## 目标

让任务模板通过统一的 `directories` 字段选择一个或多个已有目录，并由生命周期命令把这些目录以受管理链接形式放入任务工作目录。目录只能通过原生选择器添加，模板可以控制是否支持多个目录以及任务创建后是否允许更新。

## 数据模型与模板约束

- `TaskTemplateField.InputType` 新增 `directories`，值统一保存为 `[]string`。
- `multiple=false` 时只允许一个目录，`multiple=true` 时每个数组元素都是独立目录地址。
- `updatable=false` 时任务创建成功后立即锁定；前端只读展示，后端比较归一化的新旧值并拒绝变化。
- 旧模板缺少 `updatable` 时按可更新处理，目录字段缺少 `multiple` 时按单目录处理。
- 目录字段不支持默认值和生命周期环境变量注入。
- 任务保存创建时的模板 ID。全局当前模板只决定新建任务使用哪个模板；既有任务的表单、校验、HTTP 输出、命令链输入和目录链接同步始终使用任务绑定模板的最新定义。
- 模板保存拒绝已使用字段类型变化、多值目录改为单目录，以及让已有任务形成必填空值且不可更新的定义。

## 目录选择与任务表单

- Wails 导出 `SelectDirectory`，使用原生 `OpenDirectoryDialog`。
- 前端通过可替换的 `DirectoryPicker` 调用绑定，测试可注入临时绝对路径。
- 页面只展示完整路径和“添加目录”“重新选择”“移除”操作，不渲染可手工输入路径的控件。
- 取消原生选择不改变当前值；同一任务的当前目录字段拒绝重复规范目录。
- 后端要求路径为已清理的绝对路径，并在保存时验证存在、可访问且为目录。

## 链接规划与安全边界

- 默认使用来源目录 basename 作为工作目录根级链接名。
- 名称按大小写不敏感规则比较；同名时从父目录逐级添加前缀，例如 `/project-a/src` 和 `/project-b/src` 生成 `project-a-src` 与 `project-b-src`。
- 同步前完整规划全部名称，不能安全消歧时拒绝操作，不使用序号或覆盖作为回退。
- Unix 使用目录 symlink，Windows 使用原生目录 symlink；不启动 Shell，也不弹出命令窗口。
- 每个任务通过工作区根下 `.taskai-ownership` 中的所有权令牌清单管理链接。
- 同步先写 pending 清单，再收敛文件系统，最后原子提交稳定清单；进程中断后可按 pending 状态恢复。
- 只删除清单登记、仍为目录链接且仍指向登记来源的条目。普通文件、普通目录、未登记链接或被替换的登记链接都会使同步失败。
- 删除任务工作目录只删除链接本身和清单，不跟随链接遍历或删除外部来源目录。

## 生命周期接入

- 新增固定系统命令 `system.lifecycle.sync-directory-links`，名称为“同步任务目录链接”。
- 命令适用于 `beforeStart`、`postStart` 和 `updateTask`，不接受参数，成功时原样传递标准输入。
- 默认创建工作目录链在创建后同步目录链接；`iterations-ai` 和仓库更新系统链也自动包含同步命令。
- 默认预设的 `updateTask` 使用同步专用链，公司框架预设继续使用首条命令为目录同步的仓库更新链。
- 迁移只更新与旧系统形状完全一致的命令、链、预设和任务选择，用户自定义配置保持不变。
- 重试从持久化任务重新读取目录值和绑定模板，不使用失败请求中的旧缓存。

## 自动化验证

执行以下命令并通过：

```sh
go test -race ./... -count=1
cd frontend
npm test
npm run build
cd ..
openspec validate add-task-directory-links --strict
./scripts/build-linux.sh
```

验证结果：

- Go 全量 race 测试通过。
- 前端 18 个测试文件、381 项测试通过。
- TypeScript 与 Vite 生产构建通过。
- OpenSpec 严格校验通过。
- Linux 构建脚本生成可执行文件 `build/bin/taskai`。
- Windows 的目录链接、生命周期和主程序路径完成 `GOOS=windows GOARCH=amd64` 构建验证。

## Wails 集成测试

1. 创建隔离配置目录和三个来源目录：`project-a/src`、`project-b/src` 以及用于单目录验证的临时目录。
2. 使用 `XDG_CONFIG_HOME=<隔离目录> wails dev -tags webkit2_41` 启动应用，从输出取得 `http://localhost:34115`。
3. 使用 chrome-devtools 打开调试地址，在模板设置中创建 `directories` 字段，验证单目录、多目录、必填和创建后可更新开关，并确认目录字段没有默认值和环境变量设置。
4. 在浏览器中把目录选择适配器依次替换为返回 `project-a/src`、取消、`project-b/src` 和重复 `project-b/src`，验证取消不改变值、重复路径显示错误、页面没有路径文本输入框。
5. 创建多目录任务并重新编辑，确认不可更新字段立即只读，不能重新选择、移除或添加。
6. 启动任务，检查工作目录存在 `project-a-src` 和 `project-b-src`，并分别指向两个独立来源目录。
7. 切换全局当前模板后重新打开既有任务，确认详情、编辑锁定和链接同步仍使用任务绑定模板。
8. 创建单目录模板，验证选择后只保留一项，重新选择会替换原路径而不是追加。
9. 恢复真实 Wails 目录选择绑定并调用一次，确认原生“选择任务目录”窗口实际打开；随后取消并检查浏览器控制台没有错误。
10. 测试结束后关闭 `wails dev`，再使用编译产物打开隔离数据应用进行人工验收。

## 发布

功能合并、规格归档和主分支构建验证完成后，创建并推送 `v0.0.7` 标签。标签触发 `build-release` 工作流生成 Linux DEB、Windows NSIS 安装包、macOS DMG 和 `taskai-update.json`，工作流成功后核对 GitHub Release 资产。
