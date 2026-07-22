#!/usr/bin/env bash

set -euo pipefail

project_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
architecture="${1:-amd64}"

case "$architecture" in
  amd64|arm64|arm) ;;
  *)
    echo "不支持的 Linux 架构: $architecture（可选: amd64、arm64、arm）" >&2
    exit 1
    ;;
esac

if [[ "$(uname -s)" != "Linux" ]]; then
  echo "此脚本只能在 Linux 主机上运行。" >&2
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
if ! command -v pkg-config >/dev/null 2>&1 || ! pkg-config --exists gtk+-3.0 gio-unix-2.0; then
  echo "缺少 GTK 3 开发依赖。请安装 gtk+-3.0 开发包。" >&2
  exit 1
fi

build_tags=()
if pkg-config --exists webkit2gtk-4.1; then
  build_tags=(-tags webkit2_41)
elif ! pkg-config --exists webkit2gtk-4.0; then
  echo "缺少 WebKitGTK 开发依赖。请安装 libwebkit2gtk-4.1-dev 或 libwebkit2gtk-4.0-dev。" >&2
  exit 1
fi

cd "$project_dir"
wails build -platform "linux/$architecture" "${build_tags[@]}" -clean
echo "Linux 构建完成: $project_dir/build/bin/taskai"
