# 六套界面风格图鉴实施计划

> **执行说明：** 按任务逐项执行；实现时使用 `superpowers:executing-plans` 逐步验证。

**目标：** 新增一个不依赖 Wails API 的独立 React 页面，以可交互的亮色、暗色和弹窗状态展示任务工作台的六套高饱和视觉方案。

**架构：** 在 Vite 中登记第二个 HTML 入口 `theme-atlas.html`，使预览页面与现有 `index.html` 和 `App.tsx` 完全分离。`ThemeAtlas` 使用静态主题数据生成六张完整工作台预览卡片，每张卡片维护自身的视图状态；大量视觉差异由主题专属 CSS 变量、装饰层和布局变体实现。

**技术栈：** React 18、TypeScript、Vite、CSS、自带 Nunito 字体、Vitest 与 Testing Library。

---

### 任务 1：登记独立的 Vite 页面入口

**文件：**

- 新建：`frontend/theme-atlas.html`
- 新建：`frontend/src/theme-atlas.tsx`
- 修改：`frontend/vite.config.ts`

**步骤 1：建立可失败的构建检查**

运行：`npm run build`

预期：在未登记入口时，构建产物中不包含 `theme-atlas.html`，作为后续验证基线。

**步骤 2：添加独立 HTML 与 React 入口**

在 `frontend/theme-atlas.html` 建立中文页面标题和 `#root` 容器，并以模块脚本加载 `./src/theme-atlas.tsx`。在入口文件中渲染 `ThemeAtlas`，不导入 `App.tsx`、`api.ts` 或 Wails 绑定。

**步骤 3：将两个 HTML 文件声明为构建输入**

在 `frontend/vite.config.ts` 的 `build.rollupOptions.input` 中显式登记 `index.html` 与 `theme-atlas.html` 的绝对路径，确保生产构建不丢失预览页。

**步骤 4：验证入口构建**

运行：`npm run build && test -f dist/theme-atlas.html`

预期：TypeScript 与 Vite 成功完成，且 `dist/theme-atlas.html` 存在。

### 任务 2：定义六套主题的演示数据与状态约定

**文件：**

- 新建：`frontend/src/theme-atlas-data.ts`
- 新建：`frontend/src/ThemeAtlas.test.tsx`

**步骤 1：编写失败测试，确认图鉴基础内容**

在 `ThemeAtlas.test.tsx` 渲染 `<ThemeAtlas />`，断言六个 `article` 的标题依次为“果汁俱乐部、夜市电台、拼图总部、熔岩赛道、泳池派对、糖纸档案”，并且每张卡片都提供可见名称的“亮色”“暗色”“弹窗”状态按钮。

```tsx
expect(screen.getAllByRole('article')).toHaveLength(6)
expect(screen.getByRole('heading', {name: '果汁俱乐部'})).toBeInTheDocument()
expect(screen.getAllByRole('button', {name: '亮色'})).toHaveLength(6)
```

**步骤 2：运行测试确认失败**

运行：`npm test -- ThemeAtlas.test.tsx`

预期：失败，提示尚未找到 `ThemeAtlas` 模块。

**步骤 3：创建主题元数据模块**

定义 `PreviewMode = 'light' | 'dark' | 'modal'`、`ThemeConcept` 和只读主题列表。每条数据必须有稳定 `id`、中文名称、设计关键词、三枚主色、任务/终端示例文案和对应 CSS 主题标识；主题列表只在此处维护，避免 JSX 内复制颜色与文案。

**步骤 4：验证数据契约**

运行：`npm test -- ThemeAtlas.test.tsx`

预期：仍因组件尚未实现而失败，但不出现 TypeScript 数据类型错误。

### 任务 3：实现可访问的图鉴交互骨架

**文件：**

- 新建：`frontend/src/ThemeAtlas.tsx`
- 修改：`frontend/src/ThemeAtlas.test.tsx`

**步骤 1：扩展失败测试，覆盖状态切换与弹窗关闭**

为“果汁俱乐部”卡片点击“暗色”，断言其预览容器的 `data-mode` 变为 `dark`；点击“弹窗”后断言可访问名称为“新建任务”的对话框出现；点击“取消”和 Esc 后，断言该对话框关闭且卡片回到亮色状态。

```tsx
await user.click(within(card).getByRole('button', {name: '暗色'}))
expect(within(card).getByTestId('workspace-preview')).toHaveAttribute('data-mode', 'dark')
await user.click(within(card).getByRole('button', {name: '弹窗'}))
expect(within(card).getByRole('dialog', {name: '新建任务'})).toBeInTheDocument()
```

**步骤 2：运行测试确认失败**

运行：`npm test -- ThemeAtlas.test.tsx`

预期：失败，提示缺少交互元素或状态属性。

**步骤 3：实现页面语义和本地状态**

实现页面标题、用途说明、主题锚点导航和六个 `article`。以 `Record<string, PreviewMode>` 保存每张卡片状态；状态按钮使用 `aria-pressed` 并只更新所属卡片。预览应含任务树、状态标签、任务详情、终端、任务操作按钮和色彩样本。弹窗模式渲染具备 `role="dialog"`、`aria-modal="true"` 与明确标题的“新建任务”表单；遮罩、关闭按钮、取消按钮和 Esc 均调用同一个关闭函数。

**步骤 4：验证交互**

运行：`npm test -- ThemeAtlas.test.tsx`

预期：新增测试通过，且单张卡片状态切换不改变其他卡片。

### 任务 4：实现六个差异鲜明的视觉系统

**文件：**

- 新建：`frontend/src/theme-atlas.css`
- 修改：`frontend/src/ThemeAtlas.tsx`

**步骤 1：添加视觉结构测试**

在 `ThemeAtlas.test.tsx` 断言每张方案有三枚色样、任务树导航、终端区域和“新建任务”动作，避免 CSS 重构时丢失关键产品场景。

**步骤 2：运行测试确认失败**

运行：`npm test -- ThemeAtlas.test.tsx`

预期：在上述结构尚未完成时失败。

**步骤 3：编写图鉴与主题样式**

在 `theme-atlas.css` 中：

- 用 `@font-face` 引入现有 `assets/fonts/nunito-v16-latin-regular.woff2`，中文字形使用衬线回退；不引入网络字体。
- 为每个主题定义亮、暗两组 CSS 变量（画布、表面、文字、主色、强调色、边框、终端背景），由 `data-theme` 和 `data-mode` 选择。
- 分别实现果汁贴纸、夜市荧光栅格、拼图粗框、熔岩速度线、泳池波纹和糖纸档案抽屉六种装饰与布局变化，避免仅替换配色。
- 让弹窗继承所属主题的变量，并为遮罩、输入、色卡与主要操作按钮设计独立状态。
- 提供键盘焦点、悬浮、禁用与 `prefers-reduced-motion` 样式；在窄屏把固定导航转为横向索引，预览工作区改为单列。

**步骤 4：验证页面与样式**

运行：`npm test -- ThemeAtlas.test.tsx && npm run build`

预期：测试通过，构建成功，无 CSS 或 TypeScript 错误。

### 任务 5：进行集成检查与项目编译

**文件：**

- 修改：`docs/plans/2026-07-31-theme-atlas-design.md`（仅在实现与设计有偏差时更新）

**步骤 1：运行全量前端测试**

运行：`npm test`

预期：全部 Vitest 测试通过；现有 `App` 和组件测试不受独立入口影响。

**步骤 2：运行前端生产构建**

运行：`npm run build`

预期：`dist/index.html` 和 `dist/theme-atlas.html` 均生成。

**步骤 3：运行项目规定的 Linux 编译脚本**

运行：`./scripts/build-linux.sh`

预期：在安装了 Wails、GTK 3 和 WebKitGTK 依赖的 Linux 主机上完成构建并生成 `build/bin/taskai`；若环境缺少前置依赖，记录脚本的明确诊断，不为本任务改动构建环境或业务代码。

**步骤 4：人工浏览器检查**

运行：`npm run dev -- --host 127.0.0.1`

打开：`http://127.0.0.1:5173/theme-atlas.html`

预期：六套方案均可滚动访问；每张卡片的亮色、暗色和弹窗切换独立工作；弹窗可以关闭；窄屏无横向溢出。

## 版本控制说明

当前共享工作树存在与本功能无关的未提交改动。遵循共享工作区约束，本计划不包含提交操作；实现时只暂存或检查本计划涉及的文件，不触碰已有改动。
