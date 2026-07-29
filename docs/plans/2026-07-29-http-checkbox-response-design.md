# HTTP 复选框响应类型设计

## 目标

任务详情 HTTP 接口在返回动态参数时保留其输入语义：复选框参数返回 JSON 布尔值 `true` 或 `false`，文本参数继续返回 JSON 字符串。

## 根因

任务快照中的 `ExtraInfoParameter.Value` 使用字符串保存，这是现有持久化兼容模型的一部分。应用层的 `httpExtraInfo` 又将全部字段与参数放入 `map[string]string`，使复选框已知的 `inputType` 在序列化前丢失，最终响应出现字符串 `"true"` 和 `"false"`。

## 决策

仅在 HTTP 输出边界转换类型：

- `httpExtraInfo` 使用可承载不同 JSON 标量的 `map[string]any`；
- 固定字段与 `text` 参数保留字符串；
- `checkbox` 参数根据规范化后的字符串值输出布尔值；
- 任务列表接口继续不返回 `extraInfo`，Wails 绑定、前端状态和 JSON 持久化均不改变。

这一边界转换既满足 API 消费者需要的原生 JSON 类型，也不会破坏已有快照、旧数据迁移或前端输入组件依赖的字符串模型。

## 验证

应用层 HTTP 测试创建一个同时包含 Git 文本参数 `branch` 和任务级复选框参数的任务。断言详情响应中的 `branch` 是字符串，值为 `true` 与 `false` 的复选框均为布尔值，以防止将所有参数误转换为布尔值或恢复为字符串。
