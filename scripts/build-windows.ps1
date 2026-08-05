[CmdletBinding()]
param(
    [ValidateSet('amd64', 'arm64', '386')]
    [string]$Architecture = 'amd64',
    [switch]$NSIS
)

$ErrorActionPreference = 'Stop'
$ProjectDir = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
$IconPath = Join-Path $ProjectDir 'build\windows\icon.ico'

if ($env:OS -ne 'Windows_NT') {
    throw '此脚本只能在 Windows 主机上运行。'
}
if (-not (Get-Command wails -ErrorAction SilentlyContinue)) {
    throw '未找到 Wails CLI。请先安装 Wails v2。'
}
if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    throw '未找到 Go。请安装项目所需的 Go 工具链。'
}

# Wails 仅在 ICO 不存在时从 appicon.png 生成图标，删除旧文件可同步应用与 NSIS 安装程序图标。
if (Test-Path -LiteralPath $IconPath) {
    Remove-Item -LiteralPath $IconPath -Force
}

$Arguments = @('build', '-platform', "windows/$Architecture", '-clean')
if ($NSIS) {
    $Arguments += '-nsis'
}

Push-Location $ProjectDir
try {
    & wails @Arguments
    if ($LASTEXITCODE -ne 0) {
        exit $LASTEXITCODE
    }
} finally {
    Pop-Location
}

Write-Host "Windows 构建完成: $ProjectDir\build\bin\taskai.exe"
