## ADDED Requirements

### Requirement: 禁用鼠标剪贴板的终端不触发选区备注

终端记录的 `disableTaskAIMouseClipboard` 为 `true` 时，系统 MUST NOT 基于鼠标选区显示、定位或打开终端备注入口，也不得读取该选区以创建备注。该限制 MUST 与现有不自动复制、不拦截右键粘贴的行为一同生效；键盘输入、终端输出、文件拖放及其他非鼠标终端功能保持不变。

#### Scenario: 禁用终端的选区不显示备注入口

- **WHEN** 用户在 `disableTaskAIMouseClipboard` 为 `true` 的活动终端中鼠标选择非空文本
- **THEN** TaskAI 不显示备注图标、不打开备注输入框也不新增备注，终端程序继续处理该鼠标交互
