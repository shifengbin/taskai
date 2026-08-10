## Why

任务项会将描述截断为单行，用户依赖悬浮提示查看完整内容。现有提示会把原始描述中的换行折叠为空格，导致多行任务说明失去结构、难以阅读。

## What Changes

- 任务项悬浮描述提示保留任务描述中的原始换行和连续空白。
- 提示中的超长行仍可在面板宽度内自动折行，不改变任务项本身的固定两行布局、定位或空描述回退文案。

## Capabilities

### New Capabilities

- 无。

### Modified Capabilities

- `pine-night-run-visual-system`: 任务项悬浮描述提示须以保留换行的方式呈现原始描述内容。

## Impact

- 受影响代码：`frontend/src/components/TaskTree.tsx` 及其组件测试。
- 不涉及任务数据结构、Wails 绑定、后端 API 或新增依赖。
