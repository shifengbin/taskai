# 终端重绘压力脚本：模拟 codex 等 TUI 在 ConPTY 下产生的
# "逐行重绘 + 密集光标定位"合成输出流，用于验证嵌入终端是否会
# 渲染出重绘中间态（光标乱跑）。
#
# 用法：pwsh -File terminal-redraw-stress.ps1 [-Frames 300] [-Fps 15] [-Rows 20]
param(
	[int]$Frames = 300,
	[int]$Fps = 15,
	[int]$Rows = 20,
	[int]$Cols = 76
)

# 仅输出 ASCII，避免 PowerShell 5.1 控制台编码（GBK）与 ConPTY UTF-8 流不匹配导致乱码
$e = [char]27
$spin = '|/-\'
$column = $Cols - 1
for ($frame = 1; $frame -le $Frames; $frame++) {
	$builder = [System.Text.StringBuilder]::new()
	# 整帧重绘：光标归位，随后逐行定位、整行重写（ConPTY 合成流形态）
	[void]$builder.Append("$e[H")
	for ($row = 1; $row -le $Rows; $row++) {
		[void]$builder.Append("$e[${row};1H")
		$spinner = $spin[($frame + $row) % 4]
		$line = "frame $frame row $row $spinner ".PadRight($column, '.')
		[void]$builder.Append($line)
	}
	# 帧末把光标停到固定位置：行为正确的终端应始终只在固定位置看到光标
	[void]$builder.Append("$e[${Rows};8H")
	[Console]::Write($builder.ToString())
	Start-Sleep -Milliseconds ([int](1000 / $Fps))
}
[Console]::Write("$e[HDONE all $Frames frames`n")
