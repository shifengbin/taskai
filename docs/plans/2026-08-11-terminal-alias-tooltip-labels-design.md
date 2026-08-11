# Terminal Alias Tooltip Labels Design

## Goal

让设置了终端别名后的悬浮提示以清晰字段名展示会话信息。

## Decision

仅在共享 `TerminalName` 组件的悬浮提示中为既有两行值加上固定前缀：第一行使用 `标题: `，第二行使用 `命令: `。两行值仍分别来自实际终端标题和启动命令；缺失命令时继续显示既有缺省值。

## Boundaries

不修改 `TerminalRecord`、别名编辑、任务树或右侧标题栏的调用方式。实际标题的 OSC 解析和 `title-change` 状态上报继续只读取 `title`，不读取提示文案或别名。

## Verification

更新共享组件测试，断言别名存在时提示严格显示 `标题: npm run dev` 与 `命令: zsh`，并重新运行前端测试、生产构建和桌面应用。
