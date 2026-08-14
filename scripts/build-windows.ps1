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

$Arguments = @('build', '-platform', "windows/$Architecture", '-clean')
$AppVersion = 'v0.0.0-dev'
$ProductVersion = $null
if ($Version) {
    $VersionWithoutPrefix = $Version.TrimStart('v')
    if ($VersionWithoutPrefix -notmatch '^(?<product>(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*))(?:-[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?(?:\+[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?$') {
        throw "无效的语义版本: $Version"
    }
    $AppVersion = "v$VersionWithoutPrefix"
    $ProductVersion = $Matches['product']
}
$Arguments += @('-ldflags', "-X main.appVersion=$AppVersion")
if ($NSIS) {
    $Arguments += '-nsis'
}

$OriginalWailsJson = $null
$LocationPushed = $false
try {
    if ($ProductVersion) {
        $OriginalWailsJson = [System.IO.File]::ReadAllBytes($WailsJsonPath)
        $config = Get-Content -LiteralPath $WailsJsonPath -Raw | ConvertFrom-Json
        if ($null -eq $config.info) {
            $config | Add-Member -Force -NotePropertyName 'info' -NotePropertyValue ([pscustomobject]@{})
        }
        if ($null -ne $config.info.PSObject.Properties['productVersion']) {
            $config.info.productVersion = $ProductVersion
        } else {
            $config.info | Add-Member -NotePropertyName 'productVersion' -NotePropertyValue $ProductVersion
        }
        [System.IO.File]::WriteAllText($WailsJsonPath, ($config | ConvertTo-Json -Depth 10))
    }

    Push-Location $ProjectDir
    $LocationPushed = $true
    & wails @Arguments
    if ($LASTEXITCODE -ne 0) {
        exit $LASTEXITCODE
    }
} finally {
    if ($LocationPushed) {
        Pop-Location
    }
    if ($null -ne $OriginalWailsJson) {
        [System.IO.File]::WriteAllBytes($WailsJsonPath, $OriginalWailsJson)
    }
}

Write-Host "Windows 构建完成: $ProjectDir\build\bin\taskai.exe"
