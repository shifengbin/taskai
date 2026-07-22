#!/usr/bin/env bash

set -euo pipefail

project_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
architecture="${1:-universal}"

case "$architecture" in
  amd64|arm64|universal) ;;
  *)
    echo "不支持的 macOS 架构: $architecture（可选: amd64、arm64、universal）" >&2
    exit 1
    ;;
esac

if [[ "$(uname -s)" != "Darwin" ]]; then
  echo "此脚本只能在 macOS 主机上运行。" >&2
  exit 1
fi
if ! command -v wails >/dev/null 2>&1; then
  echo "未找到 Wails CLI。请先安装 Wails v2。" >&2
  exit 1
fi
if ! command -v go >/dev/null 2>&1; then
  echo "未找到 Go。请安装项目所需的 Go 工具链。" >&2
  exit 1
fi
if ! command -v xcodebuild >/dev/null 2>&1; then
  echo "未找到 Xcode 命令行工具。请先执行 xcode-select --install。" >&2
  exit 1
fi

cd "$project_dir"
wails build -platform "darwin/$architecture" -clean
echo "macOS 构建完成: $project_dir/build/bin"
