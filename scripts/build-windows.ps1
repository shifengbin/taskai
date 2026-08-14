[CmdletBinding()]
param(
    [ValidateSet('amd64', 'arm64', '386')]
    [string]$Architecture = 'amd64',
    [switch]$NSIS,
    [string]$Version
)

$ErrorActionPreference = 'Stop'
$ProjectDir = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
$IconPath = Join-Path $ProjectDir 'build\windows\icon.ico'
$WailsJsonPath = Join-Path $ProjectDir 'wails.json'

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

# 将版本号写入 wails.json，使 Wails 构建产物与 NSIS 安装程序使用与三端一致的发布版本。
if ($Version) {
    $config = Get-Content -LiteralPath $WailsJsonPath -Raw | ConvertFrom-Json
    if ($null -ne $config.PSObject.Properties['version']) {
        $config.version = $Version
    } else {
        $config | Add-Member -NotePropertyName 'version' -NotePropertyValue $Version
    }
    [System.IO.File]::WriteAllText($WailsJsonPath, ($config | ConvertTo-Json -Depth 10))
}

$Arguments = @('build', '-platform', "windows/$Architecture", '-clean')
$AppVersion = if ($Version) { "v$($Version.TrimStart('v'))" } else { 'v0.0.0-dev' }
$Arguments += @('-ldflags', "-X main.appVersion=$AppVersion")
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
