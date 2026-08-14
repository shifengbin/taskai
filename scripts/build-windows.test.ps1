$ErrorActionPreference = 'Stop'

$ProjectDir = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
$TestDir = Join-Path ([System.IO.Path]::GetTempPath()) ("taskai-windows-build-test-$([guid]::NewGuid().ToString('N'))")
$TestProject = Join-Path $TestDir 'project'
$FakeBin = Join-Path $TestDir 'bin'
$CapturePath = Join-Path $TestDir 'wails-arguments.txt'
$ConfigCapturePath = Join-Path $TestDir 'wails-config.json'
$OriginalPath = $env:PATH
$OriginalOS = $env:OS
$OriginalCapturePath = $env:TASKAI_WINDOWS_TEST_ARGS
$OriginalConfigCapturePath = $env:TASKAI_WINDOWS_TEST_CONFIG

try {
    New-Item -ItemType Directory -Force -Path (Join-Path $TestProject 'scripts'), (Join-Path $TestProject 'build\windows'), $FakeBin | Out-Null
    Copy-Item -LiteralPath (Join-Path $ProjectDir 'scripts\build-windows.ps1') -Destination (Join-Path $TestProject 'scripts\build-windows.ps1')
    $OriginalConfig = '{"name":"taskai","outputfilename":"taskai"}'
    [System.IO.File]::WriteAllText((Join-Path $TestProject 'wails.json'), $OriginalConfig)
    Set-Content -LiteralPath (Join-Path $TestProject 'build\windows\icon.ico') -Encoding ascii -Value 'stale icon'

    @'
@echo off
> "%TASKAI_WINDOWS_TEST_ARGS%" echo %*
copy /Y "%CD%\wails.json" "%TASKAI_WINDOWS_TEST_CONFIG%" >nul
exit /b 0
'@ | Set-Content -LiteralPath (Join-Path $FakeBin 'wails.cmd') -Encoding ascii
    @'
@echo off
exit /b 0
'@ | Set-Content -LiteralPath (Join-Path $FakeBin 'go.cmd') -Encoding ascii

    $env:PATH = "$FakeBin;$OriginalPath"
    $env:OS = 'Windows_NT'
    $env:TASKAI_WINDOWS_TEST_ARGS = $CapturePath
    $env:TASKAI_WINDOWS_TEST_CONFIG = $ConfigCapturePath
    $BuildScript = Join-Path $TestProject 'scripts\build-windows.ps1'

    & $BuildScript -Architecture amd64 -NSIS -Version '1.2.3-rc.1'
    $ReleaseArguments = Get-Content -LiteralPath $CapturePath -Raw
    foreach ($Expected in @('build', '-platform windows/amd64', '-clean', '-ldflags', '-X main.appVersion=v1.2.3-rc.1', '-nsis')) {
        if (-not $ReleaseArguments.Contains($Expected)) {
            throw "Windows 发布构建缺少参数 $Expected，实际参数: $ReleaseArguments"
        }
    }
    if (Test-Path -LiteralPath (Join-Path $TestProject 'build\windows\icon.ico')) {
        throw 'Windows 构建前没有移除旧图标。'
    }
    $BuildConfig = Get-Content -LiteralPath $ConfigCapturePath -Raw | ConvertFrom-Json
    if ($BuildConfig.info.productVersion -ne '1.2.3') {
        throw "wails.json 产品版本为 $($BuildConfig.info.productVersion)，预期为 1.2.3。"
    }
    if ([System.IO.File]::ReadAllText((Join-Path $TestProject 'wails.json')) -ne $OriginalConfig) {
        throw 'Windows 发布构建后没有恢复 wails.json。'
    }

    & $BuildScript -Architecture arm64
    $DevelopmentArguments = Get-Content -LiteralPath $CapturePath -Raw
    foreach ($Expected in @('-platform windows/arm64', '-X main.appVersion=v0.0.0-dev')) {
        if (-not $DevelopmentArguments.Contains($Expected)) {
            throw "Windows 开发构建缺少参数 $Expected，实际参数: $DevelopmentArguments"
        }
    }
    if ($DevelopmentArguments.Contains('-nsis')) {
        throw '未指定 NSIS 时不应传递 -nsis。'
    }

    Write-Host 'Windows 构建脚本集成测试通过'
} finally {
    $env:PATH = $OriginalPath
    $env:OS = $OriginalOS
    $env:TASKAI_WINDOWS_TEST_ARGS = $OriginalCapturePath
    $env:TASKAI_WINDOWS_TEST_CONFIG = $OriginalConfigCapturePath
    if (Test-Path -LiteralPath $TestDir) {
        Remove-Item -LiteralPath $TestDir -Recurse -Force
    }
}
