## 1. Worktree 与主题清单

- [x] 1.1 在 `taskai/.worktrees/` 创建本变更专用 worktree 和 `split-design-preview-themes` 分支，确认工作区不包含无关改动
- [x] 1.2 从现有 `design-preview/index.html` 建立 36 套主题清单，记录编号、英文短名、中文名称、主题 CSS 类、亮暗色令牌和主题专属品牌文案
- [x] 1.3 确认提取前的基线：原页面包含 36 个主题区块、每套两个变体，并记录可用于拆分后对照的主题数量和名称集合

## 2. 公共预览资源

- [x] 2.1 创建 `design-preview/_shared/`，提取 gallery 基础样式、公共布局骨架、响应式规则和减少动态效果规则
- [x] 2.2 提取 SVG 图标符号、终端示例输出和任务/终端静态 HTML 模板，确保公共脚本不再包含 36 套主题的条件分支
- [x] 2.3 为公共资源建立相对路径约定，并验证从本地文件系统直接打开主题页面时 CSS、脚本和图标均可加载

## 3. 主题页面拆分

- [x] 3.1 按 01–12 创建 `themes/<number>-<slug>/index.html`，每页保留自己的主题 CSS、主题元数据和 Light/Dark 两个预览
- [x] 3.2 按 13–24 创建 `themes/<number>-<slug>/index.html`，覆盖 Candy、Doodle、Y2K、Groovy、Disco、Burst、Acid、Riso、Memphis、Sunset、Concrete 和 Neon
- [x] 3.3 按 25–36 创建 `themes/<number>-<slug>/index.html`，覆盖 Bolt、Gummy、Stamp、Punch、Snap、Brio、Boom、Zap、Crayon、Riot、Juice 和 Pixel
- [x] 3.4 对每个页面核对主题专属品牌名、窗口标题、主题编号、页面标题、亮暗色 class 和主题 CSS 变量，确保没有串入其他主题皮肤
- [x] 3.5 保留静态选择提示、横向滚动、状态点脉冲、截图查询参数和 `prefers-reduced-motion` 行为，不引入生产应用 API 调用

## 4. 主题设计文档

- [x] 4.1 建立中文 `design.md` 统一章节模板，包含定位、关键词、色彩、字体、布局、组件、状态、交互、可读性和 React + MUI 映射
- [x] 4.2 为 01–06（Monolith、Atelier、Nebula、Wabi、Pop、Deco）编写并校对设计文档，令牌与页面 CSS 一致
- [x] 4.3 为 07–12（Comic、Holo、Arcade、Citrus、Blocks、Sunny）编写并校对设计文档，令牌与页面 CSS 一致
- [x] 4.4 为 13–18（Candy、Doodle、Y2K、Groovy、Disco、Burst）编写并校对设计文档，令牌与页面 CSS 一致
- [x] 4.5 为 19–24（Acid、Riso、Memphis、Sunset、Concrete、Neon）编写并校对设计文档，明确 hard-shadow family 的共性和每套主题差异
- [x] 4.6 为 25–30（Bolt、Gummy、Stamp、Punch、Snap、Brio）编写并校对设计文档，明确 Pop family 的共性和每套主题差异
- [x] 4.7 为 31–36（Boom、Zap、Crayon、Riot、Juice、Pixel）编写并校对设计文档，明确 Pop family 的共性和每套主题差异
- [x] 4.8 检查 36 份文档都包含亮暗色令牌、字体、组件状态和实现映射，不得只保留原页面的一句简介

## 5. 总览页与说明

- [x] 5.1 将 `design-preview/index.html` 改为主题总览页，移除已拆出的主题区块和主题专属 CSS，只保留公共说明、主题卡片/链接和导航样式
- [x] 5.2 增加 01–36 的完整主题导航，修正标题、引导文案、选择提示、页脚和主题数量描述
- [x] 5.3 更新 `design-preview/readme.md`，说明总览页、主题目录、公共资源目录、设计文档位置和直接打开方式

## 6. 结构与浏览器验证

- [x] 6.1 编写或运行静态检查，确认主题目录恰好为 36 个，每个目录同时存在 `index.html` 和 `design.md`
- [x] 6.2 检查总览页的 36 个链接、主题页面标题、文档标题和主题清单集合完全一致，并确认页面中不再误报 24 套主题
- [x] 6.3 直接打开代表性主题页面（独立主题、hard-shadow family、Pop family），验证 Light/Dark stage、任务列表、终端输出、字体回退和图标无空白或资源错误
- [x] 6.4 在窄视口验证主题预览可以横向滚动，在减少动态效果偏好下验证入场和状态脉冲被关闭但内容仍可见
- [x] 6.5 将当前分支最新提交同步/合并到 worktree 分支，在 worktree 中完成最终检查后再把 worktree 分支合并回当前分支
- [x] 6.6 按项目要求运行编译脚本，并确认静态预览拆分没有修改生产应用构建结果
