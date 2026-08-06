# 移除模态弹窗动画设计

## 目标

移除应用内模态弹窗的打开和关闭动画，避免居中弹窗关闭时 `transform` 被缩放动画覆盖而短暂跳位。

## 范围

- 修改共享的 `frontend/src/components/ui/dialog.tsx`。
- 任务、信息管理、设置及其他使用该组件的模态弹窗立即显示和隐藏。
- 下拉菜单、提示浮层和通知不在本次范围内。

## 方案

从 `DialogOverlay` 和 `DialogContent` 中移除 Radix 状态动画 class，包括 `animate-in`、`animate-out`、`fade-in-0`、`fade-out-0`、`zoom-in-95` 与 `zoom-out-95`。保留布局、遮罩颜色、焦点管理和关闭按钮交互。

`DialogContent` 的居中位移继续由 Tailwind 的 `-translate-x-1/2`、`-translate-y-1/2` 负责，不再被退出动画的 `transform` 覆盖。

## 验证

新增组件测试，断言打开的 Dialog 内容和遮罩均不带动画 class；运行前端测试、前端构建及 Linux 编译脚本。
