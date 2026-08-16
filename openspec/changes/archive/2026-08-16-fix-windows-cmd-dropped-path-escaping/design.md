# 设计：修复 cmd 拖放路径转义在 Windows 26200 构建上的回归

## 上下文

- 现状代码：`internal/terminal/file_paths.go` cmd 分支用 `strings.NewReplacer` 把 `^ & | < > % !` 全部 caret 转义后包进双引号。
- 该策略建立在旧版 cmd.exe 行为上：引号内的 caret 会被处理（`^&`→`&`），且 `^%` 能阻断 `%VAR%` 展开。
- 用户机器（Windows 11 Home China 10.0.26200.9168）上的 cmd.exe 改变了这一行为。

## 证据矩阵（真实 ConPTY cmd 会话探针，Windows 10.0.26200.9168）

探针方法：`windowsBackend.Start` 启动真实 cmd 会话，逐行写入候选格式；一部分用 cmd 自身 `for %A in (<格式>) do @echo [%~A]` 测 cmd 解析，另一部分调用真实 .exe 接收方（Go 编译、打印 `os.Args`）测 CRT argv 还原。环境预设 `HOME=expanded`，`PATH` 为系统已定义变量。

| 编号 | 写入形式 | 结果 | 结论 |
| --- | --- | --- | --- |
| P1 | `"a^&b^^f^%HOME^%x!y.txt"`（当前实现） | `a^&b^^f^%HOME^%x!y.txt` | caret 全部字面残留，当前实现即错误输出 |
| Q1 | `"C:\Work Files\a&b^f!x.txt"`（引号内裸放） | `C:\Work Files\a&b^f!x.txt` | 引号内特殊字符裸放完美还原 |
| R3 | .exe 接收 `"C:\Work Files\a&b^f x!y.txt"` | argv = 原路径 | 引号内裸放对 CRT 接收方同样完美 |
| Q2 | `"z%HOME%w.txt"` | `zexpandedw.txt` | 引号内成对 %VAR% 仍展开（唯一风险字符） |
| Q13 | `"50%.txt"` | `50%.txt` | 单个不成对 % 保持字面 |
| P3/P6 | 引号外 `a^&b^|c^<d^>e.txt` | `a&b|c<d>e.txt` | 引号外 caret 可用，但 Q5 证明引号外 `^ ` 无法保持单项 → 空格必须进引号 |
| P2/P7/P9 | 混合分段（引号段+引号外 caret 段） | FOR 项残留 `"` 杂质 | cmd 内建解析上下文脏 |
| R1/R5/R6 | 引号内 `%""` 打断变量名 | 不展开，但 CRT argv 泄漏 1 个 `"`（如 `x%"PATH%y`） | 防展开成功、argv 不干净 |
| Q3/P4a | 引号内 `%%`/`%%%%` 加倍 | 先折叠成 `%` 再二次扫描展开 | 不可用 |
| MS1/MS5/MS7 | 混合分段 + `^%`（引号外） | 部分行（MS2/MS4/MS6）argv 完美，部分结构几乎相同的行（MS1/MS5/MS7）照样展开 | %/caret/引号相位交互无法黑盒预测，不可依赖 |
| R2 | .exe 接收 `"a%PATH%b.txt"`（裸放对照） | PATH 全文灌入 argv | 成对 %VAR% 裸放的灾难后果（对照） |

参考行为：Windows Terminal 向 cmd 拖放同样只加引号、不保护 `%`（生态标准），成对 `%VAR%` 展开是 cmd.exe 全生态共同的已知限制。

## 方案

cmd 分支改为：

```go
case "cmd":
    if strings.Contains(path, `"`) {
        return "", fmt.Errorf("cmd.exe 路径不能包含双引号")
    }
    return `"` + path + `"`, nil
```

- 拒绝 `"`：Windows 文件名本就禁止 `"`，防御性保留。
- 其余内容原样 + 整体双引号：由 Q1/R3 证据覆盖 `空格 & | < > ^ !` 与单个 `%`。
- 成对 `%VAR%` 展开：接受为已知限制（无任何构造能同时满足"防展开 + argv 干净"，见矩阵 R1/MS 行）。

备选方案与否决理由：
1. 保持 caret 转义（旧策略）——26200 构建上产出字面 caret，即本次要修的 bug。
2. `%""` 打断——argv 泄漏引号（R1/R5/R6）。
3. 引号外 caret 混合分段——空格无法保持单项（Q5）、cmd 内建解析残留 `"`（P2/P7/P9）、% 相位不可预测（MS 行）。

## 测试策略

- 表驱动单测（跨平台）：cmd 用例期望改为 `"原样路径"`。
- Windows 集成单测：`cmd.exe /D /V:OFF /Q` 管道执行 `for %A in (<格式>) do @echo [%~A]`，断言输出含 `[原路径]`；测试路径用单个 `%`（`C:\Work Files\a&b^f 50% x!y.txt`）避开已知限制；断言改为子串匹配（`/Q` 下输出与提示符同行，不能按行首 `[` 匹配）。
- 端到端集成测试（wails dev + chrome-devtools）：见 tasks.md 与下方细节。

## 集成测试细节（wails dev + chrome-devtools）

前置：
1. 准备真实文件（PowerShell 单引号避免转义污染）：
   `New-Item -ItemType Directory C:\taskai-drop-test`；写入文件 `C:\taskai-drop-test\a&b^c 50% x!y.txt`（内容任意，如 `ok`）。
2. 工作区运行 `wails dev`，等待输出调试地址（浏览器侧必须用 asset server 地址 `http://localhost:34115`，vite 的 5173 没有 `window.runtime` 注入，`EventsOnMultiple` 会报 undefined）。

步骤：
1. chrome-devtools 打开 `http://localhost:34115`。
2. 创建一个任务并启动其默认终端（Windows 默认 Shell = COMSPEC = cmd.exe，无需改设置）。
3. 等待终端出现 cmd 提示符（banner 版本行 + `路径>`）。
4. 用 `evaluate_script` 调 `window.go.main.App.WriteTerminalFilePaths(taskId, terminalId, ["C:\\taskai-drop-test\\a&b^c 50% x!y.txt"])`——这与前端 OnFileDrop 回调走的同一条 Go 绑定（原生拖拽事件无法在浏览器工具里模拟，绑定层之上无前端逻辑分支，等价）。
5. 通过 xterm DOM（`.xterm-screen` 文本）确认插入文本恰为 `"C:\taskai-drop-test\a&b^c 50% x!y.txt"`（含首尾引号、内容无 caret）。
6. 键盘输入补全命令：先在调用前手工输入 `for %A in (`，drop 后输入 `) do @echo [%~A]` 并回车；读取终端输出，断言出现 `[C:\taskai-drop-test\a&b^c 50% x!y.txt]` 精确一行（cmd 解析后单一字面参数）。
7. 多文件：`WriteTerminalFilePaths(..., [路径1, 路径2])`，确认两个引号参数以单个空格连接；`for %A in (…) do @echo [%~A]` 依次输出两行精确路径。
8. `list_console_messages` 确认无 error。
9. 关闭 `wails dev`。

判定：步骤 5–8 全部满足即通过；任何出现字面 `^`、参数拆分、或控制台 error 即失败。
